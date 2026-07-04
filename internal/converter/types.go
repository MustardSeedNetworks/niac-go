package converter

import (
	"errors"

	"gopkg.in/yaml.v3"
)

// VLANTag is a segment VLAN tag. It accepts both YAML strings ("untagged",
// "200") and bare integers (200), so a config author can write `tag: 200`
// unquoted or `tag: untagged`. The raw scalar text is preserved for the loader
// to interpret.
type VLANTag string

// UnmarshalYAML reads the raw scalar so an int or a string tag both work.
func (t *VLANTag) UnmarshalYAML(node *yaml.Node) error {
	*t = VLANTag(node.Value)

	return nil
}

// Sentinel errors for converter.
var (
	ErrInvalidLoopTimeFormat      = errors.New("invalid LoopTime format")
	ErrInvalidScaleTimeFormat     = errors.New("invalid ScaleTime format")
	ErrInvalidVlanFormat          = errors.New("invalid Vlan format")
	ErrDeviceMissingMAC           = errors.New("device missing MAC address")
	ErrAddMibMissingOID           = errors.New("AddMib missing OID")
	ErrAddMibMissingType          = errors.New("AddMib missing type")
	ErrCapturePlaybackMissingFile = errors.New("CapturePlayback missing file name")
)

// Parser constants for field counts and lengths.
const (
	addMibQuotedArgs       = 3  // number of quoted args in AddMib directive
	minRegexMatchParts     = 2  // minimum parts from regex match
	ttlFieldCount          = 3  // TTL has ttl, ip, mask
	routerFieldCount       = 2  // Router has address, preference
	addMibFieldCount       = 3  // AddMib has OID, type, value
	communityIncludeFields = 2  // CommunityInclude has community, walkfile
	dnsPartsWithTTL        = 3  // DNS record with TTL
	dnsPartsWithRCode      = 4  // DNS record with RCode
	macAddressRawLen       = 12 // MAC address hex chars (XXXXXXXXXXXX)
)

// Config represents the YAML configuration structure.
type Config struct {
	IncludePath        string              `yaml:"include_path,omitempty"`
	CapturePlaybacks   []CapturePlayback   `yaml:"capture_playbacks,omitempty"   validate:"omitempty,dive"`
	DiscoveryProtocols *DiscoveryProtocols `yaml:"discovery_protocols,omitempty"`
	Devices            []Device            `yaml:"devices"                       validate:"omitempty,dive"`
	// Segments binds independent device sets to VLAN tags for multi-VLAN
	// playback (ADR 0008). When present, each segment is served as its own
	// isolated network (own IP space) on its tag, and top-level `devices` must
	// be empty. When absent, `devices` is served as a single untagged segment —
	// today's flat behavior.
	Segments []Segment `yaml:"segments,omitempty" validate:"omitempty,dive"`
}

// Segment binds a device set to a VLAN tag for multi-VLAN playback. Exactly one
// of Devices (inline) or Config (a path to a config file) is set.
type Segment struct {
	// Tag is "untagged" (the native VLAN) or a VLAN id in 1..4094.
	Tag VLANTag `yaml:"tag" validate:"required"`
	// Devices is an inline device set for this segment.
	Devices []Device `yaml:"devices,omitempty" validate:"omitempty,dive"`
	// Config is a path to a config file whose devices form this segment
	// (resolved by the loader). Mutually exclusive with Devices.
	Config string `yaml:"config,omitempty"`
}

// DiscoveryProtocols configures discovery protocol behavior.
type DiscoveryProtocols struct {
	LLDP *ProtocolConfig `yaml:"lldp,omitempty"`
	CDP  *ProtocolConfig `yaml:"cdp,omitempty"`
	EDP  *ProtocolConfig `yaml:"edp,omitempty"`
	FDP  *ProtocolConfig `yaml:"fdp,omitempty"`
}

// ProtocolConfig configures a discovery protocol.
type ProtocolConfig struct {
	Enabled  bool `yaml:"enabled"`
	Interval int  `yaml:"interval,omitempty"` // Advertisement interval in seconds
}

