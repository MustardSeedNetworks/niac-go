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

// Config represents the network configuration
type Config struct {
	Devices         []Device
	IncludePath     string           // Base path for walk files
	CapturePlayback *CapturePlayback // Optional PCAP playback config
}

// CapturePlayback represents PCAP file playback configuration
type CapturePlayback struct {
	FileName  string
	LoopTime  int     // milliseconds
	ScaleTime float64 // time scaling factor
}

// Device represents a simulated network device
type Device struct {
	Name        string
	Type        string // router, switch, ap, etc.
	MACAddress  net.HardwareAddr
	IPAddresses []net.IP
	Interfaces  []Interface
	SNMPConfig  SNMPConfig
	Properties  map[string]string
}

// Interface represents a network interface on a device
type Interface struct {
	Name      string
	Speed     int // Mbps
	Duplex    string
	AdminStatus  string // up, down
	OperStatus   string // up, down, testing
	Description string
	VLANs     []int
}

// SNMPConfig holds SNMP configuration
type SNMPConfig struct {
	Community  string
	SysName    string
	SysDescr   string
	SysContact string
	SysLocation string
	WalkFile   string // Path to SNMP walk file
}

// Load reads and parses a configuration file
// Automatically detects format based on file extension:
// - .yaml -> YAML format (converted from Java DSL)
// - .cfg, .conf, or other -> legacy key-value format
func Load(filename string) (*Config, error) {
	ext := filepath.Ext(filename)

	// Route to YAML loader for .yaml files
	if ext == ".yaml" || ext == ".yml" {
		return LoadYAML(filename)
	}

	// Legacy format loader
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	cfg := &Config{
		Devices: make([]Device, 0),
	}

	var currentDevice *Device
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		// Parse device declaration
		if strings.HasPrefix(line, "device ") {
			parts := strings.Fields(line)
			if len(parts) < 2 {
				return nil, fmt.Errorf("line %d: invalid device declaration", lineNum)
			}

			device := Device{
				Name:       parts[1],
				Interfaces: make([]Interface, 0),
				Properties: make(map[string]string),
			}
			cfg.Devices = append(cfg.Devices, device)
			currentDevice = &cfg.Devices[len(cfg.Devices)-1]
			continue
		}

		// Parse device properties
		if currentDevice != nil {
			if strings.HasPrefix(line, "}") {
				currentDevice = nil
				continue
			}

			// Parse key-value pairs
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.Trim(strings.TrimSpace(parts[1]), "\"")

				switch key {
				case "type":
					currentDevice.Type = value
				case "mac":
					mac, err := net.ParseMAC(value)
					if err == nil {
						currentDevice.MACAddress = mac
					}
				case "ip", "ipv6":
					// Both "ip" and "ipv6" work - net.ParseIP handles both IPv4 and IPv6
					ip := net.ParseIP(value)
					if ip != nil {
						currentDevice.IPAddresses = append(currentDevice.IPAddresses, ip)
					}
				case "snmp_community":
					currentDevice.SNMPConfig.Community = value
				case "sysName":
					currentDevice.SNMPConfig.SysName = value
				case "sysDescr":
					currentDevice.SNMPConfig.SysDescr = value
				case "sysContact":
					currentDevice.SNMPConfig.SysContact = value
				case "sysLocation":
					currentDevice.SNMPConfig.SysLocation = value
				case "walk":
					currentDevice.SNMPConfig.WalkFile = value
				default:
					currentDevice.Properties[key] = value
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Validate config
	if len(cfg.Devices) == 0 {
		return nil, fmt.Errorf("no devices defined in configuration")
	}

	return cfg, nil
}

// GetDeviceByMAC finds a device by MAC address
func (c *Config) GetDeviceByMAC(mac net.HardwareAddr) *Device {
	for i := range c.Devices {
		if c.Devices[i].MACAddress.String() == mac.String() {
			return &c.Devices[i]
		}
	}
	return nil
}

// GetDeviceByIP finds a device by IP address
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

// LoadYAML loads a YAML configuration file
func LoadYAML(filename string) (*Config, error) {
	// Load using converter package
	yamlConfig, err := converter.LoadYAMLConfig(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load YAML config: %w", err)
	}

	// Validate
	if err := converter.ValidateConfig(yamlConfig); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Convert to runtime config format
	cfg := &Config{
		Devices:     make([]Device, 0, len(yamlConfig.Devices)),
		IncludePath: yamlConfig.IncludePath,
	}

	// Copy CapturePlayback if present
	if yamlConfig.CapturePlayback != nil {
		cfg.CapturePlayback = &CapturePlayback{
			FileName:  yamlConfig.CapturePlayback.FileName,
			LoopTime:  yamlConfig.CapturePlayback.LoopTime,
			ScaleTime: yamlConfig.CapturePlayback.ScaleTime,
		}
	}

	// Convert devices
	for _, yamlDevice := range yamlConfig.Devices {
		device := Device{
			Name:        yamlDevice.Name,
			Type:        "unknown", // Default type
			Interfaces:  make([]Interface, 0),
			Properties:  make(map[string]string),
			SNMPConfig: SNMPConfig{
				Community: "public", // Default
				SysName:   yamlDevice.Name,
			},
		}

		// Parse MAC address
		if yamlDevice.MAC != "" {
			mac, err := net.ParseMAC(yamlDevice.MAC)
			if err != nil {
				return nil, fmt.Errorf("device %s: invalid MAC address %s: %w", yamlDevice.Name, yamlDevice.MAC, err)
			}
			device.MACAddress = mac
		}

		// Parse IP address
		if yamlDevice.IP != "" {
			ip := net.ParseIP(yamlDevice.IP)
			if ip == nil {
				return nil, fmt.Errorf("device %s: invalid IP address %s", yamlDevice.Name, yamlDevice.IP)
			}
			device.IPAddresses = append(device.IPAddresses, ip)
		}

		// Handle SNMP configuration
		if yamlDevice.SnmpAgent != nil {
			if yamlDevice.SnmpAgent.WalkFile != "" {
				// Resolve walk file path relative to include path
				walkFile := yamlDevice.SnmpAgent.WalkFile
				if cfg.IncludePath != "" && !filepath.IsAbs(walkFile) {
					walkFile = filepath.Join(cfg.IncludePath, walkFile)
				}
				device.SNMPConfig.WalkFile = walkFile
			}

			// TODO: Handle AddMibs - requires SNMP MIB storage
			// For now, store in properties for future use
			if len(yamlDevice.SnmpAgent.AddMibs) > 0 {
				device.Properties["custom_mibs_count"] = fmt.Sprintf("%d", len(yamlDevice.SnmpAgent.AddMibs))
			}
		}

		// Store VLAN if present
		if yamlDevice.VLAN > 0 {
			device.Properties["vlan"] = fmt.Sprintf("%d", yamlDevice.VLAN)
		}

		cfg.Devices = append(cfg.Devices, device)
	}

	// Validate final config
	if len(cfg.Devices) == 0 {
		return nil, fmt.Errorf("no devices defined in configuration")
	}

	return cfg, nil
}

// ParseSimpleConfig parses a simple device configuration format
// Format: DeviceName Type IP MAC [walkfile]
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
		if len(parts) < 4 {
			return nil, fmt.Errorf("line %d: insufficient fields", lineNum+1)
		}

		mac, err := net.ParseMAC(parts[3])
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid MAC address: %w", lineNum+1, err)
		}

		ip := net.ParseIP(parts[2])
		if ip == nil {
			return nil, fmt.Errorf("line %d: invalid IP address", lineNum+1)
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

		if len(parts) >= 5 {
			device.SNMPConfig.WalkFile = parts[4]
		}

		cfg.Devices = append(cfg.Devices, device)
	}

	return cfg, nil
}

// GenerateMAC generates a random MAC address
func GenerateMAC() net.HardwareAddr {
	mac := make(net.HardwareAddr, 6)
	// Set locally administered bit
	mac[0] = 0x02
	for i := 1; i < 6; i++ {
		mac[i] = byte(i * 17) // Simple pattern for testing
	}
	return mac
}

// ParseSpeed parses interface speed (e.g., "100M", "1G", "10G")
func ParseSpeed(speedStr string) (int, error) {
	speedStr = strings.ToUpper(strings.TrimSpace(speedStr))

	if strings.HasSuffix(speedStr, "G") {
		val, err := strconv.Atoi(strings.TrimSuffix(speedStr, "G"))
		if err != nil {
			return 0, err
		}
		return val * 1000, nil // Convert to Mbps
	}

	if strings.HasSuffix(speedStr, "M") {
		return strconv.Atoi(strings.TrimSuffix(speedStr, "M"))
	}

	return strconv.Atoi(speedStr)
}
