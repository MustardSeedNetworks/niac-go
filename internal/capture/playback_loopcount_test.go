package capture

import (
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// waitPasses polls until the engine reports at least want completed passes.
func waitPasses(t *testing.T, pb *PlaybackEngine, want uint64, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if pb.Progress().Passes >= want {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("passes reached %d, want %d within %v", pb.Progress().Passes, want, timeout)
}

// newLoopbackPlayback builds a playback engine over the loopback interface, or
// skips when one isn't available.
func newLoopbackPlayback(t *testing.T, cfg *config.CapturePlayback) *PlaybackEngine {
	t.Helper()

	engine, err := New(findLoopback(t), 0)
	if err != nil {
		t.Skipf("cannot create engine: %v", err)
	}
	t.Cleanup(engine.Close)

	return NewPlaybackEngine(engine, cfg, 0)
}

// TestReplay_LoopCount_BackToBack replays exactly LoopCount passes with no
// interval (LoopTime==0) and stops.
func TestReplay_LoopCount_BackToBack(t *testing.T) {
	pb := newLoopbackPlayback(t, &config.CapturePlayback{
		FileName:  createTestPCAPFile(t, 2),
		RateMode:  config.RateTopspeed,
		LoopCount: 3, // no LoopTime → back-to-back
	})

	if err := pb.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer pb.Stop()

	waitPasses(t, pb, 3, time.Second)
	// Must not overshoot: the loop exits after the 3rd pass.
	time.Sleep(50 * time.Millisecond)
	if got := pb.Progress().Passes; got != 3 {
		t.Fatalf("passes = %d after settling, want exactly 3", got)
	}
}

// TestReplay_LoopCount_Interval bounds an interval loop to LoopCount passes.
func TestReplay_LoopCount_Interval(t *testing.T) {
	pb := newLoopbackPlayback(t, &config.CapturePlayback{
		FileName:  createTestPCAPFile(t, 2),
		RateMode:  config.RateTopspeed,
		LoopTime:  10, // 10ms inter-pass interval
		LoopCount: 3,
	})

	if err := pb.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer pb.Stop()

	waitPasses(t, pb, 3, 2*time.Second)
	time.Sleep(50 * time.Millisecond)
	if got := pb.Progress().Passes; got != 3 {
		t.Fatalf("passes = %d after settling, want exactly 3", got)
	}
}

// TestReplay_LoopCount_DefaultSingleShot confirms the historical default
// (LoopCount==0, LoopTime==0) still plays exactly once.
func TestReplay_LoopCount_DefaultSingleShot(t *testing.T) {
	pb := newLoopbackPlayback(t, &config.CapturePlayback{
		FileName: createTestPCAPFile(t, 2),
		RateMode: config.RateTopspeed,
	})

	if err := pb.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer pb.Stop()

	waitPasses(t, pb, 1, time.Second)
	time.Sleep(50 * time.Millisecond)
	if got := pb.Progress().Passes; got != 1 {
		t.Fatalf("passes = %d, want exactly 1 (single shot)", got)
	}
}
