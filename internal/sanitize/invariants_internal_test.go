package sanitize

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
)

// walk builds a minimal numeric walk from the lines given.
func walk(lines ...string) []byte {
	return []byte(strings.Join(lines, "\n") + "\n")
}

// sanitized runs Content and returns the output as lines.
func sanitized(t *testing.T, content []byte) []string {
	t.Helper()

	out, _, err := Content(content, NewMapping(), DefaultOptions())
	if err != nil {
		t.Fatalf("Content: %v", err)
	}

	return strings.Split(strings.TrimRight(string(out), "\n"), "\n")
}

// valueOf returns the value half of the walk line whose OID matches prefix.
func valueOf(t *testing.T, lines []string, prefix string) string {
	t.Helper()

	for _, line := range lines {
		oid, value, found := strings.Cut(line, " = ")
		if found && strings.HasPrefix(oid, prefix) {
			return value
		}
	}
	t.Fatalf("no line with OID prefix %q in:\n%s", prefix, strings.Join(lines, "\n"))

	return ""
}

// A netmask is not an address. Hashing it like one produced
// "255.255.255.0 -> 10.3.199.4", so every sanitized device carried a nonsense
// subnet mask and no consumer could derive the prefix an interface was on.
func TestNetmasksSurviveSanitization(t *testing.T) {
	for _, mask := range []string{
		"255.255.255.0", "255.255.255.252", "255.255.0.0", "255.0.0.0", "255.255.255.128",
	} {
		t.Run(mask, func(t *testing.T) {
			lines := sanitized(t, walk(
				".1.3.6.1.2.1.4.20.1.3.172.28.161.1 = IpAddress: "+mask,
			))
			if got := valueOf(t, lines, ".1.3.6.1.2.1.4.20.1.3"); got != "IpAddress: "+mask {
				t.Errorf("mask became %q, want it untouched", got)
			}
		})
	}
}

// A device's interface address, its ARP neighbours and its routes have to keep
// agreeing after sanitization. Hashing each address independently scattered one
// real /24 across 24 different /24s, which is why the starter packs needed
// address tricks and why testers saw a device contradict itself.
func TestOneSubnetStaysOneSubnet(t *testing.T) {
	var lines []string
	for host := 1; host <= 11; host++ {
		lines = append(lines, fmt.Sprintf(
			".1.3.6.1.2.1.4.20.1.1.172.28.161.%d = IpAddress: 172.28.161.%d", host, host))
	}

	prefixes := make(map[string]struct{})
	hosts := make(map[string]struct{})
	for _, line := range sanitized(t, walk(lines...)) {
		_, value, _ := strings.Cut(line, " = ")
		address := strings.TrimPrefix(value, "IpAddress: ")
		parsed := net.ParseIP(address).To4()
		if parsed == nil {
			t.Fatalf("value %q is not an IPv4 address", value)
		}
		prefixes[fmt.Sprintf("%d.%d.%d", parsed[0], parsed[1], parsed[2])] = struct{}{}
		hosts[strconv.Itoa(int(parsed[3]))] = struct{}{}
	}

	if len(prefixes) != 1 {
		t.Errorf("11 addresses from one /24 landed in %d /24s, want 1: %v", len(prefixes), keys(prefixes))
	}
	if len(hosts) != 11 {
		t.Errorf("11 distinct hosts collapsed to %d, want 11", len(hosts))
	}
}

// The vendor domain in sysDescr is part of the fingerprint a tester classifies
// on. Rewriting every .com turned "www.cisco.com" into
// "www.cisco.niac-go.com" and damaged it. Only a domain the device's own
// identity names is customer data.
func TestVendorDomainsAreNotRewritten(t *testing.T) {
	lines := sanitized(t, walk(
		`.1.3.6.1.2.1.1.1.0 = STRING: "Cisco IOS Software, 1841 Software, `+
			`Technical Support: http://www.cisco.com/techsupport"`,
		`.1.3.6.1.2.1.1.5.0 = STRING: "rtr01.branch.acme-corp.com"`,
	))

	descr := valueOf(t, lines, ".1.3.6.1.2.1.1.1.0")
	if !strings.Contains(descr, "www.cisco.com") {
		t.Errorf("sysDescr lost the vendor domain: %s", descr)
	}
	if strings.Contains(strings.Join(lines, "\n"), "acme-corp.com") {
		t.Error("the customer's own domain survived sanitization")
	}
}

// Two devices in one pack sharing a sysName is a defect a tester sees
// immediately. The old suffix was hash mod 100 with no collision check, which
// produced 10 collisions in a 99-name corpus.
func TestSanitizedHostnamesAreUnique(t *testing.T) {
	mapping := NewMapping()
	seen := make(map[string]string)

	for i := range 200 {
		original := fmt.Sprintf("customer-device-%03d.internal.example.com", i)
		got := sanitizeHostname(original, mapping)
		if previous, clash := seen[got]; clash {
			t.Fatalf("%q and %q both sanitized to %q", previous, original, got)
		}
		seen[got] = original
	}
}

