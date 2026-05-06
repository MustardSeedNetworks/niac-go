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

// ICMPv6 message type constants.
const (
	// ICMPv6TypeDestUnreachable is the destination unreachable error type.
	ICMPv6TypeDestUnreachable = 1
	// ICMPv6TypePacketTooBig is the packet too big error type.
	ICMPv6TypePacketTooBig = 2
	// ICMPv6TypeTimeExceeded is the time exceeded error type.
	ICMPv6TypeTimeExceeded = 3
	// ICMPv6TypeParameterProblem is the parameter problem error type.
	ICMPv6TypeParameterProblem = 4

	// ICMPv6TypeEchoRequest is the echo request (ping) type.
	ICMPv6TypeEchoRequest = 128
	// ICMPv6TypeEchoReply is the echo reply (pong) type.
	ICMPv6TypeEchoReply = 129
	// ICMPv6TypeRouterSolicitation is the router solicitation type.
	ICMPv6TypeRouterSolicitation = 133
	// ICMPv6TypeRouterAdvertisement is the router advertisement type.
	ICMPv6TypeRouterAdvertisement = 134
	// ICMPv6TypeNeighborSolicitation is the neighbor solicitation type.
	ICMPv6TypeNeighborSolicitation = 135
	// ICMPv6TypeNeighborAdvertisement is the neighbor advertisement type.
	ICMPv6TypeNeighborAdvertisement = 136
	// ICMPv6TypeRedirect is the redirect type.
	ICMPv6TypeRedirect = 137
)

// ICMPv6 option types.
const (
	ICMPv6OptSourceLinkAddr = 1
	ICMPv6OptTargetLinkAddr = 2
	ICMPv6OptPrefixInfo     = 3
	ICMPv6OptRedirectedHdr  = 4
	ICMPv6OptMTU            = 5
)

// ICMPv6 Neighbor Discovery flags.
const (
	NDFlagRouter    = 0x80
	NDFlagSolicited = 0x40
	NDFlagOverride  = 0x20
)

// Neighbor Discovery prefix flags.
const (
	NDPrefixFlagOnLink     = 0x80 // Prefix can be used for on-link determination
	NDPrefixFlagAutonomous = 0x40 // Prefix can be used for autonomous address configuration
)

// ICMPv6 encoding constants.
const (
	icmpv6IPv6Version       = 6       // IPv6 version field
	icmpv6NAPayloadSize     = 32      // NA payload header (4 header + 4 flags/reserved + 16 target + 8 option)
	icmpv6DefaultHopLimit   = 64      // Default IPv6 hop limit
	icmpv6RABodyHeaderSize  = 12      // RA body header size
	icmpv6DefaultRALifetime = 1800    // Default RA lifetime in seconds (30 min)
	icmpv6OptionsCapacity   = 32      // Options initial capacity
	icmpv6DefaultMTU        = 1500    // Default MTU value
	icmpv6Uint32Size        = 4       // Size of uint32 in bytes
	icmpv6PrefixInfoOptLen  = 4       // Prefix info option length in 8-byte units
	icmpv6DefaultPrefixLen  = 64      // Default prefix length in bits
	icmpv6ValidLifetime     = 2592000 // 30 days in seconds
	icmpv6PreferredLifetime = 604800  // 7 days in seconds
	icmpv6NDPHopLimit       = 255     // NDP hop limit per RFC 4861
	icmpv6HeaderSize        = 8       // ICMPv6 header size
	icmpv6IPv6AddrBits      = 128     // IPv6 address length in bits
	icmpv6MaxUint16         = 65535   // Max uint16 for length fields
	icmpv6NSMinSize         = 20      // NS minimum size (4 reserved + 16 target address)
)

// ICMPv6Handler handles ICMPv6 packets (IPv6's version of ICMP).
type ICMPv6Handler struct {
	stack      *Stack
	debugLevel atomic.Int32
}

// NewICMPv6Handler creates a new ICMPv6 handler.
func NewICMPv6Handler(stack *Stack, debugLevel int) *ICMPv6Handler {
	h := &ICMPv6Handler{
		stack: stack,
	}
	h.debugLevel.Store(safeconv.Int32(debugLevel))
	return h
}

