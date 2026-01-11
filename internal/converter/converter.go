// Package converter implements Java NIAC DSL to YAML conversion
package converter

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the YAML configuration structure
type Config struct {
	IncludePath        string              `yaml:"include_path,omitempty"`
	CapturePlaybacks   []CapturePlayback   `yaml:"capture_playbacks,omitempty"` // Changed to array
	DiscoveryProtocols *DiscoveryProtocols `yaml:"discovery_protocols,omitempty"`
	Devices            []Device            `yaml:"devices"`
}

// DiscoveryProtocols configures discovery protocol behavior
type DiscoveryProtocols struct {
	LLDP *ProtocolConfig `yaml:"lldp,omitempty"`
	CDP  *ProtocolConfig `yaml:"cdp,omitempty"`
	EDP  *ProtocolConfig `yaml:"edp,omitempty"`
	FDP  *ProtocolConfig `yaml:"fdp,omitempty"`
}

// ProtocolConfig configures a discovery protocol
type ProtocolConfig struct {
	Enabled  bool `yaml:"enabled"`
	Interval int  `yaml:"interval,omitempty"` // Advertisement interval in seconds
}

// CapturePlayback represents PCAP playback configuration
type CapturePlayback struct {
	FileName  string  `yaml:"file_name"`
	LoopTime  int     `yaml:"loop_time,omitempty"`
	ScaleTime float64 `yaml:"scale_time,omitempty"`
}

// Device represents a network device
type Device struct {
	Name      string         `yaml:"name,omitempty"`
	MAC       string         `yaml:"mac"`
	IP        string         `yaml:"ip,omitempty"`  // Single IP (backward compatible)
	IPs       []string       `yaml:"ips,omitempty"` // Multiple IPs (new feature)
	VLAN      int            `yaml:"vlan,omitempty"`
	MapToIP   string         `yaml:"map_to_ip,omitempty"`
	Babble    bool           `yaml:"babble,omitempty"`
	TTL       *TTLConfig     `yaml:"ttl,omitempty"`
	SnmpAgent *SnmpAgent     `yaml:"snmp_agent,omitempty"`
	Dhcp      *DhcpServer    `yaml:"dhcp,omitempty"`
	Dns       *DnsServer     `yaml:"dns,omitempty"`
	Lldp      *LldpConfig    `yaml:"lldp,omitempty"`
	Cdp       *CdpConfig     `yaml:"cdp,omitempty"`
	Edp       *EdpConfig     `yaml:"edp,omitempty"`
	Fdp       *FdpConfig     `yaml:"fdp,omitempty"`
	Stp       *StpConfig     `yaml:"stp,omitempty"`
	Http      *HttpConfig    `yaml:"http,omitempty"`
	Ftp       *FtpConfig     `yaml:"ftp,omitempty"`
	Netbios   *NetbiosConfig `yaml:"netbios,omitempty"`
	Icmp      *IcmpConfig    `yaml:"icmp,omitempty"`
	Icmpv6    *Icmpv6Config  `yaml:"icmpv6,omitempty"`
	Dhcpv6    *Dhcpv6Config  `yaml:"dhcpv6,omitempty"`
	Traffic       *TrafficConfig       `yaml:"traffic,omitempty"`        // v1.6.0
	OSFingerprint *OSFingerprintConfig `yaml:"os_fingerprint,omitempty"` // v1.24.0
	IPerf3        *IPerf3Config        `yaml:"iperf3,omitempty"`         // v1.25.0
}

// IPerf3Config represents iPerf3 server emulation configuration
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

// OSFingerprintConfig represents OS fingerprinting configuration for device simulation
type OSFingerprintConfig struct {
	OSType       string `yaml:"os_type,omitempty"`        // e.g., "linux", "windows", "cisco-ios", "juniper-junos"
	TTL          uint8  `yaml:"ttl,omitempty"`            // Default IP TTL (Linux=64, Windows=128, Cisco=255)
	WindowSize   uint16 `yaml:"window_size,omitempty"`    // TCP window size
	WindowScale  uint8  `yaml:"window_scale,omitempty"`   // TCP window scale option
	MSS          uint16 `yaml:"mss,omitempty"`            // TCP maximum segment size
	SSHBanner    string `yaml:"ssh_banner,omitempty"`     // SSH version banner
	HTTPServer   string `yaml:"http_server,omitempty"`    // HTTP Server header
	FTPBanner    string `yaml:"ftp_banner,omitempty"`     // FTP welcome banner
	SMTPBanner   string `yaml:"smtp_banner,omitempty"`    // SMTP banner
	TelnetBanner string `yaml:"telnet_banner,omitempty"`  // Telnet banner
	DontFragment bool   `yaml:"dont_fragment,omitempty"`  // IP DF bit (Linux=true, Windows=false usually)
}

