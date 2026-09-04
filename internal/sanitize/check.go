package sanitize

import (
	"bufio"
	"bytes"
	"fmt"
	"net/netip"
	"regexp"
	"strings"
)

// Checking a walk for content that should never ship.
//
// Sanitizing and checking are not the same operation and cannot be expressed
// as each other. Re-sanitizing is not a check, because hostname mapping
// allocates a fresh number each run: a walk that is already clean comes back
// with `niac-core-sw-96` renumbered to `niac-core-sw-05`, so "sanitizing
// changed nothing" is never true and would fail every file.
//
// So a check tests properties of the content instead: whether the identity
// scalars carry placeholders, and whether any address sits outside the space
// the sanitizer maps into.

// Finding is one reason a walk is not safe to ship.
type Finding struct {
	Line   int
	Reason string
	Detail string
}

func (f Finding) String() string {
	return fmt.Sprintf("line %d: %s (%s)", f.Line, f.Reason, f.Detail)
}

// isPlaceholderIdentity reports whether value is one the sanitizer writes, or
// empty. Anything else in sysContact or sysLocation is a real-world value that
// survived, which is what leaks an operator's name or a site address.
func isPlaceholderIdentity(value string) bool {
	switch value {
	case "",
		"netadmin@niac-go.com",
		"Network Administrator",
		"NiAC-Go Simulated Device",
		"NiAC-Go - DC-WEST - Network Operations":
		return true
	default:
		return false
	}
}

var (
	sysContactRe  = regexp.MustCompile(`\.1\.3\.6\.1\.2\.1\.1\.4\.0|sysContact`)
	sysLocationRe = regexp.MustCompile(`\.1\.3\.6\.1\.2\.1\.1\.6\.0|sysLocation`)
	scalarValueRe = regexp.MustCompile(`=\s*STRING:\s*(.*)$`)
)

// mappedRange is the space the sanitizer rewrites addresses into. An address
// outside it either never went through the sanitizer or the sanitizer missed
// it; both mean the walk may still carry a real topology.
func mappedRange() netip.Prefix { return netip.MustParsePrefix("10.0.0.0/8") }

// Check reports why a walk is not safe to ship, or nil when it is.
//
// It deliberately does not look at MAC addresses, serial numbers or models:
// those are kept on purpose, because a simulated device that answers with no
// hardware identity is not useful to the tools this content exists to feed.
func Check(content []byte) []Finding {
	var findings []Finding

	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), maxWalkLineBytes)

	for line := 1; scanner.Scan(); line++ {
		text := scanner.Text()
		findings = append(findings, checkIdentity(text, line)...)
		findings = append(findings, checkAddresses(text, line)...)
	}

	return findings
}

func checkIdentity(line string, number int) []Finding {
	var which string
	switch {
	case sysContactRe.MatchString(line):
		which = "sysContact"
	case sysLocationRe.MatchString(line):
		which = "sysLocation"
	default:
		return nil
	}

	match := scalarValueRe.FindStringSubmatch(line)
	if match == nil {
		return nil
	}
	value := strings.TrimSpace(strings.Trim(match[1], `"`))
	if isPlaceholderIdentity(value) {
		return nil
	}

	return []Finding{{
		Line:   number,
		Reason: which + " is not a placeholder",
		Detail: value,
	}}
}

func checkAddresses(line string, number int) []Finding {
	var findings []Finding
	for _, addr := range privateAddressesOutsideMappedRange(line) {
		findings = append(findings, Finding{
			Line:   number,
			Reason: "private address outside the sanitized range",
			Detail: addr,
		})
	}

	return findings
}

// privateAddressesOutsideMappedRange finds the addresses a walk should not
// carry, on both sides of a line.
//
// The OID side is scanned arc by arc rather than with a regex over the text.
// A dotted-quad pattern matches the first four arcs of the OID itself
// (".1.3.6.1") and then tokenises from there, so it never lines up with the
// address further in -- which is how the first version of this check passed a
// walk whose tcpConnTable index carried two real peers. The value side is
// free text, so a maximal digits-and-dots run validated by net.ParseIP is
// exact there.
func privateAddressesOutsideMappedRange(line string) []string {
	var found []string

	oid, value, hasValue := strings.Cut(line, " = ")
	arcs := strings.Split(oid, ".")
	for i := 0; i+4 <= len(arcs); {
		quad := strings.Join(arcs[i:i+4], ".")
		if !isPrivateOutsideMapped(quad) {
			i++

			continue
		}
		found = append(found, quad)
		i += 4
	}

	if !hasValue {
		return found
	}
	for _, run := range numericRunRe.FindAllString(value, -1) {
		if isPrivateOutsideMapped(run) {
			found = append(found, run)
		}
	}

	return found
}

// numericRunRe matches a maximal run of digits and dots, validated as an
// address afterwards. See valueQuadRe in ip.go for why the run, rather than a
// bounded quad, is the right unit.
var numericRunRe = regexp.MustCompile(`[0-9][0-9.]*[0-9]`)

func isPrivateOutsideMapped(candidate string) bool {
	addr, err := netip.ParseAddr(candidate)
	if err != nil || !addr.Is4() {
		return false
	}

	return addr.IsPrivate() && !mappedRange().Contains(addr)
}
