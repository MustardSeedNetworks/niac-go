package protocols

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"sync/atomic"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/safeconv"
)

// IPv6 protocol constants.
const (
	// IPv6HeaderSize is the size of an IPv6 header in bytes.
	IPv6HeaderSize = 40

	// ipv6PseudoHeaderSize is the size of the IPv6 pseudo-header for checksum calculation.
	ipv6PseudoHeaderSize = 40

	// ipv6MaxUint32 is the maximum uint32 value.
	ipv6MaxUint32 = 0xFFFFFFFF

	// ipv6MaxUint16 is the maximum uint16 value for payload length.
	ipv6MaxUint16 = 65535

	// ipv6ExtHdrUnitSize is the size of one unit for extension header length field (8 bytes).
	ipv6ExtHdrUnitSize = 8

	// Checksum calculation constants.
	ipv6ChecksumByteShift = 8      // Bit shift for high byte in 16-bit checksum word
	ipv6ChecksumWordShift = 16     // Bit shift for folding 32-bit to 16-bit
	ipv6ChecksumWordMask  = 0xffff // Mask for 16-bit value in checksum fold
	ipv6ChecksumWordStep  = 2      // Step size for 16-bit word iteration

	// Note: ipv6AddrSize is declared in dhcpv6.go.

	// IPv6 multicast prefix.
	ipv6MulticastPrefix = 0xff // First byte for IPv6 multicast addresses

	// IPv6 version field.
	ipv6Version = 6 // IPv6 version

	// IPv6NextHeaderHopByHop is the hop-by-hop options header.
	IPv6NextHeaderHopByHop = 0
	// IPv6NextHeaderTCP is the TCP protocol.
	IPv6NextHeaderTCP = 6
	// IPv6NextHeaderUDP is the UDP protocol.
	IPv6NextHeaderUDP = 17
	// IPv6NextHeaderRouting is the routing header.
	IPv6NextHeaderRouting = 43
	// IPv6NextHeaderFragment is the fragment header.
	IPv6NextHeaderFragment = 44
	// IPv6NextHeaderESP is the encapsulating security payload.
	IPv6NextHeaderESP = 50
	// IPv6NextHeaderAH is the authentication header.
	IPv6NextHeaderAH = 51
	// IPv6NextHeaderICMPv6 is the ICMPv6 protocol.
	IPv6NextHeaderICMPv6 = 58
	// IPv6NextHeaderNoNext is the no next header marker.
	IPv6NextHeaderNoNext = 59
	// IPv6NextHeaderDestOptions is the destination options header.
	IPv6NextHeaderDestOptions = 60
)

// GetAllNodesMulticast returns the IPv6 all nodes multicast address.
func GetAllNodesMulticast() net.IP {
	return net.ParseIP("ff02::1")
}

// GetAllRoutersMulticast returns the IPv6 all routers multicast address.
func GetAllRoutersMulticast() net.IP {
	return net.ParseIP("ff02::2")
}

// IPv6Handler handles IPv6 packets.
type IPv6Handler struct {
	stack      *Stack
	debugLevel atomic.Int32
}

// NewIPv6Handler creates a new IPv6 handler.
func NewIPv6Handler(stack *Stack, debugLevel int) *IPv6Handler {
	h := &IPv6Handler{
		stack: stack,
	}
	h.debugLevel.Store(safeconv.Int32(debugLevel))
	return h
}

// HandlePacket processes an incoming IPv6 packet.
func (h *IPv6Handler) HandlePacket(pkt *Packet) {
	packet := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)

	ipv6 := h.parseIPv6Layer(packet, pkt.SerialNumber)
	if ipv6 == nil {
		return
	}

	if h.debugLevel.Load() >= int32(DebugLevelVerbose) {
		_, _ = fmt.Fprintf(os.Stdout, "IPv6: %s -> %s, Next Header: %d, Hop Limit: %d sn=%d\n",
			ipv6.SrcIP, ipv6.DstIP, ipv6.NextHeader, ipv6.HopLimit, pkt.SerialNumber)
	}

	devices := h.getIPv6TargetDevices(ipv6, pkt.SerialNumber)
	if devices == nil && !IsIPv6Multicast(ipv6.DstIP) {
		return
	}

	nextHeader, offset := h.walkExtensionHeaders(packet, ipv6)

	if h.debugLevel.Load() >= int32(DebugLevelVerbose) {
		_, _ = fmt.Fprintf(os.Stdout, "IPv6: Final protocol after extension headers: %d at offset %d sn=%d\n",
			nextHeader, offset, pkt.SerialNumber)
	}

	h.routeIPv6Protocol(pkt, packet, ipv6, devices, nextHeader)
}

