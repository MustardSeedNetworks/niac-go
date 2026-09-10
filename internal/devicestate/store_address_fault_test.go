package devicestate_test

import (
	"encoding/json"
	"net/netip"
	"reflect"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestAddressFaultPersistenceAndClear(t *testing.T) {
	store := faultStore()
	kind := devicestate.FaultDuplicateDHCPOffer
	address := netip.MustParseAddr("192.0.2.30")
	version := store.Version()
	if err := store.SetDeviceAddressFault(kind, address); err != nil {
		t.Fatal(err)
	}
	if store.Version() != version+1 || !store.DeviceFaultActive(kind) || store.DeviceFaultAddress(kind) != address {
		t.Fatal("address fault not committed")
	}
	if err := store.SetDeviceAddressFault(kind, address); err != nil {
		t.Fatal(err)
	}
	if store.Version() != version+1 {
		t.Fatal("unchanged payload emitted another event")
	}
	store.SaveCheckpoint("armed")
	saved := store.ExportState()
	data, err := json.Marshal(saved)
	if err != nil {
		t.Fatal(err)
	}
	var decoded devicestate.State
	if err = json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	restored := faultStore()
	if err = restored.RestoreState(decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved, restored.ExportState()) {
		t.Fatal("recovery changed address payload or event history")
	}
	if err = restored.ClearDeviceFault(kind); err != nil {
		t.Fatal(err)
	}
	if restored.DeviceFaultActive(kind) || restored.DeviceFaultAddress(kind).IsValid() {
		t.Fatal("clear retained address")
	}
	if err = restored.RestoreCheckpoint("armed"); err != nil {
		t.Fatal(err)
	}
	if restored.DeviceFaultAddress(kind) != address {
		t.Fatal("checkpoint lost address")
	}
	restored.ClearDeviceFaults()
	if restored.DeviceFaultActive(kind) {
		t.Fatal("reset retained address fault")
	}
}

func TestAddressFaultInvalidPayloadPreservesState(t *testing.T) {
	store := faultStore()
	kind := devicestate.FaultDuplicateDHCPOffer
	before := store.ExportState()
	for _, text := range []string{"", "0.0.0.0", "224.0.0.1", "255.255.255.255", "2001:db8::1", "::ffff:192.0.2.1"} {
		address, _ := netip.ParseAddr(text)
		if err := store.SetDeviceAddressFault(kind, address); err == nil {
			t.Fatalf("accepted %q", text)
		}
	}
	if err := store.SetDeviceAddressFault(devicestate.FaultLatency, netip.MustParseAddr("192.0.2.1")); err == nil {
		t.Fatal("address setter accepted numeric kind")
	}
	for _, value := range []int{0, 1, 100} {
		if err := store.SetDeviceFault(kind, value); err == nil {
			t.Fatal("numeric setter accepted address kind")
		}
	}
	if !reflect.DeepEqual(before, store.ExportState()) {
		t.Fatal("invalid fault changed state")
	}
}

func TestAddressFaultClearIsIndependentAndIdempotent(t *testing.T) {
	store := faultStore()
	kind := devicestate.FaultDuplicateDHCPOffer
	if err := store.SetDeviceFault(devicestate.FaultLatency, 17); err != nil {
		t.Fatal(err)
	}
	if err := store.SetDeviceAddressFault(kind, netip.MustParseAddr("192.0.2.1")); err != nil {
		t.Fatal(err)
	}
	version := store.Version()
	if err := store.ClearDeviceFault(kind); err != nil {
		t.Fatal(err)
	}
	if store.Version() != version+1 || store.DeviceFaultValue(devicestate.FaultLatency) != 17 {
		t.Fatal("clear changed peer fault or wrong version")
	}
	state := store.ExportState()
	last := state.Events[len(state.Events)-1]
	if last.Kind != devicestate.EventDeviceFaultCleared || last.Target != string(kind) {
		t.Fatalf("wrong clearing event: %+v", last)
	}
	if err := store.ClearDeviceFault(kind); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state, store.ExportState()) {
		t.Fatal("second clear changed state")
	}
	if err := store.ClearDeviceFault("unknown"); err == nil {
		t.Fatal("unknown clear accepted")
	}
}

func TestAddressFaultRestoreRejectsWrongFields(t *testing.T) {
	for _, fault := range []devicestate.DeviceFault{
		{Type: devicestate.FaultDuplicateDHCPOffer},
		{Type: devicestate.FaultDuplicateDHCPOffer, Value: 1, Address: netip.MustParseAddr("192.0.2.1")},
		{Type: devicestate.FaultLatency, Value: 1, Address: netip.MustParseAddr("192.0.2.1")},
		{Type: devicestate.FaultDuplicateDHCPOffer, Address: netip.MustParseAddr("224.0.0.1")},
	} {
		for _, checkpoint := range []bool{false, true} {
			store := faultStore()
			before := store.ExportState()
			state := store.ExportState()
			if checkpoint {
				state.Checkpoints = []devicestate.Checkpoint{
					{Name: "invalid", Configuration: state.Running, DeviceFaults: []devicestate.DeviceFault{fault}},
				}
			} else {
				state.DeviceFaults = []devicestate.DeviceFault{fault}
			}
			if err := store.RestoreState(state); err == nil {
				t.Fatalf("accepted invalid fault %+v", fault)
			}
			if !reflect.DeepEqual(before, store.ExportState()) {
				t.Fatal("invalid restore mutated state")
			}
		}
	}
}