// CapturePlayback represents PCAP playback configuration.
type CapturePlayback struct {
	FileName  string  `yaml:"file_name"            validate:"required"`
	LoopTime  int     `yaml:"loop_time,omitempty"  validate:"omitempty,gte=0"`
	ScaleTime float64 `yaml:"scale_time,omitempty" validate:"omitempty,gte=0"`
}

// Device represents a network device.
type Device struct {
	Name          string               `yaml:"name,omitempty"`
	Type          string               `yaml:"type,omitempty"           validate:"omitempty,oneof=router switch ap access-point firewall server host workstation iot"`
	MAC           string               `yaml:"mac"                      validate:"required,mac"`
	IPs           []string             `yaml:"ips,omitempty"            validate:"omitempty,dive,ip"`
	VLAN          int                  `yaml:"vlan,omitempty"           validate:"omitempty,gte=1,lte=4094"`
	MapToIP       string               `yaml:"map_to_ip,omitempty"      validate:"omitempty,ip"`
	Babble        bool                 `yaml:"babble,omitempty"`
	TTL           *TTLConfig           `yaml:"ttl,omitempty"`
	SnmpAgent     *SnmpAgent           `yaml:"snmp_agent,omitempty"`
	Dhcp          *DhcpServer          `yaml:"dhcp,omitempty"`
	DNS           *DNSServer           `yaml:"dns,omitempty"`
	Lldp          *LldpConfig          `yaml:"lldp,omitempty"`
	Cdp           *CdpConfig           `yaml:"cdp,omitempty"`
	Edp           *EdpConfig           `yaml:"edp,omitempty"`
	Fdp           *FdpConfig           `yaml:"fdp,omitempty"`
	Stp           *StpConfig           `yaml:"stp,omitempty"`
	HTTP          *HTTPConfig          `yaml:"http,omitempty"`
	Ftp           *FtpConfig           `yaml:"ftp,omitempty"`
	Netbios       *NetbiosConfig       `yaml:"netbios,omitempty"`
	Bgp           *BgpConfig           `yaml:"bgp,omitempty"`    // v0.86.0 — Pro
	Ospf          *OspfConfig          `yaml:"ospf,omitempty"`   // v0.86.0 — Pro
	Snmpv3        *Snmpv3Config        `yaml:"snmpv3,omitempty"` // v0.86.0 — free (safe SNMP variant)
	Icmp          *IcmpConfig          `yaml:"icmp,omitempty"`
	Icmpv6        *Icmpv6Config        `yaml:"icmpv6,omitempty"`
	Dhcpv6        *Dhcpv6Config        `yaml:"dhcpv6,omitempty"`
	Traffic       *TrafficConfig       `yaml:"traffic,omitempty"`        // v1.6.0
	OSFingerprint *OSFingerprintConfig `yaml:"os_fingerprint,omitempty"` // v1.24.0
	IPerf3        *IPerf3Config        `yaml:"iperf3,omitempty"`         // v1.25.0
	Reflector     *ReflectorConfig     `yaml:"reflector,omitempty"`      // v0.94.0 — NetAlly UDP reflector endpoint
	Interfaces    []Interface          `yaml:"interfaces,omitempty"`     // Device interface metadata
	TrunkPorts    []TrunkPort          `yaml:"trunk_ports,omitempty"`    // v1.23.0 — topology link declarations
	PortChannels  []PortChannel        `yaml:"port_channels,omitempty"`  // v1.23.0 — LAG bundling
	// Properties is a free-form vendor-metadata block used by the
	// vendor template pack (cmd/niac/templates/vendor-templates) to
	// carry vendor / model / OS / port-config strings into the
	// device list. Keys are author-defined; consumers treat unknown
	// keys as informational.
	Properties map[string]string `yaml:"properties,omitempty"`
}

