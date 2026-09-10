package devicestate

import (
	"reflect"
	"testing"
	"time"
)

func transitionStore(t *testing.T) (*Store, *time.Time) {
	t.Helper()
	now := time.Date(2026, time.September, 10, 12, 0, 0, 0, time.UTC)
	store := NewStore(Identity{Hostname: "switch"})
	store.now = func() time.Time { return now }
	store.ReplaceNetwork(Network{Interfaces: []Interface{
		{Name: "eth0", AdminUp: true, OperUp: true},
		{Name: "eth1", AdminUp: true, OperUp: true},
	}})
	if !store.InterfaceLastChange("eth0").IsZero() {
		t.Fatal("expected zero last change")
	}
	if !store.InterfaceLastChange("missing").IsZero() {
		t.Fatal("expected zero last change")
	}
	return store, &now
}

func TestInterfaceLastChangeEffectiveTransitions(t *testing.T) {
	store, now := transitionStore(t)
	var last time.Time
	steps := []struct {
		name    string
		mutate  func() error
		changed bool
	}{
		{"admin and oper down", func() error {
			return store.UpdateInterface("eth0", func(iface Interface) (Interface, error) {
				iface.AdminUp, iface.OperUp = false, false
				return iface, nil
			})
		}, true},
		{"admin only", func() error {
			return store.UpdateInterface("eth0", func(iface Interface) (Interface, error) {
				iface.AdminUp = true
				return iface, nil
			})
		}, false},
		{"oper up", func() error {
			return store.UpdateInterface("eth0", func(iface Interface) (Interface, error) {
				iface.OperUp = true
				return iface, nil
			})
		}, true},
		{"description", func() error {
			return store.UpdateInterface("eth0", func(iface Interface) (Interface, error) {
				iface.Description = "renamed"
				return iface, nil
			})
		}, false},
		{"link down", func() error { return store.SetInterfaceFault("eth0", FaultLinkDown, 1) }, true},
		{"fault magnitude", func() error { return store.SetInterfaceFault("eth0", FaultLinkDown, 100) }, false},
		{"overlapping power loss", func() error { return store.SetInterfaceFault("eth0", FaultPoELoss, 1) }, false},
		{"clear link only", func() error { return store.SetInterfaceFault("eth0", FaultLinkDown, 0) }, false},
		{"clear remaining fault", func() error { return store.ClearInterfaceFaults("eth0") }, true},
		{"counter fault", func() error { return store.SetInterfaceFault("eth0", FaultFCS, 50) }, false},
		{"device fault", func() error { return store.SetDeviceFault(FaultDNSTimeout, 90) }, false},
		{"identity", func() error {
			return store.UpdateIdentity(func(identity Identity) (Identity, error) {
				identity.Hostname = "renamed"
				return identity, nil
			})
		}, false},
		{"power loss", func() error { return store.SetInterfaceFault("eth0", FaultPoELoss, 1) }, true},
		{"raw down while faulted", func() error {
			return store.UpdateInterface("eth0", func(iface Interface) (Interface, error) {
				iface.OperUp = false
				return iface, nil
			})
		}, false},
		{"clear while raw down", func() error { store.ClearAllFaults(); return nil }, false},
	}
	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			*now = now.Add(time.Second)
			if err := step.mutate(); err != nil {
				t.Fatal(err)
			}
			if step.changed {
				last = *now
			}
			if got := store.InterfaceLastChange("eth0"); !reflect.DeepEqual(last, got) {
				t.Fatalf("got %#v, want %#v", got, last)
			}
			if !store.InterfaceLastChange("eth1").IsZero() {
				t.Fatal("expected zero last change")
			}
		})
	}
}

