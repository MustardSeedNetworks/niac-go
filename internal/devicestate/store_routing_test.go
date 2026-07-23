package devicestate_test

import (
	"errors"
	"net/netip"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestUpdateVLANRejectsUnknownID(t *testing.T) {
	store := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	store.ReplaceNetwork(devicestate.Network{VLANs: []devicestate.VLAN{{ID: 10, Active: true}}})
	called := false

	err := store.UpdateVLAN(20, func(vlan devicestate.VLAN) devicestate.VLAN {
		called = true
		return vlan
	})
	if !errors.Is(err, devicestate.ErrVLANNotFound) {
		t.Fatalf("UpdateVLAN() error = %v, want ErrVLANNotFound", err)
	}
	if called {
		t.Fatal("UpdateVLAN() invoked callback for an unknown VLAN")
	}
}

func TestUpsertRoutePreservesConnectedRouteWithSameDestination(t *testing.T) {
	store := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	destination := netip.MustParsePrefix("192.0.2.0/24")
	store.ReplaceNetwork(devicestate.Network{Routes: []devicestate.Route{{
		Destination: destination, Via: "Gi0/1", Connected: true,
	}}})

	store.UpsertRoute(devicestate.Route{
		Destination: destination, Via: "Gi0/2", NextHop: netip.MustParseAddr("198.51.100.1"),
	})

	routes := store.Snapshot().Network.Routes
	if len(routes) != 2 || !routes[0].Connected || routes[1].Connected || routes[1].Via != "Gi0/2" {
		t.Fatalf("routes = %#v", routes)
	}
}
