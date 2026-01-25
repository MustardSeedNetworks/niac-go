// Package config provides configuration file loading and parsing for network device simulation
package config

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/krisarmstrong/niac-go/internal/converter"
)

// Load reads and parses a configuration file
// Automatically detects format based on file extension:
// - .yaml -> YAML format (converted from Java DSL)
// - .cfg, .conf, or other -> legacy key-value format.
func Load(filename string) (*Config, error) {
	ext := filepath.Ext(filename)

	// Route to YAML loader for .yaml files
	if ext == ".yaml" || ext == ".yml" {
		return LoadYAML(filename)
	}

	// Route to legacy format loader
	return LoadLegacy(filename)
}

// LoadLegacy loads a legacy key-value configuration file
// Format: device <name> { key = value ... }.
func LoadLegacy(filename string) (*Config, error) {
	file, err := os.Open(filepath.Clean(filename))
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}

	defer func() { _ = file.Close() }()

	cfg := &Config{
		Devices: make([]Device, 0),
	}

	var currentDevice *Device

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if isCommentOrEmpty(line) {
			continue
		}

		if strings.HasPrefix(line, "device ") {
			device, parseErr := parseDeviceDeclaration(line, lineNum)
			if parseErr != nil {
				return nil, parseErr
			}

			cfg.Devices = append(cfg.Devices, device)
			currentDevice = &cfg.Devices[len(cfg.Devices)-1]

			continue
		}

		if currentDevice == nil {
			continue
		}

		if strings.HasPrefix(line, "}") {
			currentDevice = nil

			continue
		}

		parseLegacyKeyValue(line, currentDevice)
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return nil, fmt.Errorf("error reading config file: %w", scanErr)
	}

	if len(cfg.Devices) == 0 {
		return nil, ErrNoDevicesDefined
	}

	return cfg, nil
}

// isCommentOrEmpty checks if a line is empty or a comment.
func isCommentOrEmpty(line string) bool {
	return line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//")
}

// parseDeviceDeclaration parses a device declaration line.
func parseDeviceDeclaration(line string, lineNum int) (Device, error) {
	parts := strings.Fields(line)
	if len(parts) < minDeviceDeclParts {
		return Device{}, fmt.Errorf("%w: line %d", ErrInvalidDeviceDeclaration, lineNum)
	}

	return Device{
		Name:       parts[1],
		Interfaces: make([]Interface, 0),
		Properties: make(map[string]string),
	}, nil
}

// parseLegacyKeyValue parses a key=value pair and applies it to the device.
func parseLegacyKeyValue(line string, device *Device) {
	parts := strings.SplitN(line, "=", keyValueParts)
	if len(parts) != keyValueParts {
		return
	}

	key := strings.TrimSpace(parts[0])
	value := strings.Trim(strings.TrimSpace(parts[1]), "\"")

	applyLegacyDeviceProperty(device, key, value)
}

// applyLegacyDeviceProperty applies a parsed key-value property to a device.
func applyLegacyDeviceProperty(device *Device, key, value string) {
	switch key {
	case "type":
		device.Type = value
	case ChassisIDTypeMAC:
		if mac, err := net.ParseMAC(value); err == nil {
			device.MACAddress = mac
		}
	case "ip", "ipv6":
		if ip := net.ParseIP(value); ip != nil {
			device.IPAddresses = append(device.IPAddresses, ip)
		}
	case "snmp_community":
		device.SNMPConfig.Community = value
	case "sysName":
		device.SNMPConfig.SysName = value
	case "sysDescr":
		device.SNMPConfig.SysDescr = value
	case "sysContact":
		device.SNMPConfig.SysContact = value
	case "sysLocation":
		device.SNMPConfig.SysLocation = value
	case "walk":
		device.SNMPConfig.WalkFile = value
	default:
		device.Properties[key] = value
	}
}

// GetDeviceByMAC finds a device by MAC address.
func (c *Config) GetDeviceByMAC(mac net.HardwareAddr) *Device {
	for i := range c.Devices {
		if c.Devices[i].MACAddress.String() == mac.String() {
			return &c.Devices[i]
		}
	}

	return nil
}

// GetDeviceByIP finds a device by IP address.
func (c *Config) GetDeviceByIP(ip net.IP) *Device {
	for i := range c.Devices {
		for _, deviceIP := range c.Devices[i].IPAddresses {
			if deviceIP.Equal(ip) {
				return &c.Devices[i]
			}
		}
	}

	return nil
}

// LoadYAML loads a YAML configuration file.
func LoadYAML(filename string) (*Config, error) {
	yamlConfig, err := loadYAMLFile(filename)
	if err != nil {
		return nil, err
	}

	return buildConfigFromYAML(yamlConfig)
}

// LoadYAMLBytes builds a runtime config from in-memory YAML data.
func LoadYAMLBytes(data []byte) (*Config, error) {
	yamlConfig, err := loadYAMLBytes(data)
	if err != nil {
		return nil, err
	}

	return buildConfigFromYAML(yamlConfig)
}

// loadYAMLFile loads and validates a YAML configuration file.
func loadYAMLFile(filename string) (*converter.Config, error) {
	yamlConfig, err := converter.LoadYAMLConfig(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load YAML config: %w", err)
	}

	return validateYAMLConfig(yamlConfig)
}

