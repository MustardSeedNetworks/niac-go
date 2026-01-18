package httpapi

import (
	"encoding/json"
	"maps"
	"net/http"
)

// Schema validation constants.
const (
	maxHostnameLen          = 253     // RFC 1035 max domain name length
	lldpDefaultAdvertise    = 30      // LLDP default advertise interval seconds
	lldpDefaultTTL          = 120     // LLDP default TTL seconds
	cdpDefaultAdvertise     = 60      // CDP default advertise interval seconds
	cdpDefaultHoldtime      = 180     // CDP default holdtime seconds
	minHoldtime             = 10      // minimum holdtime seconds
	maxHoldtime             = 255     // maximum holdtime (byte max)
	maxAdvertiseInterval    = 3600    // max advertise interval (1 hour)
	maxTTLValue             = 65535   // max TTL value (uint16 max)
	stpDefaultHelloTime     = 2       // STP hello time default
	stpMaxHelloTime         = 10      // STP max hello time
	stpDefaultMaxAge        = 20      // STP max age default
	stpMinMaxAge            = 6       // STP min max age
	stpMaxMaxAge            = 40      // STP max max age
	stpDefaultForwardDelay  = 15      // STP forward delay default
	stpMinForwardDelay      = 4       // STP min forward delay
	stpMaxForwardDelay      = 30      // STP max forward delay
	stpDefaultPriority      = 32768   // STP default bridge priority
	httpDefaultStatusCode   = 200     // HTTP OK status
	httpMinStatusCode       = 100     // HTTP min status code
	httpMaxStatusCode       = 599     // HTTP max status code
	netbiosMaxNameLen       = 15      // NetBIOS name max length
	netbiosDefaultTTL       = 300     // NetBIOS default TTL seconds
	maxServerPreference     = 255     // max DHCPv6 server preference
	maxIPTTL                = 255     // max IP TTL value
	maxTCPWindowScale       = 14      // max TCP window scale option
	maxPacketLossPercent    = 100     // max packet loss percentage
	cdpDefaultVersion       = 2       // CDP default version
	edpDefaultAdvertise     = 30      // EDP default advertise interval
	fdpDefaultAdvertise     = 60      // FDP default advertise interval
	fdpDefaultHoldtime      = 180     // FDP default holdtime
	defaultDNSTTL           = 3600    // DNS default TTL (1 hour)
	defaultICMPTTL          = 64      // default ICMP TTL
	dhcpv6PreferredLifetime = 604800  // DHCPv6 preferred lifetime (7 days)
	dhcpv6ValidLifetime     = 2592000 // DHCPv6 valid lifetime (30 days)
	defaultPingInterval     = 120     // default periodic ping interval
	defaultPingPayload      = 32      // default ping payload size
	defaultTrafficInterval  = 180     // default random traffic interval
	defaultTrafficPacketCnt = 5       // default traffic packet count
	defaultIperfPort        = 5201    // default iperf port
	defaultIperfDuration    = 1000    // default iperf duration ms
	defaultBandwidthMbps    = 100     // default bandwidth in Mbps
)

// SchemaProperty represents a JSON Schema property definition.
type SchemaProperty struct {
	Type        string                     `json:"type"`
	Title       string                     `json:"title,omitempty"`
	Description string                     `json:"description,omitempty"`
	Default     any                        `json:"default,omitempty"`
	Enum        []any                      `json:"enum,omitempty"`
	EnumNames   []string                   `json:"enumNames,omitempty"`
	Minimum     *float64                   `json:"minimum,omitempty"`
	Maximum     *float64                   `json:"maximum,omitempty"`
	MinLength   *int                       `json:"minLength,omitempty"`
	MaxLength   *int                       `json:"maxLength,omitempty"`
	Pattern     string                     `json:"pattern,omitempty"`
	Format      string                     `json:"format,omitempty"`
	Items       *SchemaProperty            `json:"items,omitempty"`
	Properties  map[string]*SchemaProperty `json:"properties,omitempty"`
	Required    []string                   `json:"required,omitempty"`
	// UI hints for form rendering
	UIWidget      string `json:"ui:widget,omitempty"`
	UIHelp        string `json:"ui:help,omitempty"`
	UIPlaceholder string `json:"ui:placeholder,omitempty"`
}

// ConfigSchema represents the complete JSON Schema for device configuration.
type ConfigSchema struct {
	Schema      string                     `json:"$schema"`
	Type        string                     `json:"type"`
	Title       string                     `json:"title"`
	Description string                     `json:"description,omitempty"`
	Properties  map[string]*SchemaProperty `json:"properties"`
	Required    []string                   `json:"required"`
	Definitions map[string]*SchemaProperty `json:"definitions,omitempty"`
}

