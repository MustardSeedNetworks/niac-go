package sanitize

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// defaultDeviceType is the device type abbreviation for generic/
// unrecognized devices.
const defaultDeviceType = "dev"

// sysGroupPrefix is the numeric OID root of the SNMPv2-MIB system group.
const sysGroupPrefix = ".1.3.6.1.2.1.1."

// minIdentityScrubLen is the shortest identity value eligible for global
// scrubbing. Shorter values risk matching legitimate substrings elsewhere
// in the walk.
const minIdentityScrubLen = 5

// Precompiled patterns for sanitizeLine and its helpers. Compiling once
// (rather than per line) matters: walks run to hundreds of thousands of
// lines each.
var (
	stringValueRe   = regexp.MustCompile(`= STRING:.*`)
	stringCaptureRe = regexp.MustCompile(`= STRING: (.+)`)
	dnsLocalRe      = regexp.MustCompile(`\.local\b`)
)

// isSystemScalar reports whether line is the given system-group scalar, in
// either numeric (.1.3.6.1.2.1.1.<arc>.0) or symbolic (MIB object name)
// walk format.
func isSystemScalar(line, arc, symbolic string) bool {
	return strings.HasPrefix(line, sysGroupPrefix+arc+".0 ") || strings.Contains(line, symbolic)
}

// isIdentityScalar reports whether line is one of the system-group
// identity scalars whose value the per-line rules already replace.
func isIdentityScalar(line string) bool {
	return isSystemScalar(line, "4", "sysContact") ||
		isSystemScalar(line, "5", "sysName") ||
		isSystemScalar(line, "6", "sysLocation")
}

// identitySub is a literal find/replace for an echoed identity string.
type identitySub struct {
	from string
	to   string
}

// collectIdentitySubs scans a walk for the sysName and sysContact scalar
// values and returns literal substitutions (original -> sanitized) for the
// distinctive ones. Real devices echo these strings in vendor OIDs,
// entPhysicalName, and CDP/LLDP neighbor tables, which the per-line scalar
// rules never reach. The substitutions are ordered longest-first so
// overlapping values replace fully.
func collectIdentitySubs(lines []string, mapping *Mapping, contact, deviceType string) []identitySub {
	seen := make(map[string]struct{})
	var subs []identitySub

	add := func(orig, repl string) {
		if !isDistinctiveIdentifier(orig) || orig == repl {
			return
		}
		if _, dup := seen[orig]; dup {
			return
		}
		seen[orig] = struct{}{}
		subs = append(subs, identitySub{from: orig, to: repl})
	}

	for _, line := range lines {
		switch {
		case isSystemScalar(line, "5", "sysName"):
			if v, ok := scalarStringValue(line); ok {
				add(v, sanitizeHostnameAs(v, deviceType, mapping))
			}
		case isSystemScalar(line, "4", "sysContact"):
			if v, ok := scalarStringValue(line); ok {
				add(v, contact)
			}
		}
	}

	sort.SliceStable(subs, func(i, j int) bool { return len(subs[i].from) > len(subs[j].from) })
	return subs
}

// applyIdentitySubs replaces every occurrence of each collected identity string.
func applyIdentitySubs(line string, subs []identitySub) string {
	for _, sub := range subs {
		if strings.Contains(line, sub.from) {
			line = strings.ReplaceAll(line, sub.from, sub.to)
		}
	}
	return line
}

// scalarStringValue extracts and unquotes the STRING value of a walk line.
func scalarStringValue(line string) (string, bool) {
	matches := stringCaptureRe.FindStringSubmatch(line)
	if len(matches) <= 1 {
		return "", false
	}
	value := strings.TrimSpace(matches[1])
	value = strings.TrimSuffix(strings.TrimPrefix(value, `"`), `"`)
	return value, value != ""
}

// isDistinctiveIdentifier reports whether s is specific enough to
// blanket-replace: long enough and containing a character typical of a
// hostname or contact (digit, dot, hyphen, underscore, or @), which
// excludes plain dictionary words.
func isDistinctiveIdentifier(s string) bool {
	if len(s) < minIdentityScrubLen {
		return false
	}
	return strings.ContainsAny(s, "0123456789.-_@")
}

