package snmp

import (
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
)

const (
	dot3StatsFCSErrors     = "1.3.6.1.2.1.10.7.2.1.3"
	dot3StatsDuplexStatus  = "1.3.6.1.2.1.10.7.2.1.19"
	interfaceStatusUp      = 1
	interfaceStatusDown    = 2
	interfaceStatusTesting = 3
	duplexHalf             = 2
)

func (a *Agent) initializeIFMIB() {
	logger := slog.Default()
	device := a.device
	if device == nil {
		return
	}

	// A capture walk carries the device's real interface table (ifIndex, names,
	// counters). Synthesizing a second, sequential ifTable from trunk_ports on
	// top of it duplicates interfaces under different indices — harmless when a
	// device has one uplink, but it pollutes the interface list once downstream
	// access ports are modelled. The walk owns IF-MIB; trunk_ports only drive the
	// neighbour/forwarding topology (see initializeLLDPRemoteMIB, SynthesizePeerTopology).
	if a.hasWalkContent() {
		return
	}

	// Count interfaces from trunk_ports
	numInterfaces := len(device.TrunkPorts)
	if numInterfaces == 0 {
		// If no trunk ports, create at least one management interface
		numInterfaces = 1
	}

	// ifNumber
	a.mib.Set(ifNumber, &OIDValue{
		Type:  gosnmp.Integer,
		Value: numInterfaces,
	})

	// Create interface entries
	if len(device.TrunkPorts) == 0 {
		// Management interface only
		a.createInterfaceEntry(1, "Management", device.MACAddress.String(), NanosPerSecond)
	} else {
		for idx, trunk := range device.TrunkPorts {
			ifIdx := idx + 1
			// Determine speed based on interface name
			speed := getInterfaceSpeed(trunk.Interface)
			a.createInterfaceEntry(ifIdx, trunk.Interface, device.MACAddress.String(), speed)
		}
	}

	if a.debugLevel >= DebugLevelMinimum {
		logger.Info("Initialized IF-MIB", "interfaces", numInterfaces, "device", device.Name)
	}
}

func (a *Agent) refreshAuthoredInterfaceMIBs() {
	for _, iface := range a.device.Interfaces {
		index, ok := a.ifIndexForInterface(iface.Name)
		if !ok {
			continue
		}
		if iface.Speed > 0 {
			speedBps := uint64(iface.Speed) * MicrosPerSec
			a.mib.Set(ifSpeed+"."+index, &OIDValue{
				Type: gosnmp.Gauge32, Value: safeUint32FromUint64(min(speedBps, MaxUint32Value)),
			})
			a.mib.Set(ifHighSpeed+"."+index, &OIDValue{
				Type: gosnmp.Gauge32, Value: safeUint32FromUint64(uint64(iface.Speed)),
			})
		}
		if status := interfaceStatus(iface.AdminStatus); status != 0 {
			a.mib.Set(ifAdminStatus+"."+index, &OIDValue{Type: gosnmp.Integer, Value: status})
		}
		if status := interfaceStatus(iface.OperStatus); status != 0 {
			a.mib.Set(ifOperStatus+"."+index, &OIDValue{Type: gosnmp.Integer, Value: status})
		}
		if iface.Description != "" {
			a.mib.Set(
				ifAlias+"."+index,
				&OIDValue{Type: gosnmp.OctetString, Value: iface.Description},
			)
		}
		a.registerIfTableCounters(iface.Name, index)
		a.registerIfXTablePacketCounters(iface.Name, index)
		switch strings.ToLower(iface.Duplex) {
		case "half":
			a.mib.Set(
				dot3StatsDuplexStatus+"."+index,
				&OIDValue{Type: gosnmp.Integer, Value: duplexHalf},
			)
		case "full":
			a.mib.Set(
				dot3StatsDuplexStatus+"."+index,
				&OIDValue{Type: gosnmp.Integer, Value: DuplexFull},
			)
		}
	}
	a.refreshDeviceStateInterfaceMIBs()
}

func interfaceStatus(status string) int {
	switch strings.ToLower(status) {
	case "up":
		return interfaceStatusUp
	case "down":
		return interfaceStatusDown
	case "testing":
		return interfaceStatusTesting
	default:
		return 0
	}
}

