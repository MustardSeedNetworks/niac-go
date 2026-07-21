package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

func parseOSFingerprintConfig(yamlOSFP *converter.OSFingerprintConfig) *OSFingerprintConfig {
	if yamlOSFP == nil {
		return nil
	}

	osFP := &OSFingerprintConfig{
		OSType:       yamlOSFP.OSType,
		TTL:          yamlOSFP.TTL,
		WindowSize:   yamlOSFP.WindowSize,
		WindowScale:  yamlOSFP.WindowScale,
		MSS:          yamlOSFP.MSS,
		SSHBanner:    yamlOSFP.SSHBanner,
		HTTPServer:   yamlOSFP.HTTPServer,
		FTPBanner:    yamlOSFP.FTPBanner,
		SMTPBanner:   yamlOSFP.SMTPBanner,
		TelnetBanner: yamlOSFP.TelnetBanner,
		DontFragment: yamlOSFP.DontFragment,
	}

	// Apply OS type defaults if no specific values are set
	if osFP.OSType != "" && osFP.TTL == 0 {
		switch osFP.OSType {
		case "linux", "macos", "freebsd", "openbsd":
			osFP.TTL = 64
		case "windows", "windows-server":
			osFP.TTL = 128
		case "cisco-ios", "cisco-nxos", "juniper-junos", "arista-eos":
			osFP.TTL = 255
		default:
			osFP.TTL = 64 // Default to Linux-like
		}
	}

	// Set default window size based on OS type if not specified
	if osFP.OSType != "" && osFP.WindowSize == 0 {
		switch osFP.OSType {
		case "linux":
			osFP.WindowSize = 29200
		case "windows", "windows-server":
			osFP.WindowSize = 65535
		case "macos":
			osFP.WindowSize = 65535
		default:
			osFP.WindowSize = 65535
		}
	}

	return osFP
}

// parseReflectorConfig parses NetAlly UDP reflector configuration from YAML.
// A non-nil result enables the reflector on the device.
func parseReflectorConfig(yamlReflector *converter.ReflectorConfig) *ReflectorConfig {
	if yamlReflector == nil {
		return nil
	}

	return &ReflectorConfig{
		LatencyMs: yamlReflector.LatencyMs,
		JitterMs:  yamlReflector.JitterMs,
		DSCP:      yamlReflector.DSCP,
	}
}

// parseIPerf3Config parses iPerf3 server emulation configuration from YAML.
func parseIPerf3Config(yamlIPerf3 *converter.IPerf3Config) *IPerf3Config {
	if yamlIPerf3 == nil {
		return nil
	}

	cfg := &IPerf3Config{
		Enabled:           yamlIPerf3.Enabled,
		Port:              yamlIPerf3.Port,
		MaxBandwidthMbps:  yamlIPerf3.MaxBandwidthMbps,
		TypicalLatencyMs:  yamlIPerf3.TypicalLatencyMs,
		JitterMs:          yamlIPerf3.JitterMs,
		PacketLossPercent: yamlIPerf3.PacketLossPercent,
		UploadMbps:        yamlIPerf3.UploadMbps,
		DownloadMbps:      yamlIPerf3.DownloadMbps,
	}

	// Set defaults for unspecified values
	if cfg.Port == 0 {
		cfg.Port = 5201 // Default iPerf3 port
	}

	if cfg.UploadMbps == 0 {
		cfg.UploadMbps = 100.0 // Default 100 Mbps
	}

	if cfg.DownloadMbps == 0 {
		cfg.DownloadMbps = 100.0 // Default 100 Mbps
	}

	if cfg.MaxBandwidthMbps == 0 {
		cfg.MaxBandwidthMbps = 1000.0 // Default 1 Gbps max
	}

	if cfg.TypicalLatencyMs == 0 {
		cfg.TypicalLatencyMs = 1.0 // Default 1ms
	}

	return cfg
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

	applyNetBIOSDefaults(netbiosCfg, deviceName)
	netbiosCfg.Names = parseNetBIOSNames(yamlNetbios.Names)

	return netbiosCfg
}

