package snmp

// neighbor_mib.go contains the core initialization for all neighbor discovery MIBs.
// The actual MIB implementations are split into separate files:
// - neighbor_mib_lldp.go: LLDP-MIB (IEEE 802.1AB)
// - neighbor_mib_cdp.go: CDP-MIB (Cisco Discovery Protocol)
// - neighbor_mib_ifmib.go: IF-MIB (Interface Table)
// - neighbor_mib_ipmib.go: IP-MIB (IP Address, Routing, ARP Tables)
// - neighbor_mib_bridge.go: BRIDGE-MIB (STP, FDB)
// - neighbor_mib_helpers.go: Helper functions and utilities

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