// SnmpAgent represents SNMP agent configuration
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

// AddMib represents a MIB override or addition
type AddMib struct {
	OID   string `yaml:"oid"`
	Type  string `yaml:"type"`
	Value string `yaml:"value"`
}

// CommunityInclude represents a community-specific walk include
type CommunityInclude struct {
	Community string `yaml:"community"`
	WalkFile  string `yaml:"walk_file"`
}

// FdbTableConfig configures SNMP forwarding database table injection
type FdbTableConfig struct {
	Port int `yaml:"port,omitempty"`
	VLAN int `yaml:"vlan,omitempty"`
}

// TTLConfig configures ICMP TTL timeout behavior (traceroute simulation)
type TTLConfig struct {
	TTL  int    `yaml:"ttl,omitempty"`
	IP   string `yaml:"ip,omitempty"`
	Mask string `yaml:"mask,omitempty"`
}

// DhcpServer represents DHCP server configuration
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

// DhcpLease represents a DHCP client lease
type DhcpLease struct {
	ClientIP     string `yaml:"client_ip"`
	MacAddrValue string `yaml:"mac_addr_value,omitempty"`
	MacAddrMask  string `yaml:"mac_addr_mask,omitempty"`
}

// DnsServer represents DNS server configuration
type DnsServer struct {
	ForwardRecords []DnsRecord `yaml:"forward_records,omitempty"`
	ReverseRecords []DnsRecord `yaml:"reverse_records,omitempty"`
}

// DnsRecord represents a DNS A or PTR record
type DnsRecord struct {
	Name  string `yaml:"name"`
	IP    string `yaml:"ip"`
	TTL   int    `yaml:"ttl,omitempty"`
	RCode int    `yaml:"rcode,omitempty"`
}

// LldpConfig represents LLDP discovery protocol configuration
type LldpConfig struct {
	Enabled           bool   `yaml:"enabled,omitempty"`
	AdvertiseInterval int    `yaml:"advertise_interval,omitempty"`
	TTL               int    `yaml:"ttl,omitempty"`
	SystemDescription string `yaml:"system_description,omitempty"`
	PortDescription   string `yaml:"port_description,omitempty"`
	ChassisIDType     string `yaml:"chassis_id_type,omitempty"`
}

// CdpConfig represents CDP discovery protocol configuration
type CdpConfig struct {
	Enabled           bool   `yaml:"enabled,omitempty"`
	AdvertiseInterval int    `yaml:"advertise_interval,omitempty"`
	Holdtime          int    `yaml:"holdtime,omitempty"`
	Version           int    `yaml:"version,omitempty"`
	SoftwareVersion   string `yaml:"software_version,omitempty"`
	Platform          string `yaml:"platform,omitempty"`
	PortID            string `yaml:"port_id,omitempty"`
}

// EdpConfig represents EDP discovery protocol configuration
type EdpConfig struct {
	Enabled           bool   `yaml:"enabled,omitempty"`
	AdvertiseInterval int    `yaml:"advertise_interval,omitempty"`
	VersionString     string `yaml:"version_string,omitempty"`
	DisplayString     string `yaml:"display_string,omitempty"`
}

// FdpConfig represents FDP discovery protocol configuration
type FdpConfig struct {
	Enabled           bool   `yaml:"enabled,omitempty"`
	AdvertiseInterval int    `yaml:"advertise_interval,omitempty"`
	Holdtime          int    `yaml:"holdtime,omitempty"`
	SoftwareVersion   string `yaml:"software_version,omitempty"`
	Platform          string `yaml:"platform,omitempty"`
	PortID            string `yaml:"port_id,omitempty"`
}

// StpConfig represents STP/RSTP/MSTP configuration
type StpConfig struct {
	Enabled        bool   `yaml:"enabled,omitempty"`
	BridgePriority uint16 `yaml:"bridge_priority,omitempty"`
	HelloTime      uint16 `yaml:"hello_time,omitempty"`
	MaxAge         uint16 `yaml:"max_age,omitempty"`
	ForwardDelay   uint16 `yaml:"forward_delay,omitempty"`
	Version        string `yaml:"version,omitempty"`
}

