// Package protocols implements network protocol handlers for device simulation
package protocols

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// ARP field offsets (after Ethernet header).
const (
	ARPOperation             = 6  // Operation (request/reply)
	ARPSenderHWAddress       = 8  // Sender hardware address
	ARPSenderProtocolAddress = 14 // Sender protocol address
	ARPTargetHWAddress       = 18 // Target hardware address
	ARPTargetProtocolAddress = 24 // Target protocol address
)

// ARP address sizes.
const (
	arpHWAddressSize   = 6 // Ethernet MAC address size
	arpProtAddressSize = 4 // IPv4 address size
)

// ARPHandler handles ARP requests and replies.
type ARPHandler struct {
	stack *Stack
}

// NewARPHandler creates a new ARP handler.
func NewARPHandler(stack *Stack) *ARPHandler {
	return &ARPHandler{
		stack: stack,
	}
}

// HandlePacket processes an ARP packet.
func (h *ARPHandler) HandlePacket(pkt *Packet) {
	logger := slog.Default()
	debugLevel := h.stack.GetDebugLevel()

	// Parse using gopacket for easier handling
	packet := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)

	// Get ARP layer
	arpLayer := packet.Layer(layers.LayerTypeARP)
	if arpLayer == nil {
		if debugLevel >= DebugLevelInfo {
			logger.Debug("ARP packet missing ARP layer", "sn", pkt.SerialNumber)
		}

		return
	}

	arp, ok := arpLayer.(*layers.ARP)
	if !ok {
		return
	}

	// Only handle ARP requests for IPv4 over Ethernet
	if arp.AddrType != layers.LinkTypeEthernet || arp.Protocol != layers.EthernetTypeIPv4 {
		if debugLevel >= DebugLevelInfo {
			logger.Debug("ARP packet with unsupported type", "sn", pkt.SerialNumber)
		}

		return
	}

	switch arp.Operation {
	case layers.ARPRequest:
		h.handleARPRequest(pkt, arp)
	case layers.ARPReply:
		// Could log/track replies if needed
		h.stack.IncrementStat("arp_replies")

		if debugLevel >= DebugLevelVerbose {
			logger.Debug(
				"ARP Reply",
				"sourceIP",
				net.IP(arp.SourceProtAddress),
				"sourceMAC",
				net.HardwareAddr(arp.SourceHwAddress),
				"sn",
				pkt.SerialNumber,
			)
		}
	}
}

// handleARPRequest processes an ARP request and generates reply if we have the target IP.
func (h *ARPHandler) handleARPRequest(pkt *Packet, arp *layers.ARP) {
	debugLevel := h.stack.GetDebugLevel()

	targetIP := net.IP(arp.DstProtAddress)
	sourceIP := net.IP(arp.SourceProtAddress)
	sourceMAC := net.HardwareAddr(arp.SourceHwAddress)

	h.stack.IncrementStat("arp_requests")

	if debugLevel >= DebugLevelVerbose {
		slog.Debug("ARP Request", "targetIP", targetIP, "sourceIP", sourceIP,
			"sourceMAC", sourceMAC, "sn", pkt.SerialNumber)
	}

	devices := h.stack.GetDevices().GetByIP(targetIP)
	if len(devices) == 0 {
		if debugLevel >= DebugLevelVerbose {
			slog.Debug("ARP Request: No device found for IP", "ip", targetIP)
		}

		return
	}

	h.sendARPReplies(devices, targetIP, sourceMAC, sourceIP, debugLevel)
}

// sendARPReplies sends ARP replies for all matching devices.
func (h *ARPHandler) sendARPReplies(
	devices []*config.Device, targetIP net.IP, sourceMAC net.HardwareAddr, sourceIP net.IP, debugLevel int,
) {
	for _, device := range devices {
		if len(device.MACAddress) == 0 {
			continue
		}

		reply := h.buildARPReply(device.MACAddress, targetIP, sourceMAC, sourceIP)
		if reply != nil {
			h.stack.Send(reply)
			h.stack.IncrementStat("arp_replies")

			if debugLevel >= DebugLevelVerbose {
				slog.Debug("ARP Reply", "ip", targetIP, "mac", device.MACAddress,
					"device", device.Name, "sn", reply.SerialNumber)
			}
		}
	}
}

// buildARPReply constructs an ARP reply packet.
func (h *ARPHandler) buildARPReply(
	senderMAC net.HardwareAddr, senderIP net.IP,
	targetMAC net.HardwareAddr, targetIP net.IP,
) *Packet {
	eth := &layers.Ethernet{
		SrcMAC: senderMAC, DstMAC: targetMAC, EthernetType: layers.EthernetTypeARP,
	}

	arpLayer := &layers.ARP{
		AddrType: layers.LinkTypeEthernet, Protocol: layers.EthernetTypeIPv4,
		HwAddressSize: arpHWAddressSize, ProtAddressSize: arpProtAddressSize,
		Operation: layers.ARPReply, SourceHwAddress: senderMAC,
		SourceProtAddress: senderIP.To4(), DstHwAddress: targetMAC,
		DstProtAddress: targetIP.To4(),
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}

	if err := gopacket.SerializeLayers(buffer, opts, eth, arpLayer); err != nil {
		if h.stack.GetDebugLevel() >= DebugLevelInfo {
			slog.Debug("Error serializing ARP reply", "error", err)
		}

		return nil
	}

	h.stack.mu.Lock()
	h.stack.serialNumber++
	sn := h.stack.serialNumber
	h.stack.mu.Unlock()

	return &Packet{Buffer: buffer.Bytes(), Length: len(buffer.Bytes()), SerialNumber: sn}
}

// SendGratuitousARP sends a gratuitous ARP announcement.
func (h *ARPHandler) SendGratuitousARP(device *config.Device) error {
	if len(device.MACAddress) == 0 || len(device.IPAddresses) == 0 {
		return ErrDeviceMissingMACOrIP
	}

	ip := device.IPAddresses[0]
	eth := &layers.Ethernet{
		SrcMAC: device.MACAddress, DstMAC: net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeARP,
	}

	arpLayer := &layers.ARP{
		AddrType: layers.LinkTypeEthernet, Protocol: layers.EthernetTypeIPv4,
		HwAddressSize: arpHWAddressSize, ProtAddressSize: arpProtAddressSize,
		Operation: layers.ARPRequest, SourceHwAddress: device.MACAddress,
		SourceProtAddress: ip.To4(),
		DstHwAddress:      net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		DstProtAddress:    ip.To4(),
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}

	if err := gopacket.SerializeLayers(buffer, opts, eth, arpLayer); err != nil {
		return fmt.Errorf("error serializing gratuitous ARP: %w", err)
	}

	h.stack.mu.Lock()
	h.stack.serialNumber++
	sn := h.stack.serialNumber
	h.stack.mu.Unlock()

	h.stack.Send(&Packet{Buffer: buffer.Bytes(), Length: len(buffer.Bytes()), SerialNumber: sn})

	if h.stack.GetDebugLevel() >= DebugLevelVerbose {
		slog.Debug("Sent gratuitous ARP", "ip", ip, "mac", device.MACAddress, "device", device.Name)
	}

	return nil
}