func TestInterfaceLastChangeCheckpointAndEventEviction(t *testing.T) {
	store, now := transitionStore(t)
	store.SaveCheckpoint("healthy")
	*now = now.Add(time.Second)
	if err := store.SetInterfaceFault("eth0", FaultLinkDown, 1); err != nil {
		t.Fatal(err)
	}
	store.SaveCheckpoint("down")
	*now = now.Add(time.Second)
	if err := store.RestoreCheckpoint("healthy"); err != nil {
		t.Fatal(err)
	}
	if got := store.InterfaceLastChange("eth0"); !reflect.DeepEqual(*now, got) {
		t.Fatalf("got %#v, want %#v", got, *now)
	}
	last := *now
	*now = now.Add(time.Second)
	if err := store.RestoreCheckpoint("healthy"); err != nil {
		t.Fatal(err)
	}
	if got := store.InterfaceLastChange("eth0"); !reflect.DeepEqual(last, got) {
		t.Fatalf("got %#v, want %#v", got, last)
	}
	if err := store.RestoreCheckpoint("down"); err != nil {
		t.Fatal(err)
	}
	last = *now
	if got := store.InterfaceLastChange("eth0"); !reflect.DeepEqual(last, got) {
		t.Fatalf("got %#v, want %#v", got, last)
	}
	for range maxEventHistory + 1 {
		*now = now.Add(time.Second)
		store.SaveStartup()
	}
	if len(store.events) != maxEventHistory {
		t.Fatalf("event count = %d", len(store.events))
	}
	if got := store.events[0].Kind; !reflect.DeepEqual(EventStartupSaved, got) {
		t.Fatalf("got %#v, want %#v", got, EventStartupSaved)
	}
	if got := store.InterfaceLastChange("eth0"); !reflect.DeepEqual(last, got) {
		t.Fatalf("got %#v, want %#v", got, last)
	}
}

func TestInterfaceLastChangeRecoveryReseedsWithoutReplay(t *testing.T) {
	store, now := transitionStore(t)
	*now = now.Add(time.Second)
	if err := store.SetInterfaceFault("eth0", FaultLinkDown, 1); err != nil {
		t.Fatal(err)
	}
	state := store.ExportState()
	<-store.Changes()
	*now = now.Add(time.Second)
	if err := store.RestoreState(state); err != nil {
		t.Fatal(err)
	}
	if !store.InterfaceLastChange("eth0").IsZero() {
		t.Fatal("expected zero last change")
	}
	if store.Snapshot().Network.Interfaces[0].OperUp {
		t.Fatal("expected operationally down")
	}
	if got := store.ExportState(); !reflect.DeepEqual(state, got) {
		t.Fatalf("got %#v, want %#v", got, state)
	}
	select {
	case <-store.Changes():
		t.Fatal("recovery replayed a change notification")
	default:
	}
	*now = now.Add(time.Second)
	if err := store.SetInterfaceFault("eth0", FaultLinkDown, 0); err != nil {
		t.Fatal(err)
	}
	if got := store.InterfaceLastChange("eth0"); !reflect.DeepEqual(*now, got) {
		t.Fatalf("got %#v, want %#v", got, *now)
	}
}

func TestInterfaceLastChangeInvalidRecoveryPreservesTransition(t *testing.T) {
	store, now := transitionStore(t)
	*now = now.Add(time.Second)
	if err := store.SetInterfaceFault("eth0", FaultLinkDown, 1); err != nil {
		t.Fatal(err)
	}
	state := store.ExportState()
	state.Version = 0
	if err := store.RestoreState(state); err == nil {
		t.Fatal("invalid recovery accepted")
	}
	if got := store.InterfaceLastChange("eth0"); got != *now {
		t.Fatalf("last change = %v, want %v", got, *now)
	}
}

func TestInterfaceLastChangeUpdatedBeforeObserver(t *testing.T) {
	store, now := transitionStore(t)
	*now = now.Add(time.Second)
	var observed time.Time
	store.SetChangeObserver(func(snapshot Snapshot) {
		if !snapshot.Network.Interfaces[0].OperUp {
			observed = store.interfaceTransitions["eth0"].changedAt
		}
	})
	if err := store.SetInterfaceFault("eth0", FaultLinkDown, 1); err != nil {
		t.Fatal(err)
	}
	if got := observed; !reflect.DeepEqual(*now, got) {
		t.Fatalf("got %#v, want %#v", got, *now)
	}
}
