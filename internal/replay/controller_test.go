package replay_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcapgo"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/capture"
	"github.com/MustardSeedNetworks/niac-go/internal/replay"
)

// writeTestPCAP writes a minimal PCAP with packetCount trivial Ethernet
// frames, mirroring internal/capture's own test helper (unexported there,
// so this package needs its own copy for the black-box Status() test
// below).
func writeTestPCAP(t *testing.T, packetCount int) string {
	t.Helper()

	pcapFile := filepath.Join(t.TempDir(), "replay-progress.pcap")

	f, createErr := os.Create(pcapFile)
	if createErr != nil {
		t.Fatalf("failed to create temp PCAP: %v", createErr)
	}
	defer func() { _ = f.Close() }()

	w := pcapgo.NewWriter(f)
	if headerErr := w.WriteFileHeader(1600, layers.LinkTypeEthernet); headerErr != nil {
		t.Fatalf("failed to write PCAP header: %v", headerErr)
	}

	srcMAC := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dstMAC := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	baseTime := time.Now()

	for i := range packetCount {
		eth := &layers.Ethernet{SrcMAC: srcMAC, DstMAC: dstMAC, EthernetType: layers.EthernetTypeIPv4}
		buf := gopacket.NewSerializeBuffer()
		payload := []byte{byte(i), 0x01, 0x02, 0x03}
		opts := gopacket.SerializeOptions{}
		if serializeErr := gopacket.SerializeLayers(buf, opts, eth, gopacket.Payload(payload)); serializeErr != nil {
			t.Fatalf("failed to serialize packet: %v", serializeErr)
		}
		info := gopacket.CaptureInfo{
			Timestamp:     baseTime.Add(time.Duration(i) * time.Millisecond),
			CaptureLength: len(buf.Bytes()),
			Length:        len(buf.Bytes()),
		}
		if writeErr := w.WritePacket(info, buf.Bytes()); writeErr != nil {
			t.Fatalf("failed to write packet: %v", writeErr)
		}
	}

	return pcapFile
}

// TestStatus_ReportsProgress drives a real replay over the loopback
// interface and asserts Status() surfaces non-zero sent counters plus a
// correct percent-complete once the pass finishes. Skipped in CI: sending
// raw packets needs privileges the CI runner doesn't have (same convention
// as internal/capture's playback tests).
func TestStatus_ReportsProgress(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping replay test in CI environment")
	}

	var testInterface string
	for _, name := range []string{"lo", "lo0"} {
		if capture.InterfaceExists(name) {
			testInterface = name
			break
		}
	}
	if testInterface == "" {
		t.Skip("No loopback interface found")
	}

	engine, err := capture.New(testInterface, 0)
	if err != nil {
		t.Skipf("cannot create engine: %v", err)
	}
	defer engine.Close()

	const packetCount = 5
	pcapFile := writeTestPCAP(t, packetCount)

	c := replay.New(engine, 0)
	if _, startErr := c.Start(api.ReplayRequest{File: pcapFile}); startErr != nil {
		t.Fatalf("Start failed: %v", startErr)
	}

	// Playback of a 5-packet, sub-millisecond-spaced PCAP finishes almost
	// immediately; poll briefly rather than sleeping a fixed duration.
	deadline := time.Now().Add(2 * time.Second)
	var state api.ReplayState
	for time.Now().Before(deadline) {
		state = c.Status()
		if state.PacketsSent >= packetCount {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, stopErr := c.Stop(); stopErr != nil {
		t.Fatalf("Stop failed: %v", stopErr)
	}

	if state.PacketsSent != packetCount {
		t.Errorf("expected PacketsSent=%d, got %d", packetCount, state.PacketsSent)
	}
	if state.PacketsTotal != packetCount {
		t.Errorf("expected PacketsTotal=%d, got %d", packetCount, state.PacketsTotal)
	}
	if state.BytesSent == 0 {
		t.Error("expected non-zero BytesSent")
	}
	if state.PercentComplete != 100 {
		t.Errorf("expected PercentComplete=100, got %v", state.PercentComplete)
	}
}

// TestStart_NoEngine verifies a controller without a capture engine refuses
// to Start with the public sentinel rather than panicking. This is the
// failure mode the daemon hits when StartSimulation is called before the
// engine attaches.
func TestStart_NoEngine(t *testing.T) {
	c := replay.New(nil, 0)

	state, err := c.Start(api.ReplayRequest{File: "/tmp/example.pcap"})
	if !errors.Is(err, replay.ErrEngineUnavailable) {
		t.Fatalf("got %v, want ErrEngineUnavailable", err)
	}
	if state.Running {
		t.Errorf("state should not be Running after a failed Start, got %+v", state)
	}
}

// TestStop_FreshController is the path the daemon's shutdown sequence relies
// on — Stop on a never-started controller is a no-op and must never error.
func TestStop_FreshController(t *testing.T) {
	c := replay.New(nil, 0)

	state, err := c.Stop()
	if err != nil {
		t.Errorf("Stop on fresh controller should not error, got %v", err)
	}
	if state.Running {
		t.Errorf("state.Running should be false after Stop on fresh controller, got %+v", state)
	}
}

// TestStatus_Default returns the zero ReplayState before Start.
func TestStatus_Default(t *testing.T) {
	c := replay.New(nil, 0)
	if c.Status().Running {
		t.Errorf("fresh controller should report Running=false")
	}
}