// HttpConfig represents HTTP server configuration
type HttpConfig struct {
	Enabled    bool           `yaml:"enabled,omitempty"`
	ServerName string         `yaml:"server_name,omitempty"`
	Endpoints  []HttpEndpoint `yaml:"endpoints,omitempty"`
}

// HttpEndpoint represents an HTTP endpoint configuration
type HttpEndpoint struct {
	Path        string `yaml:"path,omitempty"`
	Method      string `yaml:"method,omitempty"`
	StatusCode  int    `yaml:"status_code,omitempty"`
	ContentType string `yaml:"content_type,omitempty"`
	Body        string `yaml:"body,omitempty"`
}

// FtpConfig represents FTP server configuration
type FtpConfig struct {
	Enabled        bool      `yaml:"enabled,omitempty"`
	WelcomeBanner  string    `yaml:"welcome_banner,omitempty"`
	SystemType     string    `yaml:"system_type,omitempty"`
	AllowAnonymous bool      `yaml:"allow_anonymous,omitempty"`
	Users          []FtpUser `yaml:"users,omitempty"`
}

// FtpUser represents an FTP user account
type FtpUser struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	HomeDir  string `yaml:"home_dir,omitempty"`
}

// NetbiosConfig represents NetBIOS service configuration
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

// NetbiosName represents a NetBIOS name entry
type NetbiosName struct {
	Name   string `yaml:"name,omitempty"`
	Suffix string `yaml:"suffix,omitempty"`
	Group  bool   `yaml:"group,omitempty"`
}

// IcmpConfig represents ICMP/ICMPv4 configuration
type IcmpConfig struct {
	Enabled             bool                     `yaml:"enabled,omitempty"`
	TTL                 uint8                    `yaml:"ttl,omitempty"`
	RateLimit           int                      `yaml:"rate_limit,omitempty"`
	AddressMaskReply    string                   `yaml:"address_mask_reply,omitempty"`
	RouterAdvertisement *IcmpRouterAdvertisement `yaml:"router_advertisement,omitempty"`
}

// IcmpRouterAdvertisement configures IPv4 router advertisements
type IcmpRouterAdvertisement struct {
	Period   int          `yaml:"period,omitempty"`
	Lifetime int          `yaml:"lifetime,omitempty"`
	Routers  []IcmpRouter `yaml:"routers,omitempty"`
}

// IcmpRouter represents an advertised router entry
type IcmpRouter struct {
	Address    string `yaml:"address,omitempty"`
	Preference int    `yaml:"preference,omitempty"`
}

// Icmpv6Config represents ICMPv6 configuration
type Icmpv6Config struct {
	Enabled             bool                       `yaml:"enabled,omitempty"`
	HopLimit            uint8                      `yaml:"hop_limit,omitempty"`
	RateLimit           int                        `yaml:"rate_limit,omitempty"`
	RouterAdvertisement *Icmpv6RouterAdvertisement `yaml:"router_advertisement,omitempty"`
}

// Icmpv6RouterAdvertisement configures IPv6 router advertisements
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

// Icmpv6PrefixInfo represents IPv6 prefix info options
type Icmpv6PrefixInfo struct {
	PrefixLength      int    `yaml:"prefix_length,omitempty"`
	Onlink            int    `yaml:"onlink,omitempty"`
	Auto              int    `yaml:"auto,omitempty"`
	ValidLifetime     int    `yaml:"valid_lifetime,omitempty"`
	PreferredLifetime int    `yaml:"preferred_lifetime,omitempty"`
	Prefix            string `yaml:"prefix,omitempty"`
}

// Dhcpv6Config represents DHCPv6 server configuration
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

// Dhcpv6Pool represents an IPv6 address pool
type Dhcpv6Pool struct {
	Network    string `yaml:"network,omitempty"`
	RangeStart string `yaml:"range_start,omitempty"`
	RangeEnd   string `yaml:"range_end,omitempty"`
}

// TrafficConfig represents traffic pattern configuration (v1.6.0)
type TrafficConfig struct {
	Enabled          bool                   `yaml:"enabled,omitempty"`
	ARPAnnouncements *ARPAnnouncementConfig `yaml:"arp_announcements,omitempty"`
	PeriodicPings    *PeriodicPingConfig    `yaml:"periodic_pings,omitempty"`
	RandomTraffic    *RandomTrafficConfig   `yaml:"random_traffic,omitempty"`
}

// ARPAnnouncementConfig configures gratuitous ARP announcements
type ARPAnnouncementConfig struct {
	Enabled  bool `yaml:"enabled,omitempty"`
	Interval int  `yaml:"interval,omitempty"` // seconds
}

