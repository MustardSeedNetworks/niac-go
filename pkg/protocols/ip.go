package protocols

import (
	"fmt"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// IP protocol numbers
const (
	IPProtocolICMP = 1
	IPProtocolTCP  = 6
	IPProtocolUDP  = 17
)

// IPHandler handles IP packets
type IPHandler struct {
	stack *Stack
}

// NewIPHandler creates a new IP handler
func NewIPHandler(stack *Stack) *IPHandler {
	return &IPHandler{
		stack: stack,
	}
}

// HandlePacket processes an IP packet
func (h *IPHandler) HandlePacket(pkt *Packet) {
	debugLevel := h.stack.GetDebugLevel()

	// Parse using gopacket
	packet := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)

	// Get IP layer
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		if debugLevel >= 2 {
			fmt.Printf("IP packet missing IPv4 layer sn=%d\n", pkt.SerialNumber)
		}
		return
	}

	ip, ok := ipLayer.(*layers.IPv4)
	if !ok {
		return
	}

	if debugLevel >= 3 {
		fmt.Printf("IP packet: %s -> %s protocol=%d sn=%d\n",
			ip.SrcIP, ip.DstIP, ip.Protocol, pkt.SerialNumber)
	}

	// Check if packet is for one of our devices
	// Also accept broadcast packets (255.255.255.255) for DHCP and other broadcast protocols
	isBroadcast := ip.DstIP.Equal([]byte{255, 255, 255, 255})
	devices := h.stack.GetDevices().GetByIP(ip.DstIP)

	if len(devices) == 0 && !isBroadcast {
		// Not for us and not broadcast
		if debugLevel >= 3 {
			fmt.Printf("IP packet not for our devices: %s sn=%d\n", ip.DstIP, pkt.SerialNumber)
		}
		return
	}

	// For broadcast packets, deliver to all devices (for DHCP, etc.)
	if isBroadcast && len(devices) == 0 {
		devices = h.stack.GetDevices().GetAll()
	}

	// If this is UDP and a device has MapToIP configured, skip TTL handling
	skipTTL := false
	if ip.Protocol == IPProtocolUDP {
		for _, device := range devices {
			if device.MapToIP != nil {
				skipTTL = true
				break
			}
		}
	}

	// Handle TTL timeout (traceroute simulation)
	if !skipTTL && h.handleTTLTimeout(pkt, ip) {
		return
	}

	// Route to layer 4 protocol handler
	switch ip.Protocol {
	case IPProtocolICMP:
		h.stack.icmpHandler.HandlePacket(pkt, ip, devices)
	case IPProtocolUDP:
		h.stack.udpHandler.HandlePacket(pkt, ip, devices)
	case IPProtocolTCP:
		h.stack.tcpHandler.HandlePacket(pkt, ip, devices)
	default:
		if debugLevel >= 2 {
			fmt.Printf("Unhandled IP protocol %d sn=%d\n", ip.Protocol, pkt.SerialNumber)
		}
	}
}

func (h *IPHandler) handleTTLTimeout(pkt *Packet, ipLayer *layers.IPv4) bool {
	if h.stack == nil || h.stack.icmpHandler == nil {
		return false
	}
	if ipLayer == nil {
		return false
	}
	ttl := int(ipLayer.TTL)
	device := h.stack.GetDevices().GetDeviceByTTL(ttl)
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
	if device.TTLConfig != nil && device.TTLConfig.IP != nil && device.TTLConfig.Mask != nil {
		if dst := ipLayer.DstIP.To4(); dst != nil && device.TTLConfig.IP.To4() != nil {
			sameSubnet := true
			for i := 0; i < 4; i++ {
				if (dst[i] & device.TTLConfig.Mask[i]) != device.TTLConfig.IP.To4()[i] {
					sameSubnet = false
					break
				}
			}
			if sameSubnet {
				return false
			}
		}
	}

	dstMAC := pkt.GetSourceMAC()
	if dstMAC == nil {
		return false
	}

	if err := h.stack.icmpHandler.SendICMPTimeExceeded(srcIP, ipLayer.SrcIP, device.MACAddress, dstMAC, ipLayer, device); err != nil {
		return false
	}

	h.stack.GetDevices().IncrementTTLCount(device)
	return true
}

// SendIPPacket sends an IP packet
func (h *IPHandler) SendIPPacket(srcIP, dstIP net.IP, protocol layers.IPProtocol, payload []byte, srcMAC, dstMAC net.HardwareAddr) error {
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
		return fmt.Errorf("error serializing IP packet: %v", err)
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
		fmt.Printf("Sent IP packet: %s -> %s protocol=%d length=%d sn=%d\n",
			srcIP, dstIP, protocol, len(payload), serialNum)
	}

	return nil
}