// handleConfigSchema returns the JSON Schema for device configuration
// GET /api/v1/config/schema.
func (s *Server) handleConfigSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	schema := buildDeviceSchema()

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(schema)
	if err != nil {
		s.logger.Error("[API] Failed to encode schema response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// buildCoreDeviceProperties returns the core device schema properties (hostname, mac, ip, etc).
func buildCoreDeviceProperties() map[string]*SchemaProperty {
	maxVLAN := 4094.0
	macLen := 17 // xx:xx:xx:xx:xx:xx

	return map[string]*SchemaProperty{
		"hostname": {
			Type:        "string",
			Title:       "Hostname",
			Description: "Unique device hostname/identifier",
			MinLength:   intPtr(1),
			MaxLength:   intPtr(maxHostnameLen),
			Pattern:     `^[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?$`,
			UIHelp:      "Enter a unique hostname for this device",
		},
		"mac": {
			Type:          "string",
			Title:         "MAC Address",
			Description:   "Device MAC address in format XX:XX:XX:XX:XX:XX",
			Pattern:       `^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`,
			MinLength:     &macLen,
			MaxLength:     &macLen,
			UIPlaceholder: "00:11:22:33:44:55",
			UIHelp:        "MAC address in colon-separated format",
		},
		"ip": {
			Type:          "string",
			Title:         "IP Address",
			Description:   "Primary IPv4 or IPv6 address",
			Format:        "ipv4",
			UIPlaceholder: "192.168.1.1",
		},
		"ips": {
			Type:        "array",
			Title:       "Additional IP Addresses",
			Description: "Additional IP addresses for this device",
			Items:       &SchemaProperty{Type: "string", Format: "ipv4"},
		},
		"type": buildDeviceTypeSchema(),
		"vlan": {
			Type:        "integer",
			Title:       "VLAN",
			Description: "VLAN membership ID (1-4094)",
			Minimum:     floatPtr(1),
			Maximum:     &maxVLAN,
		},
		"babble": {
			Type:        "boolean",
			Title:       "Enable Babble Traffic",
			Description: "Periodically emit background network traffic",
			Default:     false,
		},
		"map_to_ip": {
			Type:        "string",
			Title:       "Map to IP",
			Description: "Map UDP traffic to an external IP address",
			Format:      "ipv4",
		},
	}
}

// buildDeviceTypeSchema returns the device type enum schema.
func buildDeviceTypeSchema() *SchemaProperty {
	return &SchemaProperty{
		Type:        "string",
		Title:       "Device Type",
		Description: "Type of network device",
		Enum: []any{
			"router",
			"switch",
			"access_point",
			"firewall",
			"server",
			"workstation",
			"iot",
			"unknown",
		},
		EnumNames: []string{
			"Router",
			"Switch",
			"Access Point",
			"Firewall",
			"Server",
			"Workstation",
			"IoT Device",
			"Unknown",
		},
		Default: "switch",
	}
}

// buildProtocolSchemaProperties returns all protocol-related schema properties.
func buildProtocolSchemaProperties() map[string]*SchemaProperty {
	minZero := 0.0
	maxPort := 65535.0
	maxPriority := 61440.0
	maxTTL := 255.0

	return map[string]*SchemaProperty{
		"snmp_agent":     buildSNMPAgentSchema(),
		"lldp":           buildLLDPSchema(),
		"cdp":            buildCDPSchema(),
		"edp":            buildEDPSchema(),
		"fdp":            buildFDPSchema(),
		"stp":            buildSTPSchema(&maxPriority),
		"dhcp":           buildDHCPSchema(),
		"dns":            buildDNSSchema(),
		"http":           buildHTTPSchema(&maxPort),
		"ftp":            buildFTPSchema(),
		"netbios":        buildNetBIOSSchema(&maxTTL),
		"icmp":           buildICMPSchema(&maxTTL),
		"icmpv6":         buildICMPv6Schema(&maxTTL),
		"dhcpv6":         buildDHCPv6Schema(),
		"traffic":        buildTrafficSchema(&minZero),
		"ttl":            buildTTLConfigSchema(&maxTTL),
		"os_fingerprint": buildOSFingerprintSchema(),
		"iperf3":         buildIPerf3Schema(&maxPort),
	}
}

// buildSchemaDefinitions returns the common type definitions for the schema.
func buildSchemaDefinitions() map[string]*SchemaProperty {
	return map[string]*SchemaProperty{
		"ipv4": {Type: "string", Format: "ipv4", Pattern: `^(\d{1,3}\.){3}\d{1,3}$`},
		"ipv6": {Type: "string", Format: "ipv6"},
		"mac":  {Type: "string", Pattern: `^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`},
	}
}

// buildDeviceSchema creates the JSON Schema for device configuration.
func buildDeviceSchema() *ConfigSchema {
	// Merge core and protocol properties
	properties := buildCoreDeviceProperties()
	maps.Copy(properties, buildProtocolSchemaProperties())

	return &ConfigSchema{
		Schema:      "http://json-schema.org/draft-07/schema#",
		Type:        "object",
		Title:       "Device Configuration",
		Description: "NIAC network device configuration schema",
		Required:    []string{"hostname", "mac"},
		Properties:  properties,
		Definitions: buildSchemaDefinitions(),
	}
}

func buildSNMPAgentSchema() *SchemaProperty {
	return &SchemaProperty{
		Type:        "object",
		Title:       "SNMP Agent",
		Description: "SNMP agent configuration for device simulation",
		Properties: map[string]*SchemaProperty{
			"community": {
				Type:        "string",
				Title:       "Community String",
				Description: "SNMP community string for v1/v2c",
				Default:     "public",
			},
			"sys_name": {
				Type:        "string",
				Title:       "System Name",
				Description: "SNMP sysName (1.3.6.1.2.1.1.5)",
			},
			"sys_descr": {
				Type:        "string",
				Title:       "System Description",
				Description: "SNMP sysDescr (1.3.6.1.2.1.1.1)",
			},
			"sys_contact": {
				Type:        "string",
				Title:       "System Contact",
				Description: "SNMP sysContact (1.3.6.1.2.1.1.4)",
			},
			"sys_location": {
				Type:        "string",
				Title:       "System Location",
				Description: "SNMP sysLocation (1.3.6.1.2.1.1.6)",
			},
			"walk_file": {
				Type:        "string",
				Title:       "Walk File",
				Description: "Path to SNMP walk file",
				UIWidget:    "file-selector",
			},
			"walk_files": {
				Type:        "array",
				Title:       "Additional Walk Files",
				Description: "Paths to additional SNMP walk files",
				Items: &SchemaProperty{
					Type: "string",
				},
			},
			"add_mibs": {
				Type:        "array",
				Title:       "Custom MIB Entries",
				Description: "Custom MIB overrides/additions",
				Items: &SchemaProperty{
					Type: "object",
					Properties: map[string]*SchemaProperty{
						"oid": {
							Type:        "string",
							Title:       "OID",
							Description: "SNMP Object Identifier",
							Pattern:     `^\.?[0-9]+(\.[0-9]+)*$`,
						},
						"type": {
							Type:  "string",
							Title: "Type",
							Enum: []any{
								"STRING",
								"INTEGER",
								"Counter32",
								"Counter64",
								"Gauge32",
								"TimeTicks",
								"OID",
								"IpAddress",
								"Hex-STRING",
							},
							Default: "STRING",
						},
						"value": {
							Type:  "string",
							Title: "Value",
						},
					},
					Required: []string{"oid", "type", "value"},
				},
			},
			"access_list": {
				Type:        "array",
				Title:       "Access List",
				Description: "Allowed source IPs for SNMP queries",
				Items: &SchemaProperty{
					Type:   "string",
					Format: "ipv4",
				},
			},
		},
	}
}

func buildLLDPSchema() *SchemaProperty {
	return &SchemaProperty{
		Type:        "object",
		Title:       "LLDP Configuration",
		Description: "Link Layer Discovery Protocol settings",
		Properties: map[string]*SchemaProperty{
			"enabled": {
				Type:        "boolean",
				Title:       "Enable LLDP",
				Description: "Enable LLDP advertisements",
				Default:     false,
			},
			"advertise_interval": {
				Type:        "integer",
				Title:       "Advertise Interval",
				Description: "Advertisement interval in seconds",
				Default:     lldpDefaultAdvertise,
				Minimum:     floatPtr(1),
				Maximum:     floatPtr(maxAdvertiseInterval),
			},
			"ttl": {
				Type:        "integer",
				Title:       "TTL",
				Description: "Time-to-live in seconds",
				Default:     lldpDefaultTTL,
				Minimum:     floatPtr(1),
				Maximum:     floatPtr(maxTTLValue),
			},
			"chassis_id_type": {
				Type:      "string",
				Title:     "Chassis ID Type",
				Enum:      []any{"mac", "local", "network_address"},
				EnumNames: []string{"MAC Address", "Local", "Network Address"},
				Default:   "mac",
			},
			"system_description": {
				Type:        "string",
				Title:       "System Description",
				Description: "LLDP system description TLV",
			},
			"port_description": {
				Type:        "string",
				Title:       "Port Description",
				Description: "LLDP port description TLV",
			},
		},
	}
}

func buildCDPSchema() *SchemaProperty {
	return &SchemaProperty{
		Type:        "object",
		Title:       "CDP Configuration",
		Description: "Cisco Discovery Protocol settings",
		Properties: map[string]*SchemaProperty{
			"enabled": {
				Type:        "boolean",
				Title:       "Enable CDP",
				Description: "Enable CDP advertisements",
				Default:     false,
			},
			"advertise_interval": {
				Type:        "integer",
				Title:       "Advertise Interval",
				Description: "Advertisement interval in seconds",
				Default:     cdpDefaultAdvertise,
				Minimum:     floatPtr(1),
				Maximum:     floatPtr(maxAdvertiseInterval),
			},
			"holdtime": {
				Type:        "integer",
				Title:       "Holdtime",
				Description: "CDP holdtime in seconds",
				Default:     cdpDefaultHoldtime,
				Minimum:     floatPtr(minHoldtime),
				Maximum:     floatPtr(maxHoldtime),
			},
			"version": {
				Type:    "integer",
				Title:   "CDP Version",
				Enum:    []any{1, 2},
				Default: cdpDefaultVersion,
			},
			"platform": {
				Type:          "string",
				Title:         "Platform",
				Description:   "Device platform string (e.g., 'cisco WS-C3750X-48P')",
				UIPlaceholder: "cisco WS-C3750X-48P",
			},
			"software_version": {
				Type:        "string",
				Title:       "Software Version",
				Description: "Software version string",
			},
			"port_id": {
				Type:          "string",
				Title:         "Port ID",
				Description:   "Local port identifier",
				UIPlaceholder: "GigabitEthernet0/1",
			},
		},
	}
}

func buildEDPSchema() *SchemaProperty {
	return &SchemaProperty{
		Type:        "object",
		Title:       "EDP Configuration",
		Description: "Extreme Discovery Protocol settings",
		Properties: map[string]*SchemaProperty{
			"enabled": {
				Type:    "boolean",
				Title:   "Enable EDP",
				Default: false,
			},
			"advertise_interval": {
				Type:    "integer",
				Title:   "Advertise Interval",
				Default: edpDefaultAdvertise,
				Minimum: floatPtr(1),
			},
			"version_string": {
				Type:  "string",
				Title: "Version String",
			},
			"display_string": {
				Type:  "string",
				Title: "Display String",
			},
		},
	}
}

func buildFDPSchema() *SchemaProperty {
	return &SchemaProperty{
		Type:        "object",
		Title:       "FDP Configuration",
		Description: "Foundry Discovery Protocol settings",
		Properties: map[string]*SchemaProperty{
			"enabled": {
				Type:    "boolean",
				Title:   "Enable FDP",
				Default: false,
			},
			"advertise_interval": {
				Type:    "integer",
				Title:   "Advertise Interval",
				Default: fdpDefaultAdvertise,
				Minimum: floatPtr(1),
			},
			"holdtime": {
				Type:    "integer",
				Title:   "Holdtime",
				Default: fdpDefaultHoldtime,
			},
			"software_version": {
				Type:  "string",
				Title: "Software Version",
			},
			"platform": {
				Type:  "string",
				Title: "Platform",
			},
			"port_id": {
				Type:  "string",
				Title: "Port ID",
			},
		},
	}
}

func buildSTPSchema(maxPriority *float64) *SchemaProperty {
	return &SchemaProperty{
		Type:        "object",
		Title:       "STP Configuration",
		Description: "Spanning Tree Protocol settings",
		Properties: map[string]*SchemaProperty{
			"enabled": {
				Type:    "boolean",
				Title:   "Enable STP",
				Default: false,
			},
			"version": {
				Type:      "string",
				Title:     "STP Version",
				Enum:      []any{"stp", "rstp", "mstp"},
				EnumNames: []string{"STP", "RSTP (Rapid)", "MSTP (Multiple)"},
				Default:   "stp",
			},
			"bridge_priority": {
				Type:        "integer",
				Title:       "Bridge Priority",
				Description: "Bridge priority (0-61440, increments of 4096)",
				Default:     stpDefaultPriority,
				Minimum:     floatPtr(0),
				Maximum:     maxPriority,
			},
			"hello_time": {
				Type:        "integer",
				Title:       "Hello Time",
				Description: "Hello time in seconds",
				Default:     stpDefaultHelloTime,
				Minimum:     floatPtr(1),
				Maximum:     floatPtr(stpMaxHelloTime),
			},
			"max_age": {
				Type:        "integer",
				Title:       "Max Age",
				Description: "Max age in seconds",
				Default:     stpDefaultMaxAge,
				Minimum:     floatPtr(stpMinMaxAge),
				Maximum:     floatPtr(stpMaxMaxAge),
			},
			"forward_delay": {
				Type:        "integer",
				Title:       "Forward Delay",
				Description: "Forward delay in seconds",
				Default:     stpDefaultForwardDelay,
				Minimum:     floatPtr(stpMinForwardDelay),
				Maximum:     floatPtr(stpMaxForwardDelay),
			},
		},
	}
}

func buildDHCPSchema() *SchemaProperty {
	return &SchemaProperty{
		Type:        "object",
		Title:       "DHCP Server",
		Description: "DHCP server configuration",
		Properties: map[string]*SchemaProperty{
			"subnet_mask": {
				Type:          "string",
				Title:         "Subnet Mask",
				Format:        "ipv4",
				UIPlaceholder: "255.255.255.0",
			},
			"router": {
				Type:   "string",
				Title:  "Default Gateway",
				Format: "ipv4",
			},
			"domain_name_server": {
				Type:   "string",
				Title:  "DNS Server",
				Format: "ipv4",
			},
			"server_identifier": {
				Type:   "string",
				Title:  "Server Identifier",
				Format: "ipv4",
			},
			"pool_start": {
				Type:        "string",
				Title:       "Pool Start",
				Description: "Start of DHCP address pool",
				Format:      "ipv4",
			},
			"pool_end": {
				Type:        "string",
				Title:       "Pool End",
				Description: "End of DHCP address pool",
				Format:      "ipv4",
			},
			"domain_name": {
				Type:  "string",
				Title: "Domain Name",
			},
			"ntp_servers": {
				Type:  "array",
				Title: "NTP Servers",
				Items: &SchemaProperty{
					Type:   "string",
					Format: "ipv4",
				},
			},
			"domain_search": {
				Type:  "array",
				Title: "Domain Search List",
				Items: &SchemaProperty{
					Type: "string",
				},
			},
		},
	}
}

func buildDNSSchema() *SchemaProperty {
	return &SchemaProperty{
		Type:        "object",
		Title:       "DNS Server",
		Description: "DNS server configuration",
		Properties: map[string]*SchemaProperty{
			"forward_records": {
				Type:        "array",
				Title:       "Forward Records (A)",
				Description: "DNS A records for forward lookups",
				Items: &SchemaProperty{
					Type: "object",
					Properties: map[string]*SchemaProperty{
						"name": {
							Type:  "string",
							Title: "Hostname",
						},
						"ip": {
							Type:   "string",
							Title:  "IP Address",
							Format: "ipv4",
						},
						"ttl": {
							Type:    "integer",
							Title:   "TTL",
							Default: defaultDNSTTL,
						},
					},
					Required: []string{"name", "ip"},
				},
			},
			"reverse_records": {
				Type:        "array",
				Title:       "Reverse Records (PTR)",
				Description: "DNS PTR records for reverse lookups",
				Items: &SchemaProperty{
					Type: "object",
					Properties: map[string]*SchemaProperty{
						"name": {
							Type:  "string",
							Title: "Hostname",
						},
						"ip": {
							Type:   "string",
							Title:  "IP Address",
							Format: "ipv4",
						},
						"ttl": {
							Type:    "integer",
							Title:   "TTL",
							Default: defaultDNSTTL,
						},
					},
					Required: []string{"name", "ip"},
				},
			},
		},
	}
}

func buildHTTPSchema(_ *float64) *SchemaProperty {
	return &SchemaProperty{
		Type:        "object",
		Title:       "HTTP Server",
		Description: "HTTP server configuration",
		Properties: map[string]*SchemaProperty{
			"enabled": {
				Type:    "boolean",
				Title:   "Enable HTTP",
				Default: false,
			},
			"server_name": {
				Type:        "string",
				Title:       "Server Name",
				Description: "HTTP Server header value",
				Default:     "NIAC-Go/1.0.0",
			},
			"endpoints": {
				Type:        "array",
				Title:       "Custom Endpoints",
				Description: "Custom HTTP endpoint definitions",
				Items: &SchemaProperty{
					Type: "object",
					Properties: map[string]*SchemaProperty{
						"path": {
							Type:          "string",
							Title:         "Path",
							UIPlaceholder: "/api/info",
						},
						"method": {
							Type:    "string",
							Title:   "Method",
							Enum:    []any{"GET", "POST", "PUT", "DELETE"},
							Default: "GET",
						},
						"status_code": {
							Type:    "integer",
							Title:   "Status Code",
							Default: httpDefaultStatusCode,
							Minimum: floatPtr(httpMinStatusCode),
							Maximum: floatPtr(httpMaxStatusCode),
						},
						"content_type": {
							Type:    "string",
							Title:   "Content-Type",
							Default: "text/html",
						},
						"body": {
							Type:     "string",
							Title:    "Response Body",
							UIWidget: "textarea",
						},
					},
					Required: []string{"path"},
				},
			},
		},
	}
}

func buildFTPSchema() *SchemaProperty {
	return &SchemaProperty{
		Type:        "object",
		Title:       "FTP Server",
		Description: "FTP server configuration",
		Properties: map[string]*SchemaProperty{
			"enabled": {
				Type:    "boolean",
				Title:   "Enable FTP",
				Default: false,
			},
			"welcome_banner": {
				Type:        "string",
				Title:       "Welcome Banner",
				Description: "FTP welcome message",
			},
			"system_type": {
				Type:    "string",
				Title:   "System Type",
				Default: "UNIX Type: L8",
			},
			"allow_anonymous": {
				Type:    "boolean",
				Title:   "Allow Anonymous",
				Default: true,
			},
			"users": {
				Type:  "array",
				Title: "User Accounts",
				Items: &SchemaProperty{
					Type: "object",
					Properties: map[string]*SchemaProperty{
						"username": {
							Type:  "string",
							Title: "Username",
						},
						"password": {
							Type:     "string",
							Title:    "Password",
							UIWidget: "password",
						},
						"home_dir": {
							Type:    "string",
							Title:   "Home Directory",
							Default: "/",
						},
					},
					Required: []string{"username", "password"},
				},
			},
		},
	}
}

func buildNetBIOSSchema(_ *float64) *SchemaProperty {
	return &SchemaProperty{
		Type:        "object",
		Title:       "NetBIOS Service",
		Description: "NetBIOS name service configuration",
		Properties: map[string]*SchemaProperty{
			"enabled": {
				Type:    "boolean",
				Title:   "Enable NetBIOS",
				Default: false,
			},
			"name": {
				Type:        "string",
				Title:       "NetBIOS Name",
				Description: "NetBIOS name (max 15 characters)",
				MaxLength:   intPtr(netbiosMaxNameLen),
			},
			"workgroup": {
				Type:        "string",
				Title:       "Workgroup",
				Description: "Workgroup/domain name",
				Default:     "WORKGROUP",
			},
			"node_type": {
				Type:      "string",
				Title:     "Node Type",
				Enum:      []any{"B", "P", "M", "H"},
				EnumNames: []string{"Broadcast", "Peer", "Mixed", "Hybrid"},
				Default:   "B",
			},
			"services": {
				Type:        "array",
				Title:       "Services",
				Description: "Service types to advertise",
				Items: &SchemaProperty{
					Type: "string",
					Enum: []any{"workstation", "fileserver", "messenger"},
				},
				Default: []any{"workstation", "fileserver"},
			},
			"ttl": {
				Type:        "integer",
				Title:       "TTL",
				Description: "Name registration TTL in seconds",
				Default:     netbiosDefaultTTL,
				Minimum:     floatPtr(1),
			},
			"msbrowse": {
				Type:        "boolean",
				Title:       "Enable __MSBROWSE__",
				Description: "Enable master browser announcements",
				Default:     false,
			},
		},
	}
}

func buildICMPSchema(maxTTL *float64) *SchemaProperty {
	return &SchemaProperty{
		Type:        "object",
		Title:       "ICMP Configuration",
		Description: "ICMPv4 settings",
		Properties: map[string]*SchemaProperty{
			"enabled": {
				Type:    "boolean",
				Title:   "Enable ICMP",
				Default: true,
			},
			"ttl": {
				Type:        "integer",
				Title:       "TTL",
				Description: "Time-to-live for ICMP packets",
				Default:     defaultICMPTTL,
				Minimum:     floatPtr(1),
				Maximum:     maxTTL,
			},
			"rate_limit": {
				Type:        "integer",
				Title:       "Rate Limit",
				Description: "Max ICMP responses per second (0 = unlimited)",
				Default:     0,
				Minimum:     floatPtr(0),
			},
			"address_mask_reply": {
				Type:        "string",
				Title:       "Address Mask Reply",
				Description: "IP address for ICMP address mask replies",
				Format:      "ipv4",
			},
		},
	}
}

func buildICMPv6Schema(maxTTL *float64) *SchemaProperty {
	return &SchemaProperty{
		Type:        "object",
		Title:       "ICMPv6 Configuration",
		Description: "ICMPv6 settings",
		Properties: map[string]*SchemaProperty{
			"enabled": {
				Type:    "boolean",
				Title:   "Enable ICMPv6",
				Default: true,
			},
			"hop_limit": {
				Type:        "integer",
				Title:       "Hop Limit",
				Description: "Hop limit for ICMPv6 packets",
				Default:     defaultICMPTTL,
				Minimum:     floatPtr(1),
				Maximum:     maxTTL,
			},
			"rate_limit": {
				Type:        "integer",
				Title:       "Rate Limit",
				Description: "Max ICMPv6 responses per second (0 = unlimited)",
				Default:     0,
				Minimum:     floatPtr(0),
			},
		},
	}
}

func buildDHCPv6Schema() *SchemaProperty {
	return &SchemaProperty{
		Type:        "object",
		Title:       "DHCPv6 Server",
		Description: "DHCPv6 server configuration",
		Properties: map[string]*SchemaProperty{
			"enabled": {
				Type:    "boolean",
				Title:   "Enable DHCPv6",
				Default: false,
			},
			"pools": {
				Type:        "array",
				Title:       "Address Pools",
				Description: "IPv6 address pools",
				Items: &SchemaProperty{
					Type: "object",
					Properties: map[string]*SchemaProperty{
						"network": {
							Type:          "string",
							Title:         "Network",
							UIPlaceholder: "2001:db8::/64",
						},
						"range_start": {
							Type:  "string",
							Title: "Range Start",
						},
						"range_end": {
							Type:  "string",
							Title: "Range End",
						},
					},
				},
			},
			"preferred_lifetime": {
				Type:        "integer",
				Title:       "Preferred Lifetime",
				Description: "Preferred lifetime in seconds",
				Default:     dhcpv6PreferredLifetime,
			},
			"valid_lifetime": {
				Type:        "integer",
				Title:       "Valid Lifetime",
				Description: "Valid lifetime in seconds",
				Default:     dhcpv6ValidLifetime,
			},
			"preference": {
				Type:        "integer",
				Title:       "Server Preference",
				Description: "Server preference (0-255, higher is better)",
				Default:     0,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(maxHoldtime),
			},
			"dns_servers": {
				Type:  "array",
				Title: "DNS Servers",
				Items: &SchemaProperty{
					Type:   "string",
					Format: "ipv6",
				},
			},
			"domain_list": {
				Type:  "array",
				Title: "Domain Search List",
				Items: &SchemaProperty{
					Type: "string",
				},
			},
		},
	}
}

func buildTrafficSchema(minZero *float64) *SchemaProperty {
	return &SchemaProperty{
		Type:        "object",
		Title:       "Traffic Patterns",
		Description: "Background traffic pattern configuration",
		Properties: map[string]*SchemaProperty{
			"enabled": {
				Type:    "boolean",
				Title:   "Enable Traffic Generation",
				Default: false,
			},
			"arp_announcements": {
				Type:  "object",
				Title: "ARP Announcements",
				Properties: map[string]*SchemaProperty{
					"enabled": {
						Type:    "boolean",
						Title:   "Enable ARP Announcements",
						Default: false,
					},
					"interval": {
						Type:        "integer",
						Title:       "Interval",
						Description: "Interval in seconds",
						Default:     cdpDefaultAdvertise,
						Minimum:     floatPtr(1),
					},
				},
			},
			"periodic_pings": {
				Type:  "object",
				Title: "Periodic Pings",
				Properties: map[string]*SchemaProperty{
					"enabled": {
						Type:    "boolean",
						Title:   "Enable Periodic Pings",
						Default: false,
					},
					"interval": {
						Type:    "integer",
						Title:   "Interval",
						Default: defaultPingInterval,
						Minimum: floatPtr(1),
					},
					"payload_size": {
						Type:        "integer",
						Title:       "Payload Size",
						Description: "Payload size in bytes",
						Default:     defaultPingPayload,
						Minimum:     minZero,
					},
				},
			},
			"random_traffic": {
				Type:  "object",
				Title: "Random Traffic",
				Properties: map[string]*SchemaProperty{
					"enabled": {
						Type:    "boolean",
						Title:   "Enable Random Traffic",
						Default: false,
					},
					"interval": {
						Type:    "integer",
						Title:   "Interval",
						Default: defaultTrafficInterval,
						Minimum: floatPtr(1),
					},
					"packet_count": {
						Type:        "integer",
						Title:       "Packet Count",
						Description: "Number of packets per interval",
						Default:     defaultTrafficPacketCnt,
						Minimum:     floatPtr(1),
					},
					"patterns": {
						Type:  "array",
						Title: "Traffic Patterns",
						Items: &SchemaProperty{
							Type: "string",
							Enum: []any{"broadcast_arp", "multicast", "udp"},
						},
						Default: []any{"broadcast_arp", "multicast", "udp"},
					},
				},
			},
		},
	}
}

func buildTTLConfigSchema(maxTTL *float64) *SchemaProperty {
	return &SchemaProperty{
		Type:        "object",
		Title:       "TTL Configuration",
		Description: "ICMP TTL timeout behavior for traceroute simulation",
		Properties: map[string]*SchemaProperty{
			"ttl": {
				Type:        "integer",
				Title:       "TTL",
				Description: "TTL value at which to respond",
				Minimum:     floatPtr(1),
				Maximum:     maxTTL,
			},
			"ip": {
				Type:        "string",
				Title:       "Response IP",
				Description: "IP address to use in TTL exceeded response",
				Format:      "ipv4",
			},
			"mask": {
				Type:        "string",
				Title:       "Network Mask",
				Description: "Network mask for matching",
				Format:      "ipv4",
			},
		},
	}
}

func buildOSFingerprintSchema() *SchemaProperty {
	return &SchemaProperty{
		Type:        "object",
		Title:       "OS Fingerprint",
		Description: "OS fingerprinting configuration for realistic device simulation",
		Properties: map[string]*SchemaProperty{
			"os_type": {
				Type:        "string",
				Title:       "OS Type",
				Description: "Operating system type to emulate",
				Enum: []any{
					"linux",
					"windows",
					"macos",
					"freebsd",
					"cisco-ios",
					"cisco-nxos",
					"juniper-junos",
					"arista-eos",
				},
				EnumNames: []string{
					"Linux",
					"Windows",
					"macOS",
					"FreeBSD",
					"Cisco IOS",
					"Cisco NX-OS",
					"Juniper JunOS",
					"Arista EOS",
				},
			},
			"ttl": {
				Type:        "integer",
				Title:       "Default TTL",
				Description: "Default IP TTL (Linux=64, Windows=128, Cisco=255)",
				Minimum:     floatPtr(1),
				Maximum:     floatPtr(maxHoldtime),
			},
			"window_size": {
				Type:        "integer",
				Title:       "TCP Window Size",
				Description: "TCP window size",
			},
			"window_scale": {
				Type:        "integer",
				Title:       "TCP Window Scale",
				Description: "TCP window scale option",
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(maxTCPWindowScale),
			},
			"mss": {
				Type:        "integer",
				Title:       "TCP MSS",
				Description: "TCP maximum segment size",
			},
			"ssh_banner": {
				Type:          "string",
				Title:         "SSH Banner",
				Description:   "SSH version banner",
				UIPlaceholder: "SSH-2.0-OpenSSH_8.9p1",
			},
			"http_server": {
				Type:        "string",
				Title:       "HTTP Server Header",
				Description: "HTTP Server header value",
			},
			"ftp_banner": {
				Type:        "string",
				Title:       "FTP Banner",
				Description: "FTP welcome banner",
			},
			"dont_fragment": {
				Type:        "boolean",
				Title:       "Don't Fragment",
				Description: "IP Don't Fragment bit (Linux=true, Windows=false)",
			},
		},
	}
}

func buildIPerf3Schema(maxPort *float64) *SchemaProperty {
	return &SchemaProperty{
		Type:        "object",
		Title:       "iPerf3 Server",
		Description: "iPerf3 bandwidth testing server emulation",
		Properties: map[string]*SchemaProperty{
			"enabled": {
				Type:    "boolean",
				Title:   "Enable iPerf3",
				Default: false,
			},
			"port": {
				Type:        "integer",
				Title:       "Port",
				Description: "Listen port",
				Default:     defaultIperfPort,
				Minimum:     floatPtr(1),
				Maximum:     maxPort,
			},
			"max_bandwidth_mbps": {
				Type:        "number",
				Title:       "Max Bandwidth (Mbps)",
				Description: "Maximum simulated bandwidth",
				Default:     defaultIperfDuration,
			},
			"typical_latency_ms": {
				Type:        "number",
				Title:       "Typical Latency (ms)",
				Description: "Simulated network latency",
				Default:     1,
			},
			"jitter_ms": {
				Type:        "number",
				Title:       "Jitter (ms)",
				Description: "Simulated jitter for UDP tests",
				Default:     0,
			},
			"packet_loss_percent": {
				Type:        "number",
				Title:       "Packet Loss (%)",
				Description: "Simulated packet loss percentage",
				Default:     0,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(maxPacketLossPercent),
			},
			"upload_mbps": {
				Type:        "number",
				Title:       "Upload (Mbps)",
				Description: "Simulated upload bandwidth",
				Default:     defaultBandwidthMbps,
			},
			"download_mbps": {
				Type:        "number",
				Title:       "Download (Mbps)",
				Description: "Simulated download bandwidth",
				Default:     defaultBandwidthMbps,
			},
		},
	}
}

// Helper functions for pointer values.
func floatPtr(v float64) *float64 {
	return &v
}

func intPtr(v int) *int {
	return &v
}
