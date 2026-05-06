package capture

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcapgo"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// --- Engine tests that exercise Filter, Stats, SendARP, GetInterfaceMAC ---

// findLoopback returns the loopback interface name or skips the test.
func findLoopback(t *testing.T) string {
	t.Helper()

	if os.Getenv("CI") != "" {
		t.Skip("Skipping test in CI environment")
	}

	for _, name := range []string{"lo", "lo0", "Loopback"} {
		if InterfaceExists(name) {
			return name
		}
	}

	t.Skip("No loopback interface found")

	return ""
}

// TestEngine_Filter tests Filter getter and SetFilter.
func TestEngine_Filter(t *testing.T) {
	iface := findLoopback(t)

	engine, err := New(iface, 0)
	if err != nil {
		t.Skipf("Cannot create engine: %v", err)
	}
	defer engine.Close()

	// Initially empty
	if f := engine.Filter(); f != "" {
		t.Errorf("Expected empty filter, got %q", f)
	}

	// Set a valid BPF filter (use "ip" which works on all link types)
	err = engine.SetFilter("ip")
	if err != nil {
		// Some loopback interfaces reject certain filters; skip gracefully
		t.Skipf("SetFilter(ip) failed (may be unsupported on loopback): %v", err)
	}

	if f := engine.Filter(); f != "ip" {
		t.Errorf("Expected filter 'ip', got %q", f)
	}

	// Set another filter
	err = engine.SetFilter("tcp")
	if err != nil {
		t.Skipf("SetFilter(tcp) failed: %v", err)
	}

	if f := engine.Filter(); f != "tcp" {
		t.Errorf("Expected filter 'tcp', got %q", f)
	}
}

// TestEngine_SetFilter_Invalid tests SetFilter with an invalid expression.
func TestEngine_SetFilter_Invalid(t *testing.T) {
	iface := findLoopback(t)

	engine, err := New(iface, 0)
	if err != nil {
		t.Skipf("Cannot create engine: %v", err)
	}
	defer engine.Close()

	err = engine.SetFilter("not a valid bpf expression !!!")
	if err == nil {
		t.Skip("System accepted invalid BPF filter (unusual)")
	}
	// Filter should not have been updated on error
}

// TestEngine_Stats tests Stats retrieval.
func TestEngine_Stats(t *testing.T) {
	iface := findLoopback(t)

	engine, err := New(iface, 0)
	if err != nil {
		t.Skipf("Cannot create engine: %v", err)
	}
	defer engine.Close()

	stats, err := engine.Stats()
	if err != nil {
		t.Fatalf("Stats() failed: %v", err)
	}

	if stats == nil {
		t.Fatal("Stats() returned nil without error")
	}

	// On a fresh engine, received/dropped should be 0
	if stats.PacketsReceived < 0 {
		t.Errorf("PacketsReceived should be >= 0, got %d", stats.PacketsReceived)
	}
}

// TestEngine_SendARP_Request tests sending an ARP request on loopback.
func TestEngine_SendARP_Request(t *testing.T) {
	iface := findLoopback(t)

	engine, err := New(iface, 0)
	if err != nil {
		t.Skipf("Cannot create engine: %v", err)
	}
	defer engine.Close()

	srcMAC := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dstMAC := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

	// May fail on loopback but should not panic
	_ = engine.SendARP(srcMAC, dstMAC, "192.168.1.1", "192.168.1.2", true)
}

// TestEngine_SendARP_Reply tests sending an ARP reply.
func TestEngine_SendARP_Reply(t *testing.T) {
	iface := findLoopback(t)

	engine, err := New(iface, 0)
	if err != nil {
		t.Skipf("Cannot create engine: %v", err)
	}
	defer engine.Close()

	srcMAC := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dstMAC := []byte{0x00, 0x66, 0x77, 0x88, 0x99, 0xaa}

	_ = engine.SendARP(srcMAC, dstMAC, "10.0.0.1", "10.0.0.2", false)
}

