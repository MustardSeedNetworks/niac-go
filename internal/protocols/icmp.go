package protocols

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// ICMP default TTL values.
const (
	defaultTTLIPv4 uint8 = 64  // default TTL for IPv4 ICMP responses
	defaultTTLIPv6 uint8 = 128 // default hop limit for IPv6 ICMPv6 responses
)

// IPv4 header field constants.
const (
	icmpIPv4Version = 4  // IPv4 version field
	icmpIPv4IHL     = 5  // IPv4 header length (5 = 20 bytes)
	icmpIPv4TTL     = 64 // IPv4 default TTL
)

// ICMP Router Advertisement constants.
const (
	icmpRAHeaderSize      = 4  // RA header size (numAddrs, addrEntrySize, lifetime)
	icmpRAEntrySize       = 8  // Router entry size (2 words * 4 bytes)
	icmpOrigDatagramBytes = 8  // First 8 bytes of original datagram in error messages (RFC 792)
	icmpTimestampReplyLen = 56 // Timestamp reply message size
	// icmpUnreachableMaxOrig caps the original-datagram echo in ICMP error
	// messages: max IPv4 header (60 bytes) + 8 bytes of payload (RFC 792).
	icmpUnreachableMaxOrig = 68
)

// ICMPHandler handles ICMP packets (ping, etc.)
type ICMPHandler struct {
	stack *Stack
}

// NewICMPHandler creates a new ICMP handler.
func NewICMPHandler(stack *Stack) *ICMPHandler {
	return &ICMPHandler{
		stack: stack,
	}
}

// HandlePacket processes an ICMP packet.
func (h *ICMPHandler) HandlePacket(pkt *Packet, ipLayer *layers.IPv4, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	// Parse ICMP layer
	packet := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)

	icmpLayer := packet.Layer(layers.LayerTypeICMPv4)
	if icmpLayer == nil {
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(
				os.Stdout,
				"ICMP packet missing ICMP layer sn=%d\n",
				pkt.SerialNumber,
			)
		}

		return
	}

	icmp, ok := icmpLayer.(*layers.ICMPv4)
	if !ok {
		return
	}

	// Handle ICMP Echo Request (ping)
	switch icmp.TypeCode.Type() {
	case layers.ICMPv4TypeEchoRequest:
		h.stack.IncrementStat("icmp_requests")
		h.handleEchoRequest(pkt, ipLayer, icmp, devices)
	case layers.ICMPv4TypeAddressMaskRequest:
		h.handleAddressMaskRequest(pkt, ipLayer, icmp, devices)
	case layers.ICMPv4TypeRouterSolicitation:
		h.handleRouterSolicitation(pkt, ipLayer, icmp)
	default:
		if debugLevel >= DebugLevelVerbose {
			_, _ = fmt.Fprintf(os.Stdout, "ICMP packet type=%d code=%d sn=%d\n",
				icmp.TypeCode.Type(), icmp.TypeCode.Code(), pkt.SerialNumber)
		}
	}
}

// handleAddressMaskRequest responds to ICMP Address Mask Requests (type 17).
func (h *ICMPHandler) handleAddressMaskRequest(
	pkt *Packet,
	ipLayer *layers.IPv4,
	icmp *layers.ICMPv4,
	devices []*config.Device,
) {
	dstIP := ipLayer.SrcIP
	replyDstMAC := pkt.GetSourceMAC()
	targetMAC := pkt.GetDestMAC()
	l2Broadcast := frameIsBroadcast(pkt)

	for _, device := range devices {
		if !l2Broadcast && !bytes.Equal(device.MACAddress, targetMAC) {
			continue
		}
		srcIP := ipLayer.DstIP
		if !h.stack.deviceOwnsIPv4(device, srcIP) {
			srcIP = h.stack.firstStateIPv4Address(device)
		}
		h.sendAddressMaskReply(device, srcIP, dstIP, replyDstMAC, icmp)
	}
}

