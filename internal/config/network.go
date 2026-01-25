package config

import (
	"fmt"
	"net"
	"strconv"

	"github.com/krisarmstrong/niac-go/internal/converter"
)

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

// parseDHCPConfig parses DHCP configuration from YAML
// Returns an empty DHCPConfig if input is nil (not an error condition).
func parseDHCPConfig(yamlDhcp *converter.DhcpServer) *DHCPConfig {
	if yamlDhcp == nil {
		return &DHCPConfig{}
	}

	dhcpCfg := &DHCPConfig{}

	parseDHCPBasicOptions(dhcpCfg, yamlDhcp)
	parseDHCPPoolConfig(dhcpCfg, yamlDhcp)
	parseDHCPv4Options(dhcpCfg, yamlDhcp)
	parseDHCPv6Options(dhcpCfg, yamlDhcp)
	parseDHCPClientLeases(dhcpCfg, yamlDhcp)

	return dhcpCfg
}

// parseDHCPBasicOptions parses basic DHCP options.
func parseDHCPBasicOptions(cfg *DHCPConfig, yamlDhcp *converter.DhcpServer) {
	if yamlDhcp.SubnetMask != "" {
		if ip := net.ParseIP(yamlDhcp.SubnetMask); ip != nil {
			cfg.SubnetMask = net.IPMask(ip.To4())
		}
	}

	if yamlDhcp.Router != "" {
		cfg.Router = net.ParseIP(yamlDhcp.Router)
	}

	if yamlDhcp.DomainNameServer != "" {
		if ip := net.ParseIP(yamlDhcp.DomainNameServer); ip != nil {
			cfg.DomainNameServer = append(cfg.DomainNameServer, ip)
		}
	}

	if yamlDhcp.ServerIdentifier != "" {
		cfg.ServerIdentifier = net.ParseIP(yamlDhcp.ServerIdentifier)
	}

	if yamlDhcp.NextServerIP != "" {
		cfg.NextServerIP = net.ParseIP(yamlDhcp.NextServerIP)
	}
}

// parseDHCPPoolConfig parses DHCP address pool configuration.
func parseDHCPPoolConfig(cfg *DHCPConfig, yamlDhcp *converter.DhcpServer) {
	if yamlDhcp.PoolStart != "" {
		cfg.PoolStart = net.ParseIP(yamlDhcp.PoolStart)
	}

	if yamlDhcp.PoolEnd != "" {
		cfg.PoolEnd = net.ParseIP(yamlDhcp.PoolEnd)
	}
}

// parseDHCPv4Options parses DHCPv4 high priority options.
func parseDHCPv4Options(cfg *DHCPConfig, yamlDhcp *converter.DhcpServer) {
	cfg.NTPServers = parseIPList(yamlDhcp.NTPServers)
	cfg.DomainSearch = yamlDhcp.DomainSearch
	cfg.TFTPServerName = yamlDhcp.TFTPServerName
	cfg.BootfileName = yamlDhcp.BootfileName

	if yamlDhcp.VendorSpecific != "" {
		cfg.VendorSpecific = []byte(yamlDhcp.VendorSpecific)
	}
}

// parseDHCPv6Options parses DHCPv6 options.
func parseDHCPv6Options(cfg *DHCPConfig, yamlDhcp *converter.DhcpServer) {
	cfg.SNTPServersV6 = parseIPList(yamlDhcp.SNTPServersV6)
	cfg.NTPServersV6 = parseIPList(yamlDhcp.NTPServersV6)
	cfg.SIPServersV6 = parseIPList(yamlDhcp.SIPServersV6)
	cfg.SIPDomainsV6 = yamlDhcp.SIPDomainsV6
}

// parseDHCPClientLeases parses static DHCP lease assignments.
func parseDHCPClientLeases(cfg *DHCPConfig, yamlDhcp *converter.DhcpServer) {
	for _, lease := range yamlDhcp.ClientLeases {
		dhcpLease := parseSingleDHCPLease(lease)
		if dhcpLease != nil {
			cfg.ClientLeases = append(cfg.ClientLeases, *dhcpLease)
		}
	}
}

// parseSingleDHCPLease parses a single DHCP lease entry.
func parseSingleDHCPLease(lease converter.DhcpLease) *DHCPLease {
	clientIP := net.ParseIP(lease.ClientIP)
	if clientIP == nil {
		return nil
	}

	macAddr, err := net.ParseMAC(lease.MacAddrValue)
	if err != nil {
		return nil
	}

	dhcpLease := &DHCPLease{
		ClientIP:   clientIP,
		MACAddress: macAddr,
	}

	if lease.MacAddrMask != "" {
		if mask, maskErr := net.ParseMAC(lease.MacAddrMask); maskErr == nil {
			dhcpLease.MACMask = mask
		}
	}

	return dhcpLease
}

// parseIPList parses a list of IP address strings into a [net.IP] slice.
func parseIPList(ipStrings []string) []net.IP {
	var ips []net.IP

	for _, ipStr := range ipStrings {
		if ip := net.ParseIP(ipStr); ip != nil {
			ips = append(ips, ip)
		}
	}

	return ips
}

// parseDNSConfig parses DNS configuration from YAML
// Returns an empty DNSConfig if input is nil (not an error condition).
func parseDNSConfig(yamlDNS *converter.DNSServer, deviceName string) (*DNSConfig, error) {
	if yamlDNS == nil {
		return &DNSConfig{}, nil
	}

	dnsCfg := &DNSConfig{}

	forwardRecords, err := parseDNSRecords(yamlDNS.ForwardRecords, deviceName)
	if err != nil {
		return nil, err
	}

	dnsCfg.ForwardRecords = forwardRecords

	reverseRecords, err := parseDNSRecords(yamlDNS.ReverseRecords, deviceName)
	if err != nil {
		return nil, err
	}

	dnsCfg.ReverseRecords = reverseRecords

	return dnsCfg, nil
}

// parseDNSRecords parses a list of DNS records with TTL validation.
func parseDNSRecords(records []converter.DNSRecord, deviceName string) ([]DNSRecord, error) {
	var result []DNSRecord

	for _, record := range records {
		ip := net.ParseIP(record.IP)
		if ip == nil {
			continue
		}

		ttl, err := validateDNSTTL(record.TTL, deviceName)
		if err != nil {
			return nil, err
		}

		result = append(result, DNSRecord{
			Name:  record.Name,
			IP:    ip,
			TTL:   ttl,
			RCode: record.RCode,
		})
	}

	return result, nil
}