// PeriodicPingConfig configures periodic ICMP echo requests
type PeriodicPingConfig struct {
	Enabled     bool `yaml:"enabled,omitempty"`
	Interval    int  `yaml:"interval,omitempty"`     // seconds
	PayloadSize int  `yaml:"payload_size,omitempty"` // bytes
}

// RandomTrafficConfig configures random background traffic
type RandomTrafficConfig struct {
	Enabled     bool     `yaml:"enabled,omitempty"`
	Interval    int      `yaml:"interval,omitempty"`     // seconds
	PacketCount int      `yaml:"packet_count,omitempty"` // packets per interval
	Patterns    []string `yaml:"patterns,omitempty"`     // traffic patterns
}

// TrapsConfig represents SNMP trap configuration (v1.6.0)
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

// TrapTriggerConfig configures a simple trap trigger
type TrapTriggerConfig struct {
	Enabled   bool `yaml:"enabled,omitempty"`
	OnStartup bool `yaml:"on_startup,omitempty"`
}

// LinkStateTrapConfig configures link up/down traps
type LinkStateTrapConfig struct {
	Enabled  bool `yaml:"enabled,omitempty"`
	LinkDown bool `yaml:"link_down,omitempty"`
	LinkUp   bool `yaml:"link_up,omitempty"`
}

// ThresholdTrapConfig configures threshold-based traps
type ThresholdTrapConfig struct {
	Enabled   bool `yaml:"enabled,omitempty"`
	Threshold int  `yaml:"threshold,omitempty"` // threshold value
	Interval  int  `yaml:"interval,omitempty"`  // check interval in seconds
}

// Parser handles parsing Java DSL format
type Parser struct {
	lines   []string
	pos     int
	verbose bool
}

// ConvertFile converts a Java DSL config file to YAML
func ConvertFile(inputPath, outputPath string, verbose bool) error {
	// Read input file
	data, err := os.ReadFile(inputPath) // #nosec G304 -- user-provided file path, validated by caller
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
	if err := os.WriteFile(outputPath, yamlData, 0600); err != nil {
		return fmt.Errorf("error writing output file: %w", err)
	}

	return nil
}

// Parse parses the Java DSL format
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

// parseCapturePlayback parses a CapturePlayback block
// nolint:unparam // Error return reserved for future validation
func (p *Parser) parseCapturePlayback() (*CapturePlayback, error) {
	p.pos++ // Skip opening line
	playback := &CapturePlayback{}

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])

		if line == ")" {
			p.pos++
			break
		}

		if strings.HasPrefix(line, "FileName(") {
			playback.FileName = p.extractString(line)
		} else if strings.HasPrefix(line, "LoopTime(") {
			var loopTime int
			n, err := fmt.Sscanf(line, "LoopTime(%d)", &loopTime)
			if err != nil || n != 1 {
				return nil, fmt.Errorf("line %d: invalid LoopTime format: %s", p.pos+1, line)
			}
			playback.LoopTime = loopTime
		} else if strings.HasPrefix(line, "ScaleTime(") {
			var scaleTime float64
			n, err := fmt.Sscanf(line, "ScaleTime(%f)", &scaleTime)
			if err != nil || n != 1 {
				return nil, fmt.Errorf("line %d: invalid ScaleTime format: %s", p.pos+1, line)
			}
			playback.ScaleTime = scaleTime
		}

		p.pos++
	}

	return playback, nil
}

