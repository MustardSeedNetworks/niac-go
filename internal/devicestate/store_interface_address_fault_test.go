package devicestate_test

import (
	"encoding/json"
	"net/netip"
	"reflect"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestInterfaceAddressFaultPersistenceAndIsolation(t *testing.T) {
	store := devicestate.NewStore(devicestate.Identity{})
	store.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{
		{Name: "eth0", Address: netip.MustParsePrefix("192.0.2.1/24")},
		{Name: "eth1", Address: netip.MustParsePrefix("198.51.100.1/24")},
	}})
	network := store.Snapshot().Network
	faults := []devicestate.InterfaceAddressFault{
		{Interface: "eth0", Type: devicestate.FaultDuplicateIP, Address: netip.MustParseAddr("192.0.2.20")},
		{Interface: "eth1", Type: devicestate.FaultDuplicateIP, Address: netip.MustParseAddr("198.51.100.20")},
	}
	for _, fault := range faults {
		if err := store.SetInterfaceAddressFault(fault); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.SetInterfaceFault("eth0", devicestate.FaultUtilization, 70); err != nil {
		t.Fatal(err)
	}
	store.SaveCheckpoint("conflicts")
	before := store.ExportState()
	encoded, err := json.Marshal(before)
	if err != nil {
		t.Fatal(err)
	}
	var state devicestate.State
	if err = json.Unmarshal(encoded, &state); err != nil {
		t.Fatal(err)
	}
	if err = store.ClearInterfaceAddressFault("eth0", devicestate.FaultDuplicateIP); err != nil {
		t.Fatal(err)
	}
	if got := store.Snapshot(); !reflect.DeepEqual(got.AddressFaults, faults[1:]) || len(got.Faults) != 1 {
		t.Fatalf("clear changed another fault: %+v", got)
	}
	if !reflect.DeepEqual(network, store.Snapshot().Network) {
		t.Fatal("fault changed canonical network")
	}
	version := store.Version()
	if err = store.ClearInterfaceAddressFault("eth0", devicestate.FaultDuplicateIP); err != nil {
		t.Fatal(err)
	}
	if store.Version() != version {
		t.Fatal("second clear changed version")
	}
	if err = store.RestoreState(state); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, store.ExportState()) {
		t.Fatal("durable round trip changed state")
	}
	if err = store.ClearInterfaceFaults("eth0"); err != nil {
		t.Fatal(err)
	}
	if len(store.Snapshot().AddressFaults) != 1 {
		t.Fatal("interface clear did not include addressed fault")
	}
	if err = store.RestoreCheckpoint("conflicts"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(store.Snapshot().AddressFaults, faults) {
		t.Fatal("checkpoint lost faults")
	}
	store.ClearAllFaults()
	if len(store.Snapshot().AddressFaults) != 0 {
		t.Fatal("clear all retained addressed fault")
	}
}

func TestInterfaceAddressFaultCheckpointIndependence(t *testing.T) {
	store := devicestate.NewStore(devicestate.Identity{})
	store.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{{Name: "eth0"}}})
	fault := devicestate.InterfaceAddressFault{
		Interface: "eth0", Type: devicestate.FaultDuplicateIP, Address: netip.MustParseAddr("192.0.2.20"),
	}
	if err := store.SetInterfaceAddressFault(fault); err != nil {
		t.Fatal(err)
	}
	version := store.Version()
	if err := store.SetInterfaceAddressFault(fault); err != nil {
		t.Fatal(err)
	}
	if store.Version() != version {
		t.Fatal("same payload changed version")
	}
	store.SaveCheckpoint("conflict")
	replacement := fault
	replacement.Address = netip.MustParseAddr("192.0.2.21")
	if err := store.SetInterfaceAddressFault(replacement); err != nil {
		t.Fatal(err)
	}
	if err := store.RestoreCheckpoint("conflict"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(store.Snapshot().AddressFaults, []devicestate.InterfaceAddressFault{fault}) {
		t.Fatal("replacement changed saved checkpoint")
	}
}

func TestInterfaceAddressFaultRejectsDuplicateRestoreEntries(t *testing.T) {
	store := devicestate.NewStore(devicestate.Identity{})
	store.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{{Name: "eth0"}}})
	fault := devicestate.InterfaceAddressFault{
		Interface: "eth0", Type: devicestate.FaultDuplicateIP, Address: netip.MustParseAddr("192.0.2.20"),
	}
	before := store.ExportState()
	for _, checkpoint := range []bool{false, true} {
		state := before
		if checkpoint {
			state.Checkpoints = []devicestate.Checkpoint{{
				Name: "duplicate", Configuration: before.Running,
				AddressFaults: []devicestate.InterfaceAddressFault{fault, fault},
			}}
		} else {
			state.AddressFaults = []devicestate.InterfaceAddressFault{fault, fault}
		}
		if err := store.RestoreState(state); err == nil {
			t.Fatalf("accepted duplicate entries checkpoint=%v", checkpoint)
		}
		if !reflect.DeepEqual(before, store.ExportState()) {
			t.Fatal("rejected duplicates changed state")
		}
	}
}

func TestInterfaceAddressFaultValidation(t *testing.T) {
	store := devicestate.NewStore(devicestate.Identity{})
	store.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{{Name: "eth0"}}})
	before := store.ExportState()
	for _, fault := range []devicestate.InterfaceAddressFault{
		{Interface: "missing", Type: devicestate.FaultDuplicateIP, Address: netip.MustParseAddr("192.0.2.20")},
		{Interface: "eth0", Type: "unknown", Address: netip.MustParseAddr("192.0.2.20")},
		{Interface: "eth0", Type: devicestate.FaultDuplicateIP},
		{Interface: "eth0", Type: devicestate.FaultDuplicateIP, Address: netip.MustParseAddr("224.0.0.1")},
	} {
		if err := store.SetInterfaceAddressFault(fault); err == nil {
			t.Fatalf("accepted %+v", fault)
		}
		state := before
		state.AddressFaults = []devicestate.InterfaceAddressFault{fault}
		if err := store.RestoreState(state); err == nil {
			t.Fatalf("restored %+v", fault)
		}
		state.AddressFaults = nil
		state.Checkpoints = []devicestate.Checkpoint{
			{Name: "bad", Configuration: before.Running, AddressFaults: []devicestate.InterfaceAddressFault{fault}},
		}
		if err := store.RestoreState(state); err == nil {
			t.Fatalf("restored checkpoint %+v", fault)
		}
		if !reflect.DeepEqual(before, store.ExportState()) {
			t.Fatal("rejected input changed state")
		}
	}
}
