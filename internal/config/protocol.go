package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/krisarmstrong/niac-go/internal/converter"
)

// parseLLDPConfig parses LLDP configuration from YAML.
func parseLLDPConfig(yamlLldp *converter.LldpConfig) *LLDPConfig {
	if yamlLldp == nil {
		return nil
	}

	lldpCfg := &LLDPConfig{
		Enabled:           yamlLldp.Enabled,
		AdvertiseInterval: yamlLldp.AdvertiseInterval,
		TTL:               yamlLldp.TTL,
		SystemDescription: yamlLldp.SystemDescription,
		PortDescription:   yamlLldp.PortDescription,
		ChassisIDType:     yamlLldp.ChassisIDType,
	}
	// Set defaults if not specified
	if lldpCfg.AdvertiseInterval == 0 {
		lldpCfg.AdvertiseInterval = DefaultLLDPAdvertiseInterval
	}

	if lldpCfg.TTL == 0 {
		lldpCfg.TTL = DefaultLLDPTTL
	}

	if lldpCfg.ChassisIDType == "" {
		lldpCfg.ChassisIDType = ChassisIDTypeMAC
	}

	return lldpCfg
}

// parseCDPConfig parses CDP configuration from YAML.
func parseCDPConfig(yamlCdp *converter.CdpConfig) *CDPConfig {
	if yamlCdp == nil {
		return nil
	}

	cdpCfg := &CDPConfig{
		Enabled:           yamlCdp.Enabled,
		AdvertiseInterval: yamlCdp.AdvertiseInterval,
		Holdtime:          yamlCdp.Holdtime,
		Version:           yamlCdp.Version,
		SoftwareVersion:   yamlCdp.SoftwareVersion,
		Platform:          yamlCdp.Platform,
		PortID:            yamlCdp.PortID,
	}
	// Set defaults if not specified
	if cdpCfg.AdvertiseInterval == 0 {
		cdpCfg.AdvertiseInterval = DefaultCDPAdvertiseInterval
	}

	if cdpCfg.Holdtime == 0 {
		cdpCfg.Holdtime = DefaultCDPHoldtime
	}

	if cdpCfg.Version == 0 {
		cdpCfg.Version = DefaultCDPVersion
	}

	return cdpCfg
}

// parseEDPConfig parses EDP configuration from YAML.
func parseEDPConfig(yamlEdp *converter.EdpConfig) *EDPConfig {
	if yamlEdp == nil {
		return nil
	}

	edpCfg := &EDPConfig{
		Enabled:           yamlEdp.Enabled,
		AdvertiseInterval: yamlEdp.AdvertiseInterval,
		VersionString:     yamlEdp.VersionString,
		DisplayString:     yamlEdp.DisplayString,
	}
	// Set defaults if not specified
	if edpCfg.AdvertiseInterval == 0 {
		edpCfg.AdvertiseInterval = DefaultEDPAdvertiseInterval
	}

	return edpCfg
}

// parseFDPConfig parses FDP configuration from YAML.
func parseFDPConfig(yamlFdp *converter.FdpConfig) *FDPConfig {
	if yamlFdp == nil {
		return nil
	}

	fdpCfg := &FDPConfig{
		Enabled:           yamlFdp.Enabled,
		AdvertiseInterval: yamlFdp.AdvertiseInterval,
		Holdtime:          yamlFdp.Holdtime,
		SoftwareVersion:   yamlFdp.SoftwareVersion,
		Platform:          yamlFdp.Platform,
		PortID:            yamlFdp.PortID,
	}
	// Set defaults if not specified
	if fdpCfg.AdvertiseInterval == 0 {
		fdpCfg.AdvertiseInterval = DefaultFDPAdvertiseInterval
	}

	if fdpCfg.Holdtime == 0 {
		fdpCfg.Holdtime = DefaultFDPHoldtime
	}

	return fdpCfg
}

