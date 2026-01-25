package protocols

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/krisarmstrong/niac-go/internal/apperr"
	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/logging"
)

// GetDevices returns the device table.
func (s *Stack) GetDevices() *DeviceTable {
	return s.devices
}

// GetStats returns current statistics (copy without mutex).
func (s *Stack) GetStats() Statistics {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

	// Return copy of data without mutex
	return Statistics{
		PacketsReceived: s.stats.PacketsReceived,
		PacketsSent:     s.stats.PacketsSent,
		ARPRequests:     s.stats.ARPRequests,
		ARPReplies:      s.stats.ARPReplies,
		ICMPRequests:    s.stats.ICMPRequests,
		ICMPReplies:     s.stats.ICMPReplies,
		DNSQueries:      s.stats.DNSQueries,
		DHCPRequests:    s.stats.DHCPRequests,
		SNMPQueries:     s.stats.SNMPQueries,
		Errors:          s.stats.Errors,
	}
}

// IncrementStat increments a specific statistic.
func (s *Stack) IncrementStat(stat string) {
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()

	switch stat {
	case "arp_requests":
		s.stats.ARPRequests++
	case "arp_replies":
		s.stats.ARPReplies++
	case "icmp_requests":
		s.stats.ICMPRequests++
	case "icmp_replies":
		s.stats.ICMPReplies++
	case "dns_queries":
		s.stats.DNSQueries++
	case "dhcp_requests":
		s.stats.DHCPRequests++
	}
}

// GetDebugLevel returns the current global debug level.
func (s *Stack) GetDebugLevel() int {
	return s.debugConfig.GetGlobal()
}

// GetProtocolDebugLevel returns the debug level for a specific protocol.
func (s *Stack) GetProtocolDebugLevel(protocol string) int {
	return s.debugConfig.GetProtocolLevel(protocol)
}

// GetDebugConfig returns the debug configuration.
func (s *Stack) GetDebugConfig() *logging.DebugConfig {
	return s.debugConfig
}

// GetDHCPHandler returns the DHCP handler for configuration.
func (s *Stack) GetDHCPHandler() *DHCPHandler {
	return s.dhcpHandler
}

// GetDHCPv6Handler returns the DHCPv6 handler for configuration.
func (s *Stack) GetDHCPv6Handler() *DHCPv6Handler {
	return s.dhcpv6Handler
}

// GetDNSHandler returns the DNS handler for configuration.
func (s *Stack) GetDNSHandler() *DNSHandler {
	return s.dnsHandler
}

// GetNeighbors returns a snapshot of the current neighbor table.
func (s *Stack) GetNeighbors() []NeighborRecord {
	if s.neighbors == nil {
		return nil
	}

	return s.neighbors.list()
}

// GetErrorManager returns the error state manager.
func (s *Stack) GetErrorManager() *apperr.StateManager {
	return s.errorManager
}

func (s *Stack) currentConfig() *config.Config {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	return s.config
}

func (s *Stack) recordNeighbor(entry NeighborRecord) {
	if s.neighbors == nil {
		return
	}

	s.neighbors.upsert(entry)
}

func (s *Stack) startNeighborCleanupLoop() {
	if s.neighbors == nil {
		return
	}

	s.wg.Go(func() {
		ticker := time.NewTicker(stackNeighborCleanupSec * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.neighbors.cleanupExpired()
			case <-s.stopChan:
				return
			}
		}
	})
}

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

func (s *Stack) updateFDBTables(mac net.HardwareAddr) {
	if s == nil || len(mac) == 0 {
		return
	}

	decMac, hexMac := formatMACForFDB(mac)

	for _, device := range s.devices.GetForwardingDevices() {
		if device == nil {
			continue
		}

		s.updateDeviceFDBTables(device, decMac, hexMac)
	}
}

