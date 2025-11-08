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

// LLDP Chassis ID Type constants
const (
	ChassisIDTypeMAC            = "mac"
	ChassisIDTypeLocal          = "local"
	ChassisIDTypeNetworkAddress = "network_address"
)

// Default configuration values
const (
	// Discovery protocol defaults
	DefaultLLDPAdvertiseInterval = 30  // seconds
	DefaultLLDPTTL               = 120 // seconds
	DefaultCDPAdvertiseInterval  = 60  // seconds
	DefaultCDPHoldtime           = 180 // seconds
	DefaultCDPVersion            = 2
	DefaultEDPAdvertiseInterval  = 30  // seconds
	DefaultFDPAdvertiseInterval  = 60  // seconds
	DefaultFDPHoldtime           = 180 // seconds

	// STP defaults
	DefaultSTPBridgePriority = 32768 // Default priority
	DefaultSTPHelloTime      = 2     // seconds
	DefaultSTPMaxAge         = 20    // seconds
	DefaultSTPForwardDelay   = 15    // seconds

	// NetBIOS defaults
	DefaultNetBIOSTTL = 300 // 5 minutes in seconds

	// ICMP defaults
	DefaultICMPTTL        = 64 // Default TTL
	DefaultICMPv6HopLimit = 64 // Default hop limit (NDP uses 255)

	// DHCPv6 defaults
	DefaultDHCPv6PreferredLifetime = 604800  // 7 days in seconds
	DefaultDHCPv6ValidLifetime     = 2592000 // 30 days in seconds

	// Traffic pattern defaults
	DefaultARPAnnouncementInterval  = 60  // seconds
	DefaultPeriodicPingInterval     = 120 // seconds
	DefaultPeriodicPingPayloadSize  = 32  // bytes
	DefaultRandomTrafficInterval    = 180 // seconds
	DefaultRandomTrafficPacketCount = 5   // packets per interval

	// SNMP trap defaults
	DefaultHighCPUThreshold        = 80  // percent
	DefaultHighMemoryThreshold     = 90  // percent
	DefaultInterfaceErrorThreshold = 100 // error count
	DefaultTrapCheckInterval       = 300 // 5 minutes in seconds
	DefaultInterfaceErrorInterval  = 60  // 1 minute in seconds

	// DNS defaults
	DefaultDNSTTL = 3600 // 1 hour in seconds
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
	Name          string
	Type          string // router, switch, ap, etc.
	MACAddress    net.HardwareAddr
	IPAddresses   []net.IP
	Interfaces    []Interface
	SNMPConfig    SNMPConfig
	DHCPConfig    *DHCPConfig    // DHCP server configuration
	DNSConfig     *DNSConfig     // DNS server configuration
	LLDPConfig    *LLDPConfig    // LLDP discovery protocol configuration
	CDPConfig     *CDPConfig     // CDP discovery protocol configuration
	EDPConfig     *EDPConfig     // EDP discovery protocol configuration
	FDPConfig     *FDPConfig     // FDP discovery protocol configuration
	STPConfig     *STPConfig     // STP/RSTP/MSTP configuration
	HTTPConfig    *HTTPConfig    // HTTP server configuration
	FTPConfig     *FTPConfig     // FTP server configuration
	NetBIOSConfig *NetBIOSConfig // NetBIOS service configuration
	ICMPConfig    *ICMPConfig    // ICMP/ICMPv4 configuration
	ICMPv6Config  *ICMPv6Config  // ICMPv6 configuration
	DHCPv6Config  *DHCPv6Config  // DHCPv6 server configuration
	TrafficConfig *TrafficConfig // Traffic pattern configuration (v1.6.0)
	Properties    map[string]string
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

	// DHCPv4 Pool configuration
	PoolStart net.IP // Start of DHCP address pool
	PoolEnd   net.IP // End of DHCP address pool

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
	ClientIP   net.IP
	MACAddress net.HardwareAddr
	MACMask    net.HardwareAddr // For wildcard matching
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
	Name        string
	Speed       int // Mbps
	Duplex      string
	AdminStatus string // up, down
	OperStatus  string // up, down, testing
	Description string
	VLANs       []int
}

// SNMPConfig holds SNMP configuration
type SNMPConfig struct {
	Community   string
	SysName     string
	SysDescr    string
	SysContact  string
	SysLocation string
	WalkFile    string      // Path to SNMP walk file
	Traps       *TrapConfig // SNMP trap configuration (v1.6.0)
}

// LLDPConfig holds LLDP (Link Layer Discovery Protocol) configuration
type LLDPConfig struct {
	Enabled           bool
	AdvertiseInterval int // seconds
	TTL               int // seconds
	SystemDescription string
	PortDescription   string
	ChassisIDType     string // "mac", "local", "network_address"
}

// CDPConfig holds CDP (Cisco Discovery Protocol) configuration
type CDPConfig struct {
	Enabled           bool
	AdvertiseInterval int // seconds
	Holdtime          int // seconds
	Version           int // 1 or 2
	SoftwareVersion   string
	Platform          string
	PortID            string
}

// EDPConfig holds EDP (Extreme Discovery Protocol) configuration
type EDPConfig struct {
	Enabled           bool
	AdvertiseInterval int // seconds
	VersionString     string
	DisplayString     string
}

