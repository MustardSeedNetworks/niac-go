package sanitize

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// Network constants for the deterministic IP mapping. Original first-octet
// class selects which /16 of 10.0.0.0/8 a sanitized address lands in.
const (
	privateIPClassA = 10
	privateIPClassB = 172
	privateIPClassC = 192
	bitShiftOctet   = 8

	// managementSubnetOctet is the second octet used for addresses whose
	// original first octet is < 10 or is exactly 63 (10.100.0.0/16 -
	// Management).
	managementSubnetOctet = 100

	// ipv4OctetCount is the number of trailing OID arcs that form a
	// dotted-quad index.
	ipv4OctetCount = 4
)

// ipValueRe matches an SNMP IpAddress value.
var ipValueRe = regexp.MustCompile(`IpAddress: (\d+\.\d+\.\d+\.\d+)`)

// ipIndexedOIDRe matches the standard IPv4 table columns whose row index
// ends in a dotted-quad IP address. Only OIDs matching this have their
// trailing four arcs rewritten as an IP; applying that to an arbitrary OID
// whose last arcs merely look like octets corrupts the MIB structure.
//
//	.4.20.1 ipAddrTable      · .4.21.1 ipRouteTable
//	.4.22.1 ipNetToMediaTable · .3.1.1 atTable (legacy)
var ipIndexedOIDRe = regexp.MustCompile(`^\.1\.3\.6\.1\.2\.1\.(?:4\.2[012]\.1|3\.1\.1)\.`)

// sanitizeIP deterministically maps ip into 10.0.0.0/8, spreading across
// different /16s based on the original network. The mapping is cached in
// mapping.IPMappings so repeat calls (and concurrent callers) agree.
func sanitizeIP(ip string, mapping *Mapping) string {
	mapping.mu.RLock()
	if sanitized, exists := mapping.IPMappings[ip]; exists {
		mapping.mu.RUnlock()
		return sanitized
	}
	mapping.mu.RUnlock()

	hash := sha256.Sum256([]byte(ip))
	hashInt := binary.BigEndian.Uint32(hash[:4])

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return ip
	}

	ipBytes := parsedIP.To4()
	if ipBytes == nil {
		return ip // IPv6 not supported yet
	}

	var subnet byte
	switch {
	case ipBytes[0] == privateIPClassA:
		subnet = 0 // 10.0.0.0/16 - Data Center West
	case ipBytes[0] == privateIPClassB:
		subnet = 1 // 10.1.0.0/16 - Data Center East
	case ipBytes[0] == privateIPClassC:
		subnet = 2 // 10.2.0.0/16 - Corporate Campus
	case ipBytes[0] == 63 || ipBytes[0] < privateIPClassA:
		subnet = managementSubnetOctet // 10.100.0.0/16 - Management
	default:
		subnet = 3 // 10.3.0.0/16 - Remote Offices
	}

	octet3 := safeconv.ByteFromUint32(hashInt >> bitShiftOctet)
	octet4 := safeconv.ByteFromUint32(hashInt)

	sanitized := fmt.Sprintf("10.%d.%d.%d", subnet, octet3, octet4)

	mapping.mu.Lock()
	if existing, exists := mapping.IPMappings[ip]; exists {
		mapping.mu.Unlock()
		return existing
	}
	mapping.IPMappings[ip] = sanitized
	mapping.mu.Unlock()

	return sanitized
}

// isSpecialIP reports whether ip is a reserved/well-known address that
// must never be transformed (loopback, broadcast, multicast, etc).
func isSpecialIP(ip string) bool {
	specials := []string{
		"0.0.0.0", "255.255.255.255",
		"127.0.0.1", "127.0.0.0",
		"224.0.0.", "239.255.255.250", // Multicast
	}

	return slices.ContainsFunc(specials, func(special string) bool {
		return strings.HasPrefix(ip, special) || ip == special
	})
}

// looksLikeIPOctet reports whether s is a valid single IPv4 octet (0-255,
// digits only, 1-3 characters).
func looksLikeIPOctet(s string) bool {
	if len(s) < 1 || len(s) > 3 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return false
	}
	return val >= 0 && val <= 255
}

// rewriteOIDIndexIP rewrites the trailing dotted-quad IP index of a
// standard IPv4-indexed table column and leaves every other OID untouched,
// so structural arcs that merely look like octets are never disturbed.
// Only the OID field is considered; the value field is handled by the
// IpAddress pass in sanitizeLine.
func rewriteOIDIndexIP(line string, mapping *Mapping) string {
	sep := strings.Index(line, " = ")
	if sep < 0 {
		return line
	}
	oid := line[:sep]
	if !hasIPIndexedPrefix(oid) {
		return line
	}

	arcs := strings.Split(oid, ".")
	if len(arcs) < ipv4OctetCount {
		return line
	}
	index := arcs[len(arcs)-ipv4OctetCount:]
	for _, arc := range index {
		if !looksLikeIPOctet(arc) {
			return line
		}
	}

	ip := strings.Join(index, ".")
	if isSpecialIP(ip) {
		return line
	}

	copy(arcs[len(arcs)-ipv4OctetCount:], strings.Split(sanitizeIP(ip, mapping), "."))
	return strings.Join(arcs, ".") + line[sep:]
}

