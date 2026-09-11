package snmp

import "strings"

// Bucket names why an OID the agent serves is not byte-identical to the walk
// it was loaded from. It is the shared vocabulary of the replay-fidelity
// contract (docs/design/2026-09-replay-fidelity-contract.md): LoadWalkFile
// decides what not to load with it, and the fidelity harness accounts for
// every difference it measures with the same names. An OID that classifies as
// BucketKept must reach the wire exactly as the source carried it.
type Bucket string

const (
	// BucketKept is the contract's default and its whole point: the walk's
	// OID, type and value are served unchanged.
	BucketKept Bucket = "kept"
	// BucketAuthored is replaced by what the scenario authored — the device's
	// identity and its interface rows.
	BucketAuthored Bucket = "authored"
	// BucketLive is served by the running agent's own counters instead of the
	// capture's frozen ones.
	BucketLive Bucket = "live"
	// BucketTopology is dropped in favour of neighbours synthesized from
	// trunk_ports, because a capture's neighbours are foreign to this lab.
	BucketTopology Bucket = "topology"
	// BucketNormalized is a source line that named its OID symbolically
	// (IF-MIB::ifDescr.1) and was rewritten to numeric form on parse. It is
	// decided from the source text by NormalizedWalkOIDBase, not from the
	// numeric OID, so Classify never returns it.
	BucketNormalized Bucket = "normalized"
	// BucketAgentAdded is on the wire but not in the source: MIB-II the agent
	// synthesizes for every device. Only the harness, which sees both sides,
	// can decide it, so Classify never returns it.
	BucketAgentAdded Bucket = "agent_added"
)

// interfaceColumnBucket reports which substitution covers an ifTable or
// ifXTable column on a row the scenario authored. The configuration columns
// are the ones refreshAuthoredInterfaceMIBs overwrites from the scenario; the
// counters are the ones registerIfTableCounters and
// registerIfXTablePacketCounters make dynamic, so they report this run's
// traffic rather than the capture's.
func interfaceColumnBucket(column string) (Bucket, bool) {
	switch column {
	case ifSpeed, ifHighSpeed, ifMtu, ifType, ifConnectorPresent,
		ifAdminStatus, ifOperStatus, ifAlias, dot3StatsDuplexStatus:
		return BucketAuthored, true
	case ifInOctets, ifInUcastPkts, ifInNUcastPkts, ifInDiscards, ifInErrors,
		ifOutOctets, ifOutUcastPkts, ifOutNUcastPkts, ifOutDiscards, ifOutErrors,
		dot3StatsFCSErrors,
		ifInMulticastPkts, ifInBroadcastPkts, ifOutMulticastPkts, ifOutBroadcastPkts,
		ifHCInOctets, ifHCInUcastPkts, ifHCInMulticastPkts, ifHCInBroadcastPkts,
		ifHCOutOctets, ifHCOutUcastPkts, ifHCOutMulticastPkts, ifHCOutBroadcastPkts:
		return BucketLive, true
	}
	return BucketKept, false
}

// bridgePortColumnBucket reports which substitution covers a dot1dTpPortTable
// column on a bridge port the contract substitutes. registerDot1dTpPortEntry
// restates the port index and the device's MTU from the scenario, and makes
// the frame counters and the discard count report this run rather than the
// capture's frozen totals.
func bridgePortColumnBucket(column string) Bucket {
	switch column {
	case dot1dTpPortInFrames, dot1dTpPortOutFrames, dot1dTpPortInDiscards:
		return BucketLive
	default:
		return BucketAuthored
	}
}

// isAuthoredAddressOID reports whether oid holds a link-layer address that
// refreshAuthoredPhysicalIdentity replaces with the scenario's own. It is
// prefix-level: a row carrying no address is left alone by the refresh but
// classified here anyway, the same imprecision substitution 5 accepts for an
// authored interface row.
//
// The chassis-ID subtype is here because refreshLLDPChassisIdentity writes it
// with the value it describes: a MAC served under subtype 7 (locallyAssigned)
// would misdescribe itself. 28 of the corpus's Cisco captures declare 7, and
// without this every one of them reports an unclassified row.
func isAuthoredAddressOID(oid string) bool {
	return strings.HasPrefix(oid, ifPhysAddress+".") ||
		oid == dot1dBaseBridgeAddress ||
		oid == lldpLocChassisID ||
		oid == lldpLocChassisIDSubtype
}

