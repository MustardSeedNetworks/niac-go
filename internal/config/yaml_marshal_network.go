package config

import (
	"reflect"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

func icmpToYAML(cfg *ICMPConfig) *converter.IcmpConfig {
	if cfg == nil {
		return nil
	}
	out := &converter.IcmpConfig{
		Enabled: cfg.Enabled, TTL: cfg.TTL, RateLimit: cfg.RateLimit,
		AddressMaskReply: ipString(cfg.AddressMaskReply),
	}
	if cfg.RouterAdvertisement != nil {
		out.RouterAdvertisement = &converter.IcmpRouterAdvertisement{
			Period: cfg.RouterAdvertisement.Period, Lifetime: cfg.RouterAdvertisement.Lifetime,
		}
		for _, router := range cfg.RouterAdvertisement.Routers {
			out.RouterAdvertisement.Routers = append(
				out.RouterAdvertisement.Routers,
				converter.IcmpRouter{
					Address:    ipString(router.Address),
					Preference: router.Preference,
				},
			)
		}
	}
	return out
}

func icmpv6ToYAML(cfg *ICMPv6Config) *converter.Icmpv6Config {
	if cfg == nil {
		return nil
	}
	out := &converter.Icmpv6Config{
		Enabled:   cfg.Enabled,
		HopLimit:  cfg.HopLimit,
		RateLimit: cfg.RateLimit,
	}
	if cfg.RouterAdvertisement != nil {
		ra := cfg.RouterAdvertisement
		out.RouterAdvertisement = &converter.Icmpv6RouterAdvertisement{
			Period: ra.Period, CurHopLimit: ra.CurHopLimit, Managed: ra.Managed, Other: ra.Other,
			Lifetime: ra.Lifetime, ReachableTime: ra.ReachableTime,
			RetransTimer: ra.RetransTimer, MTU: ra.MTU,
		}
		for _, prefix := range ra.PrefixInfo {
			out.RouterAdvertisement.PrefixInfo = append(
				out.RouterAdvertisement.PrefixInfo,
				converter.Icmpv6PrefixInfo{
					PrefixLength: prefix.PrefixLength, Onlink: prefix.Onlink, Auto: prefix.Auto,
					ValidLifetime: prefix.ValidLifetime, PreferredLifetime: prefix.PreferredLifetime,
					Prefix: ipString(prefix.Prefix),
				},
			)
		}
	}
	return out
}

func dhcpv6ToYAML(cfg *DHCPv6Config) *converter.Dhcpv6Config {
	if cfg == nil {
		return nil
	}
	// A device with no `dhcpv6` block loads as a zero-value config rather than
	// nil (parseDHCPv6Config, alone among the protocol parsers). Emitting that
	// as `dhcpv6: {}` makes the block present, and a present block picks up
	// the lifetime defaults on the next load — so one save through the device
	// editor gave every device a DHCPv6 server it never authored.
	if reflect.DeepEqual(cfg, &DHCPv6Config{}) {
		return nil
	}
	out := &converter.Dhcpv6Config{
		Enabled: cfg.Enabled, PreferredLifetime: cfg.PreferredLifetime,
		ValidLifetime: cfg.ValidLifetime, Preference: cfg.Preference,
		DNSServers: ipStrings(cfg.DNSServers), DomainList: cfg.DomainList,
		SNTPServers: ipStrings(cfg.SNTPServers), NTPServers: ipStrings(cfg.NTPServers),
		SIPServers: ipStrings(cfg.SIPServers), SIPDomains: cfg.SIPDomains,
	}
	for _, pool := range cfg.Pools {
		out.Pools = append(out.Pools, converter.Dhcpv6Pool(pool))
	}
	return out
}