// sendAddressMaskReply sends an ICMP address mask reply from a device.
func (h *ICMPHandler) sendAddressMaskReply(
	device *config.Device,
	srcIP net.IP,
	dstIP net.IP,
	replyDstMAC net.HardwareAddr,
	icmp *layers.ICMPv4,
) {
	if device.ICMPConfig == nil || device.ICMPConfig.AddressMaskReply == nil {
		return
	}

	if len(device.MACAddress) == 0 {
		return
	}

	if srcIP == nil {
		return
	}

	mask := device.ICMPConfig.AddressMaskReply.To4()
	if mask == nil {
		return
	}

	icmpReply := &layers.ICMPv4{
		TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeAddressMaskReply, 0),
		Id:       icmp.Id,
		Seq:      icmp.Seq,
	}

	err := h.sendICMPWithPayload(
		srcIP,
		dstIP,
		device.MACAddress,
		replyDstMAC,
		icmpReply,
		mask,
		device,
	)
	if err != nil && h.stack.GetDebugLevel() >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "ICMP: Address Mask Reply failed: %v\n", err)
	}
}

// handleRouterSolicitation responds to ICMP Router Solicitation (type 10).
func (h *ICMPHandler) handleRouterSolicitation(
	pkt *Packet,
	ipLayer *layers.IPv4,
	icmp *layers.ICMPv4,
) {
	debugLevel := h.stack.GetDebugLevel()
	dstMAC := pkt.GetSourceMAC()

	for _, device := range h.stack.devicesFor(pkt.VLAN).GetAll() {
		if device.ICMPConfig == nil || device.ICMPConfig.RouterAdvertisement == nil {
			continue
		}

		srcIP := h.stack.firstStateIPv4Address(device)
		if srcIP == nil || len(device.MACAddress) == 0 {
			continue
		}

		payload := buildRouterAdvertisementPayload(device.ICMPConfig.RouterAdvertisement)

		icmpReply := &layers.ICMPv4{
			TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeRouterAdvertisement, 0),
			Id:       icmp.Id,
			Seq:      icmp.Seq,
		}
		err := h.sendICMPWithPayload(
			srcIP,
			ipLayer.SrcIP,
			device.MACAddress,
			dstMAC,
			icmpReply,
			payload,
			device,
		)
		if err != nil {
			if debugLevel >= DebugLevelInfo {
				_, _ = fmt.Fprintf(os.Stdout, "ICMP: Router Advertisement failed: %v\n", err)
			}
		}
	}
}

func buildRouterAdvertisementPayload(ra *config.IcmpRouterAdvertisement) []byte {
	if ra == nil {
		return nil
	}

	numAddrs := len(ra.Routers)
	payload := make([]byte, icmpRAHeaderSize+icmpRAEntrySize*numAddrs)
	payload[0] = safeconv.Byte(numAddrs)
	payload[1] = 2 // Address entry size (2 words)
	// Safe conversion: RA lifetime is bounded by protocol definition
	binary.BigEndian.PutUint16(
		payload[2:4],
		safeconv.Uint16(ra.Lifetime),
	)

	offset := 4

	for _, router := range ra.Routers {
		if ip := router.Address.To4(); ip != nil {
			copy(payload[offset:offset+4], ip)
			// Safe conversion: router preference is bounded by protocol definition
			binary.BigEndian.PutUint32(
				payload[offset+4:offset+8],
				safeconv.Uint32(router.Preference),
			)
			offset += 8
		}
	}

	return payload[:offset]
}

func (h *ICMPHandler) sendICMPWithPayload(
	srcIP, dstIP net.IP,
	srcMAC, dstMAC net.HardwareAddr,
	icmpLayer *layers.ICMPv4,
	payload []byte,
	device *config.Device,
) error {
	ttl := defaultTTLIPv4
	if device != nil && device.ICMPConfig != nil && device.ICMPConfig.TTL > 0 {
		ttl = device.ICMPConfig.TTL
	}

	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}

	ip := &layers.IPv4{
		Version:  icmpIPv4Version,
		IHL:      icmpIPv4IHL,
		TTL:      ttl,
		Protocol: layers.IPProtocolICMPv4,
		SrcIP:    srcIP,
		DstIP:    dstIP,
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buf, opts,
		eth,
		ip,
		icmpLayer,
		gopacket.Payload(payload),
	)
	if err != nil {
		return fmt.Errorf("error serializing ICMP reply: %w", err)
	}

	h.stack.mu.Lock()
	h.stack.serialNumber++
	serialNum := h.stack.serialNumber
	h.stack.mu.Unlock()

	pkt := &Packet{
		Buffer:       buf.Bytes(),
		Length:       len(buf.Bytes()),
		SerialNumber: serialNum,
		Device:       device,
	}
	h.stack.Send(pkt)

	return nil
}