// parseDevice parses a Device block
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

		if strings.HasPrefix(line, "MacAddr(") {
			device.MAC = p.formatMAC(p.extractValue(line))
		} else if strings.HasPrefix(line, "IpAddr(") {
			ip := p.extractValue(line)
			if device.IP == "" {
				device.IP = ip
			} else if ip != "" {
				device.IPs = append(device.IPs, ip)
			}
		} else if strings.HasPrefix(line, "Ip6Addr(") {
			ip := p.extractValue(line)
			if ip != "" {
				device.IPs = append(device.IPs, ip)
			}
		} else if strings.HasPrefix(line, "MapToIp(") {
			device.MapToIP = p.extractValue(line)
		} else if strings.HasPrefix(line, "Babble(") {
			device.Babble = true
		} else if strings.HasPrefix(line, "TTL(") {
			ttlCfg := p.parseTTL(line)
			if ttlCfg != nil {
				device.TTL = ttlCfg
			}
		} else if strings.HasPrefix(line, "Vlan(") {
			var vlan int
			n, err := fmt.Sscanf(line, "Vlan(%d)", &vlan)
			if err != nil || n != 1 {
				return nil, fmt.Errorf("line %d: invalid Vlan format: %s", p.pos+1, line)
			}
			device.VLAN = vlan
		} else if strings.HasPrefix(line, "SpanningTree(") {
			if device.Stp == nil {
				device.Stp = &StpConfig{Enabled: true}
			} else {
				device.Stp.Enabled = true
			}
		} else if strings.HasPrefix(line, "SnmpAgent(") {
			agent, err := p.parseSnmpAgent()
			if err != nil {
				return nil, err
			}
			device.SnmpAgent = mergeSnmpAgent(device.SnmpAgent, agent)
			continue
		} else if strings.HasPrefix(line, "SnmpAccessList(") {
			accessList, err := p.parseSnmpAccessList()
			if err != nil {
				return nil, err
			}
			if device.SnmpAgent == nil {
				device.SnmpAgent = &SnmpAgent{}
			}
			device.SnmpAgent.AccessList = append(device.SnmpAgent.AccessList, accessList...)
			continue
		} else if strings.HasPrefix(line, "NetBiosStatus(") {
			netbios, err := p.parseNetBiosStatus()
			if err != nil {
				return nil, err
			}
			device.Netbios = mergeNetbios(device.Netbios, netbios)
			continue
		} else if strings.HasPrefix(line, "Icmp(") {
			icmp, err := p.parseIcmp()
			if err != nil {
				return nil, err
			}
			device.Icmp = icmp
			continue
		} else if strings.HasPrefix(line, "Icmp6(") {
			icmp6, err := p.parseIcmp6()
			if err != nil {
				return nil, err
			}
			device.Icmpv6 = icmp6
			continue
		} else if strings.HasPrefix(line, "Dhcp(") {
			dhcp, err := p.parseDhcp()
			if err != nil {
				return nil, err
			}
			device.Dhcp = dhcp
			continue
		} else if strings.HasPrefix(line, "Dns(") {
			dns, err := p.parseDns()
			if err != nil {
				return nil, err
			}
			device.Dns = dns
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

// parseSnmpAgent parses an SnmpAgent block
// nolint:unparam // Error return reserved for future validation
func (p *Parser) parseSnmpAgent() (*SnmpAgent, error) {
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

		if strings.HasPrefix(line, "Include(") {
			walk := p.extractString(line)
			if walk != "" {
				if agent.WalkFile == "" {
					agent.WalkFile = walk
				}
				agent.WalkFiles = append(agent.WalkFiles, walk)
			}
		} else if strings.HasPrefix(line, "CommunityInclude(") {
			community, walk := p.parseCommunityInclude(line)
			if community != "" && walk != "" {
				agent.CommunityIncludes = append(agent.CommunityIncludes, CommunityInclude{
					Community: community,
					WalkFile:  walk,
				})
			}
		} else if strings.HasPrefix(line, "SnmpAddr(") {
			agent.SnmpAddr = p.extractValue(line)
		} else if strings.HasPrefix(line, "Dot1D_FdbTable(") {
			if cfg := p.parseFdbTable(line, true); cfg != nil {
				agent.Dot1DFdbTable = cfg
			}
		} else if strings.HasPrefix(line, "Dot1Q_FdbTable(") {
			if cfg := p.parseFdbTable(line, false); cfg != nil {
				agent.Dot1QFdbTable = cfg
			}
		} else if strings.HasPrefix(line, "AddMib(") {
			mibLine := p.collectQuotedDirective(line, 3)
			mib := p.parseAddMib(mibLine)
			if mib != nil {
				agent.AddMibs = append(agent.AddMibs, *mib)
			}
		}

		p.pos++
	}

	return agent, nil
}

// parseSnmpAccessList parses SnmpAccessList block.
func (p *Parser) parseSnmpAccessList() ([]string, error) {
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

	return accessList, nil
}

