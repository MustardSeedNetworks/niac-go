package protocols

import (
	"fmt"
	"os"

	"github.com/krisarmstrong/niac-go/internal/config"
)

func (s *Stack) AddPacketObserver(obs PacketObserver) {
	if obs == nil {
		return
	}
	s.observerMu.Lock()
	s.observers = append(s.observers, obs)
	s.observerMu.Unlock()
}

// notifyObservers fans a packet event out to every registered observer.
// A panicking observer is dropped — the stack's job is to forward
// packets, not to keep buggy observers safe.
func (s *Stack) notifyObservers(direction string, pkt *Packet) {
	s.observerMu.RLock()
	obs := s.observers
	s.observerMu.RUnlock()
	for _, o := range obs {
		func() {
			defer func() {
				_ = recover()
			}()
			o.OnPacket(direction, pkt)
		}()
	}
}

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

// Start starts the protocol stack processing.
