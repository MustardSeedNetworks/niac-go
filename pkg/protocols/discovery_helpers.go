package protocols

import (
	"encoding/binary"
	"strings"
)

// ethernetPayload returns the payload that follows the Ethernet header,
// handling optional single VLAN tag. Returns false if the frame is too short.
func ethernetPayload(frame []byte) ([]byte, bool) {
	if len(frame) < 14 {
		return nil, false
	}
	offset := 14
	etherType := binary.BigEndian.Uint16(frame[12:14])
	if etherType == EtherTypeVLAN {
		if len(frame) < 18 {
			return nil, false
		}
		offset += 4
	}
	if offset > len(frame) {
		return nil, false
	}
	return frame[offset:], true
}

// parseKeyValueFields splits strings shaped like "KEY:value" into a map.
func parseKeyValueFields(s string) map[string]string {
	result := make(map[string]string)
	for _, token := range strings.Fields(s) {
		parts := strings.SplitN(token, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToUpper(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		if key == "" || value == "" {
			continue
		}
		result[key] = value
	}
	return result
}

// coalesceStrings returns the first non-empty, trimmed string.
func coalesceStrings(values ...string) string {
	for _, v := range values {
		if val := strings.TrimSpace(v); val != "" {
			return val
		}
	}
	return ""
}

// joinNonEmpty concatenates non-empty strings with the provided separator.
func joinNonEmpty(sep string, values ...string) string {
	var filtered []string
	for _, v := range values {
		if val := strings.TrimSpace(v); val != "" {
			filtered = append(filtered, val)
		}
	}
	return strings.Join(filtered, sep)
}