// applyNetBIOSDefaults fills in default values for NetBIOS config fields.
func applyNetBIOSDefaults(cfg *NetBIOSConfig, deviceName string) {
	if cfg.Name == "" {
		cfg.Name = deviceName
		if len(cfg.Name) > netbiosMaxNameLen {
			cfg.Name = cfg.Name[:netbiosMaxNameLen]
		}
	}

	if cfg.Workgroup == "" {
		cfg.Workgroup = "WORKGROUP"
	}

	if cfg.NodeType == "" {
		cfg.NodeType = "B"
	}

	if len(cfg.Services) == 0 {
		cfg.Services = []string{"workstation", "fileserver"}
	}

	if cfg.TTL == 0 {
		cfg.TTL = DefaultNetBIOSTTL
	}
}

// parseNetBIOSNames parses explicit NetBIOS name entries from YAML config.
func parseNetBIOSNames(yamlNames []converter.NetbiosName) []NetBIOSName {
	var names []NetBIOSName

	for _, name := range yamlNames {
		entry := NetBIOSName{
			Name:  name.Name,
			Group: name.Group,
		}

		if name.Suffix != "" {
			if strings.HasPrefix(strings.ToLower(name.Suffix), "0x") {
				if v, err := strconv.ParseUint(name.Suffix[2:], 16, 8); err == nil {
					entry.Suffix = uint8(v)
				}
			} else if v, err := strconv.ParseUint(name.Suffix, 10, 8); err == nil {
				entry.Suffix = uint8(v)
			}
		}

		names = append(names, entry)
	}

	return names
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

	if dhcpv6Cfg.PreferredLifetime == 0 {
		dhcpv6Cfg.PreferredLifetime = DefaultDHCPv6PreferredLifetime
	}

	if dhcpv6Cfg.ValidLifetime == 0 {
		dhcpv6Cfg.ValidLifetime = DefaultDHCPv6ValidLifetime
	}

	for _, pool := range yamlDhcpv6.Pools {
		dhcpv6Cfg.Pools = append(dhcpv6Cfg.Pools, DHCPv6Pool{
			Network: pool.Network, RangeStart: pool.RangeStart, RangeEnd: pool.RangeEnd,
		})
	}

	dhcpv6Cfg.DNSServers = parseIPList(yamlDhcpv6.DNSServers)
	dhcpv6Cfg.SNTPServers = parseIPList(yamlDhcpv6.SNTPServers)
	dhcpv6Cfg.NTPServers = parseIPList(yamlDhcpv6.NTPServers)
	dhcpv6Cfg.SIPServers = parseIPList(yamlDhcpv6.SIPServers)

	return dhcpv6Cfg
}

// parseIPList parses a list of IP address strings into [net.IP] values, skipping invalid entries.
func parseIPList(ipStrings []string) []net.IP {
	var ips []net.IP

	for _, s := range ipStrings {
		if ip := net.ParseIP(s); ip != nil {
			ips = append(ips, ip)
		}
	}

	return ips
}

// parseSNMPTrapsConfig parses SNMP traps configuration from YAML.
func parseSNMPTrapsConfig(yamlTraps *converter.TrapsConfig) *TrapConfig {
	trapsCfg := &TrapConfig{
		Enabled:   yamlTraps.Enabled,
		Receivers: yamlTraps.Receivers,
		Community: yamlTraps.Community,
	}

	parseSNMPTriggerTraps(trapsCfg, yamlTraps)
	parseSNMPThresholdTraps(trapsCfg, yamlTraps)

	return trapsCfg
}

