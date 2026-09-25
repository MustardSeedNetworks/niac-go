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
	if s.fabric.advertisesAtClient(device) {
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
		strings.EqualFold(destination, STPMulticastMAC) ||
		strings.EqualFold(destination, CDPMulticastMAC) ||
		strings.EqualFold(destination, EDPMulticastMAC) ||
		strings.EqualFold(destination, FDPMulticastMAC)
}

// advertisesAtClient reports whether device is at the other end of the
// client's cable, the only place a link-local advertisement can come from. On
// a port pool that is the pool's switch alone; every other device on the
// attachment network is at least one hop further away. A network-scoped
// attachment names no port, so any device on the network may be the one.
func (r *fabricRuntime) advertisesAtClient(device *config.Device) bool {
	if r.placement != nil {
		return device.Name == r.placement.device
	}
	return r.deviceOnAttachment(device)
}

// advertisedPortName is the port a discovery advertisement from device names.
// On the pool's switch that is the client's pool port, which is where the
// cable is whatever the config says; otherwise the authored port ID, then the
// device's first interface, then fallback.
func (s *Stack) advertisedPortName(device *config.Device, authored, fallback string) string {
	if s.fabric != nil && s.fabric.placement != nil && device.Name == s.fabric.placement.device {
		return s.fabric.placement.advertisedPort().Interface
	}
	switch {
	case authored != "":
		return authored
	case len(device.Interfaces) > 0 && device.Interfaces[0].Name != "":
		return device.Interfaces[0].Name
	default:
		return fallback
	}
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
