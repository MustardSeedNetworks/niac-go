package snmp

import (
	"testing"
	"time"

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
		t.Fatalf("fractional FCS increments = %d then %d, want 0 then 1", first.FCSErrors, second.FCSErrors)
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
