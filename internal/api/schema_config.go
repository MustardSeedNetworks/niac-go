package api

// buildSNMPAgentSchema returns the SNMP agent configuration schema.
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

// buildLLDPSchema returns the LLDP configuration schema.
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

// buildCDPSchema returns the CDP configuration schema.
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

// buildEDPSchema returns the EDP configuration schema.
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

// buildFDPSchema returns the FDP configuration schema.
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

// buildSTPSchema returns the STP configuration schema.
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

// buildNetBIOSSchema returns the NetBIOS service configuration schema.
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

// buildTTLConfigSchema returns the TTL configuration schema for traceroute simulation.
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
