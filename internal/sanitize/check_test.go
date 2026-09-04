package sanitize_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/sanitize"
)

// A tcpConnTable index records who a device was actually talking to. The
// address sits in the middle of the index, between ports, and a dotted-quad
// regex over the line matches the OID's own first four arcs instead -- which
// is how the first version of this check passed a walk carrying two real
// peers.
func TestCheckFindsAddressesInsideAnOIDIndex(t *testing.T) {
	line := ".1.3.6.1.2.1.6.13.1.1.192.168.5.7.443.192.168.5.9.1024 = INTEGER: 5\n"

	findings := sanitize.Check([]byte(line))

	if len(findings) != 2 {
		t.Fatalf("findings = %d (%v), want both addresses in the index", len(findings), findings)
	}
	joined := findings[0].String() + findings[1].String()
	for _, want := range []string{"192.168.5.7", "192.168.5.9"} {
		if !strings.Contains(joined, want) {
			t.Errorf("findings %v do not name %s", findings, want)
		}
	}
}

// Free text in a value carries them too: a log line naming an "intruder IP" is
// still a real host.
func TestCheckFindsAddressesInFreeText(t *testing.T) {
	line := `.1.3.6.1.4.1.1991.1.1.2.6.2.1.4.11 = STRING: "SNMP: Auth. failure, intruder IP:  192.168.0.2."` + "\n"

	findings := sanitize.Check([]byte(line))

	if len(findings) != 1 {
		t.Fatalf("findings = %d (%v), want the address in the message", len(findings), findings)
	}
}

// The sanitizer maps into 10/8, so an address there has been through it.
func TestCheckAcceptsTheMappedRange(t *testing.T) {
	line := ".1.3.6.1.2.1.6.13.1.1.10.0.5.7.443.10.0.5.9.1024 = INTEGER: 5\n"

	if findings := sanitize.Check([]byte(line)); len(findings) != 0 {
		t.Fatalf("findings = %v, want none for addresses already in 10/8", findings)
	}
}

// Public addresses are NTP and DNS servers the sanitizer maps by rule; they
// are not the leak this gate is for.
func TestCheckIgnoresPublicAddresses(t *testing.T) {
	line := ".1.3.6.1.2.1.4.21.1.1.8.8.8.8 = IpAddress: 8.8.8.8\n"

	if findings := sanitize.Check([]byte(line)); len(findings) != 0 {
		t.Fatalf("findings = %v, want none for a public address", findings)
	}
}

func TestCheckFlagsARealContactAndLocation(t *testing.T) {
	content := `.1.3.6.1.2.1.1.4.0 = STRING: "jane.doe@customer.example"
.1.3.6.1.2.1.1.6.0 = STRING: "Rack 4, Customer DC, Leeds"
`

	findings := sanitize.Check([]byte(content))

	if len(findings) != 2 {
		t.Fatalf("findings = %d (%v), want the contact and the location", len(findings), findings)
	}
}

func TestCheckAcceptsThePlaceholdersTheSanitizerWrites(t *testing.T) {
	content := `.1.3.6.1.2.1.1.4.0 = STRING: "netadmin@niac-go.com"
.1.3.6.1.2.1.1.6.0 = STRING: "NiAC-Go - DC-WEST - Network Operations"
`

	if findings := sanitize.Check([]byte(content)); len(findings) != 0 {
		t.Fatalf("findings = %v, want none for the sanitizer's own output", findings)
	}
}

// The gate exists because these ship. If one ever fails, the walk is the bug.
func TestShippedStarterWalksAreSafeToShip(t *testing.T) {
	matches, err := filepath.Glob(filepath.Join("..", "library", "starter", "walks", "*.walk"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("no starter walks found; this test would pass by checking nothing")
	}

	for _, path := range matches {
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("read %s: %v", path, readErr)
		}
		if findings := sanitize.Check(content); len(findings) > 0 {
			t.Errorf("%s is not safe to ship: %d finding(s), first: %s",
				filepath.Base(path), len(findings), findings[0])
		}
	}
}

// The sanitizer must remove what the check reports, or the gate is a
// permanent red light rather than a guard.
func TestSanitizeRemovesWhatTheCheckReports(t *testing.T) {
	dirty := `.1.3.6.1.2.1.1.4.0 = STRING: "jane.doe@customer.example"
.1.3.6.1.2.1.6.13.1.1.192.168.5.7.443.192.168.5.9.1024 = INTEGER: 5
.1.3.6.1.4.1.1991.1.1.2.6.2.1.4.11 = STRING: "intruder IP:  172.31.4.9."
`
	if findings := sanitize.Check([]byte(dirty)); len(findings) == 0 {
		t.Fatal("the fixture is already clean; this test would prove nothing")
	}

	cleaned, _, err := sanitize.Content([]byte(dirty), nil, sanitize.DefaultOptions())
	if err != nil {
		t.Fatalf("Content: %v", err)
	}

	if findings := sanitize.Check(cleaned); len(findings) > 0 {
		t.Fatalf("sanitizing left %d finding(s), first: %s", len(findings), findings[0])
	}
}

// Sanitizing an OID must not change its shape: replacing four arcs with four
// arcs keeps the row addressable, and a dropped or added arc would corrupt
// the table.
func TestSanitizePreservesOIDArcCount(t *testing.T) {
	line := ".1.3.6.1.2.1.6.13.1.1.192.168.5.7.443.192.168.5.9.1024 = INTEGER: 5\n"

	cleaned, _, err := sanitize.Content([]byte(line), nil, sanitize.DefaultOptions())
	if err != nil {
		t.Fatalf("Content: %v", err)
	}

	before := strings.Count(strings.Split(line, " = ")[0], ".")
	after := strings.Count(strings.Split(string(cleaned), " = ")[0], ".")
	if before != after {
		t.Fatalf("OID arc count changed %d -> %d: %s", before, after, cleaned)
	}
}

// Re-sanitizing already-sanitized content must not renumber its hostname.
//
// The number is a hash of the input, so running the sanitizer over its own
// output hashed `niac-core-sw-96` and produced a different name. That is the
// real scenario: the shipped walks are already sanitized, and re-sanitizing
// one to fix something else rewrote a hostname that was fine. The fixture
// therefore starts from sanitized content, which is what an unsanitized
// starting point does not exercise.
func TestSanitizeDoesNotRenumberAnAlreadySanitizedHostname(t *testing.T) {
	walk := `.1.3.6.1.2.1.1.5.0 = STRING: niac-core-sw-96
.1.3.6.1.2.1.1.1.0 = STRING: "niac-core-sw-96 running IOS 15.2"
`

	cleaned, _, err := sanitize.Content([]byte(walk), nil, sanitize.DefaultOptions())
	if err != nil {
		t.Fatalf("Content: %v", err)
	}

	if !strings.Contains(string(cleaned), "niac-core-sw-96") {
		t.Fatalf("re-sanitizing renamed an already-sanitized host:\n%s", cleaned)
	}
}