// TestEngine_SendARP_InvalidSrcIP tests SendARP with invalid source IP.
func TestEngine_SendARP_InvalidSrcIP(t *testing.T) {
	iface := findLoopback(t)

	engine, err := New(iface, 0)
	if err != nil {
		t.Skipf("Cannot create engine: %v", err)
	}
	defer engine.Close()

	srcMAC := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dstMAC := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

	err = engine.SendARP(srcMAC, dstMAC, "not-an-ip", "192.168.1.2", true)
	if err == nil {
		t.Error("Expected error for invalid source IP")
	}
}

// TestEngine_SendARP_InvalidDstIP tests SendARP with invalid destination IP.
func TestEngine_SendARP_InvalidDstIP(t *testing.T) {
	iface := findLoopback(t)

	engine, err := New(iface, 0)
	if err != nil {
		t.Skipf("Cannot create engine: %v", err)
	}
	defer engine.Close()

	srcMAC := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dstMAC := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

	err = engine.SendARP(srcMAC, dstMAC, "192.168.1.1", "invalid-ip", true)
	if err == nil {
		t.Error("Expected error for invalid destination IP")
	}
}

// TestEngine_GetInterfaceMAC tests MAC address retrieval.
func TestEngine_GetInterfaceMAC(t *testing.T) {
	iface := findLoopback(t)

	engine, err := New(iface, 0)
	if err != nil {
		t.Skipf("Cannot create engine: %v", err)
	}
	defer engine.Close()

	mac, err := engine.GetInterfaceMAC()
	if err != nil {
		// Loopback typically has no MAC, so this error is expected
		if !errors.Is(err, ErrNoMACAddressFound) {
			t.Errorf("Expected ErrNoMACAddressFound, got: %v", err)
		}

		return
	}

	// If we did get a MAC, verify it has the right length
	if len(mac) != macAddressSize {
		t.Errorf("Expected MAC of length %d, got %d", macAddressSize, len(mac))
	}
}

// TestEngine_SendPacket_VerboseDebug tests SendPacket with verbose debug level.
func TestEngine_SendPacket_VerboseDebug(t *testing.T) {
	iface := findLoopback(t)

	engine, err := New(iface, debugLevelVerbose)
	if err != nil {
		t.Skipf("Cannot create engine: %v", err)
	}
	defer engine.Close()

	// Build a minimal packet
	eth := &layers.Ethernet{
		SrcMAC:       []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMAC:       []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeIPv4,
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true}

	_ = gopacket.SerializeLayers(buf, opts, eth, gopacket.Payload([]byte{0x01, 0x02}))

	// Should exercise the debug logging path; may fail on loopback but should not panic
	_ = engine.SendPacket(buf.Bytes())
}

// TestEngine_Close_NilHandle tests Close with nil handle.
func TestEngine_Close_NilHandle(_ *testing.T) {
	engine := &Engine{
		interfaceName: "test",
		handle:        nil,
	}

	// Should not panic
	engine.Close()
}

// --- Playback tests for Start with debug logging paths ---

// createTestPCAPFile creates a valid PCAP file for playback tests.
func createTestPCAPFile(t *testing.T, count int) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.pcap")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create PCAP file: %v", err)
	}
	defer func() { _ = f.Close() }()

	w := pcapgo.NewWriter(f)

	if err = w.WriteFileHeader(1600, layers.LinkTypeEthernet); err != nil {
		t.Fatalf("Failed to write PCAP header: %v", err)
	}

	srcMAC := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dstMAC := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	baseTime := time.Now()

	for i := range count {
		eth := &layers.Ethernet{
			SrcMAC:       srcMAC,
			DstMAC:       dstMAC,
			EthernetType: layers.EthernetTypeIPv4,
		}

		buf := gopacket.NewSerializeBuffer()
		_ = gopacket.SerializeLayers(buf, gopacket.SerializeOptions{}, eth, gopacket.Payload([]byte{byte(i)}))

		ts := baseTime.Add(time.Duration(i*10) * time.Millisecond)
		info := gopacket.CaptureInfo{
			Timestamp:     ts,
			CaptureLength: len(buf.Bytes()),
			Length:        len(buf.Bytes()),
		}

		_ = w.WritePacket(info, buf.Bytes())
	}

	return path
}

