package config

import (
	"net"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

// parseDHCPConfig parses DHCP configuration from YAML.
func parseDHCPConfig(yamlDhcp *converter.DhcpServer) *DHCPConfig {
	if yamlDhcp == nil {
		return nil
	}

	dhcpCfg := &DHCPConfig{}

	parseDHCPBasicOptions(dhcpCfg, yamlDhcp)
	parseDHCPPoolConfig(dhcpCfg, yamlDhcp)
	parseDHCPv4Options(dhcpCfg, yamlDhcp)
	parseDHCPv6Options(dhcpCfg, yamlDhcp)
	parseDHCPClientLeases(dhcpCfg, yamlDhcp)
	parseDHCPScopes(dhcpCfg, yamlDhcp)

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

// parseDHCPScopes parses the pools a server leases through DHCP relays.
func parseDHCPScopes(cfg *DHCPConfig, yamlDhcp *converter.DhcpServer) {
	for _, scope := range yamlDhcp.Scopes {
		cfg.Scopes = append(cfg.Scopes, DHCPScope{
			PoolStart: net.ParseIP(scope.PoolStart),
			PoolEnd:   net.ParseIP(scope.PoolEnd),
			Router:    net.ParseIP(scope.Router),
		})
	}
}