// parseTTL parses TTL(ttl ip mask).
func (p *Parser) parseTTL(line string) *TTLConfig {
	re := regexp.MustCompile(`\(([^)]*)\)`)
	match := re.FindStringSubmatch(line)
	if len(match) < 2 {
		return nil
	}
	parts := strings.Fields(match[1])
	if len(parts) < 3 {
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
func (p *Parser) parseNetBiosStatus() (*NetbiosConfig, error) {
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

	return netbios, nil
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
func (p *Parser) parseIcmp() (*IcmpConfig, error) {
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
			ra, err := p.parseIcmpRouterAdvertisement()
			if err != nil {
				return nil, err
			}
			icmp.RouterAdvertisement = ra
			continue
		}

		p.pos++
	}

	return icmp, nil
}

// parseIcmpRouterAdvertisement parses RouterAdvertisement block for IPv4.
func (p *Parser) parseIcmpRouterAdvertisement() (*IcmpRouterAdvertisement, error) {
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

		if strings.HasPrefix(line, "Period(") {
			var period int
			if _, err := fmt.Sscanf(line, "Period(%d)", &period); err == nil {
				ra.Period = period
			}
		} else if strings.HasPrefix(line, "Lifetime(") {
			var lifetime int
			if _, err := fmt.Sscanf(line, "Lifetime(%d)", &lifetime); err == nil {
				ra.Lifetime = lifetime
			}
		} else if strings.HasPrefix(line, "Router(") {
			re := regexp.MustCompile(`\(([^)]*)\)`)
			match := re.FindStringSubmatch(line)
			if len(match) >= 2 {
				parts := strings.Fields(match[1])
				if len(parts) >= 2 {
					var pref int
					_, _ = fmt.Sscanf(parts[1], "%d", &pref)
					ra.Routers = append(ra.Routers, IcmpRouter{
						Address:    parts[0],
						Preference: pref,
					})
				}
			}
		}

		p.pos++
	}

	return ra, nil
}

// parseIcmp6 parses Icmp6 block.
func (p *Parser) parseIcmp6() (*Icmpv6Config, error) {
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
			ra, err := p.parseIcmpv6RouterAdvertisement()
			if err != nil {
				return nil, err
			}
			icmp6.RouterAdvertisement = ra
			continue
		}

		p.pos++
	}

	return icmp6, nil
}

// parseIcmpv6RouterAdvertisement parses IPv6 RouterAdvertisement block.
func (p *Parser) parseIcmpv6RouterAdvertisement() (*Icmpv6RouterAdvertisement, error) {
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

		switch {
		case strings.HasPrefix(line, "Period("):
			var v int
			_, _ = fmt.Sscanf(line, "Period(%d)", &v)
			ra.Period = v
		case strings.HasPrefix(line, "CurHopLimit("):
			var v int
			_, _ = fmt.Sscanf(line, "CurHopLimit(%d)", &v)
			ra.CurHopLimit = v
		case strings.HasPrefix(line, "Managed("):
			var v int
			_, _ = fmt.Sscanf(line, "Managed(%d)", &v)
			ra.Managed = v
		case strings.HasPrefix(line, "Other("):
			var v int
			_, _ = fmt.Sscanf(line, "Other(%d)", &v)
			ra.Other = v
		case strings.HasPrefix(line, "Lifetime("):
			var v int
			_, _ = fmt.Sscanf(line, "Lifetime(%d)", &v)
			ra.Lifetime = v
		case strings.HasPrefix(line, "ReachableTime("):
			var v int
			_, _ = fmt.Sscanf(line, "ReachableTime(%d)", &v)
			ra.ReachableTime = v
		case strings.HasPrefix(line, "RetransTimer("):
			var v int
			_, _ = fmt.Sscanf(line, "RetransTimer(%d)", &v)
			ra.RetransTimer = v
		case strings.HasPrefix(line, "MTU("):
			var v int
			_, _ = fmt.Sscanf(line, "MTU(%d)", &v)
			ra.MTU = v
		case strings.HasPrefix(line, "PrefixInformation("):
			prefix, err := p.parseIcmpv6PrefixInformation()
			if err != nil {
				return nil, err
			}
			if prefix != nil {
				ra.PrefixInfo = append(ra.PrefixInfo, *prefix)
			}
			continue
		}

		p.pos++
	}

	return ra, nil
}

