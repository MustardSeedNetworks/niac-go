package sanitize

import (
	"strings"
	"testing"
)

func TestSanitizeIP(t *testing.T) {
	tests := []struct {
		name       string
		ip         string
		wantOctet2 string
	}{
		{"10.x private", "10.1.2.3", "0"},
		{"172.x private", "172.16.0.1", "1"},
		{"192.x private", "192.168.0.1", "2"},
		{"public IP low octet", "8.8.4.4", "100"},   // first octet < 10 maps to management
		{"public IP high octet", "100.64.0.1", "3"}, // first octet > 63 maps to remote
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping := NewMapping()

			result := sanitizeIP(tt.ip, mapping)
			if !strings.HasPrefix(result, "10.") {
				t.Errorf("sanitizeIP() = %v, want IP in 10.0.0.0/8 network", result)
			}
			parts := strings.Split(result, ".")
			if len(parts) != 4 {
				t.Fatalf("sanitizeIP() = %v, not a valid IP", result)
			}
			if parts[1] != tt.wantOctet2 {
				t.Errorf("sanitizeIP(%q) second octet = %q, want %q", tt.ip, parts[1], tt.wantOctet2)
			}

			// Determinism.
			if result2 := sanitizeIP(tt.ip, mapping); result != result2 {
				t.Errorf("sanitizeIP() not deterministic: first=%v, second=%v", result, result2)
			}
			if mapping.IPMappings[tt.ip] != result {
				t.Errorf("mapping not stored: got %v, want %v", mapping.IPMappings[tt.ip], result)
			}
		})
	}
}

func TestSanitizeIPSpecialCases(t *testing.T) {
	tests := []struct {
		name  string
		ip    string
		valid bool
	}{
		{"Invalid IP returns original", "not-an-ip", false},
		{"IPv6 returns original", "2001:db8::1", false},
		{"Valid IPv4", "192.168.1.100", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeIP(tt.ip, NewMapping())
			if tt.valid {
				if result == tt.ip {
					t.Errorf("sanitizeIP() = %v, want sanitized IP", result)
				}
			} else if result != tt.ip {
				t.Errorf("sanitizeIP() = %v, want original %v for invalid IP", result, tt.ip)
			}
		})
	}
}

func TestIsSpecialIP(t *testing.T) {
	tests := []struct {
		name      string
		ip        string
		isSpecial bool
	}{
		{"Localhost", "127.0.0.1", true},
		{"All zeros", "0.0.0.0", true},
		{"Broadcast", "255.255.255.255", true},
		{"Multicast", "224.0.0.1", true},
		{"Normal IP", "192.168.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSpecialIP(tt.ip); got != tt.isSpecial {
				t.Errorf("isSpecialIP() = %v, want %v", got, tt.isSpecial)
			}
		})
	}
}

func TestLooksLikeIPOctet(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"Valid single digit", "1", true},
		{"Valid two digits", "25", true},
		{"Valid three digits", "255", true},
		{"Invalid - too large", "256", false},
		{"Invalid - four digits", "1234", false},
		{"Invalid - not a number", "abc", false},
		{"Invalid - empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := looksLikeIPOctet(tt.s); got != tt.want {
				t.Errorf("looksLikeIPOctet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizeHostname(t *testing.T) {
	tests := []struct {
		name           string
		hostname       string
		wantDeviceType string
	}{
		{"Switch hostname", "core-sw-01", "sw"},
		{"Router hostname", "edge-rtr-nyc", "rtr"},
		{"Access Point", "wifi-ap-floor2", "ap"},
		{"Server", "db-srv-01", "srv"},
		{"Firewall", "perimeter-fw", "fw"},
		{"Unknown device", "device123", "dev"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping := NewMapping()

			result := sanitizeHostname(tt.hostname, mapping)
			if !strings.HasPrefix(result, "niac-core-") {
				t.Errorf("sanitizeHostname() = %v, want prefix niac-core-", result)
			}
			if !strings.Contains(result, tt.wantDeviceType) {
				t.Errorf("sanitizeHostname() = %v, want device type %v", result, tt.wantDeviceType)
			}

			if result2 := sanitizeHostname(tt.hostname, mapping); result != result2 {
				t.Errorf("sanitizeHostname() not deterministic: first=%v, second=%v", result, result2)
			}
			if mapping.Hostnames[tt.hostname] != result {
				t.Errorf("mapping not stored: got %v, want %v", mapping.Hostnames[tt.hostname], result)
			}
		})
	}
}

func sanitizeLineDefaults(line string) string {
	return sanitizeLine(line, NewMapping(), DefaultOptions())
}

