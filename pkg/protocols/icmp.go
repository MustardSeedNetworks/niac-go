package protocols

import (
	"fmt"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/config"
)

// ICMPHandler handles ICMP packets (ping, etc.)
type ICMPHandler struct {
	stack *Stack
}

// NewICMPHandler creates a new ICMP handler
func NewICMPHandler(stack *Stack) *ICMPHandler {
	return &ICMPHandler{
		stack: stack,
	}
}

// HandlePacket processes an ICMP packet
func (h *ICMPHandler) HandlePacket(pkt *Packet, ipLayer *layers.IPv4, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	// Parse ICMP layer
	packet := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	icmpLayer := packet.Layer(layers.LayerTypeICMPv4)
	if icmpLayer == nil {
		if debugLevel >= 2 {
			fmt.Printf("ICMP packet missing ICMP layer sn=%d\n", pkt.SerialNumber)
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
	} else {
		if debugLevel >= 3 {
			fmt.Printf("ICMP packet type=%d code=%d sn=%d\n",
				icmp.TypeCode.Type(), icmp.TypeCode.Code(), pkt.SerialNumber)
		}
	}
}

// handleEchoRequest processes ICMP Echo Request and sends Echo Reply
func (h *ICMPHandler) handleEchoRequest(pkt *Packet, ipLayer *layers.IPv4, icmp *layers.ICMPv4, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	if debugLevel >= 3 {
		fmt.Printf("ICMP Echo Request from %s to %s id=%d seq=%d sn=%d\n",
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
				fmt.Printf("Error sending ICMP reply: %v\n", err)
			}
		} else {
			h.stack.IncrementStat("icmp_replies")
			if debugLevel >= 3 {
				fmt.Printf("ICMP Echo Reply from %s (%s) to %s device=%s\n",
					ipLayer.DstIP, device.MACAddress, ipLayer.SrcIP, device.Name)
			}
		}
	}
}

// sendEchoReply sends an ICMP Echo Reply
func (h *ICMPHandler) sendEchoReply(srcMAC, dstMAC []byte, srcIP, dstIP []byte, id, seq uint16, payload []byte, device *config.Device) error {
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
		return fmt.Errorf("error serializing ICMP reply: %v", err)
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

// SendICMPUnreachable sends an ICMP Destination Unreachable message
func (h *ICMPHandler) SendICMPUnreachable(srcIP, dstIP []byte, srcMAC, dstMAC []byte, code uint8, originalPacket []byte) error {
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
		return fmt.Errorf("error serializing ICMP unreachable: %v", err)
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
		fmt.Printf("Sent ICMP Destination Unreachable (code=%d) from %s to %s sn=%d\n",
			code, srcIP, dstIP, serialNum)
	}

	return nil
}