// createInterfaceEntry creates a single interface entry in IF-MIB with complete counters.
func (a *Agent) createInterfaceEntry(ifIdx int, interfaceName, mac string, speedBps uint64) {
	idxStr := strconv.Itoa(ifIdx)
	macBytes := parseMACBytes(mac)

	// Register ifTable basic properties
	a.registerIfTableBasicOIDs(ifIdx, idxStr, interfaceName, macBytes, speedBps)

	// Register ifTable counters (dynamic traffic simulation)
	a.registerIfTableCounters(interfaceName, idxStr)

	// Register ifXTable entries (extended interface table)
	a.registerIfXTableOIDs(ifIdx, idxStr, interfaceName, speedBps)
	a.registerIfXTablePacketCounters(interfaceName, idxStr)
}

// registerIfTableBasicOIDs registers basic ifTable OIDs (index, description, type, etc.).
func (a *Agent) registerIfTableBasicOIDs(
	ifIdx int,
	idxStr, interfaceName string,
	macBytes []byte,
	speedBps uint64,
) {
	// ifIndex
	a.mib.Set(ifIndex+"."+idxStr, &OIDValue{Type: gosnmp.Integer, Value: ifIdx})

	// ifDescr
	a.mib.Set(ifDescr+"."+idxStr, &OIDValue{Type: gosnmp.OctetString, Value: interfaceName})

	// ifType (6 = ethernetCsmacd)
	a.mib.Set(ifType+"."+idxStr, &OIDValue{Type: gosnmp.Integer, Value: EthernetCsmacdType})

	// ifMtu
	a.mib.Set(ifMtu+"."+idxStr, &OIDValue{Type: gosnmp.Integer, Value: DefaultMTU})

	// ifSpeed (32-bit, wraps for speeds > 4.29 Gbps)
	ifSpeedVal := min(speedBps, MaxUint32Value)
	a.mib.Set(
		ifSpeed+"."+idxStr,
		&OIDValue{Type: gosnmp.Gauge32, Value: safeUint32FromUint64(ifSpeedVal)},
	)

	// ifPhysAddress (MAC)
	a.mib.Set(ifPhysAddress+"."+idxStr, &OIDValue{Type: gosnmp.OctetString, Value: macBytes})

	// ifAdminStatus (1 = up)
	a.mib.Set(ifAdminStatus+"."+idxStr, &OIDValue{Type: gosnmp.Integer, Value: 1})

	// ifOperStatus (1 = up)
	a.mib.Set(ifOperStatus+"."+idxStr, &OIDValue{Type: gosnmp.Integer, Value: 1})

	// ifLastChange (dynamic - timeticks since last status change)
	a.mib.SetDynamic(ifLastChange+"."+idxStr, func() *OIDValue {
		return &OIDValue{Type: gosnmp.TimeTicks, Value: uptimeTicks(time.Since(a.startTime))}
	})

	// ifSpecific (deprecated, but some tools expect it)
	a.mib.Set(ifSpecific+"."+idxStr, &OIDValue{Type: gosnmp.ObjectIdentifier, Value: "0.0"})
}

// registerIfTableCounters registers dynamic ifTable counter OIDs (traffic simulation).
func (a *Agent) registerIfTableCounters(interfaceName, idxStr string) {
	counters := []struct {
		oid   string
		value func(interfaceSnapshot) uint64
	}{
		{ifInOctets, func(s interfaceSnapshot) uint64 { return s.inOctets }},
		{ifInUcastPkts, func(s interfaceSnapshot) uint64 { return s.inUcast }},
		{ifInNUcastPkts, func(s interfaceSnapshot) uint64 { return s.inNUcast }},
		{ifOutOctets, func(s interfaceSnapshot) uint64 { return s.outOctets }},
		{ifOutUcastPkts, func(s interfaceSnapshot) uint64 { return s.outUcast }},
		{ifOutNUcastPkts, func(s interfaceSnapshot) uint64 { return s.outNUcast }},
		{ifInDiscards, func(s interfaceSnapshot) uint64 { return s.inDiscards }},
		{ifOutDiscards, func(s interfaceSnapshot) uint64 { return s.outDiscards }},
		{ifInErrors, func(s interfaceSnapshot) uint64 { return s.inErrors }},
		{ifOutErrors, func(s interfaceSnapshot) uint64 { return s.outErrors }},
		{dot3StatsFCSErrors, func(s interfaceSnapshot) uint64 { return s.fcsErrors }},
	}
	for _, counter := range counters {
		value := counter.value
		a.mib.SetDynamic(counter.oid+"."+idxStr, func() *OIDValue {
			snapshot := a.protocolStats.interfaceSnapshot(interfaceName)
			return &OIDValue{
				Type:  gosnmp.Counter32,
				Value: wrapCounter32(value(snapshot)),
			}
		})
	}
	for _, oid := range []string{ifInUnknownProtos, ifOutQLen} {
		fullOID := oid + "." + idxStr
		if a.mib.Get(fullOID) == nil {
			a.mib.Set(fullOID, &OIDValue{Type: gosnmp.Counter32, Value: uint32(0)})
		}
	}
}

