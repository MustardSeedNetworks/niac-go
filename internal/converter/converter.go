// Package converter implements Java NIAC DSL to YAML conversion
package converter

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

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
	CapturePlaybacks   []CapturePlayback   `yaml:"capture_playbacks,omitempty"` // Changed to array
	DiscoveryProtocols *DiscoveryProtocols `yaml:"discovery_protocols,omitempty"`
	Devices            []Device            `yaml:"devices"`
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
	FileName  string  `yaml:"file_name"`
	LoopTime  int     `yaml:"loop_time,omitempty"`
	ScaleTime float64 `yaml:"scale_time,omitempty"`
}

// Device represents a network device.
type Device struct {
	Name          string               `yaml:"name,omitempty"`
	Type          string               `yaml:"type,omitempty"` // Device type: router, switch, ap, firewall, server, workstation, iot
	MAC           string               `yaml:"mac"`
	IP            string               `yaml:"ip,omitempty"`  // Single IP (backward compatible)
	IPs           []string             `yaml:"ips,omitempty"` // Multiple IPs (new feature)
	VLAN          int                  `yaml:"vlan,omitempty"`
	MapToIP       string               `yaml:"map_to_ip,omitempty"`
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
	Icmp          *IcmpConfig          `yaml:"icmp,omitempty"`
	Icmpv6        *Icmpv6Config        `yaml:"icmpv6,omitempty"`
	Dhcpv6        *Dhcpv6Config        `yaml:"dhcpv6,omitempty"`
	Traffic       *TrafficConfig       `yaml:"traffic,omitempty"`        // v1.6.0
	OSFingerprint *OSFingerprintConfig `yaml:"os_fingerprint,omitempty"` // v1.24.0
	IPerf3        *IPerf3Config        `yaml:"iperf3,omitempty"`         // v1.25.0
}

// IPerf3Config represents iPerf3 server emulation configuration.
type IPerf3Config struct {
	Enabled           bool    `yaml:"enabled,omitempty"`
	Port              uint16  `yaml:"port,omitempty"`
	MaxBandwidthMbps  float64 `yaml:"max_bandwidth_mbps,omitempty"`
	TypicalLatencyMs  float64 `yaml:"typical_latency_ms,omitempty"`
	JitterMs          float64 `yaml:"jitter_ms,omitempty"`
	PacketLossPercent float64 `yaml:"packet_loss_percent,omitempty"`
	UploadMbps        float64 `yaml:"upload_mbps,omitempty"`
	DownloadMbps      float64 `yaml:"download_mbps,omitempty"`
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
	AddMibs           []AddMib           `yaml:"add_mibs,omitempty"`
	CommunityIncludes []CommunityInclude `yaml:"community_includes,omitempty"`
	AccessList        []string           `yaml:"access_list,omitempty"`
	SnmpAddr          string             `yaml:"snmp_addr,omitempty"`
	Dot1DFdbTable     *FdbTableConfig    `yaml:"dot1d_fdb_table,omitempty"`
	Dot1QFdbTable     *FdbTableConfig    `yaml:"dot1q_fdb_table,omitempty"`
	Traps             *TrapsConfig       `yaml:"traps,omitempty"` // v1.6.0
}

// AddMib represents a MIB override or addition.
type AddMib struct {
	OID   string `yaml:"oid"`
	Type  string `yaml:"type"`
	Value string `yaml:"value"`
}

// CommunityInclude represents a community-specific walk include.
type CommunityInclude struct {
	Community string `yaml:"community"`
	WalkFile  string `yaml:"walk_file"`
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
	ClientLeases     []DhcpLease `yaml:"client_leases,omitempty"`
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
	ClientIP     string `yaml:"client_ip"`
	MacAddrValue string `yaml:"mac_addr_value,omitempty"`
	MacAddrMask  string `yaml:"mac_addr_mask,omitempty"`
}

// DNSServer represents DNS server configuration.
type DNSServer struct {
	ForwardRecords []DNSRecord `yaml:"forward_records,omitempty"`
	ReverseRecords []DNSRecord `yaml:"reverse_records,omitempty"`
}

// DNSRecord represents a DNS A or PTR record.
type DNSRecord struct {
	Name  string `yaml:"name"`
	IP    string `yaml:"ip"`
	TTL   int    `yaml:"ttl,omitempty"`
	RCode int    `yaml:"rcode,omitempty"`
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

// ConvertFile converts a Java DSL config file to YAML.
func ConvertFile(inputPath, outputPath string, verbose bool) error {
	cleanInput := filepath.Clean(inputPath)

	// Check file size before reading to prevent memory exhaustion
	info, err := os.Stat(cleanInput)
	if err != nil {
		return fmt.Errorf("error accessing input file: %w", err)
	}
	if info.Size() > maxInputFileSize {
		return fmt.Errorf("input file too large: %d bytes (max %d)", info.Size(), maxInputFileSize)
	}

	// Validate output path is not empty and is clean
	if outputPath == "" {
		return errors.New("output path cannot be empty")
	}
	cleanOutput := filepath.Clean(outputPath)

	// Read input file
	data, err := os.ReadFile(cleanInput)
	if err != nil {
		return fmt.Errorf("error reading input file: %w", err)
	}

	// Parse Java DSL
	parser := &Parser{
		lines:   strings.Split(string(data), "\n"),
		pos:     0,
		verbose: verbose,
	}

	config, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("error parsing config: %w", err)
	}

	// Convert to YAML
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshaling YAML: %w", err)
	}

	// Write output file
	if writeErr := os.WriteFile(cleanOutput, yamlData, 0o600); writeErr != nil {
		return fmt.Errorf("error writing output file: %w", writeErr)
	}

	return nil
}