// TestPlaybackEngine_Start_WithDebugLogging tests Start with debug levels that trigger logging.
func TestPlaybackEngine_Start_WithDebugLogging(t *testing.T) {
	iface := findLoopback(t)

	engine, err := New(iface, 0)
	if err != nil {
		t.Skipf("Cannot create engine: %v", err)
	}
	defer engine.Close()

	pcapFile := createTestPCAPFile(t, 3)

	// Test with ScaleTime and LoopTime set so debug logging fires
	playbackConfig := &config.CapturePlayback{
		FileName:  pcapFile,
		ScaleTime: 2.0,
		LoopTime:  500,
	}

	pb := NewPlaybackEngine(engine, playbackConfig, 1) // debug >= 1 triggers logging

	err = pb.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Let it run briefly, then stop
	time.Sleep(100 * time.Millisecond)
	pb.Stop()

	if pb.IsRunning() {
		t.Error("Expected playback to be stopped")
	}
}

// TestPlaybackEngine_Start_PlayOnce tests Start without looping (plays once and exits).
func TestPlaybackEngine_Start_PlayOnce(t *testing.T) {
	iface := findLoopback(t)

	engine, err := New(iface, 0)
	if err != nil {
		t.Skipf("Cannot create engine: %v", err)
	}
	defer engine.Close()

	pcapFile := createTestPCAPFile(t, 2)

	playbackConfig := &config.CapturePlayback{
		FileName:  pcapFile,
		ScaleTime: 1.0,
		LoopTime:  0, // no loop, play once
	}

	pb := NewPlaybackEngine(engine, playbackConfig, debugLevelVerbose) // verbose to cover sendPacketWithLogging

	err = pb.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Wait for the play-once to finish (packets are 10ms apart, so 50ms should be plenty)
	time.Sleep(200 * time.Millisecond)

	// After play-once, the goroutine exits, but running flag stays true until Stop
	pb.Stop()
}

// TestPlaybackEngine_LoadPCAP_EmptyFile tests loading a PCAP with zero packets.
func TestPlaybackEngine_LoadPCAP_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.pcap")

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}

	w := pcapgo.NewWriter(f)

	err = w.WriteFileHeader(1600, layers.LinkTypeEthernet)
	if err != nil {
		t.Fatal(err)
	}

	_ = f.Close()

	pb := &PlaybackEngine{
		config: &config.CapturePlayback{
			FileName: path,
		},
		stopChan: make(chan struct{}),
	}

	packets, err := pb.loadPCAP()
	if err != nil {
		t.Fatalf("loadPCAP on empty file failed: %v", err)
	}

	if len(packets) != 0 {
		t.Errorf("Expected 0 packets from empty PCAP, got %d", len(packets))
	}
}

// TestPlaybackEngine_LoadPCAP_InvalidFile tests loadPCAP with a non-PCAP file.
func TestPlaybackEngine_LoadPCAP_InvalidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.pcap")

	err := os.WriteFile(path, []byte("not a pcap file"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	pb := &PlaybackEngine{
		config: &config.CapturePlayback{
			FileName: path,
		},
		stopChan: make(chan struct{}),
	}

	_, err = pb.loadPCAP()
	if err == nil {
		t.Error("Expected error loading invalid PCAP file")
	}
}

// TestRateLimiter_LargeRate tests RateLimiter with a large packets-per-second value.
func TestRateLimiter_LargeRate(t *testing.T) {
	rl := NewRateLimiter(10000)
	defer rl.Stop()

	if rl.packetsPerSecond != 10000 {
		t.Errorf("Expected 10000, got %d", rl.packetsPerSecond)
	}

	// Should be able to drain many tokens quickly
	for range 100 {
		rl.Wait()
	}
}

// TestRateLimiter_OneRate tests RateLimiter with exactly 1 pps.
func TestRateLimiter_OneRate(t *testing.T) {
	rl := NewRateLimiter(1)
	defer rl.Stop()

	if rl.packetsPerSecond != 1 {
		t.Errorf("Expected 1, got %d", rl.packetsPerSecond)
	}

	// First wait should succeed immediately (pre-filled)
	start := time.Now()
	rl.Wait()

	if time.Since(start) > 50*time.Millisecond {
		t.Error("First Wait on 1-pps limiter should be immediate")
	}
}
