package sanitize

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net"
	"regexp"
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
	for _, special := range specials {
		if strings.HasPrefix(ip, special) || ip == special {
			return true
		}
	}
	return false
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

// hasIPIndexedPrefix reports whether oid sits under a known IPv4-indexed
// table column.
func hasIPIndexedPrefix(oid string) bool {
	return ipIndexedOIDRe.MatchString(oid)
}