// SendICMPTimeExceeded sends an ICMP Time Exceeded message.
func (h *ICMPHandler) SendICMPTimeExceeded(
	srcIP, dstIP net.IP,
	srcMAC, dstMAC net.HardwareAddr,
	originalIP *layers.IPv4,
	device *config.Device,
	vlan int,
) error {
	if originalIP == nil {
		return ErrOriginalIPLayerMissing
	}

	origHeader := originalIP.LayerContents()

	origPayload := originalIP.Payload
	if len(origPayload) > icmpOrigDatagramBytes {
		origPayload = origPayload[:icmpOrigDatagramBytes]
	}

	origHeader = append(origHeader, origPayload...)

	icmpLayer := &layers.ICMPv4{
		TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeTimeExceeded, 0),
	}

	// A router hop replies with its own IP TTL, not the IPv6 default.
	ttl := defaultTTLIPv4
	if device != nil && device.ICMPConfig != nil && device.ICMPConfig.TTL > 0 {
		ttl = device.ICMPConfig.TTL
	}

	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}
	ipLayer := &layers.IPv4{
		Version:  icmpIPv4Version,
		IHL:      icmpIPv4IHL,
		TTL:      ttl,
		Protocol: layers.IPProtocolICMPv4,
		SrcIP:    srcIP,
		DstIP:    dstIP,
	}

	buf := gopacket.NewSerializeBuffer()

	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	err := gopacket.SerializeLayers(
		buf,
		opts,
		eth,
		ipLayer,
		icmpLayer,
		gopacket.Payload(origHeader),
	)
	if err != nil {
		return fmt.Errorf("error serializing ICMP time exceeded: %w", err)
	}

	h.stack.mu.Lock()
	h.stack.serialNumber++
	serialNum := h.stack.serialNumber
	h.stack.mu.Unlock()

	pkt := &Packet{
		Buffer:       buf.Bytes(),
		Length:       len(buf.Bytes()),
		SerialNumber: serialNum,
		Device:       device,
		VLAN:         vlan, // echo the probe's VLAN or the reply never reaches a tagged tester
	}
	h.stack.Send(pkt)

	return nil
}

func firstIPv4Address(device *config.Device) net.IP {
	if device == nil {
		return nil
	}

	for _, ip := range device.IPAddresses {
		if v4 := ip.To4(); v4 != nil {
			return v4
		}
	}

	return nil
}

// handleEchoRequest processes ICMP Echo Request and sends Echo Reply.
func (h *ICMPHandler) handleEchoRequest(
	pkt *Packet,
	ipLayer *layers.IPv4,
	icmp *layers.ICMPv4,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	if debugLevel >= DebugLevelVerbose {
		_, _ = fmt.Fprintf(os.Stdout, "ICMP Echo Request from %s to %s id=%d seq=%d sn=%d\n",
			ipLayer.SrcIP, ipLayer.DstIP, icmp.Id, icmp.Seq, pkt.SerialNumber)
	}

	// Get source MAC from original packet
	srcMAC := pkt.GetSourceMAC()

	// Send reply from each matching device
	for _, device := range devices {
		if len(device.MACAddress) == 0 {
			continue
		}

		if !h.stack.deviceOwnsIPv4(device, ipLayer.DstIP) {
			continue
		}

		// Build ICMP Echo Reply
		err := h.sendEchoReply(
			h.stack.replySourceMAC(pkt, device),
			srcMAC,
			ipLayer.DstIP,
			ipLayer.SrcIP,
			icmp.Id,
			icmp.Seq,
			icmp.Payload,
			device,
			pkt.VLAN,
		)
		if err != nil {
			if debugLevel >= DebugLevelInfo {
				_, _ = fmt.Fprintf(os.Stdout, "Error sending ICMP reply: %v\n", err)
			}
		} else {
			h.stack.IncrementStat("icmp_replies")

			if debugLevel >= DebugLevelVerbose {
				_, _ = fmt.Fprintf(os.Stdout, "ICMP Echo Reply from %s (%s) to %s device=%s\n",
					ipLayer.DstIP, device.MACAddress, ipLayer.SrcIP, device.Name)
			}
		}
	}
}

