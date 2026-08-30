package capture

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// TestRateLimiter_ZeroRate tests that zero rate defaults to 1.
func TestRateLimiter_ZeroRate(t *testing.T) {
	rl := NewRateLimiter(0)
	defer rl.Stop()

	if rl.packetsPerSecond != 1 {
		t.Errorf("Expected default of 1, got %d", rl.packetsPerSecond)
	}
}

// TestRateLimiter_NegativeRate tests that negative rate defaults to 1.
func TestRateLimiter_NegativeRate(t *testing.T) {
	rl := NewRateLimiter(-10)
	defer rl.Stop()

	if rl.packetsPerSecond != 1 {
		t.Errorf("Expected default of 1, got %d", rl.packetsPerSecond)
	}
}

// TestRateLimiter_StopIdempotent tests that Stop() is safe to call multiple times.
func TestRateLimiter_StopIdempotent(_ *testing.T) {
	rl := NewRateLimiter(50)

	// Stop multiple times should not panic
	rl.Stop()
	rl.Stop()
	rl.Stop()
}

// TestRateLimiter_DrainAndRefill tests that tokens refill after being drained.
func TestRateLimiter_DrainAndRefill(t *testing.T) {
	pps := 100
	rl := NewRateLimiter(pps)
	defer rl.Stop()

	// Drain all initial tokens
	for range pps {
		select {
		case <-rl.tokens:
		case <-time.After(50 * time.Millisecond):
			t.Fatal("Timed out draining initial tokens")
		}
	}

	// Wait for at least one token to refill
	time.Sleep(20 * time.Millisecond)

	select {
	case <-rl.tokens:
		// Got a refilled token
	case <-time.After(200 * time.Millisecond):
		t.Error("Token did not refill after waiting")
	}
}

// TestGetAllInterfaces tests listing all interfaces.
func TestGetAllInterfaces(t *testing.T) {
	interfaces, err := GetAllInterfaces()
	if err != nil {
		t.Fatalf("cannot enumerate interfaces: %v", err)
	}

	// Should return at least one interface on most systems
	if len(interfaces) == 0 {
		t.Fatal("no interfaces found; enumeration needs no privileges and must find at least loopback")
	}

	// Verify each interface has a name
	for _, iface := range interfaces {
		if iface.Name == "" {
			t.Error("Found interface with empty name")
		}
	}
}

// TestGetAllInterfaces_MatchesFindAllDevs tests consistency with pcap.FindAllDevs.
func TestGetAllInterfaces_MatchesFindAllDevs(t *testing.T) {
	interfaces, err := GetAllInterfaces()
	if err != nil {
		t.Fatalf("cannot enumerate interfaces: %v", err)
	}

	// Each returned interface should be findable by GetInterface
	for _, iface := range interfaces {
		found, getErr := GetInterface(iface.Name)
		if getErr != nil {
			t.Errorf("GetInterface(%s) failed but GetAllInterfaces returned it: %v", iface.Name, getErr)
			continue
		}
		if found.Name != iface.Name {
			t.Errorf("GetInterface returned different name: got %s, want %s", found.Name, iface.Name)
		}
	}
}

// TestPlaybackEngine_IsStoppedInitially tests isStopped on a new engine.
func TestPlaybackEngine_IsStoppedInitially(t *testing.T) {
	pb := &PlaybackEngine{
		stopChan: make(chan struct{}),
	}

	if pb.isStopped() {
		t.Error("New PlaybackEngine should not be stopped")
	}
}

// TestPlaybackEngine_IsStoppedAfterClose tests isStopped after closing stopChan.
func TestPlaybackEngine_IsStoppedAfterClose(t *testing.T) {
	pb := &PlaybackEngine{
		stopChan: make(chan struct{}),
	}

	close(pb.stopChan)

	if !pb.isStopped() {
		t.Error("PlaybackEngine should be stopped after closing stopChan")
	}
}

// TestPlaybackEngine_LogError tests logError at various debug levels.
func TestPlaybackEngine_LogError(t *testing.T) {
	tests := []struct {
		name       string
		debugLevel int
	}{
		{"debug 0 - no log", 0},
		{"debug 1 - logs", 1},
		{"debug 3 - logs", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			pb := &PlaybackEngine{
				debugLevel: tt.debugLevel,
				stopChan:   make(chan struct{}),
			}

			// Should not panic at any debug level
			pb.logError("test error", "key", "value")
		})
	}
}

// TestPlaybackEngine_LogBasic tests logBasic at various debug levels.
func TestPlaybackEngine_LogBasic(t *testing.T) {
	tests := []struct {
		name       string
		debugLevel int
	}{
		{"debug 0 - no log", 0},
		{"debug 1 - no log", 1},
		{"debug 2 - logs", 2},
		{"debug 3 - logs", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			pb := &PlaybackEngine{
				debugLevel: tt.debugLevel,
				stopChan:   make(chan struct{}),
			}

			// Should not panic at any debug level
			pb.logBasic("test message", "packets", 42)
		})
	}
}

