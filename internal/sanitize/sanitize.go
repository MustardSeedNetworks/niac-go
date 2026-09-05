// Package sanitize scrubs real network data out of captured SNMP walks so
// they are safe to share and to seed the demo replay corpus.
//
// Two properties are held at once. Nothing identifying survives: addresses,
// hostnames, contacts, community strings and serial numbers are all replaced
// deterministically, so the same input always produces the same output. And
// the walk stays internally consistent and still fingerprints as the product
// it came from: a subnet is mapped as a subnet rather than address by address,
// a serial keeps its shape, a mask stays a mask, and the vendor strings a
// tester classifies on — models, MAC OUIs, support URLs — are untouched.
//
// Both halves were once wrong in the same file. Masks were hashed like
// addresses, one /24 was scattered across 24, and "www.cisco.com" became
// "www.cisco.niac-go.com", while serial numbers and IPv6 shipped verbatim.
//
// The package is pure: Content takes and returns []byte and never
// touches the filesystem. cmd/niac's `sanitize` command and internal/api's
// library walk-sanitize route are both thin I/O wrappers around it.
package sanitize

import (
	"bufio"
	"bytes"
	"fmt"
	"sync"
)

// Options controls how identity fields are rewritten. Zero value uses no
// substitution text for Domain/Contact/Community — callers should use
// DefaultOptions() unless a specific override is required.
type Options struct {
	// Domain replaces .com/.net/.org TLDs (e.g. "niac-go.com"). Empty
	// disables DNS domain rewriting entirely.
	Domain string
	// Location feeds sysLocation: "NiAC-Go - {Location} - Network Operations".
	Location string
	// Contact replaces sysContact and any echoed contact string.
	Contact string
	// Community replaces SNMP community strings.
	Community string
}

// DefaultOptions returns the CLI's historical defaults, kept in one place
// so cmd/niac's flag defaults and internal/api's route handler stay in sync.
func DefaultOptions() Options {
	return Options{
		Domain:    "niac-go.com",
		Location:  "DC-WEST",
		Contact:   "netadmin@niac-go.com",
		Community: "public",
	}
}

// Stats reports the transformations a single Content call newly
// added to the mapping (not the mapping's running cumulative total).
type Stats struct {
	IPsTransformed       int
	HostnamesTransformed int
}

// MappingStatistics is the cumulative, persisted counterpart to Stats.
// FilesProcessed is incremented by batch callers (cmd/niac) once per file;
// Content only maintains the transformed counts.
type MappingStatistics struct {
	FilesProcessed       int `json:"files_processed"`
	IPsTransformed       int `json:"ips_transformed"`
	HostnamesTransformed int `json:"hostnames_transformed"`
}

// Mapping tracks IP and hostname transformations across one or more
// Content calls, so the same input value maps to the same output
// value everywhere it appears — within a single walk and across a batch.
// It is also the JSON shape persisted by cmd/niac's --mapping-file flag;
// field names and tags are load-bearing for that on-disk format.
type Mapping struct {
	IPMappings map[string]string `json:"ip_mappings"`
	// Prefixes maps a real /24 to the sanitized /24 that stands in for it, so
	// every address on one network keeps landing on one network. Absent from
	// files written before prefix-preserving mapping; callers must tolerate nil.
	Prefixes  map[string]string `json:"prefixes"`
	Hostnames map[string]string `json:"hostnames"`
	// hostnameSlots records which sanitized hostnames are already handed out,
	// so two devices in one pack cannot end up sharing a sysName.
	hostnameSlots map[string]struct{}
	Statistics    MappingStatistics `json:"statistics"`

	mu sync.RWMutex
}

// NewMapping returns an empty, ready-to-use Mapping.
func NewMapping() *Mapping {
	return &Mapping{
		IPMappings:    make(map[string]string),
		Prefixes:      make(map[string]string),
		Hostnames:     make(map[string]string),
		hostnameSlots: make(map[string]struct{}),
	}
}

// counts returns the current mapping sizes, used to compute the delta a
// Content call newly contributes.
func (m *Mapping) counts() (int, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.IPMappings), len(m.Hostnames)
}

// recordDelta updates the cumulative Statistics from the mapping sizes
// observed before a Content call and returns that call's Stats.
func (m *Mapping) recordDelta(initialIPs, initialHostnames int) Stats {
	m.mu.Lock()
	defer m.mu.Unlock()
	delta := Stats{
		IPsTransformed:       len(m.IPMappings) - initialIPs,
		HostnamesTransformed: len(m.Hostnames) - initialHostnames,
	}
	m.Statistics.IPsTransformed += delta.IPsTransformed
	m.Statistics.HostnamesTransformed += delta.HostnamesTransformed
	return delta
}

// maxWalkLineBytes bounds a single scanned walk line. Hex-string values can
// be large, so this is well above bufio's 64KiB default while still
// capping memory.
const maxWalkLineBytes = 8 << 20 // 8 MiB

