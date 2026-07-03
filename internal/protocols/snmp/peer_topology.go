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

	for _, trunk := range a.device.TrunkPorts {
		if trunk.RemoteDevice == "" || trunk.Interface == "" {
			continue
		}

		mac, ok := resolve(trunk.RemoteDevice)
		if !ok || len(mac) == 0 {
			continue
		}

		bridgePort, ok := a.bridgePortForInterface(trunk.Interface)
		if !ok {
			continue
		}

		a.addLearnedFDBEntry(mac, bridgePort)
		changed = true
	}

	if changed {
		a.mib.Reindex()
	}
}

// bridgePortForInterface resolves an interface name (e.g. "FastEthernet0/5") to
// its bridge port number via the walk's ifTable and dot1dBasePortTable, so the
// FDB port lines up with the real ifIndex/ifName a manager will render.
func (a *Agent) bridgePortForInterface(name string) (int, bool) {
	ifIndex, ok := a.mib.IndexSuffixForValue(ifDescr, name)
	if !ok {
		ifIndex, ok = a.mib.IndexSuffixForValue(ifName, name)
	}

	if !ok {
		return 0, false
	}

	ifIndexNum, err := strconv.Atoi(ifIndex)
	if err != nil {
		return 0, false
	}

	// dot1dBasePortIfIndex maps bridge port -> ifIndex; invert it. Honour the
	// capture's numbering where present so the port resolves to the real
	// physical interface.
	if port, found := a.mib.IndexSuffixForValue(dot1dBasePortIfIndex, ifIndex); found {
		if n, convErr := strconv.Atoi(port); convErr == nil {
			return n, true
		}
	}

	// Captures often model only a few active bridge ports, so the interface may
	// have no dot1dBasePort row. Add one keyed by the ifIndex (unique) mapping
	// straight back to it, so FDB port -> dot1dBasePortIfIndex -> ifIndex ->
	// ifName still resolves to the right interface.
	a.mib.Set(dot1dBasePort+"."+ifIndex, &OIDValue{Type: gosnmp.Integer, Value: ifIndexNum})
	a.mib.Set(dot1dBasePortIfIndex+"."+ifIndex, &OIDValue{Type: gosnmp.Integer, Value: ifIndexNum})

	return ifIndexNum, true
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