// TestPlaybackEngine_LogPlaybackComplete tests completion logging.
func TestPlaybackEngine_LogPlaybackComplete(t *testing.T) {
	tests := []struct {
		name       string
		debugLevel int
	}{
		{"below threshold - no log", 1},
		{"at threshold - logs", 2},
		{"above threshold - logs", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			pb := &PlaybackEngine{
				debugLevel: tt.debugLevel,
				stopChan:   make(chan struct{}),
			}

			// Should not panic
			pb.logPlaybackComplete(time.Now().Add(-1*time.Second), 100)
		})
	}
}

// TestPlaybackEngine_CalculatePacketDelay_NoScaling tests delay calculation without scaling.
func TestPlaybackEngine_CalculatePacketDelay_NoScaling(t *testing.T) {
	pb := &PlaybackEngine{
		config: &config.CapturePlayback{
			FileName:  "test.pcap",
			ScaleTime: 0, // No scaling (ScaleTime == 0 means no adjustment)
		},
		stopChan: make(chan struct{}),
	}

	baseTime := time.Now().Add(100 * time.Millisecond) // Future start time
	firstPacketTime := baseTime
	packetTime := baseTime.Add(50 * time.Millisecond)

	pkt := PlaybackPacket{
		Timestamp: packetTime,
	}

	delay := pb.calculatePacketDelay(pkt, baseTime, firstPacketTime)

	// With ScaleTime == 0, the condition (> 0 && != 1.0) is false, so no scaling
	// Delay should be ~50ms since target is 50ms after start
	if delay < 0 {
		t.Error("Delay should not be negative")
	}
}

// TestPlaybackEngine_CalculatePacketDelay_FirstPacket tests that first packet has zero delay.
func TestPlaybackEngine_CalculatePacketDelay_FirstPacket(t *testing.T) {
	pb := &PlaybackEngine{
		config: &config.CapturePlayback{
			FileName:  "test.pcap",
			ScaleTime: 1.0,
		},
		stopChan: make(chan struct{}),
	}

	now := time.Now()
	pkt := PlaybackPacket{
		Timestamp: now,
	}

	delay := pb.calculatePacketDelay(pkt, now, now)

	// First packet has no relative delay from itself
	if delay > 10*time.Millisecond {
		t.Errorf("First packet should have ~0 delay, got %v", delay)
	}
}

// TestPlaybackEngine_WaitForPacketTiming_Stopped tests timing wait when stopped.
func TestPlaybackEngine_WaitForPacketTiming_Stopped(t *testing.T) {
	pb := &PlaybackEngine{
		config: &config.CapturePlayback{
			FileName:  "test.pcap",
			ScaleTime: 1.0,
		},
		stopChan: make(chan struct{}),
	}

	// Close the stop channel
	close(pb.stopChan)

	now := time.Now()
	futurePacket := PlaybackPacket{
		Timestamp: now.Add(10 * time.Second), // Would normally wait a long time
	}

	// Should return false immediately because stop was signaled
	result := pb.waitForPacketTiming(futurePacket, now, now)
	if result {
		t.Error("waitForPacketTiming should return false when stopped")
	}
}

// TestPlaybackEngine_WaitForPacketTiming_ZeroDelay tests timing with no delay.
func TestPlaybackEngine_WaitForPacketTiming_ZeroDelay(t *testing.T) {
	pb := &PlaybackEngine{
		config: &config.CapturePlayback{
			FileName:  "test.pcap",
			ScaleTime: 1.0,
		},
		stopChan: make(chan struct{}),
	}

	now := time.Now()
	pkt := PlaybackPacket{
		Timestamp: now, // Same as start, no delay
	}

	result := pb.waitForPacketTiming(pkt, now, now)
	if !result {
		t.Error("waitForPacketTiming should return true for zero delay")
	}
}

// TestPlaybackEngine_DoubleStart tests that starting twice returns error.
func TestPlaybackEngine_DoubleStart(t *testing.T) {
	// Create a temp file so Start() passes the file existence check
	tmpFile, err := os.CreateTemp(t.TempDir(), "niac-test-*.pcap")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	pb := &PlaybackEngine{
		config: &config.CapturePlayback{
			FileName: tmpFile.Name(),
		},
		stopChan: make(chan struct{}),
	}

	// Manually set running to simulate already running
	pb.mu.Lock()
	pb.running = true
	pb.mu.Unlock()

	err = pb.Start()
	if !errors.Is(err, ErrPlaybackAlreadyRunning) {
		t.Errorf("Expected ErrPlaybackAlreadyRunning, got: %v", err)
	}
}