// isLiveProtocolOID reports whether oid falls in a MIB-II protocol group the
// agent always answers from its own live counters. A capture's totals would be
// frozen, and would describe a different machine's traffic. The snmp group is
// here too: the agent owns its own request statistics.
func isLiveProtocolOID(oid string) bool {
	return underRoot(oid, ipMIBBase) ||
		underRoot(oid, icmpMIBRoot) ||
		underRoot(oid, tcpMIBRoot) ||
		underRoot(oid, udpMIBRoot) ||
		underRoot(oid, egpMIBRoot) ||
		underRoot(oid, snmpGroup)
}

// isSynthesizedTopologyOID reports whether oid falls in a neighbour or
// forwarding table that trunk_ports synthesis owns.
func isSynthesizedTopologyOID(oid string) bool {
	return underRoot(oid, lldpRemoteSystemsData) || // 1.0.8802.1.1.2.1.4 — LLDP-MIB neighbours
		underRoot(oid, cdpCache) || // 1.3.6.1.4.1.9.9.23.1.2 — CDP cache neighbours
		underRoot(oid, lldpLocPortTable) || // 1.0.8802.1.1.2.1.3.7 — LLDP local ports, rebuilt with them
		underRoot(oid, dot1dTpFdbTable) || // 1.3.6.1.2.1.17.4.3 — bridge MAC→port
		underRoot(oid, dot1qFDBTable) || // 1.3.6.1.2.1.17.7.1.2.1 — VLAN FDB counters
		underRoot(oid, dot1qTpFDBTable) // 1.3.6.1.2.1.17.7.1.2.2 — VLAN MAC→port
}

const (
	authoredSysDescrOID    = "1.3.6.1.2.1.1.1.0"
	authoredSysContactOID  = "1.3.6.1.2.1.1.4.0"
	sysNameOID             = "1.3.6.1.2.1.1.5.0"
	authoredSysLocationOID = "1.3.6.1.2.1.1.6.0"
)

// WalkContract answers, for one device, which of the signed substitutions
// applies to a walk OID. Build it with Agent.WalkContract so the loader and
// the fidelity harness cannot drift apart; the five substitutions it encodes
// were signed off by the owner on 2026-09-05 and are documented in
// docs/design/2026-09-replay-fidelity-contract.md.
type WalkContract struct {
	authoredSysDescr    bool
	authoredSysContact  bool
	authoredSysLocation bool
	// ownsTopology is trunk_ports: without it the walk's own neighbours stand.
	ownsTopology bool
	// Signed substitution 6 has two conditions, because the code does: any
	// authored MAC seeds the derived serial numbers, but only an Ethernet one
	// can stand in for a link-layer address, so a device carrying a longer
	// MAC keeps the capture's addresses.
	authoredMAC         bool
	authoredEthernetMAC bool
	// authoredIfIndexes are the ifIndexes the MIB currently gives the
	// interfaces the scenario authored, resolved through ifDescr/ifName. A
	// walk can renumber them, so a contract used to judge what reached the
	// wire must be built after the load, not before.
	authoredIfIndexes map[string]struct{}
	// authoredBridgePorts are the dot1dBasePort numbers sitting in front of
	// those ifIndexes, read from the walk's own dot1dBasePortIfIndex rows. A
	// bridge port index is not an ifIndex, so this needs its own lookup.
	authoredBridgePorts    map[string]struct{}
	activeResourceOIDs     map[string]struct{}
	changedInterfaceOIDs   map[string]struct{}
	activeDeviceActionOIDs map[string]struct{}
}

// WalkContract builds the contract for this agent's device, reading the
// interface indexes out of the MIB as it stands. LoadWalkFile calls it for the
// skip decision, which does not depend on those indexes; the fidelity harness
// calls it again after the load, when the walk's own indexes are in place.
func (a *Agent) WalkContract() WalkContract {
	contract := WalkContract{
		authoredIfIndexes:   map[string]struct{}{},
		authoredBridgePorts: map[string]struct{}{},
	}
	if a.device == nil {
		return contract
	}
	contract.authoredSysDescr = a.device.SNMPConfig.SysDescr != ""
	contract.authoredSysContact = a.device.SNMPConfig.SysContact != ""
	contract.authoredSysLocation = a.device.SNMPConfig.SysLocation != ""
	contract.ownsTopology = len(a.device.TrunkPorts) > 0
	contract.authoredMAC = len(a.device.MACAddress) > 0
	contract.authoredEthernetMAC = len(a.device.MACAddress) == MACAddressOctets
	for _, iface := range a.device.Interfaces {
		if index, ok := a.ifIndexForInterface(iface.Name); ok {
			contract.authoredIfIndexes[index] = struct{}{}
		}
	}
	a.resolveAuthoredBridgePorts(&contract)
	contract.activeResourceOIDs = a.activeResourceOIDs()
	contract.changedInterfaceOIDs = a.changedInterfaceOIDs()
	contract.activeDeviceActionOIDs = a.activeDeviceActionOIDs()
	return contract
}