// FDPConfig holds FDP (Foundry Discovery Protocol) configuration
type FDPConfig struct {
	Enabled           bool
	AdvertiseInterval int // seconds
	Holdtime          int // seconds
	SoftwareVersion   string
	Platform          string
	PortID            string
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

// HTTPConfig holds HTTP server configuration
type HTTPConfig struct {
	Enabled    bool
	ServerName string         // Server header value (default: "NIAC-Go/1.0.0")
	Endpoints  []HTTPEndpoint // Custom endpoint definitions
}

// HTTPEndpoint defines a custom HTTP endpoint and response
type HTTPEndpoint struct {
	Path        string // URL path (e.g., "/", "/api/info")
	Method      string // HTTP method (default: "GET")
	StatusCode  int    // HTTP status code (default: 200)
	ContentType string // Content-Type header (default: "text/html")
	Body        string // Response body
}

// FTPConfig holds FTP server configuration
type FTPConfig struct {
	Enabled        bool
	WelcomeBanner  string    // Welcome message (default: "220 {devicename} FTP Server (NIAC-Go) ready.")
	SystemType     string    // System type string (default: "UNIX Type: L8")
	AllowAnonymous bool      // Allow anonymous login (default: true)
	Users          []FTPUser // User accounts
}

// FTPUser represents an FTP user account
type FTPUser struct {
	Username string
	Password string
	HomeDir  string // Virtual home directory path
}

// NetBIOSConfig holds NetBIOS service configuration
type NetBIOSConfig struct {
	Enabled   bool
	Name      string   // NetBIOS name (default: device name, max 15 chars)
	Workgroup string   // Workgroup/domain name (default: "WORKGROUP")
	NodeType  string   // Node type: "B" (broadcast), "P" (peer), "M" (mixed), "H" (hybrid) (default: "B")
	Services  []string // Service types to advertise (default: ["workstation", "fileserver"])
	TTL       uint32   // Name registration TTL in seconds (default: 300)
}

// ICMPConfig holds ICMP/ICMPv4 configuration
type ICMPConfig struct {
	Enabled   bool
	TTL       uint8 // Time to Live for ICMP packets (default: 64)
	RateLimit int   // Max ICMP responses per second (0 = unlimited, default: 0)
}

// ICMPv6Config holds ICMPv6 configuration
type ICMPv6Config struct {
	Enabled   bool
	HopLimit  uint8 // Hop limit for ICMPv6 packets (default: 64, NDP uses 255)
	RateLimit int   // Max ICMPv6 responses per second (0 = unlimited, default: 0)
}

// DHCPv6Config holds DHCPv6 server configuration
type DHCPv6Config struct {
	Enabled           bool
	Pools             []DHCPv6Pool // Address pools
	PreferredLifetime uint32       // Preferred lifetime in seconds (default: 604800 = 7 days)
	ValidLifetime     uint32       // Valid lifetime in seconds (default: 2592000 = 30 days)
	Preference        uint8        // Server preference (0-255, higher is better, default: 0)
	DNSServers        []net.IP     // DNS servers (IPv6)
	DomainList        []string     // Domain search list
	SNTPServers       []net.IP     // SNTP time servers (Option 31)
	NTPServers        []net.IP     // NTP servers (Option 56)
	SIPServers        []net.IP     // SIP server addresses (Option 22)
	SIPDomains        []string     // SIP domain names (Option 21)
}

// DHCPv6Pool represents an IPv6 address pool
type DHCPv6Pool struct {
	Network    string // IPv6 network (e.g., "2001:db8::/64")
	RangeStart string // Start of address range
	RangeEnd   string // End of address range
}

// TrafficConfig holds traffic pattern configuration (v1.6.0)
type TrafficConfig struct {
	Enabled          bool
	ARPAnnouncements *ARPAnnouncementConfig
	PeriodicPings    *PeriodicPingConfig
	RandomTraffic    *RandomTrafficConfig
}

// ARPAnnouncementConfig configures gratuitous ARP announcements
type ARPAnnouncementConfig struct {
	Enabled  bool
	Interval int // Interval in seconds (default: 60)
}

// PeriodicPingConfig configures periodic ICMP echo requests
type PeriodicPingConfig struct {
	Enabled     bool
	Interval    int // Interval in seconds (default: 120)
	PayloadSize int // Payload size in bytes (default: 32)
}

// RandomTrafficConfig configures random background traffic
type RandomTrafficConfig struct {
	Enabled     bool
	Interval    int      // Interval in seconds (default: 180)
	PacketCount int      // Number of packets per interval (default: 5)
	Patterns    []string // Traffic patterns: "broadcast_arp", "multicast", "udp"
}

// TrapConfig holds SNMP trap configuration (v1.6.0)
type TrapConfig struct {
	Enabled               bool
	Receivers             []string // Trap receiver addresses (IP:port format)
	Community             string   // SNMP community string (default: "public")
	ColdStart             *TrapTriggerConfig
	LinkState             *LinkStateTrapConfig
	AuthenticationFailure *TrapTriggerConfig
	HighCPU               *ThresholdTrapConfig
	HighMemory            *ThresholdTrapConfig
	InterfaceErrors       *ThresholdTrapConfig
}

// TrapTriggerConfig configures a simple trap trigger
type TrapTriggerConfig struct {
	Enabled   bool
	OnStartup bool // Send trap on device startup
}

// LinkStateTrapConfig configures link up/down traps
type LinkStateTrapConfig struct {
	Enabled  bool
	LinkDown bool // Send trap on link down
	LinkUp   bool // Send trap on link up
}

// ThresholdTrapConfig configures threshold-based traps
type ThresholdTrapConfig struct {
	Enabled   bool
	Threshold int // Threshold value (percent for CPU/Memory, count for errors)
	Interval  int // Check interval in seconds
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

	// Route to legacy format loader
	return LoadLegacy(filename)
}

// LoadLegacy loads a legacy key-value configuration file
// Format: device <name> { key = value ... }
func LoadLegacy(filename string) (*Config, error) {
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
	// Step 1: Load YAML file
	yamlConfig, err := loadYAMLFile(filename)
	if err != nil {
		return nil, err
	}

	// Step 2: Create base config with global settings
	cfg := createBaseConfig(yamlConfig)

	// Step 3: Convert devices
	for _, yamlDevice := range yamlConfig.Devices {
		device, err := convertYAMLDevice(yamlDevice, cfg.IncludePath)
		if err != nil {
			return nil, err
		}
		cfg.Devices = append(cfg.Devices, device)
	}

	// Step 4: Validate final config
	if len(cfg.Devices) == 0 {
		return nil, fmt.Errorf("no devices defined in configuration")
	}

	return cfg, nil
}

// loadYAMLFile loads and validates a YAML configuration file
func loadYAMLFile(filename string) (*converter.Config, error) {
	yamlConfig, err := converter.LoadYAMLConfig(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load YAML config: %w", err)
	}

	if err := converter.ValidateConfig(yamlConfig); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return yamlConfig, nil
}

// createBaseConfig creates the base configuration with global settings
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

// convertYAMLDevice converts a YAML device to a runtime Device
func convertYAMLDevice(yamlDevice converter.Device, includePath string) (Device, error) {
	device := Device{
		Name:       yamlDevice.Name,
		Type:       "unknown", // Default type
		Interfaces: make([]Interface, 0),
		Properties: make(map[string]string),
		SNMPConfig: SNMPConfig{
			Community: "public", // Default
			SysName:   yamlDevice.Name,
		},
	}

	// Parse MAC address
	if yamlDevice.MAC != "" {
		mac, err := net.ParseMAC(yamlDevice.MAC)
		if err != nil {
			return device, fmt.Errorf("device %s: invalid MAC address %s: %w", yamlDevice.Name, yamlDevice.MAC, err)
		}
		device.MACAddress = mac
	}

	// Parse IP addresses
	if err := parseDeviceIPAddresses(&device, &yamlDevice); err != nil {
		return device, err
	}

	// Handle SNMP configuration
	if err := parseDeviceSNMPConfig(&device, &yamlDevice, includePath); err != nil {
		return device, err
	}

	// Store VLAN if present
	if yamlDevice.VLAN > 0 {
		device.Properties["vlan"] = fmt.Sprintf("%d", yamlDevice.VLAN)
	}

	// Parse protocol configurations
	if err := parseDeviceProtocolConfigs(&device, &yamlDevice); err != nil {
		return device, err
	}

	return device, nil
}

// parseDeviceIPAddresses parses IP addresses for a device
func parseDeviceIPAddresses(device *Device, yamlDevice *converter.Device) error {
	// Support both singular 'ip' (backward compatible) and plural 'ips' (new feature)
	if yamlDevice.IP != "" {
		ip := net.ParseIP(yamlDevice.IP)
		if ip == nil {
			return fmt.Errorf("device %s: invalid IP address %s", yamlDevice.Name, yamlDevice.IP)
		}
		device.IPAddresses = append(device.IPAddresses, ip)
	}

	// Parse multiple IPs if specified
	for i, ipStr := range yamlDevice.IPs {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return fmt.Errorf("device %s: invalid IP address in ips[%d]: %s", yamlDevice.Name, i, ipStr)
		}
		device.IPAddresses = append(device.IPAddresses, ip)
	}

	return nil
}

