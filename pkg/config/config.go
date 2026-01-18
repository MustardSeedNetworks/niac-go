// Package config provides configuration file loading and parsing for network device simulation
package config

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/krisarmstrong/niac-go/internal/converter"
)

// Sentinel errors for configuration validation.
var (
	ErrInvalidDeviceDeclaration = errors.New("invalid device declaration")
	ErrNoDevicesDefined         = errors.New("no devices defined in configuration")
	ErrInvalidMapToIP           = errors.New("invalid map_to_ip")
	ErrInvalidTTLIP             = errors.New("invalid ttl ip")
	ErrInvalidTTLMask           = errors.New("invalid ttl mask")
	ErrInvalidIPAddress         = errors.New("invalid IP address")
	ErrInvalidSNMPAddr          = errors.New("invalid snmp_addr")
	ErrDNSTTLNegative           = errors.New("DNS record TTL cannot be negative")
	ErrDNSTTLExceedsMax         = errors.New("DNS record TTL exceeds maximum (2147483647)")
	ErrInsufficientFields       = errors.New("insufficient fields")
	ErrPathTraversalDetected    = errors.New("path traversal detected")
	ErrPathOutsideBaseDir       = errors.New("path outside base directory")
	ErrWalkFileNotFound         = errors.New("walk file not found")
	ErrWalkFileIsSymlink        = errors.New("walk file is a symlink (not allowed)")
	ErrWalkFileNotRegular       = errors.New("walk file is not a regular file")
)

// LLDP Chassis ID Type constants.
const (
	ChassisIDTypeMAC            = "mac"
	ChassisIDTypeLocal          = "local"
	ChassisIDTypeNetworkAddress = "network_address"
)

// Default configuration values.
const (
	// DefaultLLDPAdvertiseInterval is the default LLDP advertisement interval in seconds.
	DefaultLLDPAdvertiseInterval = 30
	// DefaultLLDPTTL is the default LLDP time-to-live in seconds.
	DefaultLLDPTTL = 120
	// DefaultCDPAdvertiseInterval is the default CDP advertisement interval in seconds.
	DefaultCDPAdvertiseInterval = 60
	// DefaultCDPHoldtime is the default CDP holdtime in seconds.
	DefaultCDPHoldtime = 180
	// DefaultCDPVersion is the default CDP protocol version.
	DefaultCDPVersion = 2
	// DefaultEDPAdvertiseInterval is the default EDP advertisement interval in seconds.
	DefaultEDPAdvertiseInterval = 30
	// DefaultFDPAdvertiseInterval is the default FDP advertisement interval in seconds.
	DefaultFDPAdvertiseInterval = 60
	// DefaultFDPHoldtime is the default FDP holdtime in seconds.
	DefaultFDPHoldtime = 180

	// DefaultSTPBridgePriority is the default STP bridge priority.
	DefaultSTPBridgePriority = 32768
	// DefaultSTPHelloTime is the default STP hello time in seconds.
	DefaultSTPHelloTime = 2
	// DefaultSTPMaxAge is the default STP max age in seconds.
	DefaultSTPMaxAge = 20
	// DefaultSTPForwardDelay is the default STP forward delay in seconds.
	DefaultSTPForwardDelay = 15

	// DefaultNetBIOSTTL is the default NetBIOS TTL in seconds.
	DefaultNetBIOSTTL = 300

	// DefaultICMPTTL is the default ICMP time-to-live.
	DefaultICMPTTL = 64
	// DefaultICMPv6HopLimit is the default ICMPv6 hop limit.
	DefaultICMPv6HopLimit = 64

	// DefaultDHCPv6PreferredLifetime is the default DHCPv6 preferred lifetime in seconds.
	DefaultDHCPv6PreferredLifetime = 604800
	// DefaultDHCPv6ValidLifetime is the default DHCPv6 valid lifetime in seconds.
	DefaultDHCPv6ValidLifetime = 2592000

	// DefaultARPAnnouncementInterval is the default ARP announcement interval in seconds.
	DefaultARPAnnouncementInterval = 60
	// DefaultPeriodicPingInterval is the default periodic ping interval in seconds.
	DefaultPeriodicPingInterval = 120
	// DefaultPeriodicPingPayloadSize is the default periodic ping payload size in bytes.
	DefaultPeriodicPingPayloadSize = 32
	// DefaultRandomTrafficInterval is the default random traffic interval in seconds.
	DefaultRandomTrafficInterval = 180
	// DefaultRandomTrafficPacketCount is the default packets per interval.
	DefaultRandomTrafficPacketCount = 5

	// DefaultSNMPCommunity is the default SNMP community string.
	DefaultSNMPCommunity = "public"

	// DefaultHighCPUThreshold is the default high CPU usage threshold in percent.
	DefaultHighCPUThreshold = 80
	// DefaultHighMemoryThreshold is the default high memory usage threshold in percent.
	DefaultHighMemoryThreshold = 90
	// DefaultInterfaceErrorThreshold is the default interface error threshold count.
	DefaultInterfaceErrorThreshold = 100
	// DefaultTrapCheckInterval is the default trap check interval in seconds.
	DefaultTrapCheckInterval = 300
	// DefaultInterfaceErrorInterval is the default interface error check interval in seconds.
	DefaultInterfaceErrorInterval = 60

	// DefaultDNSTTL is the default DNS TTL in seconds.
	DefaultDNSTTL = 3600

	// Parser constants.
	minDeviceDeclParts   = 2          // minimum parts for device declaration (device name)
	keyValueParts        = 2          // key-value pair after SplitN
	netbiosMaxNameLen    = 15         // NetBIOS name max length
	maxDNSTTL            = 2147483647 // max int32 (~68 years)
	minSimpleConfigParts = 4          // minimum fields: name type ip mac
	simpleConfigWithWalk = 5          // simple config with walkfile
	macAddressBytes      = 6          // MAC address length in bytes
	macPatternMultiplier = 17         // pattern multiplier for test MAC generation
	gbpsToMbps           = 1000       // conversion factor Gbps to Mbps
)

// Config represents the network configuration.
type Config struct {
	Devices            []Device
	IncludePath        string              // Base path for walk files
	CapturePlayback    *CapturePlayback    // Optional PCAP playback config
	DiscoveryProtocols *DiscoveryProtocols // Discovery protocol configuration
}

