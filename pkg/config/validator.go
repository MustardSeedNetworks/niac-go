package config

import (
	"fmt"
	"net"
	"strings"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Device  string
	Message string
	Level   string // "error", "warning", "info"
}

// ValidationResult contains the results of configuration validation
type ValidationResult struct {
	Errors   []ValidationError
	Warnings []ValidationError
	Info     []ValidationError
	Valid    bool
}

// Validator validates NIAC configurations
type Validator struct {
	result *ValidationResult
}

// NewValidator creates a new configuration validator
func NewValidator() *Validator {
	return &Validator{
		result: &ValidationResult{
			Errors:   make([]ValidationError, 0),
			Warnings: make([]ValidationError, 0),
			Info:     make([]ValidationError, 0),
			Valid:    true,
		},
	}
}

// Validate performs comprehensive validation on a configuration
func (v *Validator) Validate(cfg *Config) *ValidationResult {
	if cfg == nil {
		v.addError("", "", "Configuration is nil")
		return v.result
	}

	// Check if there are any devices
	if len(cfg.Devices) == 0 {
		v.addWarning("devices", "", "No devices defined in configuration")
	} else {
		v.addInfo("devices", "", fmt.Sprintf("Configuration contains %d device(s)", len(cfg.Devices)))
	}

	// Validate each device
	for i, device := range cfg.Devices {
		v.validateDevice(&device, i)
	}

	return v.result
}

// validateDevice validates a single device configuration
func (v *Validator) validateDevice(device *Device, index int) {
	deviceName := device.Name
	if deviceName == "" {
		deviceName = fmt.Sprintf("device[%d]", index)
	}

	// Check device name
	if device.Name == "" {
		v.addError("name", deviceName, "Device name is required")
	} else {
		v.addInfo("name", deviceName, fmt.Sprintf("Device name: %s", device.Name))
	}

	// Check device type
	if device.Type == "" {
		v.addWarning("type", deviceName, "Device type not specified (will use default)")
	} else {
		validTypes := []string{"router", "switch", "ap", "access-point", "server", "generic"}
		if !contains(validTypes, device.Type) {
			v.addWarning("type", deviceName, fmt.Sprintf("Unknown device type '%s' (will use as-is)", device.Type))
		} else {
			v.addInfo("type", deviceName, fmt.Sprintf("Device type: %s", device.Type))
		}
	}

	// Check MAC address
	if len(device.MACAddress) == 0 {
		v.addError("mac", deviceName, "MAC address is required")
	} else if len(device.MACAddress) != 6 {
		v.addError("mac", deviceName, fmt.Sprintf("Invalid MAC address length: %d bytes (expected 6)", len(device.MACAddress)))
	} else {
		v.addInfo("mac", deviceName, fmt.Sprintf("MAC address: %s", device.MACAddress))
	}

	// Check IP addresses
	if len(device.IPAddresses) == 0 {
		v.addWarning("ip", deviceName, "No IP addresses configured")
	} else {
		v.addInfo("ip", deviceName, fmt.Sprintf("IP addresses: %d configured", len(device.IPAddresses)))
		for i, ip := range device.IPAddresses {
			if ip == nil {
				v.addError("ip", deviceName, fmt.Sprintf("IP address %d is nil", i))
			} else if ip.IsUnspecified() {
				v.addWarning("ip", deviceName, fmt.Sprintf("IP address %d is unspecified (0.0.0.0 or ::)", i))
			}
		}
	}

	// Validate protocol configurations
	v.validateProtocolConfigs(device, deviceName)

	// Validate v1.6.0 features
	v.validateTrafficConfig(device, deviceName)
	v.validateTrapConfig(device, deviceName)
}