// Content scrubs content (a full walk file's bytes) in place and
// returns the sanitized bytes plus this call's transformation counts.
// mapping accumulates transformations across calls (deterministic even
// when nil — a fresh Mapping is created internally); pass the same
// *Mapping across a batch to persist it, or nil for a one-off call.
//
// This is a two-pass operation: pass one (collectIdentitySubs) collects
// identity strings (hostname, contact) that vendor/entity OIDs echo
// outside the system group; pass two applies the per-line rules and
// scrubs those echoes globally.
func Content(content []byte, mapping *Mapping, opts Options) ([]byte, Stats, error) {
	if mapping == nil {
		mapping = NewMapping()
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), maxWalkLineBytes)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, Stats{}, fmt.Errorf("scan content: %w", err)
	}

	initialIPs, initialHostnames := mapping.counts()
	// The device says what it is (sysDescr, sysServices); resolve that once for
	// the whole walk, before any name is rewritten.
	deviceType := deviceTypeFromWalk(lines)
	// Only the domains this device's own identity names are customer data.
	domains := collectIdentityDomains(lines, opts)
	subs := collectIdentitySubs(lines, mapping, opts.Contact, deviceType)

	var out bytes.Buffer
	for _, line := range lines {
		var sanitized string
		if isIdentityScalar(line) {
			// The scalar rules already replace this line's value; scrubbing
			// it again would double-sanitize the just-written hostname.
			sanitized = sanitizeLine(line, mapping, opts, deviceType, domains)
		} else {
			// Scrub echoed identity before the per-line DNS rewrite so a
			// full FQDN still matches, then apply IP/DNS rules.
			sanitized = sanitizeLine(applyIdentitySubs(line, subs), mapping, opts, deviceType, domains)
		}
		out.WriteString(sanitized)
		out.WriteByte('\n')
	}

	stats := mapping.recordDelta(initialIPs, initialHostnames)
	return out.Bytes(), stats, nil
}

// sanitizeLine applies every per-line transformation rule in order:
// system-group identity scalars, community strings, IP addresses (value
// and OID-index forms), and DNS domains.
func sanitizeLine(line string, mapping *Mapping, opts Options, deviceType string, domains []string) string {
	// 1-3. System-group identity scalars. Match both numeric walks
	// (.1.3.6.1.2.1.1.<n>.0, i.e. snmpwalk -On) and symbolic walks (MIB name).
	isContact := isSystemScalar(line, "4", "sysContact")
	switch {
	case isContact:
		line = stringValueRe.ReplaceAllString(line, "= STRING: "+opts.Contact)
	case isSystemScalar(line, "6", "sysLocation"):
		line = stringValueRe.ReplaceAllString(line, "= STRING: NiAC-Go - "+opts.Location+" - Network Operations")
	case isSystemScalar(line, "5", "sysName"):
		// Use the unquoted value so the scalar and the global echo scrub
		// (collectIdentitySubs) sanitize the identical hostname string.
		if v, ok := scalarStringValue(line); ok {
			line = stringValueRe.ReplaceAllString(line, "= STRING: "+sanitizeHostnameAs(v, deviceType, mapping))
		}
	}

	// 4. SNMP community strings, identified by the OID rather than the text.
	// Matching "community" anywhere on the line replaced any STRING value that
	// happened to mention the word -- vendor help text and sysDescr prose
	// included -- while telling us nothing about whether the value was a secret.
	if isCommunityOID(line) {
		line = stringValueRe.ReplaceAllString(line, "= STRING: "+opts.Community)
	}

	// 4b. Serial numbers identify one physical unit a customer owns. They were
	// kept verbatim as "structurally load-bearing", but only their shape is:
	// a format-preserving stand-in keeps every consumer working.
	line = rewriteSerial(line)

	// 5. IP addresses in IpAddress values.
	line = ipValueRe.ReplaceAllStringFunc(line, func(match string) string {
		ip := ipValueRe.FindStringSubmatch(match)[1]
		if isSpecialIP(ip) {
			return match
		}
		return "IpAddress: " + sanitizeIP(ip, mapping)
	})

	// 6. IP addresses embedded as the row index of a standard IPv4 table column.
	line = rewriteOIDIndexIP(line, mapping)
	line = rewriteEmbeddedPrivateIPs(line, mapping)
	line = rewriteValuePrivateIPs(line, mapping)
	line = rewriteValueIPv6(line, mapping)

	// 7. DNS domains the device's own identity names. Rewriting every .com/.net
	// /.org instead turned the vendor's "www.cisco.com" in sysDescr into
	// "www.cisco.niac-go.com" and damaged the fingerprint testers classify on --
	// while a vendor's support URL is not customer data in the first place.
	if opts.Domain != "" && !isContact {
		line = dnsLocalRe.ReplaceAllString(line, ".niac-go.local")
		line = rewriteIdentityDomains(line, domains, opts.Domain)
	}

	return line
}