// Parse parses the Java DSL format.
func (p *Parser) Parse() (*Config, error) {
	config := &Config{}
	deviceCount := 0

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			p.pos++

			continue
		}

		// Parse directives
		if strings.HasPrefix(line, "IncludePath(") {
			if path := p.extractString(line); path != "" {
				config.IncludePath = path
			}
			p.pos++

			continue
		}

		if strings.HasPrefix(line, "CapturePlayback(") {
			playback, err := p.parseCapturePlayback()
			if err != nil {
				return nil, err
			}
			config.CapturePlaybacks = append(config.CapturePlaybacks, *playback)

			continue
		}

		if strings.HasPrefix(line, "Device(") {
			device, err := p.parseDevice(deviceCount)
			if err != nil {
				return nil, err
			}
			config.Devices = append(config.Devices, *device)
			deviceCount++

			continue
		}

		p.pos++
	}

	return config, nil
}

// parseCapturePlayback parses a CapturePlayback block.
func (p *Parser) parseCapturePlayback() (*CapturePlayback, error) {
	p.pos++ // Skip opening line
	playback := &CapturePlayback{}

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])

		if line == ")" {
			p.pos++

			break
		}

		switch {
		case strings.HasPrefix(line, "FileName("):
			playback.FileName = p.extractString(line)
		case strings.HasPrefix(line, "LoopTime("):
			var loopTime int
			n, err := fmt.Sscanf(line, "LoopTime(%d)", &loopTime)
			if err != nil || n != 1 {
				return nil, fmt.Errorf("line %d: %w: %s", p.pos+1, ErrInvalidLoopTimeFormat, line)
			}
			playback.LoopTime = loopTime
		case strings.HasPrefix(line, "ScaleTime("):
			var scaleTime float64
			n, err := fmt.Sscanf(line, "ScaleTime(%f)", &scaleTime)
			if err != nil || n != 1 {
				return nil, fmt.Errorf("line %d: %w: %s", p.pos+1, ErrInvalidScaleTimeFormat, line)
			}
			playback.ScaleTime = scaleTime
		}

		p.pos++
	}

	return playback, nil
}

// parseDeviceSimpleField handles simple device fields that don't require nested parsing.
// Returns true if the field was handled.
func (p *Parser) parseDeviceSimpleField(line string, device *Device) bool {
	switch {
	case strings.HasPrefix(line, "MacAddr("):
		device.MAC = p.formatMAC(p.extractValue(line))
	case strings.HasPrefix(line, "IpAddr("):
		p.parseDeviceIPAddr(line, device)
	case strings.HasPrefix(line, "Ip6Addr("):
		if ip := p.extractValue(line); ip != "" {
			device.IPs = append(device.IPs, ip)
		}
	case strings.HasPrefix(line, "MapToIp("):
		device.MapToIP = p.extractValue(line)
	case strings.HasPrefix(line, "Babble("):
		device.Babble = true
	case strings.HasPrefix(line, "TTL("):
		if ttlCfg := p.parseTTL(line); ttlCfg != nil {
			device.TTL = ttlCfg
		}
	case strings.HasPrefix(line, "SpanningTree("):
		p.parseDeviceSpanningTree(device)
	default:
		return false
	}

	return true
}

// parseDeviceIPAddr handles IpAddr field parsing.
func (p *Parser) parseDeviceIPAddr(line string, device *Device) {
	ip := p.extractValue(line)
	if device.IP == "" {
		device.IP = ip
	} else if ip != "" {
		device.IPs = append(device.IPs, ip)
	}
}

// parseDeviceSpanningTree handles SpanningTree field parsing.
func (p *Parser) parseDeviceSpanningTree(device *Device) {
	if device.Stp == nil {
		device.Stp = &StpConfig{Enabled: true}
	} else {
		device.Stp.Enabled = true
	}
}

// parseDeviceVlan parses the VLAN field and returns an error if invalid.
func (p *Parser) parseDeviceVlan(line string) (int, error) {
	var vlan int
	n, err := fmt.Sscanf(line, "Vlan(%d)", &vlan)
	if err != nil || n != 1 {
		return 0, fmt.Errorf("line %d: %w: %s", p.pos+1, ErrInvalidVlanFormat, line)
	}

	return vlan, nil
}

