package behavior_test

import (
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/behavior"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

type recordingTarget struct {
	mu      sync.Mutex
	actions []behavior.Action
}

func (t *recordingTarget) SetInterfaceFault(
	device, iface string,
	faultType devicestate.FaultType,
	value int,
) error {
	t.mu.Lock()
	t.actions = append(t.actions, behavior.Action{
		Device: device, Interface: iface, Type: faultType, Value: value,
	})
	t.mu.Unlock()
	return nil
}

func TestRunnerAppliesTransitionsAndCompletes(t *testing.T) {
	target := new(recordingTarget)
	runner := behavior.New(target, []behavior.Transition{
		{Offset: 0, Phases: []string{"test: apply"}, Actions: []behavior.Action{{
			Device: "switch-1", Interface: "Gi0/1", Type: devicestate.FaultFCS, Value: 10,
		}}},
		{Offset: 5 * time.Millisecond, Phases: []string{"test: reset"}, Actions: []behavior.Action{{
			Device: "switch-1", Interface: "Gi0/1", Type: devicestate.FaultFCS, Value: 0,
		}}},
	})
	runner.Start()
	deadline := time.Now().Add(time.Second)
	for runner.Status().State != "completed" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	status := runner.Status()
	if status.State != "completed" || status.AppliedTransitions != 2 || status.TotalTransitions != 2 {
		t.Fatalf("Status() = %+v", status)
	}
	target.mu.Lock()
	defer target.mu.Unlock()
	if len(target.actions) != 2 || target.actions[0].Value != 10 || target.actions[1].Value != 0 {
		t.Fatalf("actions = %+v", target.actions)
	}
}

func TestRunnerReplaysIdenticallyAfterSimulationRestart(t *testing.T) {
	transitions := []behavior.Transition{
		{Offset: 0, Phases: []string{"exercise: degrade"}, Actions: []behavior.Action{
			{Device: "switch-1", Interface: "Gi0/48", Type: devicestate.FaultUtilization, Value: 90},
			{Device: "switch-1", Interface: "Gi0/48", Type: devicestate.FaultDiscards, Value: 8},
		}},
		{Offset: time.Millisecond, Phases: []string{"exercise: reset"}, Actions: []behavior.Action{
			{Device: "switch-1", Interface: "Gi0/48", Type: devicestate.FaultUtilization, Value: 0},
			{Device: "switch-1", Interface: "Gi0/48", Type: devicestate.FaultDiscards, Value: 0},
		}},
	}
	first := runToCompletion(t, transitions)
	second := runToCompletion(t, transitions)
	if len(first) != len(second) {
		t.Fatalf("replay action counts differ: %d != %d", len(first), len(second))
	}
	for index := range first {
		if first[index] != second[index] {
			t.Fatalf("replay action %d differs: %+v != %+v", index, first[index], second[index])
		}
	}
}

func runToCompletion(t *testing.T, transitions []behavior.Transition) []behavior.Action {
	t.Helper()
	target := new(recordingTarget)
	runner := behavior.New(target, transitions)
	runner.Start()
	deadline := time.Now().Add(time.Second)
	for runner.Status().State != "completed" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if state := runner.Status().State; state != "completed" {
		t.Fatalf("replay state = %q, want completed", state)
	}
	target.mu.Lock()
	defer target.mu.Unlock()
	return append([]behavior.Action(nil), target.actions...)
}