// parseSTPConfig parses STP configuration from YAML.
func parseSTPConfig(yamlStp *converter.StpConfig) *STPConfig {
	if yamlStp == nil {
		return nil
	}

	stpCfg := &STPConfig{
		Enabled:        yamlStp.Enabled,
		BridgePriority: yamlStp.BridgePriority,
		HelloTime:      yamlStp.HelloTime,
		MaxAge:         yamlStp.MaxAge,
		ForwardDelay:   yamlStp.ForwardDelay,
		Version:        yamlStp.Version,
	}
	// Set defaults if not specified
	if stpCfg.BridgePriority == 0 {
		stpCfg.BridgePriority = DefaultSTPBridgePriority
	}

	if stpCfg.HelloTime == 0 {
		stpCfg.HelloTime = DefaultSTPHelloTime
	}

	if stpCfg.MaxAge == 0 {
		stpCfg.MaxAge = DefaultSTPMaxAge
	}

	if stpCfg.ForwardDelay == 0 {
		stpCfg.ForwardDelay = DefaultSTPForwardDelay
	}

	if stpCfg.Version == "" {
		stpCfg.Version = "stp" // Default to STP
	}

	return stpCfg
}

// parseHTTPConfig parses HTTP configuration from YAML.
func parseHTTPConfig(yamlHTTP *converter.HTTPConfig, _ string) *HTTPConfig {
	if yamlHTTP == nil {
		return nil
	}

	httpCfg := &HTTPConfig{
		Enabled:    yamlHTTP.Enabled,
		ServerName: yamlHTTP.ServerName,
		Endpoints:  make([]HTTPEndpoint, 0),
	}
	// Set default server name if not specified
	if httpCfg.ServerName == "" {
		httpCfg.ServerName = "NIAC-Go/1.0.0"
	}
	// Parse endpoints
	for _, ep := range yamlHTTP.Endpoints {
		endpoint := HTTPEndpoint{
			Path:        ep.Path,
			Method:      ep.Method,
			StatusCode:  ep.StatusCode,
			ContentType: ep.ContentType,
			Body:        ep.Body,
		}
		// Set defaults
		if endpoint.Method == "" {
			endpoint.Method = "GET"
		}

		if endpoint.StatusCode == 0 {
			endpoint.StatusCode = 200
		}

		if endpoint.ContentType == "" {
			endpoint.ContentType = "text/html"
		}

		httpCfg.Endpoints = append(httpCfg.Endpoints, endpoint)
	}

	return httpCfg
}

// parseFTPConfig parses FTP configuration from YAML.
func parseFTPConfig(yamlFtp *converter.FtpConfig, deviceName string) *FTPConfig {
	if yamlFtp == nil {
		return nil
	}

	ftpCfg := &FTPConfig{
		Enabled:        yamlFtp.Enabled,
		WelcomeBanner:  yamlFtp.WelcomeBanner,
		SystemType:     yamlFtp.SystemType,
		AllowAnonymous: yamlFtp.AllowAnonymous,
		Users:          make([]FTPUser, 0),
	}
	// Set defaults
	if ftpCfg.WelcomeBanner == "" {
		ftpCfg.WelcomeBanner = fmt.Sprintf("220 %s FTP Server (NIAC-Go) ready.", deviceName)
	}

	if ftpCfg.SystemType == "" {
		ftpCfg.SystemType = "UNIX Type: L8"
	}
	// Parse users
	for _, u := range yamlFtp.Users {
		user := FTPUser{
			Username: u.Username,
			Password: u.Password,
			HomeDir:  u.HomeDir,
		}
		if user.HomeDir == "" {
			user.HomeDir = "/"
		}

		ftpCfg.Users = append(ftpCfg.Users, user)
	}

	return ftpCfg
}