// parseDeviceNestedBlock handles device fields that require nested block parsing.
// Returns true if parsing occurred and pos should not be incremented.
func (p *Parser) parseDeviceNestedBlock(line string, device *Device) bool {
	switch {
	case strings.HasPrefix(line, "SnmpAgent("):
		agent := p.parseSnmpAgent()
		device.SnmpAgent = mergeSnmpAgent(device.SnmpAgent, agent)

		return true
	case strings.HasPrefix(line, "SnmpAccessList("):
		accessList := p.parseSnmpAccessList()
		if device.SnmpAgent == nil {
			device.SnmpAgent = &SnmpAgent{}
		}
		device.SnmpAgent.AccessList = append(device.SnmpAgent.AccessList, accessList...)

		return true
	case strings.HasPrefix(line, "NetBiosStatus("):
		device.Netbios = mergeNetbios(device.Netbios, p.parseNetBiosStatus())

		return true
	case strings.HasPrefix(line, "Icmp("):
		device.Icmp = p.parseIcmp()

		return true
	case strings.HasPrefix(line, "Icmp6("):
		device.Icmpv6 = p.parseIcmp6()

		return true
	case strings.HasPrefix(line, "Dhcp("):
		device.Dhcp = p.parseDhcp()

		return true
	case strings.HasPrefix(line, "Dns("):
		device.DNS = p.parseDNS()

		return true
	}

	return false
}

// parseDevice parses a Device block.
func (p *Parser) parseDevice(deviceNum int) (*Device, error) {
	p.pos++ // Skip opening line
	device := &Device{
		Name: fmt.Sprintf("device%d", deviceNum+1),
	}

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])

		if line == ")" {
			p.pos++

			break
		}

		// Handle simple fields first
		if p.parseDeviceSimpleField(line, device) {
			p.pos++

			continue
		}

		// Handle VLAN separately due to error return
		if strings.HasPrefix(line, "Vlan(") {
			vlan, err := p.parseDeviceVlan(line)
			if err != nil {
				return nil, err
			}
			device.VLAN = vlan
			p.pos++

			continue
		}

		// Handle nested blocks
		if p.parseDeviceNestedBlock(line, device) {
			continue
		}

		p.pos++
	}

	return device, nil
}

// mergeSnmpAgent merges SNMP agent settings.
func mergeSnmpAgent(base *SnmpAgent, incoming *SnmpAgent) *SnmpAgent {
	if base == nil {
		return incoming
	}
	if incoming == nil {
		return base
	}
	if base.WalkFile == "" {
		base.WalkFile = incoming.WalkFile
	}
	base.WalkFiles = append(base.WalkFiles, incoming.WalkFiles...)
	base.AddMibs = append(base.AddMibs, incoming.AddMibs...)
	base.CommunityIncludes = append(base.CommunityIncludes, incoming.CommunityIncludes...)
	base.AccessList = append(base.AccessList, incoming.AccessList...)
	if base.SnmpAddr == "" {
		base.SnmpAddr = incoming.SnmpAddr
	}
	if base.Dot1DFdbTable == nil {
		base.Dot1DFdbTable = incoming.Dot1DFdbTable
	}
	if base.Dot1QFdbTable == nil {
		base.Dot1QFdbTable = incoming.Dot1QFdbTable
	}
	if base.Traps == nil {
		base.Traps = incoming.Traps
	}

	return base
}

// mergeNetbios merges NetBIOS configuration.
func mergeNetbios(base *NetbiosConfig, incoming *NetbiosConfig) *NetbiosConfig {
	if base == nil {
		return incoming
	}
	if incoming == nil {
		return base
	}
	if base.Name == "" {
		base.Name = incoming.Name
	}
	if base.Workgroup == "" {
		base.Workgroup = incoming.Workgroup
	}
	if base.NodeType == "" {
		base.NodeType = incoming.NodeType
	}
	if len(base.Services) == 0 {
		base.Services = incoming.Services
	}
	if base.TTL == 0 {
		base.TTL = incoming.TTL
	}
	base.Names = append(base.Names, incoming.Names...)
	base.MsBrowse = base.MsBrowse || incoming.MsBrowse
	base.Enabled = base.Enabled || incoming.Enabled

	return base
}

// parseSnmpAgentInclude handles Include directive for SNMP agent.
func (p *Parser) parseSnmpAgentInclude(line string, agent *SnmpAgent) {
	walk := p.extractString(line)
	if walk == "" {
		return
	}

	if agent.WalkFile == "" {
		agent.WalkFile = walk
	}
	agent.WalkFiles = append(agent.WalkFiles, walk)
}

// parseSnmpAgentCommunityInclude handles CommunityInclude directive.
func (p *Parser) parseSnmpAgentCommunityInclude(line string, agent *SnmpAgent) {
	community, walk := p.parseCommunityInclude(line)
	if community == "" || walk == "" {
		return
	}
	agent.CommunityIncludes = append(agent.CommunityIncludes, CommunityInclude{
		Community: community,
		WalkFile:  walk,
	})
}