// validateProtocolConfigs validates protocol-specific configurations
func (v *Validator) validateProtocolConfigs(device *Device, deviceName string) {
	protocolsEnabled := make([]string, 0)

	// Check LLDP
	if device.LLDPConfig != nil && device.LLDPConfig.Enabled {
		protocolsEnabled = append(protocolsEnabled, "LLDP")
		if device.LLDPConfig.TTL > 0 && device.LLDPConfig.TTL < 30 {
			v.addWarning("lldp.ttl", deviceName, fmt.Sprintf("LLDP TTL is very short (%d seconds)", device.LLDPConfig.TTL))
		}
	}

	// Check CDP
	if device.CDPConfig != nil && device.CDPConfig.Enabled {
		protocolsEnabled = append(protocolsEnabled, "CDP")
	}

	// Check EDP
	if device.EDPConfig != nil && device.EDPConfig.Enabled {
		protocolsEnabled = append(protocolsEnabled, "EDP")
	}

	// Check FDP
	if device.FDPConfig != nil && device.FDPConfig.Enabled {
		protocolsEnabled = append(protocolsEnabled, "FDP")
	}

	// Check STP
	if device.STPConfig != nil && device.STPConfig.Enabled {
		protocolsEnabled = append(protocolsEnabled, "STP")
		if device.STPConfig.BridgePriority > 61440 {
			v.addError("stp.bridge_priority", deviceName, fmt.Sprintf("STP bridge priority %d exceeds maximum 61440", device.STPConfig.BridgePriority))
		}
		if device.STPConfig.BridgePriority%4096 != 0 && device.STPConfig.BridgePriority != 0 {
			v.addWarning("stp.bridge_priority", deviceName, fmt.Sprintf("STP bridge priority %d should be a multiple of 4096", device.STPConfig.BridgePriority))
		}
	}

	// Check HTTP
	if device.HTTPConfig != nil && device.HTTPConfig.Enabled {
		protocolsEnabled = append(protocolsEnabled, "HTTP")
		if len(device.HTTPConfig.Endpoints) > 0 {
			v.addInfo("http.endpoints", deviceName, fmt.Sprintf("%d HTTP endpoint(s) configured", len(device.HTTPConfig.Endpoints)))
		}
	}

	// Check FTP
	if device.FTPConfig != nil && device.FTPConfig.Enabled {
		protocolsEnabled = append(protocolsEnabled, "FTP")
		if len(device.FTPConfig.Users) > 0 {
			v.addInfo("ftp.users", deviceName, fmt.Sprintf("%d FTP user(s) configured", len(device.FTPConfig.Users)))
		}
	}

	// Check DNS
	if device.DNSConfig != nil {
		if len(device.DNSConfig.ForwardRecords) > 0 || len(device.DNSConfig.ReverseRecords) > 0 {
			protocolsEnabled = append(protocolsEnabled, "DNS")
			totalRecords := len(device.DNSConfig.ForwardRecords) + len(device.DNSConfig.ReverseRecords)
			v.addInfo("dns.records", deviceName, fmt.Sprintf("%d DNS record(s) configured", totalRecords))
		}
	}

	// Check DHCP (note: DHCP doesn't have Enabled field or Pools - it uses other fields)
	if device.DHCPConfig != nil {
		// Check if any DHCP options are configured
		if device.DHCPConfig.Router != nil || len(device.DHCPConfig.DomainNameServer) > 0 {
			protocolsEnabled = append(protocolsEnabled, "DHCP")
		}
	}

	// Check DHCPv6
	if device.DHCPv6Config != nil && device.DHCPv6Config.Enabled {
		protocolsEnabled = append(protocolsEnabled, "DHCPv6")
		if len(device.DHCPv6Config.Pools) == 0 {
			v.addWarning("dhcpv6.pools", deviceName, "DHCPv6 enabled but no pools configured")
		}
	}

	if len(protocolsEnabled) > 0 {
		v.addInfo("protocols", deviceName, fmt.Sprintf("Protocols enabled: %s", strings.Join(protocolsEnabled, ", ")))
	} else {
		v.addWarning("protocols", deviceName, "No protocols explicitly enabled")
	}
}

// validateTrafficConfig validates v1.6.0 traffic configuration
func (v *Validator) validateTrafficConfig(device *Device, deviceName string) {
	if device.TrafficConfig != nil && device.TrafficConfig.Enabled {
		v.addInfo("traffic", deviceName, "Traffic generation enabled")

		if device.TrafficConfig.ARPAnnouncements != nil && device.TrafficConfig.ARPAnnouncements.Enabled {
			if device.TrafficConfig.ARPAnnouncements.Interval <= 0 {
				v.addWarning("traffic.arp.interval", deviceName, "ARP announcement interval <= 0, will use default")
			}
		}

		if device.TrafficConfig.PeriodicPings != nil && device.TrafficConfig.PeriodicPings.Enabled {
			if device.TrafficConfig.PeriodicPings.Interval <= 0 {
				v.addWarning("traffic.ping.interval", deviceName, "Ping interval <= 0, will use default")
			}
			if device.TrafficConfig.PeriodicPings.PayloadSize > 1400 {
				v.addWarning("traffic.ping.payload_size", deviceName, fmt.Sprintf("Large ping payload size (%d bytes) may cause fragmentation", device.TrafficConfig.PeriodicPings.PayloadSize))
			}
		}

		if device.TrafficConfig.RandomTraffic != nil && device.TrafficConfig.RandomTraffic.Enabled {
			if device.TrafficConfig.RandomTraffic.PacketCount > 1000 {
				v.addWarning("traffic.random.packet_count", deviceName, fmt.Sprintf("High random traffic packet count (%d) may impact performance", device.TrafficConfig.RandomTraffic.PacketCount))
			}
		}
	}
}

