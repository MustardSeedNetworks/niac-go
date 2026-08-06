package behavior_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/behavior"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// testClock advances only when a test says so, so a timeline replays at the
// speed of the test rather than of the wall clock.
type testClock struct {
	mu      sync.Mutex
	now     time.Time
	waiters []chan struct{}
}

func newTestClock() *testClock {
	return &testClock{now: time.Date(2026, time.August, 6, 12, 0, 0, 0, time.UTC)}
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) Wait(ctx context.Context, d time.Duration) bool {
	if ctx.Err() != nil {
		return false
	}
	if d <= 0 {
		return true
	}
	release := make(chan struct{})
	c.mu.Lock()
	c.waiters = append(c.waiters, release)
	c.mu.Unlock()
	select {
	case <-release:
		return true
	case <-ctx.Done():
		return false
	}
}

// advance releases every waiter currently blocked, moving time forward by d.
func (c *testClock) advanceOneMinute() {
	c.mu.Lock()
	c.now = c.now.Add(time.Minute)
	waiters := c.waiters
	c.waiters = nil
	c.mu.Unlock()
	for _, release := range waiters {
		close(release)
	}
}

// waiterCount reports how many transitions are parked, so a test can wait for
// the runner to reach its next transition without sleeping.
func (c *testClock) waiterCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.waiters)
}

type clockTestTarget struct {
	mu      sync.Mutex
	applied []devicestate.FaultType
}

func (r *clockTestTarget) SetInterfaceFault(
	_, _ string, faultType devicestate.FaultType, _ int,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.applied = append(r.applied, faultType)
	return nil
}

func (r *clockTestTarget) snapshot() []devicestate.FaultType {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]devicestate.FaultType(nil), r.applied...)
}

func twoStepTimeline() []behavior.Transition {
	return []behavior.Transition{
		{
			Offset:  time.Minute,
			Actions: []behavior.Action{{Device: "sw1", Interface: "Gi0/1", Type: devicestate.FaultDiscards, Value: 5}},
		},
		{
			Offset:  2 * time.Minute,
			Actions: []behavior.Action{{Device: "sw1", Interface: "Gi0/1", Type: devicestate.FaultInterface, Value: 9}},
		},
	}
}

func waitForNextTransition(t *testing.T, clock *testClock) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if clock.waiterCount() >= 1 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("runner never parked on its next transition (waiters = %d)", clock.waiterCount())
}

func TestReplayIsDrivenByTheInjectedClock(t *testing.T) {
	// Offsets are minutes apart. Under the wall clock this test would take two
	// minutes; the point of the seam is that it does not.
	clock := newTestClock()
	target := &clockTestTarget{}
	runner := behavior.NewWithClock(target, twoStepTimeline(), clock)

	runner.Start()
	waitForNextTransition(t, clock)
	if got := len(target.snapshot()); got != 0 {
		t.Fatalf("applied %d transitions before time moved, want 0", got)
	}

	clock.advanceOneMinute()
	waitForNextTransition(t, clock)
	if got := target.snapshot(); len(got) != 1 || got[0] != devicestate.FaultDiscards {
		t.Fatalf("after first advance applied = %v, want one discards fault", got)
	}

	clock.advanceOneMinute()
	runner.Stop()
	if got := target.snapshot(); len(got) != 2 || got[1] != devicestate.FaultInterface {
		t.Fatalf("after second advance applied = %v, want discards then interface errors", got)
	}
	if state := runner.Status().State; state != "completed" {
		t.Errorf("state = %q, want completed", state)
	}
}

func TestSameTimelineAndClockReplayIdentically(t *testing.T) {
	// The equivalence M3-3 is really asking for: the same timeline driven the
	// same way produces the same applied faults, every time.
	runOnce := func() []devicestate.FaultType {
		clock := newTestClock()
		target := &clockTestTarget{}
		runner := behavior.NewWithClock(target, twoStepTimeline(), clock)
		runner.Start()
		for range 2 {
			waitForNextTransition(t, clock)
			clock.advanceOneMinute()
		}
		runner.Stop()
		return target.snapshot()
	}

	first, second := runOnce(), runOnce()
	if len(first) != len(second) {
		t.Fatalf("run lengths differ: %v vs %v", first, second)
	}
	for index := range first {
		if first[index] != second[index] {
			t.Fatalf("runs diverged at %d: %v vs %v", index, first, second)
		}
	}
}

func TestResetReplaysTheSameTimelineAgain(t *testing.T) {
	clock := newTestClock()
	target := &clockTestTarget{}
	runner := behavior.NewWithClock(target, twoStepTimeline(), clock)

	runner.Start()
	waitForNextTransition(t, clock)
	clock.advanceOneMinute()
	waitForNextTransition(t, clock)
	clock.advanceOneMinute()
	runner.Stop()

	// Start alone is a no-op once a run has finished; the runner has to be
	// reset before the same scenario can be replayed.
	runner.Start()
	if got := runner.Status().AppliedTransitions; got != 2 {
		t.Fatalf("Start after completion applied = %d, want the first run's 2", got)
	}

	runner.Reset()
	if status := runner.Status(); status.State != "idle" || status.AppliedTransitions != 0 {
		t.Fatalf("after Reset status = %+v, want idle with no applied transitions", status)
	}

	runner.Start()
	waitForNextTransition(t, clock)
	clock.advanceOneMinute()
	waitForNextTransition(t, clock)
	clock.advanceOneMinute()
	runner.Stop()

	if got := len(target.snapshot()); got != 4 {
		t.Errorf("applied %d faults across two runs, want 4", got)
	}
	if state := runner.Status().State; state != "completed" {
		t.Errorf("state after replay = %q, want completed", state)
	}
}

// haltingClock reports every wait as cut short, the way a cancelled run sees
// it. The invariant under test is that the runner then stops rather than
// racing through every remaining transition whose offset has already passed.
type haltingClock struct{ *testClock }

func (haltingClock) Wait(context.Context, time.Duration) bool { return false }

func TestRunnerStopsInsteadOfApplyingEveryOverdueTransition(t *testing.T) {
	past := []behavior.Transition{
		{Offset: 0, Actions: []behavior.Action{{Device: "sw1", Interface: "Gi0/1", Type: devicestate.FaultDiscards}}},
		{Offset: 0, Actions: []behavior.Action{{Device: "sw1", Interface: "Gi0/1", Type: devicestate.FaultInterface}}},
	}
	target := &clockTestTarget{}
	runner := behavior.NewWithClock(target, past, haltingClock{testClock: newTestClock()})

	runner.Start()
	runner.Stop()

	if got := target.snapshot(); len(got) != 0 {
		t.Errorf("applied %v after the run was cut short, want none", got)
	}
	if state := runner.Status().State; state != "stopped" {
		t.Errorf("state = %q, want stopped", state)
	}
}
