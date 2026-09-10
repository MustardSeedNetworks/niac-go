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
			(!slices.Contains(s.fabric.attachmentDHCP, device) || !s.fabric.deviceOnAttachment(device)) {
			continue
		}
		if handler := s.dhcpHandlers[device]; handler != nil {
			handler.HandlePacket(pkt, ip)
		}
	}
}
