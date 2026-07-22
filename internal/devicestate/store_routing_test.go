package devicestate_test

import (
	"errors"
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