// sendEchoReply sends an ICMP Echo Reply.
func (h *ICMPHandler) sendEchoReply(
	srcMAC, dstMAC []byte,
	srcIP, dstIP []byte,
	id, seq uint16,
	payload []byte,
	device *config.Device,
	vlan int,
) error {
	// Get TTL from config, or use default
	ttl := defaultTTLIPv4
	if device.ICMPConfig != nil && device.ICMPConfig.TTL > 0 {
		ttl = device.ICMPConfig.TTL
	}

	// Build Ethernet header
	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}

	// Build IP header
	ipLayer := &layers.IPv4{
		Version:  icmpIPv4Version,
		IHL:      icmpIPv4IHL,
		TTL:      ttl,
		Protocol: layers.IPProtocolICMPv4,
		SrcIP:    srcIP,
		DstIP:    dstIP,
	}

	// Build ICMP header
	icmpLayer := &layers.ICMPv4{
		TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoReply, 0),
		Id:       id,
		Seq:      seq,
	}

	// Serialize
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts,
		eth,
		ipLayer,
		icmpLayer,
		gopacket.Payload(payload),
	)
	if err != nil {
		return fmt.Errorf("error serializing ICMP reply: %w", err)
	}

	// Get serial number
	h.stack.mu.Lock()
	h.stack.serialNumber++
	serialNum := h.stack.serialNumber
	h.stack.mu.Unlock()

	// Create and send packet
	pkt := &Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: serialNum,
		Device:       device,
		VLAN:         vlan, // reply on the VLAN the echo request arrived on
	}

	h.stack.Send(pkt)

	return nil
}

// SendICMPUnreachable sends an ICMP Destination Unreachable message.
func (h *ICMPHandler) SendICMPUnreachable(
	srcIP, dstIP []byte,
	srcMAC, dstMAC []byte,
	code uint8,
	originalPacket []byte,
	vlan int,
) error {
	// Build Ethernet header
	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}

	// Build IP header
	ipLayer := &layers.IPv4{
		Version:  icmpIPv4Version,
		IHL:      icmpIPv4IHL,
		TTL:      icmpIPv4TTL,
		Protocol: layers.IPProtocolICMPv4,
		SrcIP:    srcIP,
		DstIP:    dstIP,
	}

	// Build ICMP Destination Unreachable
	icmpLayer := &layers.ICMPv4{
		TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeDestinationUnreachable, code),
	}

	// Include original IP header + 8 bytes of data per RFC 792 §3.2.
	payload := originalPacket
	if len(payload) > icmpUnreachableMaxOrig {
		payload = payload[:icmpUnreachableMaxOrig]
	}

	// Serialize
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts,
		eth,
		ipLayer,
		icmpLayer,
		gopacket.Payload(payload),
	)
	if err != nil {
		return fmt.Errorf("error serializing ICMP unreachable: %w", err)
	}

	// Get serial number
	h.stack.mu.Lock()
	h.stack.serialNumber++
	serialNum := h.stack.serialNumber
	h.stack.mu.Unlock()

	// Create and send packet
	pkt := &Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: serialNum,
		VLAN:         vlan, // reply on the VLAN the request arrived on (tagged or untagged)
	}

	h.stack.Send(pkt)

	if h.stack.GetDebugLevel() >= DebugLevelVerbose {
		_, _ = fmt.Fprintf(
			os.Stdout,
			"Sent ICMP Destination Unreachable (code=%d) from %s to %s sn=%d\n",
			code,
			srcIP,
			dstIP,
			serialNum,
		)
	}

	return nil
}
