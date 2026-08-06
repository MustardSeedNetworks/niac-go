package devicestate_test

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

const testInterface = "Gi0/1"

func storeWithInterface(t *testing.T) *devicestate.Store {
	t.Helper()
	store := devicestate.NewStore(devicestate.Identity{Hostname: "sw1"})
	store.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{
		{Name: testInterface, AdminUp: true, OperUp: true, CarrierUp: true},
	}})
	return store
}

func faultValue(t *testing.T, store *devicestate.Store, faultType devicestate.FaultType) (int, bool) {
	t.Helper()
	for _, fault := range store.Snapshot().Faults {
		if fault.Interface == testInterface && fault.Type == faultType {
			return fault.Value, true
		}
	}
	return 0, false
}

func TestCheckpointCapturesActiveFaults(t *testing.T) {
	// A checkpoint is meant to be a point a scenario can return to. Faults are
	// part of what a scenario is doing, so restoring one that was taken while a
	// fault was active has to bring that fault back.
	store := storeWithInterface(t)
	if err := store.SetInterfaceFault("Gi0/1", devicestate.FaultDiscards, 40); err != nil {
		t.Fatal(err)
	}
	store.SaveCheckpoint("degraded")

	if err := store.ClearInterfaceFaults("Gi0/1"); err != nil {
		t.Fatal(err)
	}
	if _, present := faultValue(t, store, devicestate.FaultDiscards); present {
		t.Fatal("fault still present after clearing it")
	}

	if err := store.RestoreCheckpoint("degraded"); err != nil {
		t.Fatal(err)
	}
	value, present := faultValue(t, store, devicestate.FaultDiscards)
	if !present {
		t.Fatal("restoring a checkpoint taken while degraded did not bring the fault back")
	}
	if value != 40 {
		t.Errorf("restored fault value = %d, want 40", value)
	}
}

func TestCheckpointRestoreClearsFaultsRaisedSince(t *testing.T) {
	// The other direction: a checkpoint taken while healthy must restore to
	// healthy, not leave a later fault running.
	store := storeWithInterface(t)
	store.SaveCheckpoint("healthy")

	if err := store.SetInterfaceFault("Gi0/1", devicestate.FaultFCS, 15); err != nil {
		t.Fatal(err)
	}
	if err := store.RestoreCheckpoint("healthy"); err != nil {
		t.Fatal(err)
	}

	if _, present := faultValue(t, store, devicestate.FaultFCS); present {
		t.Error("a fault raised after the checkpoint survived the restore")
	}
}

func TestCheckpointFaultsAreIndependentOfLaterChanges(t *testing.T) {
	// The saved copy must not alias the live fault map, or changing a fault
	// after saving would quietly rewrite the checkpoint.
	store := storeWithInterface(t)
	if err := store.SetInterfaceFault("Gi0/1", devicestate.FaultUtilization, 10); err != nil {
		t.Fatal(err)
	}
	store.SaveCheckpoint("baseline")

	if err := store.SetInterfaceFault("Gi0/1", devicestate.FaultUtilization, 90); err != nil {
		t.Fatal(err)
	}
	if err := store.RestoreCheckpoint("baseline"); err != nil {
		t.Fatal(err)
	}

	value, present := faultValue(t, store, devicestate.FaultUtilization)
	if !present {
		t.Fatal("checkpointed fault missing after restore")
	}
	if value != 10 {
		t.Errorf("restored fault value = %d, want the checkpointed 10 not the later 90", value)
	}
}

func TestLinkDownFaultTakesTheInterfaceOperationallyDown(t *testing.T) {
	// Unlike the counter-rate faults, a link-down fault has to change what the
	// interface *is*: the packet path, CLI and SNMP all gate on operational
	// state, so a fault that only moved a counter would advertise an outage
	// nothing could observe.
	store := storeWithInterface(t)
	before := store.Snapshot().Network.Interfaces[0]
	if !before.OperUp || !before.CarrierUp {
		t.Fatalf("interface started down: %+v", before)
	}

	if err := store.SetInterfaceFault(testInterface, devicestate.FaultLinkDown, 1); err != nil {
		t.Fatal(err)
	}
	down := store.Snapshot().Network.Interfaces[0]
	if down.OperUp || down.CarrierUp {
		t.Errorf("interface still up under a link-down fault: %+v", down)
	}
	if down.AdminUp != before.AdminUp {
		t.Errorf("link-down changed AdminUp to %v — a lost carrier is not an admin shutdown", down.AdminUp)
	}
}

func TestClearingLinkDownRestoresTheInterface(t *testing.T) {
	// The fault is projected rather than written through, so clearing it
	// restores the interface with no saved-state bookkeeping to get wrong.
	store := storeWithInterface(t)
	if err := store.SetInterfaceFault(testInterface, devicestate.FaultLinkDown, 1); err != nil {
		t.Fatal(err)
	}
	if err := store.ClearInterfaceFaults(testInterface); err != nil {
		t.Fatal(err)
	}

	restored := store.Snapshot().Network.Interfaces[0]
	if !restored.OperUp || !restored.CarrierUp {
		t.Errorf("interface stayed down after the fault cleared: %+v", restored)
	}
}

func TestLinkDownAtZeroLeavesTheInterfaceUp(t *testing.T) {
	// Value is the shared 0-100 fault scale; zero means no outage rather than
	// an outage of zero severity.
	store := storeWithInterface(t)
	if err := store.SetInterfaceFault(testInterface, devicestate.FaultLinkDown, 0); err != nil {
		t.Fatal(err)
	}
	if iface := store.Snapshot().Network.Interfaces[0]; !iface.OperUp {
		t.Errorf("a zero-valued link-down fault took the interface down: %+v", iface)
	}
}
