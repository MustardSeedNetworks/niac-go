package scenario_test

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// The access layer is the pack's PSE. Assert on the generated pack rather than
// on the spec helper: what a tester walks is the loaded config, and the budget
// has to survive generation, YAML and the loader to get there.
func TestHospitalPackAccessSwitchesAuthorAPoEBudget(t *testing.T) {
	devices := hospitalDevices(t)

	powered := 0
	for index := range devices {
		device := &devices[index]
		if device.Properties["role"] != "access" {
			if device.PoEConfig != nil {
				t.Errorf("%s (%s) supplies PoE; only the access layer should",
					device.Name, device.Properties["role"])
			}

			continue
		}
		powered++
		if device.PoEConfig == nil {
			t.Errorf("%s is an access switch that supplies no PoE", device.Name)

			continue
		}
		if device.PoEConfig.BudgetWatts <= 0 {
			t.Errorf("%s PoE budget = %d", device.Name, device.PoEConfig.BudgetWatts)
		}
	}
	if powered == 0 {
		t.Fatal("the hospital pack has no access switches")
	}
}

// The budget has to cover what the pack actually hangs off each switch, or the
// pack ships an authoring error: the validator reports over-subscription, and a
// tester would see ports that never come up.
func TestHospitalPackPoEBudgetCoversItsPoweredDevices(t *testing.T) {
	devices := hospitalDevices(t)
	cfg := &config.Config{Devices: devices}

	result := config.NewValidator("hospital.yaml").Validate(cfg)
	for _, err := range result.Errors {
		t.Errorf("generated hospital pack is invalid: %s: %s", err.Field, err.Message)
	}

	draws := map[string]int{}
	for index := range devices {
		draws[devices[index].Name] = config.PoEDrawTenthWatts(&devices[index])
	}
	drawing := 0
	for index := range devices {
		device := &devices[index]
		if device.PoEConfig == nil {
			continue
		}
		consumed := 0
		for _, trunk := range device.TrunkPorts {
			consumed += draws[trunk.RemoteDevice]
		}
		if consumed > 0 {
			drawing++
		}
	}
	if drawing == 0 {
		t.Fatal("no access switch has a powered device behind it: " +
			"the budget assertion would be vacuous")
	}
}
