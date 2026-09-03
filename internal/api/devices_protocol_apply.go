package api

import (
	"encoding/hex"
	"fmt"
	"net"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// applyDeviceScalarFields applies the top-level scalar/list fields the UI's
// Device type carries alongside the protocol blocks. Zero-value fields are
// treated as "not set" (matching the Type/MAC/IP convention already used by
// applyPartialDeviceUpdate) except Babble, which needs a pointer to
// distinguish "leave it alone" from "turn it off" on a partial update.
func applyDeviceScalarFields(dev *config.Device, ips []string, vlan int, babble *bool, mapToIP string) error {
	if ips != nil {
		parsed, err := parseIPList(ips)
		if err != nil {
			return err
		}

		dev.IPAddresses = parsed
	}

	if vlan != 0 {
		dev.VLAN = vlan
	}

	if babble != nil {
		dev.Babble = *babble
	}

	if mapToIP != "" {
		ip, err := parseIP(mapToIP)
		if err != nil {
			return err
		}

		dev.MapToIP = ip
	}

	return nil
}

func parseIPList(ips []string) ([]net.IP, error) {
	parsed := make([]net.IP, 0, len(ips))
	for _, s := range ips {
		ip, err := parseIP(s)
		if err != nil {
			return nil, err
		}

		parsed = append(parsed, ip)
	}

	return parsed, nil
}

// applyDiscoveryProtocolRequests applies the discovery-protocol blocks
// (LLDP/CDP/EDP/FDP/STP). Each is replaced wholesale when present, the same
// whole-object-replace semantics as applyManagementRequests uses for
// SSH/Syslog.
func applyDiscoveryProtocolRequests(
	dev *config.Device, lldp *LLDPRequest, cdp *CDPRequest, edp *EDPRequest, fdp *FDPRequest, stp *STPRequest,
) {
	if lldp != nil {
		dev.LLDPConfig = &config.LLDPConfig{
			Enabled: lldp.Enabled, AdvertiseInterval: lldp.AdvertiseInterval, TTL: lldp.TTL,
			ChassisIDType: lldp.ChassisIDType, SystemDescription: lldp.SystemDescription,
			PortDescription: lldp.PortDescription,
		}
	}

	if cdp != nil {
		dev.CDPConfig = &config.CDPConfig{
			Enabled: cdp.Enabled, AdvertiseInterval: cdp.AdvertiseInterval, Holdtime: cdp.Holdtime,
			Version: cdp.Version, SoftwareVersion: cdp.SoftwareVersion, Platform: cdp.Platform, PortID: cdp.PortID,
		}
	}

	if edp != nil {
		dev.EDPConfig = &config.EDPConfig{
			Enabled: edp.Enabled, AdvertiseInterval: edp.AdvertiseInterval,
			VersionString: edp.VersionString, DisplayString: edp.DisplayString,
		}
	}

	if fdp != nil {
		dev.FDPConfig = &config.FDPConfig{
			Enabled: fdp.Enabled, AdvertiseInterval: fdp.AdvertiseInterval, Holdtime: fdp.Holdtime,
			SoftwareVersion: fdp.SoftwareVersion, Platform: fdp.Platform, PortID: fdp.PortID,
		}
	}

	if stp != nil {
		dev.STPConfig = &config.STPConfig{
			Enabled: stp.Enabled, BridgePriority: stp.BridgePriority, HelloTime: stp.HelloTime,
			MaxAge: stp.MaxAge, ForwardDelay: stp.ForwardDelay, Version: stp.Version,
		}
	}
}

// applyDHCPRequest converts a DHCPRequest into config.DHCPConfig. Enabled
// only gates whether the pointer is created — config.DHCPConfig has no
// Enabled field of its own, same as DNS/TTL/OSFingerprint below.
func applyDHCPRequest(dev *config.Device, req *DHCPRequest) error {
	if req == nil {
		return nil
	}

	if !req.Enabled {
		dev.DHCPConfig = nil
		return nil
	}

	cfg := &config.DHCPConfig{
		DomainName:     req.DomainName,
		DomainSearch:   append([]string(nil), req.DomainSearch...),
		TFTPServerName: req.TFTPServerName,
		BootfileName:   req.BootfileName,
	}

	if err := applyDHCPAddresses(cfg, req); err != nil {
		return err
	}

	if req.VendorSpecific != "" {
		raw, hexErr := hex.DecodeString(req.VendorSpecific)
		if hexErr != nil {
			return fmt.Errorf("invalid_vendor_specific: %w", hexErr)
		}

		cfg.VendorSpecific = raw
	}

	leases, err := dhcpLeasesToConfig(req.ClientLeases)
	if err != nil {
		return err
	}

	cfg.ClientLeases = leases
	dev.DHCPConfig = cfg

	return nil
}

// applyDHCPAddresses fills in the net.IP/net.IPMask fields of cfg from req.
// Split out of applyDHCPRequest to keep that function's branching readable.
func applyDHCPAddresses(cfg *config.DHCPConfig, req *DHCPRequest) error {
	var err error
	if cfg.SubnetMask, err = parseOptionalIPMask(req.SubnetMask); err != nil {
		return err
	}
	if cfg.Router, err = parseOptionalIP(req.Router); err != nil {
		return err
	}
	if cfg.ServerIdentifier, err = parseOptionalIP(req.ServerIdentifier); err != nil {
		return err
	}
	if cfg.NextServerIP, err = parseOptionalIP(req.NextServerIP); err != nil {
		return err
	}
	if cfg.PoolStart, err = parseOptionalIP(req.PoolStart); err != nil {
		return err
	}
	if cfg.PoolEnd, err = parseOptionalIP(req.PoolEnd); err != nil {
		return err
	}
	if cfg.NTPServers, err = parseIPList(req.NTPServers); err != nil {
		return err
	}

	if req.DomainNameServer != "" {
		dns, dnsErr := parseIP(req.DomainNameServer)
		if dnsErr != nil {
			return dnsErr
		}

		cfg.DomainNameServer = []net.IP{dns}
	}

	return nil
}

func dhcpLeasesToConfig(leases []DHCPLeaseRequest) ([]config.DHCPLease, error) {
	out := make([]config.DHCPLease, 0, len(leases))
	for _, lease := range leases {
		clientIP, ipErr := parseIP(lease.ClientIP)
		if ipErr != nil {
			return nil, ipErr
		}

		mac, macErr := parseMAC(lease.MACAddress)
		if macErr != nil {
			return nil, macErr
		}

		entry := config.DHCPLease{ClientIP: clientIP, MACAddress: mac}
		if lease.MACMask != "" {
			mask, maskErr := parseMAC(lease.MACMask)
			if maskErr != nil {
				return nil, maskErr
			}

			entry.MACMask = mask
		}

		out = append(out, entry)
	}

	return out, nil
}

func applyDHCPv6Request(dev *config.Device, req *DHCPv6Request) error {
	if req == nil {
		return nil
	}

	if !req.Enabled {
		dev.DHCPv6Config = nil
		return nil
	}

	cfg := &config.DHCPv6Config{
		PreferredLifetime: req.PreferredLifetime,
		ValidLifetime:     req.ValidLifetime,
		Preference:        req.Preference,
		DomainList:        append([]string(nil), req.DomainList...),
		SIPDomains:        append([]string(nil), req.SIPDomains...),
	}

	cfg.Pools = make([]config.DHCPv6Pool, 0, len(req.Pools))
	for _, pool := range req.Pools {
		cfg.Pools = append(cfg.Pools, config.DHCPv6Pool{
			Network: pool.Network, RangeStart: pool.RangeStart, RangeEnd: pool.RangeEnd,
		})
	}

	var err error
	if cfg.DNSServers, err = parseIPList(req.DNSServers); err != nil {
		return err
	}
	if cfg.SNTPServers, err = parseIPList(req.SNTPServers); err != nil {
		return err
	}
	if cfg.NTPServers, err = parseIPList(req.NTPServers); err != nil {
		return err
	}
	if cfg.SIPServers, err = parseIPList(req.SIPServers); err != nil {
		return err
	}

	dev.DHCPv6Config = cfg

	return nil
}

func applyDNSRequest(dev *config.Device, req *DNSRequest) error {
	if req == nil {
		return nil
	}

	if !req.Enabled {
		dev.DNSConfig = nil
		return nil
	}

	cfg := &config.DNSConfig{}

	var err error
	if cfg.ForwardRecords, err = dnsRecordsToConfig(req.ForwardRecords); err != nil {
		return err
	}
	if cfg.ReverseRecords, err = dnsRecordsToConfig(req.ReverseRecords); err != nil {
		return err
	}

	dev.DNSConfig = cfg

	return nil
}

func dnsRecordsToConfig(records []DNSRecordRequest) ([]config.DNSRecord, error) {
	out := make([]config.DNSRecord, 0, len(records))
	for _, rec := range records {
		ip, err := parseIP(rec.IP)
		if err != nil {
			return nil, err
		}

		out = append(out, config.DNSRecord{Name: rec.Name, IP: ip, TTL: rec.TTL, RCode: rec.RCode})
	}

	return out, nil
}

func applyHTTPRequest(dev *config.Device, req *HTTPRequest) {
	if req == nil {
		return
	}

	if !req.Enabled {
		dev.HTTPConfig = nil
		return
	}

	cfg := &config.HTTPConfig{Enabled: true, ServerName: req.ServerName}
	cfg.Endpoints = make([]config.HTTPEndpoint, 0, len(req.Endpoints))
	for _, ep := range req.Endpoints {
		cfg.Endpoints = append(cfg.Endpoints, config.HTTPEndpoint{
			Path: ep.Path, Method: ep.Method, StatusCode: ep.StatusCode, ContentType: ep.ContentType, Body: ep.Body,
		})
	}

	dev.HTTPConfig = cfg
}

func applyFTPRequest(dev *config.Device, req *FTPRequest) {
	if req == nil {
		return
	}

	if !req.Enabled {
		dev.FTPConfig = nil
		return
	}

	cfg := &config.FTPConfig{
		Enabled: true, WelcomeBanner: req.WelcomeBanner, SystemType: req.SystemType,
		AllowAnonymous: req.AllowAnonymous,
	}
	cfg.Users = make([]config.FTPUser, 0, len(req.Users))
	for _, u := range req.Users {
		cfg.Users = append(cfg.Users, config.FTPUser{Username: u.Username, Password: u.Password, HomeDir: u.HomeDir})
	}

	dev.FTPConfig = cfg
}

func applyNetBIOSRequest(dev *config.Device, req *NetBIOSRequest) {
	if req == nil {
		return
	}

	if !req.Enabled {
		dev.NetBIOSConfig = nil
		return
	}

	dev.NetBIOSConfig = &config.NetBIOSConfig{
		Enabled: true, Name: req.Name, Workgroup: req.Workgroup, NodeType: req.NodeType,
		Services: append([]string(nil), req.Services...), TTL: req.TTL, MsBrowse: req.MsBrowse,
	}
}

// applyServiceProtocolRequests applies the service-protocol blocks
// (DHCP/DHCPv6/DNS/HTTP/FTP/NetBIOS).
func applyServiceProtocolRequests(
	dev *config.Device, dhcp *DHCPRequest, dhcpv6 *DHCPv6Request, dns *DNSRequest,
	httpReq *HTTPRequest, ftp *FTPRequest, netbios *NetBIOSRequest,
) error {
	if err := applyDHCPRequest(dev, dhcp); err != nil {
		return err
	}
	if err := applyDHCPv6Request(dev, dhcpv6); err != nil {
		return err
	}
	if err := applyDNSRequest(dev, dns); err != nil {
		return err
	}

	applyHTTPRequest(dev, httpReq)
	applyFTPRequest(dev, ftp)
	applyNetBIOSRequest(dev, netbios)

	return nil
}

func applyICMPRequest(dev *config.Device, req *ICMPRequest) error {
	if req == nil {
		return nil
	}

	if !req.Enabled {
		dev.ICMPConfig = nil
		return nil
	}

	cfg := &config.ICMPConfig{Enabled: true, TTL: req.TTL, RateLimit: req.RateLimit}

	ip, err := parseOptionalIP(req.AddressMaskReply)
	if err != nil {
		return err
	}

	cfg.AddressMaskReply = ip
	dev.ICMPConfig = cfg

	return nil
}

func applyICMPv6Request(dev *config.Device, req *ICMPv6Request) {
	if req == nil {
		return
	}

	if !req.Enabled {
		dev.ICMPv6Config = nil
		return
	}

	dev.ICMPv6Config = &config.ICMPv6Config{Enabled: true, HopLimit: req.HopLimit, RateLimit: req.RateLimit}
}

// applyTTLRequest applies the device-level TTL (traceroute simulation)
// block. config.TTLConfig has no Enabled field of its own — Enabled here
// only gates whether the pointer is created.
func applyTTLRequest(dev *config.Device, req *TTLRequest) error {
	if req == nil {
		return nil
	}

	if !req.Enabled {
		dev.TTLConfig = nil
		return nil
	}

	cfg := &config.TTLConfig{TTL: req.TTL}

	ip, err := parseOptionalIP(req.IP)
	if err != nil {
		return err
	}

	cfg.IP = ip

	mask, err := parseOptionalIPMask(req.Mask)
	if err != nil {
		return err
	}

	cfg.Mask = mask
	dev.TTLConfig = cfg

	return nil
}

// applyOSFingerprintRequest applies the OS fingerprint block.
// config.OSFingerprintConfig has no Enabled field of its own — see
// applyTTLRequest.
func applyOSFingerprintRequest(dev *config.Device, req *OSFingerprintRequest) {
	if req == nil {
		return
	}

	if !req.Enabled {
		dev.OSFingerprintConfig = nil
		return
	}

	dev.OSFingerprintConfig = &config.OSFingerprintConfig{
		OSType: req.OSType, TTL: req.TTL, WindowSize: req.WindowSize, WindowScale: req.WindowScale,
		MSS: req.MSS, SSHBanner: req.SSHBanner, HTTPServer: req.HTTPServer, FTPBanner: req.FTPBanner,
		SMTPBanner: req.SMTPBanner, TelnetBanner: req.TelnetBanner, DontFragment: req.DontFragment,
	}
}

func applyIPerf3Request(dev *config.Device, req *IPerf3Request) {
	if req == nil {
		return
	}

	if !req.Enabled {
		dev.IPerf3 = nil
		return
	}

	dev.IPerf3 = &config.IPerf3Config{
		Enabled: true, Port: req.Port, MaxBandwidthMbps: req.MaxBandwidthMbps,
		TypicalLatencyMs: req.TypicalLatencyMs, JitterMs: req.JitterMs,
		PacketLossPercent: req.PacketLossPercent, UploadMbps: req.UploadMbps, DownloadMbps: req.DownloadMbps,
	}
}

// applyHostProtocolRequests applies the remaining per-device blocks
// (ICMP/ICMPv6/TTL/OSFingerprint/iPerf3).
func applyHostProtocolRequests(
	dev *config.Device, icmp *ICMPRequest, icmpv6 *ICMPv6Request, ttl *TTLRequest,
	osFingerprint *OSFingerprintRequest, iperf3 *IPerf3Request,
) error {
	if err := applyICMPRequest(dev, icmp); err != nil {
		return err
	}
	if err := applyTTLRequest(dev, ttl); err != nil {
		return err
	}

	applyICMPv6Request(dev, icmpv6)
	applyOSFingerprintRequest(dev, osFingerprint)
	applyIPerf3Request(dev, iperf3)

	return nil
}

func parseOptionalIP(s string) (net.IP, error) {
	if s == "" {
		return nil, nil
	}

	return parseIP(s)
}

func parseOptionalIPMask(s string) (net.IPMask, error) {
	if s == "" {
		return nil, nil
	}

	ip, err := parseIP(s)
	if err != nil {
		return nil, err
	}

	return net.IPMask(ip.To4()), nil
}