// Interface represents configured metadata for a device interface.
type Interface struct {
	Name        string `yaml:"name"`
	Speed       int    `yaml:"speed,omitempty"` // Mbps
	Duplex      string `yaml:"duplex,omitempty"`
	AdminStatus string `yaml:"admin_status,omitempty"`
	OperStatus  string `yaml:"oper_status,omitempty"`
	Description string `yaml:"description,omitempty"`
	VLANs       []int  `yaml:"vlans,omitempty"`
}

// TrunkPort declares a VLAN-tagged trunk link to a neighbouring device.
// The remote_device field is what BuildTopology uses to draw an edge
// between two devices on the topology graph.
type TrunkPort struct {
	Interface       string `yaml:"interface"`
	VLANs           []int  `yaml:"vlans,omitempty"`
	NativeVLAN      int    `yaml:"native_vlan,omitempty"`
	RemoteDevice    string `yaml:"remote_device,omitempty"`
	RemoteInterface string `yaml:"remote_interface,omitempty"`
}

// PortChannel declares a Link Aggregation (LACP) bundle. Doesn't
// produce topology edges on its own — trunk_ports referencing
// "port-channel<id>" as their interface are what surface edges.
type PortChannel struct {
	ID      int      `yaml:"id"`
	Members []string `yaml:"members,omitempty"`
	Mode    string   `yaml:"mode,omitempty"`
}

// IPerf3Config represents iPerf3 server emulation configuration.
type IPerf3Config struct {
	Enabled           bool    `yaml:"enabled,omitempty"`
	Port              uint16  `yaml:"port,omitempty"                validate:"omitempty,gte=1"`
	MaxBandwidthMbps  float64 `yaml:"max_bandwidth_mbps,omitempty"`
	TypicalLatencyMs  float64 `yaml:"typical_latency_ms,omitempty"`
	JitterMs          float64 `yaml:"jitter_ms,omitempty"`
	PacketLossPercent float64 `yaml:"packet_loss_percent,omitempty"`
	UploadMbps        float64 `yaml:"upload_mbps,omitempty"`
	DownloadMbps      float64 `yaml:"download_mbps,omitempty"`
}

// ReflectorConfig represents a NetAlly-style UDP reflector endpoint. When
// present, the device echoes signed reflector probes (TrueSpeed / performance
// tests) back to the sender with source/destination swapped, matching the
// niac-java Reflector() section. The presence of this block enables the
// reflector; there is no separate enable flag (Java sets isReflector on the
// section being parsed).
type ReflectorConfig struct {
	// LatencyMs delays the reflected packet by this many milliseconds,
	// simulating one-way path latency (Java Latency()). 0 = reflect
	// immediately.
	LatencyMs int `yaml:"latency_ms,omitempty" validate:"omitempty,gte=0,lte=60000"`
	// JitterMs randomises the delay by +/- this many milliseconds around
	// LatencyMs (Java Jitter()). 0 = no jitter.
	JitterMs int `yaml:"jitter_ms,omitempty" validate:"omitempty,gte=0,lte=60000"`
	// DSCP selects which ToS bits are toggled on the reflected packet:
	// true wiggles the bottom two DSCP bits (0x03, Java Dscp), false
	// wiggles the IP-precedence bit (0x01, Java IpPrecedence — the default).
	DSCP bool `yaml:"dscp,omitempty"`
}

// OSFingerprintConfig represents OS fingerprinting configuration for device simulation.
type OSFingerprintConfig struct {
	OSType       string `yaml:"os_type,omitempty"`       // e.g., "linux", "windows", "cisco-ios", "juniper-junos"
	TTL          uint8  `yaml:"ttl,omitempty"`           // Default IP TTL (Linux=64, Windows=128, Cisco=255)
	WindowSize   uint16 `yaml:"window_size,omitempty"`   // TCP window size
	WindowScale  uint8  `yaml:"window_scale,omitempty"`  // TCP window scale option
	MSS          uint16 `yaml:"mss,omitempty"`           // TCP maximum segment size
	SSHBanner    string `yaml:"ssh_banner,omitempty"`    // SSH version banner
	HTTPServer   string `yaml:"http_server,omitempty"`   // HTTP Server header
	FTPBanner    string `yaml:"ftp_banner,omitempty"`    // FTP welcome banner
	SMTPBanner   string `yaml:"smtp_banner,omitempty"`   // SMTP banner
	TelnetBanner string `yaml:"telnet_banner,omitempty"` // Telnet banner
	DontFragment bool   `yaml:"dont_fragment,omitempty"` // IP DF bit (Linux=true, Windows=false usually)
}

