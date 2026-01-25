package api

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

// buildOSFingerprintSchema returns the OS fingerprint schema for device simulation.
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
