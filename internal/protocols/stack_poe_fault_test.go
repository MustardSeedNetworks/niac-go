package protocols

import (
	"errors"
	"net"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

func poeFaultTestStack(t *testing.T) (*Stack, *config.Device) {
	t.Helper()
	switchDevice := faultTestDevice("closet-1")
	switchDevice.Type = "switch"
	switchDevice.PoEConfig = &config.PoEConfig{BudgetWatts: 370}
	switchDevice.TrunkPorts = []config.TrunkPort{
		{Interface: "Gi0/1", RemoteDevice: "phone-1"},
	}
	phone := config.Device{
		Name: "phone-1", Type: "voip-phone", IPAddresses: []net.IP{{192, 0, 2, 50}},
		LLDPConfig: &config.LLDPConfig{Enabled: true, MED: &config.LLDPMEDConfig{
			Power: &config.LLDPMEDPower{
				DeviceType: "pd", Source: "pse", Priority: "high", ValueTenthWatts: 62,
			},
		}},
	}
	cfg := &config.Config{Devices: []config.Device{switchDevice, phone}}

	return NewStack(nil, cfg, logging.NewDebugConfig(0)), &cfg.Devices[0]
}

// A PoE fault on a port that supplies no power must be refused rather than
// stored: it would report success and then be indistinguishable from link_down.
func TestStackPoEFaultRefusesNonPSEPort(t *testing.T) {
	device := faultTestDevice("edge-1")
	cfg := &config.Config{Devices: []config.Device{device}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))

	err := stack.SetInterfaceFault("edge-1", "Gi0/1", devicestate.FaultPoELoss, 1)
	if !errors.Is(err, ErrFaultNoPSEPort) {
		t.Fatalf("error = %v, want %v", err, ErrFaultNoPSEPort)
	}
	if len(stack.ActiveInterfaceFaults()) != 0 {
		t.Errorf("refused fault was stored: %v", stack.ActiveInterfaceFaults())
	}
}

func TestStackPoEFaultArmsOnAPSEPort(t *testing.T) {
	stack, _ := poeFaultTestStack(t)

	if err := stack.SetInterfaceFault("closet-1", "Gi0/1", devicestate.FaultPoELoss, 1); err != nil {
		t.Fatalf("SetInterfaceFault(poe_loss) error = %v", err)
	}
	if got := stack.ActiveInterfaceFaults()["closet-1"]["Gi0/1"][devicestate.FaultPoELoss]; got != 1 {
		t.Fatalf("active PoE fault = %d, want 1", got)
	}
}

// F7 for poe_loss. Unlike the device-service faults, this one HAS a MIB effect,
// so the clause is a positive one: arming it must change exactly the rows that
// describe the port's power and the link it can no longer hold up, and nothing
// else in the walk the agent serves.
func TestPoEFaultPerturbsOnlyItsNamedRows(t *testing.T) {
	stack, device := poeFaultTestStack(t)
	agent := stack.snmpAgents[device].baseAgent
	if agent == nil {
		t.Fatal("device has no SNMP agent to sweep")
	}

	budget := snmp.SweepBudget(0)
	baseline := sweepAgent(t, agent, budget)
	if err := stack.SetInterfaceFault("closet-1", "Gi0/1", devicestate.FaultPoELoss, 1); err != nil {
		t.Fatalf("SetInterfaceFault(poe_loss) error = %v", err)
	}
	faulted := sweepAgent(t, agent, budget)
	if err := stack.SetInterfaceFault("closet-1", "Gi0/1", devicestate.FaultPoELoss, 0); err != nil {
		t.Fatalf("SetInterfaceFault(clear) error = %v", err)
	}
	settled := sweepAgent(t, agent, budget)

	// The same bracket the device-fault clause uses: rows that move on their
	// own (sysUpTime, the SNMP group's own request counters, traffic counters)
	// have necessarily moved by the second healthy sweep too.
	drifting := changedOIDs(baseline, settled)
	if len(faulted) != len(baseline) {
		t.Fatalf("row count changed under poe_loss: %d -> %d", len(baseline), len(faulted))
	}

	// POWER-ETHERNET-MIB reports the port faulted and its class withdrawn, the
	// chassis consumption drops by what the phone drew, and the port stops
	// linking because the phone it powered went dark.
	expected := map[string]string{
		"1.3.6.1.2.1.105.1.1.1.6.1.1":  "detection status",
		"1.3.6.1.2.1.105.1.1.1.10.1.1": "power class",
		"1.3.6.1.2.1.105.1.3.1.4.1":    "chassis consumption",
		"1.3.6.1.2.1.2.2.1.8.1":        "ifOperStatus",
	}
	changed := changedOIDs(baseline, faulted)
	for oid := range changed {
		if _, drifts := drifting[oid]; drifts {
			continue
		}
		if _, named := expected[oid]; !named {
			t.Errorf("poe_loss changed an unnamed row: %s", oid)
		}
	}
	for oid, what := range expected {
		if _, moved := changed[oid]; !moved {
			t.Errorf("poe_loss left the %s (%s) unchanged", what, oid)
		}
	}
}
