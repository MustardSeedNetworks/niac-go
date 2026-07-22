package devicestate_test

import (
	"fmt"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestEventStreamPreservesExactInterfaceTransition(t *testing.T) {
	store := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	store.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{{
		Name: "Gi0/1", AdminUp: true, OperUp: true,
	}}})
	baseline := store.Snapshot().Version
	if err := store.UpdateInterface("Gi0/1", func(iface devicestate.Interface) (devicestate.Interface, error) {
		iface.AdminUp = false
		iface.OperUp = false
		return iface, nil
	}); err != nil {
		t.Fatalf("UpdateInterface() error = %v", err)
	}

	select {
	case <-store.Changes():
	default:
		t.Fatal("state transition did not signal Changes()")
	}
	events, complete := store.EventsAfter(baseline)
	if !complete {
		t.Fatal("EventsAfter() unexpectedly reported a history gap")
	}
	if len(events) != 1 || events[0].Interface == nil || events[0].Interface.OperUp {
		t.Fatalf("EventsAfter() = %#v", events)
	}
	events[0].Interface.Name = "mutated"
	stored, _ := store.EventsAfter(baseline)
	if got := stored[0].Interface.Name; got != "Gi0/1" {
		t.Fatalf("event mutation changed store: %q", got)
	}
}

func TestEventStreamBoundsHistory(t *testing.T) {
	store := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	for index := range 1100 {
		if err := store.UpdateIdentity(func(identity devicestate.Identity) (devicestate.Identity, error) {
			identity.Hostname = fmt.Sprintf("edge-%d", index)
			return identity, nil
		}); err != nil {
			t.Fatalf("UpdateIdentity() error = %v", err)
		}
	}
	events := store.Events()
	if len(events) != 1024 || events[0].Target != "edge-76" || events[len(events)-1].Target != "edge-1099" {
		t.Fatalf("bounded events = %d, first %q, last %q", len(events), events[0].Target, events[len(events)-1].Target)
	}
}

func TestEventsAfterReportsHistoryGap(t *testing.T) {
	store := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	for index := range 1100 {
		if err := store.UpdateIdentity(func(identity devicestate.Identity) (devicestate.Identity, error) {
			identity.Hostname = fmt.Sprintf("edge-%d", index)
			return identity, nil
		}); err != nil {
			t.Fatalf("UpdateIdentity() error = %v", err)
		}
	}
	events, complete := store.EventsAfter(1)
	if complete || len(events) != 1024 || events[0].Target != "edge-76" {
		t.Fatalf("EventsAfter() = (%d events, complete %t), first = %q", len(events), complete, events[0].Target)
	}
}
