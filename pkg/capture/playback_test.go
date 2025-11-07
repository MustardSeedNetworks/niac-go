package capture

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
	"github.com/krisarmstrong/niac-go/pkg/config"
)

// createTestPCAP creates a temporary PCAP file for testing
func createTestPCAP(t *testing.T, packetCount int) string {
	t.Helper()

	tmpDir := t.TempDir()
	pcapFile := filepath.Join(tmpDir, "test.pcap")

	f, err := os.Create(pcapFile)
	if err != nil {
		t.Fatalf("Failed to create temp PCAP: %v", err)
	}
	defer f.Close()

	w := pcapgo.NewWriter(f)
	if err := w.WriteFileHeader(1600, layers.LinkTypeEthernet); err != nil {
		t.Fatalf("Failed to write PCAP header: %v", err)
	}

	// Write test packets
	srcMAC := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dstMAC := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	baseTime := time.Now()

	for i := 0; i < packetCount; i++ {
		eth := &layers.Ethernet{
			SrcMAC:       srcMAC,
			DstMAC:       dstMAC,
			EthernetType: layers.EthernetTypeIPv4,
		}

		buf := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{}
		payload := []byte{byte(i), 0x01, 0x02, 0x03}

		if err := gopacket.SerializeLayers(buf, opts, eth, gopacket.Payload(payload)); err != nil {
			t.Fatalf("Failed to serialize packet: %v", err)
		}

		// Write packet with incremental timestamp
		timestamp := baseTime.Add(time.Duration(i*100) * time.Millisecond)
		info := gopacket.CaptureInfo{
			Timestamp:     timestamp,
			CaptureLength: len(buf.Bytes()),
			Length:        len(buf.Bytes()),
		}

		if err := w.WritePacket(info, buf.Bytes()); err != nil {
			t.Fatalf("Failed to write packet: %v", err)
		}
	}

	return pcapFile
}

// TestNewPlaybackEngine tests playback engine creation
func TestNewPlaybackEngine(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping playback test in CI environment")
	}

	// Need a real engine for playback
	loopbackNames := []string{"lo", "lo0"}
	var testInterface string

	for _, name := range loopbackNames {
		if InterfaceExists(name) {
			testInterface = name
			break
		}
	}

	if testInterface == "" {
		t.Skip("No loopback interface found")
	}

	engine, err := New(testInterface, 0)
	if err != nil {
		t.Skipf("Cannot create engine: %v", err)
	}
	defer engine.Close()

	playbackConfig := &config.CapturePlayback{
		FileName: "test.pcap",
	}

	pb := NewPlaybackEngine(engine, playbackConfig, 0)

	if pb == nil {
		t.Fatal("NewPlaybackEngine returned nil")
	}

	if pb.engine != engine {
		t.Error("Engine not set correctly")
	}

	if pb.config != playbackConfig {
		t.Error("Config not set correctly")
	}

	if pb.debugLevel != 0 {
		t.Errorf("Expected debug level 0, got %d", pb.debugLevel)
	}

	if pb.stopChan == nil {
		t.Error("stopChan not initialized")
	}
}

// TestPlaybackEngine_Start_NoConfig tests starting without config
func TestPlaybackEngine_Start_NoConfig(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping playback test in CI environment")
	}

	loopbackNames := []string{"lo", "lo0"}
	var testInterface string

	for _, name := range loopbackNames {
		if InterfaceExists(name) {
			testInterface = name
			break
		}
	}

	if testInterface == "" {
		t.Skip("No loopback interface found")
	}

	engine, err := New(testInterface, 0)
	if err != nil {
		t.Skipf("Cannot create engine: %v", err)
	}
	defer engine.Close()

	pb := NewPlaybackEngine(engine, nil, 0)

	err = pb.Start()
	if err == nil {
		t.Error("Expected error when starting with nil config")
	}
}

// TestPlaybackEngine_Start_NonExistentFile tests starting with missing file
func TestPlaybackEngine_Start_NonExistentFile(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping playback test in CI environment")
	}

	loopbackNames := []string{"lo", "lo0"}
	var testInterface string

	for _, name := range loopbackNames {
		if InterfaceExists(name) {
			testInterface = name
			break
		}
	}

	if testInterface == "" {
		t.Skip("No loopback interface found")
	}

	engine, err := New(testInterface, 0)
	if err != nil {
		t.Skipf("Cannot create engine: %v", err)
	}
	defer engine.Close()

	playbackConfig := &config.CapturePlayback{
		FileName: "/tmp/definitely-does-not-exist-12345.pcap",
	}

	pb := NewPlaybackEngine(engine, playbackConfig, 0)

	err = pb.Start()
	if err == nil {
		t.Error("Expected error when starting with non-existent file")
	}
}