// parseDeviceSNMPConfig parses SNMP configuration for a device
func parseDeviceSNMPConfig(device *Device, yamlDevice *converter.Device, includePath string) error {
	if yamlDevice.SnmpAgent != nil {
		if yamlDevice.SnmpAgent.WalkFile != "" {
			// Resolve and validate walk file path (security: prevent path traversal)
			walkFile, err := validateWalkFilePath(includePath, yamlDevice.SnmpAgent.WalkFile, yamlDevice.Name)
			if err != nil {
				return err
			}
			device.SNMPConfig.WalkFile = walkFile
		}

		// Store custom MIBs count for future use
		if len(yamlDevice.SnmpAgent.AddMibs) > 0 {
			device.Properties["custom_mibs_count"] = fmt.Sprintf("%d", len(yamlDevice.SnmpAgent.AddMibs))
		}

		// Parse SNMP Traps configuration
		if yamlDevice.SnmpAgent.Traps != nil {
			trapsCfg, err := parseSNMPTrapsConfig(yamlDevice.SnmpAgent.Traps)
			if err != nil {
				return err
			}
			device.SNMPConfig.Traps = trapsCfg
		}
	}

	return nil
}

// parseDeviceProtocolConfigs parses all protocol configurations for a device
func parseDeviceProtocolConfigs(device *Device, yamlDevice *converter.Device) error {
	var err error

	// Handle DHCP configuration
	if device.DHCPConfig, err = parseDHCPConfig(yamlDevice.Dhcp, yamlDevice.Name); err != nil {
		return err
	}

	// Handle DNS configuration
	if device.DNSConfig, err = parseDNSConfig(yamlDevice.Dns, yamlDevice.Name); err != nil {
		return err
	}

	// Handle discovery protocols
	device.LLDPConfig = parseLLDPConfig(yamlDevice.Lldp)
	device.CDPConfig = parseCDPConfig(yamlDevice.Cdp)
	device.EDPConfig = parseEDPConfig(yamlDevice.Edp)
	device.FDPConfig = parseFDPConfig(yamlDevice.Fdp)
	device.STPConfig = parseSTPConfig(yamlDevice.Stp)

	// Handle service protocols
	device.HTTPConfig = parseHTTPConfig(yamlDevice.Http, device.Name)
	device.FTPConfig = parseFTPConfig(yamlDevice.Ftp, device.Name)
	device.NetBIOSConfig = parseNetBIOSConfig(yamlDevice.Netbios, device.Name)

	// Handle ICMP protocols
	device.ICMPConfig = parseICMPConfig(yamlDevice.Icmp)
	device.ICMPv6Config = parseICMPv6Config(yamlDevice.Icmpv6)

	// Handle DHCPv6 configuration
	if device.DHCPv6Config, err = parseDHCPv6Config(yamlDevice.Dhcpv6); err != nil {
		return err
	}

	// Handle Traffic configuration
	device.TrafficConfig = parseTrafficConfig(yamlDevice.Traffic)

	return nil
}

