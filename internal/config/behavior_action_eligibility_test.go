package config

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestBehaviorActionsRequireServedProtocols(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		kind                 devicestate.DeviceActionType
		snmp, v3, stp, valid bool
	}{
		{"disabled SNMP", devicestate.ActionReboot, false, false, false, false},
		{"reboot", devicestate.ActionReboot, true, false, false, true},
		{"v3 reboot", devicestate.ActionReboot, false, true, false, true},
		{"no STP", devicestate.ActionSTPTopologyChange, true, false, false, false},
		{"STP", devicestate.ActionSTPTopologyChange, true, false, true, true},
		{"v3 STP", devicestate.ActionSTPTopologyChange, false, true, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			device := Device{Name: "switch", SNMPConfig: SNMPConfig{Enabled: &tc.snmp, Community: "reader"}}
			if tc.stp {
				device.STPConfig = &STPConfig{Enabled: true}
			}
			if tc.v3 {
				device.SNMPv3Config = &SNMPv3Config{Enabled: true, Users: []SNMPv3User{{Username: "reader"}}}
			}
			cfg := &Config{
				Devices: []Device{device},
				BehaviorTimelines: []BehaviorTimeline{
					{Phases: []BehaviorPhase{{Actions: []BehaviorAction{{Device: device.Name, Type: tc.kind}}}}},
				},
			}
			if err := validateBehaviorTargets(cfg); (err == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v", tc.valid, err)
			}
		})
	}
}