// sanitizedHostnameRe matches a name this function has already produced.
var sanitizedHostnameRe = regexp.MustCompile(`^niac-core-(?:sw|rtr|ap|srv|fw|dev)-\d{2,}$`)

// sanitizeHostname maps hostname to a "niac-core-{type}-{NN}" branded name.
//
// deviceType comes from the device's own sysDescr and sysServices, not from the
// name. Inferring it from substrings of the hostname read "ap" inside
// "capitol-hill-1" and called a switch an access point; 58 of the 99 names in
// the corpus fell to a wrong or generic type that way.
//
// The number is allocated, not hashed. A bare hash mod 100 gave 10 collisions
// in that same 99-name corpus, which means two devices in one pack shared a
// sysName -- something a tester sees immediately. The hash still chooses the
// starting point, so names stay stable; only a colliding one moves.
func sanitizeHostname(hostname string, mapping *Mapping) string {
	return sanitizeHostnameAs(hostname, defaultDeviceType, mapping)
}

// sanitizeHostnameAs is sanitizeHostname with the device type supplied by the
// caller, which has read the walk's sysDescr and sysServices.
func sanitizeHostnameAs(hostname, deviceType string, mapping *Mapping) string {
	// An already-sanitized name is returned unchanged, so sanitizing twice
	// gives the same answer as sanitizing once. Re-running used to renumber a
	// clean walk, so no caller could use "nothing changed" as a check.
	if sanitizedHostnameRe.MatchString(hostname) {
		return hostname
	}

	mapping.mu.RLock()
	if sanitized, exists := mapping.Hostnames[hostname]; exists {
		mapping.mu.RUnlock()

		return sanitized
	}
	mapping.mu.RUnlock()

	mapping.mu.Lock()
	defer mapping.mu.Unlock()
	if existing, exists := mapping.Hostnames[hostname]; exists {
		return existing
	}
	if mapping.hostnameSlots == nil {
		// A mapping decoded from an on-disk file: the slot set is not
		// persisted, so rebuild it from the names already handed out.
		mapping.hostnameSlots = make(map[string]struct{}, len(mapping.Hostnames))
		for _, taken := range mapping.Hostnames {
			mapping.hostnameSlots[taken] = struct{}{}
		}
	}

	hash := sha256.Sum256([]byte(hostname))
	start := int(binary.BigEndian.Uint32(hash[:4]))

	for attempt := range hostnameSlotCount {
		number := (start + attempt) % hostnameSlotCount
		candidate := fmt.Sprintf("niac-core-%s-%02d", deviceType, number)
		if _, taken := mapping.hostnameSlots[candidate]; taken {
			continue
		}
		mapping.hostnameSlots[candidate] = struct{}{}
		mapping.Hostnames[hostname] = candidate

		return candidate
	}

	// Every slot for this type is spoken for. Returning a duplicate would put
	// two devices in a pack under one sysName, which is the defect this
	// allocation exists to prevent.
	panic("sanitize: no free hostname slot for device type " + deviceType)
}

// digitCount and letterCount size the character classes a format-preserving
// serial draws from.
const (
	digitCount  = 10
	letterCount = 26
)

// hostnameSlotCount bounds the numeric suffix. Wide enough that a corpus of a
// few hundred devices of one type never exhausts a class, and still short
// enough to read.
const hostnameSlotCount = 1000

// deviceTypeFromWalk reads the device's own description of itself.
//
// sysDescr names the product ("...Aironet 1140 Series Access Point"), and
// sysServices is the standard bitmask a device sets for the layers it operates
// at. Both are statements by the device; a hostname is a statement by whoever
// named it, and the corpus shows they are frequently unrelated.
func deviceTypeFromWalk(lines []string) string {
	var descr string
	services := -1
	for _, line := range lines {
		switch {
		case isSystemScalar(line, "1", "sysDescr"):
			if v, ok := scalarStringValue(line); ok {
				descr = strings.ToLower(v)
			}
		case isSystemScalar(line, "7", "sysServices"):
			services = integerValue(line)
		}
	}

	if kind := typeFromDescription(descr); kind != "" {
		return kind
	}
	if kind := typeFromServices(services); kind != "" {
		return kind
	}

	return defaultDeviceType
}