// parseNetBIOSConfig parses NetBIOS configuration from YAML
func parseNetBIOSConfig(yamlNetbios *converter.NetbiosConfig, deviceName string) *NetBIOSConfig {
	if yamlNetbios == nil {
		return nil
	}

	netbiosCfg := &NetBIOSConfig{
		Enabled:   yamlNetbios.Enabled,
		Name:      yamlNetbios.Name,
		Workgroup: yamlNetbios.Workgroup,
		NodeType:  yamlNetbios.NodeType,
		Services:  yamlNetbios.Services,
		TTL:       yamlNetbios.TTL,
	}

	// Set defaults
	if netbiosCfg.Name == "" {
		netbiosCfg.Name = deviceName
		if len(netbiosCfg.Name) > 15 {
			netbiosCfg.Name = netbiosCfg.Name[:15]
		}
	}
	if netbiosCfg.Workgroup == "" {
		netbiosCfg.Workgroup = "WORKGROUP"
	}
	if netbiosCfg.NodeType == "" {
		netbiosCfg.NodeType = "B"
	}
	if len(netbiosCfg.Services) == 0 {
		netbiosCfg.Services = []string{"workstation", "fileserver"}
	}
	if netbiosCfg.TTL == 0 {
		netbiosCfg.TTL = DefaultNetBIOSTTL
	}

	return netbiosCfg
}

// parseICMPConfig parses ICMP configuration from YAML
func parseICMPConfig(yamlIcmp *converter.IcmpConfig) *ICMPConfig {
	if yamlIcmp == nil {
		return nil
	}

	icmpCfg := &ICMPConfig{
		Enabled:   yamlIcmp.Enabled,
		TTL:       yamlIcmp.TTL,
		RateLimit: yamlIcmp.RateLimit,
	}

	if icmpCfg.TTL == 0 {
		icmpCfg.TTL = DefaultICMPTTL
	}

	return icmpCfg
}

// parseICMPv6Config parses ICMPv6 configuration from YAML
func parseICMPv6Config(yamlIcmpv6 *converter.Icmpv6Config) *ICMPv6Config {
	if yamlIcmpv6 == nil {
		return nil
	}

	icmpv6Cfg := &ICMPv6Config{
		Enabled:   yamlIcmpv6.Enabled,
		HopLimit:  yamlIcmpv6.HopLimit,
		RateLimit: yamlIcmpv6.RateLimit,
	}

	if icmpv6Cfg.HopLimit == 0 {
		icmpv6Cfg.HopLimit = DefaultICMPv6HopLimit
	}

	return icmpv6Cfg
}