// TestPlaybackEngine_StartNilConfig tests starting with nil config.
func TestPlaybackEngine_StartNilConfig(t *testing.T) {
	pb := &PlaybackEngine{
		config:   nil,
		stopChan: make(chan struct{}),
	}

	err := pb.Start()
	if !errors.Is(err, ErrNoPlaybackConfiguration) {
		t.Errorf("Expected ErrNoPlaybackConfiguration, got: %v", err)
	}
}

// TestPlaybackPacket_ZeroTimestamp tests PlaybackPacket with zero timestamp.
func TestPlaybackPacket_ZeroTimestamp(t *testing.T) {
	pkt := PlaybackPacket{}

	if !pkt.Timestamp.IsZero() {
		t.Error("Expected zero timestamp")
	}
}

// TestPlaybackPacket_NilData tests PlaybackPacket with nil data.
func TestPlaybackPacket_NilData(t *testing.T) {
	pkt := PlaybackPacket{}

	if pkt.Data != nil {
		t.Error("Expected nil data")
	}
	if len(pkt.Data) != 0 {
		t.Error("Expected zero length data")
	}
}

// TestPlaybackEngine_GetConfig_Nil tests GetConfig with nil config.
func TestPlaybackEngine_GetConfig_Nil(t *testing.T) {
	pb := &PlaybackEngine{
		config:   nil,
		stopChan: make(chan struct{}),
	}

	if pb.GetConfig() != nil {
		t.Error("Expected nil config")
	}
}

// TestPlaybackEngine_IsRunning_ConcurrentAccess tests IsRunning is safe concurrently.
func TestPlaybackEngine_IsRunning_ConcurrentAccess(t *testing.T) {
	pb := &PlaybackEngine{
		stopChan: make(chan struct{}),
	}

	done := make(chan struct{})

	// Read running state from multiple goroutines
	for range 10 {
		go func() {
			_ = pb.IsRunning()
			done <- struct{}{}
		}()
	}

	for range 10 {
		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("Concurrent IsRunning calls deadlocked")
		}
	}
}

// TestNewPlaybackEngine_WithConfig tests creating playback engine with various configs.
func TestNewPlaybackEngine_WithConfig(t *testing.T) {
	tests := []struct {
		name       string
		config     *config.CapturePlayback
		debugLevel int
	}{
		{
			name:       "basic config",
			config:     &config.CapturePlayback{FileName: "test.pcap"},
			debugLevel: 0,
		},
		{
			name:       "config with scaling",
			config:     &config.CapturePlayback{FileName: "test.pcap", ScaleTime: 2.0},
			debugLevel: 1,
		},
		{
			name:       "config with loop",
			config:     &config.CapturePlayback{FileName: "test.pcap", LoopTime: 1000},
			debugLevel: 3,
		},
		{
			name:       "nil config",
			config:     nil,
			debugLevel: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Passing nil engine is fine since we're not calling Start()
			pb := NewPlaybackEngine(nil, tt.config, tt.debugLevel)
			if pb == nil {
				t.Fatal("NewPlaybackEngine returned nil")
			}
			if pb.config != tt.config {
				t.Error("Config not set correctly")
			}
			if pb.debugLevel != tt.debugLevel {
				t.Errorf("Debug level: got %d, want %d", pb.debugLevel, tt.debugLevel)
			}
			if pb.stopChan == nil {
				t.Error("stopChan not initialized")
			}
			if pb.IsRunning() {
				t.Error("New engine should not be running")
			}
		})
	}
}

// TestErrNoMACAddressFound tests the sentinel error.
func TestErrNoMACAddressFound(t *testing.T) {
	if ErrNoMACAddressFound.Error() != "no MAC address found for interface" {
		t.Errorf("Unexpected error message: %v", ErrNoMACAddressFound)
	}
}

// TestErrInterfaceNotFound tests the sentinel error.
func TestErrInterfaceNotFound_Message(t *testing.T) {
	if ErrInterfaceNotFound.Error() != "interface not found" {
		t.Errorf("Unexpected error message: %v", ErrInterfaceNotFound)
	}
}

// TestErrPlaybackSentinels tests playback sentinel errors.
func TestErrPlaybackSentinels(t *testing.T) {
	if ErrNoPlaybackConfiguration.Error() != "no playback configuration provided" {
		t.Errorf("Unexpected error: %v", ErrNoPlaybackConfiguration)
	}
	if ErrPlaybackAlreadyRunning.Error() != "playback already running" {
		t.Errorf("Unexpected error: %v", ErrPlaybackAlreadyRunning)
	}
}

// TestPlaybackEngine_StopNotRunning tests Stop when not running.
func TestPlaybackEngine_StopNotRunning(_ *testing.T) {
	pb := &PlaybackEngine{
		stopChan: make(chan struct{}),
	}

	// Should not panic when stopping a non-running engine
	pb.Stop()
}
