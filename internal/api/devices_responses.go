package api

import (
	"net"

	"github.com/krisarmstrong/niac-go/internal/config"
)

func collectDeviceProtocols(dev *config.Device) []string {
	protocols := make([]string, 0, deviceProtocolCapacity)

	if dev.SNMPConfig.Community != "" || dev.SNMPConfig.WalkFile != "" {
		protocols = append(protocols, "SNMP")
	}

	if dev.DHCPConfig != nil {
		protocols = append(protocols, "DHCP")
	}

	if dev.DNSConfig != nil {
		protocols = append(protocols, "DNS")
	}

	if dev.HTTPConfig != nil {
		protocols = append(protocols, "HTTP")
	}

	if dev.FTPConfig != nil {
		protocols = append(protocols, "FTP")
	}

	if dev.LLDPConfig != nil && dev.LLDPConfig.Enabled {
		protocols = append(protocols, "LLDP")
	}

	if dev.CDPConfig != nil && dev.CDPConfig.Enabled {
		protocols = append(protocols, "CDP")
	}

	if dev.NetBIOSConfig != nil && dev.NetBIOSConfig.Enabled {
		protocols = append(protocols, "NetBIOS")
	}

	if dev.STPConfig != nil && dev.STPConfig.Enabled {
		protocols = append(protocols, "STP")
	}

	return protocols
}

// buildSNMPAgentResponse creates an SNMPAgentResponse from device config.
func buildSNMPAgentResponse(dev *config.Device) *SNMPAgentResponse {
	if dev.SNMPConfig.Community == "" && dev.SNMPConfig.WalkFile == "" {
		return nil
	}

	resp := &SNMPAgentResponse{
		Enabled:     true,
		Community:   dev.SNMPConfig.Community,
		SysName:     dev.SNMPConfig.SysName,
		SysLocation: dev.SNMPConfig.SysLocation,
		SysDescr:    dev.SNMPConfig.SysDescr,
		SysContact:  dev.SNMPConfig.SysContact,
		WalkFile:    dev.SNMPConfig.WalkFile,
	}

	if len(dev.SNMPConfig.AddMibs) > 0 {
		resp.AddMibs = make([]AddMibResponse, 0, len(dev.SNMPConfig.AddMibs))
		for _, mib := range dev.SNMPConfig.AddMibs {
			resp.AddMibs = append(resp.AddMibs, AddMibResponse{
				OID:   mib.OID,
				Type:  mib.Type,
				Value: mib.Value,
			})
		}
	}

	return resp
}

// buildDHCPResponse creates a DHCPResponse from device config.
func buildDHCPResponse(dev *config.Device) *DHCPResponse {
	if dev.DHCPConfig == nil {
		return nil
	}

	resp := &DHCPResponse{Enabled: true}

	if dev.DHCPConfig.PoolStart != nil {
		resp.PoolStart = dev.DHCPConfig.PoolStart.String()
	}

	if dev.DHCPConfig.PoolEnd != nil {
		resp.PoolEnd = dev.DHCPConfig.PoolEnd.String()
	}

	if dev.DHCPConfig.SubnetMask != nil {
		resp.SubnetMask = net.IP(dev.DHCPConfig.SubnetMask).String()
	}

	if dev.DHCPConfig.Router != nil {
		resp.Router = dev.DHCPConfig.Router.String()
	}

	resp.DNS = make([]string, 0, len(dev.DHCPConfig.DomainNameServer))
	for _, dns := range dev.DHCPConfig.DomainNameServer {
		resp.DNS = append(resp.DNS, dns.String())
	}

	return resp
}

// buildTrafficConfigResponse creates a TrafficConfigResponse from device config.
func buildTrafficConfigResponse(dev *config.Device) *TrafficConfigResponse {
	if dev.TrafficConfig == nil || !dev.TrafficConfig.Enabled {
		return nil
	}

	tc := &TrafficConfigResponse{Enabled: true}

	if dev.TrafficConfig.ARPAnnouncements != nil {
		tc.ARPEnabled = dev.TrafficConfig.ARPAnnouncements.Enabled
		tc.ARPInterval = dev.TrafficConfig.ARPAnnouncements.Interval
	}

	if dev.TrafficConfig.PeriodicPings != nil {
		tc.PingEnabled = dev.TrafficConfig.PeriodicPings.Enabled
		tc.PingInterval = dev.TrafficConfig.PeriodicPings.Interval
		tc.PingPayloadSize = dev.TrafficConfig.PeriodicPings.PayloadSize
	}

	return tc
}

