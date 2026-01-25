package config

import "net"

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
