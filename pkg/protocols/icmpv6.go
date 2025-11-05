package protocols

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/config"
)

// ICMPv6 message type constants
const (
	// Error messages (type < 128)
	ICMPv6TypeDestUnreachable   = 1
	ICMPv6TypePacketTooBig      = 2
	ICMPv6TypeTimeExceeded      = 3
	ICMPv6TypeParameterProblem  = 4

	// Informational messages (type >= 128)
	ICMPv6TypeEchoRequest          = 128
	ICMPv6TypeEchoReply            = 129
	ICMPv6TypeRouterSolicitation   = 133
	ICMPv6TypeRouterAdvertisement  = 134
	ICMPv6TypeNeighborSolicitation = 135
	ICMPv6TypeNeighborAdvertisement = 136
	ICMPv6TypeRedirect             = 137
)

// ICMPv6 option types
const (
	ICMPv6OptSourceLinkAddr = 1
	ICMPv6OptTargetLinkAddr = 2
	ICMPv6OptPrefixInfo     = 3
	ICMPv6OptRedirectedHdr  = 4
	ICMPv6OptMTU            = 5
)

// ICMPv6 Neighbor Discovery flags
const (
	NDFlagRouter    = 0x80
	NDFlagSolicited = 0x40
	NDFlagOverride  = 0x20
)

// ICMPv6Handler handles ICMPv6 packets (IPv6's version of ICMP)
type ICMPv6Handler struct {
	stack      *Stack
	debugLevel int
}

// NewICMPv6Handler creates a new ICMPv6 handler
func NewICMPv6Handler(stack *Stack, debugLevel int) *ICMPv6Handler {
	return &ICMPv6Handler{
		stack:      stack,
		debugLevel: debugLevel,
	}
}

// HandlePacket processes an incoming ICMPv6 packet
func (h *ICMPv6Handler) HandlePacket(pkt *Packet, packet gopacket.Packet, ipv6Layer *layers.IPv6, devices []*config.Device) {
	icmpv6Layer := packet.Layer(layers.LayerTypeICMPv6)
	if icmpv6Layer == nil {
		if h.debugLevel >= 2 {
			fmt.Printf("ICMPv6 packet missing ICMPv6 layer sn=%d\n", pkt.SerialNumber)
		}
		return
	}

	icmpv6, ok := icmpv6Layer.(*layers.ICMPv6)
	if !ok {
		return
	}

	msgType := uint8(icmpv6.TypeCode.Type())

	if h.debugLevel >= 3 {
		fmt.Printf("ICMPv6: type %d (%s) sn=%d\n", msgType, h.getTypeName(msgType), pkt.SerialNumber)
	}

	// Handle based on ICMPv6 message type
	switch msgType {
	case ICMPv6TypeEchoRequest:
		h.stack.IncrementStat("icmp_requests")
		h.handleEchoRequest(pkt, packet, ipv6Layer, icmpv6, devices)
	case ICMPv6TypeEchoReply:
		// Silently accept echo replies
	case ICMPv6TypeNeighborSolicitation:
		h.handleNeighborSolicitation(pkt, packet, ipv6Layer)
	case ICMPv6TypeNeighborAdvertisement:
		// Silently accept neighbor advertisements
	case ICMPv6TypeRouterSolicitation:
		h.handleRouterSolicitation(pkt, packet, ipv6Layer)
	case ICMPv6TypeRouterAdvertisement:
		// Silently accept router advertisements
	default:
		if h.debugLevel >= 2 {
			fmt.Printf("ICMPv6: Unhandled type: %d sn=%d\n", msgType, pkt.SerialNumber)
		}
	}
}

// handleEchoRequest responds to ICMPv6 Echo Request (ping6)
func (h *ICMPv6Handler) handleEchoRequest(pkt *Packet, packet gopacket.Packet, ipv6 *layers.IPv6, icmpv6 *layers.ICMPv6, devices []*config.Device) {
	if len(devices) == 0 {
		if h.debugLevel >= 3 {
			fmt.Printf("ICMPv6: No device for Echo Request to %s sn=%d\n", ipv6.DstIP, pkt.SerialNumber)
		}
		return
	}

	if h.debugLevel >= 2 {
		fmt.Printf("ICMPv6: Echo Request %s -> %s sn=%d\n", ipv6.SrcIP, ipv6.DstIP, pkt.SerialNumber)
	}

	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	if ethLayer == nil {
		return
	}
	eth := ethLayer.(*layers.Ethernet)

	// Send reply from each matching device
	for _, device := range devices {
		if len(device.MACAddress) == 0 {
			continue
		}

		// Check if device has this IPv6
		hasIP := false
		for _, deviceIP := range device.IPAddresses {
			if deviceIP.Equal(ipv6.DstIP) {
				hasIP = true
				break
			}
		}
		if !hasIP {
			continue
		}

		reply := &layers.ICMPv6{
			TypeCode: layers.CreateICMPv6TypeCode(ICMPv6TypeEchoReply, 0),
		}

		err := h.sendICMPv6PacketWithDevice(
			ipv6.DstIP,
			ipv6.SrcIP,
			device.MACAddress,
			eth.SrcMAC,
			reply,
			icmpv6.Payload,
			device, // Pass device for config
		)
		if err != nil {
			if h.debugLevel >= 2 {
				fmt.Printf("ICMPv6: Error sending Echo Reply sn=%d: %v\n", pkt.SerialNumber, err)
			}
			return
		}

		h.stack.IncrementStat("icmp_replies")
		if h.debugLevel >= 2 {
			fmt.Printf("ICMPv6: Sent Echo Reply %s -> %s sn=%d\n", ipv6.DstIP, ipv6.SrcIP, pkt.SerialNumber)
		}
	}
}

