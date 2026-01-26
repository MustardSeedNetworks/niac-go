package api

import (
	"fmt"
	"net"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// Device protocol collection constant.
const deviceProtocolCapacity = 10 // Expected max protocols per device

// collectDeviceProtocols returns a list of enabled protocols for a device.
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

	if dev.CDPConfig != nil && dev.CDPConfig.Enabled {
		resp.CDP = &CDPResponse{
			Enabled:         true,
			Platform:        dev.CDPConfig.Platform,
			SoftwareVersion: dev.CDPConfig.SoftwareVersion,
			PortID:          dev.CDPConfig.PortID,
			Version:         dev.CDPConfig.Version,
			Holdtime:        dev.CDPConfig.Holdtime,
		}
	}

	if dev.LLDPConfig != nil && dev.LLDPConfig.Enabled {
		resp.LLDP = &LLDPResponse{
			Enabled:           true,
			ChassisIDType:     dev.LLDPConfig.ChassisIDType,
			PortDescription:   dev.LLDPConfig.PortDescription,
			SystemDescription: dev.LLDPConfig.SystemDescription,
			TTL:               dev.LLDPConfig.TTL,
		}
	}

	resp.DHCP = buildDHCPResponse(dev)

	if dev.DNSConfig != nil {
		resp.DNS = &DNSResponse{
			Enabled: true,
			Records: len(dev.DNSConfig.ForwardRecords) + len(dev.DNSConfig.ReverseRecords),
		}
	}

	if dev.HTTPConfig != nil && dev.HTTPConfig.Enabled {
		resp.HTTP = &HTTPResponse{
			Enabled:       true,
			ServerName:    dev.HTTPConfig.ServerName,
			EndpointCount: len(dev.HTTPConfig.Endpoints),
		}
	}

	if dev.FTPConfig != nil && dev.FTPConfig.Enabled {
		resp.FTP = &FTPResponse{
			Enabled:        true,
			WelcomeBanner:  dev.FTPConfig.WelcomeBanner,
			AllowAnonymous: dev.FTPConfig.AllowAnonymous,
		}
	}

	if dev.NetBIOSConfig != nil && dev.NetBIOSConfig.Enabled {
		resp.NetBIOS = &NetBIOSResponse{
			Enabled:   true,
			Name:      dev.NetBIOSConfig.Name,
			Workgroup: dev.NetBIOSConfig.Workgroup,
		}
	}

	if dev.STPConfig != nil && dev.STPConfig.Enabled {
		resp.STP = &STPResponse{
			Enabled:  true,
			Priority: dev.STPConfig.BridgePriority,
		}
	}

	resp.TrafficConfig = buildTrafficConfigResponse(dev)
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

// createDeviceFromRequest creates a new Device from a create request.
func createDeviceFromRequest(req DeviceCreateRequest) (*config.Device, error) {
	dev := &config.Device{
		Name: req.Hostname,
		Type: req.Type,
	}

	if req.MAC != "" {
		mac, err := parseMAC(req.MAC)
		if err != nil {
			return nil, err
		}

		dev.MACAddress = mac
	}

	if req.IP != "" {
		ip, err := parseIP(req.IP)
		if err != nil {
			return nil, err
		}

		dev.IPAddresses = []net.IP{ip}
	}

	if req.RawYAML != "" {
		parsed, err := parseDeviceFromYAML(req.RawYAML, req.Hostname)
		if err != nil {
			return nil, err
		}

		return parsed, nil
	}

	return dev, nil
}

// parseDeviceFromYAML parses a device from YAML content.
func parseDeviceFromYAML(yamlStr, hostname string) (*config.Device, error) {
	// SECURITY FIX #153: Validate YAML input before parsing
	if validateErr := validateYAMLInput(yamlStr); validateErr != nil {
		return nil, fmt.Errorf("YAML validation failed: %w", validateErr)
	}

	// This is a simplified parser - in production, use the full config loader
	// For now, return a basic device
	dev := &config.Device{
		Name: hostname,
	}

	// Parse YAML into map for basic fields
	var data map[string]any
	if unmarshalErr := yaml.Unmarshal([]byte(yamlStr), &data); unmarshalErr != nil {
		return nil, fmt.Errorf("invalid YAML: %w", unmarshalErr)
	}

	// SECURITY FIX #153: Check YAML depth to prevent DoS attacks
	if depthErr := checkYAMLDepth(data, 0); depthErr != nil {
		return nil, depthErr
	}

	if t, ok := data["type"].(string); ok {
		dev.Type = t
	}

	if mac, ok := data["mac"].(string); ok {
		if parsed, parseErr := parseMAC(mac); parseErr == nil {
			dev.MACAddress = parsed
		}
	}

	return dev, nil
}

// cloneDevice creates a deep copy of a device with new hostname, IP, and MAC.
func cloneDevice(src *config.Device, newHostname, newIP, newMAC string) *config.Device {
	// Deep copy the device
	cloned := *src
	cloned.Name = newHostname

	// Update IP if provided
	if newIP != "" {
		if ip, parseErr := parseIP(newIP); parseErr == nil {
			cloned.IPAddresses = []net.IP{ip}
		}
	}

	// Update MAC if provided
	if newMAC != "" {
		if mac, parseErr := parseMAC(newMAC); parseErr == nil {
			cloned.MACAddress = mac
		}
	}

	// Update hostname in protocol configs where applicable
	if cloned.SNMPConfig.SysName != "" {
		cloned.SNMPConfig.SysName = newHostname
	}
	// CDP/LLDP derive device ID/system name from hostname automatically
	if cloned.NetBIOSConfig != nil && cloned.NetBIOSConfig.Name != "" {
		cloned.NetBIOSConfig.Name = strings.ToUpper(newHostname)
	}

	return &cloned
}

// serializeDeviceToYAML converts a device to YAML representation.
func serializeDeviceToYAML(dev *config.Device) ([]byte, error) {
	// Build a map representation for YAML serialization
	data := map[string]any{
		"name": dev.Name,
		"type": dev.Type,
	}

	if dev.MACAddress != nil {
		data["mac"] = dev.MACAddress.String()
	}

	if len(dev.IPAddresses) > 0 {
		if len(dev.IPAddresses) == 1 {
			data["ip"] = dev.IPAddresses[0].String()
		} else {
			ips := make([]string, 0, len(dev.IPAddresses))
			for _, ip := range dev.IPAddresses {
				ips = append(ips, ip.String())
			}

			data["ips"] = ips
		}
	}

	// Add SNMP config if present
	if dev.SNMPConfig.Community != "" || dev.SNMPConfig.WalkFile != "" {
		snmp := map[string]any{
			"enabled": true,
		}
		if dev.SNMPConfig.Community != "" {
			snmp["community"] = dev.SNMPConfig.Community
		}

		if dev.SNMPConfig.SysName != "" {
			snmp["sysname"] = dev.SNMPConfig.SysName
		}

		if dev.SNMPConfig.WalkFile != "" {
			snmp["walk_file"] = dev.SNMPConfig.WalkFile
		}

		data["snmp_agent"] = snmp
	}

	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal device YAML: %w", err)
	}
	return yamlData, nil
}

// serializeConfigToYAML converts the full config to YAML.
func serializeConfigToYAML(cfg *config.Config) (string, error) {
	yamlBytes, err := config.MarshalConfigYAML(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config YAML: %w", err)
	}

	return string(yamlBytes), nil
}