// parseSnmpAgentField parses a single SNMP agent field.
func (p *Parser) parseSnmpAgentField(line string, agent *SnmpAgent) {
	switch {
	case strings.HasPrefix(line, "Include("):
		p.parseSnmpAgentInclude(line, agent)
	case strings.HasPrefix(line, "CommunityInclude("):
		p.parseSnmpAgentCommunityInclude(line, agent)
	case strings.HasPrefix(line, "SnmpAddr("):
		agent.SnmpAddr = p.extractValue(line)
	case strings.HasPrefix(line, "Dot1D_FdbTable("):
		if cfg := p.parseFdbTable(line, true); cfg != nil {
			agent.Dot1DFdbTable = cfg
		}
	case strings.HasPrefix(line, "Dot1Q_FdbTable("):
		if cfg := p.parseFdbTable(line, false); cfg != nil {
			agent.Dot1QFdbTable = cfg
		}
	case strings.HasPrefix(line, "AddMib("):
		mibLine := p.collectQuotedDirective(line, addMibQuotedArgs)
		if mib := p.parseAddMib(mibLine); mib != nil {
			agent.AddMibs = append(agent.AddMibs, *mib)
		}
	}
}

// parseSnmpAgent parses an SnmpAgent block.
func (p *Parser) parseSnmpAgent() *SnmpAgent {
	p.pos++ // Skip opening line
	agent := &SnmpAgent{}

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])

		if line == ")" {
			p.pos++

			break
		}

		// Skip comments
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			p.pos++

			continue
		}

		p.parseSnmpAgentField(line, agent)
		p.pos++
	}

	return agent
}

// parseSnmpAccessList parses SnmpAccessList block.
func (p *Parser) parseSnmpAccessList() []string {
	p.pos++ // Skip opening line
	var accessList []string

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])
		if line == ")" {
			p.pos++

			break
		}
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			p.pos++

			continue
		}
		if strings.HasPrefix(line, "IpAddr(") {
			ip := p.extractValue(line)
			if ip != "" {
				accessList = append(accessList, ip)
			}
		}
		p.pos++
	}

	return accessList
}

// parseTTL parses TTL(ttl ip mask).
func (p *Parser) parseTTL(line string) *TTLConfig {
	re := regexp.MustCompile(`\(([^)]*)\)`)
	match := re.FindStringSubmatch(line)
	if len(match) < minRegexMatchParts {
		return nil
	}
	parts := strings.Fields(match[1])
	if len(parts) < ttlFieldCount {
		return nil
	}
	var ttl int
	if _, err := fmt.Sscanf(parts[0], "%d", &ttl); err != nil {
		return nil
	}

	return &TTLConfig{
		TTL:  ttl,
		IP:   parts[1],
		Mask: parts[2],
	}
}

// parseNetBiosStatus parses NetBiosStatus block.
func (p *Parser) parseNetBiosStatus() *NetbiosConfig {
	p.pos++ // Skip opening line
	netbios := &NetbiosConfig{
		Enabled: true,
	}

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])
		if line == ")" {
			p.pos++

			break
		}

		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			p.pos++

			continue
		}

		switch {
		case strings.HasPrefix(line, "MsBrowse"):
			netbios.MsBrowse = true
		case strings.HasPrefix(line, "Bnode"):
			netbios.NodeType = "B"
		case strings.HasPrefix(line, "Pnode"):
			netbios.NodeType = "P"
		case strings.HasPrefix(line, "Mnode"):
			netbios.NodeType = "M"
		case strings.HasPrefix(line, "NetBiosName("):
			name := p.parseNetBiosName(line)
			if name != nil {
				netbios.Names = append(netbios.Names, *name)
			}
		}

		p.pos++
	}

	return netbios
}

// parseNetBiosName parses NetBiosName("name" suffix groupflag).
func (p *Parser) parseNetBiosName(line string) *NetbiosName {
	// Extract quoted name
	re := regexp.MustCompile(`"([^"]+)"`)
	matches := re.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return nil
	}
	name := matches[0][1]

	// Remove quoted name to parse remaining tokens
	remaining := re.ReplaceAllString(line, "")
	remaining = strings.ReplaceAll(remaining, "NetBiosName", "")
	remaining = strings.ReplaceAll(remaining, "(", "")
	remaining = strings.ReplaceAll(remaining, ")", "")
	tokens := strings.Fields(remaining)

	entry := &NetbiosName{Name: name}

	for _, token := range tokens {
		switch token {
		case "Machine":
			entry.Suffix = "0"
		case "MsBrowse":
			entry.Suffix = "1"
		case "MsgServ":
			entry.Suffix = "3"
		case "MbSubnet":
			entry.Suffix = "29"
		case "MbElect":
			entry.Suffix = "30"
		case "LanMan":
			entry.Suffix = "32"
		case "Group":
			entry.Group = true
		case "Unique":
			entry.Group = false
		default:
			if _, err := strconv.Atoi(token); err == nil {
				entry.Suffix = token
			}
		}
	}

	return entry
}

