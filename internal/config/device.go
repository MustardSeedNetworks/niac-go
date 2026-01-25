package config

import (
	"fmt"
	"net"
	"strconv"

	"github.com/krisarmstrong/niac-go/internal/converter"
)

// convertYAMLDevice converts a YAML device to a runtime Device.
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

	return device, nil
}

// createBaseDevice creates a device with default values.
func createBaseDevice(yamlDevice converter.Device) Device {
	deviceType := yamlDevice.Type
	if deviceType == "" {
		deviceType = "unknown"
	}

	return Device{
		Name:       yamlDevice.Name,
		Type:       deviceType,
		Interfaces: make([]Interface, 0),
		Properties: make(map[string]string),
		SNMPConfig: SNMPConfig{
			Community: DefaultSNMPCommunity,
			SysName:   yamlDevice.Name,
		},
	}
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
