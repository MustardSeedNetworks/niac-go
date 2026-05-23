package config

import (
	"fmt"
	"maps"
	"net"
	"strconv"
	"strings"

	"github.com/krisarmstrong/niac-go/internal/converter"
)

func convertYAMLDevice(yamlDevice converter.Device, includePath string) (Device, error) {
	device := createBaseDevice(yamlDevice)

	if err := parseDeviceMACAddress(&device, &yamlDevice); err != nil {
		return device, err
	}

	if err := parseDeviceIPAddresses(&device, &yamlDevice); err != nil {
		return device, err
	}

	if err := parseDeviceMapToIP(&device, &yamlDevice); err != nil {
		return device, err
	}

	device.Babble = yamlDevice.Babble

	if err := parseDeviceTTLConfig(&device, &yamlDevice); err != nil {
		return device, err
	}

	if err := parseDeviceSNMPConfig(&device, &yamlDevice, includePath); err != nil {
		return device, err
	}

	if yamlDevice.VLAN > 0 {
		device.Properties["vlan"] = strconv.Itoa(yamlDevice.VLAN)
		device.VLAN = yamlDevice.VLAN
	}

	if err := parseDeviceProtocolConfigs(&device, &yamlDevice); err != nil {
		return device, err
	}

	device.TrunkPorts = convertTrunkPorts(yamlDevice.TrunkPorts)
	device.PortChannels = convertPortChannels(yamlDevice.PortChannels)

	return device, nil
}

// convertTrunkPorts translates the YAML-shaped TrunkPort slice (from
// converter.Device) into the runtime config.TrunkPort slice the topology
// builder consults. The two types are field-identical; this is just a
// type adapter.
func convertTrunkPorts(in []converter.TrunkPort) []TrunkPort {
	if len(in) == 0 {
		return nil
	}
	out := make([]TrunkPort, len(in))
	for i, t := range in {
		out[i] = TrunkPort{
			Interface:       t.Interface,
			VLANs:           t.VLANs,
			NativeVLAN:      t.NativeVLAN,
			RemoteDevice:    t.RemoteDevice,
			RemoteInterface: t.RemoteInterface,
		}
	}
	return out
}

// convertPortChannels mirrors convertTrunkPorts for LAG bundles.
func convertPortChannels(in []converter.PortChannel) []PortChannel {
	if len(in) == 0 {
		return nil
	}
	out := make([]PortChannel, len(in))
	for i, pc := range in {
		out[i] = PortChannel{
			ID:      pc.ID,
			Members: pc.Members,
			Mode:    pc.Mode,
		}
	}
	return out
}

// createBaseDevice creates a device with default values. The YAML
// properties block (free-form vendor metadata: vendor/model/os/etc.)
// is copied in first so subsequent special-case writes (vlan,
// custom_mibs_count) extend it rather than dropping the originals.
func createBaseDevice(yamlDevice converter.Device) Device {
	props := make(map[string]string, len(yamlDevice.Properties))
	maps.Copy(props, yamlDevice.Properties)
	return Device{
		Name:       yamlDevice.Name,
		Type:       inferYAMLDeviceType(yamlDevice),
		Interfaces: make([]Interface, 0),
		Properties: props,
		SNMPConfig: SNMPConfig{
			Community: DefaultSNMPCommunity,
			SysName:   yamlDevice.Name,
		},
	}
}

func inferYAMLDeviceType(yamlDevice converter.Device) string {
	if yamlDevice.Type != "" {
		return yamlDevice.Type
	}

	name := strings.ToLower(yamlDevice.Name)
	switch {
	case containsAny(name, "router", "rtr", "gateway", "edge", "wan", "border"):
		return "router"
	case containsAny(name, "wlc", "controller", "server", "srv", "dns", "dhcp", "http", "ftp", "netbios", "web", "nas", "noc"):
		return "server"
	case containsAny(name, "ap", "wifi", "wireless"):
		return "ap"
	case containsAny(name, "switch", "sw", "spine", "leaf", "dist", "access", "core"):
		return "switch"
	default:
		return "host"
	}
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}

	return false
}

// parseDeviceMACAddress parses the MAC address for a device.
func parseDeviceMACAddress(device *Device, yamlDevice *converter.Device) error {
	if yamlDevice.MAC == "" {
		return nil
	}

	mac, err := net.ParseMAC(yamlDevice.MAC)
	if err != nil {
		return fmt.Errorf("device %s: invalid MAC address %s: %w", yamlDevice.Name, yamlDevice.MAC, err)
	}

	device.MACAddress = mac

	return nil
}

