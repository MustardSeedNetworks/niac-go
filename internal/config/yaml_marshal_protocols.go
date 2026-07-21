package config

import (
	"net"
	"strconv"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

func ttlToYAML(cfg *TTLConfig) *converter.TTLConfig {
	if cfg == nil {
		return nil
	}
	return &converter.TTLConfig{
		TTL:  cfg.TTL,
		IP:   ipString(cfg.IP),
		Mask: ipString(net.IP(cfg.Mask)),
	}
}

func snmpToYAML(cfg *SNMPConfig) *converter.SnmpAgent {
	if cfg == nil {
		return nil
	}
	out := &converter.SnmpAgent{
		Enabled: cfg.Enabled, Community: cfg.Community, SysName: cfg.SysName,
		SysDescr: cfg.SysDescr, SysContact: cfg.SysContact, SysLocation: cfg.SysLocation,
		WalkFile: cfg.WalkFile, WalkFiles: cfg.WalkFiles, AccessList: ipStrings(cfg.AccessList),
		SnmpAddr: ipString(cfg.SnmpAddr), Dot1DFdbTable: fdbToYAML(cfg.Dot1DFdbTable),
		Dot1QFdbTable: fdbToYAML(cfg.Dot1QFdbTable), Traps: trapsToYAML(cfg.Traps),
	}
	for _, mib := range cfg.AddMibs {
		out.AddMibs = append(
			out.AddMibs,
			converter.AddMib{OID: mib.OID, Type: mib.Type, Value: mib.Value},
		)
	}
	for _, include := range cfg.CommunityIncludes {
		out.CommunityIncludes = append(out.CommunityIncludes, converter.CommunityInclude{
			Community: include.Community, WalkFile: include.WalkFile,
		})
	}
	return out
}

func fdbToYAML(cfg *FdbTableConfig) *converter.FdbTableConfig {
	if cfg == nil {
		return nil
	}
	return &converter.FdbTableConfig{Port: cfg.Port, VLAN: cfg.VLAN}
}

func trapsToYAML(cfg *TrapConfig) *converter.TrapsConfig {
	if cfg == nil {
		return nil
	}
	return &converter.TrapsConfig{
		Enabled: cfg.Enabled, Receivers: cfg.Receivers, Community: cfg.Community,
		ColdStart: triggerToYAML(cfg.ColdStart), LinkState: linkStateToYAML(cfg.LinkState),
		AuthenticationFailure: triggerToYAML(cfg.AuthenticationFailure),
		HighCPU: thresholdToYAML(
			cfg.HighCPU,
		), HighMemory: thresholdToYAML(cfg.HighMemory),
		InterfaceErrors: thresholdToYAML(cfg.InterfaceErrors),
	}
}

func triggerToYAML(cfg *TrapTriggerConfig) *converter.TrapTriggerConfig {
	if cfg == nil {
		return nil
	}
	return &converter.TrapTriggerConfig{Enabled: cfg.Enabled, OnStartup: cfg.OnStartup}
}

func linkStateToYAML(cfg *LinkStateTrapConfig) *converter.LinkStateTrapConfig {
	if cfg == nil {
		return nil
	}
	return &converter.LinkStateTrapConfig{
		Enabled:  cfg.Enabled,
		LinkDown: cfg.LinkDown,
		LinkUp:   cfg.LinkUp,
	}
}

func thresholdToYAML(cfg *ThresholdTrapConfig) *converter.ThresholdTrapConfig {
	if cfg == nil {
		return nil
	}
	return &converter.ThresholdTrapConfig{
		Enabled: cfg.Enabled, Threshold: cfg.Threshold, Interval: cfg.Interval,
	}
}

func dhcpToYAML(cfg *DHCPConfig) *converter.DhcpServer {
	if cfg == nil {
		return nil
	}
	out := &converter.DhcpServer{
		SubnetMask: ipString(net.IP(cfg.SubnetMask)), Router: ipString(cfg.Router),
		DomainNameServer: firstIPString(
			cfg.DomainNameServer,
		), NextServerIP: ipString(cfg.NextServerIP),
		ServerIdentifier: ipString(cfg.ServerIdentifier), PoolStart: ipString(cfg.PoolStart),
		PoolEnd: ipString(cfg.PoolEnd), NTPServers: ipStrings(cfg.NTPServers),
		DomainSearch: cfg.DomainSearch, TFTPServerName: cfg.TFTPServerName,
		BootfileName: cfg.BootfileName, VendorSpecific: string(cfg.VendorSpecific),
		SNTPServersV6: ipStrings(cfg.SNTPServersV6), NTPServersV6: ipStrings(cfg.NTPServersV6),
		SIPServersV6: ipStrings(cfg.SIPServersV6), SIPDomainsV6: cfg.SIPDomainsV6,
	}
	for _, lease := range cfg.ClientLeases {
		out.ClientLeases = append(out.ClientLeases, converter.DhcpLease{
			ClientIP: ipString(lease.ClientIP), MacAddrValue: hardwareAddrString(lease.MACAddress),
			MacAddrMask: hardwareAddrString(lease.MACMask),
		})
	}
	return out
}

func firstIPString(ips []net.IP) string {
	if len(ips) == 0 {
		return ""
	}
	return ipString(ips[0])
}

func dnsToYAML(cfg *DNSConfig) *converter.DNSServer {
	if cfg == nil {
		return nil
	}
	return &converter.DNSServer{
		ForwardRecords: dnsRecordsToYAML(cfg.ForwardRecords),
		ReverseRecords: dnsRecordsToYAML(cfg.ReverseRecords),
	}
}

func dnsRecordsToYAML(records []DNSRecord) []converter.DNSRecord {
	out := make([]converter.DNSRecord, len(records))
	for i, record := range records {
		out[i] = converter.DNSRecord{
			Name: record.Name, IP: ipString(record.IP), TTL: int(record.TTL), RCode: record.RCode,
		}
	}
	return out
}

func lldpToYAML(cfg *LLDPConfig) *converter.LldpConfig {
	if cfg == nil {
		return nil
	}
	out := converter.LldpConfig(*cfg)
	return &out
}

func cdpToYAML(cfg *CDPConfig) *converter.CdpConfig {
	if cfg == nil {
		return nil
	}
	out := converter.CdpConfig(*cfg)
	return &out
}

func edpToYAML(cfg *EDPConfig) *converter.EdpConfig {
	if cfg == nil {
		return nil
	}
	out := converter.EdpConfig(*cfg)
	return &out
}

func fdpToYAML(cfg *FDPConfig) *converter.FdpConfig {
	if cfg == nil {
		return nil
	}
	out := converter.FdpConfig(*cfg)
	return &out
}

func stpToYAML(cfg *STPConfig) *converter.StpConfig {
	if cfg == nil {
		return nil
	}
	out := converter.StpConfig(*cfg)
	return &out
}

func httpToYAML(cfg *HTTPConfig) *converter.HTTPConfig {
	if cfg == nil {
		return nil
	}
	out := &converter.HTTPConfig{Enabled: cfg.Enabled, ServerName: cfg.ServerName}
	for _, endpoint := range cfg.Endpoints {
		out.Endpoints = append(out.Endpoints, converter.HTTPEndpoint(endpoint))
	}
	return out
}

func ftpToYAML(cfg *FTPConfig) *converter.FtpConfig {
	if cfg == nil {
		return nil
	}
	out := &converter.FtpConfig{
		Enabled: cfg.Enabled, WelcomeBanner: cfg.WelcomeBanner,
		SystemType: cfg.SystemType, AllowAnonymous: cfg.AllowAnonymous,
	}
	for _, user := range cfg.Users {
		out.Users = append(out.Users, converter.FtpUser(user))
	}
	return out
}

func netbiosToYAML(cfg *NetBIOSConfig) *converter.NetbiosConfig {
	if cfg == nil {
		return nil
	}
	out := &converter.NetbiosConfig{
		Enabled: cfg.Enabled, Name: cfg.Name, Workgroup: cfg.Workgroup,
		NodeType: cfg.NodeType, Services: cfg.Services, TTL: cfg.TTL, MsBrowse: cfg.MsBrowse,
	}
	for _, name := range cfg.Names {
		out.Names = append(out.Names, converter.NetbiosName{
			Name: name.Name, Suffix: strconv.Itoa(int(name.Suffix)), Group: name.Group,
		})
	}
	return out
}

func snmpv3ToYAML(cfg *SNMPv3Config) *converter.Snmpv3Config {
	if cfg == nil {
		return nil
	}
	out := &converter.Snmpv3Config{Enabled: cfg.Enabled, EngineID: cfg.EngineID}
	for _, user := range cfg.Users {
		out.Users = append(out.Users, converter.Snmpv3User(user))
	}
	return out
}

func osFingerprintToYAML(cfg *OSFingerprintConfig) *converter.OSFingerprintConfig {
	if cfg == nil {
		return nil
	}
	out := converter.OSFingerprintConfig(*cfg)
	return &out
}

func iperf3ToYAML(cfg *IPerf3Config) *converter.IPerf3Config {
	if cfg == nil {
		return nil
	}
	out := converter.IPerf3Config(*cfg)
	return &out
}

func reflectorToYAML(cfg *ReflectorConfig) *converter.ReflectorConfig {
	if cfg == nil {
		return nil
	}
	out := converter.ReflectorConfig(*cfg)
	return &out
}