// handleNeighborSolicitation responds to Neighbor Solicitation (NDP - like ARP for IPv6)
func (h *ICMPv6Handler) handleNeighborSolicitation(pkt *Packet, packet gopacket.Packet, ipv6 *layers.IPv6) {
	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	if ethLayer == nil {
		return
	}
	eth := ethLayer.(*layers.Ethernet)

	// Parse NS message: Type(1) | Code(1) | Checksum(2) | Reserved(4) | Target Address(16) | Options...
	data := packet.ApplicationLayer().Payload()
	if len(data) < 20 {
		if h.debugLevel >= 2 {
			fmt.Printf("ICMPv6: NS too short sn=%d\n", pkt.SerialNumber)
		}
		return
	}

	// Extract target IPv6 address (bytes 4-20)
	targetIP := net.IP(data[4:20])

	if h.debugLevel >= 2 {
		fmt.Printf("ICMPv6: NS for %s from %s sn=%d\n", targetIP, ipv6.SrcIP, pkt.SerialNumber)
	}

	// Find device with target IPv6
	devices := h.stack.devices.GetByIPv6(targetIP)
	if len(devices) == 0 {
		if h.debugLevel >= 3 {
			fmt.Printf("ICMPv6: No device for target %s sn=%d\n", targetIP, pkt.SerialNumber)
		}
		return
	}

	// Send NA for each matching device
	for _, device := range devices {
		err := h.sendNeighborAdvertisement(device, ipv6.SrcIP, eth.SrcMAC, targetIP)
		if err != nil {
			if h.debugLevel >= 2 {
				fmt.Printf("ICMPv6: Error sending NA sn=%d: %v\n", pkt.SerialNumber, err)
			}
			continue
		}

		if h.debugLevel >= 2 {
			fmt.Printf("ICMPv6: Sent NA for %s (MAC: %s) sn=%d\n", targetIP, device.MACAddress, pkt.SerialNumber)
		}
	}
}

// sendNeighborAdvertisement sends a Neighbor Advertisement response
func (h *ICMPv6Handler) sendNeighborAdvertisement(device *config.Device, dstIP net.IP,
	dstMAC net.HardwareAddr, targetIP net.IP) error {

	// Build Neighbor Advertisement message
	// Format: Type(1) | Code(1) | Checksum(2) | Flags(1) | Reserved(3) | Target Address(16) | Options...
	payload := make([]byte, 24) // 4 header + 16 target + 8 option

	// Type = 136 (Neighbor Advertisement)
	payload[0] = ICMPv6TypeNeighborAdvertisement

	// Code = 0
	payload[1] = 0

	// Checksum = 0 (will be calculated)
	payload[2] = 0
	payload[3] = 0

	// Flags: Solicited + Override
	payload[4] = NDFlagSolicited | NDFlagOverride

	// Reserved (3 bytes)
	payload[5] = 0
	payload[6] = 0
	payload[7] = 0

	// Target Address (16 bytes)
	copy(payload[8:24], targetIP.To16())

	// Option: Target Link-Layer Address
	// Type(1) | Length(1) | Link-Layer Address(6)
	payload = append(payload, ICMPv6OptTargetLinkAddr) // Type
	payload = append(payload, 1)                        // Length (in units of 8 bytes)
	payload = append(payload, device.MACAddress...)     // MAC address (6 bytes)

	// Calculate ICMPv6 checksum
	checksum := CalculateIPv6Checksum(device.IPAddresses[0], dstIP, IPv6NextHeaderICMPv6, payload)
	binary.BigEndian.PutUint16(payload[2:4], checksum)

	// Build ICMPv6 layer
	icmpv6 := &layers.ICMPv6{
		TypeCode: layers.CreateICMPv6TypeCode(ICMPv6TypeNeighborAdvertisement, 0),
	}

	// Send packet
	return h.sendICMPv6Packet(
		device.IPAddresses[0],
		dstIP,
		device.MACAddress,
		dstMAC,
		icmpv6,
		payload[4:], // Skip type, code, checksum
	)
}

