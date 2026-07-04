package protocols

import (
	"fmt"
	"os"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
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
//
// A config with explicit Segments (ADR 0008 multi-VLAN) builds one isolated
// DeviceTable per VLAN tag instead of the single flat table, so two (or more)
// independent "demos" can run at once — even reusing the same IPs — and get
// demultiplexed by devicesFor at packet-routing time. A flat config (no
// Segments) takes the original code path byte-for-byte: this branch is the
// load-bearing backward-compat guarantee, not just an optimization.
func (s *Stack) initializeDevices(cfg *config.Config) {
	if cfg == nil {
		return
	}

	s.resetDeviceState()

	if len(cfg.Segments) > 0 {
		s.initializeSegments(cfg)
	} else {
		for i := range cfg.Devices {
			device := &cfg.Devices[i]
			s.registerDevice(device)
		}

		s.applySNMPAddrMappings(cfg.Devices, s.devices)
	}

	s.configMu.Lock()
	s.config = cfg
	s.configMu.Unlock()

	if s.debugConfig.GetGlobal() >= DebugLevelBasic {
		_, _ = fmt.Fprintf(
			os.Stdout,
			"Initialized %d devices from configuration\n",
			cfg.DeviceCount(),
		)
	}
}

// initializeSegments builds one isolated DeviceTable per VLAN segment
// (ADR 0008 slice 2). Each segment's inline Devices get their own table so
// devicesFor can route a frame to the correct "demo" by VLAN tag, even when
// two segments reuse the same IP addresses.
//
// Segments that only set ConfigPath (no inline Devices) are skipped for now —
// resolving a segment config file is a separate slice — so their VLAN tag
// ends up with no table and devicesFor(tag) returns nil for it, same as any
// other unclaimed tag.
func (s *Stack) initializeSegments(cfg *config.Config) {
	s.segmentTables = make(map[int]*DeviceTable)

	for _, seg := range cfg.NormalizedSegments() {
		if len(seg.Devices) == 0 {
			// TODO(slice-2): resolve ConfigPath via config.LoadYAML
			continue
		}

		segTable := NewDeviceTable()

		for i := range seg.Devices {
			device := &seg.Devices[i]
			s.registerSegmentDevice(device, segTable)
		}

		s.applySNMPAddrMappings(seg.Devices, segTable)

		s.segmentTables[seg.Tag] = segTable
	}
}

// resetDeviceState resets all device-related state.
func (s *Stack) resetDeviceState() {
	if s.devices == nil {
		s.devices = NewDeviceTable()
	} else {
		s.devices.Reset()
	}

	// Cleared unconditionally: a reload that drops Segments from a
	// previously-segmented config must fall back to flat mode, and
	// initializeSegments rebuilds this map from scratch whenever segments
	// are present.
	s.segmentTables = nil

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

// registerDevice registers a single device with all relevant handlers,
// targeting the stack's single flat device table (the no-segments path).
func (s *Stack) registerDevice(device *config.Device) {
	s.registerDeviceAddresses(device, s.devices)
	s.registerDeviceFeatures(device, s.devices)
	s.configureDHCPServer(device)
	s.initSNMPAgent(device)

	if device.DNSConfig != nil {
		s.dnsHandler.LoadDeviceDNSConfig(device)
	}
}

// registerSegmentDevice registers a single device the same way registerDevice
// does, except its table-scoped state (addresses, TTL, FDB) goes into the
// given segment's own table instead of the flat s.devices — devicesFor(vlan)
// only ever returns segment tables in segment mode, so TTL-timeout routing
// (ip.go's handleTTLTimeout) would silently never find a registration left
// behind in s.devices. The remaining handler state (DHCP, SNMP, DNS) is keyed
// by *config.Device pointer, not by table, so it coexists safely across
// segments even when two segments reuse the same IP.
func (s *Stack) registerSegmentDevice(device *config.Device, table *DeviceTable) {
	s.registerDeviceAddresses(device, table)
	s.registerDeviceFeatures(device, table)
	s.configureDHCPServer(device)
	s.initSNMPAgent(device)

	if device.DNSConfig != nil {
		s.dnsHandler.LoadDeviceDNSConfig(device)
	}
}

// registerDeviceAddresses registers device MAC and IP addresses into table.
func (s *Stack) registerDeviceAddresses(device *config.Device, table *DeviceTable) {
	if len(device.MACAddress) > 0 {
		table.AddByMAC(device.MACAddress, device)
	}

	for _, ip := range device.IPAddresses {
		table.AddByIP(ip, device)
	}
}

// registerDeviceFeatures registers TTL and forwarding device features into
// table. Like addresses, TTL/FDB state lives on the DeviceTable itself
// (DeviceTable.ttlDevices / .forwardingDevices), so it must land in the same
// table devicesFor(vlan) will look it up from.
func (s *Stack) registerDeviceFeatures(device *config.Device, table *DeviceTable) {
	if device.TTLConfig != nil {
		table.RegisterTTL(device)
	}

	if device.SNMPConfig.Dot1DFdbTable != nil || device.SNMPConfig.Dot1QFdbTable != nil {
		table.RegisterForwardingDevice(device)
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

// applySNMPAddrMappings applies SNMP agent sharing mappings for devices,
// resolving each device's SnmpAddr against table. The flat path calls this
// with (cfg.Devices, s.devices); segment mode calls it once per segment with
// that segment's own (Devices, table) so an snmp_addr reference resolves
// within the same isolated demo instead of leaking across segments.
func (s *Stack) applySNMPAddrMappings(devices []config.Device, table *DeviceTable) {
	for i := range devices {
		device := &devices[i]
		if device.SNMPConfig.SnmpAddr == nil {
			continue
		}

		targets := table.GetByIP(device.SNMPConfig.SnmpAddr)
		if len(targets) == 0 {
			continue
		}

		if group, ok := s.snmpAgents[targets[0]]; ok {
			s.snmpAgents[device] = group
		}
	}
}

// Start starts the protocol stack processing.
