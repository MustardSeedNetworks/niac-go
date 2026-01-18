package snmp

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/krisarmstrong/niac-go/pkg/config"
)

// LLDP-MIB OID prefixes (IEEE 802.1AB)
// Reference: http://www.ieee802.org/1/files/public/MIBs/LLDP-MIB-200505060000Z.txt
const (
	// lldpMIB (1.0.8802.1.1.2).
	lldpMIBBase = "1.0.8802.1.1.2"

	// lldpObjects (1.0.8802.1.1.2.1).
	lldpObjects = lldpMIBBase + ".1"

	// lldpLocalSystemData (1.0.8802.1.1.2.1.3).
	lldpLocalSystemData     = lldpObjects + ".3"
	lldpLocChassisIDSubtype = lldpLocalSystemData + ".1.0"
	lldpLocChassisID        = lldpLocalSystemData + ".2.0"
	lldpLocSysName          = lldpLocalSystemData + ".3.0"
	lldpLocSysDesc          = lldpLocalSystemData + ".4.0"
	lldpLocSysCapSupported  = lldpLocalSystemData + ".5.0"
	lldpLocSysCapEnabled    = lldpLocalSystemData + ".6.0"

	// lldpLocPortTable (1.0.8802.1.1.2.1.3.7).
	lldpLocPortTable = lldpLocalSystemData + ".7"

	// lldpRemoteSystemsData (1.0.8802.1.1.2.1.4).
	lldpRemoteSystemsData = lldpObjects + ".4"

	// lldpRemTable (1.0.8802.1.1.2.1.4.1)
	// Index: lldpRemTimeMark, lldpRemLocalPortNum, lldpRemIndex.
	lldpRemTable = lldpRemoteSystemsData + ".1"
)

// CDP-MIB OID prefixes (Cisco Discovery Protocol)
// Reference: https://www.cisco.com/c/en/us/td/docs/net_mgmt/prime/network/3-8/reference/guide/CiscoPrimeNetworkMIBs/CiscoCdpMib.html
const (
	// ciscoCdpMIB (1.3.6.1.4.1.9.9.23).
	cdpMIBBase = "1.3.6.1.4.1.9.9.23"

	// cdpGlobal (1.3.6.1.4.1.9.9.23.1.1).
	cdpGlobal                = cdpMIBBase + ".1.1"
	cdpGlobalRun             = cdpGlobal + ".1.0" // Is CDP running (TruthValue)
	cdpGlobalMessageInterval = cdpGlobal + ".2.0" // CDP message interval (seconds)
	cdpGlobalHoldTime        = cdpGlobal + ".3.0" // CDP holdtime (seconds)
	cdpGlobalDeviceID        = cdpGlobal + ".6.0" // Local device ID

	// cdpCache (1.3.6.1.4.1.9.9.23.1.2).
	cdpCache = cdpMIBBase + ".1.2"

	// cdpCacheTable (1.3.6.1.4.1.9.9.23.1.2.1)
	// Index: cdpCacheIfIndex, cdpCacheDeviceIndex.
	cdpCacheTable = cdpCache + ".1"
)

// IP-MIB OID prefixes.
const (
	// ip (1.3.6.1.2.1.4).
	ipMIBBase = "1.3.6.1.2.1.4"

	// ipAddrTable (1.3.6.1.2.1.4.20).
	ipAddrTable         = ipMIBBase + ".20"
	ipAddrEntry         = ipAddrTable + ".1"
	ipAdEntAddr         = ipAddrEntry + ".1" // IP address (index)
	ipAdEntIfIndex      = ipAddrEntry + ".2" // Interface index
	ipAdEntNetMask      = ipAddrEntry + ".3" // Subnet mask
	ipAdEntBcastAddr    = ipAddrEntry + ".4" // Broadcast address LSB (0 or 1)
	ipAdEntReasmMaxSize = ipAddrEntry + ".5" // Max reassembly size

	// ipRouteTable (1.3.6.1.2.1.4.21) - deprecated but still commonly used.
	ipRouteTable   = ipMIBBase + ".21"
	ipRouteEntry   = ipRouteTable + ".1"
	ipRouteDest    = ipRouteEntry + ".1"  // Destination
	ipRouteIfIndex = ipRouteEntry + ".2"  // Interface index
	ipRouteMetric1 = ipRouteEntry + ".3"  // Metric
	ipRouteNextHop = ipRouteEntry + ".7"  // Next hop
	ipRouteType    = ipRouteEntry + ".8"  // Route type (1=other,2=invalid,3=direct,4=indirect)
	ipRouteProto   = ipRouteEntry + ".9"  // Protocol (1=other,2=local,3=netmgmt,4=icmp,etc.)
	ipRouteMask    = ipRouteEntry + ".11" // Subnet mask

	// ipNetToMediaTable (1.3.6.1.2.1.4.22) - ARP table.
	ipNetToMediaTable       = ipMIBBase + ".22"
	ipNetToMediaEntry       = ipNetToMediaTable + ".1"
	ipNetToMediaIfIndex     = ipNetToMediaEntry + ".1" // Interface index (index)
	ipNetToMediaPhysAddress = ipNetToMediaEntry + ".2" // MAC address
	ipNetToMediaNetAddress  = ipNetToMediaEntry + ".3" // IP address (index)
	ipNetToMediaType        = ipNetToMediaEntry + ".4" // Type (1=other,2=invalid,3=dynamic,4=static)
)