// parseDeviceMapToIP parses the MapToIP configuration.
func parseDeviceMapToIP(device *Device, yamlDevice *converter.Device) error {
	if yamlDevice.MapToIP == "" {
		return nil
	}

	ip := net.ParseIP(yamlDevice.MapToIP)
	if ip == nil {
		return fmt.Errorf("device %s: %w: %s", yamlDevice.Name, ErrInvalidMapToIP, yamlDevice.MapToIP)
	}

	device.MapToIP = ip

	return nil
}

// parseDeviceTTLConfig parses TTL configuration for traceroute simulation.
func parseDeviceTTLConfig(device *Device, yamlDevice *converter.Device) error {
	if yamlDevice.TTL == nil {
		return nil
	}

	ttlCfg := &TTLConfig{TTL: yamlDevice.TTL.TTL}

	if yamlDevice.TTL.IP != "" {
		ip := net.ParseIP(yamlDevice.TTL.IP)
		if ip == nil {
			return fmt.Errorf("device %s: %w: %s", yamlDevice.Name, ErrInvalidTTLIP, yamlDevice.TTL.IP)
		}

		ttlCfg.IP = ip.To4()
	}

	if yamlDevice.TTL.Mask != "" {
		ip := net.ParseIP(yamlDevice.TTL.Mask)
		if ip == nil {
			return fmt.Errorf("device %s: %w: %s", yamlDevice.Name, ErrInvalidTTLMask, yamlDevice.TTL.Mask)
		}

		ttlCfg.Mask = net.IPMask(ip.To4())
	}

	device.TTLConfig = ttlCfg

	return nil
}

// parseDeviceIPAddresses parses IP addresses for a device.
func parseDeviceIPAddresses(device *Device, yamlDevice *converter.Device) error {
	// Support both singular 'ip' (backward compatible) and plural 'ips' (new feature)
	if yamlDevice.IP != "" {
		ip := net.ParseIP(yamlDevice.IP)
		if ip == nil {
			return fmt.Errorf("device %s: %w: %s", yamlDevice.Name, ErrInvalidIPAddress, yamlDevice.IP)
		}

		device.IPAddresses = append(device.IPAddresses, ip)
	}

	// Parse multiple IPs if specified
	for i, ipStr := range yamlDevice.IPs {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return fmt.Errorf("device %s: %w in ips[%d]: %s", yamlDevice.Name, ErrInvalidIPAddress, i, ipStr)
		}

		device.IPAddresses = append(device.IPAddresses, ip)
	}

	return nil
}

// parseDeviceSNMPConfig parses SNMP configuration for a device.
func parseDeviceSNMPConfig(device *Device, yamlDevice *converter.Device, includePath string) error {
	if yamlDevice.SnmpAgent == nil {
		return nil
	}

	snmpAgent := yamlDevice.SnmpAgent

	if err := parseSNMPWalkFiles(device, snmpAgent, includePath, yamlDevice.Name); err != nil {
		return err
	}

	parseSNMPAddMibs(device, snmpAgent)

	if err := parseSNMPCommunityIncludes(device, snmpAgent, includePath, yamlDevice.Name); err != nil {
		return err
	}

	parseSNMPAccessList(device, snmpAgent)

	if err := parseSNMPAddr(device, snmpAgent, yamlDevice.Name); err != nil {
		return err
	}

	parseSNMPFdbTables(device, snmpAgent)

	if snmpAgent.Traps != nil {
		device.SNMPConfig.Traps = parseSNMPTrapsConfig(snmpAgent.Traps)
	}

	return nil
}

// parseSNMPWalkFiles parses and validates SNMP walk files.
func parseSNMPWalkFiles(device *Device, snmpAgent *converter.SnmpAgent, includePath, deviceName string) error {
	if snmpAgent.WalkFile != "" {
		walkFile, err := validateWalkFilePath(includePath, snmpAgent.WalkFile, deviceName)
		if err != nil {
			return err
		}

		device.SNMPConfig.WalkFile = walkFile
		device.SNMPConfig.WalkFiles = append(device.SNMPConfig.WalkFiles, walkFile)
	}

	for _, walk := range snmpAgent.WalkFiles {
		walkFile, err := validateWalkFilePath(includePath, walk, deviceName)
		if err != nil {
			return err
		}

		device.SNMPConfig.WalkFiles = append(device.SNMPConfig.WalkFiles, walkFile)
	}

	return nil
}