// populateProtocolDetails fills in protocol detail fields on DeviceResponse.
func populateProtocolDetails(resp *DeviceResponse, dev *config.Device) {
	resp.SNMPAgent = buildSNMPAgentResponse(dev)
	resp.CDP = buildCDPResponse(dev)
	resp.LLDP = buildLLDPResponse(dev)
	resp.DHCP = buildDHCPResponse(dev)
	resp.DNS = buildDNSResponse(dev)
	resp.HTTP = buildHTTPResponse(dev)
	resp.FTP = buildFTPResponse(dev)
	resp.NetBIOS = buildNetBIOSResponse(dev)
	resp.STP = buildSTPResponse(dev)
	resp.TrafficConfig = buildTrafficConfigResponse(dev)
}

func buildCDPResponse(dev *config.Device) *CDPResponse {
	if dev.CDPConfig == nil || !dev.CDPConfig.Enabled {
		return nil
	}
	return &CDPResponse{
		Enabled:         true,
		Platform:        dev.CDPConfig.Platform,
		SoftwareVersion: dev.CDPConfig.SoftwareVersion,
		PortID:          dev.CDPConfig.PortID,
		Version:         dev.CDPConfig.Version,
		Holdtime:        dev.CDPConfig.Holdtime,
	}
}

func buildLLDPResponse(dev *config.Device) *LLDPResponse {
	if dev.LLDPConfig == nil || !dev.LLDPConfig.Enabled {
		return nil
	}
	return &LLDPResponse{
		Enabled:           true,
		ChassisIDType:     dev.LLDPConfig.ChassisIDType,
		PortDescription:   dev.LLDPConfig.PortDescription,
		SystemDescription: dev.LLDPConfig.SystemDescription,
		TTL:               dev.LLDPConfig.TTL,
	}
}

func buildDNSResponse(dev *config.Device) *DNSResponse {
	if dev.DNSConfig == nil {
		return nil
	}
	return &DNSResponse{
		Enabled: true,
		Records: len(dev.DNSConfig.ForwardRecords) + len(dev.DNSConfig.ReverseRecords),
	}
}

func buildHTTPResponse(dev *config.Device) *HTTPResponse {
	if dev.HTTPConfig == nil || !dev.HTTPConfig.Enabled {
		return nil
	}
	return &HTTPResponse{
		Enabled:       true,
		ServerName:    dev.HTTPConfig.ServerName,
		EndpointCount: len(dev.HTTPConfig.Endpoints),
	}
}

func buildFTPResponse(dev *config.Device) *FTPResponse {
	if dev.FTPConfig == nil || !dev.FTPConfig.Enabled {
		return nil
	}
	return &FTPResponse{
		Enabled:        true,
		WelcomeBanner:  dev.FTPConfig.WelcomeBanner,
		AllowAnonymous: dev.FTPConfig.AllowAnonymous,
	}
}

func buildNetBIOSResponse(dev *config.Device) *NetBIOSResponse {
	if dev.NetBIOSConfig == nil || !dev.NetBIOSConfig.Enabled {
		return nil
	}
	return &NetBIOSResponse{
		Enabled:   true,
		Name:      dev.NetBIOSConfig.Name,
		Workgroup: dev.NetBIOSConfig.Workgroup,
	}
}

func buildSTPResponse(dev *config.Device) *STPResponse {
	if dev.STPConfig == nil || !dev.STPConfig.Enabled {
		return nil
	}
	return &STPResponse{
		Enabled:  true,
		Priority: dev.STPConfig.BridgePriority,
	}
}

// deviceToResponse converts a Device to DeviceResponse.
func deviceToResponse(dev *config.Device, includeDetails, includeYAML bool) DeviceResponse {
	resp := DeviceResponse{
		Hostname:  dev.Name,
		Type:      dev.Type,
		VLAN:      dev.VLAN,
		Protocols: collectDeviceProtocols(dev),
	}

	if dev.MACAddress != nil {
		resp.MAC = dev.MACAddress.String()
	}

	resp.IPs = make([]string, 0, len(dev.IPAddresses))
	for _, ip := range dev.IPAddresses {
		resp.IPs = append(resp.IPs, ip.String())
	}

	resp.Interfaces = make([]string, 0, len(dev.Interfaces))
	for _, iface := range dev.Interfaces {
		resp.Interfaces = append(resp.Interfaces, iface.Name)
	}

	if includeDetails {
		populateProtocolDetails(&resp, dev)
	}

	if includeYAML {
		yamlBytes, err := serializeDeviceToYAML(dev)
		if err == nil {
			resp.RawYAML = string(yamlBytes)
		}
	}

	return resp
}

// Helper functions.
