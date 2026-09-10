package protocols

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestStackDeviceActionEligibility(t *testing.T) {
	for _, tc := range []struct {
		name              string
		kind              devicestate.DeviceActionType
		v2, v3, stp, want bool
	}{
		{"reboot", devicestate.ActionReboot, true, false, false, true},
		{"v3 reboot", devicestate.ActionReboot, false, true, false, true},
		{"disabled SNMP", devicestate.ActionReboot, false, false, false, false},
		{"STP", devicestate.ActionSTPTopologyChange, true, false, true, true},
		{"v3 STP", devicestate.ActionSTPTopologyChange, false, true, true, true},
		{"disabled STP", devicestate.ActionSTPTopologyChange, true, false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			device := faultTestDevice("edge-1")
			device.Type = "switch"
			device.SNMPConfig.Enabled = &tc.v2
			device.STPConfig = &config.STPConfig{Enabled: tc.stp}
			if tc.v3 {
				device.SNMPv3Config = &config.SNMPv3Config{
					Enabled: true, Users: []config.SNMPv3User{{Username: "action-reader"}},
				}
			}
			stack := NewStack(
				nil,
				&config.Config{Devices: []config.Device{device}},
				logging.NewDebugConfig(0),
			)
			err := stack.ExecuteDeviceAction(device.Name, tc.kind, "one")
			if tc.want {
				if err != nil {
					t.Fatal(err)
				}
				if err = stack.ExecuteDeviceAction(device.Name, tc.kind, "one"); err != nil {
					t.Fatal(err)
				}
			} else if !errors.Is(err, ErrDeviceActionUnobservable) {
				t.Fatalf("error=%v", err)
			}
			assertConsumedActionCount(t, stack, tc.want)
		})
	}
}

func assertConsumedActionCount(t *testing.T, stack *Stack, consumed bool) {
	t.Helper()
	want := 0
	if consumed {
		want = 1
	}
	states := stack.ExportDeviceStates()
	if len(states) != 1 {
		t.Fatalf("device states=%d", len(states))
	}
	for _, state := range states {
		if len(state.ConsumedActions) != want {
			t.Fatalf("consumed=%v", state.ConsumedActions)
		}
	}
}

func TestStackDeviceActionRejectsInvalidTargetAndKind(t *testing.T) {
	stack := NewStack(
		nil,
		&config.Config{Devices: []config.Device{faultTestDevice("edge-1")}},
		logging.NewDebugConfig(0),
	)
	if err := stack.ExecuteDeviceAction("absent", devicestate.ActionReboot, "one"); err == nil {
		t.Fatal("accepted absent target")
	}
	if err := stack.ExecuteDeviceAction("edge-1", "invalid", "one"); !errors.Is(
		err,
		devicestate.ErrDeviceActionInvalid,
	) {
		t.Fatalf("invalid kind: %v", err)
	}
	for _, state := range stack.ExportDeviceStates() {
		if len(state.ConsumedActions) != 0 {
			t.Fatal("invalid action changed state")
		}
	}
}