// CapturePlayback represents PCAP file playback configuration.
type CapturePlayback struct {
	FileName  string
	LoopTime  int     // milliseconds
	ScaleTime float64 // time scaling factor
}

// DiscoveryProtocols configures discovery protocol behavior.
type DiscoveryProtocols struct {
	LLDP *ProtocolConfig
	CDP  *ProtocolConfig
	EDP  *ProtocolConfig
	FDP  *ProtocolConfig
}

// ProtocolConfig configures a discovery protocol.
type ProtocolConfig struct {
	Enabled  bool
	Interval int // Advertisement interval in seconds
}

// Device represents a simulated network device.
type Device struct {
	Name                string
	Type                string // router, switch, ap, etc.
	MACAddress          net.HardwareAddr
	IPAddresses         []net.IP
	MapToIP             net.IP     // Map UDP traffic to external IP (Java MapToIp)
	Babble              bool       // Periodically emit babble traffic
	TTLConfig           *TTLConfig // ICMP TTL timeout behavior (traceroute simulation)
	VLAN                int        // Optional VLAN membership (Java Vlan)
	Interfaces          []Interface
	SNMPConfig          SNMPConfig
	DHCPConfig          *DHCPConfig          // DHCP server configuration
	DNSConfig           *DNSConfig           // DNS server configuration
	LLDPConfig          *LLDPConfig          // LLDP discovery protocol configuration
	CDPConfig           *CDPConfig           // CDP discovery protocol configuration
	EDPConfig           *EDPConfig           // EDP discovery protocol configuration
	FDPConfig           *FDPConfig           // FDP discovery protocol configuration
	STPConfig           *STPConfig           // STP/RSTP/MSTP configuration
	HTTPConfig          *HTTPConfig          // HTTP server configuration
	FTPConfig           *FTPConfig           // FTP server configuration
	NetBIOSConfig       *NetBIOSConfig       // NetBIOS service configuration
	ICMPConfig          *ICMPConfig          // ICMP/ICMPv4 configuration
	ICMPv6Config        *ICMPv6Config        // ICMPv6 configuration
	DHCPv6Config        *DHCPv6Config        // DHCPv6 server configuration
	TrafficConfig       *TrafficConfig       // Traffic pattern configuration (v1.6.0)
	OSFingerprintConfig *OSFingerprintConfig // OS fingerprinting configuration (v1.24.0)
	IPerf3              *IPerf3Config        // iPerf3 server emulation configuration (v1.25.0)
	PortChannels        []PortChannel        // Port-channel/LAG configuration (v1.23.0)
	TrunkPorts          []TrunkPort          // Trunk port configuration (v1.23.0)
	Properties          map[string]string
}

// OSFingerprintConfig represents OS fingerprinting configuration for device simulation.
type OSFingerprintConfig struct {
	OSType       string // e.g., "linux", "windows", "cisco-ios", "juniper-junos"
	TTL          uint8  // Default IP TTL (Linux=64, Windows=128, Cisco=255)
	WindowSize   uint16 // TCP window size
	WindowScale  uint8  // TCP window scale option
	MSS          uint16 // TCP maximum segment size
	SSHBanner    string // SSH version banner
	HTTPServer   string // HTTP Server header
	FTPBanner    string // FTP welcome banner
	SMTPBanner   string // SMTP banner
	TelnetBanner string // Telnet banner
	DontFragment bool   // IP DF bit (Linux=true, Windows=false usually)
}

// IPerf3Config holds iPerf3 server emulation configuration for bandwidth testing simulation.
type IPerf3Config struct {
	Enabled           bool    // Whether iPerf3 server is enabled
	Port              uint16  // Listen port (default 5201)
	MaxBandwidthMbps  float64 // Maximum simulated bandwidth
	TypicalLatencyMs  float64 // Simulated network latency
	JitterMs          float64 // Simulated jitter (UDP tests)
	PacketLossPercent float64 // Simulated packet loss percentage
	UploadMbps        float64 // Simulated upload bandwidth
	DownloadMbps      float64 // Simulated download bandwidth
}

// DHCPConfig holds DHCP server configuration for a device.
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

// DHCPLease represents a static DHCP lease assignment.
type DHCPLease struct {
	ClientIP   net.IP
	MACAddress net.HardwareAddr
	MACMask    net.HardwareAddr // For wildcard matching
}

// DNSConfig holds DNS server configuration for a device.
type DNSConfig struct {
	ForwardRecords []DNSRecord
	ReverseRecords []DNSRecord
}

// DNSRecord represents a DNS A or PTR record.
type DNSRecord struct {
	Name  string
	IP    net.IP
	TTL   uint32
	RCode int // DNS response code override (0 = NoError)
}

// Interface represents a network interface on a device.
type Interface struct {
	Name        string
	Speed       int // Mbps
	Duplex      string
	AdminStatus string // up, down
	OperStatus  string // up, down, testing
	Description string
	VLANs       []int
}

// SNMPConfig holds SNMP configuration.
type SNMPConfig struct {
	Community         string
	SysName           string
	SysDescr          string
	SysContact        string
	SysLocation       string
	WalkFile          string             // Path to SNMP walk file (legacy single)
	WalkFiles         []string           // Paths to SNMP walk files
	AddMibs           []AddMib           // Custom MIB overrides/additions
	CommunityIncludes []CommunityInclude // Community-specific walk includes
	AccessList        []net.IP           // Allowed source IPs for SNMP
	SnmpAddr          net.IP             // SNMP agent mapped from another device
	Dot1DFdbTable     *FdbTableConfig    // FDB table injection (dot1d)
	Dot1QFdbTable     *FdbTableConfig    // FDB table injection (dot1q)
	Traps             *TrapConfig        // SNMP trap configuration (v1.6.0)
}

// AddMib represents a MIB override or addition.
type AddMib struct {
	OID   string
	Type  string
	Value string
}

// CommunityInclude represents a community-specific walk include.
type CommunityInclude struct {
	Community string
	WalkFile  string
}

// FdbTableConfig configures SNMP forwarding database table injection.
type FdbTableConfig struct {
	Port int
	VLAN int
}

