package sanitize

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// defaultDeviceType is the device type abbreviation for generic/
// unrecognized devices.
const defaultDeviceType = "dev"

// hostnameNumberModulus bounds the deterministic numeric suffix generated
// for a sanitized hostname (niac-core-{type}-{00..99}).
const hostnameNumberModulus = 100

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
	dnsTLDRe        = regexp.MustCompile(`\.(com|net|org)\b`)
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
func collectIdentitySubs(lines []string, mapping *Mapping, contact string) []identitySub {
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
				add(v, sanitizeHostname(v, mapping))
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

// sanitizeHostname deterministically maps hostname to a
// "niac-core-{type}-{NN}" branded name, inferring the device type from
// common substrings (sw/rtr/ap/srv/fw) and falling back to "dev". The
// mapping is cached in mapping.Hostnames so repeat calls agree.
func sanitizeHostname(hostname string, mapping *Mapping) string {
	mapping.mu.RLock()
	if sanitized, exists := mapping.Hostnames[hostname]; exists {
		mapping.mu.RUnlock()
		return sanitized
	}
	mapping.mu.RUnlock()

	var deviceType string
	lower := strings.ToLower(hostname)
	switch {
	case strings.Contains(lower, "sw") || strings.Contains(lower, "switch"):
		deviceType = "sw"
	case strings.Contains(lower, "rtr") || strings.Contains(lower, "router"):
		deviceType = "rtr"
	case strings.Contains(lower, "ap") || strings.Contains(lower, "access"):
		deviceType = "ap"
	case strings.Contains(lower, "srv") || strings.Contains(lower, "server"):
		deviceType = "srv"
	case strings.Contains(lower, "fw") || strings.Contains(lower, "firewall"):
		deviceType = "fw"
	default:
		deviceType = defaultDeviceType
	}

	hash := sha256.Sum256([]byte(hostname))
	num := binary.BigEndian.Uint16(hash[:2]) % hostnameNumberModulus

	sanitized := fmt.Sprintf("niac-core-%s-%02d", deviceType, num)

	mapping.mu.Lock()
	if existing, exists := mapping.Hostnames[hostname]; exists {
		mapping.mu.Unlock()
		return existing
	}
	mapping.Hostnames[hostname] = sanitized
	mapping.mu.Unlock()

	return sanitized
}