// BRIDGE-MIB OID prefixes (dot1dBridge).
const (
	// dot1dBridge (1.3.6.1.2.1.17).
	dot1dBridge = "1.3.6.1.2.1.17"

	// dot1dBase (1.3.6.1.2.1.17.1).
	dot1dBase              = dot1dBridge + ".1"
	dot1dBaseBridgeAddress = dot1dBase + ".1.0" // Bridge MAC address
	dot1dBaseNumPorts      = dot1dBase + ".2.0" // Number of ports
	dot1dBaseType          = dot1dBase + ".3.0" // Bridge type (1=unknown,2=transparent-only,3=sourceroute-only,4=srt)

	// dot1dBasePortTable (1.3.6.1.2.1.17.1.4).
	dot1dBasePortTable                 = dot1dBase + ".4"
	dot1dBasePortEntry                 = dot1dBasePortTable + ".1"
	dot1dBasePort                      = dot1dBasePortEntry + ".1" // Port number (index)
	dot1dBasePortIfIndex               = dot1dBasePortEntry + ".2" // IF-MIB interface index
	dot1dBasePortCircuit               = dot1dBasePortEntry + ".3" // Circuit ID
	dot1dBasePortDelayExceededDiscards = dot1dBasePortEntry + ".4"
	dot1dBasePortMtuExceededDiscards   = dot1dBasePortEntry + ".5"

	// dot1dStp (1.3.6.1.2.1.17.2) - Spanning Tree.
	dot1dStp                        = dot1dBridge + ".2"
	dot1dStpProtocolSpecification   = dot1dStp + ".1.0" // STP spec (1=unknown,2=decLb100,3=ieee8021d)
	dot1dStpPriority                = dot1dStp + ".2.0" // Bridge priority
	dot1dStpTimeSinceTopologyChange = dot1dStp + ".3.0"
	dot1dStpTopChanges              = dot1dStp + ".4.0"
	dot1dStpDesignatedRoot          = dot1dStp + ".5.0"
	dot1dStpRootCost                = dot1dStp + ".6.0"
	dot1dStpRootPort                = dot1dStp + ".7.0"
	dot1dStpMaxAge                  = dot1dStp + ".8.0" // hundredths of second
	dot1dStpHelloTime               = dot1dStp + ".9.0"
	dot1dStpHoldTime                = dot1dStp + ".10.0"
	dot1dStpForwardDelay            = dot1dStp + ".11.0"
	dot1dStpBridgeMaxAge            = dot1dStp + ".12.0"
	dot1dStpBridgeHelloTime         = dot1dStp + ".13.0"
	dot1dStpBridgeForwardDelay      = dot1dStp + ".14.0"

	// dot1dStpPortTable (1.3.6.1.2.1.17.2.15).
	dot1dStpPortTable              = dot1dStp + ".15"
	dot1dStpPortEntry              = dot1dStpPortTable + ".1"
	dot1dStpPort                   = dot1dStpPortEntry + ".1" // Port number (index)
	dot1dStpPortPriority           = dot1dStpPortEntry + ".2"
	dot1dStpPortState              = dot1dStpPortEntry + ".3" // 1=disabled,2=blocking,3=listening,4=learning,5=forwarding,6=broken
	dot1dStpPortEnable             = dot1dStpPortEntry + ".4" // 1=enabled,2=disabled
	dot1dStpPortPathCost           = dot1dStpPortEntry + ".5"
	dot1dStpPortDesignatedRoot     = dot1dStpPortEntry + ".6"
	dot1dStpPortDesignatedCost     = dot1dStpPortEntry + ".7"
	dot1dStpPortDesignatedBridge   = dot1dStpPortEntry + ".8"
	dot1dStpPortDesignatedPort     = dot1dStpPortEntry + ".9"
	dot1dStpPortForwardTransitions = dot1dStpPortEntry + ".10"

	// dot1dTp (1.3.6.1.2.1.17.4) - Transparent Bridging.
	dot1dTp                     = dot1dBridge + ".4"
	dot1dTpLearnedEntryDiscards = dot1dTp + ".1.0"
	dot1dTpAgingTime            = dot1dTp + ".2.0"

	// dot1dTpFdbTable (1.3.6.1.2.1.17.4.3) - Forwarding Database.
	dot1dTpFdbTable   = dot1dTp + ".3"
	dot1dTpFdbEntry   = dot1dTpFdbTable + ".1"
	dot1dTpFdbAddress = dot1dTpFdbEntry + ".1" // MAC address (index)
	dot1dTpFdbPort    = dot1dTpFdbEntry + ".2" // Port learned on
	dot1dTpFdbStatus  = dot1dTpFdbEntry + ".3" // 1=other,2=invalid,3=learned,4=self,5=mgmt

	// dot1dTpPortTable (1.3.6.1.2.1.17.4.4).
	dot1dTpPortTable      = dot1dTp + ".4"
	dot1dTpPortEntry      = dot1dTpPortTable + ".1"
	dot1dTpPort           = dot1dTpPortEntry + ".1" // Port number (index)
	dot1dTpPortMaxInfo    = dot1dTpPortEntry + ".2" // Max frame size
	dot1dTpPortInFrames   = dot1dTpPortEntry + ".3"
	dot1dTpPortOutFrames  = dot1dTpPortEntry + ".4"
	dot1dTpPortInDiscards = dot1dTpPortEntry + ".5"
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

// initializeNeighborMIBs initializes LLDP-MIB, CDP-MIB, IF-MIB, IP-MIB, and BRIDGE-MIB based on device configuration.
func (a *Agent) initializeNeighborMIBs() {
	device := a.device
	if device == nil {
		return
	}

	// Initialize IF-MIB (interface table)
	a.initializeIFMIB()

	// Initialize IP-MIB (IP address table, routing table)
	a.initializeIPMIB()

	// Initialize BRIDGE-MIB (dot1dBridge - STP and FDB)
	a.initializeBridgeMIB()

	// Initialize LLDP local system data
	if device.LLDPConfig != nil && device.LLDPConfig.Enabled {
		a.initializeLLDPLocalMIB()
		a.initializeLLDPRemoteMIB()
	}

	// Initialize CDP global and cache
	if device.CDPConfig != nil && device.CDPConfig.Enabled {
		a.initializeCDPMIB()
	}
}

// initializeIFMIB populates IF-MIB interface table.
func (a *Agent) initializeIFMIB() {
	logger := slog.Default()
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

// initializeIPMIB populates IP-MIB tables (ipAddrTable, ipRouteTable, ipNetToMediaTable).
func (a *Agent) initializeIPMIB() {
	logger := slog.Default()
	device := a.device
	if device == nil {
		return
	}

	ips := device.IPAddresses
	if len(ips) == 0 {
		return
	}

	// Register ipAddrTable entries
	a.registerIPAddrTableEntries(device, ips)

	// Register default route
	a.registerDefaultRouteEntry(ips)

	// Register ARP table entries
	a.registerARPTableEntries(device)

	if a.debugLevel >= DebugLevelMinimum {
		logger.Info("Initialized IP-MIB", "device", device.Name, "ipAddresses", len(ips))
	}
}

// registerIPAddrTableEntries registers ipAddrTable entries for each IPv4 address.
func (a *Agent) registerIPAddrTableEntries(device *config.Device, ips []net.IP) {
	for idx, ip := range ips {
		// Skip IPv6 addresses for ipAddrTable (IPv4 only)
		if ip.To4() == nil {
			continue
		}

		ipStr := ip.String()
		ifIndex := 1
		if idx < len(device.TrunkPorts) {
			ifIndex = idx + 1
		}

		a.registerIPAddrEntry(ipStr, ifIndex)
	}
}

// registerIPAddrEntry registers a single ipAddrTable entry.
func (a *Agent) registerIPAddrEntry(ipStr string, ifIndex int) {
	a.mib.Set(ipAdEntAddr+"."+ipStr, &OIDValue{Type: gosnmp.IPAddress, Value: ipStr})
	a.mib.Set(ipAdEntIfIndex+"."+ipStr, &OIDValue{Type: gosnmp.Integer, Value: ifIndex})
	a.mib.Set(ipAdEntNetMask+"."+ipStr, &OIDValue{Type: gosnmp.IPAddress, Value: "255.255.255.0"})
	a.mib.Set(ipAdEntBcastAddr+"."+ipStr, &OIDValue{Type: gosnmp.Integer, Value: 1})
	a.mib.Set(ipAdEntReasmMaxSize+"."+ipStr, &OIDValue{Type: gosnmp.Integer, Value: MaxIPPort})
}

// registerDefaultRouteEntry registers the default route (0.0.0.0) in ipRouteTable.
func (a *Agent) registerDefaultRouteEntry(ips []net.IP) {
	const defaultRoute = "0.0.0.0"

	a.mib.Set(ipRouteDest+"."+defaultRoute, &OIDValue{Type: gosnmp.IPAddress, Value: defaultRoute})
	a.mib.Set(ipRouteIfIndex+"."+defaultRoute, &OIDValue{Type: gosnmp.Integer, Value: 1})
	a.mib.Set(ipRouteMetric1+"."+defaultRoute, &OIDValue{Type: gosnmp.Integer, Value: 1})

	// Compute default gateway from first IPv4 address
	if len(ips) > 0 && ips[0].To4() != nil {
		ip4 := ips[0].To4()
		gateway := fmt.Sprintf("%d.%d.%d.1", ip4[0], ip4[1], ip4[2])
		a.mib.Set(
			ipRouteNextHop+"."+defaultRoute,
			&OIDValue{Type: gosnmp.IPAddress, Value: gateway},
		)
	}

	a.mib.Set(ipRouteType+"."+defaultRoute, &OIDValue{Type: gosnmp.Integer, Value: IPRouteTypeIndirect})
	a.mib.Set(ipRouteMask+"."+defaultRoute, &OIDValue{Type: gosnmp.IPAddress, Value: defaultRoute})
	a.mib.Set(ipRouteProto+"."+defaultRoute, &OIDValue{Type: gosnmp.Integer, Value: IPRouteProtoLocal})
}

// registerARPTableEntries registers ipNetToMediaTable (ARP) entries for neighbors.
func (a *Agent) registerARPTableEntries(device *config.Device) {
	arpIndex := 1
	for _, trunk := range device.TrunkPorts {
		if trunk.RemoteDevice == "" {
			continue
		}
		a.registerARPEntry(arpIndex)
		arpIndex++
	}
}

// registerARPEntry registers a single ARP table entry.
func (a *Agent) registerARPEntry(arpIndex int) {
	neighborIP := fmt.Sprintf("10.0.0.%d", arpIndex)
	indexStr := fmt.Sprintf("%d.%s", arpIndex, neighborIP)

	a.mib.Set(ipNetToMediaIfIndex+"."+indexStr, &OIDValue{Type: gosnmp.Integer, Value: arpIndex})
	macBytes := []byte{0x00, 0x00, 0x00, 0x00, 0x00, byte(arpIndex)}
	a.mib.Set(
		ipNetToMediaPhysAddress+"."+indexStr,
		&OIDValue{Type: gosnmp.OctetString, Value: macBytes},
	)
	a.mib.Set(
		ipNetToMediaNetAddress+"."+indexStr,
		&OIDValue{Type: gosnmp.IPAddress, Value: neighborIP},
	)
	a.mib.Set(ipNetToMediaType+"."+indexStr, &OIDValue{Type: gosnmp.Integer, Value: ARPTypeDynamic})
}

// initializeBridgeMIB populates BRIDGE-MIB (dot1dBridge) for switches.
func (a *Agent) initializeBridgeMIB() {
	logger := slog.Default()
	device := a.device
	if device == nil {
		return
	}

	numPorts := len(device.TrunkPorts)
	if numPorts == 0 {
		numPorts = 1
	}

	macBytes := parseMACBytes(device.MACAddress.String())

	// Register dot1dBase group
	a.registerDot1dBaseGroup(numPorts, macBytes)

	// Register dot1dStp group if STP is enabled
	if device.STPConfig != nil && device.STPConfig.Enabled {
		a.registerDot1dStpGroup(device.STPConfig, numPorts, macBytes)
	}

	// Register dot1dTp group (transparent bridging)
	a.registerDot1dTpGroup(device, numPorts, macBytes)

	if a.debugLevel >= DebugLevelMinimum {
		logger.Info("Initialized BRIDGE-MIB", "device", device.Name, "ports", numPorts)
	}
}

// registerDot1dBaseGroup registers dot1dBase OIDs.
func (a *Agent) registerDot1dBaseGroup(numPorts int, macBytes []byte) {
	a.mib.Set(dot1dBaseBridgeAddress, &OIDValue{Type: gosnmp.OctetString, Value: macBytes})
	a.mib.Set(dot1dBaseNumPorts, &OIDValue{Type: gosnmp.Integer, Value: numPorts})
	a.mib.Set(dot1dBaseType, &OIDValue{Type: gosnmp.Integer, Value: BridgeTypeTransparent})

	for portIdx := 1; portIdx <= numPorts; portIdx++ {
		a.registerDot1dBasePortEntry(portIdx)
	}
}

// registerDot1dBasePortEntry registers a single dot1dBasePortTable entry.
func (a *Agent) registerDot1dBasePortEntry(portIdx int) {
	portStr := strconv.Itoa(portIdx)
	a.mib.Set(dot1dBasePort+"."+portStr, &OIDValue{Type: gosnmp.Integer, Value: portIdx})
	a.mib.Set(dot1dBasePortIfIndex+"."+portStr, &OIDValue{Type: gosnmp.Integer, Value: portIdx})
	a.mib.Set(
		dot1dBasePortCircuit+"."+portStr,
		&OIDValue{Type: gosnmp.ObjectIdentifier, Value: "0.0"},
	)
	a.mib.Set(
		dot1dBasePortDelayExceededDiscards+"."+portStr,
		&OIDValue{Type: gosnmp.Counter32, Value: uint32(0)},
	)
	a.mib.Set(
		dot1dBasePortMtuExceededDiscards+"."+portStr,
		&OIDValue{Type: gosnmp.Counter32, Value: uint32(0)},
	)
}

// registerDot1dStpGroup registers dot1dStp OIDs when STP is enabled.
func (a *Agent) registerDot1dStpGroup(stp *config.STPConfig, numPorts int, macBytes []byte) {
	priority := int(stp.BridgePriority)
	if priority == 0 {
		priority = 32768
	}

	bridgeID := a.buildBridgeID(priority, macBytes)

	// Register STP scalars
	a.registerDot1dStpScalars(stp, priority, bridgeID)

	// Register STP port table
	for portIdx := 1; portIdx <= numPorts; portIdx++ {
		a.registerDot1dStpPortEntry(portIdx, bridgeID)
	}
}

// buildBridgeID constructs an 8-byte bridge ID from priority and MAC.
func (a *Agent) buildBridgeID(priority int, macBytes []byte) []byte {
	bridgeID := make([]byte, BridgeIDLength)
	bridgeID[0] = byte(priority >> BitShiftByte)
	bridgeID[1] = byte(priority)
	copy(bridgeID[2:], macBytes)
	return bridgeID
}

// registerDot1dStpScalars registers dot1dStp scalar OIDs.
func (a *Agent) registerDot1dStpScalars(stp *config.STPConfig, priority int, bridgeID []byte) {
	a.mib.Set(dot1dStpProtocolSpecification, &OIDValue{Type: gosnmp.Integer, Value: STPProtocolIEEE})
	a.mib.Set(dot1dStpPriority, &OIDValue{Type: gosnmp.Integer, Value: priority})
	a.mib.SetDynamic(dot1dStpTimeSinceTopologyChange, func() *OIDValue {
		return &OIDValue{Type: gosnmp.TimeTicks, Value: uptimeTicks(time.Since(a.startTime))}
	})
	a.mib.Set(dot1dStpTopChanges, &OIDValue{Type: gosnmp.Counter32, Value: uint32(0)})
	a.mib.Set(dot1dStpDesignatedRoot, &OIDValue{Type: gosnmp.OctetString, Value: bridgeID})
	a.mib.Set(dot1dStpRootCost, &OIDValue{Type: gosnmp.Integer, Value: 0})
	a.mib.Set(dot1dStpRootPort, &OIDValue{Type: gosnmp.Integer, Value: 0})

	// STP timers (in hundredths of seconds)
	maxAge, helloTime, forwardDelay := a.getStpTimers(stp)
	a.mib.Set(dot1dStpMaxAge, &OIDValue{Type: gosnmp.Integer, Value: maxAge * HundredthsPerSecond})
	a.mib.Set(dot1dStpHelloTime, &OIDValue{Type: gosnmp.Integer, Value: helloTime * HundredthsPerSecond})
	a.mib.Set(dot1dStpHoldTime, &OIDValue{Type: gosnmp.Integer, Value: 1})
	a.mib.Set(dot1dStpForwardDelay, &OIDValue{Type: gosnmp.Integer, Value: forwardDelay * HundredthsPerSecond})
	a.mib.Set(dot1dStpBridgeMaxAge, &OIDValue{Type: gosnmp.Integer, Value: maxAge * HundredthsPerSecond})
	a.mib.Set(dot1dStpBridgeHelloTime, &OIDValue{Type: gosnmp.Integer, Value: helloTime * HundredthsPerSecond})
	a.mib.Set(
		dot1dStpBridgeForwardDelay,
		&OIDValue{Type: gosnmp.Integer, Value: forwardDelay * HundredthsPerSecond},
	)
}

// getStpTimers returns STP timer values with defaults.
func (a *Agent) getStpTimers(stp *config.STPConfig) (int, int, int) {
	maxAge := int(stp.MaxAge)
	if maxAge == 0 {
		maxAge = 20
	}
	helloTime := int(stp.HelloTime)
	if helloTime == 0 {
		helloTime = 2
	}
	forwardDelay := int(stp.ForwardDelay)
	if forwardDelay == 0 {
		forwardDelay = 15
	}
	return maxAge, helloTime, forwardDelay
}

// registerDot1dStpPortEntry registers a single dot1dStpPortTable entry.
func (a *Agent) registerDot1dStpPortEntry(portIdx int, bridgeID []byte) {
	portStr := strconv.Itoa(portIdx)
	a.mib.Set(dot1dStpPort+"."+portStr, &OIDValue{Type: gosnmp.Integer, Value: portIdx})
	a.mib.Set(dot1dStpPortPriority+"."+portStr, &OIDValue{Type: gosnmp.Integer, Value: STPPortPriorityDefault})
	a.mib.Set(dot1dStpPortState+"."+portStr, &OIDValue{Type: gosnmp.Integer, Value: STPPortStateForwarding})
	a.mib.Set(dot1dStpPortEnable+"."+portStr, &OIDValue{Type: gosnmp.Integer, Value: 1})
	a.mib.Set(dot1dStpPortPathCost+"."+portStr, &OIDValue{Type: gosnmp.Integer, Value: STPPortPathCostDefault})
	a.mib.Set(
		dot1dStpPortDesignatedRoot+"."+portStr,
		&OIDValue{Type: gosnmp.OctetString, Value: bridgeID},
	)
	a.mib.Set(dot1dStpPortDesignatedCost+"."+portStr, &OIDValue{Type: gosnmp.Integer, Value: 0})
	a.mib.Set(
		dot1dStpPortDesignatedBridge+"."+portStr,
		&OIDValue{Type: gosnmp.OctetString, Value: bridgeID},
	)
	a.mib.Set(
		dot1dStpPortDesignatedPort+"."+portStr,
		&OIDValue{Type: gosnmp.OctetString, Value: []byte{0x80, byte(portIdx)}},
	)
	a.mib.Set(
		dot1dStpPortForwardTransitions+"."+portStr,
		&OIDValue{Type: gosnmp.Counter32, Value: uint32(1)},
	)
}

// registerDot1dTpGroup registers dot1dTp (transparent bridging) OIDs.
func (a *Agent) registerDot1dTpGroup(device *config.Device, numPorts int, macBytes []byte) {
	a.mib.Set(dot1dTpLearnedEntryDiscards, &OIDValue{Type: gosnmp.Counter32, Value: uint32(0)})
	a.mib.Set(dot1dTpAgingTime, &OIDValue{Type: gosnmp.Integer, Value: FDBAgingTimeDefault})

	// Register FDB self entry
	macIndex := macToOIDIndex(device.MACAddress.String())
	a.mib.Set(dot1dTpFdbAddress+"."+macIndex, &OIDValue{Type: gosnmp.OctetString, Value: macBytes})
	a.mib.Set(dot1dTpFdbPort+"."+macIndex, &OIDValue{Type: gosnmp.Integer, Value: 0})
	a.mib.Set(dot1dTpFdbStatus+"."+macIndex, &OIDValue{Type: gosnmp.Integer, Value: FDBStatusSelf})

	// Register port table entries
	for portIdx := 1; portIdx <= numPorts; portIdx++ {
		a.registerDot1dTpPortEntry(portIdx)
	}
}

// registerDot1dTpPortEntry registers a single dot1dTpPortTable entry.
func (a *Agent) registerDot1dTpPortEntry(portIdx int) {
	portStr := strconv.Itoa(portIdx)
	startTime := a.startTime

	a.mib.Set(dot1dTpPort+"."+portStr, &OIDValue{Type: gosnmp.Integer, Value: portIdx})
	a.mib.Set(dot1dTpPortMaxInfo+"."+portStr, &OIDValue{Type: gosnmp.Integer, Value: DefaultMTU})
	a.mib.SetDynamic(dot1dTpPortInFrames+"."+portStr, func() *OIDValue {
		elapsed := time.Since(startTime).Seconds()
		frames := uint32(elapsed * 100 * float64(portIdx%10+1))
		return &OIDValue{Type: gosnmp.Counter32, Value: frames}
	})
	a.mib.SetDynamic(dot1dTpPortOutFrames+"."+portStr, func() *OIDValue {
		elapsed := time.Since(startTime).Seconds()
		frames := uint32(elapsed * 80 * float64(portIdx%10+1))
		return &OIDValue{Type: gosnmp.Counter32, Value: frames}
	})
	a.mib.Set(
		dot1dTpPortInDiscards+"."+portStr,
		&OIDValue{Type: gosnmp.Counter32, Value: uint32(0)},
	)
}

// macToOIDIndex converts a MAC address to OID index format (decimal octets separated by dots).
func macToOIDIndex(mac string) string {
	macBytes := parseMACBytes(mac)

	parts := make([]string, MACAddressOctets)

	for i, b := range macBytes {
		parts[i] = strconv.FormatUint(uint64(b), 10)
	}

	return strings.Join(parts, ".")
}

// initializeLLDPLocalMIB populates LLDP local system data.
func (a *Agent) initializeLLDPLocalMIB() {
	logger := slog.Default()
	device := a.device

	lldp := device.LLDPConfig
	if lldp == nil {
		return
	}

	// lldpLocChassisIDSubtype (4 = macAddress)
	a.mib.Set(lldpLocChassisIDSubtype, &OIDValue{
		Type:  gosnmp.Integer,
		Value: ChassisIDSubtypeMAC,
	})

	// lldpLocChassisID (MAC address)
	macBytes := parseMACBytes(device.MACAddress.String())
	a.mib.Set(lldpLocChassisID, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: macBytes,
	})

	// lldpLocSysName - use device name since LLDP config doesn't have separate system name
	sysName := device.Name
	a.mib.Set(lldpLocSysName, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: sysName,
	})

	// lldpLocSysDesc
	sysDesc := lldp.SystemDescription
	if sysDesc == "" {
		sysDesc = fmt.Sprintf("%s %s", device.Type, device.Name)
	}

	a.mib.Set(lldpLocSysDesc, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: sysDesc,
	})

	// lldpLocSysCapSupported (bit field: bridge=2, router=4)
	capabilities := getCapabilitiesBitfield(device.Type)
	a.mib.Set(lldpLocSysCapSupported, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: []byte{byte(capabilities >> BitShiftByte), byte(capabilities)},
	})

	// lldpLocSysCapEnabled
	a.mib.Set(lldpLocSysCapEnabled, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: []byte{byte(capabilities >> BitShiftByte), byte(capabilities)},
	})

	// Local port entries
	for idx, trunk := range device.TrunkPorts {
		portNum := idx + 1
		a.createLLDPLocalPortEntry(portNum, trunk.Interface, lldp.PortDescription)
	}

	if a.debugLevel >= DebugLevelMinimum {
		logger.Info("Initialized LLDP local MIB", "device", device.Name)
	}
}

