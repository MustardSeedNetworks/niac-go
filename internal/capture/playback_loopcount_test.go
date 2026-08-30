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

// waitLoopExit blocks until the playback goroutine has returned of its own
// accord. A bounded LoopCount ends the loop without any Stop() call, so this is
// the observable "the run is over" signal — proving the loop did not overshoot
// needs no settling sleep, because there is no goroutine left to overshoot.
func waitLoopExit(t *testing.T, pb *PlaybackEngine, timeout time.Duration) {
	t.Helper()

	exited := make(chan struct{})

	go func() {
		pb.wg.Wait()
		close(exited)
	}()

	select {
	case <-exited:
	case <-time.After(timeout):
		t.Fatalf("playback goroutine still running after %v", timeout)
	}
}

// newTestPlayback builds a playback engine over a recording sender. Replay's
// streaming, pacing, filtering and looping logic all sit above the one-method
// PacketSender boundary, so the whole suite runs without a raw socket and can
// assert on the frames that reached the boundary.
func newTestPlayback(t *testing.T, cfg *config.CapturePlayback) (*PlaybackEngine, *recordingSender) {
	t.Helper()

	sender := &recordingSender{}

	return NewPlaybackEngine(sender, cfg, 0), sender
}

// TestReplay_LoopCount_BackToBack replays exactly LoopCount passes with no
// interval (LoopTime==0) and stops.
func TestReplay_LoopCount_BackToBack(t *testing.T) {
	pb, sender := newTestPlayback(t, &config.CapturePlayback{
		FileName:  createTestPCAPFile(t, 2),
		RateMode:  config.RateTopspeed,
		LoopCount: 3, // no LoopTime → back-to-back
	})

	if err := pb.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer pb.Stop()

	// The loop ends itself after the 3rd pass; waiting for the goroutine to
	// exit is what makes "did not overshoot" a real assertion.
	waitLoopExit(t, pb, 2*time.Second)

	if got := pb.Progress().Passes; got != 3 {
		t.Fatalf("passes = %d, want exactly 3", got)
	}
	if got := sender.Count(); got != 3*2 {
		t.Fatalf("sent %d frames, want 6 (3 passes x 2 packets)", got)
	}
}

// TestReplay_LoopCount_Interval bounds an interval loop to LoopCount passes.
func TestReplay_LoopCount_Interval(t *testing.T) {
	pb, sender := newTestPlayback(t, &config.CapturePlayback{
		FileName:  createTestPCAPFile(t, 2),
		RateMode:  config.RateTopspeed,
		LoopTime:  10, // 10ms inter-pass interval
		LoopCount: 3,
	})

	if err := pb.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer pb.Stop()

	waitLoopExit(t, pb, 5*time.Second)

	if got := pb.Progress().Passes; got != 3 {
		t.Fatalf("passes = %d, want exactly 3", got)
	}
	if got := sender.Count(); got != 3*2 {
		t.Fatalf("sent %d frames, want 6 (3 passes x 2 packets)", got)
	}
}

// TestReplay_LoopCount_DefaultSingleShot confirms the historical default
// (LoopCount==0, LoopTime==0) still plays exactly once.
func TestReplay_LoopCount_DefaultSingleShot(t *testing.T) {
	pb, sender := newTestPlayback(t, &config.CapturePlayback{
		FileName: createTestPCAPFile(t, 2),
		RateMode: config.RateTopspeed,
	})

	if err := pb.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer pb.Stop()

	waitLoopExit(t, pb, 2*time.Second)

	if got := pb.Progress().Passes; got != 1 {
		t.Fatalf("passes = %d, want exactly 1 (single shot)", got)
	}
	if got := sender.Count(); got != 2 {
		t.Fatalf("sent %d frames, want 2 (one pass x 2 packets)", got)
	}
}
