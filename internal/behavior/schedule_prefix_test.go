package behavior_test

import (
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/behavior"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestCompileMaskPreservesZeroAndExplicitReset(t *testing.T) {
	got := behavior.Compile([]config.BehaviorTimeline{{
		Name: "mask", RepeatCount: 1,
		Phases: []config.BehaviorPhase{{
			Name: "fault", Duration: time.Second, Reset: true,
			Faults: []config.BehaviorFault{
				{Device: "host", Interface: "eth0", Type: "bad_mask", PrefixBits: 0},
			},
		}},
	}})
	if len(got) != 2 || len(got[0].PrefixActions) != 1 || len(got[1].PrefixActions) != 1 {
		t.Fatalf("wrong transitions: %+v", got)
	}
	apply, clearAction := got[0].PrefixActions[0], got[1].PrefixActions[0]
	if apply.Clear || apply.Fault.PrefixBits != 0 || apply.Fault.Interface != "eth0" ||
		apply.Device != "host" {
		t.Fatalf("wrong apply: %+v", apply)
	}
	if !clearAction.Clear || clearAction.Device != apply.Device ||
		clearAction.Fault.Interface != apply.Fault.Interface {
		t.Fatalf("wrong clear: %+v", clearAction)
	}
	if len(got[0].Actions) != 0 || len(got[0].DeviceActions) != 0 {
		t.Fatal("mask leaked into numeric fault actions")
	}
}

func TestRunnerMaskUsesExplicitClear(t *testing.T) {
	target := new(recordingTarget)
	action := behavior.InterfacePrefixAction{
		Device: "host",
		Fault: devicestate.InterfacePrefixFault{
			Interface: "eth0", Type: devicestate.FaultBadMask, PrefixBits: 0,
		},
	}
	clearAction := action
	clearAction.Clear = true
	runner := behavior.New(target, []behavior.Transition{
		{PrefixActions: []behavior.InterfacePrefixAction{action}},
		{PrefixActions: []behavior.InterfacePrefixAction{clearAction}},
	})
	runner.Start()
	t.Cleanup(runner.Stop)
	waitForAppliedTransitions(t, runner, 2)
	target.mu.Lock()
	defer target.mu.Unlock()
	if len(target.prefixActions) != 2 || target.prefixActions[0] != action ||
		target.prefixActions[1] != clearAction {
		t.Fatalf("wrong prefix actions: %+v", target.prefixActions)
	}
	if len(target.actions) != 0 || len(target.deviceActions) != 0 ||
		len(target.addressActions) != 0 {
		t.Fatal("mask used another fault axis")
	}
}
