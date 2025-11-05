package protocols

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// IPv6 protocol constants
const (
	IPv6HeaderSize = 40

	// Next Header values (extension headers and protocols)
	IPv6NextHeaderHopByHop    = 0
	IPv6NextHeaderTCP         = 6
	IPv6NextHeaderUDP         = 17
	IPv6NextHeaderRouting     = 43
	IPv6NextHeaderFragment    = 44
	IPv6NextHeaderESP         = 50
	IPv6NextHeaderAH          = 51
	IPv6NextHeaderICMPv6      = 58
	IPv6NextHeaderNoNext      = 59
	IPv6NextHeaderDestOptions = 60
)

// IPv6 special multicast addresses
var (
	AllNodesMulticast   = net.ParseIP("ff02::1")
	AllRoutersMulticast = net.ParseIP("ff02::2")
)

// IPv6Handler handles IPv6 packets
type IPv6Handler struct {
	stack      *Stack
	debugLevel int
}

// NewIPv6Handler creates a new IPv6 handler
func NewIPv6Handler(stack *Stack, debugLevel int) *IPv6Handler {
	return &IPv6Handler{
		stack:      stack,
		debugLevel: debugLevel,
	}
}

// HandlePacket processes an incoming IPv6 packet
func (h *IPv6Handler) HandlePacket(pkt *Packet) {
	// Parse using gopacket
	packet := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)

	ipv6Layer := packet.Layer(layers.LayerTypeIPv6)
	if ipv6Layer == nil {
		if h.debugLevel >= 2 {
			fmt.Printf("IPv6 packet missing IPv6 layer sn=%d\n", pkt.SerialNumber)
		}
		return
	}

	ipv6, ok := ipv6Layer.(*layers.IPv6)
	if !ok {
		return
	}

	if h.debugLevel >= 3 {
		fmt.Printf("IPv6: %s -> %s, Next Header: %d, Hop Limit: %d sn=%d\n",
			ipv6.SrcIP, ipv6.DstIP, ipv6.NextHeader, ipv6.HopLimit, pkt.SerialNumber)
	}

	// Check if packet is for one of our devices
	devices := h.stack.GetDevices().GetByIP(ipv6.DstIP)
	if devices == nil || len(devices) == 0 {
		// Check for multicast addresses we should respond to
		if !IsIPv6Multicast(ipv6.DstIP) {
			if h.debugLevel >= 3 {
				fmt.Printf("IPv6 packet not for our devices: %s sn=%d\n", ipv6.DstIP, pkt.SerialNumber)
			}
			return
		}
		// Multicast - continue processing for NDP, MLD, etc.
	}

	// Walk extension headers to find the actual next protocol
	nextHeader, offset := h.walkExtensionHeaders(packet, ipv6)

	if h.debugLevel >= 3 {
		fmt.Printf("IPv6: Final protocol after extension headers: %d at offset %d sn=%d\n",
			nextHeader, offset, pkt.SerialNumber)
	}

	// Handle based on the final next header
	switch nextHeader {
	case layers.IPProtocolICMPv6:
		if h.stack.icmpv6Handler != nil {
			h.stack.icmpv6Handler.HandlePacket(pkt, packet, ipv6, devices)
		}
	case layers.IPProtocolUDP:
		// TODO: Pass to UDP handler with IPv6 context
		if h.debugLevel >= 2 {
			fmt.Printf("IPv6: UDP packet (IPv6 UDP not yet implemented) sn=%d\n", pkt.SerialNumber)
		}
	case layers.IPProtocolTCP:
		// TODO: Pass to TCP handler with IPv6 context
		if h.debugLevel >= 2 {
			fmt.Printf("IPv6: TCP packet (IPv6 TCP not yet implemented) sn=%d\n", pkt.SerialNumber)
		}
	case IPv6NextHeaderNoNext:
		// No next header, packet ends here
		if h.debugLevel >= 2 {
			fmt.Printf("IPv6: No next header, packet complete sn=%d\n", pkt.SerialNumber)
		}
	default:
		if h.debugLevel >= 2 {
			fmt.Printf("IPv6: Unhandled next header protocol: %d sn=%d\n", nextHeader, pkt.SerialNumber)
		}
	}
}

