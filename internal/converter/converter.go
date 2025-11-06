// Package converter implements Java NIAC DSL to YAML conversion
package converter

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
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
	Traffic   *TrafficConfig `yaml:"traffic,omitempty"` // v1.6.0
}

// SnmpAgent represents SNMP agent configuration
type SnmpAgent struct {
	WalkFile string       `yaml:"walk_file,omitempty"`
	AddMibs  []AddMib     `yaml:"add_mibs,omitempty"`
	Traps    *TrapsConfig `yaml:"traps,omitempty"` // v1.6.0
}

// AddMib represents a MIB override or addition
type AddMib struct {
	OID   string `yaml:"oid"`
	Type  string `yaml:"type"`
	Value string `yaml:"value"`
}

// DhcpServer represents DHCP server configuration
type DhcpServer struct {
	ClientLeases     []DhcpLease `yaml:"client_leases,omitempty"`
	SubnetMask       string      `yaml:"subnet_mask,omitempty"`
	Router           string      `yaml:"router,omitempty"`
	DomainNameServer string      `yaml:"domain_name_server,omitempty"`
	NextServerIP     string      `yaml:"next_server_ip,omitempty"`
	ServerIdentifier string      `yaml:"server_identifier,omitempty"`
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
	Name string `yaml:"name"`
	IP   string `yaml:"ip"`
	TTL  int    `yaml:"ttl,omitempty"`
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
	Enabled   bool     `yaml:"enabled,omitempty"`
	Name      string   `yaml:"name,omitempty"`
	Workgroup string   `yaml:"workgroup,omitempty"`
	NodeType  string   `yaml:"node_type,omitempty"`
	Services  []string `yaml:"services,omitempty"`
	TTL       uint32   `yaml:"ttl,omitempty"`
}

// IcmpConfig represents ICMP/ICMPv4 configuration
type IcmpConfig struct {
	Enabled   bool  `yaml:"enabled,omitempty"`
	TTL       uint8 `yaml:"ttl,omitempty"`
	RateLimit int   `yaml:"rate_limit,omitempty"`
}

// Icmpv6Config represents ICMPv6 configuration
type Icmpv6Config struct {
	Enabled   bool  `yaml:"enabled,omitempty"`
	HopLimit  uint8 `yaml:"hop_limit,omitempty"`
	RateLimit int   `yaml:"rate_limit,omitempty"`
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
	data, err := os.ReadFile(inputPath)
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
			_, _ = fmt.Sscanf(line, "LoopTime(%d)", &loopTime)
			playback.LoopTime = loopTime
		} else if strings.HasPrefix(line, "ScaleTime(") {
			var scaleTime float64
			_, _ = fmt.Sscanf(line, "ScaleTime(%f)", &scaleTime)
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
			device.IP = p.extractValue(line)
		} else if strings.HasPrefix(line, "Vlan(") {
			var vlan int
			_, _ = fmt.Sscanf(line, "Vlan(%d)", &vlan)
			device.VLAN = vlan
		} else if strings.HasPrefix(line, "SnmpAgent(") {
			agent, err := p.parseSnmpAgent()
			if err != nil {
				return nil, err
			}
			device.SnmpAgent = agent
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
			agent.WalkFile = p.extractString(line)
		} else if strings.HasPrefix(line, "AddMib(") {
			mib := p.parseAddMib(line)
			if mib != nil {
				agent.AddMibs = append(agent.AddMibs, *mib)
			}
		}

		p.pos++
	}

	return agent, nil
}

// parseAddMib parses an AddMib directive
func (p *Parser) parseAddMib(line string) *AddMib {
	// AddMib("OID", "type", "value")
	// Extract using regex
	re := regexp.MustCompile(`AddMib\("([^"]+)",\s*"([^"]+)",\s*"([^"]+)"\)`)
	matches := re.FindStringSubmatch(line)

	if len(matches) != 4 {
		return nil
	}

	return &AddMib{
		OID:   matches[1],
		Type:  matches[2],
		Value: matches[3],
	}
}

// parseDhcp parses a Dhcp block
// nolint:gocyclo // DHCP parser handles many option types
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

		if strings.HasPrefix(line, "Forward(") {
			record := p.parseDnsRecord(line, true)
			if record != nil {
				dns.ForwardRecords = append(dns.ForwardRecords, *record)
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
	// Reverse(IP "hostname" TTL)
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
	} else {
		// Reverse(IP "hostname" TTL)
		record.IP = parts[0]
		record.Name = strings.Trim(parts[1], "\"")
		if len(parts) >= 3 {
			_, _ = fmt.Sscanf(parts[2], "%d", &record.TTL)
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
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading YAML file: %w", err)
	}

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

	w.Flush()
}
