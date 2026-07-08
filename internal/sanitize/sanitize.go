// Package sanitize scrubs real network data out of captured SNMP walks so
// they are safe to share and to seed the demo replay corpus. IP addresses
// and hostnames are transformed deterministically (same input always
// produces the same output), while structurally load-bearing fields —
// serial numbers, MAC addresses, hardware models, interface counts/types,
// VLAN IDs — are left untouched.
//
// The package is pure: Content takes and returns []byte and never
// touches the filesystem. cmd/niac's `sanitize` command and internal/api's
// library walk-sanitize route are both thin I/O wrappers around it.
package sanitize

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
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
	Hostnames  map[string]string `json:"hostnames"`
	Statistics MappingStatistics `json:"statistics"`

	mu sync.RWMutex
}

// NewMapping returns an empty, ready-to-use Mapping.
func NewMapping() *Mapping {
	return &Mapping{
		IPMappings: make(map[string]string),
		Hostnames:  make(map[string]string),
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
	subs := collectIdentitySubs(lines, mapping, opts.Contact)

	var out bytes.Buffer
	for _, line := range lines {
		var sanitized string
		if isIdentityScalar(line) {
			// The scalar rules already replace this line's value; scrubbing
			// it again would double-sanitize the just-written hostname.
			sanitized = sanitizeLine(line, mapping, opts)
		} else {
			// Scrub echoed identity before the per-line DNS rewrite so a
			// full FQDN still matches, then apply IP/DNS rules.
			sanitized = sanitizeLine(applyIdentitySubs(line, subs), mapping, opts)
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
func sanitizeLine(line string, mapping *Mapping, opts Options) string {
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
			line = stringValueRe.ReplaceAllString(line, "= STRING: "+sanitizeHostname(v, mapping))
		}
	}

	// 4. SNMP community strings (symbolic MIB objects only; numeric community
	// columns are enterprise-specific and not represented by a fixed OID).
	if strings.Contains(line, "snmpCommunity") || strings.Contains(line, "community") {
		line = stringValueRe.ReplaceAllString(line, "= STRING: "+opts.Community)
	}

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

	// 7. DNS domains (but skip email addresses in contact strings).
	if opts.Domain != "" && !isContact {
		line = dnsLocalRe.ReplaceAllString(line, ".niac-go.local")
		line = dnsTLDRe.ReplaceAllString(line, "."+opts.Domain)
	}

	return line
}