// parseNetBIOSConfig parses NetBIOS configuration from YAML.
func parseNetBIOSConfig(yamlNetbios *converter.NetbiosConfig, deviceName string) *NetBIOSConfig {
	if yamlNetbios == nil {
		return nil
	}

	netbiosCfg := &NetBIOSConfig{
		Enabled:   yamlNetbios.Enabled,
		Name:      yamlNetbios.Name,
		Workgroup: yamlNetbios.Workgroup,
		NodeType:  yamlNetbios.NodeType,
		Services:  yamlNetbios.Services,
		TTL:       yamlNetbios.TTL,
		MsBrowse:  yamlNetbios.MsBrowse,
	}

	// Set defaults
	if netbiosCfg.Name == "" {
		netbiosCfg.Name = deviceName
		if len(netbiosCfg.Name) > netbiosMaxNameLen {
			netbiosCfg.Name = netbiosCfg.Name[:netbiosMaxNameLen]
		}
	}

	if netbiosCfg.Workgroup == "" {
		netbiosCfg.Workgroup = "WORKGROUP"
	}

	if netbiosCfg.NodeType == "" {
		netbiosCfg.NodeType = "B"
	}

	if len(netbiosCfg.Services) == 0 {
		netbiosCfg.Services = []string{"workstation", "fileserver"}
	}

	if netbiosCfg.TTL == 0 {
		netbiosCfg.TTL = DefaultNetBIOSTTL
	}

	// Parse explicit NetBIOS names
	for _, name := range yamlNetbios.Names {
		entry := NetBIOSName{
			Name:  name.Name,
			Group: name.Group,
		}

		if name.Suffix != "" {
			// Parse suffix as hex (0x..) or decimal
			if strings.HasPrefix(strings.ToLower(name.Suffix), "0x") {
				if v, err := strconv.ParseUint(name.Suffix[2:], 16, 8); err == nil {
					entry.Suffix = uint8(v)
				}
			} else if v, err := strconv.ParseUint(name.Suffix, 10, 8); err == nil {
				entry.Suffix = uint8(v)
			}
		}

		netbiosCfg.Names = append(netbiosCfg.Names, entry)
	}

	return netbiosCfg
}

// parseICMPConfig parses ICMP configuration from YAML.
func parseICMPConfig(yamlIcmp *converter.IcmpConfig) *ICMPConfig {
	if yamlIcmp == nil {
		return nil
	}

	icmpCfg := &ICMPConfig{
		Enabled:   yamlIcmp.Enabled,
		TTL:       yamlIcmp.TTL,
		RateLimit: yamlIcmp.RateLimit,
	}

	if icmpCfg.TTL == 0 {
		icmpCfg.TTL = DefaultICMPTTL
	}

	if yamlIcmp.AddressMaskReply != "" {
		if ip := net.ParseIP(yamlIcmp.AddressMaskReply); ip != nil {
			icmpCfg.AddressMaskReply = ip.To4()
		}
	}

	if yamlIcmp.RouterAdvertisement != nil {
		ra := &IcmpRouterAdvertisement{
			Period:   yamlIcmp.RouterAdvertisement.Period,
			Lifetime: yamlIcmp.RouterAdvertisement.Lifetime,
		}

		for _, r := range yamlIcmp.RouterAdvertisement.Routers {
			if ip := net.ParseIP(r.Address); ip != nil {
				ra.Routers = append(ra.Routers, IcmpRouter{
					Address:    ip.To4(),
					Preference: r.Preference,
				})
			}
		}

		icmpCfg.RouterAdvertisement = ra
	}

	return icmpCfg
}

