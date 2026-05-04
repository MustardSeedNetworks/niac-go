package config

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"slices"
	"strings"
)

// Sentinel errors for validation.
var (
	ErrThresholdOutOfRange  = errors.New("threshold must be between 0 and 100")
	ErrInvalidIPAddressVal  = errors.New("invalid IP address")
	ErrInvalidMACAddressVal = errors.New("invalid MAC address")
	ErrInvalidPort          = errors.New("invalid port (must be 1-65535)")
)

// Validation constants.
const (
	maxDomainNameLen = 253  // RFC 1035 max domain name length
	minVLANID        = 1    // Minimum valid VLAN ID
	maxVLANID        = 4094 // Maximum valid VLAN ID (IEEE 802.1Q)
)

// Validator validates configuration files.
type Validator struct {
	errors *ListError
	file   string
}

// NewValidator creates a new configuration validator.
func NewValidator(file string) *Validator {
	return &Validator{
		errors: &ListError{File: file, Valid: true},
		file:   file,
	}
}

// Validate validates a complete configuration.
func (v *Validator) Validate(cfg *Config) *ListError {
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

// validateDevice validates a single device configuration.
func (v *Validator) validateDevice(
	device *Device,
	index int,
	names map[string]bool,
	ips map[string]string,
	macs map[string]string,
) {
	prefix := fmt.Sprintf("devices[%d]", index)

	v.validateDeviceIdentity(device, prefix, names)
	v.validateDeviceMAC(device, prefix, macs)
	v.validateDeviceIPs(device, prefix, ips)

	if device.VLAN != 0 && !isValidVLANID(device.VLAN) {
		v.addError(prefix+".vlan",
			fmt.Sprintf("invalid VLAN ID: %d (must be %d-%d)", device.VLAN, minVLANID, maxVLANID))
	}

	v.validateSNMPTraps(device, prefix)
	v.validateDNSRecords(device, prefix)
	v.validateTTLConfig(device, prefix)
	v.validateSNMPAccessList(device, prefix)
	v.validateNetBIOSNames(device, prefix)
	v.validatePortChannels(device, prefix)
	v.validateTrunkPorts(device, prefix, names)
}

// validateDeviceIdentity validates device name and type fields.
func (v *Validator) validateDeviceIdentity(device *Device, prefix string, names map[string]bool) {
	if device.Name == "" {
		v.addError(prefix+".name", "device name is required")
	} else {
		if names[device.Name] {
			v.addError(prefix+".name", "duplicate device name: "+device.Name)
		}

		names[device.Name] = true
	}

	if device.Type == "" {
		v.addError(prefix+".type", "device type is required")
	} else {
		validTypes := []string{"router", "switch", "ap", "access-point", "server", "host"}
		if !contains(validTypes, device.Type) {
			v.addWarning(prefix+".type", fmt.Sprintf("unknown device type: %s (valid: %s)",
				device.Type, strings.Join(validTypes, ", ")))
		}
	}
}

// validateDeviceMAC validates MAC address and checks for duplicates.
func (v *Validator) validateDeviceMAC(device *Device, prefix string, macs map[string]string) {
	if len(device.MACAddress) == 0 {
		return
	}

	mac := device.MACAddress.String()
	if mac == "" {
		v.addError(prefix+".mac_address", "invalid MAC address format")

		return
	}

	if existingDevice, exists := macs[mac]; exists {
		v.addError(prefix+".mac_address",
			fmt.Sprintf("duplicate MAC address %s (also used by %s)", mac, existingDevice))
	}

	macs[mac] = device.Name
}

// validateDeviceIPs validates IP addresses and checks for duplicates.
func (v *Validator) validateDeviceIPs(device *Device, prefix string, ips map[string]string) {
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
}

// validateTTLConfig validates TTL configuration for traceroute simulation.
func (v *Validator) validateTTLConfig(device *Device, prefix string) {
	if device.TTLConfig == nil {
		return
	}

	if device.TTLConfig.TTL < 1 || device.TTLConfig.TTL > 255 {
		v.addError(prefix+".ttl.ttl", fmt.Sprintf("TTL must be between 1 and 255, got %d", device.TTLConfig.TTL))
	}

	if device.TTLConfig.IP != nil && device.TTLConfig.IP.To4() == nil {
		v.addError(prefix+".ttl.ip", "TTL IP must be IPv4")
	}

	if device.TTLConfig.Mask != nil && len(device.TTLConfig.Mask) != 4 {
		v.addError(prefix+".ttl.mask", "TTL mask must be IPv4 netmask")
	}
}

// validateSNMPAccessList validates SNMP access list entries.
func (v *Validator) validateSNMPAccessList(device *Device, prefix string) {
	if len(device.SNMPConfig.AccessList) == 0 {
		return
	}

	for i, ip := range device.SNMPConfig.AccessList {
		if ip == nil {
			v.addError(fmt.Sprintf("%s.snmp.access_list[%d]", prefix, i), "SNMP access list IP is nil")
		}
	}
}

// validateNetBIOSNames validates NetBIOS name entries.
func (v *Validator) validateNetBIOSNames(device *Device, prefix string) {
	if device.NetBIOSConfig == nil {
		return
	}

	for i, name := range device.NetBIOSConfig.Names {
		if name.Name == "" {
			v.addError(fmt.Sprintf("%s.netbios.names[%d].name", prefix, i), "NetBIOS name is required")

			continue
		}

		if len(name.Name) > netbiosMaxNameLen {
			v.addError(fmt.Sprintf("%s.netbios.names[%d].name", prefix, i), "NetBIOS name exceeds 15 characters")
		}
	}
}

// validateSNMPTraps validates SNMP trap configuration.
func (v *Validator) validateSNMPTraps(device *Device, prefix string) {
	if device.SNMPConfig.Traps == nil || !device.SNMPConfig.Traps.Enabled {
		return
	}

	traps := device.SNMPConfig.Traps
	trapPrefix := prefix + ".snmp.traps"

	v.validateTrapThresholds(traps, trapPrefix)
	v.validateTrapReceivers(traps.Receivers, trapPrefix)
}

// validateTrapThresholds validates SNMP trap threshold configurations.
func (v *Validator) validateTrapThresholds(traps *TrapConfig, trapPrefix string) {
	v.validateSingleThreshold(traps.HighCPU, trapPrefix+".high_cpu.threshold")
	v.validateSingleThreshold(traps.HighMemory, trapPrefix+".high_memory.threshold")
}

// validateSingleThreshold validates a single threshold trap configuration.
func (v *Validator) validateSingleThreshold(config *ThresholdTrapConfig, path string) {
	if config == nil || config.Threshold <= 0 {
		return
	}

	if err := v.validateThreshold(config.Threshold, path); err != nil {
		v.addError(path, err.Error())
	}
}

// validateTrapReceivers validates SNMP trap receiver addresses.
func (v *Validator) validateTrapReceivers(receivers []string, trapPrefix string) {
	for i, receiver := range receivers {
		receiverPath := fmt.Sprintf("%s.receivers[%d]", trapPrefix, i)
		v.validateSingleTrapReceiver(receiver, receiverPath)
	}
}

// validateSingleTrapReceiver validates a single trap receiver address.
func (v *Validator) validateSingleTrapReceiver(receiver, path string) {
	if receiver == "" {
		v.addError(path, "trap receiver cannot be empty")

		return
	}

	host, _, err := net.SplitHostPort(receiver)
	if err != nil {
		if ip := net.ParseIP(receiver); ip == nil {
			v.addError(path, "invalid trap receiver format: "+receiver)
		}

		return
	}

	if ip := net.ParseIP(host); ip == nil {
		v.addError(path, "invalid IP in trap receiver: "+host)
	}
}

// validateDNSRecords validates DNS record configurations.
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
		} else if !isValidDomainName(record.Name) {
			v.addError(recordPrefix+".name", "invalid domain name: "+record.Name)
		}

		if record.IP == nil {
			v.addError(recordPrefix+".ip", "DNS record IP is required")
		}

		if record.RCode < 0 || record.RCode > 15 {
			v.addError(recordPrefix+".rcode", fmt.Sprintf("DNS RCode must be 0-15, got %d", record.RCode))
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

		if record.RCode < 0 || record.RCode > 15 {
			v.addError(recordPrefix+".rcode", fmt.Sprintf("DNS RCode must be 0-15, got %d", record.RCode))
		}
	}
}

