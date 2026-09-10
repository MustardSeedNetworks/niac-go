//go:build linux && integration

package wiretest_test

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/gopacket/gopacket/pcap"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestTopologyChangeUsesScenarioVLANOnWire(t *testing.T) {
	requireWire(t)
	reader, err := pcap.OpenLive(testIface, 65536, true, 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	if err = reader.SetBPFFilter("ether dst 01:80:c2:00:00:00 and (ether[20] = 0x80 or ether[24] = 0x80)"); err != nil {
		t.Fatal(err)
	}
	first := actionWireDevice("vlan-one", "10.254.200.11", 11)
	second := actionWireDevice("vlan-two", "10.254.200.12", 12)
	cfg := &config.Config{Segments: []config.Segment{
		{Tag: 200, Devices: []config.Device{first}},
		{Tag: 300, Devices: []config.Device{second}},
	}}
	stack := startActionWireStack(t, cfg, nil)
	for _, name := range []string{first.Name, second.Name} {
		for range 2 {
			if err = stack.ExecuteDeviceAction(name, devicestate.ActionSTPTopologyChange, "once"); err != nil {
				t.Fatal(err)
			}
		}
	}
	frames, _, err := readFrames(reader, 0x8100, 3, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 {
		t.Fatalf("received %d tagged TCNs, want exactly2 (%s)", len(frames), captureLosses(reader))
	}
	seen := make(map[string]bool)
	want := map[string]uint16{first.MACAddress.String(): 200, second.MACAddress.String(): 300}
	for _, frame := range frames {
		if len(frame) < 25 {
			t.Fatalf("short TCN: %x", frame)
		}
		source := net.HardwareAddr(frame[6:12]).String()
		vlan, found := want[source]
		if !found || seen[source] || binary.BigEndian.Uint16(frame[14:16])&0xfff != vlan {
			t.Fatalf("wrong or duplicate TCN identity: %x", frame[:18])
		}
		seen[source] = true
		if !bytes.Equal(frame[16:25], []byte{0, 7, 0x42, 0x42, 3, 0, 0, 0, 0x80}) {
			t.Fatalf("not a short IEEE TCN: %x", frame[16:25])
		}
	}
	t.Log(
		"actual Ethernet: distinct scenario VLANs, exact TCN payload and no duplicate action emission",
	)
}