// rewriteEmbeddedPrivateIPs maps every private, non-10/8 dotted quad inside
// the OID field, wherever it sits in the index.
//
// rewriteOIDIndexIP only covers four tables whose index *ends* in an address.
// The addresses that were shipping in the starter walks were in others:
// tcpConnTable's index carries two of them with ports between
// (.6.13.1.<col>.<lAddr>.<lPort>.<rAddr>.<rPort>), and udpTable, the RFC 4022
// InetAddress tables, LLDP management addresses and two vendor-private trees
// each place theirs differently again. A tcpConnTable row records who a real
// device was actually talking to, so leaving them was the leak.
//
// Rather than encode fourteen index layouts -- two of them vendor-private,
// where the layout would be a guess -- this targets the property that
// identifies the leak: a *private* address outside the range the sanitizer
// maps into. Anything already in 10/8 has been through the mapper, and public
// addresses are handled by rule elsewhere. Four arcs are replaced by four
// arcs, so the OID's structure is unchanged either way.
//
// The residual risk is four consecutive arcs that form a private quad without
// being an address; in these MIB regions that does not occur, and the effect
// would be a renumbered index rather than a malformed OID.
func rewriteEmbeddedPrivateIPs(line string, mapping *Mapping) string {
	sep := strings.Index(line, " = ")
	if sep < 0 {
		return line
	}

	arcs := strings.Split(line[:sep], ".")
	changed := false
	// Slide a four-arc window over the OID's arcs. Scanning the text with a
	// regex instead matches the first four arcs of the OID itself
	// (".1.3.6.1"), so the window never lines up with the address further in.
	for i := 0; i+ipv4OctetCount <= len(arcs); {
		quad := strings.Join(arcs[i:i+ipv4OctetCount], ".")
		if !isPrivateOutsideMappedRange(quad) {
			i++
			continue
		}
		copy(arcs[i:i+ipv4OctetCount], strings.Split(sanitizeIP(quad, mapping), "."))
		changed = true
		// Past the address just replaced: an address cannot overlap another.
		i += ipv4OctetCount
	}
	if !changed {
		return line
	}

	return strings.Join(arcs, ".") + line[sep:]
}

// rewriteValuePrivateIPs maps private, non-10/8 addresses that appear inside a
// value, including in free text.
//
// The IpAddress rule only matches a typed SNMP address. These arrive as
// strings instead: a DISMAN-PING target, an ENTITY-MIB address, and Brocade
// log lines like `SNMP: Auth. failure, intruder IP: 192.168.0.2.` -- prose
// that still names a real host. Arc alignment is not a concern here because a
// value is text, not an OID, so a plain dotted-quad match is exact.
func rewriteValuePrivateIPs(line string, mapping *Mapping) string {
	sep := strings.Index(line, " = ")
	if sep < 0 {
		return line
	}

	value := line[sep:]
	matches := valueQuadRe.FindAllStringSubmatchIndex(value, -1)
	if matches == nil {
		return line
	}

	// Rebuilt back to front so earlier offsets stay valid. The submatch, not
	// the whole match, is the address: the pattern's boundaries are there to
	// reject a quad inside a longer dotted run, and replacing the whole match
	// would eat the characters around it.
	var builder strings.Builder
	last := 0
	for _, m := range matches {
		lo, hi := m[2], m[3]
		quad := value[lo:hi]
		if !isPrivateOutsideMappedRange(quad) {
			continue
		}
		builder.WriteString(value[last:lo])
		builder.WriteString(sanitizeIP(quad, mapping))
		last = hi
	}
	if last == 0 {
		return line
	}
	builder.WriteString(value[last:])

	return line[:sep] + builder.String()
}

// valueQuadRe matches a maximal run of digits and dots, which is then
// validated as an address.
//
// Bounding the quad directly cannot express "not part of a longer dotted run"
// in RE2, which has no lookahead: a trailing boundary of [^0-9.] rejects the
// sentence period in `intruder IP: 192.168.0.2.`, and one of [^0-9] would
// accept the first four octets of a five-octet run. Taking the whole run and
// asking net.ParseIP whether it is an address settles both -- a run ends at a
// digit, so the sentence period is outside it, and 1.2.3.4.5 parses as
// nothing.
var valueQuadRe = regexp.MustCompile(`([0-9][0-9.]*[0-9])`)

// isPrivateOutsideMappedRange reports whether quad is an RFC1918 address that
// the sanitizer has not already mapped into 10/8.
func isPrivateOutsideMappedRange(quad string) bool {
	parsed := net.ParseIP(quad)
	if parsed == nil || parsed.To4() == nil {
		return false
	}
	if !parsed.IsPrivate() {
		return false
	}

	return !mappedIPNet().Contains(parsed)
}

// mappedFirstOctet is the first octet of the range sanitizeIP maps into.
const mappedFirstOctet = 10

// mappedPrefixBits is that range's prefix length.
const mappedPrefixBits = 8

// ipv4Bits is the width of an IPv4 address.
const ipv4Bits = 32

// mappedIPNet is the range sanitizeIP maps into.
func mappedIPNet() *net.IPNet {
	return &net.IPNet{
		IP:   net.IPv4(mappedFirstOctet, 0, 0, 0),
		Mask: net.CIDRMask(mappedPrefixBits, ipv4Bits),
	}
}

// hasIPIndexedPrefix reports whether oid sits under a known IPv4-indexed
// table column.
func hasIPIndexedPrefix(oid string) bool {
	return ipIndexedOIDRe.MatchString(oid)
}