// parseDHCPv6Config parses DHCPv6 configuration from YAML
func parseDHCPv6Config(yamlDhcpv6 *converter.Dhcpv6Config) (*DHCPv6Config, error) {
	if yamlDhcpv6 == nil {
		return nil, nil
	}

	dhcpv6Cfg := &DHCPv6Config{
		Enabled:           yamlDhcpv6.Enabled,
		Pools:             make([]DHCPv6Pool, 0),
		PreferredLifetime: yamlDhcpv6.PreferredLifetime,
		ValidLifetime:     yamlDhcpv6.ValidLifetime,
		Preference:        yamlDhcpv6.Preference,
		DomainList:        yamlDhcpv6.DomainList,
		SIPDomains:        yamlDhcpv6.SIPDomains,
	}

	// Set defaults
	if dhcpv6Cfg.PreferredLifetime == 0 {
		dhcpv6Cfg.PreferredLifetime = DefaultDHCPv6PreferredLifetime
	}
	if dhcpv6Cfg.ValidLifetime == 0 {
		dhcpv6Cfg.ValidLifetime = DefaultDHCPv6ValidLifetime
	}

	// Parse address pools
	for _, pool := range yamlDhcpv6.Pools {
		dhcpv6Cfg.Pools = append(dhcpv6Cfg.Pools, DHCPv6Pool{
			Network:    pool.Network,
			RangeStart: pool.RangeStart,
			RangeEnd:   pool.RangeEnd,
		})
	}

	// Parse DNS servers
	for _, dnsStr := range yamlDhcpv6.DNSServers {
		if ip := net.ParseIP(dnsStr); ip != nil {
			dhcpv6Cfg.DNSServers = append(dhcpv6Cfg.DNSServers, ip)
		}
	}

	// Parse SNTP servers
	for _, sntpStr := range yamlDhcpv6.SNTPServers {
		if ip := net.ParseIP(sntpStr); ip != nil {
			dhcpv6Cfg.SNTPServers = append(dhcpv6Cfg.SNTPServers, ip)
		}
	}

	// Parse NTP servers
	for _, ntpStr := range yamlDhcpv6.NTPServers {
		if ip := net.ParseIP(ntpStr); ip != nil {
			dhcpv6Cfg.NTPServers = append(dhcpv6Cfg.NTPServers, ip)
		}
	}

	// Parse SIP servers
	for _, sipStr := range yamlDhcpv6.SIPServers {
		if ip := net.ParseIP(sipStr); ip != nil {
			dhcpv6Cfg.SIPServers = append(dhcpv6Cfg.SIPServers, ip)
		}
	}

	return dhcpv6Cfg, nil
}

// parseTrafficConfig parses traffic configuration from YAML
func parseTrafficConfig(yamlTraffic *converter.TrafficConfig) *TrafficConfig {
	if yamlTraffic == nil {
		return nil
	}

	trafficCfg := &TrafficConfig{
		Enabled: yamlTraffic.Enabled,
	}

	// Parse ARP Announcements
	if yamlTraffic.ARPAnnouncements != nil {
		arpCfg := &ARPAnnouncementConfig{
			Enabled:  yamlTraffic.ARPAnnouncements.Enabled,
			Interval: yamlTraffic.ARPAnnouncements.Interval,
		}
		if arpCfg.Interval == 0 {
			arpCfg.Interval = DefaultARPAnnouncementInterval
		}
		trafficCfg.ARPAnnouncements = arpCfg
	}

	// Parse Periodic Pings
	if yamlTraffic.PeriodicPings != nil {
		pingCfg := &PeriodicPingConfig{
			Enabled:     yamlTraffic.PeriodicPings.Enabled,
			Interval:    yamlTraffic.PeriodicPings.Interval,
			PayloadSize: yamlTraffic.PeriodicPings.PayloadSize,
		}
		if pingCfg.Interval == 0 {
			pingCfg.Interval = DefaultPeriodicPingInterval
		}
		if pingCfg.PayloadSize == 0 {
			pingCfg.PayloadSize = DefaultPeriodicPingPayloadSize
		}
		trafficCfg.PeriodicPings = pingCfg
	}

	// Parse Random Traffic
	if yamlTraffic.RandomTraffic != nil {
		randomCfg := &RandomTrafficConfig{
			Enabled:     yamlTraffic.RandomTraffic.Enabled,
			Interval:    yamlTraffic.RandomTraffic.Interval,
			PacketCount: yamlTraffic.RandomTraffic.PacketCount,
			Patterns:    yamlTraffic.RandomTraffic.Patterns,
		}
		if randomCfg.Interval == 0 {
			randomCfg.Interval = DefaultRandomTrafficInterval
		}
		if randomCfg.PacketCount == 0 {
			randomCfg.PacketCount = DefaultRandomTrafficPacketCount
		}
		if len(randomCfg.Patterns) == 0 {
			randomCfg.Patterns = []string{"broadcast_arp", "multicast", "udp"}
		}
		trafficCfg.RandomTraffic = randomCfg
	}

	return trafficCfg
}

