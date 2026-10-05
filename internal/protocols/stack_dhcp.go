package protocols

import (
	"slices"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func (s *Stack) handleDHCPPacket(pkt *Packet, ip *layers.IPv4, devices []*config.Device) {
	s.IncrementStat("dhcp_requests")
	for _, device := range devices {
		if s.fabric != nil &&
			(!slices.Contains(s.fabric.attachmentDHCP, device) || !s.fabric.dhcpServerReachable(device)) {
			continue
		}
		if handler := s.dhcpHandlers[device]; handler != nil {
			handler.HandlePacket(pkt, ip)
		}
	}
}

// configureRelayedDHCP points each relayed attachment DHCP server at the scope
// it serves the attachment network from. A session has one attachment
// network, so a server answers from exactly one scope for its lifetime.
func (s *Stack) configureRelayedDHCP() {
	if s.fabric == nil {
		return
	}
	for server, relay := range s.fabric.dhcpRelays {
		if handler := s.dhcpHandlers[server]; handler != nil {
			handler.serveRelayedScope(relay)
		}
	}
}
