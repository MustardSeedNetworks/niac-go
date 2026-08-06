package config

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Sentinel errors for validation.
var (
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

	// Validate devices (counting devices inside segments)
	if cfg.DeviceCount() == 0 {
		v.addWarning("devices", "no devices defined in configuration")
	}

	deviceNames := make(map[string]bool)
	knownDeviceNames := make(map[string]bool)
	deviceIPs := make(map[string]string)
	deviceMACs := make(map[string]string)

	if len(cfg.Segments) == 0 {
		for i := range cfg.Devices {
			knownDeviceNames[cfg.Devices[i].Name] = true
		}
		for i := range cfg.Devices {
			v.validateDevice(
				&cfg.Devices[i],
				fmt.Sprintf("devices[%d]", i),
				deviceNames,
				knownDeviceNames,
				deviceIPs,
				deviceMACs,
			)
		}
		return v.errors
	}

	v.validateSegmentTags(cfg.Segments)
	for i := range cfg.Segments {
		for j := range cfg.Segments[i].Devices {
			knownDeviceNames[cfg.Segments[i].Devices[j].Name] = true
		}
	}
	for i := range cfg.Segments {
		for j := range cfg.Segments[i].Devices {
			v.validateDevice(
				&cfg.Segments[i].Devices[j],
				fmt.Sprintf("segments[%d].devices[%d]", i, j),
				deviceNames,
				knownDeviceNames,
				deviceIPs,
				deviceMACs,
			)
		}
	}

	return v.errors
}

func (v *Validator) validateSegmentTags(segments []Segment) {
	conflicts := DuplicateSegmentTags(segments)
	reported := make(map[int]struct{}, len(conflicts))
	for _, segment := range segments {
		indices, duplicate := conflicts[segment.Tag]
		if !duplicate {
			continue
		}
		if _, done := reported[segment.Tag]; done {
			continue
		}
		reported[segment.Tag] = struct{}{}
		fields := make([]string, len(indices))
		for index, location := range indices {
			fields[index] = fmt.Sprintf("segments[%d].tag", location)
		}
		for index, field := range fields {
			others := append([]string(nil), fields[:index]...)
			others = append(others, fields[index+1:]...)
			v.addError(
				field,
				fmt.Sprintf(
					"duplicate segment tag %d also used by %s",
					segment.Tag,
					strings.Join(others, ", "),
				),
			)
		}
	}
}

// validateDevice validates a single device configuration.
func (v *Validator) validateDevice(
	device *Device,
	prefix string,
	names map[string]bool,
	knownNames map[string]bool,
	ips map[string]string,
	macs map[string]string,
) {
	v.validateDeviceIdentity(device, prefix, names)
	v.validateDeviceMAC(device, prefix, macs)
	v.validateDeviceIPs(device, prefix, ips)

	if device.VLAN != 0 && !isValidVLANID(device.VLAN) {
		v.addError(prefix+".vlan",
			fmt.Sprintf("invalid VLAN ID: %d (must be %d-%d)", device.VLAN, minVLANID, maxVLANID))
	}

	v.validateSNMPCommunity(device, prefix)
	v.validateSSH(device, prefix)
	v.validateSyslog(device, prefix)
	v.validateSNMPTraps(device, prefix)
	v.validateDNSRecords(device, prefix)
	v.validateTTLConfig(device, prefix)
	v.validateSNMPAccessList(device, prefix)
	v.validateNetBIOSNames(device, prefix)
	v.validatePortChannels(device, prefix)
	v.validateTrunkPorts(device, prefix, knownNames)
}

func (v *Validator) validateSSH(device *Device, prefix string) {
	if device.SSHConfig == nil || !device.SSHConfig.Enabled {
		return
	}
	if strings.TrimSpace(device.SSHConfig.Username) == "" {
		v.addError(prefix+".ssh.username", "SSH requires an explicit username")
	}
	if !validEnvironmentVariable(device.SSHConfig.PasswordEnv) {
		v.addError(prefix+".ssh.password_env", "SSH requires a valid password environment variable")
	}
}

func validEnvironmentVariable(name string) bool {
	matched, _ := regexp.MatchString(`^[A-Za-z_][A-Za-z0-9_]*$`, name)
	return matched
}

func (v *Validator) validateSyslog(device *Device, prefix string) {
	if device.SyslogConfig == nil || !device.SyslogConfig.Enabled {
		return
	}
	if len(device.SyslogConfig.Receivers) == 0 {
		v.addError(prefix+".syslog.receivers", "SYSLOG requires at least one receiver")
		return
	}
	for index, receiver := range device.SyslogConfig.Receivers {
		v.validateUDPReceiver(
			receiver,
			fmt.Sprintf("%s.syslog.receivers[%d]", prefix, index),
			"SYSLOG",
		)
	}
}

func (v *Validator) validateUDPReceiver(receiver, path, protocol string) {
	if receiver == "" {
		v.addError(path, protocol+" receiver cannot be empty")
		return
	}
	host, port, err := net.SplitHostPort(receiver)
	address, addressErr := netip.ParseAddr(host)
	if err != nil || addressErr != nil || !address.Is4() {
		v.addError(path, "invalid "+protocol+" receiver: "+receiver)
		return
	}
	value, err := strconv.ParseUint(port, 10, 16)
	if !validPortSyntax(port) || err != nil || value == 0 {
		v.addError(path, "invalid "+protocol+" receiver port: "+port)
	}
}

