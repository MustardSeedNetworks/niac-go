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

// A carrier fault is a link change: setting one records an interface event with
// the effective state on both sides, which is what link traps follow (#2472). A
// counter fault moves no link, and neither does a second carrier fault on a
// link that is already down.
func TestStoreCarrierFaultRecordsTheLinkChange(t *testing.T) {
	store := faultStore()

	cursor := store.Snapshot().Version
	if err := store.SetInterfaceFault("Gi0/2", devicestate.FaultUtilization, 90); err != nil {
		t.Fatal(err)
	}
	if events := linkEvents(store, cursor); len(events) != 0 {
		t.Fatalf("a utilization fault recorded link changes %+v", events)
	}

	cursor = store.Snapshot().Version
	if err := store.SetInterfaceFault("Gi0/2", devicestate.FaultLinkDown, 1); err != nil {
		t.Fatal(err)
	}
	wantLinkChange(t, linkEvents(store, cursor), false)

	cursor = store.Snapshot().Version
	if err := store.SetInterfaceFault("Gi0/2", devicestate.FaultPoELoss, 1); err != nil {
		t.Fatal(err)
	}
	if events := linkEvents(store, cursor); len(events) != 0 {
		t.Fatalf("a second carrier fault on a down link recorded %+v", events)
	}
}

// Every way of clearing the last carrier fault brings the link back up.
func TestStoreCarrierFaultClearRecordsTheLinkChange(t *testing.T) {
	for name, clearFaults := range map[string]func(*devicestate.Store) error{
		"zero value": func(store *devicestate.Store) error {
			if err := store.SetInterfaceFault("Gi0/2", devicestate.FaultPoELoss, 0); err != nil {
				return err
			}
			return store.SetInterfaceFault("Gi0/2", devicestate.FaultLinkDown, 0)
		},
		"interface": func(store *devicestate.Store) error { return store.ClearInterfaceFaults("Gi0/2") },
		"all":       func(store *devicestate.Store) error { store.ClearAllFaults(); return nil },
	} {
		t.Run(name, func(t *testing.T) {
			store := faultStore()
			for _, carrier := range []devicestate.FaultType{devicestate.FaultLinkDown, devicestate.FaultPoELoss} {
				if err := store.SetInterfaceFault("Gi0/2", carrier, 1); err != nil {
					t.Fatal(err)
				}
			}
			cursor := store.Snapshot().Version
			if err := clearFaults(store); err != nil {
				t.Fatal(err)
			}
			wantLinkChange(t, linkEvents(store, cursor), true)
		})
	}
}

func linkEvents(store *devicestate.Store, after uint64) []devicestate.Event {
	events, _ := store.EventsAfter(after)
	var links []devicestate.Event
	for _, event := range events {
		if event.Kind == devicestate.EventInterfaceUpdated {
			links = append(links, event)
		}
	}
	return links
}

// wantLinkChange asserts events is Gi0/2, IF-MIB index 2, going to up.
func wantLinkChange(t *testing.T, events []devicestate.Event, up bool) {
	t.Helper()
	if len(events) != 1 {
		t.Fatalf("interface events = %+v, want one", events)
	}
	event := events[0]
	if event.Target != "Gi0/2" || event.InterfaceIndex != 2 ||
		event.PreviousInterface.OperUp == up || event.Interface.OperUp != up || event.Interface.CarrierUp != up {
		t.Fatalf("interface event = %+v (%+v -> %+v), want Gi0/2 operUp %v",
			event, event.PreviousInterface, event.Interface, up)
	}
}