// parseSNMPTrapsConfig parses SNMP traps configuration from YAML
func parseSNMPTrapsConfig(yamlTraps *converter.TrapsConfig) (*TrapConfig, error) {
	trapsCfg := &TrapConfig{
		Enabled:   yamlTraps.Enabled,
		Receivers: yamlTraps.Receivers,
		Community: yamlTraps.Community,
	}

	// Parse Cold Start trap
	if yamlTraps.ColdStart != nil {
		trapsCfg.ColdStart = &TrapTriggerConfig{
			Enabled:   yamlTraps.ColdStart.Enabled,
			OnStartup: yamlTraps.ColdStart.OnStartup,
		}
	}

	// Parse Link State trap
	if yamlTraps.LinkState != nil {
		trapsCfg.LinkState = &LinkStateTrapConfig{
			Enabled:  yamlTraps.LinkState.Enabled,
			LinkDown: yamlTraps.LinkState.LinkDown,
			LinkUp:   yamlTraps.LinkState.LinkUp,
		}
	}

	// Parse Authentication Failure trap
	if yamlTraps.AuthenticationFailure != nil {
		trapsCfg.AuthenticationFailure = &TrapTriggerConfig{
			Enabled:   yamlTraps.AuthenticationFailure.Enabled,
			OnStartup: yamlTraps.AuthenticationFailure.OnStartup,
		}
	}

	// Parse High CPU trap
	if yamlTraps.HighCPU != nil {
		highCPUCfg := &ThresholdTrapConfig{
			Enabled:   yamlTraps.HighCPU.Enabled,
			Threshold: yamlTraps.HighCPU.Threshold,
			Interval:  yamlTraps.HighCPU.Interval,
		}
		if highCPUCfg.Threshold == 0 {
			highCPUCfg.Threshold = DefaultHighCPUThreshold
		}
		if highCPUCfg.Interval == 0 {
			highCPUCfg.Interval = DefaultTrapCheckInterval
		}
		trapsCfg.HighCPU = highCPUCfg
	}

	// Parse High Memory trap
	if yamlTraps.HighMemory != nil {
		highMemCfg := &ThresholdTrapConfig{
			Enabled:   yamlTraps.HighMemory.Enabled,
			Threshold: yamlTraps.HighMemory.Threshold,
			Interval:  yamlTraps.HighMemory.Interval,
		}
		if highMemCfg.Threshold == 0 {
			highMemCfg.Threshold = DefaultHighMemoryThreshold
		}
		if highMemCfg.Interval == 0 {
			highMemCfg.Interval = DefaultTrapCheckInterval
		}
		trapsCfg.HighMemory = highMemCfg
	}

	// Parse Interface Errors trap
	if yamlTraps.InterfaceErrors != nil {
		ifErrCfg := &ThresholdTrapConfig{
			Enabled:   yamlTraps.InterfaceErrors.Enabled,
			Threshold: yamlTraps.InterfaceErrors.Threshold,
			Interval:  yamlTraps.InterfaceErrors.Interval,
		}
		if ifErrCfg.Threshold == 0 {
			ifErrCfg.Threshold = DefaultInterfaceErrorThreshold
		}
		if ifErrCfg.Interval == 0 {
			ifErrCfg.Interval = DefaultInterfaceErrorInterval
		}
		trapsCfg.InterfaceErrors = ifErrCfg
	}

	return trapsCfg, nil
}