// descriptionKeywords maps a phrase a vendor puts in sysDescr to a device type.
// Ordered most specific first: an "Access Point" is also a bridge, and a
// "Wireless LAN Controller" mentions both.
//
//nolint:gochecknoglobals // an ordered lookup table, read-only after init.
var descriptionKeywords = []struct {
	phrase string
	kind   string
}{
	{"access point", "ap"},
	{"wireless lan controller", "ap"},
	{"aironet", "ap"},
	{"firewall", "fw"},
	{"security appliance", "fw"},
	{"router", "rtr"},
	{"switch", "sw"},
	{"server", "srv"},
}

// typeFromDescription matches whole phrases, never bare substrings.
func typeFromDescription(descr string) string {
	for _, candidate := range descriptionKeywords {
		if strings.Contains(descr, candidate.phrase) {
			return candidate.kind
		}
	}

	return ""
}

// sysServices bits, per SNMPv2-MIB: layer N sets bit 2^(N-1).
const (
	serviceDatalink    = 0x02 // layer 2, a bridge or switch
	serviceInternet    = 0x04 // layer 3, forwards IP
	serviceApplication = 0x40 // layer 7, an end system offering services
)

// typeFromServices reads the standard bitmask, which is a weaker signal than
// sysDescr but is present on devices whose description says nothing useful.
func typeFromServices(services int) string {
	if services < 0 {
		return ""
	}
	switch {
	case services&serviceApplication != 0 && services&serviceInternet == 0:
		return "srv"
	case services&serviceInternet != 0:
		return "rtr"
	case services&serviceDatalink != 0:
		return "sw"
	default:
		return ""
	}
}

// integerValue reads an INTEGER walk value, or -1 when the line has none.
func integerValue(line string) int {
	_, value, found := strings.Cut(line, " = ")
	if !found {
		return -1
	}
	value = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(value), "INTEGER:"))
	parsed, err := strconv.Atoi(strings.Fields(value + " ")[0])
	if err != nil {
		return -1
	}

	return parsed
}

// collectIdentityDomains returns the DNS domains this device's own identity
// names -- the suffix of an FQDN in sysName, and the domain of an email address
// in sysContact.
//
// Those are the customer's. A vendor support URL in sysDescr is not, and
// rewriting it by TLD damaged the product fingerprint without protecting
// anyone. The device's replacement domain is skipped so re-running is a no-op.
func collectIdentityDomains(lines []string, opts Options) []string {
	seen := make(map[string]struct{})
	var domains []string

	for _, line := range lines {
		for _, domain := range identityDomainsOnLine(line) {
			domain = strings.ToLower(strings.Trim(domain, ". "))
			if domain == "" || domain == opts.Domain || !strings.Contains(domain, ".") {
				continue
			}
			if _, dup := seen[domain]; dup {
				continue
			}
			seen[domain] = struct{}{}
			domains = append(domains, domain)
		}
	}

	// Longest first, so a subdomain is replaced before its parent swallows it.
	sort.SliceStable(domains, func(i, j int) bool { return len(domains[i]) > len(domains[j]) })

	return domains
}

// identityDomainsOnLine returns the domains one identity scalar names: the
// suffix of an FQDN in sysName, or the domain of each email in sysContact.
func identityDomainsOnLine(line string) []string {
	switch {
	case isSystemScalar(line, "5", "sysName"):
		value, ok := scalarStringValue(line)
		if !ok {
			return nil
		}
		if _, domain, found := strings.Cut(value, "."); found {
			return []string{domain}
		}
	case isSystemScalar(line, "4", "sysContact"):
		value, ok := scalarStringValue(line)
		if !ok {
			return nil
		}
		var domains []string
		for field := range strings.FieldsSeq(value) {
			if _, domain, found := strings.Cut(field, "@"); found {
				domains = append(domains, domain)
			}
		}

		return domains
	}

	return nil
}

