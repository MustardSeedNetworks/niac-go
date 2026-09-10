package templates_test

import (
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
	tmpl "github.com/MustardSeedNetworks/niac-go/internal/templates"
)

func TestResourcePressureTemplate(t *testing.T) {
	template, err := tmpl.Get("resource-pressure")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadYAMLBytes([]byte(template.Content))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Devices) != 2 || len(cfg.BehaviorTimelines) != 1 {
		t.Fatal("missing peer or timeline")
	}
	if validation := config.NewValidator("resource-pressure").Validate(cfg); validation.HasErrors() {
		t.Fatal(validation)
	}
	for _, device := range cfg.Devices {
		state := devicestate.NewStore(devicestate.Identity{Hostname: device.Name})
		agent := snmp.NewAgentWithState(&device, state, snmp.AgentOptions{})
		for _, mib := range device.SNMPConfig.AddMibs {
			if addErr := agent.AddMib(mib.OID, mib.Type, mib.Value); addErr != nil {
				t.Fatal(addErr)
			}
		}
		agent.Reindex()
		checkResourceInventory(t, agent)
		checkResourceTemplateValues(t, agent, []int{18, 22, 1048576, 16777216})
		if device.Name == "DEMO-APP01" {
			for _, fault := range cfg.BehaviorTimelines[0].Phases[0].Faults {
				if setErr := state.SetDeviceFault(devicestate.DeviceFaultType(fault.Type), fault.Value); setErr != nil {
					t.Fatal(setErr)
				}
			}
			checkResourceTemplateValues(t, agent, []int{95, 95, 3774873, 63753420})
			state.ClearDeviceFaults()
			checkResourceTemplateValues(t, agent, []int{18, 22, 1048576, 16777216})
		}
	}
}

func checkResourceInventory(t *testing.T, agent *snmp.Agent) {
	t.Helper()
	for oid, want := range map[string]string{
		"1.3.6.1.2.1.25.3.2.1.2.1":  "1.3.6.1.2.1.25.3.1.3",
		"1.3.6.1.2.1.25.3.2.1.2.2":  "1.3.6.1.2.1.25.3.1.3",
		"1.3.6.1.2.1.25.2.3.1.2.10": "1.3.6.1.2.1.25.2.1.2",
		"1.3.6.1.2.1.25.2.3.1.2.20": "1.3.6.1.2.1.25.2.1.4",
	} {
		value, err := agent.HandleGet(oid)
		if err != nil || value == nil || value.Type != gosnmp.ObjectIdentifier || value.Value != want {
			t.Fatalf("inventory %s = %#v (%v), want OID %s", oid, value, err, want)
		}
	}
}

func checkResourceTemplateValues(t *testing.T, agent *snmp.Agent, want []int) {
	t.Helper()
	for i, oid := range []string{
		"1.3.6.1.2.1.25.3.3.1.2.1", "1.3.6.1.2.1.25.3.3.1.2.2",
		"1.3.6.1.2.1.25.2.3.1.6.10", "1.3.6.1.2.1.25.2.3.1.6.20",
	} {
		value, err := agent.HandleGet(oid)
		if err != nil || value == nil || value.Type != gosnmp.Integer ||
			gosnmp.ToBigInt(value.Value).Int64() != int64(want[i]) {
			t.Fatalf("%s = %#v (%v), want %d", oid, value, err, want[i])
		}
	}
}