// parseSNMPAddMibs parses custom MIB overrides.
func parseSNMPAddMibs(device *Device, snmpAgent *converter.SnmpAgent) {
	if len(snmpAgent.AddMibs) == 0 {
		return
	}

	device.Properties["custom_mibs_count"] = strconv.Itoa(len(snmpAgent.AddMibs))

	for _, mib := range snmpAgent.AddMibs {
		device.SNMPConfig.AddMibs = append(device.SNMPConfig.AddMibs, AddMib{
			OID:   mib.OID,
			Type:  mib.Type,
			Value: mib.Value,
		})
	}
}

// parseSNMPCommunityIncludes parses community-specific walk includes.
func parseSNMPCommunityIncludes(device *Device, snmpAgent *converter.SnmpAgent, includePath, deviceName string) error {
	for _, include := range snmpAgent.CommunityIncludes {
		walkFile, err := validateWalkFilePath(includePath, include.WalkFile, deviceName)
		if err != nil {
			return err
		}

		device.SNMPConfig.CommunityIncludes = append(device.SNMPConfig.CommunityIncludes, CommunityInclude{
			Community: include.Community,
			WalkFile:  walkFile,
		})
	}

	return nil
}

// parseSNMPAccessList parses the SNMP access list.
func parseSNMPAccessList(device *Device, snmpAgent *converter.SnmpAgent) {
	for _, ipStr := range snmpAgent.AccessList {
		if ip := net.ParseIP(ipStr); ip != nil {
			device.SNMPConfig.AccessList = append(device.SNMPConfig.AccessList, ip)
		}
	}
}

// parseSNMPAddr parses the SNMP address mapping.
func parseSNMPAddr(device *Device, snmpAgent *converter.SnmpAgent, deviceName string) error {
	if snmpAgent.SnmpAddr == "" {
		return nil
	}

	ip := net.ParseIP(snmpAgent.SnmpAddr)
	if ip == nil {
		return fmt.Errorf("device %s: %w: %s", deviceName, ErrInvalidSNMPAddr, snmpAgent.SnmpAddr)
	}

	device.SNMPConfig.SnmpAddr = ip

	return nil
}

// parseSNMPFdbTables parses Dot1D/Dot1Q FDB table configurations.
func parseSNMPFdbTables(device *Device, snmpAgent *converter.SnmpAgent) {
	if snmpAgent.Dot1DFdbTable != nil {
		device.SNMPConfig.Dot1DFdbTable = &FdbTableConfig{
			Port: snmpAgent.Dot1DFdbTable.Port,
			VLAN: snmpAgent.Dot1DFdbTable.VLAN,
		}
	}

	if snmpAgent.Dot1QFdbTable != nil {
		device.SNMPConfig.Dot1QFdbTable = &FdbTableConfig{
			Port: snmpAgent.Dot1QFdbTable.Port,
			VLAN: snmpAgent.Dot1QFdbTable.VLAN,
		}
	}
}

// parseDeviceProtocolConfigs parses all protocol configurations for a device.
func parseDeviceProtocolConfigs(device *Device, yamlDevice *converter.Device) error {
	// Handle DHCP configuration
	device.DHCPConfig = parseDHCPConfig(yamlDevice.Dhcp)

	// Handle DNS configuration
	var err error
	if device.DNSConfig, err = parseDNSConfig(yamlDevice.DNS, yamlDevice.Name); err != nil {
		return err
	}

	// Handle discovery protocols
	device.LLDPConfig = parseLLDPConfig(yamlDevice.Lldp)
	device.CDPConfig = parseCDPConfig(yamlDevice.Cdp)
	device.EDPConfig = parseEDPConfig(yamlDevice.Edp)
	device.FDPConfig = parseFDPConfig(yamlDevice.Fdp)
	device.STPConfig = parseSTPConfig(yamlDevice.Stp)

	// Handle service protocols
	device.HTTPConfig = parseHTTPConfig(yamlDevice.HTTP, device.Name)
	device.FTPConfig = parseFTPConfig(yamlDevice.Ftp, device.Name)
	device.NetBIOSConfig = parseNetBIOSConfig(yamlDevice.Netbios, device.Name)

	// Handle ICMP protocols
	device.ICMPConfig = parseICMPConfig(yamlDevice.Icmp)
	device.ICMPv6Config = parseICMPv6Config(yamlDevice.Icmpv6)

	// Handle DHCPv6 configuration
	device.DHCPv6Config = parseDHCPv6Config(yamlDevice.Dhcpv6)

	// Handle Traffic configuration
	device.TrafficConfig = parseTrafficConfig(yamlDevice.Traffic)

	// Handle OS Fingerprint configuration
	device.OSFingerprintConfig = parseOSFingerprintConfig(yamlDevice.OSFingerprint)

	// Handle iPerf3 configuration
	device.IPerf3 = parseIPerf3Config(yamlDevice.IPerf3)

	return nil
}