func TestSanitizeLineSystemContact(t *testing.T) {
	result := sanitizeLine(
		`SNMPv2-MIB::sysContact.0 = STRING: admin@company.com`,
		NewMapping(),
		Options{Domain: "niac-go.com", Location: "DC-WEST", Contact: "ops@niac.dev", Community: "public"},
	)
	if !strings.Contains(result, "ops@niac.dev") {
		t.Errorf("Expected contact to be replaced, got: %s", result)
	}
}

func TestSanitizeLineSystemLocation(t *testing.T) {
	result := sanitizeLine(
		`SNMPv2-MIB::sysLocation.0 = STRING: Building A, Floor 3`,
		NewMapping(),
		Options{Domain: "niac-go.com", Location: "DC-EAST", Contact: "ops@niac.dev", Community: "public"},
	)
	if !strings.Contains(result, "DC-EAST") {
		t.Errorf("Expected location DC-EAST, got: %s", result)
	}
}

func TestSanitizeLineOIDIP(t *testing.T) {
	result := sanitizeLineDefaults(`.1.3.6.1.2.1.4.20.1.1.192.168.1.100 = IpAddress: 192.168.1.100`)
	if strings.Contains(result, "192.168.1.100") {
		t.Error("Expected IP to be sanitized")
	}
	if !strings.Contains(result, "10.") {
		t.Error("Expected sanitized IP to be in 10.x.x.x network")
	}
}

func TestSanitizeLineCommunity(t *testing.T) {
	result := sanitizeLineDefaults(`SNMPv2-MIB::snmpCommunity.1 = STRING: secretCommunity`)
	if !strings.Contains(result, "public") {
		t.Errorf("Expected community to be replaced with 'public', got: %s", result)
	}
	if strings.Contains(result, "secretCommunity") {
		t.Error("Original community string should be replaced")
	}
}

func TestSanitizeLineSpecialIP(t *testing.T) {
	result := sanitizeLineDefaults(`.1.3.6.1.2.1.4.20.1.1.127.0.0.1 = IpAddress: 127.0.0.1`)
	if !strings.Contains(result, "127.0.0.1") {
		t.Error("Localhost IP should not be transformed")
	}
}

func TestSanitizeLineDNS(t *testing.T) {
	tests := []struct {
		name         string
		line         string
		wantContains string
	}{
		{"local domain replaced", `hostname.local`, "niac-go.local"},
		{"com domain replaced", `server.example.com`, "niac-go.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeLineDefaults(tt.line); !strings.Contains(got, tt.wantContains) {
				t.Errorf("sanitizeLine() = %q, want to contain %q", got, tt.wantContains)
			}
		})
	}
}

func TestSanitizeLineDomainEmpty(t *testing.T) {
	result := sanitizeLine("hostname.local some text", NewMapping(),
		Options{Domain: "", Location: "DC-WEST", Contact: "admin@niac-go.com", Community: "public"})
	if strings.Contains(result, "niac-go.local") {
		t.Error("Expected no domain replacement with empty domain")
	}
}

// TestSanitizeLineDoesNotCorruptStructuralOID is the core regression guard: a
// non-IP-indexed OID whose last four arcs happen to be <=255 must be returned
// byte-for-byte unchanged. The old greedy regex rewrote these into 10.100.x.x,
// relocating real MIB nodes and destroying the system group.
func TestSanitizeLineDoesNotCorruptStructuralOID(t *testing.T) {
	structural := []string{
		`.1.3.6.1.2.1.2.2.1.1.10 = INTEGER: 10`,           // ifIndex, trailing .2.1.1.10
		`.1.3.6.1.2.1.1.9.1.2.1 = OID: .1.3.6.1.6.3.1`,    // sysORID, trailing .9.1.2.1
		`.1.3.6.1.2.1.1.3.0 = Timeticks: (498147033) 57d`, // sysUpTime
		`.1.3.6.1.4.1.12356.101.1.6200.0 = INTEGER: 1`,    // enterprise scalar
	}
	for _, line := range structural {
		if got := sanitizeLineDefaults(line); got != line {
			t.Errorf("structural OID corrupted:\n in:  %q\n out: %q", line, got)
		}
	}
}

// TestSanitizeLineRewritesIPIndexedColumns guards the intended behavior: genuine
// IPv4-indexed table columns keep having their trailing IP index rewritten.
func TestSanitizeLineRewritesIPIndexedColumns(t *testing.T) {
	cases := []struct {
		name    string
		line    string
		wantNot string // original IP must be gone from the OID index
	}{
		{"ipAddrTable", `.1.3.6.1.2.1.4.20.1.1.203.0.113.5 = IpAddress: 203.0.113.5`, ".203.0.113.5 "},
		{
			"ipNetToMediaTable ifIndex+IP",
			`.1.3.6.1.2.1.4.22.1.3.5.203.0.113.9 = IpAddress: 203.0.113.9`,
			".203.0.113.9 ",
		},
		{"ipRouteTable dest IP", `.1.3.6.1.2.1.4.21.1.1.203.0.113.0 = IpAddress: 203.0.113.0`, ".203.0.113.0 "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeLineDefaults(tc.line)
			if strings.Contains(got, tc.wantNot) {
				t.Errorf("IP index not rewritten in %s: %q", tc.name, got)
			}
			if !strings.Contains(got, "= IpAddress: 10.") {
				t.Errorf("IpAddress value not rewritten to lab range in %s: %q", tc.name, got)
			}
		})
	}
}