// createLLDPLocalPortEntry creates a local port entry in LLDP-MIB.
func (a *Agent) createLLDPLocalPortEntry(portNum int, ifName, portDesc string) {
	portNumStr := strconv.Itoa(portNum)
	baseOID := lldpLocPortTable + ".1"

	// lldpLocPortIdSubtype (5 = interfaceName)
	a.mib.Set(baseOID+".2."+portNumStr, &OIDValue{
		Type:  gosnmp.Integer,
		Value: PortIDSubtypeInterfaceName,
	})

	// lldpLocPortId
	a.mib.Set(baseOID+".3."+portNumStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: ifName,
	})

	// lldpLocPortDesc
	desc := portDesc
	if desc == "" {
		desc = ifName
	}

	a.mib.Set(baseOID+".4."+portNumStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: desc,
	})
}

// initializeLLDPRemoteMIB populates LLDP remote (neighbor) table.
func (a *Agent) initializeLLDPRemoteMIB() {
	logger := slog.Default()
	device := a.device
	if device == nil || len(device.TrunkPorts) == 0 {
		return
	}

	remIndex := 1

	for portIdx, trunk := range device.TrunkPorts {
		if trunk.RemoteDevice == "" {
			continue
		}

		portNum := portIdx + 1
		timeMark := 0 // lldpRemTimeMark

		// Create remote entry
		a.createLLDPRemoteEntry(timeMark, portNum, remIndex, trunk)

		remIndex++
	}

	if a.debugLevel >= DebugLevelMinimum {
		logger.Info("Initialized LLDP remote MIB", "neighbors", remIndex-1, "device", device.Name)
	}
}