// parseIcmp parses Icmp block.
func (p *Parser) parseIcmp() *IcmpConfig {
	p.pos++ // Skip opening line
	icmp := &IcmpConfig{Enabled: true}

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])
		if line == ")" {
			p.pos++

			break
		}
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			p.pos++

			continue
		}

		if strings.HasPrefix(line, "AddressMaskReply(") {
			icmp.AddressMaskReply = p.extractValue(line)
		} else if strings.HasPrefix(line, "RouterAdvertisement(") {
			icmp.RouterAdvertisement = p.parseIcmpRouterAdvertisement()

			continue
		}

		p.pos++
	}

	return icmp
}

// parseIcmpRouter parses a single Router directive and returns an IcmpRouter.
func (p *Parser) parseIcmpRouter(line string) *IcmpRouter {
	re := regexp.MustCompile(`\(([^)]*)\)`)
	match := re.FindStringSubmatch(line)
	if len(match) < minRegexMatchParts {
		return nil
	}

	parts := strings.Fields(match[1])
	if len(parts) < routerFieldCount {
		return nil
	}

	var pref int
	_, _ = fmt.Sscanf(parts[1], "%d", &pref)

	return &IcmpRouter{
		Address:    parts[0],
		Preference: pref,
	}
}

// parseIcmpRAField parses a single field in the RouterAdvertisement block.
func (p *Parser) parseIcmpRAField(line string, ra *IcmpRouterAdvertisement) {
	switch {
	case strings.HasPrefix(line, "Period("):
		var period int
		if _, err := fmt.Sscanf(line, "Period(%d)", &period); err == nil {
			ra.Period = period
		}
	case strings.HasPrefix(line, "Lifetime("):
		var lifetime int
		if _, err := fmt.Sscanf(line, "Lifetime(%d)", &lifetime); err == nil {
			ra.Lifetime = lifetime
		}
	case strings.HasPrefix(line, "Router("):
		if router := p.parseIcmpRouter(line); router != nil {
			ra.Routers = append(ra.Routers, *router)
		}
	}
}

// parseIcmpRouterAdvertisement parses RouterAdvertisement block for IPv4.
func (p *Parser) parseIcmpRouterAdvertisement() *IcmpRouterAdvertisement {
	p.pos++ // Skip opening line
	ra := &IcmpRouterAdvertisement{}

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])
		if line == ")" {
			p.pos++

			break
		}
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			p.pos++

			continue
		}

		p.parseIcmpRAField(line, ra)
		p.pos++
	}

	return ra
}

// parseIcmp6 parses Icmp6 block.
func (p *Parser) parseIcmp6() *Icmpv6Config {
	p.pos++ // Skip opening line
	icmp6 := &Icmpv6Config{Enabled: true}

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])
		if line == ")" {
			p.pos++

			break
		}
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			p.pos++

			continue
		}

		if strings.HasPrefix(line, "RouterAdvertisement(") {
			ra := p.parseIcmpv6RouterAdvertisement()
			icmp6.RouterAdvertisement = ra

			continue
		}

		p.pos++
	}

	return icmp6
}

// parseIcmpv6RouterAdvertisement parses IPv6 RouterAdvertisement block.
func (p *Parser) parseIcmpv6RouterAdvertisement() *Icmpv6RouterAdvertisement {
	p.pos++ // Skip opening line
	ra := &Icmpv6RouterAdvertisement{}

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])
		if line == ")" {
			p.pos++

			break
		}

		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			p.pos++

			continue
		}

		if p.parseRAField(line, ra) {
			continue
		}

		p.pos++
	}

	return ra
}

// parseRAField parses a single router advertisement field line.
// Returns true if the field was a PrefixInformation (already advanced pos).
func (p *Parser) parseRAField(line string, ra *Icmpv6RouterAdvertisement) bool {
	fieldParsers := []struct {
		prefix string
		format string
		target *int
	}{
		{"Period(", "Period(%d)", &ra.Period},
		{"CurHopLimit(", "CurHopLimit(%d)", &ra.CurHopLimit},
		{"Managed(", "Managed(%d)", &ra.Managed},
		{"Other(", "Other(%d)", &ra.Other},
		{"Lifetime(", "Lifetime(%d)", &ra.Lifetime},
		{"ReachableTime(", "ReachableTime(%d)", &ra.ReachableTime},
		{"RetransTimer(", "RetransTimer(%d)", &ra.RetransTimer},
		{"MTU(", "MTU(%d)", &ra.MTU},
	}

	for _, fp := range fieldParsers {
		if strings.HasPrefix(line, fp.prefix) {
			_, _ = fmt.Sscanf(line, fp.format, fp.target)

			return false
		}
	}

	if strings.HasPrefix(line, "PrefixInformation(") {
		prefix := p.parseIcmpv6PrefixInformation()
		if prefix != nil {
			ra.PrefixInfo = append(ra.PrefixInfo, *prefix)
		}

		return true
	}

	return false
}

