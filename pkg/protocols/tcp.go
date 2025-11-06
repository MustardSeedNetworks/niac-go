package protocols

import (
	"fmt"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/config"
)

// Well-known TCP ports
const (
	TCPPortFTP    = 21
	TCPPortSSH    = 22
	TCPPortTelnet = 23
	TCPPortHTTP   = 80
	TCPPortHTTPS  = 443
)

// TCPHandler handles TCP packets
type TCPHandler struct {
	stack *Stack
}

// NewTCPHandler creates a new TCP handler
func NewTCPHandler(stack *Stack) *TCPHandler {
	return &TCPHandler{
		stack: stack,
	}
}

// HandlePacket processes a TCP packet over IPv4
func (h *TCPHandler) HandlePacket(pkt *Packet, ipLayer *layers.IPv4, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	// Parse TCP layer
	packet := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer == nil {
		if debugLevel >= 2 {
			fmt.Printf("TCP packet missing TCP layer sn=%d\n", pkt.SerialNumber)
		}
		return
	}

	tcp, ok := tcpLayer.(*layers.TCP)
	if !ok {
		return
	}

	if debugLevel >= 3 {
		flags := ""
		if tcp.SYN {
			flags += "SYN "
		}
		if tcp.ACK {
			flags += "ACK "
		}
		if tcp.FIN {
			flags += "FIN "
		}
		if tcp.RST {
			flags += "RST "
		}
		fmt.Printf("TCP packet: %s:%d -> %s:%d flags=[%s] seq=%d ack=%d sn=%d\n",
			ipLayer.SrcIP, tcp.SrcPort, ipLayer.DstIP, tcp.DstPort,
			flags, tcp.Seq, tcp.Ack, pkt.SerialNumber)
	}

	// Route to application handlers based on destination port
	switch tcp.DstPort {
	case TCPPortHTTP:
		// HTTP traffic
		if len(tcp.Payload) > 0 {
			h.stack.httpHandler.HandleRequest(pkt, ipLayer, tcp, devices)
		}
	case TCPPortFTP:
		// FTP control connection
		if len(tcp.Payload) > 0 {
			h.stack.ftpHandler.HandleRequest(pkt, ipLayer, tcp, devices)
		}
	default:
		// For unsupported ports, send RST on SYN
		if tcp.SYN && !tcp.ACK {
			h.sendRST(ipLayer, tcp, devices)
		}
	}
}

// sendRST sends a TCP RST packet
func (h *TCPHandler) sendRST(ipLayer *layers.IPv4, tcp *layers.TCP, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	// Get source device
	for _, device := range devices {
		if len(device.MACAddress) == 0 {
			continue
		}

		// Check if device has the destination IP
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

		// Build RST packet
		// Get source MAC from device table lookup by source IP
		srcDevice := h.stack.GetDevices().GetByIP(ipLayer.SrcIP)
		var dstMAC []byte
		if len(srcDevice) > 0 && len(srcDevice[0].MACAddress) > 0 {
			dstMAC = srcDevice[0].MACAddress
		} else {
			// Use MAC from original packet (reverse lookup)
			// For now, skip if we can't find it
			if debugLevel >= 2 {
				fmt.Printf("Cannot send RST: no MAC for %s\n", ipLayer.SrcIP)
			}
			continue
		}

		// Build Ethernet header
		eth := &layers.Ethernet{
			SrcMAC:       device.MACAddress,
			DstMAC:       dstMAC,
			EthernetType: layers.EthernetTypeIPv4,
		}

		// Build IP header
		ipReply := &layers.IPv4{
			Version:  4,
			IHL:      5,
			TTL:      64,
			Protocol: layers.IPProtocolTCP,
			SrcIP:    ipLayer.DstIP,
			DstIP:    ipLayer.SrcIP,
		}

		// Build TCP header with RST
		tcpReply := &layers.TCP{
			SrcPort: tcp.DstPort,
			DstPort: tcp.SrcPort,
			Seq:     0,
			Ack:     tcp.Seq + 1,
			RST:     true,
			ACK:     true,
			Window:  0,
		}
		tcpReply.SetNetworkLayerForChecksum(ipReply)

		// Serialize
		buffer := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{
			FixLengths:       true,
			ComputeChecksums: true,
		}

		err := gopacket.SerializeLayers(buffer, opts, eth, ipReply, tcpReply)
		if err != nil {
			if debugLevel >= 2 {
				fmt.Printf("Error serializing TCP RST: %v\n", err)
			}
			continue
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

		if debugLevel >= 3 {
			fmt.Printf("Sent TCP RST from %s:%d to %s:%d device=%s sn=%d\n",
				ipReply.SrcIP, tcpReply.SrcPort, ipReply.DstIP, tcpReply.DstPort,
				device.Name, serialNum)
		}

		break // Only send one RST
	}
}

// SendTCP sends a TCP packet
func (h *TCPHandler) SendTCP(srcIP, dstIP []byte, srcPort, dstPort uint16, seq, ack uint32, flags byte, payload []byte, srcMAC, dstMAC []byte) error {
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
		Protocol: layers.IPProtocolTCP,
		SrcIP:    srcIP,
		DstIP:    dstIP,
	}

	// Build TCP header
	tcpLayer := &layers.TCP{
		SrcPort: layers.TCPPort(srcPort),
		DstPort: layers.TCPPort(dstPort),
		Seq:     seq,
		Ack:     ack,
		Window:  65535,
	}

	// Set flags
	tcpLayer.SYN = (flags & 0x02) != 0
	tcpLayer.ACK = (flags & 0x10) != 0
	tcpLayer.FIN = (flags & 0x01) != 0
	tcpLayer.RST = (flags & 0x04) != 0
	tcpLayer.PSH = (flags & 0x08) != 0

	tcpLayer.SetNetworkLayerForChecksum(ipLayer)

	// Serialize
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts,
		eth,
		ipLayer,
		tcpLayer,
		gopacket.Payload(payload),
	)
	if err != nil {
		return fmt.Errorf("error serializing TCP packet: %v", err)
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
		fmt.Printf("Sent TCP packet: %s:%d -> %s:%d length=%d sn=%d\n",
			srcIP, srcPort, dstIP, dstPort, len(payload), serialNum)
	}

	return nil
}