// parseICMPv6Config parses ICMPv6 configuration from YAML.
func parseICMPv6Config(yamlIcmpv6 *converter.Icmpv6Config) *ICMPv6Config {
	if yamlIcmpv6 == nil {
		return nil
	}

	icmpv6Cfg := &ICMPv6Config{
		Enabled:   yamlIcmpv6.Enabled,
		HopLimit:  yamlIcmpv6.HopLimit,
		RateLimit: yamlIcmpv6.RateLimit,
	}

	if icmpv6Cfg.HopLimit == 0 {
		icmpv6Cfg.HopLimit = DefaultICMPv6HopLimit
	}

	if yamlIcmpv6.RouterAdvertisement != nil {
		ra := &Icmpv6RouterAdvertisement{
			Period:        yamlIcmpv6.RouterAdvertisement.Period,
			CurHopLimit:   yamlIcmpv6.RouterAdvertisement.CurHopLimit,
			Managed:       yamlIcmpv6.RouterAdvertisement.Managed,
			Other:         yamlIcmpv6.RouterAdvertisement.Other,
			Lifetime:      yamlIcmpv6.RouterAdvertisement.Lifetime,
			ReachableTime: yamlIcmpv6.RouterAdvertisement.ReachableTime,
			RetransTimer:  yamlIcmpv6.RouterAdvertisement.RetransTimer,
			MTU:           yamlIcmpv6.RouterAdvertisement.MTU,
		}

		for _, p := range yamlIcmpv6.RouterAdvertisement.PrefixInfo {
			var prefix net.IP
			if p.Prefix != "" {
				prefix = net.ParseIP(p.Prefix)
			}

			ra.PrefixInfo = append(ra.PrefixInfo, Icmpv6PrefixInfo{
				PrefixLength:      p.PrefixLength,
				Onlink:            p.Onlink,
				Auto:              p.Auto,
				ValidLifetime:     p.ValidLifetime,
				PreferredLifetime: p.PreferredLifetime,
				Prefix:            prefix,
			})
		}

		icmpv6Cfg.RouterAdvertisement = ra
	}

	return icmpv6Cfg
}

// parseDHCPv6Config parses DHCPv6 configuration from YAML.
// Returns an empty DHCPv6Config if input is nil (not an error condition).
func parseDHCPv6Config(yamlDhcpv6 *converter.Dhcpv6Config) *DHCPv6Config {
	if yamlDhcpv6 == nil {
		return &DHCPv6Config{}
	}

	dhcpv6Cfg := &DHCPv6Config{
		Enabled:           yamlDhcpv6.Enabled,
		Pools:             make([]DHCPv6Pool, 0),
		PreferredLifetime: yamlDhcpv6.PreferredLifetime,
		ValidLifetime:     yamlDhcpv6.ValidLifetime,
		Preference:        yamlDhcpv6.Preference,
		DomainList:        yamlDhcpv6.DomainList,
		SIPDomains:        yamlDhcpv6.SIPDomains,
	}

	// Set defaults
	if dhcpv6Cfg.PreferredLifetime == 0 {
		dhcpv6Cfg.PreferredLifetime = DefaultDHCPv6PreferredLifetime
	}

	if dhcpv6Cfg.ValidLifetime == 0 {
		dhcpv6Cfg.ValidLifetime = DefaultDHCPv6ValidLifetime
	}

	// Parse address pools
	for _, pool := range yamlDhcpv6.Pools {
		dhcpv6Cfg.Pools = append(dhcpv6Cfg.Pools, DHCPv6Pool{
			Network:    pool.Network,
			RangeStart: pool.RangeStart,
			RangeEnd:   pool.RangeEnd,
		})
	}

	// Parse DNS servers
	for _, dnsStr := range yamlDhcpv6.DNSServers {
		if ip := net.ParseIP(dnsStr); ip != nil {
			dhcpv6Cfg.DNSServers = append(dhcpv6Cfg.DNSServers, ip)
		}
	}

	// Parse SNTP servers
	for _, sntpStr := range yamlDhcpv6.SNTPServers {
		if ip := net.ParseIP(sntpStr); ip != nil {
			dhcpv6Cfg.SNTPServers = append(dhcpv6Cfg.SNTPServers, ip)
		}
	}

	// Parse NTP servers
	for _, ntpStr := range yamlDhcpv6.NTPServers {
		if ip := net.ParseIP(ntpStr); ip != nil {
			dhcpv6Cfg.NTPServers = append(dhcpv6Cfg.NTPServers, ip)
		}
	}

	// Parse SIP servers
	for _, sipStr := range yamlDhcpv6.SIPServers {
		if ip := net.ParseIP(sipStr); ip != nil {
			dhcpv6Cfg.SIPServers = append(dhcpv6Cfg.SIPServers, ip)
		}
	}

	return dhcpv6Cfg
}
