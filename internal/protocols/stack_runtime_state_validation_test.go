package protocols

import (
	"errors"
	"reflect"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// A device with no record has no runtime state to lose: it starts from the
// scenario, and recovery carries on (a scenario can gain a device).
func TestRestoreDeviceStatesStartsADeviceWithoutARecordFromTheScenario(t *testing.T) {
	stack := runtimeStateTestStack(t)
	before := stack.ExportDeviceStates()
	discarded, err := stack.RestoreDeviceStates(map[string]devicestate.State{})
	if err != nil || len(discarded) != 0 {
		t.Fatalf("RestoreDeviceStates() = %v, %v", discarded, err)
	}
	if !reflect.DeepEqual(before, stack.ExportDeviceStates()) {
		t.Fatal("a device without a record did not keep its scenario state")
	}
}

// One record that no longer validates costs that device its runtime state, not
// the session: the others are restored and it starts from the scenario (#2481).
func TestRestoreDeviceStatesDiscardsOnlyTheInvalidRecord(t *testing.T) {
	stack := runtimeStateTestStack(t)
	other := &config.Device{Name: "z-invalid"}
	stack.registerDeviceState(other, NewDeviceTable())
	before := stack.ExportDeviceStates()
	states := stack.ExportDeviceStates()
	valid := states["state-router"]
	valid.Running.Identity.Hostname = "restored"
	states["state-router"] = valid
	invalid := states["z-invalid"]
	invalid.Version = 0
	states["z-invalid"] = invalid

	discarded, err := stack.RestoreDeviceStates(states)
	if err != nil {
		t.Fatalf("RestoreDeviceStates() error = %v", err)
	}
	if len(discarded) != 1 || discarded[0].Device != "z-invalid" ||
		!errors.Is(discarded[0].Err, devicestate.ErrStateInvalid) {
		t.Fatalf("discarded = %v, want z-invalid with ErrStateInvalid", discarded)
	}
	after := stack.ExportDeviceStates()
	if after["state-router"].Running.Identity.Hostname != "restored" {
		t.Fatal("the valid record was not restored")
	}
	if !reflect.DeepEqual(before["z-invalid"], after["z-invalid"]) {
		t.Fatal("the discarded record changed its device")
	}
}
