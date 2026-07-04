package snmp

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/gosnmp/gosnmp"
)

type ifCounterConfig struct {
	oid      string
	rate     float64
	snmpType gosnmp.Asn1BER
}

// registerIfTableInputCounters registers input traffic counters.

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

// createInterfaceEntry creates a single interface entry in IF-MIB with complete counters.
func (a *Agent) createInterfaceEntry(ifIdx int, interfaceName, mac string, speedBps uint64) {
	idxStr := strconv.Itoa(ifIdx)
	macBytes := parseMACBytes(mac)

	// Register ifTable basic properties
	a.registerIfTableBasicOIDs(ifIdx, idxStr, interfaceName, macBytes, speedBps)

	// Register ifTable counters (dynamic traffic simulation)
	a.registerIfTableCounters(ifIdx, idxStr)

	// Register ifXTable entries (extended interface table)
	a.registerIfXTableOIDs(ifIdx, idxStr, interfaceName, speedBps)
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
func (a *Agent) registerIfTableCounters(ifIdx int, idxStr string) {
	startTime := a.startTime

	// Input counters
	a.registerIfTableInputCounters(ifIdx, idxStr, startTime)

	// Output counters
	a.registerIfTableOutputCounters(ifIdx, idxStr, startTime)
}

// ifCounterConfig defines configuration for a traffic counter OID.

func (a *Agent) registerIfTableInputCounters(ifIdx int, idxStr string, startTime time.Time) {
	dynamicCounters := []ifCounterConfig{
		{ifInOctets, 1000000, gosnmp.Counter32}, // ~1MB/s base traffic
		{ifInUcastPkts, 1000, gosnmp.Counter32}, // unicast packets
		{ifInNUcastPkts, 10, gosnmp.Counter32},  // broadcast/multicast
	}
	a.registerDynamicCounters(dynamicCounters, ifIdx, idxStr, startTime)

	staticCounters := []ifCounterConfig{
		{ifInDiscards, 0, gosnmp.Counter32},
		{ifInErrors, 0, gosnmp.Counter32},
		{ifInUnknownProtos, 0, gosnmp.Counter32},
	}
	a.registerStaticZeroCounters(staticCounters, idxStr)
}

// registerIfTableOutputCounters registers output traffic counters.
func (a *Agent) registerIfTableOutputCounters(ifIdx int, idxStr string, startTime time.Time) {
	dynamicCounters := []ifCounterConfig{
		{ifOutOctets, 800000, gosnmp.Counter32}, // ~800KB/s base traffic
		{ifOutUcastPkts, 800, gosnmp.Counter32}, // unicast packets
		{ifOutNUcastPkts, 5, gosnmp.Counter32},  // broadcast/multicast
	}
	a.registerDynamicCounters(dynamicCounters, ifIdx, idxStr, startTime)

	staticCounters := []ifCounterConfig{
		{ifOutDiscards, 0, gosnmp.Counter32},
		{ifOutErrors, 0, gosnmp.Counter32},
		{ifOutQLen, 0, gosnmp.Gauge32},
	}
	a.registerStaticZeroCounters(staticCounters, idxStr)
}

// registerDynamicCounters registers dynamic traffic counters with time-based values.
func (a *Agent) registerDynamicCounters(
	counters []ifCounterConfig,
	ifIdx int,
	idxStr string,
	startTime time.Time,
) {
	for _, c := range counters {
		oid := c.oid + "." + idxStr
		rate := c.rate
		snmpType := c.snmpType
		a.mib.SetDynamic(oid, func() *OIDValue {
			elapsed := time.Since(startTime).Seconds()
			value := uint32((elapsed * rate * float64(ifIdx%TrafficDivisor+1)) / TrafficDivisor)
			return &OIDValue{Type: snmpType, Value: value}
		})
	}
}

// registerStaticZeroCounters registers static zero-valued counters.
func (a *Agent) registerStaticZeroCounters(counters []ifCounterConfig, idxStr string) {
	for _, c := range counters {
		a.mib.Set(c.oid+"."+idxStr, &OIDValue{Type: c.snmpType, Value: uint32(0)})
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