func (a *Agent) registerWalkStateFaultCounters() {
	if !a.hasWalkContent() || len(a.device.Interfaces) > 0 || a.deviceState == nil {
		return
	}
	snapshot := a.deviceState.Snapshot()
	if len(snapshot.Network.Interfaces) == 0 {
		return
	}
	index, ok := a.walkStateFaultIndex(snapshot.Network.Interfaces[0].Name)
	if !ok {
		return
	}
	a.registerFaultCountersWithBaseline(snapshot.Network.Interfaces[0].Name, index)
}

// InterfaceFaultObservable reports whether a fault on name has an IF-MIB counter surface.
func (a *Agent) InterfaceFaultObservable(name string) bool {
	if _, ok := a.InterfaceIndex(name); ok {
		return true
	}
	_, ok := a.walkStateFaultIndex(name)
	return ok
}

func (a *Agent) walkStateFaultIndex(name string) (string, bool) {
	if !a.hasWalkContent() || len(a.device.Interfaces) > 0 || a.deviceState == nil {
		return "", false
	}
	interfaces := a.deviceState.Snapshot().Network.Interfaces
	if len(interfaces) != 1 || interfaces[0].Name != name {
		return "", false
	}
	if index, ok := a.InterfaceIndex(name); ok {
		return strconv.Itoa(index), true
	}
	if interfaces[0].Address.IsValid() {
		address := interfaces[0].Address.Addr().Unmap().String()
		index := oidCounterValue(a.mib.Get(ipAdEntIfIndex + "." + address))
		if index > 0 {
			return strconv.FormatUint(index, 10), true
		}
	}
	return "", false
}

func (a *Agent) registerFaultCountersWithBaseline(interfaceName, index string) {
	counters := []struct {
		oid      string
		value    func(interfaceSnapshot) uint64
		snmpType gosnmp.Asn1BER
	}{
		{ifInOctets, func(s interfaceSnapshot) uint64 { return s.inOctets }, gosnmp.Counter32},
		{ifOutOctets, func(s interfaceSnapshot) uint64 { return s.outOctets }, gosnmp.Counter32},
		{ifHCInOctets, func(s interfaceSnapshot) uint64 { return s.inOctets }, gosnmp.Counter64},
		{ifHCOutOctets, func(s interfaceSnapshot) uint64 { return s.outOctets }, gosnmp.Counter64},
		{ifInDiscards, func(s interfaceSnapshot) uint64 { return s.inDiscards }, gosnmp.Counter32},
		{
			ifOutDiscards,
			func(s interfaceSnapshot) uint64 { return s.outDiscards },
			gosnmp.Counter32,
		},
		{ifInErrors, func(s interfaceSnapshot) uint64 { return s.inErrors }, gosnmp.Counter32},
		{ifOutErrors, func(s interfaceSnapshot) uint64 { return s.outErrors }, gosnmp.Counter32},
		{
			dot3StatsFCSErrors,
			func(s interfaceSnapshot) uint64 { return s.fcsErrors },
			gosnmp.Counter32,
		},
	}
	for _, counter := range counters {
		fullOID := counter.oid + "." + index
		baseline := oidCounterValue(a.mib.Get(fullOID))
		value := counter.value
		snmpType := counter.snmpType
		a.mib.SetDynamic(fullOID, func() *OIDValue {
			result := baseline + value(a.protocolStats.interfaceSnapshot(interfaceName))
			if snmpType == gosnmp.Counter64 {
				return &OIDValue{Type: snmpType, Value: result}
			}
			return &OIDValue{
				Type:  snmpType,
				Value: wrapCounter32(result),
			}
		})
	}
}

