package capture

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"
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

func TestPlaybackEngineForEachPacketExcludesTruncatedTail(t *testing.T) {
	pcapFile := createTestPCAP(t, 1)
	file, err := os.OpenFile(pcapFile, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open PCAP for truncation: %v", err)
	}

	recordHeader := make([]byte, 16)
	binary.LittleEndian.PutUint32(recordHeader[8:12], 64)
	binary.LittleEndian.PutUint32(recordHeader[12:16], 64)
	if _, err = file.Write(recordHeader); err != nil {
		_ = file.Close()
		t.Fatalf("append truncated record: %v", err)
	}
	if err = file.Close(); err != nil {
		t.Fatalf("close truncated PCAP: %v", err)
	}

	playback := NewPlaybackEngine(nil, &config.CapturePlayback{FileName: pcapFile}, 0)
	var packets int
	err = playback.forEachPacket(func(PlaybackPacket) bool {
		packets++
		return false
	})
	if err != nil {
		t.Fatalf("forEachPacket() error = %v, want valid-prefix replay", err)
	}
	if packets != 1 {
		t.Fatalf("callback count = %d, want only the one complete packet", packets)
	}
}

// TestNewPlaybackEngine tests playback engine creation.
func TestNewPlaybackEngine(t *testing.T) {
	sender := &recordingSender{}
	playbackConfig := &config.CapturePlayback{
		FileName: "test.pcap",
	}

	pb := NewPlaybackEngine(sender, playbackConfig, 0)

	if pb.engine != PacketSender(sender) {
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
	pb := NewPlaybackEngine(&recordingSender{}, nil, 0)

	if err := pb.Start(); !errors.Is(err, ErrNoPlaybackConfiguration) {
		t.Errorf("Start() error = %v, want ErrNoPlaybackConfiguration", err)
	}
}

// TestPlaybackEngine_Start_NonExistentFile tests starting with missing file.
func TestPlaybackEngine_Start_NonExistentFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "definitely-does-not-exist.pcap")
	pb := NewPlaybackEngine(&recordingSender{}, &config.CapturePlayback{FileName: missing}, 0)

	if err := pb.Start(); err == nil {
		t.Fatal("Start() with a non-existent file returned nil, want an error")
	}

	if pb.IsRunning() {
		t.Error("engine reports running after a failed Start")
	}
}

// TestPlaybackEngine_Stop tests stopping playback.
func TestPlaybackEngine_Stop(t *testing.T) {
	pb := NewPlaybackEngine(&recordingSender{}, &config.CapturePlayback{FileName: "test.pcap"}, 0)

	// Stop before start, and Stop twice, are both no-ops rather than panics.
	pb.Stop()
	pb.Stop()

	if pb.IsRunning() {
		t.Error("engine reports running after Stop on a never-started engine")
	}
}

// TestPlaybackEngine_IsRunning tests running state.
func TestPlaybackEngine_IsRunning(t *testing.T) {
	sender := &recordingSender{}
	pcapFile := createTestPCAP(t, 2)
	pb := NewPlaybackEngine(sender, &config.CapturePlayback{
		FileName: pcapFile,
		RateMode: config.RateTopspeed,
		LoopTime: 60_000, // long interval: the engine stays running after pass one
	}, 0)

	if pb.IsRunning() {
		t.Error("Expected IsRunning() to be false initially")
	}

	if err := pb.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if !pb.IsRunning() {
		t.Error("Expected IsRunning() to be true after Start")
	}

	pb.Stop()

	if pb.IsRunning() {
		t.Error("Expected IsRunning() to be false after Stop")
	}
}

// TestPlaybackEngine_GetConfig tests config retrieval.
func TestPlaybackEngine_GetConfig(t *testing.T) {
	playbackConfig := &config.CapturePlayback{
		FileName: "test.pcap",
	}

	pb := NewPlaybackEngine(&recordingSender{}, playbackConfig, 0)

	if cfg := pb.GetConfig(); cfg != playbackConfig {
		t.Error("GetConfig() returned different config")
	}
}

// TestPlaybackEngine_LoadPCAP tests PCAP file loading.
func TestPlaybackEngine_LoadPCAP(t *testing.T) {
	pcapFile := createTestPCAP(t, 5)

	pb := NewPlaybackEngine(&recordingSender{}, &config.CapturePlayback{FileName: pcapFile}, 0)

	packets, err := collectPlayback(pb)
	if err != nil {
		t.Fatalf("Failed to load PCAP: %v", err)
	}

	if len(packets) != 5 {
		t.Fatalf("Expected 5 packets, got %d", len(packets))
	}

	want := readPCAPFrames(t, pcapFile)
	for i, pkt := range packets {
		if !slices.Equal(pkt.Data, want[i]) {
			t.Errorf("packet %d data = %x, want %x", i, pkt.Data, want[i])
		}

		if pkt.Timestamp.IsZero() {
			t.Errorf("Packet %d has zero timestamp", i)
		}
	}
}

