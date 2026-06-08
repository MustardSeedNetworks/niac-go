package config

import (
	"errors"

	"gopkg.in/yaml.v3"
)

// MarshalConfigYAML builds a YAML representation of a config without relying on struct tags.
func MarshalConfigYAML(cfg *Config) ([]byte, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	data := map[string]any{}

	if cfg.IncludePath != "" {
		data["include_path"] = cfg.IncludePath
	}

	devices := make([]any, 0, len(cfg.Devices))
	for _, dev := range cfg.Devices {
		devices = append(devices, buildDeviceMap(&dev))
	}

	data["devices"] = devices

	return yaml.Marshal(data)
}

func buildDeviceMap(dev *Device) map[string]any {
	data := map[string]any{
		"name": dev.Name,
		"type": dev.Type,
	}

	buildDeviceBasicFields(data, dev)
	buildDeviceInterfaces(data, dev)
	buildDeviceSNMPAgent(data, dev)
	buildDeviceCDP(data, dev)
	buildDeviceLLDP(data, dev)

	return data
}

// buildDeviceBasicFields adds basic device fields to the map.
func buildDeviceBasicFields(data map[string]any, dev *Device) {
	if dev.MACAddress != nil {
		data["mac"] = dev.MACAddress.String()
	}

	buildDeviceIPAddresses(data, dev)

	if dev.VLAN > 0 {
		data["vlan"] = dev.VLAN
	}
}

// buildDeviceIPAddresses adds IP addresses to the device map.
func buildDeviceIPAddresses(data map[string]any, dev *Device) {
	if len(dev.IPAddresses) == 0 {
		return
	}

	if len(dev.IPAddresses) == 1 {
		data["ip"] = dev.IPAddresses[0].String()

		return
	}

	ips := make([]string, 0, len(dev.IPAddresses))
	for _, ip := range dev.IPAddresses {
		ips = append(ips, ip.String())
	}

	data["ips"] = ips
}

func buildDeviceInterfaces(data map[string]any, dev *Device) {
	if len(dev.Interfaces) == 0 {
		return
	}

	interfaces := make([]map[string]any, 0, len(dev.Interfaces))
	for _, iface := range dev.Interfaces {
		ifaceMap := map[string]any{"name": iface.Name}
		addPositiveInt(ifaceMap, "speed", iface.Speed)
		addNonEmptyString(ifaceMap, "duplex", iface.Duplex)
		addNonEmptyString(ifaceMap, "admin_status", iface.AdminStatus)
		addNonEmptyString(ifaceMap, "oper_status", iface.OperStatus)
		addNonEmptyString(ifaceMap, "description", iface.Description)
		if len(iface.VLANs) > 0 {
			ifaceMap["vlans"] = iface.VLANs
		}
		interfaces = append(interfaces, ifaceMap)
	}

	data["interfaces"] = interfaces
}

// buildDeviceSNMPAgent adds SNMP agent configuration to the device map.
func buildDeviceSNMPAgent(data map[string]any, dev *Device) {
	if dev.SNMPConfig.Community == "" && dev.SNMPConfig.WalkFile == "" {
		return
	}

	snmp := map[string]any{
		"enabled": true,
	}

	addNonEmptyString(snmp, "community", dev.SNMPConfig.Community)
	addNonEmptyString(snmp, "sysname", dev.SNMPConfig.SysName)
	addNonEmptyString(snmp, "syslocation", dev.SNMPConfig.SysLocation)
	addNonEmptyString(snmp, "sysdescr", dev.SNMPConfig.SysDescr)
	addNonEmptyString(snmp, "syscontact", dev.SNMPConfig.SysContact)
	addNonEmptyString(snmp, "walk_file", dev.SNMPConfig.WalkFile)

	buildSNMPAddMibs(snmp, dev.SNMPConfig.AddMibs)

	data["snmp_agent"] = snmp
}

// buildSNMPAddMibs adds custom MIBs to the SNMP configuration.
func buildSNMPAddMibs(snmp map[string]any, addMibs []AddMib) {
	if len(addMibs) == 0 {
		return
	}

	mibs := make([]map[string]any, 0, len(addMibs))
	for _, mib := range addMibs {
		mibs = append(mibs, map[string]any{
			"oid":   mib.OID,
			"type":  mib.Type,
			"value": mib.Value,
		})
	}

	snmp["add_mibs"] = mibs
}

// buildDeviceCDP adds CDP configuration to the device map.
func buildDeviceCDP(data map[string]any, dev *Device) {
	if dev.CDPConfig == nil || !dev.CDPConfig.Enabled {
		return
	}

	cdp := map[string]any{
		"enabled": true,
	}

	addNonEmptyString(cdp, "platform", dev.CDPConfig.Platform)
	addNonEmptyString(cdp, "software_version", dev.CDPConfig.SoftwareVersion)
	addNonEmptyString(cdp, "port_id", dev.CDPConfig.PortID)
	addPositiveInt(cdp, "version", dev.CDPConfig.Version)
	addPositiveInt(cdp, "holdtime", dev.CDPConfig.Holdtime)

	data["cdp"] = cdp
}

// buildDeviceLLDP adds LLDP configuration to the device map.
func buildDeviceLLDP(data map[string]any, dev *Device) {
	if dev.LLDPConfig == nil || !dev.LLDPConfig.Enabled {
		return
	}

	lldp := map[string]any{
		"enabled": true,
	}

	addNonEmptyString(lldp, "system_description", dev.LLDPConfig.SystemDescription)
	addNonEmptyString(lldp, "port_description", dev.LLDPConfig.PortDescription)
	addNonEmptyString(lldp, "chassis_id_type", dev.LLDPConfig.ChassisIDType)
	addPositiveInt(lldp, "ttl", dev.LLDPConfig.TTL)

	data["lldp"] = lldp
}

// addNonEmptyString adds a string value to the map if it's non-empty.
func addNonEmptyString(m map[string]any, key, value string) {
	if value != "" {
		m[key] = value
	}
}

// addPositiveInt adds an int value to the map if it's positive.
func addPositiveInt(m map[string]any, key string, value int) {
	if value > 0 {
		m[key] = value
	}
}