// HandlePacket processes an incoming ICMPv6 packet.
func (h *ICMPv6Handler) HandlePacket(
	pkt *Packet,
	packet gopacket.Packet,
	ipv6Layer *layers.IPv6,
	devices []*config.Device,
) {
	icmpv6Layer := packet.Layer(layers.LayerTypeICMPv6)
	if icmpv6Layer == nil {
		if h.debugLevel.Load() >= int32(DebugLevelInfo) {
			_, _ = fmt.Fprintf(os.Stdout, "ICMPv6 packet missing ICMPv6 layer sn=%d\n", pkt.SerialNumber)
		}

		return
	}

	icmpv6, ok := icmpv6Layer.(*layers.ICMPv6)
	if !ok {
		return
	}

	msgType := icmpv6.TypeCode.Type()

	if h.debugLevel.Load() >= int32(DebugLevelVerbose) {
		_, _ = fmt.Fprintf(os.Stdout, "ICMPv6: type %d (%s) sn=%d\n", msgType, h.getTypeName(msgType), pkt.SerialNumber)
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
		if h.debugLevel.Load() >= int32(DebugLevelInfo) {
			_, _ = fmt.Fprintf(os.Stdout, "ICMPv6: Unhandled type: %d sn=%d\n", msgType, pkt.SerialNumber)
		}
	}
}

// handleEchoRequest responds to ICMPv6 Echo Request (ping6).
func (h *ICMPv6Handler) handleEchoRequest(
	pkt *Packet,
	packet gopacket.Packet,
	ipv6 *layers.IPv6,
	icmpv6 *layers.ICMPv6,
	devices []*config.Device,
) {
	if len(devices) == 0 {
		h.logNoDeviceForEcho(ipv6, pkt)
		return
	}

	h.logEchoRequest(ipv6, pkt)

	eth := h.extractEthernetLayer(packet)
	if eth == nil {
		return
	}

	for _, device := range devices {
		h.sendEchoReply(pkt, device, ipv6, icmpv6, eth)
	}
}

// logNoDeviceForEcho logs when no device is found for an echo request.
func (h *ICMPv6Handler) logNoDeviceForEcho(ipv6 *layers.IPv6, pkt *Packet) {
	if h.debugLevel.Load() >= int32(DebugLevelVerbose) {
		_, _ = fmt.Fprintf(
			os.Stdout,
			"ICMPv6: No device for Echo Request to %s sn=%d\n",
			ipv6.DstIP,
			pkt.SerialNumber,
		)
	}
}

// logEchoRequest logs an incoming echo request.
func (h *ICMPv6Handler) logEchoRequest(ipv6 *layers.IPv6, pkt *Packet) {
	if h.debugLevel.Load() >= int32(DebugLevelInfo) {
		_, _ = fmt.Fprintf(os.Stdout, "ICMPv6: Echo Request %s -> %s sn=%d\n", ipv6.SrcIP, ipv6.DstIP, pkt.SerialNumber)
	}
}

// extractEthernetLayer extracts the Ethernet layer from a packet.
func (h *ICMPv6Handler) extractEthernetLayer(packet gopacket.Packet) *layers.Ethernet {
	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	if ethLayer == nil {
		return nil
	}

	eth, ok := ethLayer.(*layers.Ethernet)
	if !ok {
		return nil
	}

	return eth
}

// sendEchoReply sends an ICMPv6 echo reply from a device.
func (h *ICMPv6Handler) sendEchoReply(
	pkt *Packet,
	device *config.Device,
	ipv6 *layers.IPv6,
	icmpv6 *layers.ICMPv6,
	eth *layers.Ethernet,
) {
	if len(device.MACAddress) == 0 {
		return
	}

	if !h.deviceHasIP(device, ipv6.DstIP) {
		return
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
		device,
	)
	if err != nil {
		if h.debugLevel.Load() >= int32(DebugLevelInfo) {
			_, _ = fmt.Fprintf(os.Stdout, "ICMPv6: Error sending Echo Reply sn=%d: %v\n", pkt.SerialNumber, err)
		}

		return
	}

	h.stack.IncrementStat("icmp_replies")
	h.logEchoReplySent(ipv6, pkt)
}

// deviceHasIP checks if a device has a specific IP address.
func (h *ICMPv6Handler) deviceHasIP(device *config.Device, ip net.IP) bool {
	for _, deviceIP := range device.IPAddresses {
		if deviceIP.Equal(ip) {
			return true
		}
	}

	return false
}