// formatMACForFDB formats a MAC address for FDB table entries.
func formatMACForFDB(mac net.HardwareAddr) (string, string) {
	var decMacBuilder, hexMacBuilder strings.Builder

	for _, b := range mac {
		decMacBuilder.WriteString(".")
		decMacBuilder.WriteString(strconv.Itoa(int(b)))
		_, _ = fmt.Fprintf(&hexMacBuilder, "%02X ", b)
	}

	return decMacBuilder.String(), hexMacBuilder.String()
}

// updateDeviceFDBTables updates FDB tables for a single device.
func (s *Stack) updateDeviceFDBTables(device *config.Device, decMac, hexMac string) {
	group := s.ensureSNMPAgentGroup(device)
	debugLevel := s.debugConfig.GetProtocolLevel(logging.ProtocolSNMP)
	baseCommunity := s.getBaseCommunity(device)

	s.updateDot1DFdbTable(device, group, baseCommunity, decMac, hexMac, debugLevel)
	s.updateDot1QFdbTable(device, group, baseCommunity, decMac, hexMac, debugLevel)
}

// ensureSNMPAgentGroup ensures an SNMP agent group exists for a device.
func (s *Stack) ensureSNMPAgentGroup(device *config.Device) *snmpAgentGroup {
	group := s.snmpAgents[device]
	if group == nil {
		group = newSnmpAgentGroup()
		s.snmpAgents[device] = group
	}

	return group
}

// getBaseCommunity returns the base SNMP community for a device.
func (s *Stack) getBaseCommunity(device *config.Device) string {
	if device.SNMPConfig.Community != "" {
		return device.SNMPConfig.Community
	}

	return config.DefaultSNMPCommunity
}

// updateDot1DFdbTable updates the dot1d FDB table for a device.
func (s *Stack) updateDot1DFdbTable(
	device *config.Device,
	group *snmpAgentGroup,
	baseCommunity, decMac, hexMac string,
	debugLevel int,
) {
	if device.SNMPConfig.Dot1DFdbTable == nil {
		return
	}

	port := device.SNMPConfig.Dot1DFdbTable.Port
	vlan := device.SNMPConfig.Dot1DFdbTable.VLAN

	community := baseCommunity
	if vlan > 0 {
		community = fmt.Sprintf("%s@%d", baseCommunity, vlan)
	}

	addressMib := ".1.3.6.1.2.1.17.4.3.1.1" + decMac
	portMib := ".1.3.6.1.2.1.17.4.3.1.2" + decMac
	statusMib := ".1.3.6.1.2.1.17.4.3.1.3" + decMac

	agent := group.Ensure(community, device, debugLevel)
	_ = agent.AddMib(addressMib, "STRING", "fixed("+hexMac+")")
	_ = agent.AddMib(portMib, "INTEGER", fmt.Sprintf("fixed(%d)", port))
	_ = agent.AddMib(statusMib, "INTEGER", "fixed(3)")
}

// updateDot1QFdbTable updates the dot1q FDB table for a device.
func (s *Stack) updateDot1QFdbTable(
	device *config.Device,
	group *snmpAgentGroup,
	baseCommunity, decMac, hexMac string,
	debugLevel int,
) {
	if device.SNMPConfig.Dot1QFdbTable == nil {
		return
	}

	port := device.SNMPConfig.Dot1QFdbTable.Port
	vlan := device.SNMPConfig.Dot1QFdbTable.VLAN

	if vlan <= 0 {
		return
	}

	addressMib := fmt.Sprintf(".1.3.6.1.2.1.17.7.1.2.2.1.1.%d%s", vlan, decMac)
	portMib := fmt.Sprintf(".1.3.6.1.2.1.17.7.1.2.2.1.2.%d%s", vlan, decMac)
	statusMib := fmt.Sprintf(".1.3.6.1.2.1.17.7.1.2.2.1.3.%d%s", vlan, decMac)

	agent := group.Ensure(baseCommunity, device, debugLevel)
	_ = agent.AddMib(addressMib, "STRING", "fixed("+hexMac+")")
	_ = agent.AddMib(portMib, "INTEGER", fmt.Sprintf("fixed(%d)", port))
	_ = agent.AddMib(statusMib, "INTEGER", "fixed(3)")
}
