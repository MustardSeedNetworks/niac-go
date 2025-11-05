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
	Devices            []Device
	IncludePath        string              // Base path for walk files
	CapturePlayback    *CapturePlayback    // Optional PCAP playback config
	DiscoveryProtocols *DiscoveryProtocols // Discovery protocol configuration
}

// CapturePlayback represents PCAP file playback configuration
type CapturePlayback struct {
	FileName  string
	LoopTime  int     // milliseconds
	ScaleTime float64 // time scaling factor
}

// DiscoveryProtocols configures discovery protocol behavior
type DiscoveryProtocols struct {
	LLDP *ProtocolConfig
	CDP  *ProtocolConfig
	EDP  *ProtocolConfig
	FDP  *ProtocolConfig
}

// ProtocolConfig configures a discovery protocol
type ProtocolConfig struct {
	Enabled  bool
	Interval int // Advertisement interval in seconds
}

// Device represents a simulated network device
type Device struct {
	Name        string
	Type        string // router, switch, ap, etc.
	MACAddress  net.HardwareAddr
	IPAddresses []net.IP
	Interfaces  []Interface
	SNMPConfig  SNMPConfig
	DHCPConfig  *DHCPConfig // DHCP server configuration
	DNSConfig   *DNSConfig  // DNS server configuration
	LLDPConfig  *LLDPConfig // LLDP discovery protocol configuration
	CDPConfig   *CDPConfig  // CDP discovery protocol configuration
	EDPConfig   *EDPConfig  // EDP discovery protocol configuration
	FDPConfig   *FDPConfig  // FDP discovery protocol configuration
	STPConfig   *STPConfig  // STP/RSTP/MSTP configuration
	Properties  map[string]string
}

// DHCPConfig holds DHCP server configuration for a device
type DHCPConfig struct {
	// Basic DHCPv4 options
	SubnetMask       net.IPMask
	Router           net.IP
	DomainNameServer []net.IP
	ServerIdentifier net.IP
	NextServerIP     net.IP
	DomainName       string

	// DHCPv4 high priority options
	NTPServers     []net.IP
	DomainSearch   []string
	TFTPServerName string
	BootfileName   string
	VendorSpecific []byte // Hex-encoded vendor-specific data

	// DHCPv6 options
	SNTPServersV6 []net.IP
	NTPServersV6  []net.IP
	SIPServersV6  []net.IP
	SIPDomainsV6  []string

	// Static leases
	ClientLeases []DHCPLease
}

// DHCPLease represents a static DHCP lease assignment
type DHCPLease struct {
	ClientIP    net.IP
	MACAddress  net.HardwareAddr
	MACMask     net.HardwareAddr // For wildcard matching
}

// DNSConfig holds DNS server configuration for a device
type DNSConfig struct {
	ForwardRecords []DNSRecord
	ReverseRecords []DNSRecord
}

