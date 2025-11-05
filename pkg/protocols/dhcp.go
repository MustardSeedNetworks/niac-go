package protocols

import (
	"fmt"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/config"
)

// DHCP message types
const (
	DHCPDiscover = 1
	DHCPOffer    = 2
	DHCPRequest  = 3
	DHCPDecline  = 4
	DHCPAck      = 5
	DHCPNak      = 6
	DHCPRelease  = 7
	DHCPInform   = 8
)

// DHCPHandler handles DHCP server functionality
type DHCPHandler struct {
	stack *Stack
}

// NewDHCPHandler creates a new DHCP handler
func NewDHCPHandler(stack *Stack) *DHCPHandler {
	return &DHCPHandler{
		stack: stack,
	}
}

// HandlePacket processes a DHCP packet
func (h *DHCPHandler) HandlePacket(pkt *Packet, ipLayer *layers.IPv4, udpLayer *layers.UDP, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	h.stack.IncrementStat("dhcp_requests")

	// Parse DHCP layer
	packet := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	dhcpLayer := packet.Layer(layers.LayerTypeDHCPv4)
	if dhcpLayer == nil {
		if debugLevel >= 2 {
			fmt.Printf("DHCP packet missing DHCP layer sn=%d\n", pkt.SerialNumber)
		}
		return
	}

	dhcp, ok := dhcpLayer.(*layers.DHCPv4)
	if !ok {
		return
	}

	// Determine DHCP message type
	var messageType uint8
	for _, opt := range dhcp.Options {
		if opt.Type == layers.DHCPOptMessageType && len(opt.Data) > 0 {
			messageType = opt.Data[0]
			break
		}
	}

	if debugLevel >= 3 {
		msgTypeStr := h.dhcpMessageTypeString(messageType)
		fmt.Printf("DHCP %s from %s (MAC: %s) xid=0x%x sn=%d\n",
			msgTypeStr, ipLayer.SrcIP, dhcp.ClientHWAddr, dhcp.Xid, pkt.SerialNumber)
	}

	// Handle based on message type
	switch messageType {
	case DHCPDiscover:
		// TODO: Implement DHCP Discover -> Offer
		if debugLevel >= 2 {
			fmt.Printf("DHCP Discover handling not yet fully implemented sn=%d\n", pkt.SerialNumber)
		}
	case DHCPRequest:
		// TODO: Implement DHCP Request -> Ack
		if debugLevel >= 2 {
			fmt.Printf("DHCP Request handling not yet fully implemented sn=%d\n", pkt.SerialNumber)
		}
	case DHCPRelease:
		if debugLevel >= 3 {
			fmt.Printf("DHCP Release received sn=%d\n", pkt.SerialNumber)
		}
	case DHCPInform:
		if debugLevel >= 3 {
			fmt.Printf("DHCP Inform received sn=%d\n", pkt.SerialNumber)
		}
	default:
		if debugLevel >= 2 {
			fmt.Printf("Unhandled DHCP message type %d sn=%d\n", messageType, pkt.SerialNumber)
		}
	}

	// TODO: Full DHCP implementation would include:
	// 1. IP address pool management
	// 2. Lease tracking
	// 3. DHCP Offer generation
	// 4. DHCP Ack/Nak generation
	// 5. Option handling (subnet, gateway, DNS, etc.)
}

// dhcpMessageTypeString returns string representation of DHCP message type
func (h *DHCPHandler) dhcpMessageTypeString(msgType uint8) string {
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

// SendDHCPOffer sends a DHCP Offer message
func (h *DHCPHandler) SendDHCPOffer(xid uint32, clientMAC []byte, offeredIP, serverIP []byte) error {
	// TODO: Implement DHCP Offer packet construction
	// Would include:
	// - DHCP header with offer details
	// - Options: subnet mask, router, DNS, lease time, etc.
	// - Broadcast to 255.255.255.255 or unicast to client

	if h.stack.GetDebugLevel() >= 2 {
		fmt.Println("DHCP Offer generation not yet implemented")
	}

	return fmt.Errorf("not yet implemented")
}

// SendDHCPAck sends a DHCP Ack message
func (h *DHCPHandler) SendDHCPAck(xid uint32, clientMAC []byte, assignedIP, serverIP []byte) error {
	// TODO: Implement DHCP Ack packet construction
	// Similar to Offer but confirms the lease

	if h.stack.GetDebugLevel() >= 2 {
		fmt.Println("DHCP Ack generation not yet implemented")
	}

	return fmt.Errorf("not yet implemented")
}