// "any name containing ap is an access point" made 58 of 99 real names fall to
// a wrong or generic type. The device's own sysDescr and sysServices say what
// it is.
func TestDeviceTypeComesFromTheDeviceNotTheName(t *testing.T) {
	tests := []struct {
		name  string
		descr string
		host  string
		want  string
	}{
		{
			name:  "router by sysDescr",
			descr: "Cisco IOS Software, 1841 Software, Router",
			host:  "chicago-hub",
			want:  "rtr",
		},
		{
			name:  "switch by sysDescr",
			descr: "Cisco IOS Software, C2960 Software, Switch",
			host:  "chicago-hub",
			want:  "sw",
		},
		{
			name:  "access point by sysDescr",
			descr: "Cisco Aironet 1140 Series Access Point",
			host:  "chicago-hub",
			want:  "ap",
		},
		{
			name:  "firewall by sysDescr",
			descr: "Cisco Adaptive Security Appliance Firewall Version 9",
			host:  "chicago-hub",
			want:  "fw",
		},
		{
			// The bug: "ap" appears inside "capitol", and nothing about this
			// device is an access point.
			name:  "ap inside a word is not an access point",
			descr: "Cisco IOS Software, C2960 Software, Switch",
			host:  "capitol-hill-1",
			want:  "sw",
		},
		{
			name:  "unknown stays generic",
			descr: "Some unfamiliar embedded agent",
			host:  "capitol-hill-1",
			want:  "dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := sanitized(t, walk(
				`.1.3.6.1.2.1.1.1.0 = STRING: "`+tt.descr+`"`,
				`.1.3.6.1.2.1.1.5.0 = STRING: "`+tt.host+`"`,
			))
			got := valueOf(t, lines, ".1.3.6.1.2.1.1.5.0")
			if !strings.Contains(got, "-"+tt.want+"-") {
				t.Errorf("sysName = %s, want a %q device", got, tt.want)
			}
		})
	}
}

// The community rule matched the word anywhere on the line, so any STRING value
// mentioning "community" was replaced -- including sysDescr prose and a
// vendor's own help text. The OID says whether a value is a community string.
func TestCommunityRuleKeysOnTheOIDNotTheText(t *testing.T) {
	lines := sanitized(t, walk(
		`.1.3.6.1.2.1.1.1.0 = STRING: "Agent supporting the community MIB"`,
		`.1.3.6.1.6.3.18.1.1.1.2.1 = STRING: "s3cr3t-r0"`,
	))

	if descr := valueOf(t, lines, ".1.3.6.1.2.1.1.1.0"); !strings.Contains(descr, "community MIB") {
		t.Errorf("sysDescr prose was replaced by the community rule: %s", descr)
	}
	if community := valueOf(t, lines, ".1.3.6.1.6.3.18.1.1.1.2"); strings.Contains(community, "s3cr3t") {
		t.Errorf("the real community string survived: %s", community)
	}
}

// Serial numbers identify a specific unit a customer owns, and were kept
// verbatim.
func TestSerialNumbersAreReplacedInPlace(t *testing.T) {
	const serial = "FTX1124W33P"

	lines := sanitized(t, walk(
		".1.3.6.1.2.1.47.1.1.1.1.11.1 = STRING: "+serial,
	))

	got := strings.TrimPrefix(valueOf(t, lines, ".1.3.6.1.2.1.47.1.1.1.1.11"), "STRING: ")
	got = strings.Trim(got, `"`)
	if got == serial {
		t.Fatal("the real serial number survived sanitization")
	}
	if len(got) != len(serial) {
		t.Errorf("serial %q is %d characters, want %d: the format is part of the fingerprint",
			got, len(got), len(serial))
	}
	for i := range serial {
		if isDigit(serial[i]) != isDigit(got[i]) || isUpper(serial[i]) != isUpper(got[i]) {
			t.Errorf("serial %q does not preserve the shape of %q at %d", got, serial, i)
		}
	}
}

// IPv6 was passed through untouched, so a walk from a dual-stack device leaked
// its real v6 addressing.
func TestIPv6AddressesAreMapped(t *testing.T) {
	lines := sanitized(t, walk(
		`.1.3.6.1.2.1.4.34.1.3.2.16.32.1.13.184.0.0.0.0.0.0.0.0.0.0.0.1 = STRING: "2001:470:1f0b:abcd::1"`,
	))

	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "2001:470:1f0b:abcd::1") {
		t.Errorf("the real IPv6 address survived: %s", joined)
	}
}

func keys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	return out
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }
func isUpper(b byte) bool { return b >= 'A' && b <= 'Z' }
