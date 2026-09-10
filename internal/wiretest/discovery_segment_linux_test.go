//go:build linux && integration

package wiretest_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/gopacket/gopacket/pcap"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestPeriodicDiscoveryUsesScenarioVLANsOnWire(t *testing.T) {
	requireWire(t)
	reader, err := pcap.OpenLive(testIface, 65536, true, 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	if err = reader.SetBPFFilter("ether dst 01:80:c2:00:00:0e or ether dst 01:00:0c:cc:cc:cc or " +
		"ether dst 00:e0:2b:00:00:00 or ether dst 01:e0:52:cc:cc:cc"); err != nil {
		t.Fatal(err)
	}
	first := discoveryWireDevice("hospital", 21)
	second := discoveryWireDevice("warehouse", 22)
	second.VLAN = 999
	cfg := &config.Config{Segments: []config.Segment{
		{Tag: 200, Devices: []config.Device{first}},
		{Tag: 300, Devices: []config.Device{second}},
	}}
	startActionWireStack(t, cfg, nil)
	frames := readDiscoveryWindow(t, reader)
	if len(frames) != 8 {
		t.Fatalf("received %d discovery frames, want exactly 8 (%s)", len(frames), captureLosses(reader))
	}
	want := map[string]uint16{first.MACAddress.String(): 200, second.MACAddress.String(): 300}
	seen := make(map[string]bool)
	for _, frame := range frames {
		assertDiscoveryWireFrame(t, frame, want, seen)
	}
	t.Log("Stack.Start emitted exactly four discovery protocols per scenario on their authored VLANs")
}

func discoveryWireDevice(name string, suffix byte) config.Device {
	device := actionWireDevice(name, "10.254.200.11", suffix)
	device.LLDPConfig = &config.LLDPConfig{Enabled: true}
	device.CDPConfig = &config.CDPConfig{Enabled: true}
	device.EDPConfig = &config.EDPConfig{Enabled: true}
	device.FDPConfig = &config.FDPConfig{Enabled: true}
	return device
}

func readDiscoveryWindow(t *testing.T, reader *pcap.Handle) [][]byte {
	t.Helper()
	var frames [][]byte
	// Observe beyond the expected eighth frame so duplicate emissions fail too.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		data, _, err := reader.ReadPacketData()
		if errors.Is(err, pcap.NextErrorTimeoutExpired) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		frames = append(frames, bytes.Clone(data))
		if len(frames) > 8 {
			break
		}
	}
	return frames
}

func assertDiscoveryWireFrame(t *testing.T, frame []byte, want map[string]uint16, seen map[string]bool) {
	t.Helper()
	if len(frame) < 26 || binary.BigEndian.Uint16(frame[12:14]) != 0x8100 {
		t.Fatalf("short or untagged discovery: %x", frame)
	}
	source := net.HardwareAddr(frame[6:12]).String()
	vlan, found := want[source]
	if !found || binary.BigEndian.Uint16(frame[14:16])&0xfff != vlan {
		t.Fatalf("unexpected source or VLAN: %x", frame[:18])
	}
	destination := net.HardwareAddr(frame[:6]).String()
	key := source + "/" + destination
	if seen[key] {
		t.Fatalf("duplicate protocol/source pair: %s", key)
	}
	seen[key] = true
	if destination == "01:80:c2:00:00:0e" {
		if binary.BigEndian.Uint16(frame[16:18]) != 0x88cc {
			t.Fatalf("LLDP destination has incorrect EtherType: %x", frame[:18])
		}
		return
	}
	headers := map[string][]byte{
		"01:00:0c:cc:cc:cc": {0xaa, 0xaa, 3, 0, 0, 0x0c, 0x20, 0},
		"00:e0:2b:00:00:00": {0xaa, 0xaa, 3, 0, 0xe0, 0x2b, 0, 0xbb},
		"01:e0:52:cc:cc:cc": {0xaa, 0xaa, 3, 0, 0xe0, 0x52, 0x20, 0},
	}
	header, found := headers[destination]
	if !found || !bytes.Equal(frame[18:26], header) {
		t.Fatalf("incorrect discovery protocol framing: %x", frame[:26])
	}
}
