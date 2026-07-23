package snmp

import (
	"net/netip"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestFaultTelemetryAdvancesObservableCounters(t *testing.T) {
	start := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	accumulator := newFaultCounterAccumulator(start)
	faults := []devicestate.InterfaceFault{
		{Interface: "Gi0/1", Type: devicestate.FaultFCS, Value: 4},
		{Interface: "Gi0/1", Type: devicestate.FaultDiscards, Value: 2},
		{Interface: "Gi0/1", Type: devicestate.FaultInterface, Value: 2},
		{Interface: "Gi0/1", Type: devicestate.FaultUtilization, Value: 50},
	}

	delta := accumulator.advance(
		start.Add(1500*time.Millisecond), faults, map[string]int{"Gi0/1": 100},
	)["Gi0/1"]
	if delta.FCSErrors != 6 || delta.InErrors != 9 || delta.OutErrors != 3 {
		t.Fatalf("error counters = %#v", delta)
	}
	if delta.InDiscards != 3 || delta.OutDiscards != 3 {
		t.Fatalf("discard counters = %#v", delta)
	}
	if delta.InOctets != 9_375_000 || delta.OutOctets != 9_375_000 {
		t.Fatalf("octet counters = %#v", delta)
	}
}

func TestFaultTelemetryCarriesFractionalProgress(t *testing.T) {
	start := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	accumulator := newFaultCounterAccumulator(start)
	faults := []devicestate.InterfaceFault{{
		Interface: "Gi0/1", Type: devicestate.FaultFCS, Value: 3,
	}}

	first := accumulator.advance(start.Add(250*time.Millisecond), faults, nil)["Gi0/1"]
	second := accumulator.advance(start.Add(500*time.Millisecond), faults, nil)["Gi0/1"]
	if first.FCSErrors != 0 || second.FCSErrors != 1 {
		t.Fatalf(
			"fractional FCS increments = %d then %d, want 0 then 1",
			first.FCSErrors,
			second.FCSErrors,
		)
	}
}

func TestFaultTelemetryStopsWhenFaultClears(t *testing.T) {
	start := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	accumulator := newFaultCounterAccumulator(start)
	faults := []devicestate.InterfaceFault{{
		Interface: "Gi0/1", Type: devicestate.FaultDiscards, Value: 10,
	}}

	beforeClear := accumulator.advance(start.Add(time.Second), faults, nil)["Gi0/1"]
	afterClear := accumulator.advance(start.Add(2*time.Second), nil, nil)
	if beforeClear.InDiscards != 10 || beforeClear.OutDiscards != 10 {
		t.Fatalf("before-clear counters = %#v", beforeClear)
	}
	if len(afterClear) != 0 {
		t.Fatalf("after-clear deltas = %#v, want none", afterClear)
	}
}

func TestFaultTelemetryDropsFractionalProgressWhenFaultClears(t *testing.T) {
	start := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	accumulator := newFaultCounterAccumulator(start)
	faults := []devicestate.InterfaceFault{{
		Interface: "Gi0/1", Type: devicestate.FaultFCS, Value: 3,
	}}

	accumulator.advance(start.Add(250*time.Millisecond), faults, nil)
	accumulator.advance(start.Add(500*time.Millisecond), nil, nil)
	afterReenable := accumulator.advance(start.Add(750*time.Millisecond), faults, nil)["Gi0/1"]
	if afterReenable.FCSErrors != 0 {
		t.Fatalf("FCS increment after re-enable = %d, want 0", afterReenable.FCSErrors)
	}
}

func TestFaultTelemetryUsesDefaultInterfaceSpeed(t *testing.T) {
	start := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	accumulator := newFaultCounterAccumulator(start)
	faults := []devicestate.InterfaceFault{{
		Interface: "Gi0/1", Type: devicestate.FaultUtilization, Value: 100,
	}}

	delta := accumulator.advance(start.Add(time.Second), faults, nil)["Gi0/1"]
	if delta.InOctets != 125_000_000 || delta.OutOctets != 125_000_000 {
		t.Fatalf("default-speed octets = %#v", delta)
	}
}

func TestAgentGetAdvancesFaultCountersFromDeviceState(t *testing.T) {
	device := &config.Device{
		Name:       "edge-1",
		Interfaces: []config.Interface{{Name: "Gi0/1", Speed: 100}},
		TrunkPorts: []config.TrunkPort{{Interface: "Gi0/1"}},
	}
	state := devicestate.NewStore(devicestate.Identity{Hostname: device.Name})
	state.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{{Name: "Gi0/1"}}})
	telemetry := NewProtocolTelemetry()
	telemetry.faultCounters = newFaultCounterAccumulator(time.Now().Add(-time.Second))
	agent := NewAgentWithState(device, state, AgentOptions{Telemetry: telemetry})
	if err := state.SetInterfaceFault("Gi0/1", devicestate.FaultFCS, 100); err != nil {
		t.Fatalf("SetInterfaceFault() error = %v", err)
	}

	value, err := agent.HandleGet(dot3StatsFCSErrors + ".1")
	if err != nil || value.Value.(uint32) < 100 {
		t.Fatalf("dot3StatsFCSErrors = %#v, %v; want at least 100", value, err)
	}
}

