package devicestate_test

import (
	"errors"
	"net/netip"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestUpdateInterface(t *testing.T) {
	store := stateWithInterface()

	err := store.UpdateInterface("Gi0/1", func(iface devicestate.Interface) (devicestate.Interface, error) {
		iface.AdminUp = false
		iface.OperUp = false
		return iface, nil
	})
	if err != nil {
		t.Fatalf("UpdateInterface() error = %v", err)
	}
	if store.Snapshot().Network.Interfaces[0].AdminUp {
		t.Fatal("interface remained administratively up")
	}
}

func TestUpdateInterfaceReplacesConnectedRouteAtomically(t *testing.T) {
	store := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	store.ReplaceNetwork(devicestate.Network{
		Interfaces: []devicestate.Interface{{
			Name: "Gi0/1", Address: netip.MustParsePrefix("10.0.0.1/24"), AdminUp: true, OperUp: true,
		}},
		Routes: []devicestate.Route{
			{Destination: netip.MustParsePrefix("10.0.0.0/24"), Via: "Gi0/1", Connected: true},
			{Destination: netip.MustParsePrefix("0.0.0.0/0"), Via: "Gi0/1"},
		},
	})

	err := store.UpdateInterface("Gi0/1", func(iface devicestate.Interface) (devicestate.Interface, error) {
		iface.Address = netip.MustParsePrefix("10.1.0.1/24")
		return iface, nil
	})
	if err != nil {
		t.Fatalf("UpdateInterface() error = %v", err)
	}
	routes := store.Snapshot().Network.Routes
	if len(routes) != 2 || routes[0].Connected || !routes[1].Connected ||
		routes[1].Destination != netip.MustParsePrefix("10.1.0.0/24") {
		t.Fatalf("routes = %#v", routes)
	}
}

func TestUpdateInterfaceRejectsUnknownName(t *testing.T) {
	store := stateWithInterface()
	err := store.UpdateInterface("Gi0/2", interfaceNoop)
	if !errors.Is(err, devicestate.ErrInterfaceNotFound) {
		t.Fatalf("UpdateInterface() error = %v, want ErrInterfaceNotFound", err)
	}
}

func TestUpdateInterfaceRejectsRename(t *testing.T) {
	store := stateWithInterface()
	err := store.UpdateInterface("Gi0/1", func(iface devicestate.Interface) (devicestate.Interface, error) {
		iface.Name = "Gi0/2"
		iface.Address = netip.MustParsePrefix("10.1.0.1/24")
		return iface, nil
	})
	if !errors.Is(err, devicestate.ErrInterfaceRename) {
		t.Fatalf("UpdateInterface() error = %v, want ErrInterfaceRename", err)
	}
	if got := store.Snapshot().Network.Interfaces[0].Name; got != "Gi0/1" {
		t.Fatalf("interface name = %q after rejected rename", got)
	}
}

func stateWithInterface() *devicestate.Store {
	store := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	store.ReplaceNetwork(devicestate.Network{
		Interfaces: []devicestate.Interface{{Name: "Gi0/1", AdminUp: true, OperUp: true}},
	})
	return store
}

func interfaceNoop(iface devicestate.Interface) (devicestate.Interface, error) {
	return iface, nil
}