// TTLConfig configures ICMP TTL timeout behavior (traceroute simulation).
type TTLConfig struct {
	TTL  int
	IP   net.IP
	Mask net.IPMask
}

// LLDPConfig holds LLDP (Link Layer Discovery Protocol) configuration.
type LLDPConfig struct {
	Enabled           bool
	AdvertiseInterval int // seconds
	TTL               int // seconds
	SystemDescription string
	PortDescription   string
	ChassisIDType     string // "mac", "local", "network_address"
}

// CDPConfig holds CDP (Cisco Discovery Protocol) configuration.
type CDPConfig struct {
	Enabled           bool
	AdvertiseInterval int // seconds
	Holdtime          int // seconds
	Version           int // 1 or 2
	SoftwareVersion   string
	Platform          string
	PortID            string
}

// EDPConfig holds EDP (Extreme Discovery Protocol) configuration.
type EDPConfig struct {
	Enabled           bool
	AdvertiseInterval int // seconds
	VersionString     string
	DisplayString     string
}

// FDPConfig holds FDP (Foundry Discovery Protocol) configuration.
type FDPConfig struct {
	Enabled           bool
	AdvertiseInterval int // seconds
	Holdtime          int // seconds
	SoftwareVersion   string
	Platform          string
	PortID            string
}

// STPConfig holds STP (Spanning Tree Protocol) configuration.
type STPConfig struct {
	Enabled        bool
	BridgePriority uint16 // 0-61440 in increments of 4096 (default: 32768)
	HelloTime      uint16 // seconds (default: 2)
	MaxAge         uint16 // seconds (default: 20)
	ForwardDelay   uint16 // seconds (default: 15)
	Version        string // "stp", "rstp", "mstp" (default: "stp")
}

// HTTPConfig holds HTTP server configuration.
type HTTPConfig struct {
	Enabled    bool
	ServerName string         // Server header value (default: "NIAC-Go/1.0.0")
	Endpoints  []HTTPEndpoint // Custom endpoint definitions
}

// HTTPEndpoint defines a custom HTTP endpoint and response.
type HTTPEndpoint struct {
	Path        string // URL path (e.g., "/", "/api/info")
	Method      string // HTTP method (default: "GET")
	StatusCode  int    // HTTP status code (default: 200)
	ContentType string // Content-Type header (default: "text/html")
	Body        string // Response body
}

// FTPConfig holds FTP server configuration.
type FTPConfig struct {
	Enabled        bool
	WelcomeBanner  string    // Welcome message (default: "220 {devicename} FTP Server (NIAC-Go) ready.")
	SystemType     string    // System type string (default: "UNIX Type: L8")
	AllowAnonymous bool      // Allow anonymous login (default: true)
	Users          []FTPUser // User accounts
}

// FTPUser represents an FTP user account.
type FTPUser struct {
	Username string
	Password string
	HomeDir  string // Virtual home directory path
}

// NetBIOSConfig holds NetBIOS service configuration.
type NetBIOSConfig struct {
	Enabled   bool
	Name      string        // NetBIOS name (default: device name, max 15 chars)
	Workgroup string        // Workgroup/domain name (default: "WORKGROUP")
	NodeType  string        // Node type: "B" (broadcast), "P" (peer), "M" (mixed), "H" (hybrid) (default: "B")
	Services  []string      // Service types to advertise (default: ["workstation", "fileserver"])
	TTL       uint32        // Name registration TTL in seconds (default: 300)
	Names     []NetBIOSName // Explicit NetBIOS status names
	MsBrowse  bool          // Enable __MSBROWSE__ group name
}

// NetBIOSName represents a NetBIOS name entry.
type NetBIOSName struct {
	Name   string
	Suffix uint8
	Group  bool
}

// ICMPConfig holds ICMP/ICMPv4 configuration.
type ICMPConfig struct {
	Enabled             bool
	TTL                 uint8 // Time to Live for ICMP packets (default: 64)
	RateLimit           int   // Max ICMP responses per second (0 = unlimited, default: 0)
	AddressMaskReply    net.IP
	RouterAdvertisement *IcmpRouterAdvertisement
}

// IcmpRouterAdvertisement configures IPv4 router advertisements.
type IcmpRouterAdvertisement struct {
	Period   int
	Lifetime int
	Routers  []IcmpRouter
}

// IcmpRouter represents an advertised router entry.
type IcmpRouter struct {
	Address    net.IP
	Preference int
}

// ICMPv6Config holds ICMPv6 configuration.
type ICMPv6Config struct {
	Enabled             bool
	HopLimit            uint8 // Hop limit for ICMPv6 packets (default: 64, NDP uses 255)
	RateLimit           int   // Max ICMPv6 responses per second (0 = unlimited, default: 0)
	RouterAdvertisement *Icmpv6RouterAdvertisement
}

// Icmpv6RouterAdvertisement configures IPv6 router advertisements.
type Icmpv6RouterAdvertisement struct {
	Period        int
	CurHopLimit   int
	Managed       int
	Other         int
	Lifetime      int
	ReachableTime int
	RetransTimer  int
	MTU           int
	PrefixInfo    []Icmpv6PrefixInfo
}

// Icmpv6PrefixInfo represents IPv6 prefix information options.
type Icmpv6PrefixInfo struct {
	PrefixLength      int
	Onlink            int
	Auto              int
	ValidLifetime     int
	PreferredLifetime int
	Prefix            net.IP
}

// DHCPv6Config holds DHCPv6 server configuration.
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

// DHCPv6Pool represents an IPv6 address pool.
type DHCPv6Pool struct {
	Network    string // IPv6 network (e.g., "2001:db8::/64")
	RangeStart string // Start of address range
	RangeEnd   string // End of address range
}

// TrafficConfig holds traffic pattern configuration (v1.6.0).
type TrafficConfig struct {
	Enabled          bool
	ARPAnnouncements *ARPAnnouncementConfig
	PeriodicPings    *PeriodicPingConfig
	RandomTraffic    *RandomTrafficConfig
}

// ARPAnnouncementConfig configures gratuitous ARP announcements.
type ARPAnnouncementConfig struct {
	Enabled  bool
	Interval int // Interval in seconds (default: 60)
}

