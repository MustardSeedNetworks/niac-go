package api

// buildDHCPSchema returns the DHCP server configuration schema.
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

// buildDNSSchema returns the DNS server configuration schema.
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

// buildHTTPSchema returns the HTTP server configuration schema.
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

// buildFTPSchema returns the FTP server configuration schema.
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

// buildDHCPv6Schema returns the DHCPv6 server configuration schema.
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
