package protocols

import (
	"reflect"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestRestoreDeviceStatesRejectsMissingDevice(t *testing.T) {
	stack := runtimeStateTestStack(t)
	if err := stack.RestoreDeviceStates(map[string]devicestate.State{}); err == nil {
		t.Fatal("missing device state was accepted")
	}
}

func TestRestoreDeviceStatesValidatesAllBeforeApplying(t *testing.T) {
	stack := runtimeStateTestStack(t)
	other := &config.Device{Name: "z-invalid"}
	stack.registerDeviceState(other, NewDeviceTable())
	before := stack.ExportDeviceStates()
	states := stack.ExportDeviceStates()
	valid := states["state-router"]
	valid.Running.Identity.Hostname = "must-not-apply"
	states["state-router"] = valid
	invalid := states["z-invalid"]
	invalid.Version = 0
	states["z-invalid"] = invalid
	if err := stack.RestoreDeviceStates(states); err == nil {
		t.Fatal("invalid record was accepted")
	}
	if !reflect.DeepEqual(before, stack.ExportDeviceStates()) {
		t.Fatal("invalid second device partially restored the first")
	}
}
