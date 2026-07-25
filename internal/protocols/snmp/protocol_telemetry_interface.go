package snmp

import (
	"sync/atomic"
	"time"
)

type interfaceTelemetry struct {
	inOctets, inUcast, inNUcast, outOctets, outUcast, outNUcast atomic.Uint64
	inBroadcast, outBroadcast                                   atomic.Uint64
	inDiscards, outDiscards, inErrors, outErrors, fcsErrors     atomic.Uint64
}

type interfaceSnapshot struct {
	inOctets, inUcast, inNUcast, outOctets, outUcast, outNUcast uint64
	inBroadcast, outBroadcast                                   uint64
	inDiscards, outDiscards, inErrors, outErrors, fcsErrors     uint64
}

// InterfaceCounterDelta contains monotonic additions to SNMP interface counters.
type InterfaceCounterDelta struct {
	InOctets, OutOctets     uint64
	InDiscards, OutDiscards uint64
	InErrors, OutErrors     uint64
	FCSErrors               uint64
}

// RecordInterfaceInbound records a frame delivered through one authored interface.
func (t *ProtocolTelemetry) RecordInterfaceInbound(
	name string,
	octets int,
	nonUnicast, broadcast bool,
) {
	entry := t.interfaceEntry(name)
	if entry == nil {
		return
	}
	addOctets(&entry.inOctets, octets)
	if nonUnicast {
		entry.inNUcast.Add(1)
		if broadcast {
			entry.inBroadcast.Add(1)
		}
		return
	}
	entry.inUcast.Add(1)
}

// RecordInterfaceOutbound records a frame transmitted through one authored interface.
func (t *ProtocolTelemetry) RecordInterfaceOutbound(
	name string,
	octets int,
	nonUnicast, broadcast bool,
) {
	entry := t.interfaceEntry(name)
	if entry == nil {
		return
	}
	addOctets(&entry.outOctets, octets)
	if nonUnicast {
		entry.outNUcast.Add(1)
		if broadcast {
			entry.outBroadcast.Add(1)
		}
		return
	}
	entry.outUcast.Add(1)
}

func addOctets(counter *atomic.Uint64, octets int) {
	if octets > 0 {
		counter.Add(uint64(octets))
	}
}

// RecordInterfaceCounters adds externally generated interface telemetry.
func (t *ProtocolTelemetry) RecordInterfaceCounters(name string, delta InterfaceCounterDelta) {
	entry := t.interfaceEntry(name)
	if entry == nil {
		return
	}
	entry.inOctets.Add(delta.InOctets)
	entry.outOctets.Add(delta.OutOctets)
	entry.inDiscards.Add(delta.InDiscards)
	entry.outDiscards.Add(delta.OutDiscards)
	entry.inErrors.Add(delta.InErrors)
	entry.outErrors.Add(delta.OutErrors)
	entry.fcsErrors.Add(delta.FCSErrors)
}

func (t *ProtocolTelemetry) interfaceEntry(name string) *interfaceTelemetry {
	if t == nil || name == "" {
		return nil
	}
	entry, _ := t.interfaces.LoadOrStore(name, &interfaceTelemetry{})
	result, _ := entry.(*interfaceTelemetry)
	return result
}

func (t *ProtocolTelemetry) interfaceSnapshot(name string) interfaceSnapshot {
	entry := t.interfaceEntry(name)
	if entry == nil {
		return interfaceSnapshot{}
	}
	return snapshotInterfaceTelemetry(entry)
}

func snapshotInterfaceTelemetry(entry *interfaceTelemetry) interfaceSnapshot {
	return interfaceSnapshot{
		inOctets: entry.inOctets.Load(), inUcast: entry.inUcast.Load(), inNUcast: entry.inNUcast.Load(),
		outOctets: entry.outOctets.Load(), outUcast: entry.outUcast.Load(), outNUcast: entry.outNUcast.Load(),
		inBroadcast: entry.inBroadcast.Load(), outBroadcast: entry.outBroadcast.Load(),
		inDiscards: entry.inDiscards.Load(), outDiscards: entry.outDiscards.Load(),
		inErrors: entry.inErrors.Load(), outErrors: entry.outErrors.Load(), fcsErrors: entry.fcsErrors.Load(),
	}
}

func (a *Agent) interfaceSnapshot(name string) interfaceSnapshot {
	snapshot := a.protocolStats.interfaceSnapshot(name)
	if a.device == nil {
		return snapshot
	}
	for i := range a.device.Interfaces {
		iface := &a.device.Interfaces[i]
		if iface.Name != name || iface.Speed <= 0 {
			continue
		}
		elapsed := max(time.Since(a.startTime).Seconds(), 0)
		addSimulatedTraffic(
			&snapshot,
			trafficOctets(iface.Speed, iface.InUtilization, elapsed),
			trafficOctets(iface.Speed, iface.OutUtilization, elapsed),
		)
		break
	}
	return snapshot
}

func trafficOctets(speedMbps int, utilization, elapsedSeconds float64) uint64 {
	if utilization <= 0 || elapsedSeconds <= 0 {
		return 0
	}
	bytesPerSecond := float64(speedMbps) * MicrosPerSec / bitsPerOctet
	return uint64(bytesPerSecond * utilization / 100 * elapsedSeconds)
}

func addSimulatedTraffic(snapshot *interfaceSnapshot, inOctets, outOctets uint64) {
	snapshot.inOctets += inOctets
	snapshot.outOctets += outOctets
	addSimulatedPackets(&snapshot.inUcast, &snapshot.inNUcast, &snapshot.inBroadcast, inOctets)
	addSimulatedPackets(
		&snapshot.outUcast,
		&snapshot.outNUcast,
		&snapshot.outBroadcast,
		outOctets,
	)
}

func addSimulatedPackets(unicast, nonUnicast, broadcast *uint64, octets uint64) {
	total := octets / averageFrameOctets
	simulatedNonUnicast := total / nonUnicastPacketRatio
	*unicast += total - simulatedNonUnicast
	*nonUnicast += simulatedNonUnicast
	*broadcast += total / broadcastPacketRatio
}
