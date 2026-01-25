package protocols

import (
	"fmt"
	"log/slog"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/logging"
)

// dhcpPacketInfo holds parsed DHCP packet information.
type dhcpPacketInfo struct {
	dhcp        *layers.DHCPv4
	messageType uint8
	hostname    string
}

// parseDHCPPacket parses a DHCP packet and extracts relevant information.
func (h *DHCPHandler) parseDHCPPacket(pkt *Packet) *dhcpPacketInfo {
	packet := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)

	dhcpLayer := packet.Layer(layers.LayerTypeDHCPv4)
	if dhcpLayer == nil {
		return nil
	}

	dhcp, ok := dhcpLayer.(*layers.DHCPv4)
	if !ok {
		return nil
	}

	info := &dhcpPacketInfo{dhcp: dhcp}

	// Extract message type and hostname from options
	for _, opt := range dhcp.Options {
		//exhaustive:ignore
		switch opt.Type {
		case layers.DHCPOptMessageType:
			if len(opt.Data) > 0 {
				info.messageType = opt.Data[0]
			}
		case layers.DHCPOptHostname:
			if len(opt.Data) > 0 {
				info.hostname = string(opt.Data)
			}
		default:
			// Ignore other DHCP options - only interested in message type and hostname
		}
	}

	return info
}

// HandlePacket processes a DHCP packet.
func (h *DHCPHandler) HandlePacket(pkt *Packet, ipLayer *layers.IPv4, _ *layers.UDP, devices []*config.Device) {
	logger := slog.Default()
	debugLevel := h.stack.GetDebugLevel()

	h.stack.IncrementStat("dhcp_requests")

	info := h.parseDHCPPacket(pkt)
	if info == nil {
		if debugLevel >= DebugLevelInfo {
			logger.Debug("DHCP packet missing DHCP layer", "sn", pkt.SerialNumber)
		}

		return
	}

	if debugLevel >= DebugLevelVerbose {
		logger.Debug("DHCP message received",
			"type", dhcpMessageTypeString(info.messageType),
			"srcIP", ipLayer.SrcIP,
			"mac", info.dhcp.ClientHWAddr,
			"xid", info.dhcp.Xid,
			"sn", pkt.SerialNumber)
	}

	serverDevice := findServerDevice(devices)
	if serverDevice == nil {
		if debugLevel >= DebugLevelInfo {
			logger.Debug("DHCP: No server device configured", "sn", pkt.SerialNumber)
		}

		return
	}

	h.dispatchDHCPMessage(info, serverDevice, pkt.SerialNumber, debugLevel)
}

// dispatchDHCPMessage routes the DHCP message to the appropriate handler.
func (h *DHCPHandler) dispatchDHCPMessage(
	info *dhcpPacketInfo,
	serverDevice *config.Device,
	serialNum int,
	debugLevel int,
) {
	logger := slog.Default()

	switch info.messageType {
	case DHCPDiscover:
		h.handleDHCPDiscover(info, serverDevice, serialNum, debugLevel)
	case DHCPRequest:
		h.handleDHCPRequest(info, serverDevice, serialNum, debugLevel)
	case DHCPRelease:
		if debugLevel >= DebugLevelInfo {
			logger.Debug("DHCP: Release", "mac", info.dhcp.ClientHWAddr, "sn", serialNum)
		}

		h.mu.Lock()
		delete(h.leases, info.dhcp.ClientHWAddr.String())
		h.mu.Unlock()
	case DHCPInform:
		if debugLevel >= DebugLevelVerbose {
			logger.Debug("DHCP: Inform", "mac", info.dhcp.ClientHWAddr, "sn", serialNum)
		}
	default:
		if debugLevel >= DebugLevelInfo {
			logger.Debug("DHCP: Unhandled message type", "type", info.messageType, "sn", serialNum)
		}
	}
}

// handleDHCPDiscover handles a DHCP Discover message.
func (h *DHCPHandler) handleDHCPDiscover(
	info *dhcpPacketInfo,
	serverDevice *config.Device,
	serialNum int,
	debugLevel int,
) {
	logger := slog.Default()

	if debugLevel >= DebugLevelInfo {
		logger.Debug("DHCP: Processing Discover", "mac", info.dhcp.ClientHWAddr, "sn", serialNum)
	}

	lease, err := h.allocateLease(info.dhcp.ClientHWAddr, nil, info.hostname)
	if err != nil {
		if debugLevel >= 1 {
			logging.ProtocolDebugf("DHCP", debugLevel, 1, "Failed to allocate IP: %v sn=%d", err, serialNum)
		}

		return
	}

	offerErr := h.SendDHCPOffer(
		info.dhcp.Xid,
		info.dhcp.ClientHWAddr,
		lease.IP,
		serverDevice.IPAddresses[0],
		serverDevice.MACAddress,
	)
	if offerErr != nil {
		if debugLevel >= 1 {
			logging.ProtocolDebugf("DHCP", debugLevel, 1, "Failed to send Offer: %v sn=%d", offerErr, serialNum)
		}

		return
	}

	h.stack.IncrementStat("dhcp_offers")

	if debugLevel >= debugLevelInfo {
		logging.ProtocolDebugf("DHCP", debugLevel, debugLevelInfo, "Sent Offer IP=%s to %s sn=%d",
			lease.IP, info.dhcp.ClientHWAddr, serialNum)
	}
}

// handleDHCPRequest handles a DHCP Request message.
func (h *DHCPHandler) handleDHCPRequest(
	info *dhcpPacketInfo,
	serverDevice *config.Device,
	serialNum int,
	debugLevel int,
) {
	logger := slog.Default()

	if debugLevel >= DebugLevelInfo {
		logger.Debug("DHCP: Processing Request", "mac", info.dhcp.ClientHWAddr, "sn", serialNum)
	}

	requestedIP := getRequestedIP(info.dhcp)

	lease, err := h.allocateLease(info.dhcp.ClientHWAddr, requestedIP, info.hostname)
	if err != nil {
		if debugLevel >= 1 {
			logging.ProtocolDebugf("DHCP", debugLevel, 1, "Failed to confirm lease: %v sn=%d", err, serialNum)
		}

		return
	}

	ackErr := h.SendDHCPAck(
		info.dhcp.Xid,
		info.dhcp.ClientHWAddr,
		lease.IP,
		serverDevice.IPAddresses[0],
		serverDevice.MACAddress,
	)
	if ackErr != nil {
		if debugLevel >= 1 {
			logging.ProtocolDebugf("DHCP", debugLevel, 1, "Failed to send Ack: %v sn=%d", ackErr, serialNum)
		}

		return
	}

	h.stack.IncrementStat("dhcp_acks")
	h.updateFDBTables(info.dhcp.ClientHWAddr)

	if debugLevel >= debugLevelInfo {
		logging.ProtocolDebugf("DHCP", debugLevel, debugLevelInfo, "Sent Ack IP=%s to %s sn=%d",
			lease.IP, info.dhcp.ClientHWAddr, serialNum)
	}
}

// dhcpMessageTypeString returns string representation of DHCP message type.
func dhcpMessageTypeString(msgType uint8) string {
	switch msgType {
	case DHCPDiscover:
		return "DISCOVER"
	case DHCPOffer:
		return "OFFER"
	case DHCPRequest:
		return "REQUEST"
	case DHCPDecline:
		return "DECLINE"
	case DHCPAck:
		return "ACK"
	case DHCPNak:
		return "NAK"
	case DHCPRelease:
		return "RELEASE"
	case DHCPInform:
		return "INFORM"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", msgType)
	}
}