// parseIcmpv6PrefixInformation parses PrefixInformation block.
func (p *Parser) parseIcmpv6PrefixInformation() *Icmpv6PrefixInfo {
	p.pos++ // Skip opening line
	prefix := &Icmpv6PrefixInfo{}

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])
		if line == ")" {
			p.pos++

			break
		}
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			p.pos++

			continue
		}
		switch {
		case strings.HasPrefix(line, "PrefixLength("):
			var v int
			_, _ = fmt.Sscanf(line, "PrefixLength(%d)", &v)
			prefix.PrefixLength = v
		case strings.HasPrefix(line, "Onlink("):
			var v int
			_, _ = fmt.Sscanf(line, "Onlink(%d)", &v)
			prefix.Onlink = v
		case strings.HasPrefix(line, "Auto("):
			var v int
			_, _ = fmt.Sscanf(line, "Auto(%d)", &v)
			prefix.Auto = v
		case strings.HasPrefix(line, "ValidLifetime("):
			var v int
			_, _ = fmt.Sscanf(line, "ValidLifetime(%d)", &v)
			prefix.ValidLifetime = v
		case strings.HasPrefix(line, "PreferredLifetime("):
			var v int
			_, _ = fmt.Sscanf(line, "PreferredLifetime(%d)", &v)
			prefix.PreferredLifetime = v
		case strings.HasPrefix(line, "Prefix("):
			prefix.Prefix = p.extractValue(line)
		}
		p.pos++
	}

	return prefix
}

// collectQuotedDirective joins lines until at least minQuotes quoted strings are present.
func (p *Parser) collectQuotedDirective(line string, minQuotes int) string {
	combined := stripInlineComment(line)
	re := regexp.MustCompile(`"([^"]+)"`)
	for p.pos+1 < len(p.lines) && len(re.FindAllStringSubmatch(combined, -1)) < minQuotes {
		p.pos++
		next := stripInlineComment(strings.TrimSpace(p.lines[p.pos]))
		if next == "" {
			continue
		}
		combined += " " + next
	}

	return combined
}

func stripInlineComment(line string) string {
	inQuote := false
	for i := range len(line) - 1 {
		if line[i] == '"' {
			inQuote = !inQuote

			continue
		}
		if !inQuote && line[i] == '/' && line[i+1] == '/' {
			return strings.TrimSpace(line[:i])
		}
		if !inQuote && line[i] == '#' {
			return strings.TrimSpace(line[:i])
		}
	}

	return strings.TrimSpace(line)
}

// parseAddMib parses an AddMib directive.
func (p *Parser) parseAddMib(line string) *AddMib {
	// Extract quoted strings (OID, type, value)
	re := regexp.MustCompile(`"([^"]+)"`)
	matches := re.FindAllStringSubmatch(line, -1)
	if len(matches) < addMibFieldCount {
		return nil
	}

	return &AddMib{
		OID:   matches[0][1],
		Type:  matches[1][1],
		Value: matches[2][1],
	}
}

// parseCommunityInclude parses CommunityInclude("community" "walkfile").
func (p *Parser) parseCommunityInclude(line string) (string, string) {
	re := regexp.MustCompile(`"([^"]+)"`)
	matches := re.FindAllStringSubmatch(line, -1)
	if len(matches) < communityIncludeFields {
		return "", ""
	}

	return matches[0][1], matches[1][1]
}

// parseFdbTable parses Dot1D_FdbTable or Dot1Q_FdbTable directives.
func (p *Parser) parseFdbTable(line string, dot1d bool) *FdbTableConfig {
	re := regexp.MustCompile(`\(([^)]*)\)`)
	match := re.FindStringSubmatch(line)
	if len(match) < minRegexMatchParts {
		return nil
	}
	parts := strings.Fields(match[1])
	if len(parts) == 0 {
		return nil
	}
	cfg := &FdbTableConfig{}
	if _, err := fmt.Sscanf(parts[0], "%d", &cfg.Port); err != nil {
		return nil
	}
	if len(parts) > 1 {
		_, _ = fmt.Sscanf(parts[1], "%d", &cfg.VLAN)
	} else if !dot1d {
		return nil
	}

	return cfg
}

// dhcpParseState holds state during DHCP parsing.
type dhcpParseState struct {
	dhcp         *DhcpServer
	currentLease *DhcpLease
}

// parseDhcpClientIP parses YourClientIpAddr and starts a new lease.
func (p *Parser) parseDhcpClientIP(line string) *DhcpLease {
	ip := p.extractValue(line)
	if ip == "" {
		// No closing paren on same line, extract everything after (
		if _, after, found := strings.Cut(line, "("); found {
			ip = strings.TrimSpace(after)
		}
	}

	return &DhcpLease{ClientIP: ip}
}