// TestSanitizeLineIPIndexPreservesIfIndex ensures only the IP portion of a
// multi-part index is rewritten, not the leading ifIndex arc.
func TestSanitizeLineIPIndexPreservesIfIndex(t *testing.T) {
	got := sanitizeLineDefaults(`.1.3.6.1.2.1.4.22.1.3.5.203.0.113.9 = IpAddress: 203.0.113.9`)
	if !strings.HasPrefix(got, ".1.3.6.1.2.1.4.22.1.3.5.10.") {
		t.Errorf("ifIndex arc (.5) not preserved before rewritten IP: %q", got)
	}
}

func TestCollectAndApplyIdentitySubsScrubsEchoedHostname(t *testing.T) {
	lines := []string{
		`.1.3.6.1.2.1.1.5.0 = STRING: "COS_Lab_R00.fnet.eng"`,              // sysName
		`.1.3.6.1.4.1.9.9.23.1.2.1.1.6.1 = STRING: "COS_Lab_R00.fnet.eng"`, // CDP neighbor echo
	}
	subs := collectIdentitySubs(lines, NewMapping(), "netadmin@niac-go.com")
	got := applyIdentitySubs(lines[1], subs)
	if strings.Contains(got, "COS_Lab_R00.fnet.eng") {
		t.Errorf("echoed hostname not scrubbed: %q", got)
	}
	if !strings.Contains(got, "niac-") {
		t.Errorf("echo not replaced with sanitized hostname: %q", got)
	}
}

func TestCollectAndApplyIdentitySubsScrubsEchoedContact(t *testing.T) {
	lines := []string{
		`.1.3.6.1.2.1.1.4.0 = STRING: "netops@ucdenver.pvt"`,
		`.1.3.6.1.4.1.9.2.1.61.0 = STRING: "escalate: netops@ucdenver.pvt"`,
	}
	subs := collectIdentitySubs(lines, NewMapping(), "netadmin@niac-go.com")
	got := applyIdentitySubs(lines[1], subs)
	if strings.Contains(got, "netops@ucdenver.pvt") {
		t.Errorf("echoed contact not scrubbed: %q", got)
	}
}

// TestCollectIdentitySubsSkipsGenericNames guards against over-scrubbing: a plain
// dictionary sysName with no digit/./-/_ is not distinctive enough to blanket-
// replace across the file (it could be a substring of legitimate model strings).
func TestCollectIdentitySubsSkipsGenericNames(t *testing.T) {
	lines := []string{`.1.3.6.1.2.1.1.5.0 = STRING: "Switch"`}
	subs := collectIdentitySubs(lines, NewMapping(), "c@example.test")
	for _, s := range subs {
		if s.from == "Switch" {
			t.Errorf("generic name %q should not become a global substitution", s.from)
		}
	}
}

// TestApplyIdentitySubsLongestFirst ensures overlapping identifiers replace fully
// (an FQDN before its bare-host prefix).
func TestApplyIdentitySubsLongestFirst(t *testing.T) {
	subs := collectIdentitySubs([]string{
		`.1.3.6.1.2.1.1.5.0 = STRING: "sw1.corp.example"`,
	}, NewMapping(), "c@example.test")
	got := applyIdentitySubs(`x = STRING: "sw1.corp.example neighbor"`, subs)
	if strings.Contains(got, "sw1.corp.example") {
		t.Errorf("FQDN not fully scrubbed: %q", got)
	}
}

func TestSanitizeHostnameDeviceTypes(t *testing.T) {
	tests := []struct {
		name           string
		hostname       string
		wantDeviceType string
	}{
		{"switch keyword", "switch-floor1", "sw"},
		{"router keyword", "router-dc1", "rtr"},
		{"access point", "access-point-3", "ap"},
		{"server keyword", "server-web01", "srv"},
		{"firewall keyword", "firewall-edge", "fw"},
		{"generic device", "unknown-device-42", "dev"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeHostname(tt.hostname, NewMapping())
			if !strings.Contains(result, tt.wantDeviceType) {
				t.Errorf("sanitizeHostname(%q) = %q, want device type %q", tt.hostname, result, tt.wantDeviceType)
			}
		})
	}
}
