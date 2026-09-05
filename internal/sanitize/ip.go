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

	// managementFirstOctet is the one public first octet treated as management.
	managementFirstOctet = 63

	// Class octets: which /16 of 10/8 each address class maps into.
	corporateClassOctet = 2 // 10.2.0.0/16 - Corporate Campus
	remoteClassOctet    = 3 // 10.3.0.0/16 - Remote Offices

	// prefixSlotsPerClass is how many /24s one class octet can hold.
	prefixSlotsPerClass = 256

	// overflowClassOctet starts the range used when a class fills up.
	overflowClassOctet = 200

	// maxOctet bounds an octet value; maxOctetValue is the largest one.
	maxOctet      = 256
	maxOctetValue = 255

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

// sanitizeIP maps ip into 10.0.0.0/8 prefix-preservingly: the /24 it belongs to
// is mapped as a unit and the host octet is carried across unchanged.
//
// Hashing each address independently was the original design and it broke the
// walk's internal consistency. One real /24 on cisco-1841-01 landed in 24
// different /24s, so a device's interface address, its ARP neighbours and its
// routes no longer agreed with each other. Every consumer that derives topology
// from a walk then saw a device contradict itself, which is why the starter
// packs needed address tricks to look plausible.
//
// Mapping the prefix instead keeps every relationship inside the file: same
// subnet in, same subnet out; different subnets stay different.
func sanitizeIP(ip string, mapping *Mapping) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ip
	}
	ipBytes := parsed.To4()
	if ipBytes == nil {
		return sanitizeIPv6(ip, parsed, mapping)
	}
	if isSpecialIP(ip) || isNetmask(ipBytes) {
		return ip
	}

	mapping.mu.RLock()
	if sanitized, exists := mapping.IPMappings[ip]; exists {
		mapping.mu.RUnlock()

		return sanitized
	}
	mapping.mu.RUnlock()

	host := ipBytes[3]
	sanitized := mapPrefix(ipBytes, mapping) + "." + strconv.Itoa(int(host))

	mapping.mu.Lock()
	defer mapping.mu.Unlock()
	if existing, exists := mapping.IPMappings[ip]; exists {
		return existing
	}
	mapping.IPMappings[ip] = sanitized

	return sanitized
}

// mapPrefix returns the sanitized "10.x.y" prefix for an address's /24,
// allocating one on first sight.
//
// The class octet keeps the original scheme's meaning (which /16 of 10/8 a
// network lands in); the third octet starts at a hash of the prefix and probes
// forward until it finds a free slot, so two real /24s can never share one
// sanitized /24 -- which would silently merge two networks into one.
func mapPrefix(ipBytes net.IP, mapping *Mapping) string {
	prefix := fmt.Sprintf("%d.%d.%d", ipBytes[0], ipBytes[1], ipBytes[2])

	mapping.mu.RLock()
	if mapped, exists := mapping.Prefixes[prefix]; exists {
		mapping.mu.RUnlock()

		return mapped
	}
	mapping.mu.RUnlock()

	mapping.mu.Lock()
	defer mapping.mu.Unlock()
	if mapped, exists := mapping.Prefixes[prefix]; exists {
		return mapped
	}
	if mapping.Prefixes == nil {
		// A mapping loaded from an on-disk file written before prefixes existed.
		mapping.Prefixes = make(map[string]string)
	}

	taken := make(map[string]struct{}, len(mapping.Prefixes))
	for _, mapped := range mapping.Prefixes {
		taken[mapped] = struct{}{}
	}

	hash := sha256.Sum256([]byte(prefix))
	start := binary.BigEndian.Uint32(hash[:4])
	class := classOctet(ipBytes[0])
	for attempt := range prefixSlotsPerClass {
		octet3 := safeconv.ByteFromUint32((start + uint32(attempt)) % prefixSlotsPerClass)
		candidate := fmt.Sprintf("10.%d.%d", class, octet3)
		if _, used := taken[candidate]; used {
			continue
		}
		mapping.Prefixes[prefix] = candidate

		return candidate
	}

	// Every slot in the class is spoken for: 256 distinct /24s from one address
	// class in a single mapping. Spilling into the overflow class keeps the
	// guarantee that two networks never merge, at the cost of the class hint.
	for octet2 := overflowClassOctet; octet2 < maxOctet; octet2++ {
		for octet3 := range prefixSlotsPerClass {
			candidate := fmt.Sprintf("10.%d.%d", octet2, octet3)
			if _, used := taken[candidate]; used {
				continue
			}
			mapping.Prefixes[prefix] = candidate

			return candidate
		}
	}

	// 10/8 is exhausted -- ~16 million distinct /24s in one mapping. Returning
	// the prefix unmapped would leak it, so refuse to pretend.
	panic("sanitize: no free /24 left in 10.0.0.0/8")
}