func TestManagementInterfaceFaultCountersAreObservable(t *testing.T) {
	device := &config.Device{Name: "edge-1"}
	state := devicestate.NewStore(devicestate.Identity{Hostname: device.Name})
	state.ReplaceNetwork(
		devicestate.Network{Interfaces: []devicestate.Interface{{Name: "Management"}}},
	)
	telemetry := NewProtocolTelemetry()
	telemetry.faultCounters = newFaultCounterAccumulator(time.Now().Add(-time.Second))
	agent := NewAgentWithState(device, state, AgentOptions{Telemetry: telemetry})
	if err := state.SetInterfaceFault("Management", devicestate.FaultFCS, 100); err != nil {
		t.Fatal(err)
	}

	value, err := agent.HandleGet(dot3StatsFCSErrors + ".1")
	if err != nil || value.Value.(uint32) < 100 {
		t.Fatalf("management FCS counter = %#v, %v; want at least 100", value, err)
	}
}

func TestGetBulkAdvancesFaultTelemetry(t *testing.T) {
	device := &config.Device{Name: "edge-1"}
	state := devicestate.NewStore(devicestate.Identity{Hostname: device.Name})
	state.ReplaceNetwork(
		devicestate.Network{Interfaces: []devicestate.Interface{{Name: "Management"}}},
	)
	telemetry := NewProtocolTelemetry()
	telemetry.faultCounters = newFaultCounterAccumulator(time.Now().Add(-time.Second))
	agent := NewAgentWithState(device, state, AgentOptions{Telemetry: telemetry})
	if err := state.SetInterfaceFault("Management", devicestate.FaultInterface, 5); err != nil {
		t.Fatal(err)
	}

	response := agent.ProcessPDU(
		gosnmp.GetBulkRequest,
		[]gosnmp.SnmpPDU{{Name: ifInErrors}},
		0,
		1,
	)
	if len(response) != 1 || response[0].Name != ifInErrors+".1" ||
		response[0].Value.(uint32) < 5 {
		t.Fatalf("GET-BULK response = %#v, want advancing ifInErrors.1", response)
	}
}

func TestWalkBackedManagementFaultCountersPreserveBaseline(t *testing.T) {
	device := &config.Device{
		Name: "edge-1",
		SNMPConfig: config.SNMPConfig{
			WalkFile: "capture.walk",
		},
	}
	state := devicestate.NewStore(devicestate.Identity{Hostname: device.Name})
	state.ReplaceNetwork(
		devicestate.Network{Interfaces: []devicestate.Interface{{
			Name: "Management", Address: netip.MustParsePrefix("192.0.2.1/24"),
		}}},
	)
	telemetry := NewProtocolTelemetry()
	telemetry.faultCounters = newFaultCounterAccumulator(time.Now().Add(-time.Second))
	agent := NewAgentWithState(device, state, AgentOptions{Telemetry: telemetry})
	agent.mib.Set(ifDescr+".10001", &OIDValue{Type: gosnmp.OctetString, Value: "lo"})
	agent.mib.Set(ifDescr+".10002", &OIDValue{Type: gosnmp.OctetString, Value: "eth0"})
	agent.mib.Set(ipAdEntIfIndex+".192.0.2.1", &OIDValue{Type: gosnmp.Integer, Value: 10002})
	agent.mib.Set(ifInErrors+".10002", &OIDValue{Type: gosnmp.Counter32, Value: uint32(12)})
	agent.mib.Set(ifInOctets+".10002", &OIDValue{Type: gosnmp.Counter32, Value: uint32(1000)})
	agent.mib.Set(ifHCInOctets+".10002", &OIDValue{Type: gosnmp.Counter64, Value: uint64(1000)})
	agent.registerWalkStateFaultCounters()
	if !agent.InterfaceFaultObservable("Management") {
		t.Fatal("walk-backed Management interface is not observable")
	}
	if err := state.UpdateInterface(
		"Management",
		func(iface devicestate.Interface) (devicestate.Interface, error) {
			iface.Address = netip.MustParsePrefix("192.0.2.2/24")
			return iface, nil
		},
	); err != nil {
		t.Fatal(err)
	}
	if !agent.InterfaceFaultObservable("Management") {
		t.Fatal("walk-backed Management interface lost observability after address change")
	}
	if err := state.SetInterfaceFault("Management", devicestate.FaultInterface, 5); err != nil {
		t.Fatal(err)
	}
	if err := state.SetInterfaceFault("Management", devicestate.FaultUtilization, 50); err != nil {
		t.Fatal(err)
	}

	value, err := agent.HandleGet(ifInErrors + ".10002")
	if err != nil || value.Value.(uint32) < 17 {
		t.Fatalf("walk ifInErrors = %#v, %v; want at least baseline 12 + fault 5", value, err)
	}
	octets32, err := agent.HandleGet(ifInOctets + ".10002")
	if err != nil || octets32.Value.(uint32) <= 1000 {
		t.Fatalf("walk ifInOctets = %#v, %v; want above baseline 1000", octets32, err)
	}
	octets64, err := agent.HandleGet(ifHCInOctets + ".10002")
	if err != nil || octets64.Value.(uint64) <= 1000 {
		t.Fatalf("walk ifHCInOctets = %#v, %v; want above baseline 1000", octets64, err)
	}
}