// logEchoReplySent logs when an echo reply is sent.
func (h *ICMPv6Handler) logEchoReplySent(ipv6 *layers.IPv6, pkt *Packet) {
	if h.debugLevel.Load() >= int32(DebugLevelInfo) {
		_, _ = fmt.Fprintf(
			os.Stdout,
			"ICMPv6: Sent Echo Reply %s -> %s sn=%d\n",
			ipv6.DstIP,
			ipv6.SrcIP,
			pkt.SerialNumber,
		)
	}
}

// handleNeighborSolicitation responds to Neighbor Solicitation (NDP - like ARP for IPv6).
func (h *ICMPv6Handler) handleNeighborSolicitation(pkt *Packet, packet gopacket.Packet, ipv6 *layers.IPv6) {
	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	if ethLayer == nil {
		return
	}

	eth, ok := ethLayer.(*layers.Ethernet)
	if !ok {
		return
	}

	// Parse NS message: Type(1) | Code(1) | Checksum(2) | Reserved(4) | Target Address(16) | Options...
	appLayer := packet.ApplicationLayer()
	if appLayer == nil {
		return
	}
	data := appLayer.Payload()
	if len(data) < icmpv6NSMinSize {
		if h.debugLevel.Load() >= int32(DebugLevelInfo) {
			_, _ = fmt.Fprintf(os.Stdout, "ICMPv6: NS too short sn=%d\n", pkt.SerialNumber)
		}

		return
	}

	// Extract target IPv6 address (bytes 4-20)
	targetIP := net.IP(data[4:20])

	if h.debugLevel.Load() >= int32(DebugLevelInfo) {
		_, _ = fmt.Fprintf(os.Stdout, "ICMPv6: NS for %s from %s sn=%d\n", targetIP, ipv6.SrcIP, pkt.SerialNumber)
	}

	// Find device with target IPv6
	devices := h.stack.devices.GetByIPv6(targetIP)
	if len(devices) == 0 {
		if h.debugLevel.Load() >= int32(DebugLevelVerbose) {
			_, _ = fmt.Fprintf(os.Stdout, "ICMPv6: No device for target %s sn=%d\n", targetIP, pkt.SerialNumber)
		}

		return
	}

	// Send NA for each matching device
	for _, device := range devices {
		err := h.sendNeighborAdvertisement(device, ipv6.SrcIP, eth.SrcMAC, targetIP)
		if err != nil {
			if h.debugLevel.Load() >= int32(DebugLevelInfo) {
				_, _ = fmt.Fprintf(os.Stdout, "ICMPv6: Error sending NA sn=%d: %v\n", pkt.SerialNumber, err)
			}

			continue
		}

		if h.debugLevel.Load() >= int32(DebugLevelInfo) {
			_, _ = fmt.Fprintf(
				os.Stdout,
				"ICMPv6: Sent NA for %s (MAC: %s) sn=%d\n",
				targetIP,
				device.MACAddress,
				pkt.SerialNumber,
			)
		}
	}
}