// SnmpAgent represents SNMP agent configuration.
type SnmpAgent struct {
	WalkFile          string             `yaml:"walk_file,omitempty"`
	WalkFiles         []string           `yaml:"walk_files,omitempty"`
	AddMibs           []AddMib           `yaml:"add_mibs,omitempty"           validate:"omitempty,dive"`
	CommunityIncludes []CommunityInclude `yaml:"community_includes,omitempty" validate:"omitempty,dive"`
	AccessList        []string           `yaml:"access_list,omitempty"`
	SnmpAddr          string             `yaml:"snmp_addr,omitempty"`
	Dot1DFdbTable     *FdbTableConfig    `yaml:"dot1d_fdb_table,omitempty"`
	Dot1QFdbTable     *FdbTableConfig    `yaml:"dot1q_fdb_table,omitempty"`
	Traps             *TrapsConfig       `yaml:"traps,omitempty"` // v1.6.0
}

// AddMib represents a MIB override or addition.
type AddMib struct {
	OID   string `yaml:"oid"   validate:"required"`
	Type  string `yaml:"type"  validate:"required"`
	Value string `yaml:"value" validate:"required"`
}

// CommunityInclude represents a community-specific walk include.
type CommunityInclude struct {
	Community string `yaml:"community" validate:"required"`
	WalkFile  string `yaml:"walk_file" validate:"required"`
}

// FdbTableConfig configures SNMP forwarding database table injection.
type FdbTableConfig struct {
	Port int `yaml:"port,omitempty"`
	VLAN int `yaml:"vlan,omitempty"`
}

// TTLConfig configures ICMP TTL timeout behavior (traceroute simulation).
type TTLConfig struct {
	TTL  int    `yaml:"ttl,omitempty"`
	IP   string `yaml:"ip,omitempty"`
	Mask string `yaml:"mask,omitempty"`
}

// DhcpServer represents DHCP server configuration.
type DhcpServer struct {
	ClientLeases     []DhcpLease `yaml:"client_leases,omitempty"      validate:"omitempty,dive"`
	SubnetMask       string      `yaml:"subnet_mask,omitempty"`
	Router           string      `yaml:"router,omitempty"`
	DomainNameServer string      `yaml:"domain_name_server,omitempty"`
	NextServerIP     string      `yaml:"next_server_ip,omitempty"`
	ServerIdentifier string      `yaml:"server_identifier,omitempty"`
	// Pool configuration
	PoolStart string `yaml:"pool_start,omitempty"` // Start of DHCP address pool
	PoolEnd   string `yaml:"pool_end,omitempty"`   // End of DHCP address pool
	// DHCPv4 high priority options
	NTPServers     []string `yaml:"ntp_servers,omitempty"`      // Option 42
	DomainSearch   []string `yaml:"domain_search,omitempty"`    // Option 119
	TFTPServerName string   `yaml:"tftp_server_name,omitempty"` // Option 66
	BootfileName   string   `yaml:"bootfile_name,omitempty"`    // Option 67
	VendorSpecific string   `yaml:"vendor_specific,omitempty"`  // Option 43 (hex string)
	// DHCPv6 options
	SNTPServersV6 []string `yaml:"sntp_servers_v6,omitempty"` // Option 31
	NTPServersV6  []string `yaml:"ntp_servers_v6,omitempty"`  // Option 56
	SIPServersV6  []string `yaml:"sip_servers_v6,omitempty"`  // Option 22
	SIPDomainsV6  []string `yaml:"sip_domains_v6,omitempty"`  // Option 21
}

