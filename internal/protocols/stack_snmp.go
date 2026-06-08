package protocols

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

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

// IncrementStat increments a specific statistic.
