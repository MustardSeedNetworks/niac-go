package devicestate_test

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestReloadRestoresStartupConfiguration(t *testing.T) {
	store := configuredStore()
	mutateDevice(t, store, "branch-1", false)

	store.ReloadStartup()

	assertDeviceState(t, store, "edge-1", true)
}

func TestCheckpointRestoresNamedRunningConfiguration(t *testing.T) {
	store := configuredStore()
	mutateDevice(t, store, "branch-1", false)
	store.SaveCheckpoint("lesson-start")
	mutateDevice(t, store, "temporary", true)

	if err := store.RestoreCheckpoint("lesson-start"); err != nil {
		t.Fatalf("RestoreCheckpoint() error = %v", err)
	}
	assertDeviceState(t, store, "branch-1", false)
	if err := store.RestoreCheckpoint("missing"); !errors.Is(err, devicestate.ErrCheckpointNotFound) {
		t.Fatalf("RestoreCheckpoint(missing) error = %v", err)
	}
}

func TestSaveAndReloadPreserveRunningConfiguration(t *testing.T) {
	store := configuredStore()
	mutateDevice(t, store, "branch-1", false)
	store.SaveStartup()
	mutateDevice(t, store, "temporary", true)

	store.ReloadStartup()

	assertDeviceState(t, store, "branch-1", false)
}

func TestResetRestoresAuthoredConfigurationForRunningAndStartup(t *testing.T) {
	store := configuredStore()
	mutateDevice(t, store, "branch-1", false)
	store.SaveStartup()

	store.ResetAuthored()
	assertDeviceState(t, store, "edge-1", true)
	mutateDevice(t, store, "temporary", false)
	store.ReloadStartup()
	assertDeviceState(t, store, "edge-1", true)
}

func TestConfigurationChangesEmitOrderedEvents(t *testing.T) {
	store := configuredStore()
	mutateDevice(t, store, "branch-1", false)
	store.SaveStartup()
	store.ReloadStartup()
	store.EraseStartup()
	store.ResetAuthored()

	events := store.Events()
	want := []devicestate.EventKind{
		devicestate.EventNetworkInstalled,
		devicestate.EventIdentityUpdated,
		devicestate.EventInterfaceUpdated,
		devicestate.EventStartupSaved,
		devicestate.EventStartupReloaded,
		devicestate.EventStartupErased,
		devicestate.EventAuthoredReset,
	}
	if len(events) != len(want) {
		t.Fatalf("events = %#v, want %d events", events, len(want))
	}
	for index, kind := range want {
		if events[index].Kind != kind || events[index].Version == 0 {
			t.Fatalf("events[%d] = %#v, want kind %q", index, events[index], kind)
		}
	}
	if events[2].Target != "Gi0/1" {
		t.Fatalf("interface event target = %q, want Gi0/1", events[2].Target)
	}
}

func configuredStore() *devicestate.Store {
	store := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	store.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{
		{Name: "Gi0/1", AdminUp: true, OperUp: true},
	}})
	return store
}

func mutateDevice(t *testing.T, store *devicestate.Store, hostname string, interfaceUp bool) {
	t.Helper()
	if err := store.UpdateIdentity(func(identity devicestate.Identity) (devicestate.Identity, error) {
		identity.Hostname = hostname
		return identity, nil
	}); err != nil {
		t.Fatalf("UpdateIdentity() error = %v", err)
	}
	if err := store.UpdateInterface("Gi0/1", func(iface devicestate.Interface) (devicestate.Interface, error) {
		iface.AdminUp = interfaceUp
		iface.OperUp = interfaceUp
		return iface, nil
	}); err != nil {
		t.Fatalf("UpdateInterface() error = %v", err)
	}
}

func assertDeviceState(t *testing.T, store *devicestate.Store, hostname string, interfaceUp bool) {
	t.Helper()
	snapshot := store.Snapshot()
	if snapshot.Identity.Hostname != hostname || snapshot.Network.Interfaces[0].AdminUp != interfaceUp {
		t.Fatalf("state = %#v, want hostname %q interface up %t", snapshot, hostname, interfaceUp)
	}
}