// walkExtensionHeaders walks through IPv6 extension headers to find the final protocol
// Returns the final next header value and its offset in the packet
func (h *IPv6Handler) walkExtensionHeaders(packet gopacket.Packet, ipv6 *layers.IPv6) (layers.IPProtocol, int) {
	// Start with the next header from the IPv6 header
	nextHeader := ipv6.NextHeader
	offset := IPv6HeaderSize

	// Get the raw packet data
	data := packet.Data()
	if len(data) < offset {
		return nextHeader, offset
	}

	// Extension header types that need processing
	extensionHeaders := map[layers.IPProtocol]bool{
		IPv6NextHeaderHopByHop:    true,
		IPv6NextHeaderRouting:     true,
		IPv6NextHeaderFragment:    true,
		IPv6NextHeaderDestOptions: true,
		IPv6NextHeaderAH:          true,
		IPv6NextHeaderESP:         true,
	}

	// Walk through extension headers
	for extensionHeaders[nextHeader] {
		if len(data) < offset+2 {
			break
		}

		// Handle fragment header specially (fixed 8 bytes)
		if nextHeader == IPv6NextHeaderFragment {
			if len(data) < offset+8 {
				break
			}
			nextHeader = layers.IPProtocol(data[offset])
			offset += 8
			continue
		}

		// Standard extension header format:
		// Byte 0: Next Header
		// Byte 1: Header Extension Length (in 8-byte units, excluding first 8 bytes)
		nextHeader = layers.IPProtocol(data[offset])
		hdrExtLen := int(data[offset+1])

		// Calculate extension header size
		// Length is in 8-byte units, not including the first 8 bytes
		extHeaderSize := (hdrExtLen + 1) * 8
		offset += extHeaderSize

		if h.debugLevel >= 3 {
			fmt.Printf("IPv6: Processed extension header, next: %d, length: %d bytes\n",
				nextHeader, extHeaderSize)
		}
	}

	return nextHeader, offset
}

// IPv6MulticastToMAC converts an IPv6 multicast address to an Ethernet multicast MAC
// Per RFC 2464: 33:33 followed by the last 4 bytes of the IPv6 address
func IPv6MulticastToMAC(ipv6 net.IP) net.HardwareAddr {
	// Ensure we have a 16-byte IPv6 address
	if len(ipv6) == 16 {
		mac := make(net.HardwareAddr, 6)
		mac[0] = 0x33
		mac[1] = 0x33
		// Copy last 4 bytes of IPv6 address
		copy(mac[2:], ipv6[12:16])
		return mac
	}
	return nil
}

// IsIPv6Multicast checks if an IPv6 address is multicast (starts with ff)
func IsIPv6Multicast(ipv6 net.IP) bool {
	if len(ipv6) == 16 {
		return ipv6[0] == 0xff
	}
	return false
}

// CalculateIPv6Checksum calculates the checksum for IPv6 upper-layer protocols
// Uses the IPv6 pseudo-header per RFC 2460
func CalculateIPv6Checksum(srcIP, dstIP net.IP, nextHeader uint8, payload []byte) uint16 {
	// IPv6 pseudo-header:
	// - Source Address (16 bytes)
	// - Destination Address (16 bytes)
	// - Upper-Layer Packet Length (4 bytes)
	// - Zero (3 bytes)
	// - Next Header (1 byte)

	pseudoHeader := make([]byte, 40)

	// Source IP
	copy(pseudoHeader[0:16], srcIP.To16())

	// Destination IP
	copy(pseudoHeader[16:32], dstIP.To16())

	// Upper-layer packet length (32-bit)
	binary.BigEndian.PutUint32(pseudoHeader[32:36], uint32(len(payload)))

	// Zero padding (3 bytes) at 36:39

	// Next header
	pseudoHeader[39] = nextHeader

	// Calculate checksum over pseudo-header + payload
	sum := uint32(0)

	// Sum pseudo-header
	for i := 0; i < len(pseudoHeader); i += 2 {
		sum += uint32(pseudoHeader[i])<<8 | uint32(pseudoHeader[i+1])
	}

	// Sum payload
	for i := 0; i < len(payload)-1; i += 2 {
		sum += uint32(payload[i])<<8 | uint32(payload[i+1])
	}

	// Handle odd-length payload
	if len(payload)%2 == 1 {
		sum += uint32(payload[len(payload)-1]) << 8
	}

	// Fold 32-bit sum to 16 bits
	for sum > 0xffff {
		sum = (sum >> 16) + (sum & 0xffff)
	}

	// Return one's complement
	return ^uint16(sum)
}

// SendIPv6Packet constructs and sends an IPv6 packet
func (h *IPv6Handler) SendIPv6Packet(srcIP, dstIP net.IP, srcMAC, dstMAC net.HardwareAddr,
	nextHeader layers.IPProtocol, hopLimit uint8, payload []byte) error {

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
		Length:       uint16(len(payload)),
		NextHeader:   nextHeader,
		HopLimit:     hopLimit,
		SrcIP:        srcIP,
		DstIP:        dstIP,
	}

	// Serialize packet
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buf, opts,
		eth,
		ipv6,
		gopacket.Payload(payload),
	)
	if err != nil {
		return fmt.Errorf("failed to serialize IPv6 packet: %w", err)
	}

	if h.debugLevel >= 3 {
		fmt.Printf("IPv6: Sending packet %s -> %s, protocol: %d, size: %d bytes\n",
			srcIP, dstIP, nextHeader, len(buf.Bytes()))
	}

	// Send the packet
	return h.stack.SendRawPacket(buf.Bytes())
}

// SetDebugLevel updates the debug level
func (h *IPv6Handler) SetDebugLevel(level int) {
	h.debugLevel = level
}
