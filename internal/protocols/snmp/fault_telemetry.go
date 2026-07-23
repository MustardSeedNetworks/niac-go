package snmp

import (
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

const defaultFaultInterfaceSpeedMbps = 1000

type faultCounterAccumulator struct {
	last       time.Time
	remainders map[faultRateKey]uint64
}

type faultRateKey struct {
	interfaceName string
	faultType     devicestate.FaultType
}

func newFaultCounterAccumulator(start time.Time) *faultCounterAccumulator {
	return &faultCounterAccumulator{last: start, remainders: make(map[faultRateKey]uint64)}
}

// AdvanceInterfaceFaults adds elapsed active-fault telemetry to interface counters.
func (t *ProtocolTelemetry) AdvanceInterfaceFaults(
	now time.Time,
	faults []devicestate.InterfaceFault,
	speedsMbps map[string]int,
) {
	if t == nil {
		return
	}
	t.faultMu.Lock()
	defer t.faultMu.Unlock()
	if t.faultCounters == nil {
		t.faultCounters = newFaultCounterAccumulator(now)
		return
	}
	for name, delta := range t.faultCounters.advance(now, faults, speedsMbps) {
		t.RecordInterfaceCounters(name, delta)
	}
}

func (a *faultCounterAccumulator) advance(
	now time.Time,
	faults []devicestate.InterfaceFault,
	speedsMbps map[string]int,
) map[string]InterfaceCounterDelta {
	if a == nil || !now.After(a.last) {
		return nil
	}
	elapsed := now.Sub(a.last)
	a.last = now
	deltas := make(map[string]InterfaceCounterDelta)
	active := make(map[faultRateKey]struct{}, len(faults))
	for _, fault := range faults {
		rate := faultRate(fault, speedsMbps)
		key := faultRateKey{interfaceName: fault.Interface, faultType: fault.Type}
		active[key] = struct{}{}
		addFaultDelta(deltas, fault, a.advanceRate(key, rate, elapsed))
	}
	for key := range a.remainders {
		if _, ok := active[key]; !ok {
			delete(a.remainders, key)
		}
	}
	return deltas
}

func faultRate(fault devicestate.InterfaceFault, speedsMbps map[string]int) uint64 {
	if fault.Value <= 0 {
		return 0
	}
	value := uint64(fault.Value)
	if fault.Type != devicestate.FaultUtilization {
		return value
	}
	speed := speedsMbps[fault.Interface]
	if speed <= 0 {
		speed = defaultFaultInterfaceSpeedMbps
	}
	return uint64(speed) * 1_000_000 / 8 * value / ThresholdPercentage
}

func addFaultDelta(
	deltas map[string]InterfaceCounterDelta,
	fault devicestate.InterfaceFault,
	increment uint64,
) {
	if increment == 0 {
		return
	}
	delta := deltas[fault.Interface]
	switch fault.Type {
	case devicestate.FaultFCS:
		delta.FCSErrors += increment
		delta.InErrors += increment
	case devicestate.FaultDiscards:
		delta.InDiscards += increment
		delta.OutDiscards += increment
	case devicestate.FaultInterface:
		delta.InErrors += increment
		delta.OutErrors += increment
	case devicestate.FaultUtilization:
		delta.InOctets += increment
		delta.OutOctets += increment
	}
	deltas[fault.Interface] = delta
}

func (a *faultCounterAccumulator) advanceRate(
	key faultRateKey,
	ratePerSecond uint64,
	elapsed time.Duration,
) uint64 {
	if elapsed <= 0 {
		return 0
	}
	const denominator = uint64(time.Second)
	nanoseconds := safeUint64(elapsed.Nanoseconds())
	seconds := nanoseconds / denominator
	fraction := nanoseconds % denominator
	quotient, remainder := ratePerSecond/denominator, ratePerSecond%denominator
	numerator := remainder*fraction + a.remainders[key]
	a.remainders[key] = numerator % denominator
	return ratePerSecond*seconds + quotient*fraction + numerator/denominator
}
