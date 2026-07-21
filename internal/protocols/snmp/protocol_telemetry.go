package snmp

import (
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gosnmp/gosnmp"
)

const (
	protocolICMP         = 1
	protocolTCP          = 6
	protocolUDP          = 17
	tcpAddressIndexParts = 10
)

// ProtocolEvent describes one valid IPv4 datagram delivered to or transmitted
// by a simulated device. Transport details are optional and only drive the
// matching RFC 1213 counters.
type ProtocolEvent struct {
	Protocol        uint8
	ICMPType        uint8
	TCPSYN          bool
	TCPACK          bool
	TCPRST          bool
	TCPFIN          bool
	SourceIP        string
	DestinationIP   string
	SourcePort      uint16
	DestinationPort uint16
	MoreFragments   bool
	FragmentOffset  uint16
}

// ProtocolTelemetry is the per-device event source shared by every community
// agent for that device.
type ProtocolTelemetry struct {
	ipInReceives, ipInDelivers, ipInUnknownProtos, ipForwDatagrams, ipOutRequests atomic.Uint32
	ipReasmReqds, ipReasmOKs, ipReasmFails, ipFragCreates                         atomic.Uint32
	icmpInMsgs, icmpOutMsgs                                                       atomic.Uint32
	icmpInTypes, icmpOutTypes                                                     [19]atomic.Uint32
	tcpActiveOpens, tcpPassiveOpens, tcpInSegs, tcpOutSegs, tcpOutRsts            atomic.Uint32
	tcpAttemptFails, tcpEstabResets                                               atomic.Uint32
	tcpFlows                                                                      *tcpFlowTracker
	udpInDatagrams, udpNoPorts, udpOutDatagrams                                   atomic.Uint32
	interfaces                                                                    sync.Map
	mibsMu                                                                        sync.Mutex
	mibs                                                                          map[*MIB]struct{}
	dynamicTCP                                                                    map[*MIB]map[string]struct{}
}

type interfaceTelemetry struct {
	inOctets, inUcast, inNUcast, outOctets, outUcast, outNUcast atomic.Uint64
	inBroadcast, outBroadcast                                   atomic.Uint64
}

type interfaceSnapshot struct {
	inOctets, inUcast, inNUcast, outOctets, outUcast, outNUcast uint64
	inBroadcast, outBroadcast                                   uint64
}

// NewProtocolTelemetry creates an empty per-device telemetry source.
func NewProtocolTelemetry() *ProtocolTelemetry {
	return &ProtocolTelemetry{
		tcpFlows: newTCPFlowTracker(), mibs: make(map[*MIB]struct{}),
		dynamicTCP: make(map[*MIB]map[string]struct{}),
	}
}

func (t *ProtocolTelemetry) attachMIB(mib *MIB) {
	if t == nil || mib == nil {
		return
	}
	t.mibsMu.Lock()
	t.mibs[mib] = struct{}{}
	t.mibsMu.Unlock()
	t.refreshTCPTable()
}

func (t *ProtocolTelemetry) refreshTCPTable() {
	if t == nil || t.tcpFlows == nil {
		return
	}
	entries := make(map[string]*OIDValue)
	for _, flow := range t.tcpFlows.snapshot() {
		base := tcpMIBRoot + ".13.1." + tcpAddressIndex(flow.key)
		entries[base+".1"] = &OIDValue{Type: gosnmp.Integer, Value: flow.state}
		entries[base+".2"] = &OIDValue{Type: gosnmp.IPAddress, Value: flow.key.localIP}
		entries[base+".3"] = &OIDValue{Type: gosnmp.Integer, Value: flow.key.localPort}
		entries[base+".4"] = &OIDValue{Type: gosnmp.IPAddress, Value: flow.key.remoteIP}
		entries[base+".5"] = &OIDValue{Type: gosnmp.Integer, Value: flow.key.remotePort}
	}
	t.mibsMu.Lock()
	defer t.mibsMu.Unlock()
	for mib := range t.mibs {
		for oid := range t.dynamicTCP[mib] {
			mib.Delete(oid)
		}
		t.dynamicTCP[mib] = make(map[string]struct{}, len(entries))
		for oid, value := range entries {
			mib.Set(oid, value)
			t.dynamicTCP[mib][oid] = struct{}{}
		}
	}
}

func tcpAddressIndex(key tcpFlowKey) string {
	parts := make([]string, 0, tcpAddressIndexParts)
	parts = append(parts, strings.Split(key.localIP, ".")...)
	parts = append(parts, strconv.Itoa(int(key.localPort)))
	parts = append(parts, strings.Split(key.remoteIP, ".")...)
	parts = append(parts, strconv.Itoa(int(key.remotePort)))
	return strings.Join(parts, ".")
}