// parseIcmpv6PrefixInformation parses PrefixInformation block.
func (p *Parser) parseIcmpv6PrefixInformation() (*Icmpv6PrefixInfo, error) {
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

	return prefix, nil
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
	for i := 0; i < len(line)-1; i++ {
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
	if len(matches) < 3 {
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
	if len(matches) < 2 {
		return "", ""
	}
	return matches[0][1], matches[1][1]
}

// parseFdbTable parses Dot1D_FdbTable or Dot1Q_FdbTable directives.
func (p *Parser) parseFdbTable(line string, dot1d bool) *FdbTableConfig {
	re := regexp.MustCompile(`\(([^)]*)\)`)
	match := re.FindStringSubmatch(line)
	if len(match) < 2 {
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

// parseDhcp parses a Dhcp block
// nolint:gocyclo // DHCP parser handles many option types
// nolint:unparam // Error return reserved for future validation
func (p *Parser) parseDhcp() (*DhcpServer, error) {
	p.pos++ // Skip opening line
	dhcp := &DhcpServer{
		ClientLeases: make([]DhcpLease, 0),
	}

	var currentLease *DhcpLease

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])

		if line == ")" {
			// Save current lease if exists
			if currentLease != nil {
				dhcp.ClientLeases = append(dhcp.ClientLeases, *currentLease)
			}
			p.pos++
			break
		}

		// Skip comments
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			p.pos++
			continue
		}

		if strings.HasPrefix(line, "YourClientIpAddr") {
			// Handle multiline format: YourClientIpAddr(10.250.1.138
			// where the IP is on the same line but closing paren is on a later line
			ip := p.extractValue(line)
			if ip == "" {
				// No closing paren on same line, extract everything after (
				start := strings.Index(line, "(")
				if start != -1 {
					ip = strings.TrimSpace(line[start+1:])
				}
			}
			currentLease = &DhcpLease{ClientIP: ip}
		} else if strings.HasPrefix(line, "MacAddrValue") {
			if currentLease != nil {
				currentLease.MacAddrValue = p.formatMAC(p.extractValue(line))
			}
		} else if strings.HasPrefix(line, "MacAddrMask") {
			if currentLease != nil {
				currentLease.MacAddrMask = p.formatMAC(p.extractValue(line))
			}
			// End of this lease, save it
			if currentLease != nil {
				dhcp.ClientLeases = append(dhcp.ClientLeases, *currentLease)
				currentLease = nil
			}
			// Skip the closing paren of YourClientIpAddr block on next line
			p.pos++
			if p.pos < len(p.lines) && strings.TrimSpace(p.lines[p.pos]) == ")" {
				p.pos++ // Skip the closing paren
			}
			continue // Continue to next iteration without incrementing again
		} else if strings.HasPrefix(line, "SubnetMask") {
			dhcp.SubnetMask = p.extractValue(line)
		} else if strings.HasPrefix(line, "Router") {
			// Extract just the IP, ignore priority number
			value := p.extractValue(line)
			if value != "" {
				dhcp.Router = strings.Fields(value)[0]
			}
		} else if strings.HasPrefix(line, "DomainNameServer") {
			dhcp.DomainNameServer = p.extractValue(line)
		} else if strings.HasPrefix(line, "NextServerIpAddr") {
			dhcp.NextServerIP = p.extractValue(line)
		} else if strings.HasPrefix(line, "ServerIdentifier") {
			dhcp.ServerIdentifier = p.extractValue(line)
		}

		p.pos++
	}

	return dhcp, nil
}

// parseDns parses a Dns block
// nolint:unparam // Error return reserved for future validation
func (p *Parser) parseDns() (*DnsServer, error) {
	p.pos++ // Skip opening line
	dns := &DnsServer{
		ForwardRecords: make([]DnsRecord, 0),
		ReverseRecords: make([]DnsRecord, 0),
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

		if strings.HasPrefix(line, "Forward2(") {
			record := p.parseDnsRecord(line, true)
			if record != nil {
				dns.ForwardRecords = append(dns.ForwardRecords, *record)
			}
		} else if strings.HasPrefix(line, "Forward(") {
			record := p.parseDnsRecord(line, true)
			if record != nil {
				dns.ForwardRecords = append(dns.ForwardRecords, *record)
			}
		} else if strings.HasPrefix(line, "Reverse2(") {
			record := p.parseDnsRecord(line, false)
			if record != nil {
				dns.ReverseRecords = append(dns.ReverseRecords, *record)
			}
		} else if strings.HasPrefix(line, "Reverse(") {
			record := p.parseDnsRecord(line, false)
			if record != nil {
				dns.ReverseRecords = append(dns.ReverseRecords, *record)
			}
		}

		p.pos++
	}

	return dns, nil
}