// parseDhcpMacAddrMask handles MacAddrMask and finalizes the current lease.
// Returns true if the caller should continue (skip pos increment).
func (p *Parser) parseDhcpMacAddrMask(line string, state *dhcpParseState) bool {
	if state.currentLease == nil {
		return false
	}

	state.currentLease.MacAddrMask = p.formatMAC(p.extractValue(line))
	state.dhcp.ClientLeases = append(state.dhcp.ClientLeases, *state.currentLease)
	state.currentLease = nil

	// Skip the closing paren of YourClientIpAddr block on next line
	p.pos++
	if p.pos < len(p.lines) && strings.TrimSpace(p.lines[p.pos]) == ")" {
		p.pos++ // Skip the closing paren
	}

	return true
}

// parseDhcpServerField parses server-level DHCP fields (not lease-specific).
func (p *Parser) parseDhcpServerField(line string, dhcp *DhcpServer) {
	switch {
	case strings.HasPrefix(line, "SubnetMask"):
		dhcp.SubnetMask = p.extractValue(line)
	case strings.HasPrefix(line, "Router"):
		if value := p.extractValue(line); value != "" {
			dhcp.Router = strings.Fields(value)[0]
		}
	case strings.HasPrefix(line, "DomainNameServer"):
		dhcp.DomainNameServer = p.extractValue(line)
	case strings.HasPrefix(line, "NextServerIpAddr"):
		dhcp.NextServerIP = p.extractValue(line)
	case strings.HasPrefix(line, "ServerIdentifier"):
		dhcp.ServerIdentifier = p.extractValue(line)
	}
}

// parseDhcp parses a Dhcp block.
func (p *Parser) parseDhcp() *DhcpServer {
	p.pos++ // Skip opening line
	state := &dhcpParseState{
		dhcp: &DhcpServer{
			ClientLeases: make([]DhcpLease, 0),
		},
	}

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])

		if line == ")" {
			// Save current lease if exists
			if state.currentLease != nil {
				state.dhcp.ClientLeases = append(state.dhcp.ClientLeases, *state.currentLease)
			}
			p.pos++

			break
		}

		// Skip comments
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			p.pos++

			continue
		}

		// Handle lease-specific fields
		switch {
		case strings.HasPrefix(line, "YourClientIpAddr"):
			state.currentLease = p.parseDhcpClientIP(line)
		case strings.HasPrefix(line, "MacAddrValue"):
			if state.currentLease != nil {
				state.currentLease.MacAddrValue = p.formatMAC(p.extractValue(line))
			}
		case strings.HasPrefix(line, "MacAddrMask"):
			if p.parseDhcpMacAddrMask(line, state) {
				continue
			}
		default:
			// Handle server-level fields
			p.parseDhcpServerField(line, state.dhcp)
		}

		p.pos++
	}

	return state.dhcp
}

// parseDNS parses a DNS block.
func (p *Parser) parseDNS() *DNSServer {
	p.pos++ // Skip opening line
	dns := &DNSServer{
		ForwardRecords: make([]DNSRecord, 0),
		ReverseRecords: make([]DNSRecord, 0),
	}

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])

		if line == ")" {
			p.pos++

			break
		}

		// Skip comments
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			p.pos++

			continue
		}

		switch {
		case strings.HasPrefix(line, "Forward2("), strings.HasPrefix(line, "Forward("):
			record := p.parseDNSRecord(line, true)
			if record != nil {
				dns.ForwardRecords = append(dns.ForwardRecords, *record)
			}
		case strings.HasPrefix(line, "Reverse2("), strings.HasPrefix(line, "Reverse("):
			record := p.parseDNSRecord(line, false)
			if record != nil {
				dns.ReverseRecords = append(dns.ReverseRecords, *record)
			}
		}

		p.pos++
	}

	return dns
}

// parseDNSRecordTTLAndRCode parses optional TTL and RCode fields from DNS record parts.
func (p *Parser) parseDNSRecordTTLAndRCode(parts []string, record *DNSRecord) {
	if len(parts) >= dnsPartsWithTTL {
		_, _ = fmt.Sscanf(parts[2], "%d", &record.TTL)
	}

	if len(parts) >= dnsPartsWithRCode {
		_, _ = fmt.Sscanf(parts[3], "%d", &record.RCode)
	}
}

