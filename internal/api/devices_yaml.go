package api

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// validateYAMLInput checks YAML input for size and complexity limits.
// SECURITY FIX #153: Prevents memory exhaustion and DoS attacks.
func validateYAMLInput(input string) error {
	// Check size
	if len(input) > MaxYAMLSize {
		return fmt.Errorf("YAML input too large: %d bytes (max %d)", len(input), MaxYAMLSize)
	}

	// Check for empty input
	if strings.TrimSpace(input) == "" {
		return errors.New("YAML input is empty")
	}

	return nil
}

// checkYAMLDepth verifies parsed YAML doesn't exceed maximum nesting depth.
// SECURITY FIX #153: Prevents billion laughs and similar attacks.
func checkYAMLDepth(data any, currentDepth int) error {
	if currentDepth > MaxYAMLDepth {
		return fmt.Errorf("YAML nesting depth exceeds maximum of %d", MaxYAMLDepth)
	}

	switch v := data.(type) {
	case map[string]any:
		for _, val := range v {
			err := checkYAMLDepth(val, currentDepth+1)
			if err != nil {
				return err
			}
		}
	case []any:
		for _, item := range v {
			err := checkYAMLDepth(item, currentDepth+1)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func parseDeviceFromYAML(yamlStr, hostname string) (*config.Device, error) {
	// SECURITY FIX #153: Validate YAML input before parsing
	if validateErr := validateYAMLInput(yamlStr); validateErr != nil {
		return nil, fmt.Errorf("YAML validation failed: %w", validateErr)
	}

	dev := &config.Device{
		Name: hostname,
	}

	// Parse YAML into map for all fields
	var data map[string]any
	if unmarshalErr := yaml.Unmarshal([]byte(yamlStr), &data); unmarshalErr != nil {
		return nil, fmt.Errorf("invalid YAML: %w", unmarshalErr)
	}

	// SECURITY FIX #153: Check YAML depth to prevent DoS attacks
	if depthErr := checkYAMLDepth(data, 0); depthErr != nil {
		return nil, depthErr
	}

	// FIX #288: Parse all device fields from YAML

	// Basic fields
	if t, ok := data["type"].(string); ok {
		dev.Type = t
	}

	if mac, ok := data["mac"].(string); ok {
		if parsed, parseErr := parseMAC(mac); parseErr == nil {
			dev.MACAddress = parsed
		}
	}

	// Parse IP addresses
	parseDeviceIPs(data, dev)

	// Parse VLAN
	if vlan, ok := data["vlan"].(int); ok {
		dev.VLAN = vlan
	}

	// Parse SNMP config
	parseDeviceSNMP(data, dev)

	// Parse LLDP config
	parseDeviceLLDP(data, dev)

	// Parse CDP config
	parseDeviceCDP(data, dev)

	// Parse DHCP config
	parseDeviceDHCP(data, dev)

	// Parse interfaces
	parseDeviceInterfaces(data, dev)

	// Parse traffic config
	parseDeviceTraffic(data, dev)

	return dev, nil
}

// parseDeviceIPs extracts IP addresses from YAML data.
func parseDeviceIPs(data map[string]any, dev *config.Device) {
	if ip, ok := data["ip"].(string); ok {
		if parsed := net.ParseIP(ip); parsed != nil {
			dev.IPAddresses = []net.IP{parsed}
		}
	}

	if ips, ok := data["ips"].([]any); ok {
		for _, ipVal := range ips {
			if ipStr, strOK := ipVal.(string); strOK {
				if parsed := net.ParseIP(ipStr); parsed != nil {
					dev.IPAddresses = append(dev.IPAddresses, parsed)
				}
			}
		}
	}
}

// parseDeviceSNMP extracts SNMP configuration from YAML data.
func parseDeviceSNMP(data map[string]any, dev *config.Device) {
	snmpData, ok := data["snmp_agent"].(map[string]any)
	if !ok {
		return
	}

	if community, cOK := snmpData["community"].(string); cOK {
		dev.SNMPConfig.Community = community
	}

	if sysName, sOK := snmpData["sysname"].(string); sOK {
		dev.SNMPConfig.SysName = sysName
	}

	if sysLocation, lOK := snmpData["syslocation"].(string); lOK {
		dev.SNMPConfig.SysLocation = sysLocation
	}

	if sysDescr, dOK := snmpData["sysdescr"].(string); dOK {
		dev.SNMPConfig.SysDescr = sysDescr
	}

	if sysContact, cOK := snmpData["syscontact"].(string); cOK {
		dev.SNMPConfig.SysContact = sysContact
	}

	if walkFile, wOK := snmpData["walk_file"].(string); wOK {
		dev.SNMPConfig.WalkFile = walkFile
	}

	// Parse add_mibs
	if mibs, mOK := snmpData["add_mibs"].([]any); mOK {
		dev.SNMPConfig.AddMibs = parseSNMPMibs(mibs)
	}
}

// parseSNMPMibs extracts a slice of AddMib from raw YAML list data.
func parseSNMPMibs(mibs []any) []config.AddMib {
	result := make([]config.AddMib, 0, len(mibs))

	for _, mibVal := range mibs {
		mibMap, mapOK := mibVal.(map[string]any)
		if !mapOK {
			continue
		}

		mib := config.AddMib{}
		if oid, oOK := mibMap["oid"].(string); oOK {
			mib.OID = oid
		}

		if mType, tOK := mibMap["type"].(string); tOK {
			mib.Type = mType
		}

		if val, vOK := mibMap["value"].(string); vOK {
			mib.Value = val
		}

		result = append(result, mib)
	}

	return result
}

// parseDeviceLLDP extracts LLDP configuration from YAML data.
func parseDeviceLLDP(data map[string]any, dev *config.Device) {
	lldpData, ok := data["lldp"].(map[string]any)
	if !ok {
		return
	}

	lldp := &config.LLDPConfig{}

	if enabled, eOK := lldpData["enabled"].(bool); eOK {
		lldp.Enabled = enabled
	}

	if chassisIDType, cOK := lldpData["chassis_id_type"].(string); cOK {
		lldp.ChassisIDType = chassisIDType
	}

	if portDesc, pOK := lldpData["port_description"].(string); pOK {
		lldp.PortDescription = portDesc
	}

	if sysDesc, sOK := lldpData["system_description"].(string); sOK {
		lldp.SystemDescription = sysDesc
	}

	if ttl, tOK := lldpData["ttl"].(int); tOK {
		lldp.TTL = ttl
	}

	dev.LLDPConfig = lldp
}

// parseDeviceCDP extracts CDP configuration from YAML data.
func parseDeviceCDP(data map[string]any, dev *config.Device) {
	cdpData, ok := data["cdp"].(map[string]any)
	if !ok {
		return
	}

	cdp := &config.CDPConfig{}

	if enabled, eOK := cdpData["enabled"].(bool); eOK {
		cdp.Enabled = enabled
	}

	if platform, pOK := cdpData["platform"].(string); pOK {
		cdp.Platform = platform
	}

	if sv, sOK := cdpData["software_version"].(string); sOK {
		cdp.SoftwareVersion = sv
	}

	if portID, pOK := cdpData["port_id"].(string); pOK {
		cdp.PortID = portID
	}

	if version, vOK := cdpData["version"].(int); vOK {
		cdp.Version = version
	}

	if holdtime, hOK := cdpData["holdtime"].(int); hOK {
		cdp.Holdtime = holdtime
	}

	dev.CDPConfig = cdp
}

// parseDeviceDHCP extracts DHCP configuration from YAML data.
func parseDeviceDHCP(data map[string]any, dev *config.Device) {
	dhcpData, ok := data["dhcp"].(map[string]any)
	if !ok {
		return
	}

	dhcp := &config.DHCPConfig{}

	if poolStart, pOK := dhcpData["pool_start"].(string); pOK {
		dhcp.PoolStart = net.ParseIP(poolStart)
	}

	if poolEnd, pOK := dhcpData["pool_end"].(string); pOK {
		dhcp.PoolEnd = net.ParseIP(poolEnd)
	}

	if subnetMask, sOK := dhcpData["subnet_mask"].(string); sOK {
		mask := net.ParseIP(subnetMask)
		if mask != nil {
			dhcp.SubnetMask = net.IPMask(mask.To4())
		}
	}

	if router, rOK := dhcpData["router"].(string); rOK {
		dhcp.Router = net.ParseIP(router)
	}

	if dns, dOK := dhcpData["dns"].([]any); dOK {
		for _, dnsVal := range dns {
			if dnsStr, strOK := dnsVal.(string); strOK {
				if parsed := net.ParseIP(dnsStr); parsed != nil {
					dhcp.DomainNameServer = append(dhcp.DomainNameServer, parsed)
				}
			}
		}
	}

	dev.DHCPConfig = dhcp
}

// parseDeviceInterfaces extracts interface configuration from YAML data.
func parseDeviceInterfaces(data map[string]any, dev *config.Device) {
	interfaces, ok := data["interfaces"].([]any)
	if !ok {
		return
	}

	for _, ifaceVal := range interfaces {
		ifaceMap, mapOK := ifaceVal.(map[string]any)
		if !mapOK {
			continue
		}

		iface := config.Interface{}
		if name, nOK := ifaceMap["name"].(string); nOK {
			iface.Name = name
		}

		dev.Interfaces = append(dev.Interfaces, iface)
	}
}

// parseDeviceTraffic extracts traffic configuration from YAML data.
func parseDeviceTraffic(data map[string]any, dev *config.Device) {
	trafficData, ok := data["traffic"].(map[string]any)
	if !ok {
		return
	}

	traffic := &config.TrafficConfig{}

	if enabled, eOK := trafficData["enabled"].(bool); eOK {
		traffic.Enabled = enabled
	}

	if arp, aOK := trafficData["arp_announcements"].(map[string]any); aOK {
		traffic.ARPAnnouncements = &config.ARPAnnouncementConfig{}
		if enabled, eOK := arp["enabled"].(bool); eOK {
			traffic.ARPAnnouncements.Enabled = enabled
		}

		if interval, iOK := arp["interval"].(int); iOK {
			traffic.ARPAnnouncements.Interval = interval
		}
	}

	if pings, pOK := trafficData["periodic_pings"].(map[string]any); pOK {
		traffic.PeriodicPings = &config.PeriodicPingConfig{}
		if enabled, eOK := pings["enabled"].(bool); eOK {
			traffic.PeriodicPings.Enabled = enabled
		}

		if interval, iOK := pings["interval"].(int); iOK {
			traffic.PeriodicPings.Interval = interval
		}

		if payloadSize, psOK := pings["payload_size"].(int); psOK {
			traffic.PeriodicPings.PayloadSize = payloadSize
		}
	}

	dev.TrafficConfig = traffic
}

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

func serializeConfigToYAML(cfg *config.Config) (string, error) {
	yamlBytes, err := config.MarshalConfigYAML(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config YAML: %w", err)
	}

	return string(yamlBytes), nil
}
