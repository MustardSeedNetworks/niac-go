// Package protocols implements network protocol handlers for device simulation
package protocols

import (
	"fmt"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/config"
)

// ARP field offsets (after Ethernet header)
const (
	ARPOperation             = 6  // Operation (request/reply)
	ARPSenderHWAddress       = 8  // Sender hardware address
	ARPSenderProtocolAddress = 14 // Sender protocol address
	ARPTargetHWAddress       = 18 // Target hardware address
	ARPTargetProtocolAddress = 24 // Target protocol address
)

// ARPHandler handles ARP requests and replies
type ARPHandler struct {
	stack *Stack
}

// NewARPHandler creates a new ARP handler
func NewARPHandler(stack *Stack) *ARPHandler {
	return &ARPHandler{
		stack: stack,
	}
}

// HandlePacket processes an ARP packet
func (h *ARPHandler) HandlePacket(pkt *Packet) {
	debugLevel := h.stack.GetDebugLevel()

	// Parse using gopacket for easier handling
	packet := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)

	// Get ARP layer
	arpLayer := packet.Layer(layers.LayerTypeARP)
	if arpLayer == nil {
		if debugLevel >= 2 {
			fmt.Printf("ARP packet missing ARP layer sn=%d\n", pkt.SerialNumber)
		}
		return
	}

	arp, ok := arpLayer.(*layers.ARP)
	if !ok {
		return
	}

	// Only handle ARP requests for IPv4 over Ethernet
	if arp.AddrType != layers.LinkTypeEthernet || arp.Protocol != layers.EthernetTypeIPv4 {
		if debugLevel >= 2 {
			fmt.Printf("ARP packet with unsupported type sn=%d\n", pkt.SerialNumber)
		}
		return
	}

	if arp.Operation == layers.ARPRequest {
		h.handleARPRequest(pkt, arp)
	} else if arp.Operation == layers.ARPReply {
		// Could log/track replies if needed
		h.stack.IncrementStat("arp_replies")
		if debugLevel >= 3 {
			fmt.Printf("ARP Reply from %s (%s) sn=%d\n",
				net.IP(arp.SourceProtAddress), net.HardwareAddr(arp.SourceHwAddress), pkt.SerialNumber)
		}
	}
}

// handleARPRequest processes an ARP request and generates reply if we have the target IP
func (h *ARPHandler) handleARPRequest(pkt *Packet, arp *layers.ARP) {
	debugLevel := h.stack.GetDebugLevel()

	targetIP := net.IP(arp.DstProtAddress)
	sourceIP := net.IP(arp.SourceProtAddress)
	sourceMAC := net.HardwareAddr(arp.SourceHwAddress)

	h.stack.IncrementStat("arp_requests")

	if debugLevel >= 3 {
		fmt.Printf("ARP Request: Who has %s? Tell %s (%s) sn=%d\n",
			targetIP, sourceIP, sourceMAC, pkt.SerialNumber)
	}

	// Look up devices with this IP (considering VLAN)
	devices := h.stack.GetDevices().GetByIP(targetIP)
	if len(devices) == 0 {
		if debugLevel >= 3 {
			fmt.Printf("ARP Request: No device found for IP %s\n", targetIP)
		}
		return
	}

	// Send reply for each matching device
	for _, device := range devices {
		// Check VLAN match if applicable (tracked in issue #77 - VLAN-aware ARP)
		// For now, respond to all

		if len(device.MACAddress) == 0 {
			continue
		}

		// Create ARP reply
		reply := h.buildARPReply(device.MACAddress, targetIP, sourceMAC, sourceIP)
		if reply != nil {
			h.stack.Send(reply)
			h.stack.IncrementStat("arp_replies")

			if debugLevel >= 3 {
				fmt.Printf("ARP Reply: %s is at %s (device: %s) sn=%d\n",
					targetIP, device.MACAddress, device.Name, reply.SerialNumber)
			}
		}
	}
}

// buildARPReply constructs an ARP reply packet
func (h *ARPHandler) buildARPReply(senderMAC net.HardwareAddr, senderIP net.IP, targetMAC net.HardwareAddr, targetIP net.IP) *Packet {
	// Build Ethernet header
	eth := &layers.Ethernet{
		SrcMAC:       senderMAC,
		DstMAC:       targetMAC,
		EthernetType: layers.EthernetTypeARP,
	}

	// Build ARP header
	arpLayer := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPReply,
		SourceHwAddress:   senderMAC,
		SourceProtAddress: senderIP.To4(),
		DstHwAddress:      targetMAC,
		DstProtAddress:    targetIP.To4(),
	}

	// Serialize packet
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, arpLayer)
	if err != nil {
		if h.stack.GetDebugLevel() >= 2 {
			fmt.Printf("Error serializing ARP reply: %v\n", err)
		}
		return nil
	}

	// Get serial number
	h.stack.mu.Lock()
	h.stack.serialNumber++
	serialNum := h.stack.serialNumber
	h.stack.mu.Unlock()

	// Create packet
	pkt := &Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: serialNum,
	}

	return pkt
}

// SendGratuitousARP sends a gratuitous ARP announcement
func (h *ARPHandler) SendGratuitousARP(device *config.Device) error {
	if len(device.MACAddress) == 0 || len(device.IPAddresses) == 0 {
		return fmt.Errorf("device missing MAC or IP address")
	}

	// Use first IP
	ip := device.IPAddresses[0]

	// Build gratuitous ARP (announce our IP)
	eth := &layers.Ethernet{
		SrcMAC:       device.MACAddress,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, // Broadcast
		EthernetType: layers.EthernetTypeARP,
	}

	arpLayer := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPRequest,
		SourceHwAddress:   device.MACAddress,
		SourceProtAddress: ip.To4(),
		DstHwAddress:      net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		DstProtAddress:    ip.To4(), // Target is self for gratuitous ARP
	}

	// Serialize
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, arpLayer)
	if err != nil {
		return fmt.Errorf("error serializing gratuitous ARP: %v", err)
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
		fmt.Printf("Sent gratuitous ARP for %s (%s) from device %s\n", ip, device.MACAddress, device.Name)
	}

	return nil
}