// PeriodicPingConfig configures periodic ICMP echo requests.
type PeriodicPingConfig struct {
	Enabled     bool
	Interval    int // Interval in seconds (default: 120)
	PayloadSize int // Payload size in bytes (default: 32)
}

// RandomTrafficConfig configures random background traffic.
type RandomTrafficConfig struct {
	Enabled     bool
	Interval    int      // Interval in seconds (default: 180)
	PacketCount int      // Number of packets per interval (default: 5)
	Patterns    []string // Traffic patterns: "broadcast_arp", "multicast", "udp"
}

// TrapConfig holds SNMP trap configuration (v1.6.0).
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

// TrapTriggerConfig configures a simple trap trigger.
type TrapTriggerConfig struct {
	Enabled   bool
	OnStartup bool // Send trap on device startup
}

// LinkStateTrapConfig configures link up/down traps.
type LinkStateTrapConfig struct {
	Enabled  bool
	LinkDown bool // Send trap on link down
	LinkUp   bool // Send trap on link up
}

// ThresholdTrapConfig configures threshold-based traps.
type ThresholdTrapConfig struct {
	Enabled   bool
	Threshold int // Threshold value (percent for CPU/Memory, count for errors)
	Interval  int // Check interval in seconds
}

// PortChannel represents a port-channel (LAG/Link Aggregation Group) configuration (v1.23.0).
type PortChannel struct {
	ID      int      // Port-channel ID (e.g., 1 for Port-channel1)
	Members []string // Member interface names (e.g., ["Ethernet1/1", "Ethernet1/2"])
	Mode    string   // LACP mode: "active", "passive", "on" (static)
}

// TrunkPort represents a trunk port configuration with VLAN tagging (v1.23.0).
type TrunkPort struct {
	Interface       string // Local interface name (can be physical or port-channel)
	VLANs           []int  // List of allowed VLANs
	NativeVLAN      int    // Native VLAN (untagged, default: 1)
	RemoteDevice    string // Remote device name (for topology validation)
	RemoteInterface string // Remote interface name (for LLDP/CDP neighbor)
}

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

func validateYAMLConfig(yamlConfig *converter.Config) (*converter.Config, error) {
	err := converter.ValidateConfig(yamlConfig)
	if err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return yamlConfig, nil
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

// convertYAMLDevice converts a YAML device to a runtime Device.
func convertYAMLDevice(yamlDevice converter.Device, includePath string) (Device, error) {
	device := createBaseDevice(yamlDevice)

	if err := parseDeviceMACAddress(&device, &yamlDevice); err != nil {
		return device, err
	}

	if err := parseDeviceIPAddresses(&device, &yamlDevice); err != nil {
		return device, err
	}

	if err := parseDeviceMapToIP(&device, &yamlDevice); err != nil {
		return device, err
	}

	device.Babble = yamlDevice.Babble

	if err := parseDeviceTTLConfig(&device, &yamlDevice); err != nil {
		return device, err
	}

	if err := parseDeviceSNMPConfig(&device, &yamlDevice, includePath); err != nil {
		return device, err
	}

	if yamlDevice.VLAN > 0 {
		device.Properties["vlan"] = strconv.Itoa(yamlDevice.VLAN)
		device.VLAN = yamlDevice.VLAN
	}

	if err := parseDeviceProtocolConfigs(&device, &yamlDevice); err != nil {
		return device, err
	}

	return device, nil
}

// createBaseDevice creates a device with default values.
func createBaseDevice(yamlDevice converter.Device) Device {
	deviceType := yamlDevice.Type
	if deviceType == "" {
		deviceType = "unknown"
	}

	return Device{
		Name:       yamlDevice.Name,
		Type:       deviceType,
		Interfaces: make([]Interface, 0),
		Properties: make(map[string]string),
		SNMPConfig: SNMPConfig{
			Community: DefaultSNMPCommunity,
			SysName:   yamlDevice.Name,
		},
	}
}

// parseDeviceMACAddress parses the MAC address for a device.
func parseDeviceMACAddress(device *Device, yamlDevice *converter.Device) error {
	if yamlDevice.MAC == "" {
		return nil
	}

	mac, err := net.ParseMAC(yamlDevice.MAC)
	if err != nil {
		return fmt.Errorf("device %s: invalid MAC address %s: %w", yamlDevice.Name, yamlDevice.MAC, err)
	}

	device.MACAddress = mac

	return nil
}

// parseDeviceMapToIP parses the MapToIP configuration.
func parseDeviceMapToIP(device *Device, yamlDevice *converter.Device) error {
	if yamlDevice.MapToIP == "" {
		return nil
	}

	ip := net.ParseIP(yamlDevice.MapToIP)
	if ip == nil {
		return fmt.Errorf("device %s: %w: %s", yamlDevice.Name, ErrInvalidMapToIP, yamlDevice.MapToIP)
	}

	device.MapToIP = ip

	return nil
}

// parseDeviceTTLConfig parses TTL configuration for traceroute simulation.
func parseDeviceTTLConfig(device *Device, yamlDevice *converter.Device) error {
	if yamlDevice.TTL == nil {
		return nil
	}

	ttlCfg := &TTLConfig{TTL: yamlDevice.TTL.TTL}

	if yamlDevice.TTL.IP != "" {
		ip := net.ParseIP(yamlDevice.TTL.IP)
		if ip == nil {
			return fmt.Errorf("device %s: %w: %s", yamlDevice.Name, ErrInvalidTTLIP, yamlDevice.TTL.IP)
		}

		ttlCfg.IP = ip.To4()
	}

	if yamlDevice.TTL.Mask != "" {
		ip := net.ParseIP(yamlDevice.TTL.Mask)
		if ip == nil {
			return fmt.Errorf("device %s: %w: %s", yamlDevice.Name, ErrInvalidTTLMask, yamlDevice.TTL.Mask)
		}

		ttlCfg.Mask = net.IPMask(ip.To4())
	}

	device.TTLConfig = ttlCfg

	return nil
}

// parseDeviceIPAddresses parses IP addresses for a device.
func parseDeviceIPAddresses(device *Device, yamlDevice *converter.Device) error {
	// Support both singular 'ip' (backward compatible) and plural 'ips' (new feature)
	if yamlDevice.IP != "" {
		ip := net.ParseIP(yamlDevice.IP)
		if ip == nil {
			return fmt.Errorf("device %s: %w: %s", yamlDevice.Name, ErrInvalidIPAddress, yamlDevice.IP)
		}

		device.IPAddresses = append(device.IPAddresses, ip)
	}

	// Parse multiple IPs if specified
	for i, ipStr := range yamlDevice.IPs {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return fmt.Errorf("device %s: %w in ips[%d]: %s", yamlDevice.Name, ErrInvalidIPAddress, i, ipStr)
		}

		device.IPAddresses = append(device.IPAddresses, ip)
	}

	return nil
}