// RecordInterfaceInbound records a frame delivered through one authored interface.
func (t *ProtocolTelemetry) RecordInterfaceInbound(name string, octets int, nonUnicast, broadcast bool) {
	entry := t.interfaceEntry(name)
	if entry == nil {
		return
	}
	if octets > 0 {
		entry.inOctets.Add(uint64(octets))
	}
	if nonUnicast {
		entry.inNUcast.Add(1)
		if broadcast {
			entry.inBroadcast.Add(1)
		}
	} else {
		entry.inUcast.Add(1)
	}
}

// RecordInterfaceOutbound records a frame transmitted through one authored interface.
func (t *ProtocolTelemetry) RecordInterfaceOutbound(name string, octets int, nonUnicast, broadcast bool) {
	entry := t.interfaceEntry(name)
	if entry == nil {
		return
	}
	if octets > 0 {
		entry.outOctets.Add(uint64(octets))
	}
	if nonUnicast {
		entry.outNUcast.Add(1)
		if broadcast {
			entry.outBroadcast.Add(1)
		}
	} else {
		entry.outUcast.Add(1)
	}
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
	return interfaceSnapshot{
		inOctets: entry.inOctets.Load(), inUcast: entry.inUcast.Load(), inNUcast: entry.inNUcast.Load(),
		outOctets: entry.outOctets.Load(), outUcast: entry.outUcast.Load(), outNUcast: entry.outNUcast.Load(),
		inBroadcast: entry.inBroadcast.Load(), outBroadcast: entry.outBroadcast.Load(),
	}
}

// RecordForwarded records a datagram received and forwarded by an L3 device.
func (t *ProtocolTelemetry) RecordForwarded() {
	if t == nil {
		return
	}
	t.ipInReceives.Add(1)
	t.ipForwDatagrams.Add(1)
}

// RecordUDPNoPort records a valid datagram addressed to a closed UDP port.
func (t *ProtocolTelemetry) RecordUDPNoPort() {
	if t != nil {
		t.udpNoPorts.Add(1)
	}
}

// RecordReassemblySuccess records delivery of one completed IPv4 datagram.
func (t *ProtocolTelemetry) RecordReassemblySuccess(event ProtocolEvent) {
	if t == nil {
		return
	}
	t.ipReasmOKs.Add(1)
	event.MoreFragments, event.FragmentOffset = false, 0
	t.recordDelivered(event)
}

// RecordReassemblyFailures records invalid or expired fragment sets.
func (t *ProtocolTelemetry) RecordReassemblyFailures(count uint32) {
	if t != nil {
		t.ipReasmFails.Add(count)
	}
}

// RecordInbound records one valid IPv4 datagram delivered to the device.
func (t *ProtocolTelemetry) RecordInbound(event ProtocolEvent) {
	if t == nil {
		return
	}
	t.ipInReceives.Add(1)
	if event.MoreFragments || event.FragmentOffset > 0 {
		t.ipReasmReqds.Add(1)

		return
	}
	t.recordDelivered(event)
}

func (t *ProtocolTelemetry) recordDelivered(event ProtocolEvent) {
	t.ipInDelivers.Add(1)
	switch event.Protocol {
	case protocolICMP:
		t.icmpInMsgs.Add(1)
		if int(event.ICMPType) < len(t.icmpInTypes) {
			t.icmpInTypes[event.ICMPType].Add(1)
		}
	case protocolTCP:
		t.tcpInSegs.Add(1)
		attemptFails, estabResets, _ := t.tcpFlows.record(event, true)
		t.tcpAttemptFails.Add(attemptFails)
		t.tcpEstabResets.Add(estabResets)
	case protocolUDP:
		t.udpInDatagrams.Add(1)
	default:
		t.ipInUnknownProtos.Add(1)
	}
	t.refreshTCPTable()
}

// RecordOutbound records one successfully transmitted IPv4 datagram.
func (t *ProtocolTelemetry) RecordOutbound(event ProtocolEvent) {
	if t == nil {
		return
	}
	t.ipOutRequests.Add(1)
	if event.MoreFragments || event.FragmentOffset > 0 {
		t.ipFragCreates.Add(1)
	}
	switch event.Protocol {
	case protocolICMP:
		t.icmpOutMsgs.Add(1)
		if int(event.ICMPType) < len(t.icmpOutTypes) {
			t.icmpOutTypes[event.ICMPType].Add(1)
		}
	case protocolTCP:
		t.tcpOutSegs.Add(1)
		attemptFails, estabResets, opened := t.tcpFlows.record(event, false)
		t.tcpAttemptFails.Add(attemptFails)
		t.tcpEstabResets.Add(estabResets)
		if event.TCPSYN {
			if event.TCPACK && opened {
				t.tcpPassiveOpens.Add(1)
			} else if !event.TCPACK {
				t.tcpActiveOpens.Add(1)
			}
		}
		if event.TCPRST {
			t.tcpOutRsts.Add(1)
		}
	case protocolUDP:
		t.udpOutDatagrams.Add(1)
	}
	t.refreshTCPTable()
}
