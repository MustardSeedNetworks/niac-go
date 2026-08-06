package devicestate_test

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func storeWithInterface(t *testing.T, name string) *devicestate.Store {
	t.Helper()
	store := devicestate.NewStore(devicestate.Identity{Hostname: "sw1"})
	store.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{
		{Name: name, AdminUp: true, OperUp: true},
	}})
	return store
}

const testInterface = "Gi0/1"

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
	store := storeWithInterface(t, "Gi0/1")
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
	store := storeWithInterface(t, "Gi0/1")
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
	store := storeWithInterface(t, "Gi0/1")
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