// createLLDPRemoteEntry creates a remote (neighbor) entry in LLDP-MIB.
func (a *Agent) createLLDPRemoteEntry(timeMark, portNum, remIndex int, trunk config.TrunkPort) {
	// lldpRemEntry index: timeMark.portNum.remIndex
	indexStr := fmt.Sprintf("%d.%d.%d", timeMark, portNum, remIndex)
	entryBase := lldpRemTable + ".1"

	// lldpRemChassisIdSubtype (4 = macAddress)
	a.mib.Set(entryBase+".4."+indexStr, &OIDValue{
		Type:  gosnmp.Integer,
		Value: ChassisIDSubtypeMAC,
	})

	// lldpRemChassisId - Use remote device name as chassis ID for now
	// In a real implementation, we'd look up the remote device's MAC
	a.mib.Set(entryBase+".5."+indexStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: trunk.RemoteDevice,
	})

	// lldpRemPortIdSubtype (5 = interfaceName)
	a.mib.Set(entryBase+".6."+indexStr, &OIDValue{
		Type:  gosnmp.Integer,
		Value: PortIDSubtypeInterfaceName,
	})

	// lldpRemPortId
	a.mib.Set(entryBase+".7."+indexStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: trunk.RemoteInterface,
	})

	// lldpRemPortDesc
	a.mib.Set(entryBase+".8."+indexStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: trunk.RemoteInterface,
	})

	// lldpRemSysName
	a.mib.Set(entryBase+".9."+indexStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: trunk.RemoteDevice,
	})

	// lldpRemSysDesc (use device name as placeholder)
	a.mib.Set(entryBase+".10."+indexStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: "LLDP neighbor: " + trunk.RemoteDevice,
	})
}