// validateThreshold validates a threshold value (0-100).
func (v *Validator) validateThreshold(value int, _ string) error {
	if value < 0 || value > 100 {
		return fmt.Errorf("%w: got %d", ErrThresholdOutOfRange, value)
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
	return slices.Contains(slice, item)
}

var domainRegex = regexp.MustCompile(
	`^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`,
)

func isValidDomainName(domain string) bool {
	if len(domain) > maxDomainNameLen {
		return false
	}

	return domainRegex.MatchString(domain)
}

// ValidateIPAddress validates an IP address string.
func ValidateIPAddress(ipStr string) error {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return fmt.Errorf("%w: %s", ErrInvalidIPAddressVal, ipStr)
	}

	return nil
}

// validatePortChannels validates port-channel configuration (v1.23.0).
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

// validateTrunkPorts validates trunk port configuration (v1.23.0).
func (v *Validator) validateTrunkPorts(device *Device, prefix string, deviceNames map[string]bool) {
	if len(device.TrunkPorts) == 0 {
		return
	}

	seenInterfaces := make(map[string]bool)

	for i, trunk := range device.TrunkPorts {
		trunkPrefix := fmt.Sprintf("%s.trunk_ports[%d]", prefix, i)
		v.validateSingleTrunkPort(&trunk, trunkPrefix, seenInterfaces, deviceNames)
	}
}