// classOctet keeps the original scheme's meaning for the second octet: which
// /16 of 10/8 a network lands in, chosen by the address class it came from.
func classOctet(firstOctet byte) byte {
	switch {
	case firstOctet == privateIPClassA:
		return 0 // 10.0.0.0/16 - Data Center West
	case firstOctet == privateIPClassB:
		return 1 // 10.1.0.0/16 - Data Center East
	case firstOctet == privateIPClassC:
		return corporateClassOctet
	case firstOctet == managementFirstOctet || firstOctet < privateIPClassA:
		return managementSubnetOctet // 10.100.0.0/16 - Management
	default:
		return remoteClassOctet
	}
}

// isNetmask reports whether the four bytes are a contiguous subnet mask rather
// than an address.
//
// A mask shares the IpAddress SNMP type with an address, so the value alone
// cannot be told apart by type -- and hashing one produced
// "255.255.255.0 -> 10.3.199.4", leaving every sanitized device with a nonsense
// prefix length. Requiring a leading 255 keeps this to the masks devices
// actually use (/8 and longer) rather than every address with a run of high
// bits.
func isNetmask(ipBytes net.IP) bool {
	if ipBytes[0] != maxOctetValue {
		return false
	}
	value := binary.BigEndian.Uint32(ipBytes)
	inverted := ^value

	// A contiguous mask's complement is one less than a power of two.
	return inverted&(inverted+1) == 0
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

// sanitizeIPv6 maps an IPv6 address into the documentation prefix
// 2001:db8::/32, preserving the interface identifier the same way the v4 path
// preserves the host octet.
//
// IPv6 was passed through untouched, so a walk from a dual-stack device shipped
// its real v6 addressing verbatim -- and a global unicast v6 address is at
// least as identifying as a v4 one, often more so, since it frequently carries
// the provider's allocation.
func sanitizeIPv6(original string, parsed net.IP, mapping *Mapping) string {
	if parsed.IsLoopback() || parsed.IsUnspecified() || parsed.IsMulticast() ||
		parsed.IsLinkLocalUnicast() || documentationPrefix().Contains(parsed) {
		return original
	}

	mapping.mu.RLock()
	if sanitized, exists := mapping.IPMappings[original]; exists {
		mapping.mu.RUnlock()

		return sanitized
	}
	mapping.mu.RUnlock()

	sixteen := parsed.To16()
	// The /64 is the network; the interface identifier below it is the "host
	// octet" of v6 and carries no customer information once the prefix is gone.
	network := fmt.Sprintf("%x", sixteen[:ipv6NetworkBytes])
	hash := sha256.Sum256([]byte(network))

	mapped := make(net.IP, net.IPv6len)
	copy(mapped, documentationPrefixBytes())
	// Two bytes of hash fill the subnet field the documentation prefix leaves
	// free, keeping distinct /64s distinct.
	mapped[6], mapped[7] = hash[0], hash[1]
	copy(mapped[ipv6NetworkBytes:], sixteen[ipv6NetworkBytes:])

	sanitized := mapped.String()

	mapping.mu.Lock()
	defer mapping.mu.Unlock()
	if existing, exists := mapping.IPMappings[original]; exists {
		return existing
	}
	mapping.IPMappings[original] = sanitized

	return sanitized
}

// ipv6NetworkBytes is the /64 boundary: the network half of an address.
const ipv6NetworkBytes = 8

// documentationPrefixBytes is 2001:db8:: as raw bytes.
func documentationPrefixBytes() net.IP {
	return net.IP{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
}

// documentationPrefix is 2001:db8::/32, reserved by RFC 3849 for exactly this.
func documentationPrefix() *net.IPNet {
	return &net.IPNet{
		IP:   documentationPrefixBytes(),
		Mask: net.CIDRMask(ipv6DocPrefixBits, ipv6Bits),
	}
}

const (
	ipv6DocPrefixBits = 32
	ipv6Bits          = 128
)

// ipv6ValueRe matches an IPv6 address inside a walk value. Requiring two
// colons keeps it off timestamps and MAC addresses, which use one separator
// style or the other but never a run of two colons.
var ipv6ValueRe = regexp.MustCompile(`\b[0-9a-fA-F:]*::?[0-9a-fA-F:]*:[0-9a-fA-F:]+\b`)

// rewriteValueIPv6 maps IPv6 addresses that appear in a value.
func rewriteValueIPv6(line string, mapping *Mapping) string {
	sep := strings.Index(line, " = ")
	if sep < 0 {
		return line
	}

	value := line[sep:]
	rewritten := ipv6ValueRe.ReplaceAllStringFunc(value, func(match string) string {
		parsed := net.ParseIP(match)
		if parsed == nil || parsed.To4() != nil {
			return match
		}

		return sanitizeIP(match, mapping)
	})
	if rewritten == value {
		return line
	}

	return line[:sep] + rewritten
}