// DNSRecord represents a DNS A or PTR record
type DNSRecord struct {
	Name string
	IP   net.IP
	TTL  uint32
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

// LLDPConfig holds LLDP (Link Layer Discovery Protocol) configuration
type LLDPConfig struct {
	Enabled           bool
	AdvertiseInterval int    // seconds
	TTL               int    // seconds
	SystemDescription string
	PortDescription   string
	ChassisIDType     string // "mac", "local", "network_address"
}

// CDPConfig holds CDP (Cisco Discovery Protocol) configuration
type CDPConfig struct {
	Enabled          bool
	AdvertiseInterval int    // seconds
	Holdtime         int    // seconds
	Version          int    // 1 or 2
	SoftwareVersion  string
	Platform         string
	PortID           string
}

// EDPConfig holds EDP (Extreme Discovery Protocol) configuration
type EDPConfig struct {
	Enabled           bool
	AdvertiseInterval int    // seconds
	VersionString     string
	DisplayString     string
}

// FDPConfig holds FDP (Foundry Discovery Protocol) configuration
type FDPConfig struct {
	Enabled          bool
	AdvertiseInterval int    // seconds
	Holdtime         int    // seconds
	SoftwareVersion  string
	Platform         string
	PortID           string
}

// STPConfig holds STP (Spanning Tree Protocol) configuration
type STPConfig struct {
	Enabled        bool
	BridgePriority uint16 // 0-61440 in increments of 4096 (default: 32768)
	HelloTime      uint16 // seconds (default: 2)
	MaxAge         uint16 // seconds (default: 20)
	ForwardDelay   uint16 // seconds (default: 15)
	Version        string // "stp", "rstp", "mstp" (default: "stp")
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

	// Copy CapturePlayback if present (use first one from array for now)
	// TODO: Support multiple PCAP playbacks in runtime
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

		// Parse IP address(es)
		// Support both singular 'ip' (backward compatible) and plural 'ips' (new feature)
		if yamlDevice.IP != "" {
			ip := net.ParseIP(yamlDevice.IP)
			if ip == nil {
				return nil, fmt.Errorf("device %s: invalid IP address %s", yamlDevice.Name, yamlDevice.IP)
			}
			device.IPAddresses = append(device.IPAddresses, ip)
		}

		// Parse multiple IPs if specified
		for i, ipStr := range yamlDevice.IPs {
			ip := net.ParseIP(ipStr)
			if ip == nil {
				return nil, fmt.Errorf("device %s: invalid IP address in ips[%d]: %s", yamlDevice.Name, i, ipStr)
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

		// Handle DHCP configuration
		if yamlDevice.Dhcp != nil {
			dhcpCfg := &DHCPConfig{}

			// Basic options
			if yamlDevice.Dhcp.SubnetMask != "" {
				if ip := net.ParseIP(yamlDevice.Dhcp.SubnetMask); ip != nil {
					dhcpCfg.SubnetMask = net.IPMask(ip.To4())
				}
			}
			if yamlDevice.Dhcp.Router != "" {
				dhcpCfg.Router = net.ParseIP(yamlDevice.Dhcp.Router)
			}
			if yamlDevice.Dhcp.DomainNameServer != "" {
				if ip := net.ParseIP(yamlDevice.Dhcp.DomainNameServer); ip != nil {
					dhcpCfg.DomainNameServer = append(dhcpCfg.DomainNameServer, ip)
				}
			}
			if yamlDevice.Dhcp.ServerIdentifier != "" {
				dhcpCfg.ServerIdentifier = net.ParseIP(yamlDevice.Dhcp.ServerIdentifier)
			}
			if yamlDevice.Dhcp.NextServerIP != "" {
				dhcpCfg.NextServerIP = net.ParseIP(yamlDevice.Dhcp.NextServerIP)
			}
			// Domain name is separate from domain name server
			// Note: YAML doesn't have a separate domain_name field yet, so we'll leave this empty for now

			// DHCPv4 high priority options
			for _, ntpStr := range yamlDevice.Dhcp.NTPServers {
				if ip := net.ParseIP(ntpStr); ip != nil {
					dhcpCfg.NTPServers = append(dhcpCfg.NTPServers, ip)
				}
			}
			dhcpCfg.DomainSearch = yamlDevice.Dhcp.DomainSearch
			dhcpCfg.TFTPServerName = yamlDevice.Dhcp.TFTPServerName
			dhcpCfg.BootfileName = yamlDevice.Dhcp.BootfileName
			if yamlDevice.Dhcp.VendorSpecific != "" {
				// Parse hex string to bytes
				dhcpCfg.VendorSpecific = []byte(yamlDevice.Dhcp.VendorSpecific)
			}

			// DHCPv6 options
			for _, sntpStr := range yamlDevice.Dhcp.SNTPServersV6 {
				if ip := net.ParseIP(sntpStr); ip != nil {
					dhcpCfg.SNTPServersV6 = append(dhcpCfg.SNTPServersV6, ip)
				}
			}
			for _, ntpStr := range yamlDevice.Dhcp.NTPServersV6 {
				if ip := net.ParseIP(ntpStr); ip != nil {
					dhcpCfg.NTPServersV6 = append(dhcpCfg.NTPServersV6, ip)
				}
			}
			for _, sipStr := range yamlDevice.Dhcp.SIPServersV6 {
				if ip := net.ParseIP(sipStr); ip != nil {
					dhcpCfg.SIPServersV6 = append(dhcpCfg.SIPServersV6, ip)
				}
			}
			dhcpCfg.SIPDomainsV6 = yamlDevice.Dhcp.SIPDomainsV6

			// Static leases
			for _, lease := range yamlDevice.Dhcp.ClientLeases {
				clientIP := net.ParseIP(lease.ClientIP)
				if clientIP == nil {
					continue
				}
				macAddr, err := net.ParseMAC(lease.MacAddrValue)
				if err != nil {
					continue
				}
				dhcpLease := DHCPLease{
					ClientIP:   clientIP,
					MACAddress: macAddr,
				}
				if lease.MacAddrMask != "" {
					if mask, err := net.ParseMAC(lease.MacAddrMask); err == nil {
						dhcpLease.MACMask = mask
					}
				}
				dhcpCfg.ClientLeases = append(dhcpCfg.ClientLeases, dhcpLease)
			}

			device.DHCPConfig = dhcpCfg
		}

		// Handle DNS configuration
		if yamlDevice.Dns != nil {
			dnsCfg := &DNSConfig{}

			// Forward records (A records)
			for _, record := range yamlDevice.Dns.ForwardRecords {
				ip := net.ParseIP(record.IP)
				if ip == nil {
					continue
				}
				ttl := uint32(3600) // Default TTL
				if record.TTL > 0 {
					ttl = uint32(record.TTL)
				}
				dnsCfg.ForwardRecords = append(dnsCfg.ForwardRecords, DNSRecord{
					Name: record.Name,
					IP:   ip,
					TTL:  ttl,
				})
			}

			// Reverse records (PTR records)
			for _, record := range yamlDevice.Dns.ReverseRecords {
				ip := net.ParseIP(record.IP)
				if ip == nil {
					continue
				}
				ttl := uint32(3600) // Default TTL
				if record.TTL > 0 {
					ttl = uint32(record.TTL)
				}
				dnsCfg.ReverseRecords = append(dnsCfg.ReverseRecords, DNSRecord{
					Name: record.Name,
					IP:   ip,
					TTL:  ttl,
				})
			}

			device.DNSConfig = dnsCfg
		}

		// Handle LLDP configuration
		if yamlDevice.Lldp != nil {
			lldpCfg := &LLDPConfig{
				Enabled:           yamlDevice.Lldp.Enabled,
				AdvertiseInterval: yamlDevice.Lldp.AdvertiseInterval,
				TTL:               yamlDevice.Lldp.TTL,
				SystemDescription: yamlDevice.Lldp.SystemDescription,
				PortDescription:   yamlDevice.Lldp.PortDescription,
				ChassisIDType:     yamlDevice.Lldp.ChassisIDType,
			}
			// Set defaults if not specified
			if lldpCfg.AdvertiseInterval == 0 {
				lldpCfg.AdvertiseInterval = 30
			}
			if lldpCfg.TTL == 0 {
				lldpCfg.TTL = 120
			}
			if lldpCfg.ChassisIDType == "" {
				lldpCfg.ChassisIDType = "mac"
			}
			device.LLDPConfig = lldpCfg
		}

		// Handle CDP configuration
		if yamlDevice.Cdp != nil {
			cdpCfg := &CDPConfig{
				Enabled:           yamlDevice.Cdp.Enabled,
				AdvertiseInterval: yamlDevice.Cdp.AdvertiseInterval,
				Holdtime:          yamlDevice.Cdp.Holdtime,
				Version:           yamlDevice.Cdp.Version,
				SoftwareVersion:   yamlDevice.Cdp.SoftwareVersion,
				Platform:          yamlDevice.Cdp.Platform,
				PortID:            yamlDevice.Cdp.PortID,
			}
			// Set defaults if not specified
			if cdpCfg.AdvertiseInterval == 0 {
				cdpCfg.AdvertiseInterval = 60
			}
			if cdpCfg.Holdtime == 0 {
				cdpCfg.Holdtime = 180
			}
			if cdpCfg.Version == 0 {
				cdpCfg.Version = 2
			}
			device.CDPConfig = cdpCfg
		}

		// Handle EDP configuration
		if yamlDevice.Edp != nil {
			edpCfg := &EDPConfig{
				Enabled:           yamlDevice.Edp.Enabled,
				AdvertiseInterval: yamlDevice.Edp.AdvertiseInterval,
				VersionString:     yamlDevice.Edp.VersionString,
				DisplayString:     yamlDevice.Edp.DisplayString,
			}
			// Set defaults if not specified
			if edpCfg.AdvertiseInterval == 0 {
				edpCfg.AdvertiseInterval = 30
			}
			device.EDPConfig = edpCfg
		}

		// Handle FDP configuration
		if yamlDevice.Fdp != nil {
			fdpCfg := &FDPConfig{
				Enabled:           yamlDevice.Fdp.Enabled,
				AdvertiseInterval: yamlDevice.Fdp.AdvertiseInterval,
				Holdtime:          yamlDevice.Fdp.Holdtime,
				SoftwareVersion:   yamlDevice.Fdp.SoftwareVersion,
				Platform:          yamlDevice.Fdp.Platform,
				PortID:            yamlDevice.Fdp.PortID,
			}
			// Set defaults if not specified
			if fdpCfg.AdvertiseInterval == 0 {
				fdpCfg.AdvertiseInterval = 60
			}
			if fdpCfg.Holdtime == 0 {
				fdpCfg.Holdtime = 180
			}
			device.FDPConfig = fdpCfg
		}

		// Handle STP configuration
		if yamlDevice.Stp != nil {
			stpCfg := &STPConfig{
				Enabled:        yamlDevice.Stp.Enabled,
				BridgePriority: yamlDevice.Stp.BridgePriority,
				HelloTime:      yamlDevice.Stp.HelloTime,
				MaxAge:         yamlDevice.Stp.MaxAge,
				ForwardDelay:   yamlDevice.Stp.ForwardDelay,
				Version:        yamlDevice.Stp.Version,
			}
			// Set defaults if not specified
			if stpCfg.BridgePriority == 0 {
				stpCfg.BridgePriority = 32768 // Default priority
			}
			if stpCfg.HelloTime == 0 {
				stpCfg.HelloTime = 2 // Default hello time
			}
			if stpCfg.MaxAge == 0 {
				stpCfg.MaxAge = 20 // Default max age
			}
			if stpCfg.ForwardDelay == 0 {
				stpCfg.ForwardDelay = 15 // Default forward delay
			}
			if stpCfg.Version == "" {
				stpCfg.Version = "stp" // Default to STP
			}
			device.STPConfig = stpCfg
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
