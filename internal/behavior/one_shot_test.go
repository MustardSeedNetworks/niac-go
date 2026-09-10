package behavior_test

import (
	"errors"
	"reflect"
	"testing"
	"testing/synctest"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/behavior"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func oneShotSchedule() []behavior.Transition {
	return behavior.Compile([]config.BehaviorTimeline{{
		Name: "actions", RepeatCount: 2,
		Phases: []config.BehaviorPhase{{
			Name: "change", Duration: time.Second, Reset: true,
			Actions: []config.BehaviorAction{
				{Device: "switch", Type: devicestate.ActionReboot},
				{Device: "switch", Type: devicestate.ActionSTPTopologyChange},
			},
		}},
	}})
}

func TestCompileOneShotActions(t *testing.T) {
	schedule := oneShotSchedule()
	want := []behavior.OneShotAction{
		{Device: "switch", Type: devicestate.ActionReboot, ID: "0:0:0:0"},
		{Device: "switch", Type: devicestate.ActionSTPTopologyChange, ID: "0:0:0:1"},
	}
	if len(schedule) != 3 || !reflect.DeepEqual(schedule[0].OneShotActions, want) {
		t.Fatalf("compiled schedule: %+v", schedule)
	}
	want[0].ID, want[1].ID = "0:1:0:0", "0:1:0:1"
	if !reflect.DeepEqual(schedule[1].OneShotActions, want) || len(schedule[2].OneShotActions) != 0 {
		t.Fatalf("repetition/reset: %+v", schedule)
	}
	if !reflect.DeepEqual(schedule, oneShotSchedule()) {
		t.Fatal("action identities changed on recompilation")
	}
}

type oneShotTarget struct {
	recordingTarget

	calls []behavior.OneShotAction
	fail  bool
}

func (target *oneShotTarget) ExecuteDeviceAction(device string, kind devicestate.DeviceActionType, id string) error {
	target.calls = append(target.calls, behavior.OneShotAction{Device: device, Type: kind, ID: id})
	if target.fail {
		return errors.New("action refused")
	}
	return nil
}

func TestRunnerOneShotActions(t *testing.T) {
	for _, outcome := range []string{"complete", "cancel", "fail", "reset"} {
		t.Run(outcome, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				target := &oneShotTarget{fail: outcome == "fail"}
				runner := behavior.New(target, oneShotSchedule())
				runner.Start()
				synctest.Wait()
				wantCalls, wantState := 4, "completed"
				switch outcome {
				case "cancel":
					runner.Stop()
					wantCalls, wantState = 2, "stopped"
				case "fail":
					wantCalls, wantState = 1, "failed"
				default:
					time.Sleep(3 * time.Second)
					synctest.Wait()
				}
				if got := runner.Status(); got.State != wantState || len(target.calls) != wantCalls {
					t.Fatalf("status=%+v calls=%+v", got, target.calls)
				}
				assertActionOrder(t, target.calls)
				if outcome == "fail" && runner.Status().LastError != "action refused" {
					t.Fatal("lost execution failure")
				}
				if outcome == "reset" {
					assertResetActionIdentity(t, runner, target)
				}
			})
		})
	}
}

func assertActionOrder(t *testing.T, calls []behavior.OneShotAction) {
	t.Helper()
	schedule := oneShotSchedule()
	want := append([]behavior.OneShotAction(nil), schedule[0].OneShotActions...)
	want = append(want, schedule[1].OneShotActions...)
	if !reflect.DeepEqual(calls, want[:len(calls)]) {
		t.Fatalf("execution order or identity changed: %+v", calls)
	}
}

func assertResetActionIdentity(t *testing.T, runner *behavior.Runner, target *oneShotTarget) {
	t.Helper()
	first := append([]behavior.OneShotAction(nil), target.calls...)
	runner.Reset()
	runner.Start()
	time.Sleep(3 * time.Second)
	synctest.Wait()
	if len(target.calls) != 8 || !reflect.DeepEqual(first, target.calls[4:]) {
		t.Fatalf("reset changed action identity: %+v", target.calls)
	}
}