// parseDeviceSNMPConfig parses SNMP configuration for a device.
func parseDeviceSNMPConfig(device *Device, yamlDevice *converter.Device, includePath string) error {
	if yamlDevice.SnmpAgent == nil {
		return nil
	}

	snmpAgent := yamlDevice.SnmpAgent

	if err := parseSNMPWalkFiles(device, snmpAgent, includePath, yamlDevice.Name); err != nil {
		return err
	}

	parseSNMPAddMibs(device, snmpAgent)

	if err := parseSNMPCommunityIncludes(device, snmpAgent, includePath, yamlDevice.Name); err != nil {
		return err
	}

	parseSNMPAccessList(device, snmpAgent)

	if err := parseSNMPAddr(device, snmpAgent, yamlDevice.Name); err != nil {
		return err
	}

	parseSNMPFdbTables(device, snmpAgent)

	if snmpAgent.Traps != nil {
		device.SNMPConfig.Traps = parseSNMPTrapsConfig(snmpAgent.Traps)
	}

	return nil
}

// parseSNMPWalkFiles parses and validates SNMP walk files.
func parseSNMPWalkFiles(device *Device, snmpAgent *converter.SnmpAgent, includePath, deviceName string) error {
	if snmpAgent.WalkFile != "" {
		walkFile, err := validateWalkFilePath(includePath, snmpAgent.WalkFile, deviceName)
		if err != nil {
			return err
		}

		device.SNMPConfig.WalkFile = walkFile
		device.SNMPConfig.WalkFiles = append(device.SNMPConfig.WalkFiles, walkFile)
	}

	for _, walk := range snmpAgent.WalkFiles {
		walkFile, err := validateWalkFilePath(includePath, walk, deviceName)
		if err != nil {
			return err
		}

		device.SNMPConfig.WalkFiles = append(device.SNMPConfig.WalkFiles, walkFile)
	}

	return nil
}

// parseSNMPAddMibs parses custom MIB overrides.
func parseSNMPAddMibs(device *Device, snmpAgent *converter.SnmpAgent) {
	if len(snmpAgent.AddMibs) == 0 {
		return
	}

	device.Properties["custom_mibs_count"] = strconv.Itoa(len(snmpAgent.AddMibs))

	for _, mib := range snmpAgent.AddMibs {
		device.SNMPConfig.AddMibs = append(device.SNMPConfig.AddMibs, AddMib{
			OID:   mib.OID,
			Type:  mib.Type,
			Value: mib.Value,
		})
	}
}

// parseSNMPCommunityIncludes parses community-specific walk includes.
func parseSNMPCommunityIncludes(device *Device, snmpAgent *converter.SnmpAgent, includePath, deviceName string) error {
	for _, include := range snmpAgent.CommunityIncludes {
		walkFile, err := validateWalkFilePath(includePath, include.WalkFile, deviceName)
		if err != nil {
			return err
		}

		device.SNMPConfig.CommunityIncludes = append(device.SNMPConfig.CommunityIncludes, CommunityInclude{
			Community: include.Community,
			WalkFile:  walkFile,
		})
	}

	return nil
}

// parseSNMPAccessList parses the SNMP access list.
func parseSNMPAccessList(device *Device, snmpAgent *converter.SnmpAgent) {
	for _, ipStr := range snmpAgent.AccessList {
		if ip := net.ParseIP(ipStr); ip != nil {
			device.SNMPConfig.AccessList = append(device.SNMPConfig.AccessList, ip)
		}
	}
}

// parseSNMPAddr parses the SNMP address mapping.
func parseSNMPAddr(device *Device, snmpAgent *converter.SnmpAgent, deviceName string) error {
	if snmpAgent.SnmpAddr == "" {
		return nil
	}

	ip := net.ParseIP(snmpAgent.SnmpAddr)
	if ip == nil {
		return fmt.Errorf("device %s: %w: %s", deviceName, ErrInvalidSNMPAddr, snmpAgent.SnmpAddr)
	}

	device.SNMPConfig.SnmpAddr = ip

	return nil
}

// parseSNMPFdbTables parses Dot1D/Dot1Q FDB table configurations.
func parseSNMPFdbTables(device *Device, snmpAgent *converter.SnmpAgent) {
	if snmpAgent.Dot1DFdbTable != nil {
		device.SNMPConfig.Dot1DFdbTable = &FdbTableConfig{
			Port: snmpAgent.Dot1DFdbTable.Port,
			VLAN: snmpAgent.Dot1DFdbTable.VLAN,
		}
	}

	if snmpAgent.Dot1QFdbTable != nil {
		device.SNMPConfig.Dot1QFdbTable = &FdbTableConfig{
			Port: snmpAgent.Dot1QFdbTable.Port,
			VLAN: snmpAgent.Dot1QFdbTable.VLAN,
		}
	}
}

