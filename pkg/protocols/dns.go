package protocols

import (
	"fmt"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/config"
)

// DNSHandler handles DNS queries and responses
type DNSHandler struct {
	stack *Stack
}

// NewDNSHandler creates a new DNS handler
func NewDNSHandler(stack *Stack) *DNSHandler {
	return &DNSHandler{
		stack: stack,
	}
}

// HandleQuery processes a DNS query
func (h *DNSHandler) HandleQuery(pkt *Packet, ipLayer *layers.IPv4, udpLayer *layers.UDP, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	// Parse DNS layer
	packet := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	dnsLayer := packet.Layer(layers.LayerTypeDNS)
	if dnsLayer == nil {
		if debugLevel >= 2 {
			fmt.Printf("DNS packet missing DNS layer sn=%d\n", pkt.SerialNumber)
		}
		return
	}

	dns, ok := dnsLayer.(*layers.DNS)
	if !ok {
		return
	}

	h.stack.IncrementStat("dns_queries")

	if debugLevel >= 3 {
		for _, q := range dns.Questions {
			fmt.Printf("DNS Query: %s type=%s class=%s from %s sn=%d\n",
				string(q.Name), q.Type, q.Class, ipLayer.SrcIP, pkt.SerialNumber)
		}
	}

	// For now, respond with NXDOMAIN (not implemented)
	// In a full implementation, this would:
	// 1. Check if query is for one of our simulated devices
	// 2. Return appropriate A/AAAA/PTR records
	// 3. Handle recursive vs authoritative queries
	// 4. Implement proper DNS response format

	if debugLevel >= 2 {
		fmt.Printf("DNS query handling not yet fully implemented sn=%d\n", pkt.SerialNumber)
	}

	// TODO: Implement full DNS response
}

// SendDNSResponse sends a DNS response
func (h *DNSHandler) SendDNSResponse(query *layers.DNS, srcIP, dstIP []byte, srcMAC, dstMAC []byte) error {
	// Build response
	response := &layers.DNS{
		ID:           query.ID,
		QR:           true, // Response
		OpCode:       query.OpCode,
		AA:           true, // Authoritative
		TC:           false,
		RD:           query.RD,
		RA:           true, // Recursion available
		ResponseCode: layers.DNSResponseCodeNoErr,
		Questions:    query.Questions,
		Answers:      []layers.DNSResourceRecord{}, // TODO: Add answers
	}

	// Serialize DNS
	dnsBuffer := gopacket.NewSerializeBuffer()
	err := response.SerializeTo(dnsBuffer, gopacket.SerializeOptions{})
	if err != nil {
		return fmt.Errorf("error serializing DNS response: %v", err)
	}

	// Send as UDP
	return h.stack.udpHandler.SendUDP(srcIP, dstIP, 53, 53, dnsBuffer.Bytes(), srcMAC, dstMAC)
}
