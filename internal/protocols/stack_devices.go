package protocols

import (
	"fmt"
	"os"

	"github.com/krisarmstrong/niac-go/internal/config"
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

// selectDiscoveryDevice selects a device for discovery protocol.
func (s *Stack) selectDiscoveryDevice(proto string) *config.Device {
	cfg := s.currentConfig()
	if cfg == nil {
		return nil
	}

	for i := range cfg.Devices {
		dev := &cfg.Devices[i]
		if s.isDeviceEnabledForProtocol(dev, proto) {
			return dev
		}
	}

	if len(cfg.Devices) > 0 {
		return &cfg.Devices[0]
	}

	return nil
}

// isDeviceEnabledForProtocol checks if a device is enabled for a discovery protocol.
func (s *Stack) isDeviceEnabledForProtocol(dev *config.Device, proto string) bool {
	switch proto {
	case ProtocolLLDP:
		return dev.LLDPConfig == nil || dev.LLDPConfig.Enabled
	case ProtocolCDP:
		return dev.CDPConfig == nil || dev.CDPConfig.Enabled
	case ProtocolEDP:
		return dev.EDPConfig == nil || dev.EDPConfig.Enabled
	case ProtocolFDP:
		return dev.FDPConfig == nil || dev.FDPConfig.Enabled
	default:
		return true
	}
}
