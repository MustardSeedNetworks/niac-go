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