func validPortSyntax(port string) bool {
	if len(port) == 0 || len(port) > 5 || port[0] == '0' {
		return false
	}
	for _, character := range port {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func (v *Validator) validateSNMPCommunity(device *Device, prefix string) {
	cfg := device.SNMPConfig
	if cfg.Enabled != nil && !*cfg.Enabled {
		return
	}

	configured := cfg.Enabled != nil || cfg.SysName != "" || cfg.SysDescr != "" || cfg.SysContact != "" ||
		cfg.SysLocation != "" || cfg.WalkFile != "" || len(cfg.WalkFiles) > 0 || len(cfg.AddMibs) > 0 ||
		len(cfg.CommunityIncludes) > 0 || len(cfg.AccessList) > 0 || cfg.SnmpAddr != nil ||
		cfg.Dot1DFdbTable != nil ||
		cfg.Dot1QFdbTable != nil ||
		cfg.Traps != nil
	if configured && strings.TrimSpace(cfg.Community) == "" {
		v.addError(prefix+".snmp_agent.community", "SNMPv1/v2c requires an explicit community")
	}
	v.validateSNMPAddMibs(cfg.AddMibs, prefix)
}

func (v *Validator) validateSNMPAddMibs(mibs []AddMib, prefix string) {
	seen := make(map[string]struct{}, len(mibs))
	for i, mib := range mibs {
		oid := strings.TrimPrefix(strings.TrimSpace(mib.OID), ".")
		if !isValidOIDText(oid) {
			v.addError(
				fmt.Sprintf("%s.snmp_agent.add_mibs[%d].oid", prefix, i),
				"MIB OID must contain numeric dotted components",
			)
		}
		if _, duplicate := seen[oid]; duplicate {
			v.addError(
				fmt.Sprintf("%s.snmp_agent.add_mibs[%d].oid", prefix, i),
				"duplicate MIB OID",
			)
		}
		seen[oid] = struct{}{}
		if strings.TrimSpace(mib.Type) == "" {
			v.addError(
				fmt.Sprintf("%s.snmp_agent.add_mibs[%d].type", prefix, i),
				"MIB type is required",
			)
		}
	}
}

func isValidOIDText(oid string) bool {
	if oid == "" {
		return false
	}
	for part := range strings.SplitSeq(oid, ".") {
		if part == "" {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
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
		validTypes := []string{
			"router", "switch", "layer3-switch", "ap", "access-point", "server", "host", "workstation",
			"iot", "printer", "voip-phone", "firewall",
		}
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
		v.addError(
			prefix+".ttl.ttl",
			fmt.Sprintf("TTL must be between 1 and 255, got %d", device.TTLConfig.TTL),
		)
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
			v.addError(
				fmt.Sprintf("%s.snmp.access_list[%d]", prefix, i),
				"SNMP access list IP is nil",
			)
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
			v.addError(
				fmt.Sprintf("%s.netbios.names[%d].name", prefix, i),
				"NetBIOS name is required",
			)

			continue
		}

		if len(name.Name) > netbiosMaxNameLen {
			v.addError(
				fmt.Sprintf("%s.netbios.names[%d].name", prefix, i),
				"NetBIOS name exceeds 15 characters",
			)
		}
	}
}

// validateSNMPTraps validates SNMP trap configuration.
func (v *Validator) validateSNMPTraps(device *Device, prefix string) {
	if device.SNMPConfig.Traps == nil || !device.SNMPConfig.Traps.Enabled {
		return
	}

	traps := device.SNMPConfig.Traps
	trapPrefix := prefix + ".snmp_agent.traps"

	v.validateTrapReceivers(traps.Receivers, trapPrefix)
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

	host, port, err := net.SplitHostPort(receiver)
	if err != nil {
		if ip := net.ParseIP(receiver); ip == nil {
			v.addError(path, "invalid trap receiver format: "+receiver)
		}

		return
	}

	if ip := net.ParseIP(host); ip == nil {
		v.addError(path, "invalid IP in trap receiver: "+host)
		return
	}
	value, err := strconv.ParseUint(port, 10, 16)
	if !validPortSyntax(port) || err != nil || value == 0 {
		v.addError(path, "invalid trap receiver port: "+port)
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
			v.addError(
				recordPrefix+".rcode",
				fmt.Sprintf("DNS RCode must be 0-15, got %d", record.RCode),
			)
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
			v.addError(
				recordPrefix+".rcode",
				fmt.Sprintf("DNS RCode must be 0-15, got %d", record.RCode),
			)
		}
	}
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
				v.addError(
					fmt.Sprintf("%s.members[%d]", pcPrefix, j),
					"member interface name cannot be empty",
				)
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
		if !IsRoutedTopologyLink(device.Type, trunk) {
			v.validateTrunkVLANs(trunk.VLANs, trunkPrefix)
		}
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
	v.validateTrunkNativeVLAN(trunk.NativeVLAN, trunkPrefix)
	v.validateTrunkRemoteDevice(trunk.RemoteDevice, trunk.RemoteInterface, trunkPrefix, deviceNames)
}

// validateTrunkInterface validates trunk interface name and checks for duplicates.
func (v *Validator) validateTrunkInterface(
	iface, trunkPrefix string,
	seenInterfaces map[string]bool,
) {
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
			fmt.Sprintf(
				"invalid native VLAN: %d (must be %d-%d)",
				nativeVLAN,
				minVLANID,
				maxVLANID,
			),
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
