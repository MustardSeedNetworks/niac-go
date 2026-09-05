package config

import (
	"errors"
	"net"
	"time"
)

// Sentinel errors for configuration validation.
var (
	ErrInvalidDeviceDeclaration = errors.New("invalid device declaration")
	ErrNoDevicesDefined         = errors.New("no devices defined in configuration")

	ErrSegmentsAndTopLevelDevices = errors.New("use either top-level devices or segments, not both")
	ErrSegmentDevicesXORConfig    = errors.New("segment must set exactly one of devices or config")
	ErrInvalidSegmentTag          = errors.New("invalid segment tag")
	ErrDuplicateSegmentTag        = errors.New("duplicate segment tag")
	ErrInvalidMapToIP             = errors.New("invalid map_to_ip")
	ErrInvalidTTLIP               = errors.New("invalid ttl ip")
	ErrInvalidTTLMask             = errors.New("invalid ttl mask")
	ErrInvalidIPAddress           = errors.New("invalid IP address")
	ErrInvalidSNMPAddr            = errors.New("invalid snmp_addr")
	ErrDNSTTLNegative             = errors.New("DNS record TTL cannot be negative")
	ErrDNSTTLExceedsMax           = errors.New("DNS record TTL exceeds maximum (2147483647)")
	ErrInsufficientFields         = errors.New("insufficient fields")
	ErrPathTraversalDetected      = errors.New("path traversal detected")
	ErrPathOutsideBaseDir         = errors.New("path outside base directory")
	ErrWalkFileNotFound           = errors.New("walk file not found")
	ErrWalkFileIsSymlink          = errors.New("walk file is a symlink (not allowed)")
	ErrWalkFileNotRegular         = errors.New("walk file is not a regular file")
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

	// DefaultSNMPCommunity is the default SNMP community string.
	DefaultSNMPCommunity = "public"

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
	IncludePath        string           // Base path for walk files
	CapturePlayback    *CapturePlayback // Optional PCAP playback config
	BehaviorTimelines  []BehaviorTimeline
	DiscoveryProtocols *DiscoveryProtocols // Discovery protocol configuration
	Segments           []Segment           // Multi-VLAN playback bindings (ADR 0008); empty = flat/untagged
	Networks           []Network
	Attachments        []LogicalAttachment
}

// BehaviorTimeline is one saved sequence replayed from simulation start.
type BehaviorTimeline struct {
	Name        string
	StartOffset time.Duration
	RepeatCount int
	Phases      []BehaviorPhase
}

// BehaviorPhase applies traffic and faults for one bounded interval.
type BehaviorPhase struct {
	Name        string
	StartOffset time.Duration
	Duration    time.Duration
	Reset       bool
	Traffic     []BehaviorTraffic
	Faults      []BehaviorFault
}

// BehaviorTraffic sets observable utilization on one interface.
type BehaviorTraffic struct {
	Device      string
	Interface   string
	Utilization int
}

// BehaviorFault sets one supported interface fault rate.
type BehaviorFault struct {
	Device    string
	Interface string
	Type      string
	Value     int
}

// Network declares one internal routed IPv4 network.
type Network struct {
	Name        string
	Subnet      string
	VirtualVLAN int
}

// LogicalAttachment identifies the virtual network exposed at start time.
type LogicalAttachment struct {
	Name    string
	Network string
}

// Route declares an IPv4 static route through a named device interface.
type Route struct {
	Destination string
	Via         string
	NextHop     string
}

// UntaggedTag is the Segment.Tag value for the native/untagged VLAN.
const UntaggedTag = 0

// Segment binds a resolved device set to a VLAN tag for multi-VLAN playback.
type Segment struct {
	Tag     int      // UntaggedTag (0) for the native VLAN, else a VLAN id 1..4094
	Devices []Device // Resolved device set for this segment
}