// resolveAuthoredBridgePorts maps the authored ifIndexes onto bridge port
// numbers through dot1dBasePortIfIndex. Those rows are the walk's, so this is
// only meaningful on a contract built after the load.
func (a *Agent) resolveAuthoredBridgePorts(contract *WalkContract) {
	if len(contract.authoredIfIndexes) == 0 {
		return
	}
	for _, oid := range a.mib.snapshotOIDs() {
		port, isBasePort := strings.CutPrefix(oid, dot1dBasePortIfIndex+".")
		if !isBasePort {
			continue
		}
		if _, authored := contract.authoredIfIndexes[oidValueString(a.mib.Get(oid))]; authored {
			contract.authoredBridgePorts[port] = struct{}{}
		}
	}
}

// Classify reports which substitution covers oid, or BucketKept when the walk's
// value must be served unchanged. It never returns BucketNormalized or
// BucketAgentAdded: neither is decidable from a numeric OID alone.
func (c WalkContract) Classify(oid string) Bucket {
	oid = strings.TrimPrefix(oid, ".")
	if _, active := c.activeDeviceActionOIDs[oid]; active {
		return BucketLive
	}
	if _, armed := c.activeResourceOIDs[oid]; armed {
		return BucketLive
	}
	if _, changed := c.changedInterfaceOIDs[oid]; changed {
		return BucketLive
	}
	switch {
	case c.isAuthoredIdentity(oid):
		return BucketAuthored
	case isLiveProtocolOID(oid):
		return BucketLive
	case c.ownsTopology && isSynthesizedTopologyOID(oid):
		return BucketTopology
	case c.authoredEthernetMAC && isAuthoredAddressOID(oid):
		return BucketAuthored
	case c.authoredMAC && strings.HasPrefix(oid, entPhysicalSerialNumber+"."):
		return BucketAuthored
	}
	column, index, split := splitInterfaceColumn(oid)
	if !split {
		return BucketKept
	}
	if underRoot(column, dot1dTpPortEntry) {
		if c.substitutesBridgePort(index) {
			return bridgePortColumnBucket(column)
		}
		return BucketKept
	}
	if _, authored := c.authoredIfIndexes[index]; !authored {
		return BucketKept
	}
	if bucket, covered := interfaceColumnBucket(column); covered {
		return bucket
	}
	return BucketKept
}

// substitutesBridgePort reports whether the dot1dTpPortTable row for this
// bridge port is the agent's to write. trunk_ports hands it the whole
// forwarding topology; otherwise only a port an authored interface sits
// behind is in scope, and every other port arrives byte-identical.
func (c WalkContract) substitutesBridgePort(port string) bool {
	if c.ownsTopology {
		return true
	}
	_, authored := c.authoredBridgePorts[port]
	return authored
}

// DropsFromWalk reports whether LoadWalkFile must not load oid at all, as
// opposed to loading it and overwriting it afterwards. Only the substitutions
// that do not need the walk's own IF-MIB indexes can be decided this early —
// the authored interface rows are overwritten after the load instead.
func (c WalkContract) DropsFromWalk(oid string) bool {
	switch c.Classify(oid) {
	case BucketAuthored:
		return c.isAuthoredIdentity(strings.TrimPrefix(oid, "."))
	case BucketLive:
		return isLiveProtocolOID(strings.TrimPrefix(oid, "."))
	case BucketTopology:
		return true
	case BucketKept, BucketNormalized, BucketAgentAdded:
		return false
	}
	return false
}

// isAuthoredIdentity takes an OID with any leading dot already trimmed.
func (c WalkContract) isAuthoredIdentity(oid string) bool {
	return oid == sysNameOID ||
		(oid == authoredSysDescrOID && c.authoredSysDescr) ||
		(oid == authoredSysContactOID && c.authoredSysContact) ||
		(oid == authoredSysLocationOID && c.authoredSysLocation)
}

// splitInterfaceColumn splits a table cell into its column OID and row index,
// for the single-integer indexes ifTable and ifXTable use.
func splitInterfaceColumn(oid string) (string, string, bool) {
	cut := strings.LastIndex(oid, ".")
	if cut <= 0 || cut == len(oid)-1 {
		return "", "", false
	}
	return oid[:cut], oid[cut+1:], true
}

func underRoot(oid, root string) bool {
	return oid == root || strings.HasPrefix(oid, root+".")
}
