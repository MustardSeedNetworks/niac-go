// Package config provides configuration validation
package config

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

// Validator validates configuration files
type Validator struct {
	errors *ConfigErrorList
	file   string
}

// NewValidator creates a new configuration validator
func NewValidator(file string) *Validator {
	return &Validator{
		errors: &ConfigErrorList{File: file, Valid: true},
		file:   file,
	}
}

// Validate validates a complete configuration
func (v *Validator) Validate(cfg *Config) *ConfigErrorList {
	if cfg == nil {
		v.addError("", "configuration is nil")
		return v.errors
	}

	// Validate devices
	if len(cfg.Devices) == 0 {
		v.addWarning("devices", "no devices defined in configuration")
	}

	deviceNames := make(map[string]bool)
	deviceIPs := make(map[string]string)
	deviceMACs := make(map[string]string)

	for i, device := range cfg.Devices {
		v.validateDevice(&device, i, deviceNames, deviceIPs, deviceMACs)
	}

	return v.errors
}

// validateDevice validates a single device configuration
func (v *Validator) validateDevice(device *Device, index int, names map[string]bool, ips map[string]string, macs map[string]string) {
	prefix := fmt.Sprintf("devices[%d]", index)

	// Validate device name
	if device.Name == "" {
		v.addError(prefix+".name", "device name is required")
	} else {
		if names[device.Name] {
			v.addError(prefix+".name", fmt.Sprintf("duplicate device name: %s", device.Name))
		}
		names[device.Name] = true
	}

	// Validate device type
	if device.Type == "" {
		v.addError(prefix+".type", "device type is required")
	} else {
		validTypes := []string{"router", "switch", "ap", "access-point", "server", "host"}
		if !contains(validTypes, device.Type) {
			v.addWarning(prefix+".type", fmt.Sprintf("unknown device type: %s (valid: %s)",
				device.Type, strings.Join(validTypes, ", ")))
		}
	}

	// Validate MAC address
	if len(device.MACAddress) > 0 {
		mac := net.HardwareAddr(device.MACAddress).String()
		if mac == "" {
			v.addError(prefix+".mac_address", "invalid MAC address format")
		} else {
			if existingDevice, exists := macs[mac]; exists {
				v.addError(prefix+".mac_address",
					fmt.Sprintf("duplicate MAC address %s (also used by %s)", mac, existingDevice))
			}
			macs[mac] = device.Name
		}
	}

	// Validate IP addresses
	for j, ip := range device.IPAddresses {
		if ip == nil {
			v.addError(fmt.Sprintf("%s.ip_addresses[%d]", prefix, j), "IP address is nil")
			continue
		}

		ipStr := ip.String()
		if existingDevice, exists := ips[ipStr]; exists {
			v.addError(fmt.Sprintf("%s.ip_addresses[%d]", prefix, j),
				fmt.Sprintf("duplicate IP address %s (also used by %s)", ipStr, existingDevice))
		}
		ips[ipStr] = device.Name
	}

	// Validate protocol-specific configurations
	v.validateSNMPTraps(device, prefix)
	v.validateDNSRecords(device, prefix)

	// Validate topology configurations (v1.23.0)
	v.validatePortChannels(device, prefix)
	v.validateTrunkPorts(device, prefix, names)
}

// validateSNMPTraps validates SNMP trap configuration
func (v *Validator) validateSNMPTraps(device *Device, prefix string) {
	if device.SNMPConfig.Traps == nil || !device.SNMPConfig.Traps.Enabled {
		return
	}

	traps := device.SNMPConfig.Traps
	trapPrefix := prefix + ".snmp.traps"

	// Validate threshold configurations
	if traps.HighCPU != nil && traps.HighCPU.Threshold > 0 {
		if err := v.validateThreshold(traps.HighCPU.Threshold, trapPrefix+".high_cpu.threshold"); err != nil {
			v.addError(trapPrefix+".high_cpu.threshold", err.Error())
		}
	}

	if traps.HighMemory != nil && traps.HighMemory.Threshold > 0 {
		if err := v.validateThreshold(traps.HighMemory.Threshold, trapPrefix+".high_memory.threshold"); err != nil {
			v.addError(trapPrefix+".high_memory.threshold", err.Error())
		}
	}

	// Validate trap receivers
	for i, receiver := range traps.Receivers {
		if receiver == "" {
			v.addError(fmt.Sprintf("%s.receivers[%d]", trapPrefix, i), "trap receiver cannot be empty")
			continue
		}

		// Parse receiver format (should be IP:port or just IP)
		host, _, err := net.SplitHostPort(receiver)
		if err != nil {
			// Try parsing as just IP
			if ip := net.ParseIP(receiver); ip == nil {
				v.addError(fmt.Sprintf("%s.receivers[%d]", trapPrefix, i),
					fmt.Sprintf("invalid trap receiver format: %s", receiver))
			}
		} else {
			if ip := net.ParseIP(host); ip == nil {
				v.addError(fmt.Sprintf("%s.receivers[%d]", trapPrefix, i),
					fmt.Sprintf("invalid IP in trap receiver: %s", host))
			}
		}
	}
}

// validateDNSRecords validates DNS record configurations
func (v *Validator) validateDNSRecords(device *Device, prefix string) {
	if device.DNSConfig == nil {
		return
	}

	dns := device.DNSConfig
	dnsPrefix := prefix + ".dns"

	// Validate forward records
	for i, record := range dns.ForwardRecords {
		recordPrefix := fmt.Sprintf("%s.forward_records[%d]", dnsPrefix, i)

		if record.Name == "" {
			v.addError(recordPrefix+".name", "DNS record name is required")
		} else {
			if !isValidDomainName(record.Name) {
				v.addError(recordPrefix+".name", fmt.Sprintf("invalid domain name: %s", record.Name))
			}
		}

		if record.IP == nil {
			v.addError(recordPrefix+".ip", "DNS record IP is required")
		}
	}

	// Validate reverse records
	for i, record := range dns.ReverseRecords {
		recordPrefix := fmt.Sprintf("%s.reverse_records[%d]", dnsPrefix, i)

		if record.IP == nil {
			v.addError(recordPrefix+".ip", "reverse DNS record IP is required")
		}

		if record.Name == "" {
			v.addError(recordPrefix+".name", "reverse DNS record name is required")
		}
	}
}

