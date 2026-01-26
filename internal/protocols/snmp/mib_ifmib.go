package snmp

import (
	"strconv"
	"time"

	"github.com/gosnmp/gosnmp"
)

// IF-MIB OID prefixes.
const (
	// ifMIB (1.3.6.1.2.1.2).
	ifMIBBase = "1.3.6.1.2.1.2"
	ifNumber  = ifMIBBase + ".1.0" // Number of interfaces

	// ifTable (1.3.6.1.2.1.2.2)
	// Index: ifIndex.
	ifTable           = ifMIBBase + ".2"
	ifEntry           = ifTable + ".1"
	ifIndex           = ifEntry + ".1"  // Interface index
	ifDescr           = ifEntry + ".2"  // Interface description
	ifType            = ifEntry + ".3"  // Interface type (6=ethernetCsmacd)
	ifMtu             = ifEntry + ".4"  // MTU
	ifSpeed           = ifEntry + ".5"  // Speed (bps) - 32-bit, may overflow for high speeds
	ifPhysAddress     = ifEntry + ".6"  // MAC address
	ifAdminStatus     = ifEntry + ".7"  // Admin status (up=1, down=2, testing=3)
	ifOperStatus      = ifEntry + ".8"  // Oper status (up=1, down=2, etc.)
	ifLastChange      = ifEntry + ".9"  // Last state change (timeticks)
	ifInOctets        = ifEntry + ".10" // Input octets (bytes)
	ifInUcastPkts     = ifEntry + ".11" // Input unicast packets
	ifInNUcastPkts    = ifEntry + ".12" // Input non-unicast packets (broadcast/multicast)
	ifInDiscards      = ifEntry + ".13" // Input discards
	ifInErrors        = ifEntry + ".14" // Input errors
	ifInUnknownProtos = ifEntry + ".15" // Unknown protocol packets
	ifOutOctets       = ifEntry + ".16" // Output octets
	ifOutUcastPkts    = ifEntry + ".17" // Output unicast packets
	ifOutNUcastPkts   = ifEntry + ".18" // Output non-unicast packets
	ifOutDiscards     = ifEntry + ".19" // Output discards
	ifOutErrors       = ifEntry + ".20" // Output errors
	ifOutQLen         = ifEntry + ".21" // Output queue length
	ifSpecific        = ifEntry + ".22" // Specific pointer (deprecated)

	// ifXTable (1.3.6.1.2.1.31.1.1) - Extended interface table for high-speed interfaces.
	ifMIBObjects               = "1.3.6.1.2.1.31.1"
	ifXTable                   = ifMIBObjects + ".1"
	ifXEntry                   = ifXTable + ".1"
	ifName                     = ifXEntry + ".1"  // Interface name (canonical)
	ifInMulticastPkts          = ifXEntry + ".2"  // Input multicast packets
	ifInBroadcastPkts          = ifXEntry + ".3"  // Input broadcast packets
	ifOutMulticastPkts         = ifXEntry + ".4"  // Output multicast packets
	ifOutBroadcastPkts         = ifXEntry + ".5"  // Output broadcast packets
	ifHCInOctets               = ifXEntry + ".6"  // Input octets (64-bit)
	ifHCInUcastPkts            = ifXEntry + ".7"  // Input unicast packets (64-bit)
	ifHCInMulticastPkts        = ifXEntry + ".8"  // Input multicast packets (64-bit)
	ifHCInBroadcastPkts        = ifXEntry + ".9"  // Input broadcast packets (64-bit)
	ifHCOutOctets              = ifXEntry + ".10" // Output octets (64-bit)
	ifHCOutUcastPkts           = ifXEntry + ".11" // Output unicast packets (64-bit)
	ifHCOutMulticastPkts       = ifXEntry + ".12" // Output multicast packets (64-bit)
	ifHCOutBroadcastPkts       = ifXEntry + ".13" // Output broadcast packets (64-bit)
	ifLinkUpDownTrapEnable     = ifXEntry + ".14" // Link trap enable
	ifHighSpeed                = ifXEntry + ".15" // Interface speed in Mbps (for high-speed)
	ifPromiscuousMode          = ifXEntry + ".16" // Promiscuous mode
	ifConnectorPresent         = ifXEntry + ".17" // Physical connector present
	ifAlias                    = ifXEntry + ".18" // Interface alias/description
	ifCounterDiscontinuityTime = ifXEntry + ".19" // Counter discontinuity time
)

// initializeIFMIB populates IF-MIB interface table.
func (a *Agent) initializeIFMIB() {
	device := a.device
	if device == nil {
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
		a.logger.Info("Initialized IF-MIB", "interfaces", numInterfaces, "device", device.Name)
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
type ifCounterConfig struct {
	oid      string
	rate     float64
	snmpType gosnmp.Asn1BER
}

// registerIfTableInputCounters registers input traffic counters.
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
