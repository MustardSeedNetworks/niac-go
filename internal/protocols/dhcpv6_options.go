package protocols

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"time"

	"github.com/krisarmstrong/niac-go/internal/safeconv"
)

// generateDUID generates a DUID-LL (Link-Layer) for the server.
func generateDUID() []byte {
	// DUID-LL format: Type(2) + HW Type(2) + Link-Layer Address(variable)
	duid := make([]byte, duidLLSize)
	binary.BigEndian.PutUint16(duid[0:2], DUIDTypeLL) // DUID-LL
	binary.BigEndian.PutUint16(duid[2:4], 1)          // Ethernet

	// Generate random MAC for server DUID
	mac := make([]byte, macAddrSize)
	_, _ = rand.Read(mac)                               // crypto/rand read errors will result in zero bytes
	mac[0] = (mac[0] | macLocalBitSet) & macUnicastMask // Set local, clear multicast
	copy(duid[4:10], mac)

	return duid
}

// duidString converts DUID bytes to hex string for map key.
func duidString(duid []byte) string {
	return hex.EncodeToString(duid)
}

// parseDHCPv6Message parses a DHCPv6 message from bytes.
func (h *DHCPv6Handler) parseDHCPv6Message(data []byte) (*DHCPv6Message, error) {
	if len(data) < dhcpv6MinMsgSize {
		return nil, fmt.Errorf("%w: %d bytes", ErrMessageTooShort, len(data))
	}

	msg := &DHCPv6Message{
		MessageType: data[0],
		Options:     make([]DHCPv6Option, 0),
	}
	copy(msg.TransactionID[:], data[1:4])

	// Parse options
	offset := 4
	for offset < len(data) {
		if offset+4 > len(data) {
			break
		}

		opt := DHCPv6Option{
			Code:   binary.BigEndian.Uint16(data[offset : offset+2]),
			Length: binary.BigEndian.Uint16(data[offset+2 : offset+4]),
		}
		offset += 4

		if offset+int(opt.Length) > len(data) {
			return nil, ErrOptionDataExceedsLen
		}

		opt.Data = make([]byte, opt.Length)
		copy(opt.Data, data[offset:offset+int(opt.Length)])
		offset += int(opt.Length)

		msg.Options = append(msg.Options, opt)
	}

	return msg, nil
}

// findOption finds an option by code in the message.
func (h *DHCPv6Handler) findOption(msg *DHCPv6Message, code uint16) *DHCPv6Option {
	for i := range msg.Options {
		if msg.Options[i].Code == code {
			return &msg.Options[i]
		}
	}

	return nil
}

// extractClientDUID extracts the client DUID from the message.
func (h *DHCPv6Handler) extractClientDUID(msg *DHCPv6Message) []byte {
	opt := h.findOption(msg, DHCPv6OptClientID)
	if opt != nil {
		return opt.Data
	}

	return nil
}

// extractIANA extracts IANA option from message.
func (h *DHCPv6Handler) extractIANA(msg *DHCPv6Message) (uint32, bool) {
	opt := h.findOption(msg, DHCPv6OptIANA)
	if opt != nil && len(opt.Data) >= 4 {
		iaid := binary.BigEndian.Uint32(opt.Data[0:4])

		return iaid, true
	}

	return 0, false
}

// encodeIPv6List encodes a list of IPv6 addresses into bytes.
func encodeIPv6List(ips []net.IP) []byte {
	data := make([]byte, 0, len(ips)*ipv6AddrSize)
	for _, ip := range ips {
		data = append(data, ip.To16()...)
	}

	return data
}

// encodeDomainList encodes domain names for DHCPv6 Domain Search List option.
func (h *DHCPv6Handler) encodeDomainList(domains []string) []byte {
	data := make([]byte, 0)

	for _, domain := range domains {
		// DNS name encoding: length-prefixed labels
		labels := splitDomainLabels(domain)
		for _, label := range labels {
			data = append(data, byte(len(label)))
			data = append(data, []byte(label)...)
		}

		data = append(data, 0) // Null terminator
	}

	return data
}

// splitDomainLabels splits a domain name into labels.
func splitDomainLabels(domain string) []string {
	if domain == "" {
		return []string{}
	}

	labels := make([]string, 0)
	start := 0

	for i := range len(domain) {
		if domain[i] == '.' {
			if i > start {
				labels = append(labels, domain[start:i])
			}

			start = i + 1
		}
	}

	if start < len(domain) {
		labels = append(labels, domain[start:])
	}

	return labels
}