// validateThreshold validates a threshold value (0-100)
func (v *Validator) validateThreshold(value int, field string) error {
	if value < 0 || value > 100 {
		return fmt.Errorf("threshold must be between 0 and 100, got %d", value)
	}
	return nil
}

// Helper functions

func (v *Validator) addError(field, message string) {
	err := NewConfigError(v.file, field, message)
	v.errors.Add(err)
}

func (v *Validator) addWarning(field, message string) {
	warn := NewConfigWarning(v.file, field, message)
	v.errors.Add(warn)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

var domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)

func isValidDomainName(domain string) bool {
	if len(domain) > 253 {
		return false
	}
	return domainRegex.MatchString(domain)
}

// ValidateIPAddress validates an IP address string
func ValidateIPAddress(ipStr string) error {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return fmt.Errorf("invalid IP address: %s", ipStr)
	}
	return nil
}

// validatePortChannels validates port-channel configuration (v1.23.0)
func (v *Validator) validatePortChannels(device *Device, prefix string) {
	if len(device.PortChannels) == 0 {
		return
	}

	seenIDs := make(map[int]bool)
	memberInterfaces := make(map[string]int) // interface -> port-channel ID

	for i, pc := range device.PortChannels {
		pcPrefix := fmt.Sprintf("%s.port_channels[%d]", prefix, i)

		// Validate port-channel ID
		if pc.ID <= 0 {
			v.addError(pcPrefix+".id", "port-channel ID must be positive")
		} else if seenIDs[pc.ID] {
			v.addError(pcPrefix+".id", fmt.Sprintf("duplicate port-channel ID: %d", pc.ID))
		}
		seenIDs[pc.ID] = true

		// Validate members
		if len(pc.Members) == 0 {
			v.addError(pcPrefix+".members", "port-channel must have at least one member interface")
		}

		for j, member := range pc.Members {
			if member == "" {
				v.addError(fmt.Sprintf("%s.members[%d]", pcPrefix, j), "member interface name cannot be empty")
			} else if existingPC, exists := memberInterfaces[member]; exists {
				v.addError(fmt.Sprintf("%s.members[%d]", pcPrefix, j),
					fmt.Sprintf("interface %s already belongs to port-channel %d", member, existingPC))
			} else {
				memberInterfaces[member] = pc.ID
			}
		}

		// Validate mode
		if pc.Mode != "" {
			validModes := []string{"active", "passive", "on"}
			if !contains(validModes, pc.Mode) {
				v.addWarning(pcPrefix+".mode", fmt.Sprintf("unknown LACP mode: %s (valid: %s)",
					pc.Mode, strings.Join(validModes, ", ")))
			}
		}
	}
}

// validateTrunkPorts validates trunk port configuration (v1.23.0)
func (v *Validator) validateTrunkPorts(device *Device, prefix string, deviceNames map[string]bool) {
	if len(device.TrunkPorts) == 0 {
		return
	}

	seenInterfaces := make(map[string]bool)

	for i, trunk := range device.TrunkPorts {
		trunkPrefix := fmt.Sprintf("%s.trunk_ports[%d]", prefix, i)

		// Validate interface
		if trunk.Interface == "" {
			v.addError(trunkPrefix+".interface", "trunk interface name is required")
		} else if seenInterfaces[trunk.Interface] {
			v.addError(trunkPrefix+".interface", fmt.Sprintf("duplicate trunk configuration for interface: %s", trunk.Interface))
		}
		seenInterfaces[trunk.Interface] = true

		// Validate VLANs
		if len(trunk.VLANs) == 0 {
			v.addWarning(trunkPrefix+".vlans", "trunk port has no allowed VLANs configured")
		}

		for j, vlan := range trunk.VLANs {
			if vlan < 1 || vlan > 4094 {
				v.addError(fmt.Sprintf("%s.vlans[%d]", trunkPrefix, j),
					fmt.Sprintf("invalid VLAN ID: %d (must be 1-4094)", vlan))
			}
		}

		// Validate native VLAN
		if trunk.NativeVLAN != 0 && (trunk.NativeVLAN < 1 || trunk.NativeVLAN > 4094) {
			v.addError(trunkPrefix+".native_vlan", fmt.Sprintf("invalid native VLAN: %d (must be 1-4094)", trunk.NativeVLAN))
		}

		// Validate remote device reference
		if trunk.RemoteDevice != "" {
			if !deviceNames[trunk.RemoteDevice] {
				v.addWarning(trunkPrefix+".remote_device",
					fmt.Sprintf("remote device %s not found in configuration", trunk.RemoteDevice))
			}
		}

		// Check if remote interface is specified when remote device is specified
		if trunk.RemoteDevice != "" && trunk.RemoteInterface == "" {
			v.addWarning(trunkPrefix+".remote_interface",
				"remote_interface should be specified when remote_device is set")
		}
	}
}

// ValidateMACAddress validates a MAC address string
func ValidateMACAddress(macStr string) error {
	_, err := net.ParseMAC(macStr)
	if err != nil {
		return fmt.Errorf("invalid MAC address: %s", macStr)
	}
	return nil
}

// ValidatePort validates a port number
func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port: %d (must be 1-65535)", port)
	}
	return nil
}