// sendNeighborAdvertisement sends a Neighbor Advertisement response.
func (h *ICMPv6Handler) sendNeighborAdvertisement(device *config.Device, dstIP net.IP,
	dstMAC net.HardwareAddr, targetIP net.IP,
) error {
	// Require a source IPv6 and MAC on the device.
	srcIP := firstIPv6Address(device)
	if srcIP == nil || len(device.MACAddress) == 0 {
		return ErrDeviceMissingMACOrIP
	}

	// Build Neighbor Advertisement message
	// Format: Type(1) | Code(1) | Checksum(2) | Flags(1) | Reserved(3) | Target Address(16) | Options...
	payloadHeader := make([]byte, icmpv6NAPayloadSize) // 4 header + 4 flags/reserved + 16 target + 8 option bytes

	// Type = 136 (Neighbor Advertisement)
	payloadHeader[0] = ICMPv6TypeNeighborAdvertisement

	// Code = 0
	payloadHeader[1] = 0

	// Checksum = 0 (will be calculated)
	payloadHeader[2] = 0
	payloadHeader[3] = 0

	// Flags: Solicited + Override
	payloadHeader[4] = NDFlagSolicited | NDFlagOverride

	// Reserved (3 bytes)
	payloadHeader[5] = 0
	payloadHeader[6] = 0
	payloadHeader[7] = 0

	// Target Address (16 bytes)
	copy(payloadHeader[8:24], targetIP.To16())

	// Option: Target Link-Layer Address
	// Type(1) | Length(1) | Link-Layer Address(6)
	payloadHeader[24] = ICMPv6OptTargetLinkAddr // Type
	payloadHeader[25] = 1                       // Length (in units of 8 bytes)
	copy(payloadHeader[26:], device.MACAddress) // MAC address (6 bytes)

	// Calculate ICMPv6 checksum
	checksum := CalculateIPv6Checksum(srcIP, dstIP, IPv6NextHeaderICMPv6, payloadHeader)
	binary.BigEndian.PutUint16(payloadHeader[2:4], checksum)

	// Build ICMPv6 layer
	icmpv6 := &layers.ICMPv6{
		TypeCode: layers.CreateICMPv6TypeCode(ICMPv6TypeNeighborAdvertisement, 0),
	}

	// Send packet
	return h.sendICMPv6Packet(
		srcIP,
		dstIP,
		device.MACAddress,
		dstMAC,
		icmpv6,
		payloadHeader[4:], // Skip type, code, checksum
	)
}

// handleRouterSolicitation responds to Router Solicitation messages.
func (h *ICMPv6Handler) handleRouterSolicitation(pkt *Packet, packet gopacket.Packet, ipv6 *layers.IPv6) {
	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	if ethLayer == nil {
		return
	}

	eth, ok := ethLayer.(*layers.Ethernet)
	if !ok {
		return
	}

	dstMAC := eth.SrcMAC

	if h.debugLevel.Load() >= int32(DebugLevelInfo) {
		_, _ = fmt.Fprintf(os.Stdout, "ICMPv6: Router Solicitation from %s sn=%d\n", ipv6.SrcIP, pkt.SerialNumber)
	}

	for _, device := range h.stack.devices.GetAll() {
		if !deviceCanAdvertiseIPv6(device) {
			continue
		}

		srcIP := firstIPv6Address(device)
		if srcIP == nil {
			continue
		}
		err := h.sendRouterAdvertisement(device, srcIP, ipv6.SrcIP, dstMAC)
		if err != nil {
			if h.debugLevel.Load() >= int32(DebugLevelInfo) {
				_, _ = fmt.Fprintf(
					os.Stdout,
					"ICMPv6: Failed to send RA from %s: %v sn=%d\n",
					device.Name,
					err,
					pkt.SerialNumber,
				)
			}

			continue
		}

		if h.debugLevel.Load() >= int32(DebugLevelInfo) {
			_, _ = fmt.Fprintf(os.Stdout,
				"ICMPv6: Sent Router Advertisement from %s to %s sn=%d\n",
				device.Name,
				ipv6.SrcIP,
				pkt.SerialNumber,
			)
		}
	}
}

func (h *ICMPv6Handler) sendRouterAdvertisement(
	device *config.Device,
	srcIP, dstIP net.IP,
	dstMAC net.HardwareAddr,
) error {
	body := h.buildRouterAdvertisementBody(device, srcIP)
	icmpv6 := &layers.ICMPv6{
		TypeCode: layers.CreateICMPv6TypeCode(ICMPv6TypeRouterAdvertisement, 0),
	}

	return h.sendICMPv6PacketWithDevice(
		srcIP,
		dstIP,
		device.MACAddress,
		dstMAC,
		icmpv6,
		body,
		device,
	)
}

func (h *ICMPv6Handler) buildRouterAdvertisementBody(device *config.Device, srcIP net.IP) []byte {
	raCfg := getRAConfig(device)
	bodyHeader := buildRAHeader(device, raCfg)
	options := buildRAOptions(device, raCfg, srcIP)

	payload := make([]byte, len(bodyHeader)+len(options))
	copy(payload, bodyHeader)
	copy(payload[len(bodyHeader):], options)

	return payload
}

// getRAConfig returns the router advertisement config for a device.
func getRAConfig(device *config.Device) *config.Icmpv6RouterAdvertisement {
	if device == nil || device.ICMPv6Config == nil {
		return nil
	}

	return device.ICMPv6Config.RouterAdvertisement
}

