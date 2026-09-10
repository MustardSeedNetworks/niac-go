package devicestate_test

import (
	"net/netip"
	"reflect"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestRestoreStatePreservesAllowedRuntimeMutations(t *testing.T) {
	source := newStateTestStore(t)
	if err := source.UpdateIdentity(func(identity devicestate.Identity) (devicestate.Identity, error) {
		identity.Hostname = "renamed-runtime-host"
		return identity, nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := source.UpdateInterface("eth0", func(iface devicestate.Interface) (devicestate.Interface, error) {
		iface.Address = netip.MustParsePrefix("198.51.100.5/24")
		iface.Network = "runtime-network"
		iface.Description = "new description"
		iface.OperUp = false
		return iface, nil
	}); err != nil {
		t.Fatal(err)
	}
	source.SaveStartup()
	source.SaveCheckpoint("renumbered")
	target := newStateTestStore(t)
	state := source.ExportState()
	if err := target.RestoreState(state); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state, target.ExportState()) {
		t.Fatal("restore changed valid runtime state")
	}
}

func TestRestoreStateAcceptsRetainedAndSameVersionEvents(t *testing.T) {
	source := newStateTestStore(t)
	for range 1100 {
		source.SaveStartup()
	}
	if err := source.UpdateInterface("eth0", func(iface devicestate.Interface) (devicestate.Interface, error) {
		iface.OperUp = false
		return iface, nil
	}); err != nil {
		t.Fatal(err)
	}
	source.ReloadStartup()
	state := source.ExportState()
	last := len(state.Events) - 1
	if state.Events[last].Version != state.Events[last-1].Version {
		t.Fatal("fixture lacks same-transaction events")
	}
	if err := newStateTestStore(t).RestoreState(state); err != nil {
		t.Fatalf("valid bounded history rejected: %v", err)
	}
}
