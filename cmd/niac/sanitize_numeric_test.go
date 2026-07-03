package main

import (
	"strings"
	"testing"
)

// Real device SNMP walks in the corpus are numeric (snmpwalk -On), not symbolic
// MIB names. These tests lock the sanitizer's behavior on numeric input:
//   - system-group identity scalars must be scrubbed by numeric OID, and
//   - the IP-in-OID rewrite must only touch genuine IP-indexed table columns,
//     never the trailing arcs of an arbitrary structural OID.

func sanitizeLineDefaults(line string) string {
	return sanitizeLine(line, newSanitizationMapping(), "niac-go.com", "DC-WEST", "netadmin@niac-go.com", "public")
}

func TestSanitizeLineNumericSystemContact(t *testing.T) {
	line := `.1.3.6.1.2.1.1.4.0 = STRING: "Operational Services"`
	got := sanitizeLineDefaults(line)
	if strings.Contains(got, "Operational Services") {
		t.Errorf("numeric sysContact not scrubbed: %q", got)
	}
	if !strings.Contains(got, "netadmin@niac-go.com") {
		t.Errorf("numeric sysContact not replaced with contact: %q", got)
	}
}

func TestSanitizeLineNumericSystemLocation(t *testing.T) {
	line := `.1.3.6.1.2.1.1.6.0 = STRING: "GS2 - London"`
	got := sanitizeLineDefaults(line)
	if strings.Contains(got, "London") {
		t.Errorf("numeric sysLocation not scrubbed: %q", got)
	}
	if !strings.Contains(got, "NiAC-Go") {
		t.Errorf("numeric sysLocation not branded: %q", got)
	}
}

func TestSanitizeLineNumericSystemName(t *testing.T) {
	line := `.1.3.6.1.2.1.1.5.0 = STRING: "GS2620b01"`
	got := sanitizeLineDefaults(line)
	if strings.Contains(got, "GS2620b01") {
		t.Errorf("numeric sysName not scrubbed: %q", got)
	}
	if !strings.Contains(got, "niac-") {
		t.Errorf("numeric sysName not replaced with branded hostname: %q", got)
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
		{
			name:    "ipAddrTable",
			line:    `.1.3.6.1.2.1.4.20.1.1.203.0.113.5 = IpAddress: 203.0.113.5`,
			wantNot: ".203.0.113.5 ",
		},
		{
			name:    "ipNetToMediaTable ifIndex+IP",
			line:    `.1.3.6.1.2.1.4.22.1.3.5.203.0.113.9 = IpAddress: 203.0.113.9`,
			wantNot: ".203.0.113.9 ",
		},
		{
			name:    "ipRouteTable dest IP",
			line:    `.1.3.6.1.2.1.4.21.1.1.203.0.113.0 = IpAddress: 203.0.113.0`,
			wantNot: ".203.0.113.0 ",
		},
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
	line := `.1.3.6.1.2.1.4.22.1.3.5.203.0.113.9 = IpAddress: 203.0.113.9`
	got := sanitizeLineDefaults(line)
	if !strings.HasPrefix(got, ".1.3.6.1.2.1.4.22.1.3.5.10.") {
		t.Errorf("ifIndex arc (.5) not preserved before rewritten IP: %q", got)
	}
}

// Real device hostnames/contacts are echoed outside the system group (vendor
// OIDs, entPhysicalName, CDP/LLDP neighbor tables), where the per-line scalar
// rules never reach. collectIdentitySubs + applyIdentitySubs form a second pass
// that scrubs those echoes globally.

func TestCollectAndApplyIdentitySubsScrubsEchoedHostname(t *testing.T) {
	lines := []string{
		`.1.3.6.1.2.1.1.5.0 = STRING: "COS_Lab_R00.fnet.eng"`,              // sysName
		`.1.3.6.1.4.1.9.9.23.1.2.1.1.6.1 = STRING: "COS_Lab_R00.fnet.eng"`, // CDP neighbor echo
	}
	subs := collectIdentitySubs(lines, newSanitizationMapping(), "netadmin@niac-go.com")
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
	subs := collectIdentitySubs(lines, newSanitizationMapping(), "netadmin@niac-go.com")
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
	subs := collectIdentitySubs(lines, newSanitizationMapping(), "c@example.test")
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
	}, newSanitizationMapping(), "c@example.test")
	got := applyIdentitySubs(`x = STRING: "sw1.corp.example neighbor"`, subs)
	if strings.Contains(got, "sw1.corp.example") {
		t.Errorf("FQDN not fully scrubbed: %q", got)
	}
}