// parseIPv6Layer extracts the IPv6 layer from a packet.
func (h *IPv6Handler) parseIPv6Layer(packet gopacket.Packet, serialNum int) *layers.IPv6 {
	ipv6Layer := packet.Layer(layers.LayerTypeIPv6)
	if ipv6Layer == nil {
		if h.debugLevel.Load() >= int32(DebugLevelInfo) {
			_, _ = fmt.Fprintf(os.Stdout, "IPv6 packet missing IPv6 layer sn=%d\n", serialNum)
		}

		return nil
	}

	ipv6, ok := ipv6Layer.(*layers.IPv6)
	if !ok {
		return nil
	}

	return ipv6
}

// getIPv6TargetDevices returns devices that should receive this packet, or nil if not for us.
func (h *IPv6Handler) getIPv6TargetDevices(ipv6 *layers.IPv6, serialNum int) []*config.Device {
	devices := h.stack.GetDevices().GetByIP(ipv6.DstIP)
	if len(devices) > 0 {
		return devices
	}

	if h.debugLevel.Load() >= int32(DebugLevelVerbose) {
		_, _ = fmt.Fprintf(os.Stdout, "IPv6 packet not for our devices: %s sn=%d\n", ipv6.DstIP, serialNum)
	}

	return nil
}

// routeIPv6Protocol dispatches the packet to the appropriate protocol handler.
func (h *IPv6Handler) routeIPv6Protocol(
	pkt *Packet,
	packet gopacket.Packet,
	ipv6 *layers.IPv6,
	devices []*config.Device,
	nextHeader layers.IPProtocol,
) {
	//exhaustive:ignore
	switch nextHeader {
	case layers.IPProtocolICMPv6:
		if h.stack.icmpv6Handler != nil {
			h.stack.icmpv6Handler.HandlePacket(pkt, packet, ipv6, devices)
		}
	case layers.IPProtocolUDP:
		if h.stack.udpHandler != nil {
			h.stack.udpHandler.HandlePacketV6(pkt, packet, ipv6, devices)
		}
	case layers.IPProtocolTCP:
		if h.stack.tcpHandler != nil {
			h.stack.tcpHandler.HandlePacketV6(pkt, packet, ipv6, devices)
		}
	case IPv6NextHeaderNoNext:
		if h.debugLevel.Load() >= int32(DebugLevelInfo) {
			_, _ = fmt.Fprintf(os.Stdout, "IPv6: No next header, packet complete sn=%d\n", pkt.SerialNumber)
		}
	default:
		if h.debugLevel.Load() >= int32(DebugLevelInfo) {
			_, _ = fmt.Fprintf(os.Stdout, "IPv6: Unhandled next header protocol: %d sn=%d\n",
				nextHeader, pkt.SerialNumber)
		}
	}
}

// walkExtensionHeaders walks through IPv6 extension headers to find the final protocol
// Returns the final next header value and its offset in the packet.
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
	//exhaustive:ignore
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
		extHeaderSize := (hdrExtLen + 1) * ipv6ExtHdrUnitSize
		offset += extHeaderSize

		if h.debugLevel.Load() >= int32(DebugLevelVerbose) {
			_, _ = fmt.Fprintf(os.Stdout, "IPv6: Processed extension header, next: %d, length: %d bytes\n",
				nextHeader, extHeaderSize)
		}
	}

	return nextHeader, offset
}

// IPv6MulticastToMAC converts an IPv6 multicast address to an Ethernet multicast MAC
// Per RFC 2464: 33:33 followed by the last 4 bytes of the IPv6 address.
func IPv6MulticastToMAC(ipv6 net.IP) net.HardwareAddr {
	// Ensure we have a 16-byte IPv6 address
	if len(ipv6) == ipv6AddrSize {
		// Create MAC with initial multicast prefix 33:33
		// then copy last 4 bytes of IPv6 address
		mac := net.HardwareAddr{0x33, 0x33, ipv6[12], ipv6[13], ipv6[14], ipv6[15]}

		return mac
	}

	return nil
}

