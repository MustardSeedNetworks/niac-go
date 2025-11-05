package protocols

import (
	"fmt"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/config"
)

// Well-known UDP ports
const (
	UDPPortDNS   = 53
	UDPPortDHCP  = 67
	UDPPortDHCPC = 68
	UDPPortSNMP  = 161
)

// UDPHandler handles UDP packets
type UDPHandler struct {
	stack *Stack
}

// NewUDPHandler creates a new UDP handler
func NewUDPHandler(stack *Stack) *UDPHandler {
	return &UDPHandler{
		stack: stack,
	}
}

// HandlePacket processes a UDP packet
func (h *UDPHandler) HandlePacket(pkt *Packet, ipLayer *layers.IPv4, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	// Parse UDP layer
	packet := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	udpLayer := packet.Layer(layers.LayerTypeUDP)
	if udpLayer == nil {
		if debugLevel >= 2 {
			fmt.Printf("UDP packet missing UDP layer sn=%d\n", pkt.SerialNumber)
		}
		return
	}

	udp, ok := udpLayer.(*layers.UDP)
	if !ok {
		return
	}

	if debugLevel >= 3 {
		fmt.Printf("UDP packet: %s:%d -> %s:%d length=%d sn=%d\n",
			ipLayer.SrcIP, udp.SrcPort, ipLayer.DstIP, udp.DstPort, len(udp.Payload), pkt.SerialNumber)
	}

	// Route to application handler based on port
	switch udp.DstPort {
	case UDPPortDNS:
		// DNS query
		h.stack.dnsHandler.HandleQuery(pkt, ipLayer, udp, devices)
	case UDPPortDHCP:
		// DHCP server port
		h.stack.dhcpHandler.HandlePacket(pkt, ipLayer, udp, devices)
	case UDPPortSNMP:
		// SNMP query
		// TODO: Add SNMP handler
		if debugLevel >= 2 {
			fmt.Printf("SNMP query received (not yet implemented) sn=%d\n", pkt.SerialNumber)
		}
	case NetBIOSNameServicePort:
		// NetBIOS Name Service
		h.stack.netbiosHandler.HandleNameService(pkt, packet, udp, devices)
	case NetBIOSDatagramServicePort:
		// NetBIOS Datagram Service
		h.stack.netbiosHandler.HandleDatagramService(pkt, packet, udp, devices)
	default:
		if debugLevel >= 3 {
			fmt.Printf("UDP packet to unhandled port %d sn=%d\n", udp.DstPort, pkt.SerialNumber)
		}
	}
}

// SendUDP sends a UDP packet
func (h *UDPHandler) SendUDP(srcIP, dstIP []byte, srcPort, dstPort uint16, payload []byte, srcMAC, dstMAC []byte) error {
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
		Protocol: layers.IPProtocolUDP,
		SrcIP:    srcIP,
		DstIP:    dstIP,
	}

	// Build UDP header
	udpLayer := &layers.UDP{
		SrcPort: layers.UDPPort(srcPort),
		DstPort: layers.UDPPort(dstPort),
	}
	udpLayer.SetNetworkLayerForChecksum(ipLayer)

	// Serialize
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts,
		eth,
		ipLayer,
		udpLayer,
		gopacket.Payload(payload),
	)
	if err != nil {
		return fmt.Errorf("error serializing UDP packet: %v", err)
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
		fmt.Printf("Sent UDP packet: %s:%d -> %s:%d length=%d sn=%d\n",
			srcIP, srcPort, dstIP, dstPort, len(payload), serialNum)
	}

	return nil
}

// HandlePacketV6 processes a UDP packet over IPv6
func (h *UDPHandler) HandlePacketV6(pkt *Packet, packet gopacket.Packet, ipv6 *layers.IPv6, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	// Parse UDP layer
	udpLayer := packet.Layer(layers.LayerTypeUDP)
	if udpLayer == nil {
		if debugLevel >= 2 {
			fmt.Printf("UDP/IPv6 packet missing UDP layer sn=%d\n", pkt.SerialNumber)
		}
		return
	}

	udp, ok := udpLayer.(*layers.UDP)
	if !ok {
		return
	}

	if debugLevel >= 3 {
		fmt.Printf("UDP/IPv6 packet: [%s]:%d -> [%s]:%d length=%d sn=%d\n",
			ipv6.SrcIP, udp.SrcPort, ipv6.DstIP, udp.DstPort, len(udp.Payload), pkt.SerialNumber)
	}

	// Route to application handler based on port
	switch udp.DstPort {
	case UDPPortDNS:
		// DNS query over IPv6
		h.stack.dnsHandler.HandleQueryV6(pkt, packet, ipv6, udp, devices)
	case UDPPortSNMP:
		// SNMP query over IPv6
		if debugLevel >= 2 {
			fmt.Printf("SNMP/IPv6 query received (not yet implemented) sn=%d\n", pkt.SerialNumber)
		}
	case 547:
		// DHCPv6 server port
		h.stack.dhcpv6Handler.HandlePacket(pkt, ipv6, udp, devices)
	case NetBIOSNameServicePort:
		// NetBIOS Name Service over IPv6
		h.stack.netbiosHandler.HandleNameService(pkt, packet, udp, devices)
	case NetBIOSDatagramServicePort:
		// NetBIOS Datagram Service over IPv6
		h.stack.netbiosHandler.HandleDatagramService(pkt, packet, udp, devices)
	default:
		if debugLevel >= 3 {
			fmt.Printf("UDP/IPv6 packet to unhandled port %d sn=%d\n", udp.DstPort, pkt.SerialNumber)
		}
	}
}