// DhcpLease represents a DHCP client lease.
type DhcpLease struct {
	ClientIP     string `yaml:"client_ip"                validate:"required,ip"`
	MacAddrValue string `yaml:"mac_addr_value,omitempty"`
	MacAddrMask  string `yaml:"mac_addr_mask,omitempty"`
}

// DNSServer represents DNS server configuration.
type DNSServer struct {
	ForwardRecords []DNSRecord `yaml:"forward_records,omitempty" validate:"omitempty,dive"`
	ReverseRecords []DNSRecord `yaml:"reverse_records,omitempty" validate:"omitempty,dive"`
}

// DNSRecord represents a DNS A or PTR record.
type DNSRecord struct {
	Name  string `yaml:"name"            validate:"required"`
	IP    string `yaml:"ip"              validate:"required,ip"`
	TTL   int    `yaml:"ttl,omitempty"   validate:"omitempty,gte=0"`
	RCode int    `yaml:"rcode,omitempty" validate:"omitempty,gte=0,lte=15"`
}

// LldpConfig represents LLDP discovery protocol configuration.
type LldpConfig struct {
	Enabled           bool   `yaml:"enabled,omitempty"`
	AdvertiseInterval int    `yaml:"advertise_interval,omitempty"`
	TTL               int    `yaml:"ttl,omitempty"`
	SystemDescription string `yaml:"system_description,omitempty"`
	PortDescription   string `yaml:"port_description,omitempty"`
	ChassisIDType     string `yaml:"chassis_id_type,omitempty"`
}

// CdpConfig represents CDP discovery protocol configuration.
type CdpConfig struct {
	Enabled           bool   `yaml:"enabled,omitempty"`
	AdvertiseInterval int    `yaml:"advertise_interval,omitempty"`
	Holdtime          int    `yaml:"holdtime,omitempty"`
	Version           int    `yaml:"version,omitempty"`
	SoftwareVersion   string `yaml:"software_version,omitempty"`
	Platform          string `yaml:"platform,omitempty"`
	PortID            string `yaml:"port_id,omitempty"`
}

// EdpConfig represents EDP discovery protocol configuration.
type EdpConfig struct {
	Enabled           bool   `yaml:"enabled,omitempty"`
	AdvertiseInterval int    `yaml:"advertise_interval,omitempty"`
	VersionString     string `yaml:"version_string,omitempty"`
	DisplayString     string `yaml:"display_string,omitempty"`
}

// FdpConfig represents FDP discovery protocol configuration.
type FdpConfig struct {
	Enabled           bool   `yaml:"enabled,omitempty"`
	AdvertiseInterval int    `yaml:"advertise_interval,omitempty"`
	Holdtime          int    `yaml:"holdtime,omitempty"`
	SoftwareVersion   string `yaml:"software_version,omitempty"`
	Platform          string `yaml:"platform,omitempty"`
	PortID            string `yaml:"port_id,omitempty"`
}

// StpConfig represents STP/RSTP/MSTP configuration.
type StpConfig struct {
	Enabled        bool   `yaml:"enabled,omitempty"`
	BridgePriority uint16 `yaml:"bridge_priority,omitempty"`
	HelloTime      uint16 `yaml:"hello_time,omitempty"`
	MaxAge         uint16 `yaml:"max_age,omitempty"`
	ForwardDelay   uint16 `yaml:"forward_delay,omitempty"`
	Version        string `yaml:"version,omitempty"`
}

// HTTPConfig represents HTTP server configuration.
type HTTPConfig struct {
	Enabled    bool           `yaml:"enabled,omitempty"`
	ServerName string         `yaml:"server_name,omitempty"`
	Endpoints  []HTTPEndpoint `yaml:"endpoints,omitempty"`
}

// HTTPEndpoint represents an HTTP endpoint configuration.
type HTTPEndpoint struct {
	Path        string `yaml:"path,omitempty"`
	Method      string `yaml:"method,omitempty"`
	StatusCode  int    `yaml:"status_code,omitempty"`
	ContentType string `yaml:"content_type,omitempty"`
	Body        string `yaml:"body,omitempty"`
}

