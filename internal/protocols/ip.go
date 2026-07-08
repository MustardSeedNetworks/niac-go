package protocols

import (
	"fmt"
	"net"
	"os"
	"slices"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// IP protocol numbers.
const (
	IPProtocolICMP = 1
	IPProtocolTCP  = 6
	IPProtocolUDP  = 17
)

// IPv4 header field constants.
const (
	ipIPv4Version = 4  // IPv4 version field
	ipIPv4IHL     = 5  // IPv4 header length (5 = 20 bytes)
	ipIPv4TTL     = 64 // IPv4 default TTL
)

// IPHandler handles IP packets.
type IPHandler struct {
	stack *Stack
}

// NewIPHandler creates a new IP handler.
func NewIPHandler(stack *Stack) *IPHandler {
	return &IPHandler{
		stack: stack,
	}
}

// HandlePacket processes an IP packet.
func (h *IPHandler) HandlePacket(pkt *Packet) {
	debugLevel := h.stack.GetDebugLevel()

	ip := h.parseIPv4Layer(pkt, debugLevel)
	if ip == nil {
		return
	}

	if debugLevel >= DebugLevelVerbose {
		_, _ = fmt.Fprintf(os.Stdout, "IP packet: %s -> %s protocol=%d sn=%d\n",
			ip.SrcIP, ip.DstIP, ip.Protocol, pkt.SerialNumber)
	}

	devices := h.getTargetDevices(ip, pkt.SerialNumber, debugLevel, pkt.VLAN)
	if devices == nil {
		return
	}

	if h.shouldProcessTTL(ip, devices) && h.handleTTLTimeout(pkt, ip) {
		return
	}

	h.routeToProtocolHandler(pkt, ip, devices, debugLevel)
}

// parseIPv4Layer extracts the IPv4 layer from a packet.
func (h *IPHandler) parseIPv4Layer(pkt *Packet, debugLevel int) *layers.IPv4 {
	packet := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)

	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "IP packet missing IPv4 layer sn=%d\n", pkt.SerialNumber)
		}

		return nil
	}

	ip, ok := ipLayer.(*layers.IPv4)
	if !ok {
		return nil
	}

	return ip
}

// getTargetDevices returns devices that should receive this packet, scoped to
// the VLAN segment (if any) the packet arrived on.
func (h *IPHandler) getTargetDevices(
	ip *layers.IPv4,
	serialNum int,
	debugLevel int,
	vlan int,
) []*config.Device {
	isBroadcast := ip.DstIP.Equal([]byte{255, 255, 255, 255})
	devices := h.stack.devicesFor(vlan).GetByIP(ip.DstIP)

	if len(devices) == 0 && !isBroadcast {
		if debugLevel >= DebugLevelVerbose {
			_, _ = fmt.Fprintf(os.Stdout, "IP packet not for our devices: %s sn=%d\n", ip.DstIP, serialNum)
		}

		return nil
	}

	if isBroadcast && len(devices) == 0 {
		devices = h.stack.devicesFor(vlan).GetAll()
	}

	return devices
}

// shouldProcessTTL determines if TTL handling should be applied.
func (h *IPHandler) shouldProcessTTL(ip *layers.IPv4, devices []*config.Device) bool {
	if ip.Protocol != IPProtocolUDP {
		return true
	}

	return !slices.ContainsFunc(devices, func(device *config.Device) bool {
		return device.MapToIP != nil
	})
}

// routeToProtocolHandler dispatches the packet to the appropriate L4 handler.
func (h *IPHandler) routeToProtocolHandler(pkt *Packet, ip *layers.IPv4, devices []*config.Device, debugLevel int) {
	//exhaustive:ignore
	switch ip.Protocol {
	case IPProtocolICMP:
		h.stack.icmpHandler.HandlePacket(pkt, ip, devices)
	case IPProtocolUDP:
		h.stack.udpHandler.HandlePacket(pkt, ip, devices)
	case IPProtocolTCP:
		h.stack.tcpHandler.HandlePacket(pkt, ip, devices)
	default:
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "Unhandled IP protocol %d sn=%d\n", ip.Protocol, pkt.SerialNumber)
		}
	}
}

// isDestInTTLSubnet checks if the destination IP is in the device's TTL subnet.
func isDestInTTLSubnet(device *config.Device, dstIP net.IP) bool {
	if device.TTLConfig == nil || device.TTLConfig.IP == nil || device.TTLConfig.Mask == nil {
		return false
	}

	dst := dstIP.To4()
	ttlIP := device.TTLConfig.IP.To4()

	if dst == nil || ttlIP == nil {
		return false
	}

	for i := range 4 {
		if (dst[i] & device.TTLConfig.Mask[i]) != ttlIP[i] {
			return false
		}
	}

	return true
}

func (h *IPHandler) handleTTLTimeout(pkt *Packet, ipLayer *layers.IPv4) bool {
	if h.stack == nil || h.stack.icmpHandler == nil {
		return false
	}

	if ipLayer == nil {
		return false
	}

	ttl := int(ipLayer.TTL)

	device := h.stack.devicesFor(pkt.VLAN).GetDeviceByTTL(ttl)
	if device == nil {
		return false
	}

	srcIP := firstIPv4Address(device)
	if srcIP == nil || len(device.MACAddress) == 0 {
		return false
	}

	// If the TTL device is the destination, ignore.
	for _, ip := range device.IPAddresses {
		if ip.Equal(ipLayer.DstIP) {
			return false
		}
	}

	// Skip if destination is in TTL subnet
	if isDestInTTLSubnet(device, ipLayer.DstIP) {
		return false
	}

	dstMAC := pkt.GetSourceMAC()
	if dstMAC == nil {
		return false
	}

	err := h.stack.icmpHandler.SendICMPTimeExceeded(
		srcIP,
		ipLayer.SrcIP,
		device.MACAddress,
		dstMAC,
		ipLayer,
		device,
		pkt.VLAN,
	)
	if err != nil {
		return false
	}

	h.stack.devicesFor(pkt.VLAN).IncrementTTLCount(device)

	return true
}

// SendIPPacket sends an IP packet.
func (h *IPHandler) SendIPPacket(
	srcIP, dstIP net.IP,
	protocol layers.IPProtocol,
	payload []byte,
	srcMAC, dstMAC net.HardwareAddr,
) error {
	// Build Ethernet header
	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}

	// Build IP header
	ipLayer := &layers.IPv4{
		Version:  ipIPv4Version,
		IHL:      ipIPv4IHL,
		TTL:      ipIPv4TTL,
		Protocol: protocol,
		SrcIP:    srcIP,
		DstIP:    dstIP,
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
		gopacket.Payload(payload),
	)
	if err != nil {
		return fmt.Errorf("error serializing IP packet: %w", err)
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

	if h.stack.GetDebugLevel() >= DebugLevelVerbose {
		_, _ = fmt.Fprintf(os.Stdout, "Sent IP packet: %s -> %s protocol=%d length=%d sn=%d\n",
			srcIP, dstIP, protocol, len(payload), serialNum)
	}

	return nil
}
