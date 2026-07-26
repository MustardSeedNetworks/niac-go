package snmp

import (
	"slices"
	"strings"
)

// LLDP-MIB OID prefixes (IEEE 802.1AB)

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
	ipRouteDest    = ipRouteEntry + ".1" // Destination
	ipRouteIfIndex = ipRouteEntry + ".2" // Interface index
	ipRouteMetric1 = ipRouteEntry + ".3" // Metric
	ipRouteMetric2 = ipRouteEntry + ".4"
	ipRouteMetric3 = ipRouteEntry + ".5"
	ipRouteMetric4 = ipRouteEntry + ".6"
	ipRouteNextHop = ipRouteEntry + ".7" // Next hop
	ipRouteType    = ipRouteEntry + ".8" // Route type (1=other,2=invalid,3=direct,4=indirect)
	ipRouteProto   = ipRouteEntry + ".9" // Protocol (1=other,2=local,3=netmgmt,4=icmp,etc.)
	ipRouteAge     = ipRouteEntry + ".10"
	ipRouteMask    = ipRouteEntry + ".11" // Subnet mask
	ipRouteInfo    = ipRouteEntry + ".12"
	ipRouteMetric5 = ipRouteEntry + ".13"

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

func (a *Agent) initializeNeighborMIBs() {
	device := a.device
	if device == nil {
		return
	}

	// Initialize IF-MIB (interface table)
	a.initializeIFMIB()

	// Initialize IP-MIB (IP address table, routing table)
	a.initializeIPMIB()

	// Discovery managers use bridge tables as device-role evidence. Only bridge
	// profiles synthesize them; authoritative walk data remains untouched.
	if a.supportsSynthesizedBridgeMIB() {
		a.initializeBridgeMIB()
		a.initializeQBridgeMIB()
	}

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

// refreshAuthoredDiscoveryMIBs rebuilds local and remote discovery rows after
// a walk has supplied its authoritative IF-MIB indexes.
func (a *Agent) refreshAuthoredDiscoveryMIBs() {
	prefixes := []string{lldpLocPortTable, lldpRemoteSystemsData, cdpCache}
	for _, oid := range a.mib.AllOIDs() {
		if slices.ContainsFunc(prefixes, func(prefix string) bool {
			return oid == prefix || strings.HasPrefix(oid, prefix+".")
		}) {
			a.mib.Delete(oid)
		}
	}
	if a.device.LLDPConfig != nil && a.device.LLDPConfig.Enabled {
		a.initializeLLDPLocalMIB()
		a.initializeLLDPRemoteMIB()
	}
	if a.device.CDPConfig != nil && a.device.CDPConfig.Enabled {
		a.initializeCDPMIB()
	}
	if a.supportsBridgeTopology() {
		a.initializeQBridgePVIDs()
	}
}
