package protocols

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"github.com/krisarmstrong/niac-go/internal/config"
)

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

// buildDHCPv6Response builds a DHCPv6 response message with all options.
// Caller must hold h.mu (at least RLock).
func (h *DHCPv6Handler) buildDHCPv6Response(
	msgType uint8,
	clientMsg *DHCPv6Message,
	lease *DHCPv6Lease,
	device *config.Device,
	infoOnly bool,
) *DHCPv6Message {
	response := &DHCPv6Message{
		MessageType:   msgType,
		TransactionID: clientMsg.TransactionID,
		Options:       make([]DHCPv6Option, 0),
	}

	// Add Server ID
	serverDUIDLen := min(len(h.serverDUID), maxUint16Val)
	response.Options = append(response.Options, DHCPv6Option{
		Code:   DHCPv6OptServerID,
		Length: safeUint16(serverDUIDLen),
		Data:   h.serverDUID,
	})

	// Add Client ID (echo from request)
	if clientID := h.findOption(clientMsg, DHCPv6OptClientID); clientID != nil {
		response.Options = append(response.Options, *clientID)
	}

	// Add IA_NA with address (if not info-only)
	if !infoOnly && lease != nil {
		ianaOpt := h.buildIANAOption(lease)
		response.Options = append(response.Options, ianaOpt)
	}

	// Add configured options
	h.appendDNSOptions(response)
	h.appendTimeServerOptions(response)
	h.appendSIPOptions(response)
	h.appendPreferenceOption(response, device)

	return response
}

// appendDNSOptions adds DNS server and domain list options.
func (h *DHCPv6Handler) appendDNSOptions(response *DHCPv6Message) {
	if len(h.dnsServers) > 0 {
		dnsData := encodeIPv6List(h.dnsServers)
		response.Options = append(response.Options, DHCPv6Option{
			Code:   DHCPv6OptDNSServers,
			Length: safeUint16(min(len(dnsData), maxUint16Val)),
			Data:   dnsData,
		})
	}

	if len(h.domainList) > 0 {
		domainData := h.encodeDomainList(h.domainList)
		response.Options = append(response.Options, DHCPv6Option{
			Code:   DHCPv6OptDomainList,
			Length: safeUint16(min(len(domainData), maxUint16Val)),
			Data:   domainData,
		})
	}
}

// appendTimeServerOptions adds SNTP and NTP server options.
func (h *DHCPv6Handler) appendTimeServerOptions(response *DHCPv6Message) {
	if len(h.sntpServers) > 0 {
		sntpData := encodeIPv6List(h.sntpServers)
		response.Options = append(response.Options, DHCPv6Option{
			Code:   DHCPv6OptSNTPServers,
			Length: safeUint16(min(len(sntpData), maxUint16Val)),
			Data:   sntpData,
		})
	}

	if len(h.ntpServers) > 0 {
		ntpData := encodeIPv6List(h.ntpServers)
		response.Options = append(response.Options, DHCPv6Option{
			Code:   DHCPv6OptNTPServer,
			Length: safeUint16(min(len(ntpData), maxUint16Val)),
			Data:   ntpData,
		})
	}
}

// appendSIPOptions adds SIP server address and domain options.
func (h *DHCPv6Handler) appendSIPOptions(response *DHCPv6Message) {
	if len(h.sipServers) > 0 {
		sipData := encodeIPv6List(h.sipServers)
		response.Options = append(response.Options, DHCPv6Option{
			Code:   DHCPv6OptSIPServerAddrs,
			Length: safeUint16(min(len(sipData), maxUint16Val)),
			Data:   sipData,
		})
	}

	if len(h.sipDomains) > 0 {
		sipDomainData := h.encodeDomainList(h.sipDomains)
		response.Options = append(response.Options, DHCPv6Option{
			Code:   DHCPv6OptSIPServers,
			Length: safeUint16(min(len(sipDomainData), maxUint16Val)),
			Data:   sipDomainData,
		})
	}
}