// parseDNSRecord parses a Forward() or Reverse() DNS record.
func (p *Parser) parseDNSRecord(line string, isForward bool) *DNSRecord {
	// Forward("hostname" IP TTL)
	// Forward2("hostname" IP TTL RCODE)
	// Reverse(IP "hostname" TTL)
	// Reverse2(IP "hostname" TTL RCODE)
	re := regexp.MustCompile(`\((.*?)\)`)
	match := re.FindStringSubmatch(line)
	if len(match) < minRegexMatchParts {
		return nil
	}

	parts := strings.Fields(match[1])
	if len(parts) < routerFieldCount {
		return nil
	}

	record := &DNSRecord{}

	if isForward {
		// Forward("hostname" IP TTL)
		record.Name = strings.Trim(parts[0], "\"")
		record.IP = parts[1]
	} else {
		// Reverse(IP "hostname" TTL)
		record.IP = parts[0]
		record.Name = strings.Trim(parts[1], "\"")
	}

	p.parseDNSRecordTTLAndRCode(parts, record)

	return record
}

// extractString extracts a quoted string from a directive.
func (p *Parser) extractString(line string) string {
	start := strings.Index(line, "\"")
	if start == -1 {
		return ""
	}
	end := strings.Index(line[start+1:], "\"")
	if end == -1 {
		return ""
	}

	return line[start+1 : start+1+end]
}

// extractValue extracts a value from parentheses (no quotes).
func (p *Parser) extractValue(line string) string {
	start := strings.Index(line, "(")
	if start == -1 {
		return ""
	}
	end := strings.Index(line[start+1:], ")")
	if end == -1 {
		return ""
	}

	return line[start+1 : start+1+end]
}

// formatMAC converts XXXXXXXXXXXX to XX:XX:XX:XX:XX:XX.
func (p *Parser) formatMAC(mac string) string {
	if len(mac) != macAddressRawLen {
		return mac
	}

	return fmt.Sprintf("%s:%s:%s:%s:%s:%s",
		mac[0:2], mac[2:4], mac[4:6], mac[6:8], mac[8:10], mac[10:12])
}

// LoadYAMLConfig loads a YAML config file into Go config structure.
func LoadYAMLConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		return nil, fmt.Errorf("error reading YAML file: %w", err)
	}

	return LoadYAMLConfigFromBytes(data)
}

// LoadYAMLConfigFromBytes converts in-memory YAML data into a Go config structure.
func LoadYAMLConfigFromBytes(data []byte) (*Config, error) {
	var config Config
	err := yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing YAML: %w", err)
	}

	return &config, nil
}

// ValidateConfig validates that no functionality was lost in conversion.
func ValidateConfig(config *Config) error {
	// Validate devices have required fields
	for i, device := range config.Devices {
		if device.MAC == "" {
			return fmt.Errorf("%w: device %d", ErrDeviceMissingMAC, i)
		}
		// IP address is optional in Java configs (some devices don't have IPs)

		// If SNMP agent specified, validate it (empty SNMP agents are allowed)
		if device.SnmpAgent != nil && len(device.SnmpAgent.AddMibs) > 0 {
			// Validate AddMibs have required fields
			for j, mib := range device.SnmpAgent.AddMibs {
				if mib.OID == "" {
					return fmt.Errorf("%w: device %d AddMib %d", ErrAddMibMissingOID, i, j)
				}
				if mib.Type == "" {
					return fmt.Errorf("%w: device %d AddMib %d", ErrAddMibMissingType, i, j)
				}
			}
		}
	}

	// If capture playbacks specified, validate them
	for i, playback := range config.CapturePlaybacks {
		if playback.FileName == "" {
			return fmt.Errorf("%w: index %d", ErrCapturePlaybackMissingFile, i)
		}
	}

	return nil
}

// PrintSummary prints a summary of the config.
func PrintSummary(config *Config, w *bufio.Writer) {
	_, _ = fmt.Fprintf(w, "Configuration Summary:\n")
	_, _ = fmt.Fprintf(w, "  Devices: %d\n", len(config.Devices))

	if config.IncludePath != "" {
		_, _ = fmt.Fprintf(w, "  Include Path: %s\n", config.IncludePath)
	}

	if len(config.CapturePlaybacks) > 0 {
		_, _ = fmt.Fprintf(w, "  PCAP Playbacks: %d\n", len(config.CapturePlaybacks))
		for i, playback := range config.CapturePlaybacks {
			_, _ = fmt.Fprintf(w, "    [%d] %s\n", i+1, playback.FileName)
			if playback.LoopTime > 0 {
				_, _ = fmt.Fprintf(w, "        Loop Time: %d ms\n", playback.LoopTime)
			}
			if playback.ScaleTime > 0 {
				_, _ = fmt.Fprintf(w, "        Scale Time: %.2f\n", playback.ScaleTime)
			}
		}
	}

	snmpCount := 0
	mibCount := 0
	for _, device := range config.Devices {
		if device.SnmpAgent != nil {
			snmpCount++
			mibCount += len(device.SnmpAgent.AddMibs)
		}
	}

	if snmpCount > 0 {
		_, _ = fmt.Fprintf(w, "  SNMP Agents: %d\n", snmpCount)
		if mibCount > 0 {
			_, _ = fmt.Fprintf(w, "  Custom MIBs: %d\n", mibCount)
		}
	}

	_ = w.Flush()
}