// initializeCDPMIB populates Cisco CDP MIB.
func (a *Agent) initializeCDPMIB() {
	logger := slog.Default()
	device := a.device

	cdp := device.CDPConfig
	if cdp == nil {
		return
	}

	// cdpGlobalRun (TruthValue: 1=true)
	a.mib.Set(cdpGlobalRun, &OIDValue{
		Type:  gosnmp.Integer,
		Value: 1,
	})

	// cdpGlobalMessageInterval
	interval := cdp.AdvertiseInterval
	if interval == 0 {
		interval = 60
	}

	a.mib.Set(cdpGlobalMessageInterval, &OIDValue{
		Type:  gosnmp.Integer,
		Value: interval,
	})

	// cdpGlobalHoldTime
	holdtime := cdp.Holdtime
	if holdtime == 0 {
		holdtime = 180
	}

	a.mib.Set(cdpGlobalHoldTime, &OIDValue{
		Type:  gosnmp.Integer,
		Value: holdtime,
	})

	// cdpGlobalDeviceID
	a.mib.Set(cdpGlobalDeviceID, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: device.Name,
	})

	// CDP Cache entries (neighbors)
	deviceIndex := 1

	for ifIdx, trunk := range device.TrunkPorts {
		if trunk.RemoteDevice == "" {
			continue
		}

		ifIndex := ifIdx + 1
		a.createCDPCacheEntry(ifIndex, deviceIndex, trunk, cdp)

		deviceIndex++
	}

	if a.debugLevel >= DebugLevelMinimum {
		logger.Info("Initialized CDP MIB", "neighbors", deviceIndex-1, "device", device.Name)
	}
}