// parseDnsRecord parses a Forward() or Reverse() DNS record
func (p *Parser) parseDnsRecord(line string, isForward bool) *DnsRecord {
	// Forward("hostname" IP TTL)
	// Forward2("hostname" IP TTL RCODE)
	// Reverse(IP "hostname" TTL)
	// Reverse2(IP "hostname" TTL RCODE)
	re := regexp.MustCompile(`\((.*?)\)`)
	match := re.FindStringSubmatch(line)
	if len(match) < 2 {
		return nil
	}

	parts := strings.Fields(match[1])
	if len(parts) < 2 {
		return nil
	}

	record := &DnsRecord{}

	if isForward {
		// Forward("hostname" IP TTL)
		record.Name = strings.Trim(parts[0], "\"")
		record.IP = parts[1]
		if len(parts) >= 3 {
			_, _ = fmt.Sscanf(parts[2], "%d", &record.TTL)
		}
		if len(parts) >= 4 {
			_, _ = fmt.Sscanf(parts[3], "%d", &record.RCode)
		}
	} else {
		// Reverse(IP "hostname" TTL)
		record.IP = parts[0]
		record.Name = strings.Trim(parts[1], "\"")
		if len(parts) >= 3 {
			_, _ = fmt.Sscanf(parts[2], "%d", &record.TTL)
		}
		if len(parts) >= 4 {
			_, _ = fmt.Sscanf(parts[3], "%d", &record.RCode)
		}
	}

	return record
}

// extractString extracts a quoted string from a directive
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

// extractValue extracts a value from parentheses (no quotes)
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

// formatMAC converts XXXXXXXXXXXX to XX:XX:XX:XX:XX:XX
func (p *Parser) formatMAC(mac string) string {
	if len(mac) != 12 {
		return mac
	}
	return fmt.Sprintf("%s:%s:%s:%s:%s:%s",
		mac[0:2], mac[2:4], mac[4:6], mac[6:8], mac[8:10], mac[10:12])
}

// LoadYAMLConfig loads a YAML config file into Go config structure
func LoadYAMLConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename) // #nosec G304 -- user-provided file path, validated by caller
	if err != nil {
		return nil, fmt.Errorf("error reading YAML file: %w", err)
	}
	return LoadYAMLConfigFromBytes(data)
}

// LoadYAMLConfigFromBytes converts in-memory YAML data into a Go config structure.
func LoadYAMLConfigFromBytes(data []byte) (*Config, error) {
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing YAML: %w", err)
	}
	return &config, nil
}

// ValidateConfig validates that no functionality was lost in conversion
func ValidateConfig(config *Config) error {
	// Validate devices have required fields
	for i, device := range config.Devices {
		if device.MAC == "" {
			return fmt.Errorf("device %d missing MAC address", i)
		}
		// IP address is optional in Java configs (some devices don't have IPs)

		// If SNMP agent specified, validate it (empty SNMP agents are allowed)
		if device.SnmpAgent != nil && len(device.SnmpAgent.AddMibs) > 0 {
			// Validate AddMibs have required fields
			for j, mib := range device.SnmpAgent.AddMibs {
				if mib.OID == "" {
					return fmt.Errorf("device %d AddMib %d missing OID", i, j)
				}
				if mib.Type == "" {
					return fmt.Errorf("device %d AddMib %d missing type", i, j)
				}
			}
		}
	}

	// If capture playbacks specified, validate them
	for i, playback := range config.CapturePlaybacks {
		if playback.FileName == "" {
			return fmt.Errorf("CapturePlayback %d missing file name", i)
		}
	}

	return nil
}

// PrintSummary prints a summary of the config
func PrintSummary(config *Config, w *bufio.Writer) {
	fmt.Fprintf(w, "Configuration Summary:\n")
	fmt.Fprintf(w, "  Devices: %d\n", len(config.Devices))

	if config.IncludePath != "" {
		fmt.Fprintf(w, "  Include Path: %s\n", config.IncludePath)
	}

	if len(config.CapturePlaybacks) > 0 {
		fmt.Fprintf(w, "  PCAP Playbacks: %d\n", len(config.CapturePlaybacks))
		for i, playback := range config.CapturePlaybacks {
			fmt.Fprintf(w, "    [%d] %s\n", i+1, playback.FileName)
			if playback.LoopTime > 0 {
				fmt.Fprintf(w, "        Loop Time: %d ms\n", playback.LoopTime)
			}
			if playback.ScaleTime > 0 {
				fmt.Fprintf(w, "        Scale Time: %.2f\n", playback.ScaleTime)
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
		fmt.Fprintf(w, "  SNMP Agents: %d\n", snmpCount)
		if mibCount > 0 {
			fmt.Fprintf(w, "  Custom MIBs: %d\n", mibCount)
		}
	}

	w.Flush() // #nosec G104 -- error logged or non-critical
}