// FtpConfig represents FTP server configuration.
type FtpConfig struct {
	Enabled        bool      `yaml:"enabled,omitempty"`
	WelcomeBanner  string    `yaml:"welcome_banner,omitempty"`
	SystemType     string    `yaml:"system_type,omitempty"`
	AllowAnonymous bool      `yaml:"allow_anonymous,omitempty"`
	Users          []FtpUser `yaml:"users,omitempty"`
}

// FtpUser represents an FTP user account.
type FtpUser struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	HomeDir  string `yaml:"home_dir,omitempty"`
}

// NetbiosConfig represents NetBIOS service configuration.
type NetbiosConfig struct {
	Enabled   bool          `yaml:"enabled,omitempty"`
	Name      string        `yaml:"name,omitempty"`
	Workgroup string        `yaml:"workgroup,omitempty"`
	NodeType  string        `yaml:"node_type,omitempty"`
	Services  []string      `yaml:"services,omitempty"`
	TTL       uint32        `yaml:"ttl,omitempty"`
	Names     []NetbiosName `yaml:"names,omitempty"`
	MsBrowse  bool          `yaml:"msbrowse,omitempty"`
}

// NetbiosName represents a NetBIOS name entry.
type NetbiosName struct {
	Name   string `yaml:"name,omitempty"`
	Suffix string `yaml:"suffix,omitempty"`
	Group  bool   `yaml:"group,omitempty"`
}

// BgpConfig represents BGP (Border Gateway Protocol) speaker configuration.
//
// Added 2026-05-27 for license-gating purposes. Presence on a device
// activates the bgp Pro feature gate; the protocol speaker itself is
// a separate workstream (RFC 4271).
type BgpConfig struct {
	Enabled   bool          `yaml:"enabled,omitempty"`
	ASN       uint32        `yaml:"asn,omitempty"        validate:"omitempty,gte=1,lte=4294967295"`
	RouterID  string        `yaml:"router_id,omitempty"  validate:"omitempty,ip"`
	HoldTime  uint16        `yaml:"hold_time,omitempty"`  // seconds (default 180)
	KeepAlive uint16        `yaml:"keep_alive,omitempty"` // seconds (default 60)
	Neighbors []BgpNeighbor `yaml:"neighbors,omitempty"`
}

// BgpNeighbor represents one BGP peer entry.
type BgpNeighbor struct {
	IP       string `yaml:"ip"                 validate:"required,ip"`
	RemoteAS uint32 `yaml:"remote_as"          validate:"required,gte=1"`
	Password string `yaml:"password,omitempty"`
}

// OspfConfig represents OSPF (Open Shortest Path First) configuration.
//
// Added 2026-05-27 for license-gating purposes. Presence on a device
// activates the ospf Pro feature gate; the protocol speaker itself is
// a separate workstream (RFC 2328).
type OspfConfig struct {
	Enabled   bool       `yaml:"enabled,omitempty"`
	RouterID  string     `yaml:"router_id,omitempty"  validate:"omitempty,ip"`
	HelloTime uint16     `yaml:"hello_time,omitempty"` // seconds (default 10)
	DeadTime  uint16     `yaml:"dead_time,omitempty"`  // seconds (default 40)
	Areas     []OspfArea `yaml:"areas,omitempty"`
}

// OspfArea represents an OSPF area declaration.
type OspfArea struct {
	AreaID   string   `yaml:"area_id"            validate:"required"`
	Networks []string `yaml:"networks,omitempty"`
	Stub     bool     `yaml:"stub,omitempty"`
}

// Snmpv3Config represents SNMPv3 user / auth / priv configuration.
//
// Added 2026-05-27. NOT license-gated — SNMPv3 is the only safe SNMP
// version (v1/v2c send credentials in cleartext) and is free for all
// NIAC tiers. The actual SNMPv3 packet path lives in
// internal/protocols/snmp/.
type Snmpv3Config struct {
	Enabled  bool         `yaml:"enabled,omitempty"`
	EngineID string       `yaml:"engine_id,omitempty"`
	Users    []Snmpv3User `yaml:"users,omitempty"`
}