// appendPreferenceOption adds the preference option.
func (h *DHCPv6Handler) appendPreferenceOption(response *DHCPv6Message, device *config.Device) {
	preference := uint8(0)
	if device != nil && device.DHCPv6Config != nil {
		preference = device.DHCPv6Config.Preference
	}

	response.Options = append(response.Options, DHCPv6Option{
		Code:   DHCPv6OptPreference,
		Length: 1,
		Data:   []byte{preference},
	})
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
		Length: safeUint16(ianaDataLen),
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

// sendDHCPv6Response sends a DHCPv6 response message.
func (h *DHCPv6Handler) sendDHCPv6Response(msgType uint8, clientMsg *DHCPv6Message, lease *DHCPv6Lease,
	clientIP, serverIP net.IP, serverMAC net.HardwareAddr, device *config.Device, infoOnly bool,
) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	response := h.buildDHCPv6Response(msgType, clientMsg, lease, device, infoOnly)

	// Serialize message
	msgBytes := h.serializeDHCPv6Message(response)

	// Build UDP layer
	udp := &layers.UDP{
		SrcPort: DHCPv6ServerPort,
		DstPort: DHCPv6ClientPort,
	}

	// Build IPv6 layer - send to client's link-local or use multicast
	dstIP := clientIP
	dstMAC := net.HardwareAddr{0x33, 0x33, 0x00, 0x01, 0x00, 0x02} // All DHCP relay agents and servers

	// If client IP is link-local, use it directly
	if clientIP.IsLinkLocalUnicast() {
		// Calculate multicast MAC from client IP
		if clientIP.To4() == nil && len(clientIP) == 16 {
			dstMAC = IPv6MulticastToMAC(GetAllDHCPRelayAgentsAndServers())
		}
	}

	ipv6 := &layers.IPv6{
		Version:    ipv6VersionVal,
		HopLimit:   ipv6DefaultHopLimit,
		NextHeader: layers.IPProtocolUDP,
		SrcIP:      serverIP,
		DstIP:      dstIP,
	}

	// Build Ethernet layer
	eth := &layers.Ethernet{
		SrcMAC:       serverMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv6,
	}

	// Calculate UDP length and checksum
	msgBytesLen := min(len(msgBytes), maxUDPPayload)

	udp.Length = safeUint16(udpHeaderSize + msgBytesLen)

	// Serialize packet
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	// Set UDP payload
	payload := gopacket.Payload(msgBytes)

	_ = udp.SetNetworkLayerForChecksum(ipv6) // error is non-critical for simulation

	err := gopacket.SerializeLayers(buf, opts, eth, ipv6, udp, payload)
	if err != nil {
		return fmt.Errorf("failed to serialize DHCPv6 response: %w", err)
	}

	// Send packet
	return h.stack.SendRawPacket(buf.Bytes())
}

// sendAdvertise sends a DHCPv6 Advertise message.
func (h *DHCPv6Handler) sendAdvertise(
	clientMsg *DHCPv6Message,
	lease *DHCPv6Lease,
	clientIP, serverIP net.IP,
	serverMAC net.HardwareAddr,
	device *config.Device,
) error {
	return h.sendDHCPv6Response(DHCPv6Advertise, clientMsg, lease, clientIP, serverIP, serverMAC, device, false)
}

// sendReply sends a DHCPv6 Reply message.
func (h *DHCPv6Handler) sendReply(
	clientMsg *DHCPv6Message,
	lease *DHCPv6Lease,
	clientIP, serverIP net.IP,
	serverMAC net.HardwareAddr,
	device *config.Device,
) error {
	return h.sendDHCPv6Response(DHCPv6Reply, clientMsg, lease, clientIP, serverIP, serverMAC, device, false)
}

// sendInfoReply sends a DHCPv6 Reply for Information-Request.
func (h *DHCPv6Handler) sendInfoReply(
	clientMsg *DHCPv6Message,
	clientIP, serverIP net.IP,
	serverMAC net.HardwareAddr,
	device *config.Device,
) error {
	return h.sendDHCPv6Response(DHCPv6Reply, clientMsg, nil, clientIP, serverIP, serverMAC, device, true)
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