// parseDeviceProtocolConfigs parses all protocol configurations for a device.
func parseDeviceProtocolConfigs(device *Device, yamlDevice *converter.Device) error {
	// Handle DHCP configuration
	device.DHCPConfig = parseDHCPConfig(yamlDevice.Dhcp)

	// Handle DNS configuration
	var err error
	if device.DNSConfig, err = parseDNSConfig(yamlDevice.DNS, yamlDevice.Name); err != nil {
		return err
	}

	// Handle discovery protocols
	device.LLDPConfig = parseLLDPConfig(yamlDevice.Lldp)
	device.CDPConfig = parseCDPConfig(yamlDevice.Cdp)
	device.EDPConfig = parseEDPConfig(yamlDevice.Edp)
	device.FDPConfig = parseFDPConfig(yamlDevice.Fdp)
	device.STPConfig = parseSTPConfig(yamlDevice.Stp)

	// Handle service protocols
	device.HTTPConfig = parseHTTPConfig(yamlDevice.HTTP, device.Name)
	device.FTPConfig = parseFTPConfig(yamlDevice.Ftp, device.Name)
	device.NetBIOSConfig = parseNetBIOSConfig(yamlDevice.Netbios, device.Name)

	// Handle ICMP protocols
	device.ICMPConfig = parseICMPConfig(yamlDevice.Icmp)
	device.ICMPv6Config = parseICMPv6Config(yamlDevice.Icmpv6)

	// Handle DHCPv6 configuration
	device.DHCPv6Config = parseDHCPv6Config(yamlDevice.Dhcpv6)

	// Handle Traffic configuration
	device.TrafficConfig = parseTrafficConfig(yamlDevice.Traffic)

	// Handle OS Fingerprint configuration
	device.OSFingerprintConfig = parseOSFingerprintConfig(yamlDevice.OSFingerprint)

	// Handle iPerf3 configuration
	device.IPerf3 = parseIPerf3Config(yamlDevice.IPerf3)

	return nil
}

// parseOSFingerprintConfig parses OS fingerprinting configuration from YAML.
func parseOSFingerprintConfig(yamlOSFP *converter.OSFingerprintConfig) *OSFingerprintConfig {
	if yamlOSFP == nil {
		return nil
	}

	osFP := &OSFingerprintConfig{
		OSType:       yamlOSFP.OSType,
		TTL:          yamlOSFP.TTL,
		WindowSize:   yamlOSFP.WindowSize,
		WindowScale:  yamlOSFP.WindowScale,
		MSS:          yamlOSFP.MSS,
		SSHBanner:    yamlOSFP.SSHBanner,
		HTTPServer:   yamlOSFP.HTTPServer,
		FTPBanner:    yamlOSFP.FTPBanner,
		SMTPBanner:   yamlOSFP.SMTPBanner,
		TelnetBanner: yamlOSFP.TelnetBanner,
		DontFragment: yamlOSFP.DontFragment,
	}

	// Apply OS type defaults if no specific values are set
	if osFP.OSType != "" && osFP.TTL == 0 {
		switch osFP.OSType {
		case "linux", "macos", "freebsd", "openbsd":
			osFP.TTL = 64
		case "windows", "windows-server":
			osFP.TTL = 128
		case "cisco-ios", "cisco-nxos", "juniper-junos", "arista-eos":
			osFP.TTL = 255
		default:
			osFP.TTL = 64 // Default to Linux-like
		}
	}

	// Set default window size based on OS type if not specified
	if osFP.OSType != "" && osFP.WindowSize == 0 {
		switch osFP.OSType {
		case "linux":
			osFP.WindowSize = 29200
		case "windows", "windows-server":
			osFP.WindowSize = 65535
		case "macos":
			osFP.WindowSize = 65535
		default:
			osFP.WindowSize = 65535
		}
	}

	return osFP
}

// parseIPerf3Config parses iPerf3 server emulation configuration from YAML.
func parseIPerf3Config(yamlIPerf3 *converter.IPerf3Config) *IPerf3Config {
	if yamlIPerf3 == nil {
		return nil
	}

	cfg := &IPerf3Config{
		Enabled:           yamlIPerf3.Enabled,
		Port:              yamlIPerf3.Port,
		MaxBandwidthMbps:  yamlIPerf3.MaxBandwidthMbps,
		TypicalLatencyMs:  yamlIPerf3.TypicalLatencyMs,
		JitterMs:          yamlIPerf3.JitterMs,
		PacketLossPercent: yamlIPerf3.PacketLossPercent,
		UploadMbps:        yamlIPerf3.UploadMbps,
		DownloadMbps:      yamlIPerf3.DownloadMbps,
	}

	// Set defaults for unspecified values
	if cfg.Port == 0 {
		cfg.Port = 5201 // Default iPerf3 port
	}

	if cfg.UploadMbps == 0 {
		cfg.UploadMbps = 100.0 // Default 100 Mbps
	}

	if cfg.DownloadMbps == 0 {
		cfg.DownloadMbps = 100.0 // Default 100 Mbps
	}

	if cfg.MaxBandwidthMbps == 0 {
		cfg.MaxBandwidthMbps = 1000.0 // Default 1 Gbps max
	}

	if cfg.TypicalLatencyMs == 0 {
		cfg.TypicalLatencyMs = 1.0 // Default 1ms
	}

	return cfg
}

// parseNetBIOSConfig parses NetBIOS configuration from YAML.
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
		MsBrowse:  yamlNetbios.MsBrowse,
	}

	// Set defaults
	if netbiosCfg.Name == "" {
		netbiosCfg.Name = deviceName
		if len(netbiosCfg.Name) > netbiosMaxNameLen {
			netbiosCfg.Name = netbiosCfg.Name[:netbiosMaxNameLen]
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

	// Parse explicit NetBIOS names
	for _, name := range yamlNetbios.Names {
		entry := NetBIOSName{
			Name:  name.Name,
			Group: name.Group,
		}

		if name.Suffix != "" {
			// Parse suffix as hex (0x..) or decimal
			if strings.HasPrefix(strings.ToLower(name.Suffix), "0x") {
				if v, err := strconv.ParseUint(name.Suffix[2:], 16, 8); err == nil {
					entry.Suffix = uint8(v)
				}
			} else if v, err := strconv.ParseUint(name.Suffix, 10, 8); err == nil {
				entry.Suffix = uint8(v)
			}
		}

		netbiosCfg.Names = append(netbiosCfg.Names, entry)
	}

	return netbiosCfg
}

// parseICMPConfig parses ICMP configuration from YAML.
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

	if yamlIcmp.AddressMaskReply != "" {
		if ip := net.ParseIP(yamlIcmp.AddressMaskReply); ip != nil {
			icmpCfg.AddressMaskReply = ip.To4()
		}
	}

	if yamlIcmp.RouterAdvertisement != nil {
		ra := &IcmpRouterAdvertisement{
			Period:   yamlIcmp.RouterAdvertisement.Period,
			Lifetime: yamlIcmp.RouterAdvertisement.Lifetime,
		}

		for _, r := range yamlIcmp.RouterAdvertisement.Routers {
			if ip := net.ParseIP(r.Address); ip != nil {
				ra.Routers = append(ra.Routers, IcmpRouter{
					Address:    ip.To4(),
					Preference: r.Preference,
				})
			}
		}

		icmpCfg.RouterAdvertisement = ra
	}

	return icmpCfg
}