// Snmpv3User represents one SNMPv3 USM user record.
type Snmpv3User struct {
	Username     string `yaml:"username"                validate:"required"`
	AuthProtocol string `yaml:"auth_protocol,omitempty" validate:"omitempty,oneof=none md5 sha sha256 sha512"`
	AuthPassword string `yaml:"auth_password,omitempty"`
	PrivProtocol string `yaml:"priv_protocol,omitempty" validate:"omitempty,oneof=none des aes aes192 aes256"`
	PrivPassword string `yaml:"priv_password,omitempty"`
}

// IcmpConfig represents ICMP/ICMPv4 configuration.
type IcmpConfig struct {
	Enabled             bool                     `yaml:"enabled,omitempty"`
	TTL                 uint8                    `yaml:"ttl,omitempty"`
	RateLimit           int                      `yaml:"rate_limit,omitempty"`
	AddressMaskReply    string                   `yaml:"address_mask_reply,omitempty"`
	RouterAdvertisement *IcmpRouterAdvertisement `yaml:"router_advertisement,omitempty"`
}

// IcmpRouterAdvertisement configures IPv4 router advertisements.
type IcmpRouterAdvertisement struct {
	Period   int          `yaml:"period,omitempty"`
	Lifetime int          `yaml:"lifetime,omitempty"`
	Routers  []IcmpRouter `yaml:"routers,omitempty"`
}

// IcmpRouter represents an advertised router entry.
type IcmpRouter struct {
	Address    string `yaml:"address,omitempty"`
	Preference int    `yaml:"preference,omitempty"`
}

// Icmpv6Config represents ICMPv6 configuration.
type Icmpv6Config struct {
	Enabled             bool                       `yaml:"enabled,omitempty"`
	HopLimit            uint8                      `yaml:"hop_limit,omitempty"`
	RateLimit           int                        `yaml:"rate_limit,omitempty"`
	RouterAdvertisement *Icmpv6RouterAdvertisement `yaml:"router_advertisement,omitempty"`
}

// Icmpv6RouterAdvertisement configures IPv6 router advertisements.
type Icmpv6RouterAdvertisement struct {
	Period        int                `yaml:"period,omitempty"`
	CurHopLimit   int                `yaml:"cur_hop_limit,omitempty"`
	Managed       int                `yaml:"managed,omitempty"`
	Other         int                `yaml:"other,omitempty"`
	Lifetime      int                `yaml:"lifetime,omitempty"`
	ReachableTime int                `yaml:"reachable_time,omitempty"`
	RetransTimer  int                `yaml:"retrans_timer,omitempty"`
	MTU           int                `yaml:"mtu,omitempty"`
	PrefixInfo    []Icmpv6PrefixInfo `yaml:"prefix_info,omitempty"`
}

// Icmpv6PrefixInfo represents IPv6 prefix info options.
type Icmpv6PrefixInfo struct {
	PrefixLength      int    `yaml:"prefix_length,omitempty"`
	Onlink            int    `yaml:"onlink,omitempty"`
	Auto              int    `yaml:"auto,omitempty"`
	ValidLifetime     int    `yaml:"valid_lifetime,omitempty"`
	PreferredLifetime int    `yaml:"preferred_lifetime,omitempty"`
	Prefix            string `yaml:"prefix,omitempty"`
}

// Dhcpv6Config represents DHCPv6 server configuration.
type Dhcpv6Config struct {
	Enabled           bool         `yaml:"enabled,omitempty"`
	Pools             []Dhcpv6Pool `yaml:"pools,omitempty"`
	PreferredLifetime uint32       `yaml:"preferred_lifetime,omitempty"`
	ValidLifetime     uint32       `yaml:"valid_lifetime,omitempty"`
	Preference        uint8        `yaml:"preference,omitempty"`
	DNSServers        []string     `yaml:"dns_servers,omitempty"`
	DomainList        []string     `yaml:"domain_list,omitempty"`
	SNTPServers       []string     `yaml:"sntp_servers,omitempty"`
	NTPServers        []string     `yaml:"ntp_servers,omitempty"`
	SIPServers        []string     `yaml:"sip_servers,omitempty"`
	SIPDomains        []string     `yaml:"sip_domains,omitempty"`
}