// IsIPv6Multicast checks if an IPv6 address is multicast (starts with ff).
func IsIPv6Multicast(ipv6 net.IP) bool {
	if len(ipv6) == ipv6AddrSize {
		return ipv6[0] == ipv6MulticastPrefix
	}

	return false
}

// CalculateIPv6Checksum calculates the checksum for IPv6 upper-layer protocols
// Uses the IPv6 pseudo-header per RFC 2460.
func CalculateIPv6Checksum(srcIP, dstIP net.IP, nextHeader uint8, payload []byte) uint16 {
	// IPv6 pseudo-header:
	// - Source Address (16 bytes)
	// - Destination Address (16 bytes)
	// - Upper-Layer Packet Length (4 bytes)
	// - Zero (3 bytes)
	// - Next Header (1 byte)
	pseudoHeader := make([]byte, ipv6PseudoHeaderSize)

	// Source IP
	copy(pseudoHeader[0:16], srcIP.To16())

	// Destination IP
	copy(pseudoHeader[16:32], dstIP.To16())

	// Upper-layer packet length (32-bit)
	payloadLen2 := min(len(payload), ipv6MaxUint32)

	binary.BigEndian.PutUint32(pseudoHeader[32:36], safeconv.Uint32(payloadLen2))

	// Zero padding (3 bytes) at 36:39

	// Next header
	pseudoHeader[39] = nextHeader

	// Calculate checksum over pseudo-header + payload
	sum := uint32(0)

	// Sum pseudo-header
	for i := 0; i < len(pseudoHeader); i += ipv6ChecksumWordStep {
		sum += ipv6SumWord(pseudoHeader, i)
	}

	// Sum payload
	for i := 0; i < len(payload)-1; i += ipv6ChecksumWordStep {
		sum += uint32(payload[i])<<ipv6ChecksumByteShift | uint32(payload[i+1])
	}

	// Handle odd-length payload
	if len(payload)%ipv6ChecksumWordStep == 1 {
		sum += uint32(payload[len(payload)-1]) << ipv6ChecksumByteShift
	}

	// Fold 32-bit sum to 16 bits
	for sum > ipv6ChecksumWordMask {
		sum = (sum >> ipv6ChecksumWordShift) + (sum & ipv6ChecksumWordMask)
	}

	// Return one's complement
	return ^uint16(sum)
}

// ipv6SumWord returns the big-endian 16-bit word at offset i in buf, widened to uint32.
// Callers must ensure len(buf) is even and i+1 is within bounds.
func ipv6SumWord(buf []byte, i int) uint32 {
	return uint32(buf[i])<<ipv6ChecksumByteShift | uint32(buf[i+1])
}

// SendIPv6Packet constructs and sends an IPv6 packet.
func (h *IPv6Handler) SendIPv6Packet(srcIP, dstIP net.IP, srcMAC, dstMAC net.HardwareAddr,
	nextHeader layers.IPProtocol, hopLimit uint8, payload []byte,
) error {
	// Build Ethernet layer
	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv6,
	}

	// Build IPv6 layer
	payloadLen := min(len(payload), ipv6MaxUint16)

	ipv6 := &layers.IPv6{
		Version:      ipv6Version,
		TrafficClass: 0,
		FlowLabel:    0,
		Length:       safeconv.Uint16(payloadLen),
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

	if h.debugLevel.Load() >= int32(DebugLevelVerbose) {
		_, _ = fmt.Fprintf(os.Stdout, "IPv6: Sending packet %s -> %s, protocol: %d, size: %d bytes\n",
			srcIP, dstIP, nextHeader, len(buf.Bytes()))
	}

	// Send the packet
	return h.stack.SendRawPacket(buf.Bytes())
}

// SetDebugLevel updates the debug level.
func (h *IPv6Handler) SetDebugLevel(level int) {
	h.debugLevel.Store(safeconv.Int32(level))
}