func oidCounterValue(value *OIDValue) uint64 {
	if value == nil {
		return 0
	}
	switch number := value.Value.(type) {
	case uint:
		return uint64(number)
	case uint32:
		return uint64(number)
	case uint64:
		return number
	case int32:
		if number > 0 {
			return uint64(number)
		}
	case int:
		if number > 0 {
			return uint64(number)
		}
	case int64:
		if number > 0 {
			return uint64(number)
		}
	}
	return 0
}

func (a *Agent) registerIfXTablePacketCounters(interfaceName, idxStr string) {
	counters := []struct {
		oid      string
		value    func(interfaceSnapshot) uint64
		snmpType gosnmp.Asn1BER
	}{
		{
			ifInMulticastPkts,
			func(s interfaceSnapshot) uint64 { return s.inNUcast - s.inBroadcast },
			gosnmp.Counter32,
		},
		{
			ifInBroadcastPkts,
			func(s interfaceSnapshot) uint64 { return s.inBroadcast },
			gosnmp.Counter32,
		},
		{
			ifOutMulticastPkts,
			func(s interfaceSnapshot) uint64 { return s.outNUcast - s.outBroadcast },
			gosnmp.Counter32,
		},
		{
			ifOutBroadcastPkts,
			func(s interfaceSnapshot) uint64 { return s.outBroadcast },
			gosnmp.Counter32,
		},
		{ifHCInOctets, func(s interfaceSnapshot) uint64 { return s.inOctets }, gosnmp.Counter64},
		{ifHCInUcastPkts, func(s interfaceSnapshot) uint64 { return s.inUcast }, gosnmp.Counter64},
		{
			ifHCInMulticastPkts,
			func(s interfaceSnapshot) uint64 { return s.inNUcast - s.inBroadcast },
			gosnmp.Counter64,
		},
		{
			ifHCInBroadcastPkts,
			func(s interfaceSnapshot) uint64 { return s.inBroadcast },
			gosnmp.Counter64,
		},
		{ifHCOutOctets, func(s interfaceSnapshot) uint64 { return s.outOctets }, gosnmp.Counter64},
		{
			ifHCOutUcastPkts,
			func(s interfaceSnapshot) uint64 { return s.outUcast },
			gosnmp.Counter64,
		},
		{
			ifHCOutMulticastPkts,
			func(s interfaceSnapshot) uint64 { return s.outNUcast - s.outBroadcast },
			gosnmp.Counter64,
		},
		{
			ifHCOutBroadcastPkts,
			func(s interfaceSnapshot) uint64 { return s.outBroadcast },
			gosnmp.Counter64,
		},
	}
	for _, counter := range counters {
		value, snmpType := counter.value, counter.snmpType
		a.mib.SetDynamic(counter.oid+"."+idxStr, func() *OIDValue {
			result := value(a.protocolStats.interfaceSnapshot(interfaceName))
			if snmpType == gosnmp.Counter32 {
				return &OIDValue{Type: snmpType, Value: wrapCounter32(result)}
			}
			return &OIDValue{Type: snmpType, Value: result}
		})
	}
}

// registerIfXTableOIDs registers ifXTable (extended interface table) OIDs.
func (a *Agent) registerIfXTableOIDs(ifIdx int, idxStr, interfaceName string, speedBps uint64) {
	// ifName - canonical interface name
	a.mib.Set(ifName+"."+idxStr, &OIDValue{Type: gosnmp.OctetString, Value: interfaceName})

	// Register 32-bit extended counters
	a.registerIfXTable32BitCounters(ifIdx, idxStr)

	// Register 64-bit high-capacity counters
	a.registerIfXTable64BitCounters(ifIdx, idxStr)

	// Register extended properties
	a.registerIfXTableProperties(idxStr, interfaceName, speedBps)
}