// createCDPCacheEntry creates a CDP cache (neighbor) entry.
func (a *Agent) createCDPCacheEntry(
	ifIndex, deviceIndex int,
	trunk config.TrunkPort,
	cdp *config.CDPConfig,
) {
	// cdpCacheEntry index: ifIndex.deviceIndex
	indexStr := fmt.Sprintf("%d.%d", ifIndex, deviceIndex)
	entryBase := cdpCacheTable + ".1"

	// cdpCacheAddressType (1 = IP)
	a.mib.Set(entryBase+".3."+indexStr, &OIDValue{
		Type:  gosnmp.Integer,
		Value: 1,
	})

	// cdpCacheAddress (placeholder - would be remote device IP)
	a.mib.Set(entryBase+".4."+indexStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: []byte{10, 0, 0, 1}, // Placeholder IP
	})

	// cdpCacheVersion
	version := cdp.SoftwareVersion
	if version == "" {
		version = unknownPlaceholder
	}

	a.mib.Set(entryBase+".5."+indexStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: version,
	})

	// cdpCacheDeviceId
	a.mib.Set(entryBase+".6."+indexStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: trunk.RemoteDevice,
	})

	// cdpCacheDevicePort
	a.mib.Set(entryBase+".7."+indexStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: trunk.RemoteInterface,
	})

	// cdpCachePlatform
	platform := cdp.Platform
	if platform == "" {
		platform = unknownPlaceholder
	}

	a.mib.Set(entryBase+".8."+indexStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: platform,
	})

	// cdpCacheCapabilities (bit field)
	a.mib.Set(entryBase+".9."+indexStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: []byte{0x00, 0x00, 0x00, 0x29}, // Router + Switch + IGMP
	})

	// cdpCacheVTPMgmtDomain
	a.mib.Set(entryBase+".10."+indexStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: "",
	})

	// cdpCacheNativeVLAN
	nativeVLAN := trunk.NativeVLAN
	if nativeVLAN == 0 {
		nativeVLAN = 1
	}

	a.mib.Set(entryBase+".11."+indexStr, &OIDValue{
		Type:  gosnmp.Integer,
		Value: nativeVLAN,
	})

	// cdpCacheDuplex (1=unknown, 2=half, 3=full)
	a.mib.Set(entryBase+".12."+indexStr, &OIDValue{
		Type:  gosnmp.Integer,
		Value: DuplexFull,
	})
}