func loadYAMLBytes(data []byte) (*converter.Config, error) {
	yamlConfig, err := converter.LoadYAMLConfigFromBytes(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return validateYAMLConfig(yamlConfig)
}

func buildConfigFromYAML(yamlConfig *converter.Config) (*Config, error) {
	cfg := createBaseConfig(yamlConfig)

	for _, yamlDevice := range yamlConfig.Devices {
		device, err := convertYAMLDevice(yamlDevice, cfg.IncludePath)
		if err != nil {
			return nil, err
		}

		cfg.Devices = append(cfg.Devices, device)
	}

	if len(cfg.Devices) == 0 {
		return nil, ErrNoDevicesDefined
	}

	return cfg, nil
}

// createBaseConfig creates the base configuration with global settings.
func createBaseConfig(yamlConfig *converter.Config) *Config {
	cfg := &Config{
		Devices:     make([]Device, 0, len(yamlConfig.Devices)),
		IncludePath: yamlConfig.IncludePath,
	}

	// Copy CapturePlayback if present (use first one from array for now)
	if len(yamlConfig.CapturePlaybacks) > 0 {
		cfg.CapturePlayback = &CapturePlayback{
			FileName:  yamlConfig.CapturePlaybacks[0].FileName,
			LoopTime:  yamlConfig.CapturePlaybacks[0].LoopTime,
			ScaleTime: yamlConfig.CapturePlaybacks[0].ScaleTime,
		}
	}

	// Copy DiscoveryProtocols if present
	if yamlConfig.DiscoveryProtocols != nil {
		cfg.DiscoveryProtocols = &DiscoveryProtocols{}

		if yamlConfig.DiscoveryProtocols.LLDP != nil {
			cfg.DiscoveryProtocols.LLDP = &ProtocolConfig{
				Enabled:  yamlConfig.DiscoveryProtocols.LLDP.Enabled,
				Interval: yamlConfig.DiscoveryProtocols.LLDP.Interval,
			}
		}

		if yamlConfig.DiscoveryProtocols.CDP != nil {
			cfg.DiscoveryProtocols.CDP = &ProtocolConfig{
				Enabled:  yamlConfig.DiscoveryProtocols.CDP.Enabled,
				Interval: yamlConfig.DiscoveryProtocols.CDP.Interval,
			}
		}

		if yamlConfig.DiscoveryProtocols.EDP != nil {
			cfg.DiscoveryProtocols.EDP = &ProtocolConfig{
				Enabled:  yamlConfig.DiscoveryProtocols.EDP.Enabled,
				Interval: yamlConfig.DiscoveryProtocols.EDP.Interval,
			}
		}

		if yamlConfig.DiscoveryProtocols.FDP != nil {
			cfg.DiscoveryProtocols.FDP = &ProtocolConfig{
				Enabled:  yamlConfig.DiscoveryProtocols.FDP.Enabled,
				Interval: yamlConfig.DiscoveryProtocols.FDP.Interval,
			}
		}
	}

	return cfg
}

// ParseSimpleConfig parses a simple device configuration format
// Format: DeviceName Type IP MAC [walkfile].
func ParseSimpleConfig(lines []string) (*Config, error) {
	cfg := &Config{
		Devices: make([]Device, 0),
	}

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < minSimpleConfigParts {
			return nil, fmt.Errorf("%w: line %d", ErrInsufficientFields, lineNum+1)
		}

		mac, err := net.ParseMAC(parts[3])
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid MAC address: %w", lineNum+1, err)
		}

		ip := net.ParseIP(parts[2])
		if ip == nil {
			return nil, fmt.Errorf("%w: line %d", ErrInvalidIPAddress, lineNum+1)
		}

		device := Device{
			Name:        parts[0],
			Type:        parts[1],
			MACAddress:  mac,
			IPAddresses: []net.IP{ip},
			Properties:  make(map[string]string),
			SNMPConfig: SNMPConfig{
				Community: "public",
				SysName:   parts[0],
			},
		}

		if len(parts) >= simpleConfigWithWalk {
			device.SNMPConfig.WalkFile = parts[4]
		}

		cfg.Devices = append(cfg.Devices, device)
	}

	return cfg, nil
}

// GenerateMAC generates a random MAC address.
func GenerateMAC() net.HardwareAddr {
	mac := make(net.HardwareAddr, macAddressBytes)
	// Set locally administered bit
	mac[0] = 0x02
	for i := 1; i < 6; i++ {
		mac[i] = byte(i * macPatternMultiplier) // Simple pattern for testing
	}

	return mac
}

// ParseSpeed parses interface speed (e.g., "100M", "1G", "10G").
func ParseSpeed(speedStr string) (int, error) {
	speedStr = strings.ToUpper(strings.TrimSpace(speedStr))

	if val, found := strings.CutSuffix(speedStr, "G"); found {
		num, err := strconv.Atoi(val)
		if err != nil {
			return 0, fmt.Errorf("failed to parse speed value: %w", err)
		}

		return num * gbpsToMbps, nil // Convert to Mbps
	}

	if val, found := strings.CutSuffix(speedStr, "M"); found {
		num, err := strconv.Atoi(val)
		if err != nil {
			return 0, fmt.Errorf("failed to parse speed value: %w", err)
		}

		return num, nil
	}

	num, err := strconv.Atoi(speedStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse speed value: %w", err)
	}

	return num, nil
}
