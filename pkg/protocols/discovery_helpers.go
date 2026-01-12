package protocols

import (
	"encoding/binary"
	"net"
	"strings"

	"github.com/krisarmstrong/niac-go/pkg/config"
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

	for token := range strings.FieldsSeq(s) {
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

// sendDiscoveryFrame builds and sends an 802.3 Ethernet frame for discovery protocols (CDP, FDP).
// It takes the destination multicast MAC, device, payload, and stack to send through.
func sendDiscoveryFrame(dstMACString string, device *config.Device, payload []byte, stack *Stack) error {
	// Build Ethernet header
	dstMAC, _ := net.ParseMAC(dstMACString)

	// Discovery protocols use length field instead of EtherType (802.3 format)
	payloadLen := min(len(payload), 65535)

	length := uint16(payloadLen) // #nosec G115 -- payloadLen capped at 65535 above

	// Build raw Ethernet frame with 802.3 format
	frame := make([]byte, 14+len(payload))

	// Destination MAC
	copy(frame[0:6], dstMAC)

	// Source MAC
	copy(frame[6:12], device.MACAddress)

	// Length field (instead of EtherType)
	binary.BigEndian.PutUint16(frame[12:14], length)

	// Payload (LLC/SNAP + protocol data)
	copy(frame[14:], payload)

	// Get serial number
	stack.mu.Lock()
	stack.serialNumber++
	serialNum := stack.serialNumber
	stack.mu.Unlock()

	// Create and send packet
	pkt := &Packet{
		Buffer:       frame,
		Length:       len(frame),
		SerialNumber: serialNum,
		Device:       device,
	}

	stack.Send(pkt)

	return nil
}
