package devicestate_test

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestStoreInterfaceFaultsAreIndependent(t *testing.T) {
	store := faultStore()

	if err := store.SetInterfaceFault("Gi0/1", devicestate.FaultFCS, 25); err != nil {
		t.Fatalf("SetInterfaceFault(FCS) error = %v", err)
	}
	if err := store.SetInterfaceFault("Gi0/1", devicestate.FaultDiscards, 40); err != nil {
		t.Fatalf("SetInterfaceFault(discards) error = %v", err)
	}

	faults := store.Snapshot().Faults
	if len(faults) != 2 {
		t.Fatalf("fault count = %d, want 2", len(faults))
	}
	if faults[0].Type != devicestate.FaultFCS || faults[0].Value != 25 {
		t.Fatalf("first fault = %#v", faults[0])
	}
	if faults[1].Type != devicestate.FaultDiscards || faults[1].Value != 40 {
		t.Fatalf("second fault = %#v", faults[1])
	}
	if got := store.Snapshot().Version; got != 4 {
		t.Fatalf("version = %d, want 4", got)
	}
}

func TestStoreInterfaceFaultRequiresAuthoredInterface(t *testing.T) {
	store := faultStore()

	err := store.SetInterfaceFault("Gi0/9", devicestate.FaultFCS, 10)
	if !errors.Is(err, devicestate.ErrInterfaceNotFound) {
		t.Fatalf("SetInterfaceFault() error = %v, want ErrInterfaceNotFound", err)
	}
	if got := store.Snapshot().Version; got != 2 {
		t.Fatalf("version = %d, want unchanged 2", got)
	}
}

func TestStoreInterfaceFaultZeroClearsOnlyNamedFault(t *testing.T) {
	store := faultStore()
	for _, fault := range []struct {
		faultType devicestate.FaultType
		value     int
	}{
		{devicestate.FaultFCS, 25},
		{devicestate.FaultDiscards, 40},
	} {
		if err := store.SetInterfaceFault("Gi0/1", fault.faultType, fault.value); err != nil {
			t.Fatalf("SetInterfaceFault(%s) error = %v", fault.faultType, err)
		}
	}

	if err := store.SetInterfaceFault("Gi0/1", devicestate.FaultFCS, 0); err != nil {
		t.Fatalf("SetInterfaceFault(clear) error = %v", err)
	}

	faults := store.Snapshot().Faults
	if len(faults) != 1 || faults[0].Type != devicestate.FaultDiscards {
		t.Fatalf("faults after clear = %#v", faults)
	}
}

func TestStoreClearInterfaceFaults(t *testing.T) {
	store := faultStore()
	for _, name := range []string{"Gi0/1", "Gi0/2"} {
		if err := store.SetInterfaceFault(name, devicestate.FaultInterface, 10); err != nil {
			t.Fatalf("SetInterfaceFault(%s) error = %v", name, err)
		}
	}

	if err := store.ClearInterfaceFaults("Gi0/1"); err != nil {
		t.Fatalf("ClearInterfaceFaults() error = %v", err)
	}
	if faults := store.Snapshot().Faults; len(faults) != 1 || faults[0].Interface != "Gi0/2" {
		t.Fatalf("faults after interface clear = %#v", faults)
	}

	store.ClearAllFaults()
	if faults := store.Snapshot().Faults; len(faults) != 0 {
		t.Fatalf("faults after clear all = %#v", faults)
	}
}

func faultStore() *devicestate.Store {
	store := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	store.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{
		{Name: "Gi0/1", AdminUp: true, OperUp: true},
		{Name: "Gi0/2", AdminUp: true, OperUp: true},
	}})
	return store
}