// DeviceCount returns the total number of devices this config describes,
// counting devices inside segments (a `segments:` config has no top-level
// Devices, so len(c.Devices) alone would report zero).
func (c *Config) DeviceCount() int {
	if len(c.Segments) == 0 {
		return len(c.Devices)
	}

	total := 0
	for _, seg := range c.Segments {
		total += len(seg.Devices)
	}

	return total
}

// NormalizedSegments returns the list of engine bindings this config describes:
// its explicit Segments if any, otherwise a single untagged segment wrapping the
// flat Devices list. This is the one place the "bare devices = one untagged
// segment" backward-compatibility rule lives, so every consumer sees a uniform
// list of (tag, device-set) engines.
func (c *Config) NormalizedSegments() []Segment {
	if len(c.Segments) > 0 {
		return c.Segments
	}

	return []Segment{{Tag: UntaggedTag, Devices: c.Devices}}
}

// DuplicateSegmentTags returns every tag mapped to its conflicting segment indices.
func DuplicateSegmentTags(segments []Segment) map[int][]int {
	indices := make(map[int][]int, len(segments))
	for index, segment := range segments {
		indices[segment.Tag] = append(indices[segment.Tag], index)
	}
	for tag, locations := range indices {
		if len(locations) == 1 {
			delete(indices, tag)
		}
	}
	return indices
}

// RateMode selects how replay paces its packet sends. ScaleTime applies only
// in RateTiming.
type RateMode string

const (
	// RateTiming (the default, also the zero value "") replays packets at their
	// original inter-packet spacing, optionally sped up/slowed by ScaleTime.
	RateTiming RateMode = "timing"
	// RateTopspeed sends packets back-to-back with no inter-packet delay,
	// ignoring the captured timestamps.
	RateTopspeed RateMode = "topspeed"
	// RatePPS paces sends to a fixed packets-per-second (PacketsPerSec),
	// ignoring the captured timestamps.
	RatePPS RateMode = "pps"
	// RateMbps paces sends to cap average throughput at MbpsCap megabits/sec,
	// ignoring the captured timestamps.
	RateMbps RateMode = "mbps"
)

// CapturePlayback represents PCAP file playback configuration.
type CapturePlayback struct {
	FileName string
	// RootDir, when set, is the validated directory the file must resolve
	// under. Playback opens the file through an os.Root anchored here, so
	// the kernel rejects any component (including an intermediate-directory
	// symlink) that escapes it — closing the validate→open TOCTOU across the
	// whole path. The API replay handler sets this to the allow-listed pcap
	// directory. When empty (operator-authored config / CLI paths), playback
	// falls back to rooting at the file's own parent directory.
	RootDir   string
	LoopTime  int     // milliseconds; inter-pass interval when looping
	ScaleTime float64 // time scaling factor (RateTiming only)
	// LoopCount bounds the number of replay passes: 0 keeps the existing
	// behavior (infinite when LoopTime>0, single-shot otherwise); N>0 stops
	// after N passes. With LoopTime==0 and LoopCount>0 the passes run
	// back-to-back.
	LoopCount int

	// RateMode selects the pacing strategy; empty means RateTiming.
	RateMode RateMode
	// PacketsPerSec is the target rate for RatePPS (packets/second).
	PacketsPerSec float64
	// MbpsCap is the throughput cap for RateMbps (megabits/second).
	MbpsCap float64
	// BPFFilter, when set, replays only packets matching this tcpdump-style
	// filter (e.g. "udp port 53"). Empty replays every packet.
	BPFFilter string
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
	MACVendor           string
	MACSuffix           uint32
	IPAddresses         []net.IP
	MapToIP             net.IP     // Map UDP traffic to external IP (Java MapToIp)
	Babble              bool       // Periodically emit babble traffic
	TTLConfig           *TTLConfig // ICMP TTL timeout behavior (traceroute simulation)
	VLAN                int        // Optional VLAN membership (Java Vlan)
	Interfaces          []Interface
	Routes              []Route
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
	MDNSConfig          *MDNSConfig          // multicast DNS (Bonjour/Avahi) advertisement
	SNMPv3Config        *SNMPv3Config        // SNMPv3 USM users (v0.86.0 — free, safe SNMP variant)
	ICMPConfig          *ICMPConfig          // ICMP/ICMPv4 configuration
	ICMPv6Config        *ICMPv6Config        // ICMPv6 configuration
	DHCPv6Config        *DHCPv6Config        // DHCPv6 server configuration
	OSFingerprintConfig *OSFingerprintConfig // OS fingerprinting configuration (v1.24.0)
	SSHConfig           *SSHConfig           // authenticated vendor-like command service
	SyslogConfig        *SyslogConfig        // configuration-state event output
	IPerf3              *IPerf3Config        // iPerf3 server emulation configuration (v1.25.0)
	ReflectorConfig     *ReflectorConfig     // NetAlly UDP reflector endpoint (v0.94.0)
	PortChannels        []PortChannel        // Port-channel/LAG configuration (v1.23.0)
	TrunkPorts          []TrunkPort          // Trunk port configuration (v1.23.0)
	Properties          map[string]string
}

