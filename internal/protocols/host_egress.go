package protocols

import (
	"errors"
	"net/netip"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

var errHostEgressUnavailable = errors.New("faulted host has no active local egress route")

type hostEgressRoute struct {
	iface  devicestate.Interface
	target netip.Addr
}

func hostMaskRole(device *config.Device) bool {
	switch device.Type {
	case "router", "layer3-switch", "firewall":
		return false
	default:
		return true
	}
}

func (s *Stack) hostPacketRoute(packet *Packet) (hostEgressRoute, bool, error) {
	var empty hostEgressRoute
	device := packet.generatedHost
	if device == nil || !hostMaskRole(device) {
		return empty, false, nil
	}
	store := s.deviceStates[device]
	if store == nil {
		if packet.hostEgressInterface != "" {
			return empty, true, errHostEgressUnavailable
		}
		return empty, false, nil
	}
	if packet.hostEgressInterface == "" && !store.HasInterfacePrefixFaults() {
		return empty, false, nil
	}
	snapshot := store.Snapshot()
	if len(snapshot.PrefixFaults) == 0 && packet.hostEgressInterface == "" {
		return empty, false, nil
	}
	source, destination := hostPacketAddresses(packet)
	if !source.IsValid() {
		return empty, false, nil
	}
	if packet.hostEgressInterface != "" && (!pendingHostSourceValid(snapshot, packet, source.Unmap()) ||
		!s.conflictInterfaceAvailable(device, packet.hostEgressInterface) || !s.hostPacketVLAN(packet)) {
		return empty, true, errHostEgressUnavailable
	}
	iface, faulted := hostFaultInterface(snapshot, source.Unmap())
	if faulted {
		if !s.conflictInterfaceAvailable(device, iface.Name) || !s.hostPacketVLAN(packet) {
			return empty, true, errHostEgressUnavailable
		}
		return hostMaskRoute(snapshot, iface, destination.Unmap())
	}
	return empty, false, nil
}

func hostPacketAddresses(packet *Packet) (netip.Addr, netip.Addr) {
	decoded := gopacket.NewPacket(packet.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	ip, _ := decoded.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	eth, _ := decoded.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	if ip == nil || eth == nil || len(eth.DstMAC) != SizeOfMac || eth.DstMAC[0]&1 != 0 {
		return netip.Addr{}, netip.Addr{}
	}
	source, sourceOK := netip.AddrFromSlice(ip.SrcIP)
	destination, destinationOK := netip.AddrFromSlice(ip.DstIP)
	if !sourceOK || !destinationOK || !devicestate.ValidFaultAddress(destination.Unmap()) {
		return netip.Addr{}, netip.Addr{}
	}
	return source.Unmap(), destination.Unmap()
}

func hostFaultInterface(snapshot devicestate.Snapshot, source netip.Addr) (devicestate.Interface, bool) {
	for _, fault := range snapshot.PrefixFaults {
		for _, iface := range snapshot.Network.Interfaces {
			if iface.Name == fault.Interface && iface.Address.Addr() == source {
				return iface, true
			}
		}
	}
	return devicestate.Interface{}, false
}

func pendingHostSourceValid(snapshot devicestate.Snapshot, packet *Packet, source netip.Addr) bool {
	for _, iface := range snapshot.Network.Interfaces {
		if iface.Name == packet.hostEgressInterface {
			return iface.AdminUp && iface.OperUp && iface.Address.Addr() == source &&
				iface.Network == packet.hostEgressNetwork
		}
	}
	return false
}

func (s *Stack) hostPacketVLAN(packet *Packet) bool {
	expected := s.discoveryVLAN(packet.generatedHost)
	if s.fabric != nil {
		expected = int(s.fabric.binding.AccessVLAN)
		if !s.fabric.binding.WireTagged {
			expected = config.UntaggedTag
		}
	}
	return newNotificationNeighborKey(
		packet.VLAN,
		netip.Addr{},
	).vlan == newNotificationNeighborKey(
		expected,
		netip.Addr{},
	).vlan
}

func hostMaskRoute(
	snapshot devicestate.Snapshot,
	original devicestate.Interface,
	destination netip.Addr,
) (hostEgressRoute, bool, error) {
	network := snapshot.EffectiveHostNetwork()
	for _, iface := range network.Interfaces {
		if iface.Name != original.Name {
			continue
		}
		bestBits := -1
		target := netip.Addr{}
		if iface.Address.Contains(destination) {
			bestBits, target = iface.Address.Bits(), destination
		}
		for _, route := range network.Routes {
			if route.Via == iface.Name && route.NextHop.Is4() && route.Destination.Contains(destination) &&
				route.Destination.Bits() > bestBits {
				bestBits, target = route.Destination.Bits(), route.NextHop
			}
		}
		if !target.IsValid() {
			return hostEgressRoute{}, true, errHostEgressUnavailable
		}
		return hostEgressRoute{iface: iface, target: target}, true, nil
	}
	return hostEgressRoute{}, true, errHostEgressUnavailable
}
