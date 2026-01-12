package protocols

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/config"
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
		if debugLevel >= 2 {
			_, _ = fmt.Fprintf(os.Stdout, "ICMP packet missing ICMP layer sn=%d\n", pkt.SerialNumber)
		}

		return
	}

	icmp, ok := icmpLayer.(*layers.ICMPv4)
	if !ok {
		return
	}

	// Handle ICMP Echo Request (ping)
	if icmp.TypeCode.Type() == layers.ICMPv4TypeEchoRequest {
		h.stack.IncrementStat("icmp_requests")
		h.handleEchoRequest(pkt, ipLayer, icmp, devices)
	} else if icmp.TypeCode.Type() == layers.ICMPv4TypeAddressMaskRequest {
		h.handleAddressMaskRequest(pkt, ipLayer, icmp)
	} else if icmp.TypeCode.Type() == layers.ICMPv4TypeRouterSolicitation {
		h.handleRouterSolicitation(pkt, ipLayer, icmp)
	} else {
		if debugLevel >= 3 {
			_, _ = fmt.Fprintf(os.Stdout, "ICMP packet type=%d code=%d sn=%d\n",
				icmp.TypeCode.Type(), icmp.TypeCode.Code(), pkt.SerialNumber)
		}
	}
}

// handleAddressMaskRequest responds to ICMP Address Mask Requests (type 17).
func (h *ICMPHandler) handleAddressMaskRequest(pkt *Packet, ipLayer *layers.IPv4, icmp *layers.ICMPv4) {
	debugLevel := h.stack.GetDebugLevel()

	// Use broadcast devices if request was broadcast
	dstMAC := pkt.GetDestMAC()
	isBroadcast := dstMAC != nil && dstMAC.String() == "ff:ff:ff:ff:ff:ff"

	var targets []*config.Device

	if isBroadcast {
		for _, dev := range h.stack.GetDevices().GetAll() {
			if dev.ICMPConfig != nil && dev.ICMPConfig.AddressMaskReply != nil {
				targets = append(targets, dev)
			}
		}
	} else {
		targets = h.stack.GetDevices().GetByIP(ipLayer.DstIP)
	}

	if len(targets) == 0 {
		return
	}

	// Determine destination IP and MAC for reply
	dstIP := ipLayer.SrcIP
	replyDstMAC := pkt.GetSourceMAC()

	for _, device := range targets {
		if device.ICMPConfig == nil || device.ICMPConfig.AddressMaskReply == nil {
			continue
		}

		if len(device.MACAddress) == 0 {
			continue
		}

		srcIP := firstIPv4Address(device)
		if srcIP == nil {
			continue
		}

		// Build Address Mask Reply payload (4 bytes mask)
		mask := device.ICMPConfig.AddressMaskReply.To4()
		if mask == nil {
			continue
		}

		icmpReply := &layers.ICMPv4{
			TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeAddressMaskReply, 0),
			Id:       icmp.Id,
			Seq:      icmp.Seq,
		}

		err := h.sendICMPWithPayload(srcIP, dstIP, device.MACAddress, replyDstMAC, icmpReply, mask, device)
		if err != nil && debugLevel >= 2 {
			_, _ = fmt.Fprintf(os.Stdout, "ICMP: Address Mask Reply failed: %v\n", err)
		}
	}
}

// handleRouterSolicitation responds to ICMP Router Solicitation (type 10).
func (h *ICMPHandler) handleRouterSolicitation(pkt *Packet, ipLayer *layers.IPv4, icmp *layers.ICMPv4) {
	debugLevel := h.stack.GetDebugLevel()
	dstMAC := pkt.GetSourceMAC()

	for _, device := range h.stack.GetDevices().GetAll() {
		if device.ICMPConfig == nil || device.ICMPConfig.RouterAdvertisement == nil {
			continue
		}

		srcIP := firstIPv4Address(device)
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
			if debugLevel >= 2 {
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
	payload := make([]byte, 4+8*numAddrs)
	payload[0] = byte(numAddrs)
	payload[1] = 2 // Address entry size (2 words)
	binary.BigEndian.PutUint16(
		payload[2:4],
		uint16(ra.Lifetime),
	)

	offset := 4

	for _, router := range ra.Routers {
		if ip := router.Address.To4(); ip != nil {
			copy(payload[offset:offset+4], ip)
			binary.BigEndian.PutUint32(
				payload[offset+4:offset+8],
				uint32(router.Preference),
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
	ttl := uint8(64)
	if device != nil && device.ICMPConfig != nil && device.ICMPConfig.TTL > 0 {
		ttl = device.ICMPConfig.TTL
	}

	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}

	ip := &layers.IPv4{
		Version:  4,
		IHL:      5,
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
) error {
	if originalIP == nil {
		return ErrOriginalIPLayerMissing
	}

	origHeader := originalIP.LayerContents()

	origPayload := originalIP.Payload
	if len(origPayload) > 8 {
		origPayload = origPayload[:8]
	}

	payload := append(origHeader, origPayload...)

	icmpLayer := &layers.ICMPv4{
		TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeTimeExceeded, 0),
	}

	ttl := uint8(128)
	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}
	ipLayer := &layers.IPv4{
		Version:  4,
		IHL:      5,
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
	err := gopacket.SerializeLayers(buf, opts, eth, ipLayer, icmpLayer, gopacket.Payload(payload))
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

	if debugLevel >= 3 {
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

		// Check if device has this IP
		hasIP := false

		for _, deviceIP := range device.IPAddresses {
			if deviceIP.Equal(ipLayer.DstIP) {
				hasIP = true

				break
			}
		}

		if !hasIP {
			continue
		}

		// Build ICMP Echo Reply
		err := h.sendEchoReply(
			device.MACAddress,
			srcMAC,
			ipLayer.DstIP,
			ipLayer.SrcIP,
			icmp.Id,
			icmp.Seq,
			icmp.Payload,
			device,
		)
		if err != nil {
			if debugLevel >= 2 {
				_, _ = fmt.Fprintf(os.Stdout, "Error sending ICMP reply: %v\n", err)
			}
		} else {
			h.stack.IncrementStat("icmp_replies")

			if debugLevel >= 3 {
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
) error {
	// Get TTL from config, or use default
	ttl := uint8(64)
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
		Version:  4,
		IHL:      5,
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
) error {
	// Build Ethernet header
	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}

	// Build IP header
	ipLayer := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		Protocol: layers.IPProtocolICMPv4,
		SrcIP:    srcIP,
		DstIP:    dstIP,
	}

	// Build ICMP Destination Unreachable
	icmpLayer := &layers.ICMPv4{
		TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeDestinationUnreachable, code),
	}

	// Include original IP header + 8 bytes of data
	payload := originalPacket
	if len(payload) > 56 {
		payload = payload[:56]
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
	}

	h.stack.Send(pkt)

	if h.stack.GetDebugLevel() >= 3 {
		_, _ = fmt.Fprintf(os.Stdout, "Sent ICMP Destination Unreachable (code=%d) from %s to %s sn=%d\n",
			code, srcIP, dstIP, serialNum)
	}

	return nil
}