// buildIANAOption builds an IA_NA option with IA Address.
func (h *DHCPv6Handler) buildIANAOption(lease *DHCPv6Lease) DHCPv6Option {
	// IA_NA option format:
	// IAID (4 bytes) + T1 (4 bytes) + T2 (4 bytes) + IA_NA options
	iaAddrOpt := h.buildIAAddrOption(lease)
	iaAddrBytes := h.serializeOption(iaAddrOpt)
	ianaHeader := make([]byte, ianaHeaderBase+len(iaAddrBytes))

	// IAID
	binary.BigEndian.PutUint32(ianaHeader[0:4], lease.IAID)

	// T1 (renewal time) - 50% of preferred lifetime
	t1 := uint32(h.preferredLifetime.Seconds() / t1DivisorV6)
	binary.BigEndian.PutUint32(ianaHeader[4:8], t1)

	// T2 (rebinding time) - 80% of preferred lifetime
	t2 := uint32(h.preferredLifetime.Seconds() * t2MultiplicandV6 / t2DivisorV6)
	binary.BigEndian.PutUint32(ianaHeader[8:12], t2)

	// Build IA Address option
	copy(ianaHeader[12:], iaAddrBytes)

	ianaDataLen := min(len(ianaHeader), maxUint16Val)

	return DHCPv6Option{
		Code:   DHCPv6OptIANA,
		Length: safeconv.Uint16(ianaDataLen),
		Data:   ianaHeader,
	}
}

// buildIAAddrOption builds an IA Address option.
func (h *DHCPv6Handler) buildIAAddrOption(lease *DHCPv6Lease) DHCPv6Option {
	// IA Address option format:
	// IPv6 address (16 bytes) + preferred-lifetime (4 bytes) + valid-lifetime (4 bytes)
	iaAddrData := make([]byte, iaAddrDataSize)

	// IPv6 address
	copy(iaAddrData[0:16], lease.Address.To16())

	// Preferred lifetime (in seconds)
	preferred := uint32(time.Until(lease.PreferredLifetime).Seconds())
	if time.Now().After(lease.PreferredLifetime) {
		preferred = 0
	}

	binary.BigEndian.PutUint32(iaAddrData[16:20], preferred)

	// Valid lifetime (in seconds)
	valid := uint32(time.Until(lease.ValidLifetime).Seconds())
	if time.Now().After(lease.ValidLifetime) {
		valid = 0
	}

	binary.BigEndian.PutUint32(iaAddrData[20:24], valid)

	return DHCPv6Option{
		Code:   DHCPv6OptIAAddr,
		Length: iaAddrDataSize,
		Data:   iaAddrData,
	}
}

// serializeDHCPv6Message serializes a DHCPv6 message to bytes.
func (h *DHCPv6Handler) serializeDHCPv6Message(msg *DHCPv6Message) []byte {
	// Calculate total size
	size := dhcpv6MinMsgSize // Message type (1) + Transaction ID (3)
	for _, opt := range msg.Options {
		size += optionHeaderSize + int(opt.Length) // Code (2) + Length (2) + Data
	}

	buf := make([]byte, size)
	buf[0] = msg.MessageType
	copy(buf[1:4], msg.TransactionID[:])

	// Serialize options
	offset := 4
	for _, opt := range msg.Options {
		offset += h.serializeOptionAt(buf[offset:], opt)
	}

	return buf
}

// serializeOption serializes a single option.
func (h *DHCPv6Handler) serializeOption(opt DHCPv6Option) []byte {
	buf := make([]byte, optionHeaderSize+opt.Length)
	h.serializeOptionAt(buf, opt)

	return buf
}

// serializeOptionAt serializes an option into a buffer.
func (h *DHCPv6Handler) serializeOptionAt(buf []byte, opt DHCPv6Option) int {
	binary.BigEndian.PutUint16(buf[0:2], opt.Code)
	binary.BigEndian.PutUint16(buf[2:4], opt.Length)
	copy(buf[4:], opt.Data)

	return optionHeaderSize + int(opt.Length)
}

// messageTypeString returns string representation of DHCPv6 message type.
func (h *DHCPv6Handler) messageTypeString(msgType uint8) string {
	switch msgType {
	case DHCPv6Solicit:
		return "SOLICIT"
	case DHCPv6Advertise:
		return "ADVERTISE"
	case DHCPv6Request:
		return "REQUEST"
	case DHCPv6Confirm:
		return "CONFIRM"
	case DHCPv6Renew:
		return "RENEW"
	case DHCPv6Rebind:
		return "REBIND"
	case DHCPv6Reply:
		return "REPLY"
	case DHCPv6Release:
		return "RELEASE"
	case DHCPv6Decline:
		return "DECLINE"
	case DHCPv6Reconfigure:
		return "RECONFIGURE"
	case DHCPv6InfoRequest:
		return "INFORMATION-REQUEST"
	case DHCPv6RelayForw:
		return "RELAY-FORW"
	case DHCPv6RelayRepl:
		return "RELAY-REPL"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", msgType)
	}
}
