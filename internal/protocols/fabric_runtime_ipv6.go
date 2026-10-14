package protocols

import (
	"bytes"
	"net"
	"net/netip"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// IPv6 on the fabric follows the IPv4 rules, keyed by each interface's
// address_v6: a neighbour answers only on the attachment network, and an
// off-link destination is reached through a router there (niac-go#2411). The
// fabric authors no IPv6 static routes, so that router reaches only the
// networks its own interfaces number.

func (r *fabricRuntime) indexInterfacesV6(interfaces []fabric.Interface) {
	r.interfacesByAddrV6 = make(map[netip.Addr]fabricEndpoint)
	for _, iface := range interfaces {
		device := r.devicesByName[iface.Device]
		if device == nil || !iface.AddressV6.IsValid() {
			continue
		}
		r.interfacesByAddrV6[iface.AddressV6.Addr()] = fabricEndpoint{
			device: device, interfaceName: iface.Name,
			network: iface.Network, mac: cloneMAC(device.MACAddress),
		}
	}
}

// acceptsIPv6Source admits what a host on the attachment network can send
// from: its link-local address, an address in the network's IPv6 prefix, or
// the unspecified address, which duplicate address detection sends only to a
// multicast group.
func (r *fabricRuntime) acceptsIPv6Source(sourceIP, destinationIP net.IP) bool {
	source, sourceOK := netip.AddrFromSlice(sourceIP)
	destination, destinationOK := netip.AddrFromSlice(destinationIP)
	if !sourceOK || !destinationOK || !source.Is6() || source.Is4In6() || source.IsMulticast() {
		return false
	}
	if source.IsUnspecified() {
		return destination.IsMulticast()
	}
	if source.IsLinkLocalUnicast() {
		return true
	}
	for _, network := range r.topology.Networks {
		if network.Name == r.attachmentNetwork {
			return network.PrefixV6.IsValid() && network.PrefixV6.Contains(source)
		}
	}
	return false
}

func (r *fabricRuntime) resolveIPv6(
	dst netip.Addr,
	ingressMAC net.HardwareAddr,
) (fabricResolution, bool) {
	if dst.IsLinkLocalUnicast() {
		device, found := r.resolveLinkLocal(dst)
		if !found {
			return fabricResolution{}, false
		}
		return fabricResolution{
			device: device, replySourceMAC: cloneMAC(device.MACAddress), egressNetwork: r.attachmentNetwork,
		}, true
	}
	endpoint, exists := r.interfacesByAddrV6[dst]
	if !exists || !r.interfaceAvailable(endpoint.device, endpoint.interfaceName) {
		return fabricResolution{}, false
	}
	if endpoint.network == r.attachmentNetwork {
		return fabricResolution{
			device: endpoint.device, replySourceMAC: cloneMAC(endpoint.mac), egressNetwork: endpoint.network,
		}, true
	}

	router, via := r.routeForV6(dst, ingressMAC)
	if router == nil {
		return fabricResolution{}, false
	}
	firstHopIP, _ := netip.AddrFromSlice(linkLocalAddress(router.device))
	return fabricResolution{
		device: endpoint.device, replySourceMAC: cloneMAC(router.mac), routed: true,
		firstHopIP:  firstHopIP,
		firstHopMAC: cloneMAC(router.mac), firstHopDevice: router.device,
		egressNetwork: endpoint.network, routeVia: via,
	}, true
}

// routeForV6 is the attachment router the frame was addressed to, when one of
// its usable interfaces numbers the destination's network: its connected
// IPv6 route, longest prefix first.
func (r *fabricRuntime) routeForV6(dst netip.Addr, ingressMAC net.HardwareAddr) (*fabricRouter, string) {
	bestBits := -1
	var best *fabricRouter
	var bestVia string
	for i := range r.attachmentRouters {
		router := &r.attachmentRouters[i]
		if !bytes.Equal(router.mac, ingressMAC) ||
			!r.interfaceAvailable(router.device, router.attachmentInterface) {
			continue
		}
		for _, iface := range r.topology.Interfaces {
			if iface.Device != router.device.Name || !iface.AddressV6.IsValid() ||
				!iface.AddressV6.Contains(dst) || iface.AddressV6.Bits() <= bestBits ||
				!r.interfaceAvailable(router.device, iface.Name) {
				continue
			}
			bestBits = iface.AddressV6.Bits()
			best = router
			bestVia = iface.Name
		}
	}
	return best, bestVia
}

// resolveNeighbor is the device that answers a Neighbor Solicitation: only an
// address on the attachment network is on the tester's link.
func (r *fabricRuntime) resolveNeighbor(target net.IP) (*config.Device, bool) {
	address, ok := netip.AddrFromSlice(target)
	if !ok {
		return nil, false
	}
	if address.IsLinkLocalUnicast() {
		return r.resolveLinkLocal(address)
	}
	endpoint, exists := r.interfacesByAddrV6[address]
	if !exists || endpoint.network != r.attachmentNetwork ||
		!r.interfaceAvailable(endpoint.device, endpoint.interfaceName) {
		return nil, false
	}
	return endpoint.device, true
}

// resolveLinkLocal is the device on the attachment network whose link-local
// address is target, authored or derived from its MAC. It must speak IPv6
// there: an address_v6 on that interface, or an IPv6 address in its ips list,
// which is what devicesAtLinkLocal asks without a fabric.
func (r *fabricRuntime) resolveLinkLocal(target netip.Addr) (*config.Device, bool) {
	ip := net.IP(target.AsSlice())
	for _, iface := range r.topology.Interfaces {
		if iface.Network != r.attachmentNetwork {
			continue
		}
		device := r.devicesByName[iface.Device]
		if device == nil || !linkLocalAddress(device).Equal(ip) ||
			!r.interfaceAvailable(device, iface.Name) {
			continue
		}
		if iface.AddressV6.IsValid() || firstIPv6Address(device) != nil {
			return device, true
		}
	}
	return nil, false
}

func (r *fabricRuntime) deviceOwnsIPv6(device *config.Device, ip net.IP) bool {
	address, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	if address.IsLinkLocalUnicast() {
		owner, found := r.resolveLinkLocal(address)
		return found && owner == device
	}
	endpoint, exists := r.interfacesByAddrV6[address]
	return exists && endpoint.device == device
}
