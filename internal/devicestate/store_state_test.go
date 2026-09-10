package devicestate_test

import (
	"errors"
	"net/netip"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestExportStateCarriesFaultsCheckpointsAndHistory(t *testing.T) {
	store := newStateTestStore(t)
	if err := store.SetInterfaceFault("eth0", devicestate.FaultLinkDown, 1); err != nil {
		t.Fatalf("SetInterfaceFault() error = %v", err)
	}
	if err := store.SetDeviceFault(devicestate.FaultLatency, 250); err != nil {
		t.Fatalf("SetDeviceFault() error = %v", err)
	}
	store.SaveCheckpoint("faulted")

	state := store.ExportState()
	if len(state.InterfaceFaults) != 1 || state.InterfaceFaults[0].Type != devicestate.FaultLinkDown {
		t.Fatalf("exported interface faults = %#v", state.InterfaceFaults)
	}
	if len(state.DeviceFaults) != 1 || state.DeviceFaults[0].Value != 250 {
		t.Fatalf("exported device faults = %#v", state.DeviceFaults)
	}
	if len(state.Checkpoints) != 1 || state.Checkpoints[0].Name != "faulted" ||
		len(state.Checkpoints[0].InterfaceFaults) != 1 {
		t.Fatalf("exported checkpoints = %#v", state.Checkpoints)
	}
	if len(state.Events) == 0 || state.Version == 0 {
		t.Fatalf("exported history = %d events at version %d", len(state.Events), state.Version)
	}

	// The running bank is exported raw. A snapshot would carry the link-down
	// projection, and restoring that would latch the interface down.
	for _, iface := range state.Running.Network.Interfaces {
		if iface.Name == "eth0" && (!iface.OperUp || !iface.CarrierUp) {
			t.Fatalf("exported running state carries the fault projection: %#v", iface)
		}
	}
}

func TestRestoreStateReinstallsAndKeepsCheckpointsUsable(t *testing.T) {
	source := newStateTestStore(t)
	if err := source.SetInterfaceFault("eth0", devicestate.FaultFCS, 40); err != nil {
		t.Fatalf("SetInterfaceFault() error = %v", err)
	}
	source.SaveCheckpoint("faulted")
	source.ClearAllFaults()
	state := source.ExportState()

	target := newStateTestStore(t)
	observed := 0
	target.SetChangeObserver(func(devicestate.Snapshot) { observed++ })
	before := observed
	if err := target.RestoreState(state); err != nil {
		t.Fatalf("RestoreState() error = %v", err)
	}
	if observed == before {
		t.Fatal("RestoreState did not notify the change observer")
	}
	if target.Version() != state.Version {
		t.Fatalf("restored version = %d, want %d", target.Version(), state.Version)
	}
	if got := target.Events(); len(got) != len(state.Events) {
		t.Fatalf("restored history = %d events, want %d", len(got), len(state.Events))
	}
	if err := target.RestoreCheckpoint("faulted"); err != nil {
		t.Fatalf("RestoreCheckpoint() error = %v", err)
	}
	if faults := target.Snapshot().Faults; len(faults) != 1 || faults[0].Value != 40 {
		t.Fatalf("checkpoint restored faults = %#v", faults)
	}
}

func TestRestoreStateRejectsInvalidRecords(t *testing.T) {
	valid := newStateTestStore(t).ExportState()
	cases := map[string]devicestate.State{
		"zero version": {},
		"unknown interface fault": {
			Version:         valid.Version,
			InterfaceFaults: []devicestate.InterfaceFault{{Interface: "eth0", Type: "not_a_fault", Value: 1}},
		},
		"unnamed interface": {
			Version:         valid.Version,
			InterfaceFaults: []devicestate.InterfaceFault{{Type: devicestate.FaultLinkDown, Value: 1}},
		},
		"interface fault above the rate ceiling": {
			Version:         valid.Version,
			InterfaceFaults: []devicestate.InterfaceFault{{Interface: "eth0", Type: devicestate.FaultFCS, Value: 101}},
		},
		"unknown device fault": {
			Version:      valid.Version,
			DeviceFaults: []devicestate.DeviceFault{{Type: "not_a_fault", Value: 1}},
		},
		"device fault above its own ceiling": {
			Version:      valid.Version,
			DeviceFaults: []devicestate.DeviceFault{{Type: devicestate.FaultLatency, Value: latencyFaultCeiling() + 1}},
		},
	}
	for name, state := range cases {
		t.Run(name, func(t *testing.T) {
			state.Running = valid.Running
			state.Startup = valid.Startup
			store := newStateTestStore(t)
			version := store.Version()
			if err := store.RestoreState(state); !errors.Is(err, devicestate.ErrStateInvalid) {
				t.Fatalf("RestoreState() error = %v, want devicestate.ErrStateInvalid", err)
			}
			if store.Version() != version {
				t.Fatalf("a rejected record changed the store: version %d", store.Version())
			}
		})
	}
}

func newStateTestStore(t *testing.T) *devicestate.Store {
	t.Helper()
	store := devicestate.NewStore(devicestate.Identity{Hostname: "state-test"})
	store.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{{
		Name: "eth0", Address: netip.MustParsePrefix("192.0.2.5/24"),
		AdminUp: true, OperUp: true, CarrierUp: true,
	}}})
	return store
}

func latencyFaultCeiling() int {
	for _, definition := range devicestate.DeviceFaultDefinitions() {
		if definition.Type == devicestate.FaultLatency {
			return definition.MaxValue
		}
	}
	return 0
}
