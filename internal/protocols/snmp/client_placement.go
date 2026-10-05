package snmp

// PlaceLearnedClient records mac as learned on the bridge port behind
// interfaceName, in dot1dTpFdbTable and, when vlan is set, in dot1qTpFdbTable
// under that VLAN. vlan is the port's scenario VLAN, not the wire tag.
//
// It reports false, and changes nothing, when this agent models no bridge or
// its IF-MIB has no row for the interface: an entry whose port resolves to no
// ifIndex is one a manager cannot place.
func (a *Agent) PlaceLearnedClient(mac []byte, interfaceName string, vlan int) bool {
	if a == nil || a.device == nil || !a.supportsBridgeTopology() {
		return false
	}

	offset, hasOffset := a.basePortOffset()
	bridgePort, ok := a.bridgePortForInterface(interfaceName, offset, hasOffset)
	if !ok {
		return false
	}

	a.addLearnedFDBEntry(mac, bridgePort)
	if vlan > 0 {
		a.addLearnedQBridgeFDBEntry(vlan, mac, bridgePort)
	}
	a.raiseBaseNumPorts(bridgePort)
	a.mib.Reindex()

	return true
}

// ForgetLearnedClient removes the entries PlaceLearnedClient recorded for mac:
// the client was unplugged from this bridge, so its entry ages out at once.
func (a *Agent) ForgetLearnedClient(mac []byte, vlan int) {
	if a == nil || a.device == nil || !a.supportsBridgeTopology() {
		return
	}

	index := macBytesToOIDIndex(mac)
	a.mib.Delete(dot1dTpFdbAddress + "." + index)
	a.mib.Delete(dot1dTpFdbPort + "." + index)
	a.mib.Delete(dot1dTpFdbStatus + "." + index)
	if vlan > 0 {
		a.removeLearnedQBridgeFDBEntry(vlan, mac)
	}
	a.mib.Reindex()
}
