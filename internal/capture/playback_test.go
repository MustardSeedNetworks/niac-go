package capture

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcapgo"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// createTestPCAP creates a temporary PCAP file for testing.
func createTestPCAP(t *testing.T, packetCount int) string {
	t.Helper()

	tmpDir := t.TempDir()
	pcapFile := filepath.Join(tmpDir, "test.pcap")

	f, err := os.Create(pcapFile)
	if err != nil {
		t.Fatalf("Failed to create temp PCAP: %v", err)
	}

	defer func() { _ = f.Close() }()

	w := pcapgo.NewWriter(f)
	headerErr := w.WriteFileHeader(1600, layers.LinkTypeEthernet)
	if headerErr != nil {
		t.Fatalf("Failed to write PCAP header: %v", headerErr)
	}

	// Write test packets
	srcMAC := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dstMAC := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	baseTime := time.Now()

	for i := range packetCount {
		eth := &layers.Ethernet{
			SrcMAC:       srcMAC,
			DstMAC:       dstMAC,
			EthernetType: layers.EthernetTypeIPv4,
		}

		buf := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{}
		payload := []byte{byte(i), 0x01, 0x02, 0x03}

		serializeErr := gopacket.SerializeLayers(buf, opts, eth, gopacket.Payload(payload))
		if serializeErr != nil {
			t.Fatalf("Failed to serialize packet: %v", serializeErr)
		}

		// Write packet with incremental timestamp
		timestamp := baseTime.Add(time.Duration(i*100) * time.Millisecond)
		info := gopacket.CaptureInfo{
			Timestamp:     timestamp,
			CaptureLength: len(buf.Bytes()),
			Length:        len(buf.Bytes()),
		}

		err = w.WritePacket(info, buf.Bytes())
		if err != nil {
			t.Fatalf("Failed to write packet: %v", err)
		}
	}

	return pcapFile
}

// TestNewPlaybackEngine tests playback engine creation.
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

// TestPlaybackEngine_Start_NoConfig tests starting without config.
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

// TestPlaybackEngine_Start_NonExistentFile tests starting with missing file.
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

// TestPlaybackEngine_Stop tests stopping playback.
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

// TestPlaybackEngine_IsRunning tests running state.
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

// TestPlaybackEngine_GetConfig tests config retrieval.
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

// TestPlaybackEngine_LoadPCAP tests PCAP file loading.
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
	packets, err := collectPlayback(pb)
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

// TestPlaybackEngine_CalculatePacketDelay tests timing calculation.
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

// TestPlaybackEngine_CalculatePacketDelay_PastDue tests handling of past-due packets.
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

// TestPlaybackPacket_Structure tests PlaybackPacket struct.
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

// TestPlaybackEngine_RestartAfterStop verifies Start can be called again
// after Stop with a real PCAP file. Regression for #464.
func TestPlaybackEngine_RestartAfterStop(t *testing.T) {
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

	pcapFile := createTestPCAP(t, 3)
	playbackConfig := &config.CapturePlayback{FileName: pcapFile}
	pb := NewPlaybackEngine(engine, playbackConfig, 0)

	for i := range 3 {
		if startErr := pb.Start(); startErr != nil {
			t.Fatalf("iter %d: Start failed: %v", i, startErr)
		}
		// Tiny window for the goroutine to be alive then bail.
		time.Sleep(20 * time.Millisecond)
		pb.Stop()
	}
}

// TestPlaybackEngine_ConcurrentStartStop tests concurrent start/stop calls.
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

	for range 10 {
		go func() {
			pb.Stop()

			done <- struct{}{}
		}()
	}

	// Wait for all goroutines
	for range 10 {
		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("Concurrent Stop() calls deadlocked")
		}
	}
}

// TestPlaybackEngine_Progress_ZeroValue tests that a freshly constructed
// engine reports an all-zero progress snapshot (no total known yet).
func TestPlaybackEngine_Progress_ZeroValue(t *testing.T) {
	pb := &PlaybackEngine{stopChan: make(chan struct{})}

	progress := pb.Progress()
	if progress.PacketsSent != 0 || progress.BytesSent != 0 {
		t.Errorf("expected zero sent counters, got %+v", progress)
	}
	if progress.TotalPackets != 0 || progress.TotalBytes != 0 {
		t.Errorf("expected zero total counters, got %+v", progress)
	}
}

// TestPlaybackEngine_Progress_TracksCounters is a white-box test that drives
// the atomic counters directly (same package) to verify Progress() reflects
// them without needing a live capture engine.
func TestPlaybackEngine_Progress_TracksCounters(t *testing.T) {
	pb := &PlaybackEngine{stopChan: make(chan struct{})}

	pb.totalPackets.Store(10)
	pb.totalBytes.Store(1000)
	pb.packetsSent.Store(3)
	pb.bytesSent.Store(300)

	progress := pb.Progress()
	if progress.PacketsSent != 3 || progress.BytesSent != 300 {
		t.Errorf("expected sent=3/300, got %+v", progress)
	}
	if progress.TotalPackets != 10 || progress.TotalBytes != 1000 {
		t.Errorf("expected total=10/1000, got %+v", progress)
	}
}

// TestPlaybackEngine_Progress_AfterPlayOnce runs a real playOnce() pass over
// a loopback interface (same CI-skip convention as the rest of this file,
// since sending packets needs raw-socket privileges) and asserts the
// progress counters match the packets actually written to the PCAP.
func TestPlaybackEngine_Progress_AfterPlayOnce(t *testing.T) {
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

	const packetCount = 5
	pcapFile := createTestPCAP(t, packetCount)

	playbackConfig := &config.CapturePlayback{FileName: pcapFile}
	pb := NewPlaybackEngine(engine, playbackConfig, 0)

	// playOnce is synchronous, so no polling/timing is needed to observe the
	// final counters.
	pb.playOnce()

	progress := pb.Progress()
	if progress.TotalPackets != packetCount {
		t.Errorf("expected TotalPackets=%d, got %d", packetCount, progress.TotalPackets)
	}
	if progress.PacketsSent != packetCount {
		t.Errorf("expected PacketsSent=%d, got %d", packetCount, progress.PacketsSent)
	}
	if progress.TotalBytes == 0 {
		t.Error("expected non-zero TotalBytes")
	}
	if progress.BytesSent != progress.TotalBytes {
		t.Errorf("expected BytesSent (%d) to equal TotalBytes (%d) after a full pass",
			progress.BytesSent, progress.TotalBytes)
	}
}
