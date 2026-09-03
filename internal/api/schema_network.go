package api

// Network layer schema builders (ICMP, ICMPv6, DHCPv6, Traffic, TTL, OS Fingerprint, iPerf3).

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

// buildReflectorSchema creates the JSON Schema for the NetAlly UDP reflector.
func buildReflectorSchema() *SchemaProperty {
	return &SchemaProperty{
		Type:        "object",
		Title:       "UDP Reflector",
		Description: "NetAlly-style UDP reflector endpoint for performance/TrueSpeed tests",
		Properties: map[string]*SchemaProperty{
			"latency_ms": {
				Type:        "integer",
				Title:       "Latency (ms)",
				Description: "Delay before reflecting a probe",
				Default:     0,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(maxReflectorDelayMs),
			},
			"jitter_ms": {
				Type:        "integer",
				Title:       "Jitter (ms)",
				Description: "Random +/- delay around latency",
				Default:     0,
				Minimum:     floatPtr(0),
				Maximum:     floatPtr(maxReflectorDelayMs),
			},
			"dscp": {
				Type:        "boolean",
				Title:       "DSCP wiggle",
				Description: "Toggle the DSCP bottom-2 bits instead of the IP-precedence bit",
				Default:     false,
			},
		},
	}
}