// SSHConfig enables authenticated command access for a simulated device.
type SSHConfig struct {
	Enabled     bool
	Username    string
	PasswordEnv string
}

// SyslogConfig sends configuration-state messages to RFC 5424 collectors.
type SyslogConfig struct {
	Enabled   bool
	Receivers []string
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

// ReflectorConfig holds NetAlly UDP reflector settings for a device. A device
// with this set echoes signed reflector probes back to the sender (see
// UDPHandler reflect path). Presence enables the reflector.
type ReflectorConfig struct {
	LatencyMs int  // Delay before reflecting, in milliseconds (0 = immediate)
	JitterMs  int  // Random +/- delay around LatencyMs, in milliseconds
	DSCP      bool // true: toggle DSCP bottom-2 bits (0x03); false: IP-precedence bit (0x01)
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
	Name           string
	Type           string
	Network        string
	Address        string
	MTU            int
	Speed          int // Mbps
	Duplex         string
	AdminStatus    string // up, down, testing
	OperStatus     string // up, down, testing
	Description    string
	InUtilization  float64 // percentage of interface capacity
	OutUtilization float64 // percentage of interface capacity
	VLANs          []int
}

// SNMPConfig holds SNMP configuration.
type SNMPConfig struct {
	Enabled           *bool
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
	// MED carries the TIA-1057 extensions a phone, camera or access point
	// advertises. Nil means the device sends plain 802.1AB, which is what a
	// switch or router does.
	MED *LLDPMEDConfig
}

// LLDPMEDConfig holds the LLDP-MED (TIA-1057) extensions.
//
// These are what a discovery tool reads to tell a phone from a printer and to
// learn the voice VLAN it should be on. Without them an IP phone appears as an
// anonymous endpoint, so a tester checking that NIAC's hospital pack looks like
// a hospital sees a rack of unidentified MAC addresses.
type LLDPMEDConfig struct {
	// DeviceType is the MED device class: "endpoint_class1" (generic),
	// "endpoint_class2" (media), "endpoint_class3" (communication device, e.g.
	// a phone) or "network_connectivity" (the switch side).
	DeviceType string
	// NetworkPolicies are the per-application VLAN/priority/DSCP assignments
	// the device advertises or expects.
	NetworkPolicies []LLDPMEDNetworkPolicy
	// Power is the extended power-via-MDI advertisement. Nil when the device
	// says nothing about power.
	Power *LLDPMEDPower
	// Inventory is the hardware inventory set. Nil when the device advertises
	// none of it.
	Inventory *LLDPMEDInventory
}

// LLDPMEDNetworkPolicy is one TIA-1057 Network Policy TLV.
type LLDPMEDNetworkPolicy struct {
	// Application is the traffic class: "voice", "voice_signaling",
	// "guest_voice", "guest_voice_signaling", "softphone_voice",
	// "video_conferencing", "streaming_video" or "video_signaling".
	Application string
	// Unknown marks the policy as not yet known to the endpoint, which is how a
	// phone asks the switch what VLAN to use.
	Unknown bool
	// Tagged reports whether the application's frames carry an 802.1Q tag.
	Tagged bool
	// VLANID is the VLAN the application uses; 0 with Tagged false means the
	// device uses the port's untagged VLAN.
	VLANID int
	// Priority is the 802.1p user priority, 0-7.
	Priority int
	// DSCP is the DiffServ code point, 0-63.
	DSCP int
}

// LLDPMEDPower is the TIA-1057 Extended Power-via-MDI TLV.
type LLDPMEDPower struct {
	// DeviceType is "pse" (the switch supplying power) or "pd" (the powered
	// device drawing it).
	DeviceType string
	// Source is "unknown", "primary", "backup", "pse", "local" or "pse_local".
	Source string
	// Priority is "unknown", "critical", "high" or "low".
	Priority string
	// ValueTenthWatts is the power value in tenths of a watt, matching the
	// TLV's own unit so a config value and a captured frame read the same.
	ValueTenthWatts int
}

// LLDPMEDInventory is the TIA-1057 inventory set. Empty fields are omitted from
// the advertisement rather than sent blank.
type LLDPMEDInventory struct {
	HardwareRevision string
	FirmwareRevision string
	SoftwareRevision string
	SerialNumber     string
	Manufacturer     string
	ModelName        string
	AssetID          string
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

// MDNSConfig publishes a device on multicast DNS, the way Bonjour and Avahi
// announce a host and its services on the local link.
type MDNSConfig struct {
	Enabled  bool
	Hostname string // published as <Hostname>.local
	Services []MDNSService
	TTL      uint32
}

// MDNSService is one advertised DNS-SD service, such as _ipp._tcp.
type MDNSService struct {
	Type string
	Port uint16
	TXT  []string
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

// SNMPv3Config holds SNMPv3 USM (User-based Security Model)
// configuration: engine ID + users with auth/priv credentials.
//
// Added 2026-05-27 — NOT license-gated. SNMPv3 is the only safe SNMP
// version (v1/v2c send credentials in cleartext) and so stays free
// for every NIAC tier. The actual SNMP packet handling lives in
// internal/protocols/snmp/.
type SNMPv3Config struct {
	Enabled  bool
	EngineID string
	Users    []SNMPv3User
}

// SNMPv3User represents one SNMPv3 USM user record.
type SNMPv3User struct {
	Username     string
	AuthProtocol string // "none" | "md5" | "sha" | "sha256" | "sha512"
	AuthPassword string
	PrivProtocol string // "none" | "des" | "aes" | "aes192" | "aes256"
	PrivPassword string
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

// TrapConfig holds SNMP trap configuration (v1.6.0).
type TrapConfig struct {
	Enabled   bool
	Receivers []string // Trap receiver addresses (IP:port format)
	Community string   // SNMP community string (default: "public")
	// Version is "v2c" (the default) or "v3". A v3 notification is
	// authenticated and encrypted with a USM user from the device's snmpv3
	// block, which is what a manager configured for v3-only will accept.
	Version string
	// SecurityUser names the snmpv3 user a v3 notification is sent as. Empty
	// uses the device's first configured user.
	SecurityUser string
	// Inform sends an InformRequest instead of a trap. An inform is
	// acknowledged, so the sender knows the manager received it; a trap is
	// fire-and-forget and a dropped one is invisible.
	Inform bool
	// InformRetries is how many times an unacknowledged inform is resent.
	InformRetries int
	// InformTimeoutSeconds is how long to wait for an acknowledgement before
	// resending.
	InformTimeoutSeconds int
	ColdStart            *TrapTriggerConfig
	LinkState            *LinkStateTrapConfig
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
	FDBOnly         bool   // Learn the peer MAC without advertising an LLDP/CDP neighbor
}

// Load reads and parses a configuration file
// Automatically detects format based on file extension:
// - .yaml -> YAML format (converted from Java DSL)
