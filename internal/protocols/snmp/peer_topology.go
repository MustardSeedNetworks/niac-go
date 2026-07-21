package snmp

import (
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"
)

// PeerMACResolver returns the MAC bytes for a device name, and whether it is
// known. The stack builds it from the full config roster — an agent alone only
// knows its own device.
type PeerMACResolver func(deviceName string) ([]byte, bool)

// hasWalkContent reports whether this device is backed by a capture walk, which
// then owns the IF-MIB / BRIDGE-MIB content that construction would otherwise
// synthesize from trunk_ports.
func (a *Agent) hasWalkContent() bool {
	if a.device == nil {
		return false
	}

	return a.device.SNMPConfig.WalkFile != "" || len(a.device.SNMPConfig.WalkFiles) > 0
}

// SynthesizePeerTopology fills in the bridge forwarding entries that let a
// discovery tool answer "which switch/port is this host on" (NetAlly's Nearest
// Switch). Construction seeds only a self FDB entry because a single agent does
// not know its neighbours' MACs; here — after the fleet is loaded — each
// trunk_port whose remote device resolves to a MAC becomes a learned FDB entry
// on the port's bridge index, mapped through the walk's real ifIndex.
//
// Ports whose interface name is absent from the device's own walk (e.g. an
// inter-switch uplink named for a chassis the walk doesn't model) are skipped:
// those links already show via the CDP/LLDP neighbour tables; the FDB is for
// attached hosts. Call after all MIB content is loaded, then Reindex.
func (a *Agent) SynthesizePeerTopology(resolve PeerMACResolver) {
	if a == nil || a.device == nil || resolve == nil {
		return
	}

	changed := false
	maxPort := 0

	// Infer the capture's bridgePort→ifIndex offset once, up front, from its own
	// table. Sampling per-port would drift as we add rows below.
	offset, hasOffset := a.basePortOffset()

	for _, trunk := range a.device.TrunkPorts {
		if trunk.RemoteDevice == "" || trunk.Interface == "" {
			continue
		}

		mac, ok := resolve(trunk.RemoteDevice)
		if !ok || len(mac) == 0 {
			continue
		}

		bridgePort, ok := a.bridgePortForInterface(trunk.Interface, offset, hasOffset)
		if !ok {
			continue
		}

		a.addLearnedFDBEntry(mac, bridgePort)

		maxPort = max(maxPort, bridgePort)

		changed = true
	}

	if changed {
		// A learned FDB port must be a valid bridge port: managers reject entries
		// whose port exceeds dot1dBaseNumPorts. Captures model only a few ports,
		// so raise the count to cover the ports we just wired.
		a.raiseBaseNumPorts(maxPort)
		a.mib.Reindex()
	}
}

// bridgePortForInterface resolves an interface name (e.g. "FastEthernet0/5") to
// its bridge port number, preferring the walk's own dot1dBasePortTable so the
// port lines up with the real ifIndex/ifName a manager renders. When the capture
// doesn't model that port, it derives the physical port number from the name and
// adds the dot1dBasePort row — a small, in-range port, not the ifIndex (which a
// manager would reject as out of range).
func (a *Agent) bridgePortForInterface(name string, offset int, hasOffset bool) (int, bool) {
	ifIndex, ok := a.ifIndexForInterface(name)
	if !ok {
		return 0, false
	}

	// Honour the capture's numbering where the port is modelled.
	if port, found := a.mib.IndexSuffixForValue(dot1dBasePortIfIndex, ifIndex); found {
		if n, convErr := strconv.Atoi(port); convErr == nil {
			return n, true
		}
	}

	ifIndexNum, err := strconv.Atoi(ifIndex)
	if err != nil {
		return 0, false
	}

	// The capture models only a few bridge ports, so derive one. dot1dBasePortIfIndex
	// is linear (ifIndex = bridgePort + offset), so apply the offset inferred from
	// the capture. This keeps distinct interface types distinct — Fa0/1 and Gi0/1
	// have different ifIndex, so different bridge ports — unlike naming the port by
	// the trailing number, which would collide them. Add the row so
	// FDB port -> dot1dBasePortIfIndex -> ifIndex -> ifName resolves.
	port := ifIndexNum
	if hasOffset && ifIndexNum-offset > 0 {
		port = ifIndexNum - offset
	}

	portStr := strconv.Itoa(port)
	a.mib.Set(dot1dBasePort+"."+portStr, &OIDValue{Type: gosnmp.Integer, Value: port})
	a.mib.Set(dot1dBasePortIfIndex+"."+portStr, &OIDValue{Type: gosnmp.Integer, Value: ifIndexNum})

	return port, true
}

// ifIndexForInterface resolves an authored interface name through the loaded
// IF-MIB, preferring ifDescr and then ifName as discovery managers do.
func (a *Agent) ifIndexForInterface(name string) (string, bool) {
	ifIndex, ok := a.mib.IndexSuffixForValue(ifDescr, name)
	if !ok {
		ifIndex, ok = a.mib.IndexSuffixForValue(ifName, name)
	}
	return ifIndex, ok
}

// basePortOffset infers the capture's linear bridgePort→ifIndex offset (ifIndex =
// bridgePort + offset) from a sampled dot1dBasePortIfIndex row.
func (a *Agent) basePortOffset() (int, bool) {
	prefix := dot1dBasePortIfIndex + "."
	counts := make(map[int]int)
	a.mib.mu.RLock()
	defer a.mib.mu.RUnlock()
	for oid, value := range a.mib.entries {
		if !strings.HasPrefix(oid, prefix) {
			continue
		}
		port, portErr := strconv.Atoi(strings.TrimPrefix(oid, prefix))
		ifIndex, indexErr := strconv.Atoi(oidValueString(value))
		if portErr == nil && indexErr == nil {
			counts[ifIndex-port]++
		}
	}
	bestOffset, bestCount := 0, 0
	for offset, count := range counts {
		if count > bestCount || (count == bestCount && absInt(offset) > absInt(bestOffset)) {
			bestOffset, bestCount = offset, count
		}
	}
	return bestOffset, bestCount > 0
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

// raiseBaseNumPorts ensures dot1dBaseNumPorts is at least n.
func (a *Agent) raiseBaseNumPorts(n int) {
	cur := 0
	if v := a.mib.Get(dot1dBaseNumPorts); v != nil {
		if iv, ok := v.Value.(int); ok {
			cur = iv
		}
	}

	if n > cur {
		a.mib.Set(dot1dBaseNumPorts, &OIDValue{Type: gosnmp.Integer, Value: n})
	}
}

// addLearnedFDBEntry registers a dot1dTpFdbTable learned entry: MAC seen on the
// given bridge port.
func (a *Agent) addLearnedFDBEntry(mac []byte, bridgePort int) {
	idx := macBytesToOIDIndex(mac)

	a.mib.Set(dot1dTpFdbAddress+"."+idx, &OIDValue{Type: gosnmp.OctetString, Value: mac})
	a.mib.Set(dot1dTpFdbPort+"."+idx, &OIDValue{Type: gosnmp.Integer, Value: bridgePort})
	a.mib.Set(dot1dTpFdbStatus+"."+idx, &OIDValue{Type: gosnmp.Integer, Value: FDBStatusLearned})
}

// macBytesToOIDIndex renders MAC bytes as the dotted-decimal OID index the
// dot1dTpFdbTable is keyed by.
func macBytesToOIDIndex(mac []byte) string {
	parts := make([]string, len(mac))
	for i, b := range mac {
		parts[i] = strconv.Itoa(int(b))
	}

	return strings.Join(parts, ".")
}
