package api

// Network services schema builders (DHCP, DNS, HTTP, FTP, NetBIOS).

func buildDHCPSchema() *SchemaProperty {
	return &SchemaProperty{
		Type:        "object",
		Title:       "DHCP Server",
		Description: "DHCP server configuration",
		Properties: map[string]*SchemaProperty{
			"subnetMask": {
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
			"poolStart": {
				Type:        "string",
				Title:       "Pool Start",
				Description: "Start of DHCP address pool",
				Format:      "ipv4",
			},
			"poolEnd": {
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
			"serverName": {
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
			"welcomeBanner": {
				Type:        "string",
				Title:       "Welcome Banner",
				Description: "FTP welcome message",
			},
			"system_type": {
				Type:    "string",
				Title:   "System Type",
				Default: "UNIX Type: L8",
			},
			"allowAnonymous": {
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

func buildSSHSchema() *SchemaProperty {
	return &SchemaProperty{
		Type: "object", Title: "SSH Command Service",
		Description: "Authenticated command access for the simulated device",
		Properties: map[string]*SchemaProperty{
			"enabled":  {Type: "boolean", Title: "Enable SSH", Default: false},
			"username": {Type: "string", Title: "Username"},
			"passwordEnv": {
				Type: "string", Title: "Password Environment Variable",
				Pattern: `^[A-Za-z_][A-Za-z0-9_]*$`,
			},
		},
		Required: []string{"enabled"},
		AllOf: []*SchemaProperty{{
			If: &SchemaProperty{Properties: map[string]*SchemaProperty{
				"enabled": {Const: true},
			}},
			Then: &SchemaProperty{Required: []string{"username", "passwordEnv"}},
		}},
	}
}

func buildSyslogSchema() *SchemaProperty {
	receiverPattern := `^(?:(?:25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\.){3}` +
		`(?:25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9]):` +
		`(?:6553[0-5]|655[0-2][0-9]|65[0-4][0-9]{2}|6[0-4][0-9]{3}|[1-5][0-9]{4}|[1-9][0-9]{0,3})$`
	minReceivers := 1
	return &SchemaProperty{
		Type: "object", Title: "SYSLOG Output",
		Description: "Configuration-state messages sent to RFC 5424 collectors",
		Properties: map[string]*SchemaProperty{
			"enabled": {Type: "boolean", Title: "Enable SYSLOG", Default: false},
			"receivers": {
				Type: "array", Title: "Receivers",
				Items: &SchemaProperty{Type: "string", Pattern: receiverPattern},
			},
		},
		Required: []string{"enabled"},
		AllOf: []*SchemaProperty{{
			If: &SchemaProperty{Properties: map[string]*SchemaProperty{
				"enabled": {Const: true},
			}},
			Then: &SchemaProperty{
				Required: []string{"receivers"},
				Properties: map[string]*SchemaProperty{
					"receivers": {MinItems: &minReceivers},
				},
			},
		}},
	}
}