// HandlePacketV6 processes a TCP packet over IPv6
func (h *TCPHandler) HandlePacketV6(pkt *Packet, packet gopacket.Packet, ipv6 *layers.IPv6, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	// Parse TCP layer
	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer == nil {
		if debugLevel >= 2 {
			fmt.Printf("TCP/IPv6 packet missing TCP layer sn=%d\n", pkt.SerialNumber)
		}
		return
	}

	tcp, ok := tcpLayer.(*layers.TCP)
	if !ok {
		return
	}

	if debugLevel >= 3 {
		flags := ""
		if tcp.SYN {
			flags += "SYN "
		}
		if tcp.ACK {
			flags += "ACK "
		}
		if tcp.FIN {
			flags += "FIN "
		}
		if tcp.RST {
			flags += "RST "
		}
		fmt.Printf("TCP/IPv6 packet: [%s]:%d -> [%s]:%d flags=[%s] seq=%d ack=%d sn=%d\n",
			ipv6.SrcIP, tcp.SrcPort, ipv6.DstIP, tcp.DstPort,
			flags, tcp.Seq, tcp.Ack, pkt.SerialNumber)
	}

	// Route to application handlers based on destination port
	switch tcp.DstPort {
	case TCPPortHTTP:
		// HTTP traffic over IPv6
		if len(tcp.Payload) > 0 {
			h.stack.httpHandler.HandleRequestV6(pkt, packet, ipv6, tcp, devices)
		}
	case TCPPortFTP:
		// FTP control connection over IPv6
		if len(tcp.Payload) > 0 {
			h.stack.ftpHandler.HandleRequestV6(pkt, packet, ipv6, tcp, devices)
		}
	default:
		// For unsupported ports, send RST on SYN
		if tcp.SYN && !tcp.ACK {
			h.sendRSTV6(ipv6, tcp, devices)
		}
	}
}

// sendRSTV6 sends a TCP RST packet over IPv6
func (h *TCPHandler) sendRSTV6(ipv6 *layers.IPv6, tcp *layers.TCP, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	// Get source device
	for _, device := range devices {
		if len(device.MACAddress) == 0 {
			continue
		}

		// Check if device has the destination IPv6
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

		// Get destination MAC
		srcDevice := h.stack.GetDevices().GetByIP(ipv6.SrcIP)
		var dstMAC []byte
		if len(srcDevice) > 0 && len(srcDevice[0].MACAddress) > 0 {
			dstMAC = srcDevice[0].MACAddress
		} else {
			if debugLevel >= 2 {
				fmt.Printf("Cannot send RST: no MAC for [%s]\n", ipv6.SrcIP)
			}
			continue
		}

		// Build Ethernet header
		eth := &layers.Ethernet{
			SrcMAC:       device.MACAddress,
			DstMAC:       dstMAC,
			EthernetType: layers.EthernetTypeIPv6,
		}

		// Build IPv6 header
		ipv6Reply := &layers.IPv6{
			Version:    6,
			HopLimit:   64,
			NextHeader: layers.IPProtocolTCP,
			SrcIP:      ipv6.DstIP,
			DstIP:      ipv6.SrcIP,
		}

		// Build TCP header with RST
		tcpReply := &layers.TCP{
			SrcPort: tcp.DstPort,
			DstPort: tcp.SrcPort,
			Seq:     0,
			Ack:     tcp.Seq + 1,
			RST:     true,
			ACK:     true,
			Window:  0,
		}
		tcpReply.SetNetworkLayerForChecksum(ipv6Reply)

		// Serialize
		buffer := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{
			FixLengths:       true,
			ComputeChecksums: true,
		}

		err := gopacket.SerializeLayers(buffer, opts, eth, ipv6Reply, tcpReply)
		if err != nil {
			if debugLevel >= 2 {
				fmt.Printf("Error serializing TCP/IPv6 RST: %v\n", err)
			}
			continue
		}

		// Send packet
		h.stack.mu.Lock()
		h.stack.serialNumber++
		serialNum := h.stack.serialNumber
		h.stack.mu.Unlock()

		pkt := &Packet{
			Buffer:       buffer.Bytes(),
			Length:       len(buffer.Bytes()),
			SerialNumber: serialNum,
			Device:       device,
		}

		h.stack.Send(pkt)

		if debugLevel >= 3 {
			fmt.Printf("Sent TCP/IPv6 RST from [%s]:%d to [%s]:%d device=%s sn=%d\n",
				ipv6Reply.SrcIP, tcpReply.SrcPort, ipv6Reply.DstIP, tcpReply.DstPort,
				device.Name, serialNum)
		}

		break
	}
}
