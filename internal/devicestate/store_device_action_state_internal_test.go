package devicestate

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestDeviceActionConsumptionSurvivesHistoryEviction(t *testing.T) {
	store := NewStore(Identity{Hostname: "switch-1"})
	if _, err := store.ExecuteDeviceAction(ActionReboot, "first"); err != nil {
		t.Fatal(err)
	}
	for range maxEventHistory + 1 {
		store.SaveStartup()
	}
	for _, event := range store.Events() {
		if event.Kind == EventDeviceRebooted {
			t.Fatal("fixture did not evict reboot event")
		}
	}
	recovered := NewStore(Identity{Hostname: "switch-1"})
	if err := recovered.RestoreState(store.ExportState()); err != nil {
		t.Fatal(err)
	}
	if applied, err := recovered.ExecuteDeviceAction(ActionReboot, "first"); err != nil || applied {
		t.Fatalf("history eviction rearmed action: applied=%v err=%v", applied, err)
	}
}

func TestConcurrentDeviceActionCommitsOnce(t *testing.T) {
	store := NewStore(Identity{Hostname: "switch-1"})
	var workers sync.WaitGroup
	for range 20 {
		workers.Go(func() {
			if _, err := store.ExecuteDeviceAction(ActionSTPTopologyChange, "same-phase"); err != nil {
				t.Error(err)
			}
		})
	}
	workers.Wait()
	if got := store.DeviceTelemetry().STPChanges; got != 1 {
		t.Fatalf("concurrent deliveries made %d changes", got)
	}
	if got := len(store.Events()); got != 1 {
		t.Fatalf("concurrent deliveries emitted %d events", got)
	}
	before := store.ExportState()
	if _, err := store.ExecuteDeviceAction(ActionReboot, "same-phase"); err == nil {
		t.Fatal("accepted identity reused for a different action")
	}
	if !reflect.DeepEqual(store.ExportState(), before) {
		t.Fatal("conflicting identity changed state")
	}
}

func TestInvalidDeviceActionStateIsRejectedAtomically(t *testing.T) {
	for name, corrupt := range map[string]func(*State){
		"missing consumed history": func(state *State) { state.ConsumedActions = nil },
		"mismatched consumed type": func(state *State) { state.ConsumedActions[0].Type = ActionSTPTopologyChange },
		"empty identity":           func(state *State) { state.ConsumedActions[0].ID = "" },
		"unknown action":           func(state *State) { state.ConsumedActions[0].Type = "unknown" },
		"duplicate identity": func(state *State) {
			state.ConsumedActions = append(state.ConsumedActions, state.ConsumedActions[0])
		},
		"count without timestamp": func(state *State) { state.Telemetry.STPChanges = 1 },
		"invalid reboot time":     func(state *State) { state.Telemetry.RebootedAt = time.Unix(-1, 0) },
		"invalid checkpoint time": func(state *State) {
			state.Checkpoints[0].Telemetry.STPChangedAt = time.Unix(-1, 0)
		},
	} {
		t.Run(name, func(t *testing.T) {
			store := NewStore(Identity{Hostname: "switch-1"})
			if _, err := store.ExecuteDeviceAction(ActionReboot, "phase-1"); err != nil {
				t.Fatal(err)
			}
			store.SaveCheckpoint("saved")
			before := store.ExportState()
			invalid := store.ExportState()
			corrupt(&invalid)
			if err := store.RestoreState(invalid); err == nil {
				t.Fatal("accepted corrupt action state")
			}
			if !reflect.DeepEqual(store.ExportState(), before) {
				t.Fatal("invalid restore changed state")
			}
		})
	}
}
