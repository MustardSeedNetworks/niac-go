package snmp

import (
	"net"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// PeerIdentity contains the fleet-wide identity needed to synthesize discovery
// and forwarding topology for a remote device.
type PeerIdentity struct {
	MAC               []byte
	Address           net.IP
	Type              string
	SystemDescription string
	CDPEnabled        bool
	CDPPlatform       string
	CDPVersion        string
}

// PeerResolver returns the identity for a remote device and interface. The
// stack builds it from the full config roster — an agent alone only knows its
// own device.
type PeerResolver func(deviceName, interfaceName string) (PeerIdentity, bool)

// hasWalkContent reports whether this device is backed by a capture walk, which
// then owns the IF-MIB / BRIDGE-MIB content that construction would otherwise
// synthesize from trunk_ports.
func (a *Agent) hasWalkContent() bool {
	if a.device == nil {
		return false
	}

	return a.device.SNMPConfig.WalkFile != "" || len(a.device.SNMPConfig.WalkFiles) > 0
}

// SynthesizePeerTopology fills the remote CDP address and bridge forwarding
// entries that a discovery tool uses to identify and place connected devices.
// A single agent does not know its neighbours' addresses or MACs; here — after
// the fleet is loaded — each trunk_port resolves through the complete roster.
//
// Ports whose interface name is absent from the device's own walk (e.g. an
// inter-switch uplink named for a chassis the walk doesn't model) are skipped:
// those links already show via the CDP/LLDP neighbour tables; the FDB is for
// attached hosts. Call after all MIB content is loaded, then Reindex.
func (a *Agent) SynthesizePeerTopology(resolve PeerResolver) {
	if a == nil || a.device == nil || resolve == nil {
		return
	}

	changed := false
	maxPort := 0

	// Infer the capture's bridgePort→ifIndex offset once, up front, from its own
	// table. Sampling per-port would drift as we add rows below.
	offset, hasOffset := a.basePortOffset()

	remoteIndex := 0
	for portIndex, trunk := range a.device.TrunkPorts {
		if trunk.RemoteDevice == "" {
			continue
		}

		if !trunk.FDBOnly {
			remoteIndex++
		}
		peer, ok := resolve(trunk.RemoteDevice, trunk.RemoteInterface)
		if !ok {
			continue
		}

		changed = a.setPeerDiscoveryIdentity(
			trunk,
			peer,
			portIndex+1,
			remoteIndex,
		) || changed

		if !a.supportsBridgeLink(trunk) {
			continue
		}
		if trunk.Interface == "" || len(peer.MAC) == 0 {
			continue
		}
		bridgePort, ok := a.bridgePortForInterface(trunk.Interface, offset, hasOffset)
		if !ok {
			continue
		}

		a.addLearnedFDBEntry(peer.MAC, bridgePort)
		if vlan := forwardingVLAN(trunk); vlan > 0 {
			a.addLearnedQBridgeFDBEntry(vlan, peer.MAC, bridgePort)
		}

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

func (a *Agent) supportsSynthesizedBridgeMIB() bool {
	return a != nil && a.device != nil && systemProfile(a.device).IncludeBridge
}

func (a *Agent) supportsBridgeTopology() bool {
	return a.supportsSynthesizedBridgeMIB() ||
		(a.hasWalkContent() && a.mib.Get(dot1dBaseNumPorts) != nil)
}

func (a *Agent) setPeerDiscoveryIdentity(
	trunk config.TrunkPort,
	peer PeerIdentity,
	fallbackIfIndex, remoteIndex int,
) bool {
	if trunk.FDBOnly {
		return false
	}
	ifIndex := a.discoveryIfIndex(trunk.Interface, fallbackIfIndex)
	cdpChanged := a.setCDPPeerIdentity(trunk, peer, ifIndex, remoteIndex)
	lldpChanged := a.setLLDPPeerIdentity(trunk, peer, ifIndex, remoteIndex)
	return cdpChanged || lldpChanged
}

func (a *Agent) setCDPPeerIdentity(
	trunk config.TrunkPort,
	peer PeerIdentity,
	ifIndex, deviceIndex int,
) bool {
	if trunk.FDBOnly || a.device.CDPConfig == nil || !a.device.CDPConfig.Enabled {
		return false
	}

	row := strconv.Itoa(ifIndex) + "." + strconv.Itoa(deviceIndex)
	entry := cdpCacheTable + ".1"
	if !peer.CDPEnabled {
		changed := false
		for column := 3; column <= 12; column++ {
			oid := entry + "." + strconv.Itoa(column) + "." + row
			if a.mib.Get(oid) != nil {
				a.mib.Delete(oid)
				changed = true
			}
		}
		return changed
	}

	a.mib.Set(entry+".3."+row, &OIDValue{Type: gosnmp.Integer, Value: 1})
	if ipv4 := peer.Address.To4(); ipv4 != nil {
		a.mib.Set(entry+".4."+row, &OIDValue{Type: gosnmp.OctetString, Value: []byte(ipv4)})
	}
	version := peer.CDPVersion
	if version == "" {
		version = unknownPlaceholder
	}
	a.mib.Set(entry+".5."+row, &OIDValue{Type: gosnmp.OctetString, Value: version})
	a.mib.Set(entry+".6."+row, &OIDValue{Type: gosnmp.OctetString, Value: trunk.RemoteDevice})
	a.mib.Set(entry+".7."+row, &OIDValue{Type: gosnmp.OctetString, Value: trunk.RemoteInterface})
	platform := peer.CDPPlatform
	if platform == "" {
		platform = unknownPlaceholder
	}
	a.mib.Set(entry+".8."+row, &OIDValue{Type: gosnmp.OctetString, Value: platform})
	a.mib.Set(entry+".9."+row, &OIDValue{
		Type: gosnmp.OctetString, Value: cdpCapabilities(peer.Type),
	})
	return true
}

func (a *Agent) setLLDPPeerIdentity(
	trunk config.TrunkPort,
	peer PeerIdentity,
	ifIndex, remoteIndex int,
) bool {
	if trunk.FDBOnly || a.device.LLDPConfig == nil || !a.device.LLDPConfig.Enabled ||
		len(peer.MAC) == 0 {
		return false
	}

	row := "0." + strconv.Itoa(ifIndex) + "." + strconv.Itoa(remoteIndex)
	entry := lldpRemTable + ".1"
	a.mib.Set(entry+".4."+row, &OIDValue{Type: gosnmp.Integer, Value: ChassisIDSubtypeMAC})
	a.mib.Set(entry+".5."+row, &OIDValue{Type: gosnmp.OctetString, Value: peer.MAC})
	if peer.SystemDescription != "" {
		a.mib.Set(entry+".10."+row, &OIDValue{
			Type: gosnmp.OctetString, Value: peer.SystemDescription,
		})
	}
	capabilities := safeconv.Uint32(getCapabilitiesBitfield(peer.Type))
	bits := []byte{
		safeconv.ByteFromUint32(capabilities >> BitShiftByte),
		safeconv.ByteFromUint32(capabilities),
	}
	a.mib.Set(entry+".11."+row, &OIDValue{Type: gosnmp.OctetString, Value: bits})
	a.mib.Set(entry+".12."+row, &OIDValue{Type: gosnmp.OctetString, Value: bits})
	return true
}

func forwardingVLAN(trunk config.TrunkPort) int {
	if trunk.NativeVLAN > 0 {
		return trunk.NativeVLAN
	}
	return 1
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
