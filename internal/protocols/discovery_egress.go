package protocols

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

var errDiscoveryOffAttachment = errors.New("discovery advertisement is not on attachment network")

func (s *Stack) validateDiscoveryEgress(pkt *Packet) error {
	if s.fabric == nil || pkt == nil || !isDiscoveryAdvertisement(pkt.Buffer) {
		return nil
	}
	device, ok := pkt.Device.(*config.Device)
	if !ok {
		return fmt.Errorf("discovery advertisement has invalid device identity %T", pkt.Device)
	}
	if s.fabric.deviceOnAttachment(device) {
		return nil
	}
	return fmt.Errorf(
		"%w: %q",
		errDiscoveryOffAttachment,
		device.Name,
	)
}

func isDiscoveryAdvertisement(frame []byte) bool {
	if len(frame) < SizeOfMac {
		return false
	}
	destination := net.HardwareAddr(frame[:SizeOfMac]).String()
	return strings.EqualFold(destination, LLDPMulticastMAC) ||
		strings.EqualFold(destination, CDPMulticastMAC) ||
		strings.EqualFold(destination, EDPMulticastMAC) ||
		strings.EqualFold(destination, FDPMulticastMAC)
}

func (r *fabricRuntime) deviceOnAttachment(device *config.Device) bool {
	for _, iface := range r.topology.Interfaces {
		if iface.Device == device.Name &&
			iface.Network == r.attachmentNetwork &&
			r.interfaceAvailable(device, iface.Name) {
			return true
		}
	}
	return false
}

func (s *Stack) recordDiscoveryEgressDrop(pkt *Packet) {
	s.stats.mu.Lock()
	s.stats.FabricDrops++
	pkt.fabricTrace.IngressNetwork = s.fabric.attachmentNetwork
	pkt.fabricTrace.PhysicalVLAN = s.fabric.binding.AccessVLAN
	pkt.fabricTrace.RouteDecision = fabricRouteDecisionDropped
	pkt.fabricTrace.RejectionReason = "discovery_not_on_attachment"
	s.stats.mu.Unlock()
	s.notifyObservers("tx", pkt)
}
