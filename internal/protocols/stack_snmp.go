package protocols

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

func (s *Stack) initSNMPAgent(device *config.Device) {
	v2Enabled := snmpEnabled(device.SNMPConfig)
	v3Enabled := snmpv3Enabled(device.SNMPv3Config)
	if !v2Enabled && !v3Enabled {
		return
	}

	debugLevel := s.debugConfig.GetProtocolLevel(logging.ProtocolSNMP)
	group := newSnmpAgentGroup(s.deviceStates[device])

	var baseAgent *snmp.Agent
	if v2Enabled {
		baseAgent = group.Ensure(device.SNMPConfig.Community, device, debugLevel)
	} else {
		baseAgent = snmp.NewAgentWithState(device, group.state, snmp.AgentOptions{
			DebugLevel: debugLevel,
			Telemetry:  group.telemetry,
		})
	}
	group.baseAgent = baseAgent

	s.initSNMPv3Engine(device, group, baseAgent, debugLevel)

	// Load each configured walk once into the base community agent. The YAML
	// adapter retains the singular path in WalkFile and WalkFiles, so iterating
	// both fields independently would replay the same capture twice.
	for _, walkFile := range configuredWalkFiles(device.SNMPConfig) {
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

	// Apply authored MIB entries to the base agent used by the enabled SNMP
	// transport. An SNMPv3-only device must not gain a v2c community as a side
	// effect of loading these values.
	for _, mib := range device.SNMPConfig.AddMibs {
		err := baseAgent.AddMib(mib.OID, mib.Type, mib.Value)
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

	// Now that the MIB is loaded, fill downstream bridge FDB entries from the
	// fleet roster (a host MAC on the access port it hangs off) so a scanner can
	// answer "nearest switch/port" for discovered hosts.
	if !v2Enabled {
		baseAgent.SynthesizePeerTopology(s.peerResolver())
	}
	group.SynthesizePeerTopologyAll(s.peerResolver())

	// Every MIB is now fully loaded (walk files, AddMib, topology, peer FDB).
	// Build the sorted OID indexes eagerly so the first GetNext of a discovery
	// does not trigger a large sort on the stack's single decode goroutine —
	// which would stall SNMP responses fleet-wide when a scanner walks every
	// device at once.
	if !v2Enabled {
		baseAgent.Reindex()
	}
	group.ReindexAll()

	s.snmpAgents[device] = group
}

func configuredWalkFiles(cfg config.SNMPConfig) []string {
	files := make([]string, 0, len(cfg.WalkFiles)+1)
	seen := make(map[string]struct{}, len(cfg.WalkFiles)+1)
	candidates := append([]string(nil), cfg.WalkFiles...)
	candidates = append(candidates, cfg.WalkFile)
	for _, file := range candidates {
		if file == "" {
			continue
		}
		if _, exists := seen[file]; exists {
			continue
		}
		seen[file] = struct{}{}
		files = append(files, file)
	}
	return files
}

// peerResolver returns topology identity over the whole config roster. An SNMP
// agent knows only its own device, so cross-device resolution has to come from
// the stack.
func (s *Stack) peerResolver() snmp.PeerResolver {
	byName := make(map[string]*config.Device, len(s.config.Devices))

	for i := range s.config.Devices {
		dev := &s.config.Devices[i]
		byName[dev.Name] = dev
	}

	return func(name, interfaceName string) (snmp.PeerIdentity, bool) {
		device, ok := byName[name]
		if !ok {
			return snmp.PeerIdentity{}, false
		}

		return snmp.PeerIdentity{
			MAC:     device.MACAddress,
			Address: peerInterfaceAddress(device, interfaceName),
		}, true
	}
}

func peerInterfaceAddress(device *config.Device, interfaceName string) net.IP {
	for _, iface := range device.Interfaces {
		if iface.Name != interfaceName {
			continue
		}
		address, _, err := net.ParseCIDR(iface.Address)
		if err == nil {
			if ipv4 := address.To4(); ipv4 != nil {
				return ipv4
			}
		}
		break
	}

	return firstIPv4Address(device)
}

// initSNMPv3Engine builds and attaches the device's SNMPv3 authoritative
// engine to the agent group. The base community agent backs the MIB for all
// USM users (v3 has no community dimension).
func (s *Stack) initSNMPv3Engine(
	device *config.Device,
	group *snmpAgentGroup,
	baseAgent *snmp.Agent,
	debugLevel int,
) {
	if !snmpv3Enabled(device.SNMPv3Config) {
		return
	}

	engine, err := snmp.NewV3Engine(device.SNMPv3Config, device.MACAddress)
	if err != nil {
		if debugLevel >= 1 {
			_, _ = fmt.Fprintf(os.Stdout, "SNMP: v3 engine init failed for %s: %v\n", device.Name, err)
		}
		return
	}
	if engine == nil {
		return
	}

	group.v3 = engine
	group.v3Agent = baseAgent
}

// snmpv3Enabled reports whether a device has usable SNMPv3 USM configuration.
func snmpv3Enabled(cfg *config.SNMPv3Config) bool {
	return cfg != nil && cfg.Enabled && len(cfg.Users) > 0
}

func snmpEnabled(cfg config.SNMPConfig) bool {
	if cfg.Enabled != nil && !*cfg.Enabled {
		return false
	}

	return strings.TrimSpace(cfg.Community) != ""
}

func (s *Stack) getSNMPAgents(device *config.Device) *snmpAgentGroup {
	if s == nil {
		return nil
	}

	return s.snmpAgents[device]
}

func (s *Stack) interfaceIndexResolver(device *config.Device) func(string) (int, bool) {
	group := s.snmpAgents[device]
	if group == nil {
		return nil
	}
	return group.interfaceIndex
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
	if !snmpEnabled(device.SNMPConfig) {
		return
	}

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
		group = newSnmpAgentGroup(s.deviceStates[device])
		s.snmpAgents[device] = group
	}

	return group
}

// getBaseCommunity returns the base SNMP community for a device.
func (s *Stack) getBaseCommunity(device *config.Device) string {
	return strings.TrimSpace(device.SNMPConfig.Community)
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