// registerIfXTable32BitCounters registers 32-bit ifXTable counters.
func (a *Agent) registerIfXTable32BitCounters(ifIdx int, idxStr string) {
	startTime := a.startTime

	// ifInMulticastPkts
	a.mib.SetDynamic(ifInMulticastPkts+"."+idxStr, func() *OIDValue {
		elapsed := time.Since(startTime).Seconds()
		pkts := uint32((elapsed * 5 * float64(ifIdx%TrafficDivisor+1)) / TrafficDivisor)
		return &OIDValue{Type: gosnmp.Counter32, Value: pkts}
	})

	// ifInBroadcastPkts
	a.mib.SetDynamic(ifInBroadcastPkts+"."+idxStr, func() *OIDValue {
		elapsed := time.Since(startTime).Seconds()
		pkts := uint32((elapsed * 2 * float64(ifIdx%TrafficDivisor+1)) / TrafficDivisor)
		return &OIDValue{Type: gosnmp.Counter32, Value: pkts}
	})

	// ifOutMulticastPkts
	a.mib.SetDynamic(ifOutMulticastPkts+"."+idxStr, func() *OIDValue {
		elapsed := time.Since(startTime).Seconds()
		pkts := uint32((elapsed * 3 * float64(ifIdx%TrafficDivisor+1)) / TrafficDivisor)
		return &OIDValue{Type: gosnmp.Counter32, Value: pkts}
	})

	// ifOutBroadcastPkts
	a.mib.SetDynamic(ifOutBroadcastPkts+"."+idxStr, func() *OIDValue {
		elapsed := time.Since(startTime).Seconds()
		pkts := uint32((elapsed * 1 * float64(ifIdx%TrafficDivisor+1)) / TrafficDivisor)
		return &OIDValue{Type: gosnmp.Counter32, Value: pkts}
	})
}

// registerIfXTable64BitCounters registers 64-bit high-capacity counters.
func (a *Agent) registerIfXTable64BitCounters(ifIdx int, idxStr string) {
	startTime := a.startTime

	a.mib.SetDynamic(ifHCInOctets+"."+idxStr, func() *OIDValue {
		elapsed := time.Since(startTime).Seconds()
		octets := uint64(elapsed * 1000000 * float64(ifIdx%TrafficDivisor+1) / TrafficDivisor)
		return &OIDValue{Type: gosnmp.Counter64, Value: octets}
	})

	a.mib.SetDynamic(ifHCInUcastPkts+"."+idxStr, func() *OIDValue {
		elapsed := time.Since(startTime).Seconds()
		pkts := uint64(elapsed * 1000 * float64(ifIdx%TrafficDivisor+1) / TrafficDivisor)
		return &OIDValue{Type: gosnmp.Counter64, Value: pkts}
	})

	a.mib.SetDynamic(ifHCOutOctets+"."+idxStr, func() *OIDValue {
		elapsed := time.Since(startTime).Seconds()
		octets := uint64(elapsed * 800000 * float64(ifIdx%TrafficDivisor+1) / TrafficDivisor)
		return &OIDValue{Type: gosnmp.Counter64, Value: octets}
	})

	a.mib.SetDynamic(ifHCOutUcastPkts+"."+idxStr, func() *OIDValue {
		elapsed := time.Since(startTime).Seconds()
		pkts := uint64(elapsed * 800 * float64(ifIdx%TrafficDivisor+1) / TrafficDivisor)
		return &OIDValue{Type: gosnmp.Counter64, Value: pkts}
	})
}

// registerIfXTableProperties registers ifXTable static properties.
func (a *Agent) registerIfXTableProperties(idxStr, interfaceName string, speedBps uint64) {
	// ifLinkUpDownTrapEnable (1 = enabled)
	a.mib.Set(ifLinkUpDownTrapEnable+"."+idxStr, &OIDValue{Type: gosnmp.Integer, Value: 1})

	// ifHighSpeed (speed in Mbps for high-speed interfaces)
	highSpeedMbps := speedBps / MicrosPerSec
	a.mib.Set(
		ifHighSpeed+"."+idxStr,
		&OIDValue{Type: gosnmp.Gauge32, Value: safeUint32FromUint64(highSpeedMbps)},
	)

	// ifPromiscuousMode (2 = false)
	a.mib.Set(ifPromiscuousMode+"."+idxStr, &OIDValue{Type: gosnmp.Integer, Value: TruthValueFalse})

	// ifConnectorPresent (1 = true)
	a.mib.Set(ifConnectorPresent+"."+idxStr, &OIDValue{Type: gosnmp.Integer, Value: TruthValueTrue})

	// ifAlias - interface description/alias
	a.mib.Set(ifAlias+"."+idxStr, &OIDValue{Type: gosnmp.OctetString, Value: interfaceName})

	// ifCounterDiscontinuityTime
	a.mib.Set(
		ifCounterDiscontinuityTime+"."+idxStr,
		&OIDValue{Type: gosnmp.TimeTicks, Value: uint32(0)},
	)
}