// validateSingleTrunkPort validates a single trunk port configuration.
func (v *Validator) validateSingleTrunkPort(
	trunk *TrunkPort,
	trunkPrefix string,
	seenInterfaces map[string]bool,
	deviceNames map[string]bool,
) {
	v.validateTrunkInterface(trunk.Interface, trunkPrefix, seenInterfaces)
	v.validateTrunkVLANs(trunk.VLANs, trunkPrefix)
	v.validateTrunkNativeVLAN(trunk.NativeVLAN, trunkPrefix)
	v.validateTrunkRemoteDevice(trunk.RemoteDevice, trunk.RemoteInterface, trunkPrefix, deviceNames)
}

// validateTrunkInterface validates trunk interface name and checks for duplicates.
func (v *Validator) validateTrunkInterface(iface, trunkPrefix string, seenInterfaces map[string]bool) {
	if iface == "" {
		v.addError(trunkPrefix+".interface", "trunk interface name is required")

		return
	}

	if seenInterfaces[iface] {
		v.addError(trunkPrefix+".interface", "duplicate trunk configuration for interface: "+iface)

		return
	}

	seenInterfaces[iface] = true
}

// validateTrunkVLANs validates VLAN IDs on a trunk port.
func (v *Validator) validateTrunkVLANs(vlans []int, trunkPrefix string) {
	if len(vlans) == 0 {
		v.addWarning(trunkPrefix+".vlans", "trunk port has no allowed VLANs configured")

		return
	}

	for j, vlan := range vlans {
		if !isValidVLANID(vlan) {
			v.addError(
				fmt.Sprintf("%s.vlans[%d]", trunkPrefix, j),
				fmt.Sprintf("invalid VLAN ID: %d (must be %d-%d)", vlan, minVLANID, maxVLANID),
			)
		}
	}
}

// validateTrunkNativeVLAN validates the native VLAN ID.
func (v *Validator) validateTrunkNativeVLAN(nativeVLAN int, trunkPrefix string) {
	if nativeVLAN == 0 {
		return
	}

	if !isValidVLANID(nativeVLAN) {
		v.addError(
			trunkPrefix+".native_vlan",
			fmt.Sprintf("invalid native VLAN: %d (must be %d-%d)", nativeVLAN, minVLANID, maxVLANID),
		)
	}
}

// validateTrunkRemoteDevice validates remote device references.
func (v *Validator) validateTrunkRemoteDevice(
	remoteDevice, remoteInterface, trunkPrefix string,
	deviceNames map[string]bool,
) {
	if remoteDevice == "" {
		return
	}

	if !deviceNames[remoteDevice] {
		v.addWarning(
			trunkPrefix+".remote_device",
			fmt.Sprintf("remote device %s not found in configuration", remoteDevice),
		)
	}

	if remoteInterface == "" {
		v.addWarning(
			trunkPrefix+".remote_interface",
			"remote_interface should be specified when remote_device is set",
		)
	}
}

// isValidVLANID checks if a VLAN ID is within valid range.
func isValidVLANID(vlan int) bool {
	return vlan >= minVLANID && vlan <= maxVLANID
}

// ValidateMACAddress validates a MAC address string.
func ValidateMACAddress(macStr string) error {
	_, err := net.ParseMAC(macStr)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidMACAddressVal, macStr)
	}

	return nil
}

// ValidatePort validates a port number.
func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%w: %d", ErrInvalidPort, port)
	}

	return nil
}
