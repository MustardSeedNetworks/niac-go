package snmp

import (
	"strconv"
	"strings"
)

// parseOIDToSubIDs parses an OID string into sub-IDs.
func parseOIDToSubIDs(oid string) []int {
	oid = strings.TrimPrefix(oid, ".")
	parts := strings.Split(oid, ".")

	subIDs := make([]int, len(parts))

	for i, p := range parts {
		n, _ := strconv.Atoi(p)
		subIDs[i] = n
	}

	return subIDs
}

// encodeOID encodes an OID string to BER format.
func encodeOID(oid string) []byte {
	subIDs := parseOIDToSubIDs(oid)
	if len(subIDs) < OIDPartsMinPDU {
		return nil
	}

	// First two octets are combined
	buf := []byte{byte(40*subIDs[0] + subIDs[1])}

	for i := 2; i < len(subIDs); i++ {
		subID := subIDs[i]
		if subID < OIDSubIDThreshold {
			buf = append(buf, byte(subID))
		} else {
			// Variable length encoding
			var encoded []byte
			for subID > 0 {
				encoded = append([]byte{byte(subID&BERLowMask) | BERHighBitMask}, encoded...)
				subID >>= OIDShiftBits
			}

			encoded[len(encoded)-1] &= BERLowMask // Clear high bit of last byte
			buf = append(buf, encoded...)
		}
	}

	return buf
}

// decodeOID decodes a BER-encoded OID to string.
func decodeOID(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	// First byte encodes first two sub-IDs
	subIDs := []int{int(data[0]) / 40, int(data[0]) % 40}

	i := 1
	for i < len(data) {
		var subID int

		for {
			b := data[i]
			i++

			subID = (subID << OIDShiftBits) | int(b&BERLowMask)

			if (b & BERHighBitMask) == 0 {
				break
			}

			if i >= len(data) {
				break
			}
		}

		subIDs = append(subIDs, subID)
	}

	parts := make([]string, len(subIDs))
	for i, id := range subIDs {
		parts[i] = strconv.Itoa(id)
	}

	return strings.Join(parts, ".")
}

// CompareOIDs compares two OID strings lexicographically.
func CompareOIDs(oid1, oid2 string) int {
	subIDs1 := parseOIDToSubIDs(oid1)
	subIDs2 := parseOIDToSubIDs(oid2)

	minLen := min(len(subIDs1), len(subIDs2))

	for i := range minLen {
		if subIDs1[i] < subIDs2[i] {
			return -1
		}

		if subIDs1[i] > subIDs2[i] {
			return 1
		}
	}

	if len(subIDs1) < len(subIDs2) {
		return -1
	}

	if len(subIDs1) > len(subIDs2) {
		return 1
	}

	return 0
}
