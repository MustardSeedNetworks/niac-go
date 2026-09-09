package devicestate_test

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// TestPoELossTakesTheCarrier asserts the physical consequence: a powered device
// with no power stops linking, so the port reports operationally down the same
// way an unplugged cable does. What separates the two is POWER-ETHERNET-MIB,
// not the interface, which is why both faults land here identically.
func TestPoELossTakesTheCarrier(t *testing.T) {
	store := poeFaultStore()

	if err := store.SetInterfaceFault("Gi0/1", devicestate.FaultPoELoss, 1); err != nil {
		t.Fatalf("SetInterfaceFault(poe_loss) error = %v", err)
	}

	interfaces := store.Snapshot().Network.Interfaces
	powered, unpowered := interfaces[0], interfaces[1]
	if powered.CarrierUp || powered.OperUp {
		t.Errorf("Gi0/1 carrier=%t oper=%t, want both down under poe_loss",
			powered.CarrierUp, powered.OperUp)
	}
	if !unpowered.CarrierUp || !unpowered.OperUp {
		t.Errorf("Gi0/2 carrier=%t oper=%t, want both up: the fault is per port",
			unpowered.CarrierUp, unpowered.OperUp)
	}
}

func TestPoELossClearsRestoresTheCarrier(t *testing.T) {
	store := poeFaultStore()

	if err := store.SetInterfaceFault("Gi0/1", devicestate.FaultPoELoss, 1); err != nil {
		t.Fatalf("SetInterfaceFault(arm) error = %v", err)
	}
	if err := store.SetInterfaceFault("Gi0/1", devicestate.FaultPoELoss, 0); err != nil {
		t.Fatalf("SetInterfaceFault(clear) error = %v", err)
	}

	restored := store.Snapshot().Network.Interfaces[0]
	if !restored.CarrierUp || !restored.OperUp {
		t.Errorf("Gi0/1 carrier=%t oper=%t after clear, want both up",
			restored.CarrierUp, restored.OperUp)
	}
}

// TestPoELossIsInTheInterfaceCatalog keeps the operator-facing label and the
// wire value in step: the API and CLI address the fault by label.
func TestPoELossIsInTheInterfaceCatalog(t *testing.T) {
	if got := devicestate.FaultPoELoss.Label(); got != "PoE Loss" {
		t.Errorf("FaultPoELoss.Label() = %q, want %q", got, "PoE Loss")
	}
	faultType, ok := devicestate.ParseFaultLabel("PoE Loss")
	if !ok || faultType != devicestate.FaultPoELoss {
		t.Errorf("ParseFaultLabel(\"PoE Loss\") = %q, %t", faultType, ok)
	}
}

func poeFaultStore() *devicestate.Store {
	store := devicestate.NewStore(devicestate.Identity{Hostname: "closet-1"})
	store.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{
		{Name: "Gi0/1", AdminUp: true, OperUp: true, CarrierUp: true},
		{Name: "Gi0/2", AdminUp: true, OperUp: true, CarrierUp: true},
	}})

	return store
}