// buildRAHeader builds the router advertisement body header.
func buildRAHeader(device *config.Device, raCfg *config.Icmpv6RouterAdvertisement) []byte {
	hopLimit := getRAHopLimit(device, raCfg)
	flags := getRAFlags(raCfg)
	lifetime := getRALifetime(raCfg)
	reachable, retrans := getRATimers(raCfg)

	bodyHeader := make([]byte, icmpv6RABodyHeaderSize)
	bodyHeader[0] = hopLimit
	bodyHeader[1] = flags
	binary.BigEndian.PutUint16(bodyHeader[2:4], lifetime)
	binary.BigEndian.PutUint32(bodyHeader[4:8], reachable)
	binary.BigEndian.PutUint32(bodyHeader[8:12], retrans)

	return bodyHeader
}

func getRAHopLimit(device *config.Device, raCfg *config.Icmpv6RouterAdvertisement) uint8 {
	hopLimit := uint8(icmpv6DefaultHopLimit)
	if device != nil && device.ICMPv6Config != nil && device.ICMPv6Config.HopLimit > 0 {
		hopLimit = device.ICMPv6Config.HopLimit
	}

	if raCfg != nil && raCfg.CurHopLimit > 0 {
		hopLimit = safeconv.Uint8(raCfg.CurHopLimit)
	}

	return hopLimit
}

func getRAFlags(raCfg *config.Icmpv6RouterAdvertisement) byte {
	if raCfg == nil {
		return 0
	}

	flags := byte(0)
	if raCfg.Managed != 0 {
		flags |= 0x80
	}

	if raCfg.Other != 0 {
		flags |= 0x40
	}

	return flags
}

func getRALifetime(raCfg *config.Icmpv6RouterAdvertisement) uint16 {
	if raCfg != nil && raCfg.Lifetime > 0 {
		return safeconv.Uint16(raCfg.Lifetime)
	}

	return uint16(icmpv6DefaultRALifetime)
}

func getRATimers(raCfg *config.Icmpv6RouterAdvertisement) (uint32, uint32) {
	if raCfg == nil {
		return 0, 0
	}

	return safeconv.Uint32(raCfg.ReachableTime), safeconv.Uint32(raCfg.RetransTimer)
}

// buildRAOptions builds the router advertisement options.
func buildRAOptions(device *config.Device, raCfg *config.Icmpv6RouterAdvertisement, srcIP net.IP) []byte {
	options := make([]byte, 0, icmpv6OptionsCapacity)

	// Source link-layer option
	options = append(options, ICMPv6OptSourceLinkAddr, 1)
	options = append(options, device.MACAddress...)

	// MTU option
	options = appendMTUOption(options, raCfg)

	// Prefix options
	options = appendPrefixOptions(options, raCfg, srcIP)

	return options
}

func appendMTUOption(options []byte, raCfg *config.Icmpv6RouterAdvertisement) []byte {
	mtuVal := uint32(icmpv6DefaultMTU)
	if raCfg != nil && raCfg.MTU > 0 {
		mtuVal = safeconv.Uint32(raCfg.MTU)
	}

	options = append(options, ICMPv6OptMTU, 1, 0, 0)
	mtu := make([]byte, icmpv6Uint32Size)
	binary.BigEndian.PutUint32(mtu, mtuVal)

	return append(options, mtu...)
}

func appendPrefixOptions(options []byte, raCfg *config.Icmpv6RouterAdvertisement, srcIP net.IP) []byte {
	if raCfg != nil && len(raCfg.PrefixInfo) > 0 {
		return appendConfiguredPrefixes(options, raCfg.PrefixInfo)
	}

	return appendDefaultPrefix(options, srcIP)
}

func appendConfiguredPrefixes(options []byte, prefixes []config.Icmpv6PrefixInfo) []byte {
	for _, p := range prefixes {
		if p.Prefix == nil || p.Prefix.To16() == nil {
			continue
		}

		options = appendSinglePrefix(options, &p)
	}

	return options
}