// rewriteIdentityDomains replaces each collected domain with the replacement.
func rewriteIdentityDomains(line string, domains []string, replacement string) string {
	for _, domain := range domains {
		if idx := strings.Index(strings.ToLower(line), domain); idx >= 0 {
			line = line[:idx] + replacement + line[idx+len(domain):]
		}
	}

	return line
}

// communityOIDs are the objects whose value is an SNMP community string.
// Keyed on the OID because the value is just text: "public" and a real secret
// look alike, and prose mentioning the word is not a secret at all.
//
//nolint:gochecknoglobals // a lookup table, read-only after init.
var communityOIDs = []string{
	".1.3.6.1.6.3.18.1.1.1.2", // snmpCommunityName (SNMP-COMMUNITY-MIB)
	".1.3.6.1.6.3.18.1.1.1.3", // snmpCommunitySecurityName
	".1.3.6.1.6.3.18.1.1.1.6", // snmpCommunityTransportTag
}

// communitySymbolicRe matches the symbolic spelling of a community object in
// the OID field of a walk written with MIB names.
var communitySymbolicRe = regexp.MustCompile(`(?i)^[^ ]*\bsnmpCommunity(?:Name|SecurityName|TransportTag)?\b`)

// isCommunityOID reports whether the line's OID names a community string.
func isCommunityOID(line string) bool {
	oid, _, found := strings.Cut(line, " = ")
	if !found {
		return false
	}
	for _, prefix := range communityOIDs {
		if strings.HasPrefix(oid, prefix+".") || oid == prefix {
			return true
		}
	}

	return communitySymbolicRe.MatchString(oid)
}

// serialOIDs are the objects carrying a hardware serial number.
//
//nolint:gochecknoglobals // a lookup table, read-only after init.
var serialOIDs = []string{
	".1.3.6.1.2.1.47.1.1.1.1.11", // entPhysicalSerialNum (ENTITY-MIB)
}

// serialSymbolicRe matches the symbolic spelling of a serial-number object.
var serialSymbolicRe = regexp.MustCompile(`(?i)^[^ ]*\b(?:entPhysicalSerialNum|serialNumber|SerialNum)\b`)

// rewriteSerial replaces a serial number with a format-preserving stand-in:
// same length, digits stay digits and letters stay letters, so anything that
// validates or parses the format still works while the unit is unidentifiable.
func rewriteSerial(line string) string {
	oid, value, found := strings.Cut(line, " = ")
	if !found || !isSerialOID(oid) {
		return line
	}
	text, ok := scalarStringValue(line)
	if !ok {
		return line
	}

	quoted := strings.Contains(value, `"`)
	replacement := fakeSerial(text)
	if quoted {
		replacement = `"` + replacement + `"`
	}

	return oid + " = STRING: " + replacement
}

// isSerialOID reports whether oid names a hardware serial number.
func isSerialOID(oid string) bool {
	for _, prefix := range serialOIDs {
		if strings.HasPrefix(oid, prefix+".") || oid == prefix {
			return true
		}
	}

	return serialSymbolicRe.MatchString(oid)
}

// fakeSerial derives a stand-in of the same shape from a hash of the original,
// so the same unit always reads the same and two units never collide by luck.
func fakeSerial(serial string) string {
	hash := sha256.Sum256([]byte(serial))
	out := []byte(serial)
	for i := range out {
		b := hash[i%len(hash)]
		switch {
		case out[i] >= '0' && out[i] <= '9':
			out[i] = '0' + b%digitCount
		case out[i] >= 'A' && out[i] <= 'Z':
			out[i] = 'A' + b%letterCount
		case out[i] >= 'a' && out[i] <= 'z':
			out[i] = 'a' + b%letterCount
		}
	}

	return string(out)
}