// parseICMPv6Config parses ICMPv6 configuration from YAML.
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

	if yamlIcmpv6.RouterAdvertisement != nil {
		ra := &Icmpv6RouterAdvertisement{
			Period:        yamlIcmpv6.RouterAdvertisement.Period,
			CurHopLimit:   yamlIcmpv6.RouterAdvertisement.CurHopLimit,
			Managed:       yamlIcmpv6.RouterAdvertisement.Managed,
			Other:         yamlIcmpv6.RouterAdvertisement.Other,
			Lifetime:      yamlIcmpv6.RouterAdvertisement.Lifetime,
			ReachableTime: yamlIcmpv6.RouterAdvertisement.ReachableTime,
			RetransTimer:  yamlIcmpv6.RouterAdvertisement.RetransTimer,
			MTU:           yamlIcmpv6.RouterAdvertisement.MTU,
		}

		for _, p := range yamlIcmpv6.RouterAdvertisement.PrefixInfo {
			var prefix net.IP
			if p.Prefix != "" {
				prefix = net.ParseIP(p.Prefix)
			}

			ra.PrefixInfo = append(ra.PrefixInfo, Icmpv6PrefixInfo{
				PrefixLength:      p.PrefixLength,
				Onlink:            p.Onlink,
				Auto:              p.Auto,
				ValidLifetime:     p.ValidLifetime,
				PreferredLifetime: p.PreferredLifetime,
				Prefix:            prefix,
			})
		}

		icmpv6Cfg.RouterAdvertisement = ra
	}

	return icmpv6Cfg
}

