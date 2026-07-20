// Package protocols implements network protocol handlers for device simulation
package protocols

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
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
	targetIP := net.IP(arp.DstProtAddress)
	sourceIP := net.IP(arp.SourceProtAddress)
	sourceMAC := net.HardwareAddr(arp.SourceHwAddress)

	h.stack.IncrementStat("arp_requests")
	h.logARPRequest(pkt, targetIP, sourceIP, sourceMAC)

	devices := h.targetDevices(targetIP, pkt.VLAN)
	if len(devices) == 0 {
		h.logDebug("ARP Request: No device found for IP", "ip", targetIP)
		return
	}

	h.sendARPReplies(devices, targetIP, sourceMAC, sourceIP, pkt.VLAN)
}

func (h *ARPHandler) targetDevices(targetIP net.IP, vlan int) []*config.Device {
	if h.stack.fabric != nil {
		device, ok := h.stack.fabric.resolveARP(targetIP)
		if !ok {
			return nil
		}
		return []*config.Device{device}
	}
	return h.stack.devicesFor(vlan).GetByIP(targetIP)
}

// logARPRequest logs an ARP request at verbose debug level.
func (h *ARPHandler) logARPRequest(
	pkt *Packet,
	targetIP, sourceIP net.IP,
	sourceMAC net.HardwareAddr,
) {
	if h.stack.GetDebugLevel() >= DebugLevelVerbose {
		slog.Default().Debug("ARP Request",
			"targetIP", targetIP, "sourceIP", sourceIP, "sourceMAC", sourceMAC, "sn", pkt.SerialNumber)
	}
}

// logDebug logs a message at verbose debug level.
func (h *ARPHandler) logDebug(msg string, args ...any) {
	if h.stack.GetDebugLevel() >= DebugLevelVerbose {
		slog.Default().Debug(msg, args...)
	}
}

// sendARPReplies sends ARP replies for all matching devices.
func (h *ARPHandler) sendARPReplies(
	devices []*config.Device, targetIP net.IP, sourceMAC net.HardwareAddr, sourceIP net.IP, vlan int,
) {
	for _, device := range devices {
		if len(device.MACAddress) == 0 {
			continue
		}
		reply := h.buildARPReply(device.MACAddress, targetIP, sourceMAC, sourceIP)
		if reply != nil {
			reply.VLAN = vlan // answer on the VLAN the request arrived on
			h.stack.Send(reply)
			h.stack.IncrementStat("arp_replies")
			h.logDebug(
				"ARP Reply",
				"ip",
				targetIP,
				"mac",
				device.MACAddress,
				"device",
				device.Name,
				"sn",
				reply.SerialNumber,
			)
		}
	}
}

// buildARPReply constructs an ARP reply packet.
func (h *ARPHandler) buildARPReply(
	senderMAC net.HardwareAddr, senderIP net.IP, targetMAC net.HardwareAddr, targetIP net.IP,
) *Packet {
	eth := &layers.Ethernet{
		SrcMAC:       senderMAC,
		DstMAC:       targetMAC,
		EthernetType: layers.EthernetTypeARP,
	}
	arpLayer := buildARPLayer(layers.ARPReply, senderMAC, senderIP, targetMAC, targetIP)

	buffer, err := serializeARPPacket(eth, arpLayer)
	if err != nil {
		h.logDebug("Error serializing ARP reply", "error", err)
		return nil
	}

	return h.createPacket(buffer)
}

// buildARPLayer creates an ARP layer with the given parameters.
func buildARPLayer(
	op uint16,
	srcMAC net.HardwareAddr,
	srcIP net.IP,
	dstMAC net.HardwareAddr,
	dstIP net.IP,
) *layers.ARP {
	return &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     arpHWAddressSize,
		ProtAddressSize:   arpProtAddressSize,
		Operation:         op,
		SourceHwAddress:   srcMAC,
		SourceProtAddress: srcIP.To4(),
		DstHwAddress:      dstMAC,
		DstProtAddress:    dstIP.To4(),
	}
}

// serializeARPPacket serializes Ethernet and ARP layers.
func serializeARPPacket(eth *layers.Ethernet, arpLayer *layers.ARP) ([]byte, error) {
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buffer, opts, eth, arpLayer); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// createPacket creates a packet with a new serial number.
func (h *ARPHandler) createPacket(buffer []byte) *Packet {
	h.stack.mu.Lock()
	h.stack.serialNumber++
	serialNum := h.stack.serialNumber
	h.stack.mu.Unlock()

	return &Packet{Buffer: buffer, Length: len(buffer), SerialNumber: serialNum}
}

// SendGratuitousARP sends a gratuitous ARP announcement.
func (h *ARPHandler) SendGratuitousARP(device *config.Device) error {
	if len(device.MACAddress) == 0 || len(device.IPAddresses) == 0 {
		return ErrDeviceMissingMACOrIP
	}

	ip := device.IPAddresses[0]
	broadcastMAC := net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	zeroMAC := net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	eth := &layers.Ethernet{
		SrcMAC:       device.MACAddress,
		DstMAC:       broadcastMAC,
		EthernetType: layers.EthernetTypeARP,
	}
	arpLayer := buildARPLayer(layers.ARPRequest, device.MACAddress, ip, zeroMAC, ip)

	buffer, err := serializeARPPacket(eth, arpLayer)
	if err != nil {
		return fmt.Errorf("error serializing gratuitous ARP: %w", err)
	}

	h.stack.Send(h.createPacket(buffer))
	h.logDebug("Sent gratuitous ARP", "ip", ip, "mac", device.MACAddress, "device", device.Name)

	return nil
}