// Helper functions

// parseMACBytes parses a MAC address string to bytes.
func parseMACBytes(mac string) []byte {
	// Remove colons, dashes, etc.
	mac = strings.ReplaceAll(mac, ":", "")
	mac = strings.ReplaceAll(mac, "-", "")
	mac = strings.ReplaceAll(mac, ".", "")

	bytes, err := hex.DecodeString(mac)
	if err != nil || len(bytes) != MACAddressOctets {
		return []byte{0, 0, 0, 0, 0, 0}
	}

	return bytes
}

// getInterfaceSpeed returns interface speed based on interface name.
func getInterfaceSpeed(ifName string) uint64 {
	ifNameLower := strings.ToLower(ifName)

	switch {
	case strings.Contains(ifNameLower, "hundredgig") || strings.Contains(ifNameLower, "100g"):
		return Speed100Gbps // 100 Gbps
	case strings.Contains(ifNameLower, "fortygig") || strings.Contains(ifNameLower, "40g"):
		return Speed40Gbps // 40 Gbps
	case strings.Contains(ifNameLower, "twentyfivegig") || strings.Contains(ifNameLower, "25g"):
		return Speed25Gbps // 25 Gbps
	case strings.Contains(ifNameLower, "tengig") || strings.Contains(ifNameLower, "10g"):
		return Speed10Gbps // 10 Gbps
	case strings.Contains(ifNameLower, "fivegig") || strings.Contains(ifNameLower, "5g"):
		return Speed5Gbps // 5 Gbps
	case strings.Contains(ifNameLower, "twogig") || strings.Contains(ifNameLower, "2.5g"):
		return Speed2p5Gbps // 2.5 Gbps
	case strings.Contains(ifNameLower, "gigabit") || strings.Contains(ifNameLower, "ge") || strings.Contains(ifNameLower, "1g"):
		return NanosPerSecond // 1 Gbps
	case strings.Contains(ifNameLower, "fastethernet") || strings.Contains(ifNameLower, "fa"):
		return Speed100Mbps // 100 Mbps
	case strings.Contains(ifNameLower, "ethernet"):
		return NanosPerSecond // Default to 1 Gbps
	default:
		return NanosPerSecond // Default to 1 Gbps
	}
}

// getCapabilitiesBitfield returns LLDP/CDP capability bits based on device type.
func getCapabilitiesBitfield(deviceType string) int {
	// LLDP System Capabilities bitmap:
	// Bit 0: Other
	// Bit 1: Repeater
	// Bit 2: Bridge
	// Bit 3: WLAN Access Point
	// Bit 4: Router
	// Bit 5: Telephone
	// Bit 6: DOCSIS cable device
	// Bit 7: Station Only
	deviceTypeLower := strings.ToLower(deviceType)

	switch deviceTypeLower {
	case "router":
		return CapabilityRouterBridge // Router + Bridge
	case "switch":
		return CapabilityBridge // Bridge
	case "ap", "access-point":
		return CapabilityWLANAP // WLAN AP
	case "server", "host":
		return CapabilityStationOnly // Station Only
	default:
		return LLDPCapabilityOther
	}
}