// parseDHCPv6Config parses DHCPv6 configuration from YAML.
// Returns an empty DHCPv6Config if input is nil (not an error condition).
func parseDHCPv6Config(yamlDhcpv6 *converter.Dhcpv6Config) *DHCPv6Config {
	if yamlDhcpv6 == nil {
		return &DHCPv6Config{}
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

	return dhcpv6Cfg
}

// parseTrafficConfig parses traffic configuration from YAML.
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

// parseSNMPTrapsConfig parses SNMP traps configuration from YAML.
func parseSNMPTrapsConfig(yamlTraps *converter.TrapsConfig) *TrapConfig {
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

	return trapsCfg
}

// parseDHCPConfig parses DHCP configuration from YAML
// Returns an empty DHCPConfig if input is nil (not an error condition).
func parseDHCPConfig(yamlDhcp *converter.DhcpServer) *DHCPConfig {
	if yamlDhcp == nil {
		return &DHCPConfig{}
	}

	dhcpCfg := &DHCPConfig{}

	parseDHCPBasicOptions(dhcpCfg, yamlDhcp)
	parseDHCPPoolConfig(dhcpCfg, yamlDhcp)
	parseDHCPv4Options(dhcpCfg, yamlDhcp)
	parseDHCPv6Options(dhcpCfg, yamlDhcp)
	parseDHCPClientLeases(dhcpCfg, yamlDhcp)

	return dhcpCfg
}

// parseDHCPBasicOptions parses basic DHCP options.
func parseDHCPBasicOptions(cfg *DHCPConfig, yamlDhcp *converter.DhcpServer) {
	if yamlDhcp.SubnetMask != "" {
		if ip := net.ParseIP(yamlDhcp.SubnetMask); ip != nil {
			cfg.SubnetMask = net.IPMask(ip.To4())
		}
	}

	if yamlDhcp.Router != "" {
		cfg.Router = net.ParseIP(yamlDhcp.Router)
	}

	if yamlDhcp.DomainNameServer != "" {
		if ip := net.ParseIP(yamlDhcp.DomainNameServer); ip != nil {
			cfg.DomainNameServer = append(cfg.DomainNameServer, ip)
		}
	}

	if yamlDhcp.ServerIdentifier != "" {
		cfg.ServerIdentifier = net.ParseIP(yamlDhcp.ServerIdentifier)
	}

	if yamlDhcp.NextServerIP != "" {
		cfg.NextServerIP = net.ParseIP(yamlDhcp.NextServerIP)
	}
}

// parseDHCPPoolConfig parses DHCP address pool configuration.
func parseDHCPPoolConfig(cfg *DHCPConfig, yamlDhcp *converter.DhcpServer) {
	if yamlDhcp.PoolStart != "" {
		cfg.PoolStart = net.ParseIP(yamlDhcp.PoolStart)
	}

	if yamlDhcp.PoolEnd != "" {
		cfg.PoolEnd = net.ParseIP(yamlDhcp.PoolEnd)
	}
}

// parseDHCPv4Options parses DHCPv4 high priority options.
func parseDHCPv4Options(cfg *DHCPConfig, yamlDhcp *converter.DhcpServer) {
	cfg.NTPServers = parseIPList(yamlDhcp.NTPServers)
	cfg.DomainSearch = yamlDhcp.DomainSearch
	cfg.TFTPServerName = yamlDhcp.TFTPServerName
	cfg.BootfileName = yamlDhcp.BootfileName

	if yamlDhcp.VendorSpecific != "" {
		cfg.VendorSpecific = []byte(yamlDhcp.VendorSpecific)
	}
}

// parseDHCPv6Options parses DHCPv6 options.
func parseDHCPv6Options(cfg *DHCPConfig, yamlDhcp *converter.DhcpServer) {
	cfg.SNTPServersV6 = parseIPList(yamlDhcp.SNTPServersV6)
	cfg.NTPServersV6 = parseIPList(yamlDhcp.NTPServersV6)
	cfg.SIPServersV6 = parseIPList(yamlDhcp.SIPServersV6)
	cfg.SIPDomainsV6 = yamlDhcp.SIPDomainsV6
}

// parseDHCPClientLeases parses static DHCP lease assignments.
func parseDHCPClientLeases(cfg *DHCPConfig, yamlDhcp *converter.DhcpServer) {
	for _, lease := range yamlDhcp.ClientLeases {
		dhcpLease := parseSingleDHCPLease(lease)
		if dhcpLease != nil {
			cfg.ClientLeases = append(cfg.ClientLeases, *dhcpLease)
		}
	}
}

// parseSingleDHCPLease parses a single DHCP lease entry.
func parseSingleDHCPLease(lease converter.DhcpLease) *DHCPLease {
	clientIP := net.ParseIP(lease.ClientIP)
	if clientIP == nil {
		return nil
	}

	macAddr, err := net.ParseMAC(lease.MacAddrValue)
	if err != nil {
		return nil
	}

	dhcpLease := &DHCPLease{
		ClientIP:   clientIP,
		MACAddress: macAddr,
	}

	if lease.MacAddrMask != "" {
		if mask, maskErr := net.ParseMAC(lease.MacAddrMask); maskErr == nil {
			dhcpLease.MACMask = mask
		}
	}

	return dhcpLease
}

// parseIPList parses a list of IP address strings into a [net.IP] slice.
func parseIPList(ipStrings []string) []net.IP {
	var ips []net.IP

	for _, ipStr := range ipStrings {
		if ip := net.ParseIP(ipStr); ip != nil {
			ips = append(ips, ip)
		}
	}

	return ips
}

// parseDNSConfig parses DNS configuration from YAML
// Returns an empty DNSConfig if input is nil (not an error condition).
func parseDNSConfig(yamlDNS *converter.DNSServer, deviceName string) (*DNSConfig, error) {
	if yamlDNS == nil {
		return &DNSConfig{}, nil
	}

	dnsCfg := &DNSConfig{}

	forwardRecords, err := parseDNSRecords(yamlDNS.ForwardRecords, deviceName)
	if err != nil {
		return nil, err
	}

	dnsCfg.ForwardRecords = forwardRecords

	reverseRecords, err := parseDNSRecords(yamlDNS.ReverseRecords, deviceName)
	if err != nil {
		return nil, err
	}

	dnsCfg.ReverseRecords = reverseRecords

	return dnsCfg, nil
}

// parseDNSRecords parses a list of DNS records with TTL validation.
func parseDNSRecords(records []converter.DNSRecord, deviceName string) ([]DNSRecord, error) {
	var result []DNSRecord

	for _, record := range records {
		ip := net.ParseIP(record.IP)
		if ip == nil {
			continue
		}

		ttl, err := validateDNSTTL(record.TTL, deviceName)
		if err != nil {
			return nil, err
		}

		result = append(result, DNSRecord{
			Name:  record.Name,
			IP:    ip,
			TTL:   ttl,
			RCode: record.RCode,
		})
	}

	return result, nil
}

// validateDNSTTL validates and returns the DNS TTL value.
func validateDNSTTL(ttl int, deviceName string) (uint32, error) {
	if ttl <= 0 {
		return uint32(DefaultDNSTTL), nil
	}

	if ttl < 0 {
		return 0, fmt.Errorf("device %s: %w: %d", deviceName, ErrDNSTTLNegative, ttl)
	}

	if ttl > maxDNSTTL {
		return 0, fmt.Errorf("device %s: %w: %d", deviceName, ErrDNSTTLExceedsMax, ttl)
	}

	return uint32(ttl), nil
}

// parseLLDPConfig parses LLDP configuration from YAML.
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

// parseCDPConfig parses CDP configuration from YAML.
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

// parseEDPConfig parses EDP configuration from YAML.
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

// parseFDPConfig parses FDP configuration from YAML.
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

// parseSTPConfig parses STP configuration from YAML.
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

// parseHTTPConfig parses HTTP configuration from YAML.
func parseHTTPConfig(yamlHTTP *converter.HTTPConfig, _ string) *HTTPConfig {
	if yamlHTTP == nil {
		return nil
	}

	httpCfg := &HTTPConfig{
		Enabled:    yamlHTTP.Enabled,
		ServerName: yamlHTTP.ServerName,
		Endpoints:  make([]HTTPEndpoint, 0),
	}
	// Set default server name if not specified
	if httpCfg.ServerName == "" {
		httpCfg.ServerName = "NIAC-Go/1.0.0"
	}
	// Parse endpoints
	for _, ep := range yamlHTTP.Endpoints {
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

// parseFTPConfig parses FTP configuration from YAML.
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

// validateWalkFilePath validates and resolves SNMP walk file paths
// Prevents path traversal attacks and ensures file exists.
func validateWalkFilePath(basePath, walkFile, deviceName string) (string, error) {
	// Clean the path to normalize it FIRST
	cleanPath := filepath.Clean(walkFile)

	// Security: Check for traversal AFTER cleaning
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("device %s: %w: %s", deviceName, ErrPathTraversalDetected, walkFile)
	}

	// Build full path
	var fullPath string

	switch {
	case filepath.IsAbs(cleanPath):
		fullPath = cleanPath
	case basePath != "":
		fullPath = filepath.Join(basePath, cleanPath)
	default:
		fullPath = cleanPath
	}

	// CRITICAL: Verify resolved path stays within base directory
	if basePath != "" {
		absBase, err := filepath.Abs(basePath)
		if err != nil {
			return "", fmt.Errorf("device %s: invalid base path: %w", deviceName, err)
		}

		absFull, err := filepath.Abs(fullPath)
		if err != nil {
			return "", fmt.Errorf("device %s: invalid file path: %w", deviceName, err)
		}

		// Ensure path starts with base (add separator to prevent partial match)
		if !strings.HasPrefix(absFull+string(filepath.Separator), absBase+string(filepath.Separator)) {
			return "", fmt.Errorf("device %s: %w: %s", deviceName, ErrPathOutsideBaseDir, walkFile)
		}
	}

	// Use Lstat to detect symlinks (doesn't follow them)
	info, err := os.Lstat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("device %s: %w: %s", deviceName, ErrWalkFileNotFound, fullPath)
		}

		return "", fmt.Errorf("device %s: cannot access walk file %s: %w", deviceName, fullPath, err)
	}

	// Reject symlinks for security
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("device %s: %w: %s", deviceName, ErrWalkFileIsSymlink, fullPath)
	}

	// Verify it's a regular file, not a directory or device
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("device %s: %w: %s", deviceName, ErrWalkFileNotRegular, fullPath)
	}

	return fullPath, nil
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
