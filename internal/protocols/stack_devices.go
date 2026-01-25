package protocols

import (
	"fmt"
	"os"

	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/logging"
)

// initializeDevices repopulates device-dependent state from the provided config.
func (s *Stack) initializeDevices(cfg *config.Config) {
	if cfg == nil {
		return
	}

	s.resetDeviceState()

	for i := range cfg.Devices {
		device := &cfg.Devices[i]
		s.registerDevice(device)
	}

	s.applySNMPAddrMappings(cfg)

	s.configMu.Lock()
	s.config = cfg
	s.configMu.Unlock()

	if s.debugConfig.GetGlobal() >= DebugLevelBasic {
		_, _ = fmt.Fprintf(
			os.Stdout,
			"Initialized %d devices from configuration\n",
			len(cfg.Devices),
		)
	}
}

// resetDeviceState resets all device-related state.
func (s *Stack) resetDeviceState() {
	if s.devices == nil {
		s.devices = NewDeviceTable()
	} else {
		s.devices.Reset()
	}

	s.snmpAgents = make(map[*config.Device]*snmpAgentGroup)

	if s.dhcpHandler != nil {
		s.dhcpHandler.Reset()
	}

	if s.dhcpv6Handler != nil {
		s.dhcpv6Handler.Reset()
	}

	if s.dnsHandler != nil {
		s.dnsHandler.Reset()
	}
}

// registerDevice registers a single device with all relevant handlers.
func (s *Stack) registerDevice(device *config.Device) {
	s.registerDeviceAddresses(device)
	s.registerDeviceFeatures(device)
	s.configureDHCPServer(device)
	s.initSNMPAgent(device)

	if device.DNSConfig != nil {
		s.dnsHandler.LoadDeviceDNSConfig(device)
	}
}

// registerDeviceAddresses registers device MAC and IP addresses.
func (s *Stack) registerDeviceAddresses(device *config.Device) {
	if len(device.MACAddress) > 0 {
		s.devices.AddByMAC(device.MACAddress, device)
	}

	for _, ip := range device.IPAddresses {
		s.devices.AddByIP(ip, device)
	}
}

// registerDeviceFeatures registers TTL and forwarding device features.
func (s *Stack) registerDeviceFeatures(device *config.Device) {
	if device.TTLConfig != nil {
		s.devices.RegisterTTL(device)
	}

	if device.SNMPConfig.Dot1DFdbTable != nil || device.SNMPConfig.Dot1QFdbTable != nil {
		s.devices.RegisterForwardingDevice(device)
	}
}

// configureDHCPServer configures DHCP server for a device if it has DHCP config.
func (s *Stack) configureDHCPServer(device *config.Device) {
	if device.DHCPConfig == nil {
		return
	}

	if device.DHCPConfig.PoolStart != nil && device.DHCPConfig.PoolEnd != nil {
		s.dhcpHandler.SetPool(device.DHCPConfig.PoolStart, device.DHCPConfig.PoolEnd)
	}

	serverIP := device.DHCPConfig.ServerIdentifier
	if serverIP == nil && len(device.IPAddresses) > 0 {
		serverIP = device.IPAddresses[0]
	}

	s.dhcpHandler.SetServerConfig(
		serverIP,
		device.DHCPConfig.Router,
		device.DHCPConfig.DomainNameServer,
		device.DHCPConfig.DomainName,
	)

	s.dhcpHandler.SetAdvancedOptions(
		device.DHCPConfig.NTPServers,
		device.DHCPConfig.DomainSearch,
		device.DHCPConfig.TFTPServerName,
		device.DHCPConfig.BootfileName,
		device.DHCPConfig.VendorSpecific,
	)
	s.dhcpHandler.SetStaticLeases(device.DHCPConfig.ClientLeases)

	if s.debugConfig.GetGlobal() >= DebugLevelBasic {
		_, _ = fmt.Fprintf(os.Stdout, "Configured DHCP server for device %s\n", device.Name)
	}
}

// applySNMPAddrMappings applies SNMP agent sharing mappings.
func (s *Stack) applySNMPAddrMappings(cfg *config.Config) {
	for i := range cfg.Devices {
		device := &cfg.Devices[i]
		if device.SNMPConfig.SnmpAddr == nil {
			continue
		}

		targets := s.devices.GetByIP(device.SNMPConfig.SnmpAddr)
		if len(targets) == 0 {
			continue
		}

		if group, ok := s.snmpAgents[targets[0]]; ok {
			s.snmpAgents[device] = group
		}
	}
}

func (s *Stack) initSNMPAgent(device *config.Device) {
	if !snmpEnabled(device.SNMPConfig) {
		return
	}

	debugLevel := s.debugConfig.GetProtocolLevel(logging.ProtocolSNMP)
	group := newSnmpAgentGroup()

	baseCommunity := device.SNMPConfig.Community
	if baseCommunity == "" {
		baseCommunity = config.DefaultSNMPCommunity
	}

	baseAgent := group.Ensure(baseCommunity, device, debugLevel)

	// Load walk files into base community agent
	for _, walkFile := range device.SNMPConfig.WalkFiles {
		err := baseAgent.LoadWalkFile(walkFile)
		if err != nil && debugLevel >= 1 {
			_, _ = fmt.Fprintf(
				os.Stdout,
				"SNMP: failed to load walk file for %s: %v\n",
				device.Name,
				err,
			)
		}
	}

	if device.SNMPConfig.WalkFile != "" {
		err := baseAgent.LoadWalkFile(device.SNMPConfig.WalkFile)
		if err != nil && debugLevel >= 1 {
			_, _ = fmt.Fprintf(
				os.Stdout,
				"SNMP: failed to load walk file for %s: %v\n",
				device.Name,
				err,
			)
		}
	}

	// Load community-specific walk files
	for _, include := range device.SNMPConfig.CommunityIncludes {
		agent := group.Ensure(include.Community, device, debugLevel)
		err := agent.LoadWalkFile(include.WalkFile)
		if err != nil && debugLevel >= 1 {
			_, _ = fmt.Fprintf(
				os.Stdout,
				"SNMP: failed to load walk file for %s (%s): %v\n",
				device.Name,
				include.Community,
				err,
			)
		}
	}

	// Apply AddMib entries to base community (Java uses public)
	for _, mib := range device.SNMPConfig.AddMibs {
		agent := group.Ensure(config.DefaultSNMPCommunity, device, debugLevel)
		err := agent.AddMib(mib.OID, mib.Type, mib.Value)
		if err != nil && debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(
				os.Stdout,
				"SNMP: AddMib failed for %s oid=%s err=%v\n",
				device.Name,
				mib.OID,
				err,
			)
		}
	}

	s.snmpAgents[device] = group
}

func snmpEnabled(cfg config.SNMPConfig) bool {
	if cfg.Community != "" || cfg.WalkFile != "" || len(cfg.WalkFiles) > 0 || cfg.SysName != "" ||
		cfg.SysDescr != "" || cfg.SysContact != "" || cfg.SysLocation != "" {
		return true
	}

	if len(cfg.AddMibs) > 0 || len(cfg.CommunityIncludes) > 0 || len(cfg.AccessList) > 0 ||
		cfg.SnmpAddr != nil {
		return true
	}

	if cfg.Dot1DFdbTable != nil || cfg.Dot1QFdbTable != nil {
		return true
	}

	if cfg.Traps != nil && cfg.Traps.Enabled {
		return true
	}

	return false
}

func (s *Stack) getSNMPAgents(device *config.Device) *snmpAgentGroup {
	if s == nil {
		return nil
	}

	return s.snmpAgents[device]
}