// parseDHCPConfig parses DHCP configuration from YAML
func parseDHCPConfig(yamlDhcp *converter.DhcpServer, deviceName string) (*DHCPConfig, error) {
	if yamlDhcp == nil {
		return nil, nil
	}

	dhcpCfg := &DHCPConfig{}

	// Basic options
	if yamlDhcp.SubnetMask != "" {
		if ip := net.ParseIP(yamlDhcp.SubnetMask); ip != nil {
			dhcpCfg.SubnetMask = net.IPMask(ip.To4())
		}
	}
	if yamlDhcp.Router != "" {
		dhcpCfg.Router = net.ParseIP(yamlDhcp.Router)
	}
	if yamlDhcp.DomainNameServer != "" {
		if ip := net.ParseIP(yamlDhcp.DomainNameServer); ip != nil {
			dhcpCfg.DomainNameServer = append(dhcpCfg.DomainNameServer, ip)
		}
	}
	if yamlDhcp.ServerIdentifier != "" {
		dhcpCfg.ServerIdentifier = net.ParseIP(yamlDhcp.ServerIdentifier)
	}
	if yamlDhcp.NextServerIP != "" {
		dhcpCfg.NextServerIP = net.ParseIP(yamlDhcp.NextServerIP)
	}

	// Pool configuration
	if yamlDhcp.PoolStart != "" {
		dhcpCfg.PoolStart = net.ParseIP(yamlDhcp.PoolStart)
	}
	if yamlDhcp.PoolEnd != "" {
		dhcpCfg.PoolEnd = net.ParseIP(yamlDhcp.PoolEnd)
	}

	// DHCPv4 high priority options
	for _, ntpStr := range yamlDhcp.NTPServers {
		if ip := net.ParseIP(ntpStr); ip != nil {
			dhcpCfg.NTPServers = append(dhcpCfg.NTPServers, ip)
		}
	}
	dhcpCfg.DomainSearch = yamlDhcp.DomainSearch
	dhcpCfg.TFTPServerName = yamlDhcp.TFTPServerName
	dhcpCfg.BootfileName = yamlDhcp.BootfileName
	if yamlDhcp.VendorSpecific != "" {
		// Parse hex string to bytes
		dhcpCfg.VendorSpecific = []byte(yamlDhcp.VendorSpecific)
	}

	// DHCPv6 options
	for _, sntpStr := range yamlDhcp.SNTPServersV6 {
		if ip := net.ParseIP(sntpStr); ip != nil {
			dhcpCfg.SNTPServersV6 = append(dhcpCfg.SNTPServersV6, ip)
		}
	}
	for _, ntpStr := range yamlDhcp.NTPServersV6 {
		if ip := net.ParseIP(ntpStr); ip != nil {
			dhcpCfg.NTPServersV6 = append(dhcpCfg.NTPServersV6, ip)
		}
	}
	for _, sipStr := range yamlDhcp.SIPServersV6 {
		if ip := net.ParseIP(sipStr); ip != nil {
			dhcpCfg.SIPServersV6 = append(dhcpCfg.SIPServersV6, ip)
		}
	}
	dhcpCfg.SIPDomainsV6 = yamlDhcp.SIPDomainsV6

	// Static leases
	for _, lease := range yamlDhcp.ClientLeases {
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

	return dhcpCfg, nil
}

// parseDNSConfig parses DNS configuration from YAML
func parseDNSConfig(yamlDns *converter.DnsServer, deviceName string) (*DNSConfig, error) {
	if yamlDns == nil {
		return nil, nil
	}

	dnsCfg := &DNSConfig{}

	// Forward records (A records)
	for _, record := range yamlDns.ForwardRecords {
		ip := net.ParseIP(record.IP)
		if ip == nil {
			continue
		}
		ttl := uint32(DefaultDNSTTL)
		if record.TTL > 0 {
			// Validate TTL is in reasonable range before conversion
			if record.TTL < 0 {
				return nil, fmt.Errorf("device %s: DNS forward record TTL cannot be negative: %d",
					deviceName, record.TTL)
			}
			if record.TTL > 2147483647 { // Max int32 (~68 years)
				return nil, fmt.Errorf("device %s: DNS forward record TTL exceeds maximum (2147483647): %d",
					deviceName, record.TTL)
			}
			ttl = uint32(record.TTL)
		}
		dnsCfg.ForwardRecords = append(dnsCfg.ForwardRecords, DNSRecord{
			Name: record.Name,
			IP:   ip,
			TTL:  ttl,
		})
	}

	// Reverse records (PTR records)
	for _, record := range yamlDns.ReverseRecords {
		ip := net.ParseIP(record.IP)
		if ip == nil {
			continue
		}
		ttl := uint32(DefaultDNSTTL)
		if record.TTL > 0 {
			// Validate TTL is in reasonable range before conversion
			if record.TTL < 0 {
				return nil, fmt.Errorf("device %s: DNS reverse record TTL cannot be negative: %d",
					deviceName, record.TTL)
			}
			if record.TTL > 2147483647 { // Max int32 (~68 years)
				return nil, fmt.Errorf("device %s: DNS reverse record TTL exceeds maximum (2147483647): %d",
					deviceName, record.TTL)
			}
			ttl = uint32(record.TTL)
		}
		dnsCfg.ReverseRecords = append(dnsCfg.ReverseRecords, DNSRecord{
			Name: record.Name,
			IP:   ip,
			TTL:  ttl,
		})
	}

	return dnsCfg, nil
}

// parseLLDPConfig parses LLDP configuration from YAML
func parseLLDPConfig(yamlLldp *converter.LldpConfig) *LLDPConfig {
	if yamlLldp == nil {
		return nil
	}

	lldpCfg := &LLDPConfig{
		Enabled:           yamlLldp.Enabled,
		AdvertiseInterval: yamlLldp.AdvertiseInterval,
		TTL:               yamlLldp.TTL,
		SystemDescription: yamlLldp.SystemDescription,
		PortDescription:   yamlLldp.PortDescription,
		ChassisIDType:     yamlLldp.ChassisIDType,
	}
	// Set defaults if not specified
	if lldpCfg.AdvertiseInterval == 0 {
		lldpCfg.AdvertiseInterval = DefaultLLDPAdvertiseInterval
	}
	if lldpCfg.TTL == 0 {
		lldpCfg.TTL = DefaultLLDPTTL
	}
	if lldpCfg.ChassisIDType == "" {
		lldpCfg.ChassisIDType = ChassisIDTypeMAC
	}
	return lldpCfg
}

// parseCDPConfig parses CDP configuration from YAML
func parseCDPConfig(yamlCdp *converter.CdpConfig) *CDPConfig {
	if yamlCdp == nil {
		return nil
	}

	cdpCfg := &CDPConfig{
		Enabled:           yamlCdp.Enabled,
		AdvertiseInterval: yamlCdp.AdvertiseInterval,
		Holdtime:          yamlCdp.Holdtime,
		Version:           yamlCdp.Version,
		SoftwareVersion:   yamlCdp.SoftwareVersion,
		Platform:          yamlCdp.Platform,
		PortID:            yamlCdp.PortID,
	}
	// Set defaults if not specified
	if cdpCfg.AdvertiseInterval == 0 {
		cdpCfg.AdvertiseInterval = DefaultCDPAdvertiseInterval
	}
	if cdpCfg.Holdtime == 0 {
		cdpCfg.Holdtime = DefaultCDPHoldtime
	}
	if cdpCfg.Version == 0 {
		cdpCfg.Version = DefaultCDPVersion
	}
	return cdpCfg
}

// parseEDPConfig parses EDP configuration from YAML
func parseEDPConfig(yamlEdp *converter.EdpConfig) *EDPConfig {
	if yamlEdp == nil {
		return nil
	}

	edpCfg := &EDPConfig{
		Enabled:           yamlEdp.Enabled,
		AdvertiseInterval: yamlEdp.AdvertiseInterval,
		VersionString:     yamlEdp.VersionString,
		DisplayString:     yamlEdp.DisplayString,
	}
	// Set defaults if not specified
	if edpCfg.AdvertiseInterval == 0 {
		edpCfg.AdvertiseInterval = DefaultEDPAdvertiseInterval
	}
	return edpCfg
}

// parseFDPConfig parses FDP configuration from YAML
func parseFDPConfig(yamlFdp *converter.FdpConfig) *FDPConfig {
	if yamlFdp == nil {
		return nil
	}

	fdpCfg := &FDPConfig{
		Enabled:           yamlFdp.Enabled,
		AdvertiseInterval: yamlFdp.AdvertiseInterval,
		Holdtime:          yamlFdp.Holdtime,
		SoftwareVersion:   yamlFdp.SoftwareVersion,
		Platform:          yamlFdp.Platform,
		PortID:            yamlFdp.PortID,
	}
	// Set defaults if not specified
	if fdpCfg.AdvertiseInterval == 0 {
		fdpCfg.AdvertiseInterval = DefaultFDPAdvertiseInterval
	}
	if fdpCfg.Holdtime == 0 {
		fdpCfg.Holdtime = DefaultFDPHoldtime
	}
	return fdpCfg
}

// parseSTPConfig parses STP configuration from YAML
func parseSTPConfig(yamlStp *converter.StpConfig) *STPConfig {
	if yamlStp == nil {
		return nil
	}

	stpCfg := &STPConfig{
		Enabled:        yamlStp.Enabled,
		BridgePriority: yamlStp.BridgePriority,
		HelloTime:      yamlStp.HelloTime,
		MaxAge:         yamlStp.MaxAge,
		ForwardDelay:   yamlStp.ForwardDelay,
		Version:        yamlStp.Version,
	}
	// Set defaults if not specified
	if stpCfg.BridgePriority == 0 {
		stpCfg.BridgePriority = DefaultSTPBridgePriority
	}
	if stpCfg.HelloTime == 0 {
		stpCfg.HelloTime = DefaultSTPHelloTime
	}
	if stpCfg.MaxAge == 0 {
		stpCfg.MaxAge = DefaultSTPMaxAge
	}
	if stpCfg.ForwardDelay == 0 {
		stpCfg.ForwardDelay = DefaultSTPForwardDelay
	}
	if stpCfg.Version == "" {
		stpCfg.Version = "stp" // Default to STP
	}
	return stpCfg
}

// parseHTTPConfig parses HTTP configuration from YAML
func parseHTTPConfig(yamlHttp *converter.HttpConfig, deviceName string) *HTTPConfig {
	if yamlHttp == nil {
		return nil
	}

	httpCfg := &HTTPConfig{
		Enabled:    yamlHttp.Enabled,
		ServerName: yamlHttp.ServerName,
		Endpoints:  make([]HTTPEndpoint, 0),
	}
	// Set default server name if not specified
	if httpCfg.ServerName == "" {
		httpCfg.ServerName = "NIAC-Go/1.0.0"
	}
	// Parse endpoints
	for _, ep := range yamlHttp.Endpoints {
		endpoint := HTTPEndpoint{
			Path:        ep.Path,
			Method:      ep.Method,
			StatusCode:  ep.StatusCode,
			ContentType: ep.ContentType,
			Body:        ep.Body,
		}
		// Set defaults
		if endpoint.Method == "" {
			endpoint.Method = "GET"
		}
		if endpoint.StatusCode == 0 {
			endpoint.StatusCode = 200
		}
		if endpoint.ContentType == "" {
			endpoint.ContentType = "text/html"
		}
		httpCfg.Endpoints = append(httpCfg.Endpoints, endpoint)
	}
	return httpCfg
}

// parseFTPConfig parses FTP configuration from YAML
func parseFTPConfig(yamlFtp *converter.FtpConfig, deviceName string) *FTPConfig {
	if yamlFtp == nil {
		return nil
	}

	ftpCfg := &FTPConfig{
		Enabled:        yamlFtp.Enabled,
		WelcomeBanner:  yamlFtp.WelcomeBanner,
		SystemType:     yamlFtp.SystemType,
		AllowAnonymous: yamlFtp.AllowAnonymous,
		Users:          make([]FTPUser, 0),
	}
	// Set defaults
	if ftpCfg.WelcomeBanner == "" {
		ftpCfg.WelcomeBanner = fmt.Sprintf("220 %s FTP Server (NIAC-Go) ready.", deviceName)
	}
	if ftpCfg.SystemType == "" {
		ftpCfg.SystemType = "UNIX Type: L8"
	}
	// Parse users
	for _, u := range yamlFtp.Users {
		user := FTPUser{
			Username: u.Username,
			Password: u.Password,
			HomeDir:  u.HomeDir,
		}
		if user.HomeDir == "" {
			user.HomeDir = "/"
		}
		ftpCfg.Users = append(ftpCfg.Users, user)
	}
	return ftpCfg
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

// validateWalkFilePath validates and resolves SNMP walk file paths
// Prevents path traversal attacks and ensures file exists
func validateWalkFilePath(basePath, walkFile, deviceName string) (string, error) {
	// Clean the path to normalize it
	cleanPath := filepath.Clean(walkFile)

	// Security: Prevent directory traversal
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("device %s: walk file path contains invalid traversal: %s", deviceName, walkFile)
	}

	// Build full path
	var fullPath string
	if filepath.IsAbs(cleanPath) {
		fullPath = cleanPath
	} else if basePath != "" {
		fullPath = filepath.Join(basePath, cleanPath)
	} else {
		fullPath = cleanPath
	}

	// Verify file exists and is accessible
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("device %s: walk file not found: %s", deviceName, fullPath)
		}
		return "", fmt.Errorf("device %s: cannot access walk file %s: %w", deviceName, fullPath, err)
	}

	// Verify it's a regular file, not a directory or device
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("device %s: walk file is not a regular file: %s", deviceName, fullPath)
	}

	return fullPath, nil
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