func appendSinglePrefix(options []byte, p *config.Icmpv6PrefixInfo) []byte {
	options = append(options, ICMPv6OptPrefixInfo, icmpv6PrefixInfoOptLen)
	options = append(options, safeconv.Byte(p.PrefixLength))

	pFlags := byte(0)
	if p.Onlink != 0 {
		pFlags |= NDPrefixFlagOnLink
	}

	if p.Auto != 0 {
		pFlags |= NDPrefixFlagAutonomous
	}

	options = append(options, pFlags)

	valid := make([]byte, icmpv6Uint32Size)
	binary.BigEndian.PutUint32(valid, safeconv.Uint32(p.ValidLifetime))
	options = append(options, valid...)
	binary.BigEndian.PutUint32(valid, safeconv.Uint32(p.PreferredLifetime))
	options = append(options, valid...)
	options = append(options, []byte{0, 0, 0, 0}...)

	return append(options, p.Prefix.To16()...)
}

func appendDefaultPrefix(options []byte, srcIP net.IP) []byte {
	prefix := deriveIPv6Prefix(srcIP, icmpv6DefaultPrefixLen)

	options = append(options, ICMPv6OptPrefixInfo, icmpv6PrefixInfoOptLen)
	options = append(options, byte(icmpv6DefaultPrefixLen))
	options = append(options, NDPrefixFlagOnLink|NDPrefixFlagAutonomous)

	valid := make([]byte, icmpv6Uint32Size)
	binary.BigEndian.PutUint32(valid, icmpv6ValidLifetime)
	options = append(options, valid...)
	binary.BigEndian.PutUint32(valid, icmpv6PreferredLifetime)
	options = append(options, valid...)
	options = append(options, []byte{0, 0, 0, 0}...)

	return append(options, prefix.To16()...)
}

// sendICMPv6Packet sends an ICMPv6 packet.
func (h *ICMPv6Handler) sendICMPv6Packet(srcIP, dstIP net.IP, srcMAC, dstMAC net.HardwareAddr,
	icmpv6 *layers.ICMPv6, payload []byte,
) error {
	return h.sendICMPv6PacketWithDevice(srcIP, dstIP, srcMAC, dstMAC, icmpv6, payload, nil)
}

// sendICMPv6PacketWithDevice sends an ICMPv6 packet with device config.
func (h *ICMPv6Handler) sendICMPv6PacketWithDevice(srcIP, dstIP net.IP, srcMAC, dstMAC net.HardwareAddr,
	icmpv6 *layers.ICMPv6, payload []byte, device *config.Device,
) error {
	// Determine hop limit based on ICMPv6 type
	hopLimit := uint8(icmpv6NDPHopLimit) // Default to 255 for NDP (RFC 4861)
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
	icmpLen := min(
		// ICMPv6 header + payload
		icmpv6HeaderSize+len(payload), icmpv6MaxUint16)

	ipv6 := &layers.IPv6{
		Version:      icmpv6IPv6Version,
		TrafficClass: 0,
		FlowLabel:    0,
		Length:       safeconv.Uint16(icmpLen),
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

	if h.debugLevel.Load() >= int32(DebugLevelVerbose) {
		_, _ = fmt.Fprintf(os.Stdout, "ICMPv6: Sending packet %s -> %s, type %d, size %d bytes\n",
			srcIP, dstIP, icmpv6.TypeCode.Type(), len(buf.Bytes()))
	}

	// Send the packet
	return h.stack.SendRawPacket(buf.Bytes())
}

// getTypeName returns a human-readable name for an ICMPv6 type.
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

// SetDebugLevel updates the debug level.
func (h *ICMPv6Handler) SetDebugLevel(level int) {
	h.debugLevel.Store(safeconv.Int32(level))
}

func deviceCanAdvertiseIPv6(device *config.Device) bool {
	if device == nil {
		return false
	}

	if device.ICMPv6Config == nil || device.ICMPv6Config.RouterAdvertisement == nil {
		return false
	}

	return firstIPv6Address(device) != nil
}

func firstIPv6Address(device *config.Device) net.IP {
	if device == nil {
		return nil
	}

	for _, ip := range device.IPAddresses {
		if ip.To4() == nil && ip.To16() != nil {
			return ip
		}
	}

	return nil
}

func deriveIPv6Prefix(ip net.IP, prefixLen int) net.IP {
	mask := net.CIDRMask(prefixLen, icmpv6IPv6AddrBits)
	masked := ip.Mask(mask)
	prefix := make(net.IP, len(masked))
	copy(prefix, masked)

	return prefix
}