// Dhcpv6Pool represents an IPv6 address pool.
type Dhcpv6Pool struct {
	Network    string `yaml:"network,omitempty"`
	RangeStart string `yaml:"range_start,omitempty"`
	RangeEnd   string `yaml:"range_end,omitempty"`
}

// TrafficConfig represents traffic pattern configuration (v1.6.0).
type TrafficConfig struct {
	Enabled          bool                   `yaml:"enabled,omitempty"`
	ARPAnnouncements *ARPAnnouncementConfig `yaml:"arp_announcements,omitempty"`
	PeriodicPings    *PeriodicPingConfig    `yaml:"periodic_pings,omitempty"`
	RandomTraffic    *RandomTrafficConfig   `yaml:"random_traffic,omitempty"`
}

// ARPAnnouncementConfig configures gratuitous ARP announcements.
type ARPAnnouncementConfig struct {
	Enabled  bool `yaml:"enabled,omitempty"`
	Interval int  `yaml:"interval,omitempty"` // seconds
}

// PeriodicPingConfig configures periodic ICMP echo requests.
type PeriodicPingConfig struct {
	Enabled     bool `yaml:"enabled,omitempty"`
	Interval    int  `yaml:"interval,omitempty"`     // seconds
	PayloadSize int  `yaml:"payload_size,omitempty"` // bytes
}

// RandomTrafficConfig configures random background traffic.
type RandomTrafficConfig struct {
	Enabled     bool     `yaml:"enabled,omitempty"`
	Interval    int      `yaml:"interval,omitempty"`     // seconds
	PacketCount int      `yaml:"packet_count,omitempty"` // packets per interval
	Patterns    []string `yaml:"patterns,omitempty"`     // traffic patterns
}

// TrapsConfig represents SNMP trap configuration (v1.6.0).
type TrapsConfig struct {
	Enabled               bool                 `yaml:"enabled,omitempty"`
	Receivers             []string             `yaml:"receivers,omitempty"`
	Community             string               `yaml:"community,omitempty"` // SNMP community string
	ColdStart             *TrapTriggerConfig   `yaml:"cold_start,omitempty"`
	LinkState             *LinkStateTrapConfig `yaml:"link_state,omitempty"`
	AuthenticationFailure *TrapTriggerConfig   `yaml:"authentication_failure,omitempty"`
	HighCPU               *ThresholdTrapConfig `yaml:"high_cpu,omitempty"`
	HighMemory            *ThresholdTrapConfig `yaml:"high_memory,omitempty"`
	InterfaceErrors       *ThresholdTrapConfig `yaml:"interface_errors,omitempty"`
}

// TrapTriggerConfig configures a simple trap trigger.
type TrapTriggerConfig struct {
	Enabled   bool `yaml:"enabled,omitempty"`
	OnStartup bool `yaml:"on_startup,omitempty"`
}

// LinkStateTrapConfig configures link up/down traps.
type LinkStateTrapConfig struct {
	Enabled  bool `yaml:"enabled,omitempty"`
	LinkDown bool `yaml:"link_down,omitempty"`
	LinkUp   bool `yaml:"link_up,omitempty"`
}

// ThresholdTrapConfig configures threshold-based traps.
type ThresholdTrapConfig struct {
	Enabled   bool `yaml:"enabled,omitempty"`
	Threshold int  `yaml:"threshold,omitempty"` // threshold value
	Interval  int  `yaml:"interval,omitempty"`  // check interval in seconds
}

// Parser handles parsing Java DSL format.
type Parser struct {
	lines   []string
	pos     int
	verbose bool
}

// maxInputFileSize is the maximum allowed input file size (10 MB).
const maxInputFileSize = 10 * 1024 * 1024
