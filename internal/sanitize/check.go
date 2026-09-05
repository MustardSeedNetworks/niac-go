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
// It deliberately does not look at MAC addresses or models: those are kept on
// purpose, because a simulated device that answers with no hardware identity is
// not useful to the tools this content exists to feed.
//
// Serial numbers are replaced by the sanitizer, but no file-level check can
// confirm it: the stand-in is format-preserving by design, so a replaced serial
// and a real one are the same shape, and the checker does not know the
// customer's. That invariant is held by the round-trip test instead, which has
// the input to compare against.
func Check(content []byte) []Finding {
	var findings []Finding

	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), maxWalkLineBytes)

	// Invariants that only make sense across the whole file: a mask has to
	// stay a mask, one real subnet has to remain one subnet, and no two devices
	// may end up under one name.
	names := make(map[string]int)

	for line := 1; scanner.Scan(); line++ {
		text := scanner.Text()
		findings = append(findings, checkIdentity(text, line)...)
		findings = append(findings, checkAddresses(text, line)...)
		findings = append(findings, checkMask(text, line)...)
		findings = append(findings, checkVendorDomain(text, line)...)
		findings = append(findings, recordSysName(text, line, names)...)
	}

	return findings
}

// checkMask catches a subnet mask that was hashed like an address. The
// sanitizer used to do exactly that, leaving every device with a mask such as
// 10.3.199.4 and no way to derive the prefix an interface sat on.
func checkMask(line string, number int) []Finding {
	oid, value, found := strings.Cut(line, " = ")
	if !found || !isMaskColumn(oid) {
		return nil
	}
	address, ok := strings.CutPrefix(strings.TrimSpace(value), "IpAddress: ")
	if !ok {
		return nil
	}
	address = strings.TrimSpace(address)
	addr, err := netip.ParseAddr(address)
	if err != nil || !addr.Is4() {
		return nil
	}
	if isContiguousMask(addr) {
		return nil
	}

	return []Finding{{
		Line:   number,
		Reason: "subnet mask column does not hold a mask",
		Detail: address,
	}}
}

// maskColumns are the standard columns whose value is a subnet mask.
//
//nolint:gochecknoglobals // a lookup table, read-only after init.
var maskColumns = []string{
	".1.3.6.1.2.1.4.20.1.3",  // ipAdEntNetMask
	".1.3.6.1.2.1.4.21.1.11", // ipRouteMask
}

func isMaskColumn(oid string) bool {
	for _, prefix := range maskColumns {
		if strings.HasPrefix(oid, prefix+".") {
			return true
		}
	}

	return strings.Contains(oid, "NetMask") || strings.Contains(oid, "RouteMask")
}

// isContiguousMask reports whether addr is a run of ones followed by zeros.
func isContiguousMask(addr netip.Addr) bool {
	bytes4 := addr.As4()
	value := uint32(bytes4[0])<<24 | uint32(bytes4[1])<<16 | uint32(bytes4[2])<<8 | uint32(bytes4[3])
	inverted := ^value

	return inverted&(inverted+1) == 0
}

// vendorDomains are product fingerprints a tester classifies on. A sanitizer
// that rewrites them by TLD damages the walk without protecting anyone, and
// "www.cisco.niac-go.com" is the shape that leaves behind.
//
//nolint:gochecknoglobals // a lookup table, read-only after init.
var vendorDomains = []string{
	"cisco", "juniper", "arista", "aruba", "hpe", "hp", "dell",
	"extreme", "brocade", "netgear", "ubnt", "ui", "mikrotik", "fortinet",
	"paloaltonetworks", "alcatel-lucent", "nokia", "huawei",
}

// mangledVendorRe matches a vendor domain with the replacement domain spliced
// into it, which is what the TLD rule produced.
var mangledVendorRe = regexp.MustCompile(`(?i)\b([a-z0-9-]+)\.niac-go\.com\b`)

func checkVendorDomain(line string, number int) []Finding {
	var findings []Finding
	for _, match := range mangledVendorRe.FindAllStringSubmatch(line, -1) {
		label := strings.ToLower(match[1])
		for _, vendor := range vendorDomains {
			if label != vendor {
				continue
			}
			findings = append(findings, Finding{
				Line:   number,
				Reason: "vendor domain was rewritten, damaging the product fingerprint",
				Detail: match[0],
			})
		}
	}

	return findings
}

// recordSysName reports a sanitized hostname used by more than one device.
func recordSysName(line string, number int, names map[string]int) []Finding {
	if !isSystemScalar(line, "5", "sysName") {
		return nil
	}
	value, ok := scalarStringValue(line)
	if !ok {
		return nil
	}
	if first, seen := names[value]; seen {
		return []Finding{{
			Line:   number,
			Reason: "sysName is already used by another device in this file",
			Detail: fmt.Sprintf("%s, first seen on line %d", value, first),
		}}
	}
	names[value] = number

	return nil
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