// handleRouterSolicitation responds to Router Solicitation messages
func (h *ICMPv6Handler) handleRouterSolicitation(pkt *Packet, packet gopacket.Packet, ipv6 *layers.IPv6) {
	if h.debugLevel >= 2 {
		fmt.Printf("ICMPv6: Router Solicitation from %s sn=%d\n", ipv6.SrcIP, pkt.SerialNumber)
	}

	// Find devices configured as routers
	allDevices := h.stack.devices.GetAll()
	for _, device := range allDevices {
		// Only respond if device is a router with IPv6
		if device.Type == "router" && len(device.IPAddresses) > 0 {
			// TODO: Implement Router Advertisement
			if h.debugLevel >= 2 {
				fmt.Printf("ICMPv6: Would send RA from %s (not implemented) sn=%d\n", device.Name, pkt.SerialNumber)
			}
		}
	}
}

// sendICMPv6Packet sends an ICMPv6 packet
func (h *ICMPv6Handler) sendICMPv6Packet(srcIP, dstIP net.IP, srcMAC, dstMAC net.HardwareAddr,
	icmpv6 *layers.ICMPv6, payload []byte) error {
	return h.sendICMPv6PacketWithDevice(srcIP, dstIP, srcMAC, dstMAC, icmpv6, payload, nil)
}

// sendICMPv6PacketWithDevice sends an ICMPv6 packet with device config
func (h *ICMPv6Handler) sendICMPv6PacketWithDevice(srcIP, dstIP net.IP, srcMAC, dstMAC net.HardwareAddr,
	icmpv6 *layers.ICMPv6, payload []byte, device *config.Device) error {

	// Determine hop limit based on ICMPv6 type
	hopLimit := uint8(255) // Default to 255 for NDP (RFC 4861)
	msgType := icmpv6.TypeCode.Type()

	// NDP types MUST use hop limit 255 per RFC 4861
	isNDP := msgType == ICMPv6TypeNeighborSolicitation ||
		msgType == ICMPv6TypeNeighborAdvertisement ||
		msgType == ICMPv6TypeRouterSolicitation ||
		msgType == ICMPv6TypeRouterAdvertisement ||
		msgType == ICMPv6TypeRedirect

	// For non-NDP types (like Echo Reply), use configured value
	if !isNDP && device != nil && device.ICMPv6Config != nil && device.ICMPv6Config.HopLimit > 0 {
		hopLimit = device.ICMPv6Config.HopLimit
	}

	// Build Ethernet layer
	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv6,
	}

	// Build IPv6 layer
	ipv6 := &layers.IPv6{
		Version:      6,
		TrafficClass: 0,
		FlowLabel:    0,
		Length:       uint16(8 + len(payload)), // ICMPv6 header + payload
		NextHeader:   layers.IPProtocolICMPv6,
		HopLimit:     hopLimit,
		SrcIP:        srcIP,
		DstIP:        dstIP,
	}

	// Set ICMPv6 payload
	icmpv6.Payload = payload

	// Serialize packet
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buf, opts, eth, ipv6, icmpv6)
	if err != nil {
		return fmt.Errorf("failed to serialize ICMPv6 packet: %w", err)
	}

	if h.debugLevel >= 3 {
		fmt.Printf("ICMPv6: Sending packet %s -> %s, type %d, size %d bytes\n",
			srcIP, dstIP, icmpv6.TypeCode.Type(), len(buf.Bytes()))
	}

	// Send the packet
	return h.stack.SendRawPacket(buf.Bytes())
}

// getTypeName returns a human-readable name for an ICMPv6 type
func (h *ICMPv6Handler) getTypeName(msgType uint8) string {
	switch msgType {
	case ICMPv6TypeDestUnreachable:
		return "Destination Unreachable"
	case ICMPv6TypePacketTooBig:
		return "Packet Too Big"
	case ICMPv6TypeTimeExceeded:
		return "Time Exceeded"
	case ICMPv6TypeParameterProblem:
		return "Parameter Problem"
	case ICMPv6TypeEchoRequest:
		return "Echo Request"
	case ICMPv6TypeEchoReply:
		return "Echo Reply"
	case ICMPv6TypeRouterSolicitation:
		return "Router Solicitation"
	case ICMPv6TypeRouterAdvertisement:
		return "Router Advertisement"
	case ICMPv6TypeNeighborSolicitation:
		return "Neighbor Solicitation"
	case ICMPv6TypeNeighborAdvertisement:
		return "Neighbor Advertisement"
	case ICMPv6TypeRedirect:
		return "Redirect"
	default:
		return fmt.Sprintf("Unknown (%d)", msgType)
	}
}

// SetDebugLevel updates the debug level
func (h *ICMPv6Handler) SetDebugLevel(level int) {
	h.debugLevel = level
}
