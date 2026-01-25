package api

// buildICMPSchema returns the ICMPv4 configuration schema.
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

// buildICMPv6Schema returns the ICMPv6 configuration schema.
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

// buildTrafficSchema returns the traffic patterns configuration schema.
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

// buildIPerf3Schema returns the iPerf3 server configuration schema.
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