// parseSNMPTriggerTraps parses ColdStart, LinkState, and AuthenticationFailure trap configs.
func parseSNMPTriggerTraps(trapsCfg *TrapConfig, yamlTraps *converter.TrapsConfig) {
	if yamlTraps.ColdStart != nil {
		trapsCfg.ColdStart = &TrapTriggerConfig{
			Enabled: yamlTraps.ColdStart.Enabled, OnStartup: yamlTraps.ColdStart.OnStartup,
		}
	}

	if yamlTraps.LinkState != nil {
		trapsCfg.LinkState = &LinkStateTrapConfig{
			Enabled:  yamlTraps.LinkState.Enabled,
			LinkDown: yamlTraps.LinkState.LinkDown,
			LinkUp:   yamlTraps.LinkState.LinkUp,
		}
	}

	if yamlTraps.AuthenticationFailure != nil {
		trapsCfg.AuthenticationFailure = &TrapTriggerConfig{
			Enabled: yamlTraps.AuthenticationFailure.Enabled, OnStartup: yamlTraps.AuthenticationFailure.OnStartup,
		}
	}
}

// parseSNMPThresholdTraps parses HighCPU, HighMemory, and InterfaceErrors trap configs.
func parseSNMPThresholdTraps(trapsCfg *TrapConfig, yamlTraps *converter.TrapsConfig) {
	if yamlTraps.HighCPU != nil {
		trapsCfg.HighCPU = parseThresholdTrap(
			yamlTraps.HighCPU, DefaultHighCPUThreshold, DefaultTrapCheckInterval,
		)
	}

	if yamlTraps.HighMemory != nil {
		trapsCfg.HighMemory = parseThresholdTrap(
			yamlTraps.HighMemory, DefaultHighMemoryThreshold, DefaultTrapCheckInterval,
		)
	}

	if yamlTraps.InterfaceErrors != nil {
		trapsCfg.InterfaceErrors = parseThresholdTrap(
			yamlTraps.InterfaceErrors, DefaultInterfaceErrorThreshold, DefaultInterfaceErrorInterval,
		)
	}
}

// parseThresholdTrap parses a threshold-based trap config with defaults.
func parseThresholdTrap(
	yaml *converter.ThresholdTrapConfig,
	defaultThreshold, defaultInterval int,
) *ThresholdTrapConfig {
	cfg := &ThresholdTrapConfig{
		Enabled: yaml.Enabled, Threshold: yaml.Threshold, Interval: yaml.Interval,
	}

	if cfg.Threshold == 0 {
		cfg.Threshold = defaultThreshold
	}

	if cfg.Interval == 0 {
		cfg.Interval = defaultInterval
	}

	return cfg
}

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

// validateDNSTTL validates and returns the DNS TTL value.
func validateDNSTTL(ttl int, deviceName string) (uint32, error) {
	if ttl <= 0 {
		return uint32(DefaultDNSTTL), nil
	}

	if ttl < 0 {
		return 0, fmt.Errorf("device %s: %w: %d", deviceName, ErrDNSTTLNegative, ttl)
	}

	if ttl > maxDNSTTL {
		return 0, fmt.Errorf("device %s: %w: %d", deviceName, ErrDNSTTLExceedsMax, ttl)
	}

	return uint32(ttl), nil
}

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

// parseSNMPv3Config parses SNMPv3 USM user configuration from YAML.
// Returns nil for absent input. NOT license-gated — SNMPv3 is free
// for all tiers (the only safe SNMP version).
func parseSNMPv3Config(yamlV3 *converter.Snmpv3Config) *SNMPv3Config {
	if yamlV3 == nil {
		return nil
	}

	cfg := &SNMPv3Config{
		Enabled:  yamlV3.Enabled,
		EngineID: yamlV3.EngineID,
		Users:    make([]SNMPv3User, 0, len(yamlV3.Users)),
	}
	for _, u := range yamlV3.Users {
		cfg.Users = append(cfg.Users, SNMPv3User{
			Username:     u.Username,
			AuthProtocol: u.AuthProtocol,
			AuthPassword: u.AuthPassword,
			PrivProtocol: u.PrivProtocol,
			PrivPassword: u.PrivPassword,
		})
	}

	return cfg
}

// ParseSimpleConfig parses a simple device configuration format
