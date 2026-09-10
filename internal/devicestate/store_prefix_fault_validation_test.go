package devicestate_test

import (
	"encoding/json"
	"net/netip"
	"reflect"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestPrefixFaultValidation(t *testing.T) {
	for _, fault := range []devicestate.InterfacePrefixFault{
		{Interface: "missing", Type: devicestate.FaultBadMask, PrefixBits: 16},
		{Interface: "eth0", Type: "unknown", PrefixBits: 16},
		{Interface: "eth0", Type: devicestate.FaultBadMask, PrefixBits: -1},
		{Interface: "eth0", Type: devicestate.FaultBadMask, PrefixBits: 33},
	} {
		store := prefixStore()
		before := store.ExportState()
		if err := store.SetInterfacePrefixFault(fault); err == nil {
			t.Fatalf("accepted %+v", fault)
		}
		for _, checkpoint := range []bool{false, true} {
			state := store.ExportState()
			if checkpoint {
				state.Checkpoints = []devicestate.Checkpoint{
					{
						Name:          "invalid",
						Configuration: state.Running,
						PrefixFaults:  []devicestate.InterfacePrefixFault{fault},
					},
				}
			} else {
				state.PrefixFaults = []devicestate.InterfacePrefixFault{fault}
			}
			if err := store.RestoreState(state); err == nil {
				t.Fatalf("restored %+v checkpoint=%v", fault, checkpoint)
			}
		}
		if !reflect.DeepEqual(before, store.ExportState()) {
			t.Fatal("invalid input changed state")
		}
	}
}

func TestPrefixFaultRecoveryAndIndependentClear(t *testing.T) {
	store := prefixStore()
	for _, iface := range []string{"eth0", "eth1"} {
		if err := store.SetInterfacePrefixFault(
			devicestate.InterfacePrefixFault{Interface: iface, Type: devicestate.FaultBadMask, PrefixBits: 0},
		); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.SetInterfaceAddressFault(
		devicestate.InterfaceAddressFault{
			Interface: "eth0",
			Type:      devicestate.FaultDuplicateIP,
			Address:   netip.MustParseAddr("192.0.2.20"),
		},
	); err != nil {
		t.Fatal(err)
	}
	if err := store.SetInterfaceFault("eth0", devicestate.FaultUtilization, 50); err != nil {
		t.Fatal(err)
	}
	before := store.ExportState()
	encoded, err := json.Marshal(before)
	if err != nil {
		t.Fatal(err)
	}
	var state devicestate.State
	if err = json.Unmarshal(encoded, &state); err != nil {
		t.Fatal(err)
	}
	store.ClearAllFaults()
	if err = store.RestoreState(state); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, store.ExportState()) {
		t.Fatal("JSON round trip changed state")
	}
	version := store.Version()
	if err = store.SetInterfacePrefixFault(state.PrefixFaults[0]); err != nil {
		t.Fatal(err)
	}
	if store.Version() != version {
		t.Fatal("repeated set changed version")
	}
	if err = store.ClearInterfacePrefixFault("eth0", devicestate.FaultBadMask); err != nil {
		t.Fatal(err)
	}
	snapshot := store.Snapshot()
	if len(snapshot.PrefixFaults) != 1 || snapshot.PrefixFaults[0].Interface != "eth1" ||
		len(snapshot.AddressFaults) != 1 ||
		len(snapshot.Faults) != 1 {
		t.Fatal("clear changed unrelated fault")
	}
	version = store.Version()
	if err = store.ClearInterfacePrefixFault("eth0", devicestate.FaultBadMask); err != nil {
		t.Fatal(err)
	}
	if store.Version() != version {
		t.Fatal("repeated clear changed version")
	}
	if err = store.ClearInterfaceFaults("eth1"); err != nil {
		t.Fatal(err)
	}
	if len(store.Snapshot().PrefixFaults) != 0 {
		t.Fatal("interface clear retained prefix fault")
	}
}

func TestPrefixFaultRejectsDuplicateRecovery(t *testing.T) {
	store := prefixStore()
	fault := devicestate.InterfacePrefixFault{Interface: "eth0", Type: devicestate.FaultBadMask, PrefixBits: 16}
	before := store.ExportState()
	for _, checkpoint := range []bool{false, true} {
		state := store.ExportState()
		if checkpoint {
			state.Checkpoints = []devicestate.Checkpoint{
				{
					Name:          "duplicate",
					Configuration: state.Running,
					PrefixFaults:  []devicestate.InterfacePrefixFault{fault, fault},
				},
			}
		} else {
			state.PrefixFaults = []devicestate.InterfacePrefixFault{fault, fault}
		}
		if err := store.RestoreState(state); err == nil {
			t.Fatal("accepted duplicate keys")
		}
		if !reflect.DeepEqual(before, store.ExportState()) {
			t.Fatal("invalid recovery changed state")
		}
	}
}

func TestPrefixFaultRequiresIPv4OnlyWhenArmed(t *testing.T) {
	for _, address := range []netip.Prefix{{}, netip.MustParsePrefix("2001:db8::1/64")} {
		store := prefixStore()
		fault := devicestate.InterfacePrefixFault{Interface: "eth0", Type: devicestate.FaultBadMask, PrefixBits: 16}
		if err := store.SetInterfacePrefixFault(fault); err != nil {
			t.Fatal(err)
		}
		if err := store.UpdateInterface("eth0", func(iface devicestate.Interface) (devicestate.Interface, error) {
			iface.Address = address
			return iface, nil
		}); err != nil {
			t.Fatal(err)
		}
		state := store.ExportState()
		if err := store.RestoreState(state); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(store.Snapshot().EffectiveHostNetwork(), store.Snapshot().Network) {
			t.Fatal("mask fault changed non-IPv4 interface")
		}
		if err := store.ClearInterfacePrefixFault("eth0", devicestate.FaultBadMask); err != nil {
			t.Fatal(err)
		}
		if err := store.SetInterfacePrefixFault(fault); err == nil {
			t.Fatal("armed fault without IPv4")
		}
	}
}
