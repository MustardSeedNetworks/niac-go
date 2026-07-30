package protocols

import (
	"errors"
	"net/netip"
	"runtime"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/behavior"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

type reloadBlockingTarget struct {
	stack   *Stack
	entered chan struct{}
	release chan struct{}
}

func (t *reloadBlockingTarget) SetInterfaceFault(string, string, devicestate.FaultType, int) error {
	close(t.entered)
	<-t.release
	t.stack.reloadMu.RLock()
	_ = t.stack.fabric
	t.stack.reloadMu.RUnlock()
	return nil
}

func TestSafeReloadDoesNotWaitForBehaviorWhileHoldingReloadLock(t *testing.T) {
	cfg, topology, _ := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)
	target := &reloadBlockingTarget{
		stack: stack, entered: make(chan struct{}), release: make(chan struct{}),
	}
	stack.behaviorRunner = behavior.New(target, []behavior.Transition{{
		Actions: []behavior.Action{{Type: devicestate.FaultFCS}},
	}})
	stack.behaviorRunner.Start()
	<-target.entered

	replacement, _, _ := forwardingFixture(t)
	done := make(chan error, 1)
	go func() { done <- stack.ReloadConfig(replacement) }()
	deadline := time.Now().Add(time.Second)
	for stack.reloadLifecycleMu.TryLock() {
		stack.reloadLifecycleMu.Unlock()
		if time.Now().After(deadline) {
			t.Fatal("ReloadConfig() did not acquire the lifecycle lock")
		}
		runtime.Gosched()
	}
	close(target.release)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("ReloadConfig() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ReloadConfig() deadlocked with a behavior transition")
	}
}

func TestRejectedReloadPreservesBehaviorSchedulePosition(t *testing.T) {
	cfg, topology, _ := forwardingFixture(t)
	cfg.BehaviorTimelines = []config.BehaviorTimeline{{
		Name: "degradation", RepeatCount: 1,
		Phases: []config.BehaviorPhase{{
			Name: "active", Duration: time.Second, Reset: true,
			Faults: []config.BehaviorFault{{
				Device: "server", Interface: "eth0", Type: "fcs_errors", Value: 5,
			}},
		}},
	}}
	stack := newStack(lifecycleCapture{}, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)
	if err := stack.Start(); err != nil {
		t.Fatal(err)
	}
	defer stack.Stop()
	waitForBehaviorTransition(t, stack)
	startedAt := stack.BehaviorStatus().StartedAt

	replacement, _, _ := forwardingFixture(t)
	replacement.Devices[1].Interfaces[0].Address = "10.20.0.1/24"
	if err := stack.ReloadConfig(replacement); !errors.Is(err, ErrUnsafeFabricReload) {
		t.Fatalf("ReloadConfig() error = %v, want %v", err, ErrUnsafeFabricReload)
	}
	if got := stack.BehaviorStatus().StartedAt; got != startedAt {
		t.Fatalf("behavior schedule restarted at %s, want %s", got, startedAt)
	}
}

func waitForBehaviorTransition(t *testing.T, stack *Stack) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for stack.BehaviorStatus().AppliedTransitions == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if stack.BehaviorStatus().AppliedTransitions == 0 {
		t.Fatal("behavior schedule did not start")
	}
}

func TestReloadConfigRecompilesRoutedFabric(t *testing.T) {
	cfg, topology, routerMAC := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)

	replacement, _, _ := forwardingFixture(t)
	replacement.Devices[1].Interfaces[0].Address = "10.20.0.20/24"
	if err := stack.ReloadConfig(replacement); err != nil {
		t.Fatalf("ReloadConfig(): %v", err)
	}

	if _, ok := stack.fabric.resolveIPv4(netip.MustParseAddr("10.20.0.10"), routerMAC); ok {
		t.Fatal("stale routed endpoint survived reload")
	}
	resolution, ok := stack.fabric.resolveIPv4(netip.MustParseAddr("10.20.0.20"), routerMAC)
	if !ok || resolution.device != &replacement.Devices[1] {
		t.Fatalf("replacement resolution = %#v, want replacement server", resolution)
	}
}

func TestReloadConfigRejectsUnsafeRoutedReplacementTransactionally(t *testing.T) {
	cfg, topology, routerMAC := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)

	replacement, _, _ := forwardingFixture(t)
	replacement.Devices[1].Interfaces[0].Address = "10.20.0.1/24"
	err := stack.ReloadConfig(replacement)
	if !errors.Is(err, ErrUnsafeFabricReload) {
		t.Fatalf("ReloadConfig() error = %v, want %v", err, ErrUnsafeFabricReload)
	}

	resolution, ok := stack.fabric.resolveIPv4(netip.MustParseAddr("10.20.0.10"), routerMAC)
	if !ok || resolution.device != &cfg.Devices[1] {
		t.Fatalf("original resolution after rejected reload = %#v", resolution)
	}
}