// TestPlaybackEngine_Stop tests stopping playback
func TestPlaybackEngine_Stop(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping playback test in CI environment")
	}

	loopbackNames := []string{"lo", "lo0"}
	var testInterface string

	for _, name := range loopbackNames {
		if InterfaceExists(name) {
			testInterface = name
			break
		}
	}

	if testInterface == "" {
		t.Skip("No loopback interface found")
	}

	engine, err := New(testInterface, 0)
	if err != nil {
		t.Skipf("Cannot create engine: %v", err)
	}
	defer engine.Close()

	playbackConfig := &config.CapturePlayback{
		FileName: "test.pcap",
	}

	pb := NewPlaybackEngine(engine, playbackConfig, 0)

	// Stop before start should not panic
	pb.Stop()

	// Stop twice should not panic
	pb.Stop()
}

// TestPlaybackEngine_IsRunning tests running state
func TestPlaybackEngine_IsRunning(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping playback test in CI environment")
	}

	loopbackNames := []string{"lo", "lo0"}
	var testInterface string

	for _, name := range loopbackNames {
		if InterfaceExists(name) {
			testInterface = name
			break
		}
	}

	if testInterface == "" {
		t.Skip("No loopback interface found")
	}

	engine, err := New(testInterface, 0)
	if err != nil {
		t.Skipf("Cannot create engine: %v", err)
	}
	defer engine.Close()

	playbackConfig := &config.CapturePlayback{
		FileName: "test.pcap",
	}

	pb := NewPlaybackEngine(engine, playbackConfig, 0)

	if pb.IsRunning() {
		t.Error("Expected IsRunning() to be false initially")
	}
}

// TestPlaybackEngine_GetConfig tests config retrieval
func TestPlaybackEngine_GetConfig(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping playback test in CI environment")
	}

	loopbackNames := []string{"lo", "lo0"}
	var testInterface string

	for _, name := range loopbackNames {
		if InterfaceExists(name) {
			testInterface = name
			break
		}
	}

	if testInterface == "" {
		t.Skip("No loopback interface found")
	}

	engine, err := New(testInterface, 0)
	if err != nil {
		t.Skipf("Cannot create engine: %v", err)
	}
	defer engine.Close()

	playbackConfig := &config.CapturePlayback{
		FileName: "test.pcap",
	}

	pb := NewPlaybackEngine(engine, playbackConfig, 0)

	cfg := pb.GetConfig()
	if cfg != playbackConfig {
		t.Error("GetConfig() returned different config")
	}
}

// TestPlaybackEngine_LoadPCAP tests PCAP file loading
func TestPlaybackEngine_LoadPCAP(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping playback test in CI environment")
	}

	// Create test PCAP file
	pcapFile := createTestPCAP(t, 5)

	loopbackNames := []string{"lo", "lo0"}
	var testInterface string

	for _, name := range loopbackNames {
		if InterfaceExists(name) {
			testInterface = name
			break
		}
	}

	if testInterface == "" {
		t.Skip("No loopback interface found")
	}

	engine, err := New(testInterface, 0)
	if err != nil {
		t.Skipf("Cannot create engine: %v", err)
	}
	defer engine.Close()

	playbackConfig := &config.CapturePlayback{
		FileName: pcapFile,
	}

	pb := NewPlaybackEngine(engine, playbackConfig, 0)

	// Test loading PCAP
	packets, err := pb.loadPCAP()
	if err != nil {
		t.Fatalf("Failed to load PCAP: %v", err)
	}

	if len(packets) != 5 {
		t.Errorf("Expected 5 packets, got %d", len(packets))
	}

	// Verify packet data
	for i, pkt := range packets {
		if len(pkt.Data) == 0 {
			t.Errorf("Packet %d has no data", i)
		}

		if pkt.Timestamp.IsZero() {
			t.Errorf("Packet %d has zero timestamp", i)
		}
	}
}

