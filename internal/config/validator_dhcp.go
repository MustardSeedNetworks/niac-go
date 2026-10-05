package config

import (
	"fmt"
	"net"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

func (v *Validator) validateDHCPv4(cfg *DHCPConfig, prefix string) {
	if cfg == nil {
		return
	}
	if !converter.ValidDHCPv4Mask(cfg.SubnetMask) {
		v.addError(prefix+".subnet_mask", "must be a contiguous IPv4 subnet mask")
	}
	if !converter.ValidDHCPv4Pool(cfg.PoolStart, cfg.PoolEnd) {
		v.addError(
			prefix+".pool_end",
			fmt.Sprintf(
				"pool endpoints must be paired IPv4 addresses in ascending order with at most %d addresses",
				converter.MaxDHCPv4PoolSize,
			),
		)
	}
	for field, address := range map[string]net.IP{
		"router": cfg.Router, "server_identifier": cfg.ServerIdentifier, "next_server_ip": cfg.NextServerIP,
	} {
		if address != nil && address.To4() == nil {
			v.addError(prefix+"."+field, "must be an IPv4 address")
		}
	}
	v.validateDHCPv4Addresses(cfg, prefix)
}

func (v *Validator) validateDHCPv4Addresses(cfg *DHCPConfig, prefix string) {
	for field, addresses := range map[string][]net.IP{
		"domain_name_server": cfg.DomainNameServer, "ntp_servers": cfg.NTPServers,
	} {
		for index, address := range addresses {
			if address.To4() == nil {
				v.addError(fmt.Sprintf("%s.%s[%d]", prefix, field, index), "must be an IPv4 address")
			}
		}
	}
	for index, lease := range cfg.ClientLeases {
		if lease.ClientIP.To4() == nil {
			v.addError(fmt.Sprintf("%s.client_leases[%d].client_ip", prefix, index), "must be an IPv4 address")
		}
	}
}

// validateDHCPRelayNeedsNetworks refuses relayed DHCP in a scenario with no
// routed networks: nothing there can carry a request to another network, so
// the scope or relay would load and never answer.
func (v *Validator) validateDHCPRelayNeedsNetworks(cfg *Config) {
	check := func(device *Device, prefix string) {
		if device.DHCPConfig != nil && len(device.DHCPConfig.Scopes) > 0 {
			v.addError(prefix+".dhcp.scopes", "relayed DHCP scopes need routed networks")
		}
		for i, iface := range device.Interfaces {
			if iface.DHCPRelay != "" {
				v.addError(
					fmt.Sprintf("%s.interfaces[%d].dhcp_relay", prefix, i),
					"a DHCP relay needs routed networks",
				)
			}
		}
	}
	for i := range cfg.Devices {
		check(&cfg.Devices[i], fmt.Sprintf("devices[%d]", i))
	}
	for i := range cfg.Segments {
		for j := range cfg.Segments[i].Devices {
			check(&cfg.Segments[i].Devices[j], fmt.Sprintf("segments[%d].devices[%d]", i, j))
		}
	}
}