// validateTrapConfig validates v1.6.0 SNMP trap configuration
func (v *Validator) validateTrapConfig(device *Device, deviceName string) {
	trapConfig := device.SNMPConfig.Traps
	if trapConfig != nil && trapConfig.Enabled {
		v.addInfo("snmp.traps", deviceName, "SNMP traps enabled")

		if len(trapConfig.Receivers) == 0 {
			v.addError("snmp.traps.receivers", deviceName, "SNMP traps enabled but no receivers configured")
		} else {
			v.addInfo("snmp.traps.receivers", deviceName, fmt.Sprintf("%d trap receiver(s) configured", len(trapConfig.Receivers)))

			// Validate receiver addresses
			for i, receiver := range trapConfig.Receivers {
				host, port, err := net.SplitHostPort(receiver)
				if err != nil {
					// Assume port 162 if not specified
					host = receiver
				}

				// Validate host
				if net.ParseIP(host) == nil {
					// Not an IP, might be hostname - just warn
					v.addWarning("snmp.traps.receivers", deviceName, fmt.Sprintf("Receiver %d (%s) is not an IP address (hostname lookups may fail)", i, host))
				}

				// Check port if specified
				if port != "" {
					var portNum uint16
					if _, err := fmt.Sscanf(port, "%d", &portNum); err != nil || portNum == 0 {
						v.addError("snmp.traps.receivers", deviceName, fmt.Sprintf("Receiver %d has invalid port: %s", i, port))
					}
				}
			}
		}

		// Check threshold configurations
		if trapConfig.HighCPU != nil && trapConfig.HighCPU.Enabled {
			if trapConfig.HighCPU.Threshold > 100 {
				v.addError("snmp.traps.high_cpu.threshold", deviceName, fmt.Sprintf("CPU threshold %d exceeds 100%%", trapConfig.HighCPU.Threshold))
			}
			if trapConfig.HighCPU.Threshold < 50 {
				v.addWarning("snmp.traps.high_cpu.threshold", deviceName, fmt.Sprintf("CPU threshold %d is quite low, may generate many traps", trapConfig.HighCPU.Threshold))
			}
		}

		if trapConfig.HighMemory != nil && trapConfig.HighMemory.Enabled {
			if trapConfig.HighMemory.Threshold > 100 {
				v.addError("snmp.traps.high_memory.threshold", deviceName, fmt.Sprintf("Memory threshold %d exceeds 100%%", trapConfig.HighMemory.Threshold))
			}
		}
	}
}

// addError adds an error to the validation result
func (v *Validator) addError(field, device, message string) {
	v.result.Errors = append(v.result.Errors, ValidationError{
		Field:   field,
		Device:  device,
		Message: message,
		Level:   "error",
	})
	v.result.Valid = false
}

// addWarning adds a warning to the validation result
func (v *Validator) addWarning(field, device, message string) {
	v.result.Warnings = append(v.result.Warnings, ValidationError{
		Field:   field,
		Device:  device,
		Message: message,
		Level:   "warning",
	})
}

// addInfo adds an info message to the validation result
func (v *Validator) addInfo(field, device, message string) {
	v.result.Info = append(v.result.Info, ValidationError{
		Field:   field,
		Device:  device,
		Message: message,
		Level:   "info",
	})
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// FormatValidationResult formats a validation result for display
func FormatValidationResult(result *ValidationResult, verbose bool) string {
	var output strings.Builder

	// Show errors
	if len(result.Errors) > 0 {
		output.WriteString("\n❌ Errors found:\n")
		for _, err := range result.Errors {
			if err.Device != "" {
				output.WriteString(fmt.Sprintf("  • [%s] %s: %s\n", err.Device, err.Field, err.Message))
			} else {
				output.WriteString(fmt.Sprintf("  • %s: %s\n", err.Field, err.Message))
			}
		}
	}

	// Show warnings
	if len(result.Warnings) > 0 {
		output.WriteString("\n⚠️  Warnings:\n")
		for _, warn := range result.Warnings {
			if warn.Device != "" {
				output.WriteString(fmt.Sprintf("  • [%s] %s: %s\n", warn.Device, warn.Field, warn.Message))
			} else {
				output.WriteString(fmt.Sprintf("  • %s: %s\n", warn.Field, warn.Message))
			}
		}
	}

	// Show info if verbose
	if verbose && len(result.Info) > 0 {
		output.WriteString("\nℹ️  Configuration details:\n")
		for _, info := range result.Info {
			if info.Device != "" {
				output.WriteString(fmt.Sprintf("  • [%s] %s\n", info.Device, info.Message))
			} else {
				output.WriteString(fmt.Sprintf("  • %s\n", info.Message))
			}
		}
	}

	// Summary
	output.WriteString("\n")
	if result.Valid {
		output.WriteString("✅ Configuration is valid")
		if len(result.Warnings) > 0 {
			output.WriteString(fmt.Sprintf(" (%d warning(s))", len(result.Warnings)))
		}
		output.WriteString("\n")
	} else {
		output.WriteString(fmt.Sprintf("❌ Configuration is invalid (%d error(s), %d warning(s))\n", len(result.Errors), len(result.Warnings)))
	}

	return output.String()
}
