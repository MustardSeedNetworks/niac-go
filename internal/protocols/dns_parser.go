package protocols

import (
	"encoding/hex"
	"net"
	"strconv"
	"strings"
)

func parsePTRName(name []byte) (net.IP, bool, bool) {
	ptrName := strings.ToLower(strings.TrimSuffix(string(name), "."))

	switch {
	case strings.HasSuffix(ptrName, ".in-addr.arpa"):
		ip, ok := parseIPv4PTRName(ptrName)

		return ip, ok, false
	case strings.HasSuffix(ptrName, ".ip6.arpa"):
		ip, ok := parseIPv6PTRName(ptrName)

		return ip, ok, true
	default:
		return nil, false, false
	}
}

func parseIPv4PTRName(name string) (net.IP, bool) {
	base := strings.TrimSuffix(name, ".in-addr.arpa")

	parts := strings.Split(strings.Trim(base, "."), ".")
	if len(parts) != dnsIPv4Octets {
		return nil, false
	}

	ip := net.IPv4(0, 0, 0, 0).To4()

	for i := range dnsIPv4Octets {
		val, err := strconv.Atoi(parts[len(parts)-1-i])
		if err != nil || val < 0 || val > dnsMaxByteValue {
			return nil, false
		}

		ip[i] = byte(val)
	}

	return ip, true
}

func parseIPv6PTRName(name string) (net.IP, bool) {
	base := strings.TrimSuffix(name, ".ip6.arpa")

	nibbles := strings.Split(strings.Trim(base, "."), ".")
	if len(nibbles) != dnsIPv6NibbleLen {
		return nil, false
	}

	var builder strings.Builder

	builder.Grow(dnsIPv6NibbleLen)

	for i := len(nibbles) - 1; i >= 0; i-- {
		if len(nibbles[i]) != 1 {
			return nil, false
		}

		builder.WriteString(nibbles[i])
	}

	data, err := hex.DecodeString(builder.String())
	if err != nil || len(data) != net.IPv6len {
		return nil, false
	}

	return net.IP(data), true
}

// isValidDNSName validates DNS name length per RFC 1035
// SECURITY FIX MEDIUM-2: Prevents malformed DNS responses.
func isValidDNSName(name []byte) bool {
	// RFC 1035: Maximum domain name length is 255 bytes
	if len(name) > dnsMaxNameLen {
		return false
	}

	// Validate individual label lengths (max 63 bytes per label)
	labels := strings.SplitSeq(string(name), ".")
	for label := range labels {
		if len(label) > dnsMaxLabelLen {
			return false
		}
	}

	return true
}

func encodeDNSName(name []byte) []byte {
	trimmed := strings.TrimSuffix(string(name), ".")
	if trimmed == "" {
		return []byte{0}
	}

	labels := strings.Split(trimmed, ".")
	buf := make([]byte, 0, len(trimmed)+dnsBufPadding)

	for _, label := range labels {
		if label == "" {
			continue
		}

		buf = append(buf, byte(len(label)))
		buf = append(buf, []byte(label)...)
	}

	return buf
}