// TestPlaybackEngine_CalculatePacketDelay tests timing calculation
func TestPlaybackEngine_CalculatePacketDelay(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping playback test in CI environment")
	}

	loopbackNames := []string{"lo", "lo0"}
	var testInterface string

	for _, name := range loopbackNames {
		if InterfaceExists(name) {
			testInterface = name
			break
		}
	}

	if testInterface == "" {
		t.Skip("No loopback interface found")
	}

	engine, err := New(testInterface, 0)
	if err != nil {
		t.Skipf("Cannot create engine: %v", err)
	}
	defer engine.Close()

	tests := []struct {
		name          string
		scaleTime     float64
		packetOffset  time.Duration
		expectedDelay time.Duration
		tolerance     time.Duration
	}{
		{"no scaling", 1.0, 100 * time.Millisecond, 100 * time.Millisecond, 50 * time.Millisecond},
		{"2x speed", 2.0, 100 * time.Millisecond, 200 * time.Millisecond, 50 * time.Millisecond},
		{"0.5x speed", 0.5, 100 * time.Millisecond, 50 * time.Millisecond, 25 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			playbackConfig := &config.CapturePlayback{
				FileName:  "test.pcap",
				ScaleTime: tt.scaleTime,
			}

			pb := NewPlaybackEngine(engine, playbackConfig, 0)

			baseTime := time.Now()
			firstPacketTime := baseTime
			packetTime := baseTime.Add(tt.packetOffset)

			pkt := PlaybackPacket{
				Timestamp: packetTime,
			}

			delay := pb.calculatePacketDelay(pkt, baseTime, firstPacketTime)

			if delay < tt.expectedDelay-tt.tolerance || delay > tt.expectedDelay+tt.tolerance {
				t.Errorf("Expected delay ~%v, got %v", tt.expectedDelay, delay)
			}
		})
	}
}

// TestPlaybackEngine_CalculatePacketDelay_PastDue tests handling of past-due packets
func TestPlaybackEngine_CalculatePacketDelay_PastDue(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping playback test in CI environment")
	}

	loopbackNames := []string{"lo", "lo0"}
	var testInterface string

	for _, name := range loopbackNames {
		if InterfaceExists(name) {
			testInterface = name
			break
		}
	}

	if testInterface == "" {
		t.Skip("No loopback interface found")
	}

	engine, err := New(testInterface, 0)
	if err != nil {
		t.Skipf("Cannot create engine: %v", err)
	}
	defer engine.Close()

	playbackConfig := &config.CapturePlayback{
		FileName:  "test.pcap",
		ScaleTime: 1.0,
	}

	pb := NewPlaybackEngine(engine, playbackConfig, 0)

	// Simulate packet that should have been sent in the past
	baseTime := time.Now().Add(-1 * time.Second) // Started 1 second ago
	firstPacketTime := baseTime
	packetTime := baseTime.Add(100 * time.Millisecond) // This packet was due 900ms ago

	pkt := PlaybackPacket{
		Timestamp: packetTime,
	}

	delay := pb.calculatePacketDelay(pkt, baseTime, firstPacketTime)

	// Should return 0 for past-due packets (send immediately)
	if delay != 0 {
		t.Errorf("Expected 0 delay for past-due packet, got %v", delay)
	}
}

// TestPlaybackPacket_Structure tests PlaybackPacket struct
func TestPlaybackPacket_Structure(t *testing.T) {
	pkt := PlaybackPacket{
		Data:      []byte{0x01, 0x02, 0x03},
		Timestamp: time.Now(),
	}

	if len(pkt.Data) != 3 {
		t.Errorf("Expected 3 bytes, got %d", len(pkt.Data))
	}

	if pkt.Timestamp.IsZero() {
		t.Error("Timestamp is zero")
	}
}

// TestPlaybackEngine_ConcurrentStartStop tests concurrent start/stop calls
func TestPlaybackEngine_ConcurrentStartStop(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping playback test in CI environment")
	}

	loopbackNames := []string{"lo", "lo0"}
	var testInterface string

	for _, name := range loopbackNames {
		if InterfaceExists(name) {
			testInterface = name
			break
		}
	}

	if testInterface == "" {
		t.Skip("No loopback interface found")
	}

	engine, err := New(testInterface, 0)
	if err != nil {
		t.Skipf("Cannot create engine: %v", err)
	}
	defer engine.Close()

	playbackConfig := &config.CapturePlayback{
		FileName: "test.pcap",
	}

	pb := NewPlaybackEngine(engine, playbackConfig, 0)

	// Multiple Stop() calls should not panic
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			pb.Stop()
			done <- struct{}{}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("Concurrent Stop() calls deadlocked")
		}
	}
}
