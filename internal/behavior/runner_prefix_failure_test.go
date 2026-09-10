package behavior_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/behavior"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

type prefixFaultTarget struct {
	recordingTarget

	entered chan struct{}
	release <-chan struct{}
	err     error
	calls   int
}

func (t *prefixFaultTarget) SetInterfacePrefixFault(string, devicestate.InterfacePrefixFault) error {
	return t.applyPrefix()
}

func (t *prefixFaultTarget) ClearInterfacePrefixFault(string, string, devicestate.InterfacePrefixFaultType) error {
	return t.applyPrefix()
}

func (t *prefixFaultTarget) applyPrefix() error {
	t.calls++
	t.entered <- struct{}{}
	if t.release != nil {
		<-t.release
	}
	return t.err
}

func prefixFailureTransitions(clearFault bool) []behavior.Transition {
	action := behavior.InterfacePrefixAction{
		Device: "host", Clear: clearFault,
		Fault: devicestate.InterfacePrefixFault{Interface: "eth0", Type: devicestate.FaultBadMask, PrefixBits: 0},
	}
	return []behavior.Transition{{PrefixActions: []behavior.InterfacePrefixAction{action, action}}}
}

func TestRunnerPrefixFailureStopsRemainingActions(t *testing.T) {
	for _, clearFault := range []bool{false, true} {
		target := &prefixFaultTarget{entered: make(chan struct{}, 2), err: errors.New("prefix target unavailable")}
		runner := behavior.New(target, prefixFailureTransitions(clearFault))
		t.Cleanup(runner.Stop)
		runner.Start()
		awaitPrefixSignal(t, target.entered)
		runner.Stop()
		status := runner.Status()
		if status.State != "failed" || status.LastError != target.err.Error() ||
			status.AppliedTransitions != 0 || target.calls != 1 {
			t.Fatalf("clear=%v calls=%d status=%+v", clearFault, target.calls, status)
		}
	}
}

type prefixContextClock struct{ observed chan context.Context }

func (prefixContextClock) Now() time.Time { return time.Unix(0, 0) }

func (c prefixContextClock) Wait(ctx context.Context, _ time.Duration) bool {
	c.observed <- ctx
	return true
}

func TestRunnerPrefixCancellationStopsWithinTransition(t *testing.T) {
	release := make(chan struct{})
	target := &prefixFaultTarget{entered: make(chan struct{}, 2), release: release}
	clock := prefixContextClock{observed: make(chan context.Context, 1)}
	runner := behavior.NewWithClock(target, prefixFailureTransitions(false), clock)
	released := false
	t.Cleanup(func() {
		if !released {
			close(release)
		}
		runner.Stop()
	})
	runner.Start()
	awaitPrefixSignal(t, target.entered)
	var ctx context.Context
	select {
	case ctx = <-clock.observed:
	case <-time.After(time.Second):
		t.Fatal("runner did not expose its cancellation context")
	}
	stopped := make(chan struct{})
	go func() { runner.Stop(); close(stopped) }()
	awaitPrefixSignal(t, ctx.Done())
	close(release)
	released = true
	awaitPrefixSignal(t, stopped)
	if status := runner.Status(); status.State != "stopped" || status.LastError != "" ||
		status.AppliedTransitions != 0 || target.calls != 1 {
		t.Fatalf("calls=%d status=%+v", target.calls, status)
	}
}

func awaitPrefixSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for prefix runner boundary")
	}
}