// TestPlaybackEngine_CalculatePacketDelay tests timing calculation.
func TestPlaybackEngine_CalculatePacketDelay(t *testing.T) {
	tests := []struct {
		name          string
		scaleTime     float64
		packetOffset  time.Duration
		expectedDelay time.Duration
	}{
		{"no scaling", 1.0, 100 * time.Millisecond, 100 * time.Millisecond},
		{"2x speed", 2.0, 100 * time.Millisecond, 200 * time.Millisecond},
		{"0.5x speed", 0.5, 100 * time.Millisecond, 50 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewPlaybackEngine(&recordingSender{}, &config.CapturePlayback{
				FileName:  "test.pcap",
				ScaleTime: tt.scaleTime,
			}, 0)

			// A frozen clock makes the expected delay exact, so the assertion
			// needs no tolerance window and cannot flake under a loaded runner.
			baseTime := time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)
			pb.now = func() time.Time { return baseTime }

			pkt := PlaybackPacket{Timestamp: baseTime.Add(tt.packetOffset)}

			delay := pb.calculatePacketDelay(pkt, baseTime, baseTime)
			if delay != tt.expectedDelay {
				t.Errorf("calculatePacketDelay = %v, want %v", delay, tt.expectedDelay)
			}
		})
	}
}

// TestPlaybackEngine_CalculatePacketDelay_PastDue tests handling of past-due packets.
func TestPlaybackEngine_CalculatePacketDelay_PastDue(t *testing.T) {
	pb := NewPlaybackEngine(&recordingSender{}, &config.CapturePlayback{
		FileName:  "test.pcap",
		ScaleTime: 1.0,
	}, 0)

	// Playback started a second ago; this packet was due 900ms back.
	nowTime := time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)
	pb.now = func() time.Time { return nowTime }

	baseTime := nowTime.Add(-1 * time.Second)
	pkt := PlaybackPacket{Timestamp: baseTime.Add(100 * time.Millisecond)}

	if delay := pb.calculatePacketDelay(pkt, baseTime, baseTime); delay != 0 {
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
// after Stop, and that each restarted run replays the whole capture.
// Regression for #464.
func TestPlaybackEngine_RestartAfterStop(t *testing.T) {
	const packetCount = 3

	pcapFile := createTestPCAP(t, packetCount)

	for i := range 3 {
		sender := &recordingSender{}
		pb := NewPlaybackEngine(sender, &config.CapturePlayback{
			FileName: pcapFile,
			RateMode: config.RateTopspeed,
		}, 0)

		if startErr := pb.Start(); startErr != nil {
			t.Fatalf("iter %d: Start failed: %v", i, startErr)
		}

		// Synchronise on the pass completing rather than on a fixed sleep.
		waitPasses(t, pb, 1, 2*time.Second)
		pb.Stop()

		if got := sender.Count(); got != packetCount {
			t.Fatalf("iter %d: sent %d frames, want %d", i, got, packetCount)
		}

		assertFramesMatchPCAP(t, sender, pcapFile)
	}
}

// TestPlaybackEngine_ConcurrentStartStop tests concurrent start/stop calls.
func TestPlaybackEngine_ConcurrentStartStop(t *testing.T) {
	pb := NewPlaybackEngine(&recordingSender{}, &config.CapturePlayback{
		FileName: createTestPCAP(t, 2),
		RateMode: config.RateTopspeed,
		LoopTime: 60_000,
	}, 0)

	if err := pb.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Concurrent Stop() calls must all return, and exactly once must actually
	// halt the engine.
	const stoppers = 10

	var wg sync.WaitGroup

	wg.Add(stoppers)

	for range stoppers {
		go func() {
			defer wg.Done()
			pb.Stop()
		}()
	}

	done := make(chan struct{})

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Concurrent Stop() calls deadlocked")
	}

	if pb.IsRunning() {
		t.Error("engine still reports running after concurrent Stop")
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

// TestPlaybackEngine_Progress_AfterPlayOnce runs a full playOnce() pass through
// a recording sender and asserts both the progress counters and the frames that
// reached the egress boundary match the PCAP that was replayed.
func TestPlaybackEngine_Progress_AfterPlayOnce(t *testing.T) {
	const packetCount = 5

	pcapFile := createTestPCAP(t, packetCount)
	sender := &recordingSender{}

	pb := NewPlaybackEngine(sender, &config.CapturePlayback{
		FileName: pcapFile,
		RateMode: config.RateTopspeed,
	}, 0)

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

	assertFramesMatchPCAP(t, sender, pcapFile)
}

// TestPlaybackEngine_PlayOnce_SendFailureIsNotCounted proves the progress
// counters track what the egress boundary accepted, not what was read: a
// sender that rejects from the third frame onward leaves PacketsSent at two
// while the pass still completes and reports the true total.
func TestPlaybackEngine_PlayOnce_SendFailureIsNotCounted(t *testing.T) {
	const packetCount = 5

	pcapFile := createTestPCAP(t, packetCount)
	sender := &recordingSender{failFrom: 3, failErr: errSendRefused}

	pb := NewPlaybackEngine(sender, &config.CapturePlayback{
		FileName: pcapFile,
		RateMode: config.RateTopspeed,
	}, 0)

	pb.playOnce()

	progress := pb.Progress()
	if progress.PacketsSent != 2 {
		t.Errorf("PacketsSent = %d, want 2 (sends 3-5 were refused)", progress.PacketsSent)
	}
	if progress.TotalPackets != packetCount {
		t.Errorf("TotalPackets = %d, want %d", progress.TotalPackets, packetCount)
	}
	if got := sender.Count(); got != 2 {
		t.Errorf("sender accepted %d frames, want 2", got)
	}
}

// errSendRefused stands in for a driver rejecting a frame.
var errSendRefused = errors.New("send refused")
