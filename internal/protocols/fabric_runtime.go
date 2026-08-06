package protocols

import (
	"bytes"
	"net"
	"net/netip"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

type fabricRuntime struct {
	binding           fabric.CompiledBinding
	topology          fabric.Topology
	attachmentNetwork string
	devicesByName     map[string]*config.Device
	interfacesByAddr  map[netip.Addr]fabricEndpoint
	attachmentRouters []fabricRouter
	attachmentDHCP    *config.Device
	deviceStates      map[*config.Device]*devicestate.Store
}

type fabricEndpoint struct {
	device        *config.Device
	interfaceName string
	network       string
	mac           net.HardwareAddr
}

type fabricRouter struct {
	device              *config.Device
	attachmentIP        netip.Addr
	attachmentInterface string
	mac                 net.HardwareAddr
	routes              []fabricRoute
}

type fabricRoute struct {
	destination netip.Prefix
	via         string
	nextHop     netip.Addr
	connected   bool
}

const ipv4PointToPointPrefixBits = 30

type fabricResolution struct {
	device         *config.Device
	replySourceMAC net.HardwareAddr
	routed         bool
	firstHopIP     netip.Addr
	firstHopMAC    net.HardwareAddr
	firstHopDevice *config.Device
	egressNetwork  string
	routeVia       string
}

type fabricNotificationEgress struct {
	device *config.Device
	source netip.Addr
	target netip.Addr
}

func newFabricRuntime(topology *fabric.Topology, cfg *config.Config) *fabricRuntime {
	if topology == nil || cfg == nil {
		return nil
	}

	runtime := &fabricRuntime{
		binding: topology.Binding,
		topology: fabric.Topology{
			Binding:    topology.Binding,
			Networks:   append([]fabric.Network(nil), topology.Networks...),
			Interfaces: append([]fabric.Interface(nil), topology.Interfaces...),
			Routes:     append([]fabric.Route(nil), topology.Routes...),
			DHCPScopes: append([]fabric.DHCPScope(nil), topology.DHCPScopes...),
		},
		attachmentNetwork: topology.Binding.Network,
		devicesByName:     make(map[string]*config.Device, len(cfg.Devices)),
		interfacesByAddr:  make(map[netip.Addr]fabricEndpoint, len(topology.Interfaces)),
	}
	for i := range cfg.Devices {
		device := &cfg.Devices[i]
		runtime.devicesByName[device.Name] = device
	}
	runtime.indexInterfaces(topology.Interfaces)
	runtime.indexAttachmentRouters(topology)
	runtime.indexAttachmentDHCP(topology.DHCPScopes)

	return runtime
}

func (r *fabricRuntime) acceptsFrame(vlan int, tagged bool) bool {
	if r == nil {
		return true
	}

	switch r.binding.Mode {
	case fabric.ModeDirect, fabric.ModeAccess:
		return !tagged && vlan <= 0
	case fabric.ModeTrunk:
		return tagged && vlan == int(r.binding.AccessVLAN)
	default:
		return false
	}
}

func (r *fabricRuntime) acceptsIPv4Source(sourceIP, destinationIP net.IP, protocol uint8) bool {
	if r == nil {
		return true
	}
	source, sourceOK := netip.AddrFromSlice(sourceIP)
	destination, destinationOK := netip.AddrFromSlice(destinationIP)
	if !sourceOK || !destinationOK {
		return false
	}
	source = source.Unmap()
	destination = destination.Unmap()
	if source == netip.IPv4Unspecified() {
		return protocol == IPProtocolUDP &&
			destination == netip.AddrFrom4([4]byte{0xff, 0xff, 0xff, 0xff})
	}
	if !source.Is4() || source.IsMulticast() {
		return false
	}
	for _, network := range r.topology.Networks {
		if network.Name == r.attachmentNetwork {
			return validAttachmentHost(network.Prefix, source)
		}
	}
	return false
}

func validAttachmentHost(prefix netip.Prefix, address netip.Addr) bool {
	if !prefix.IsValid() || !prefix.Addr().Is4() || !prefix.Contains(address) {
		return false
	}
	if prefix.Bits() > ipv4PointToPointPrefixBits {
		return true
	}
	network := prefix.Masked().Addr().As4()
	host := address.As4()
	if host == network {
		return false
	}
	mask := net.CIDRMask(prefix.Bits(), address.BitLen())
	broadcast := network
	for index := range broadcast {
		broadcast[index] |= ^mask[index]
	}
	return host != broadcast
}

func (r *fabricRuntime) indexAttachmentDHCP(scopes []fabric.DHCPScope) {
	for _, scope := range scopes {
		if scope.Network == r.attachmentNetwork {
			r.attachmentDHCP = r.devicesByName[scope.Device]
			return
		}
	}
}

func (r *fabricRuntime) indexInterfaces(interfaces []fabric.Interface) {
	for _, iface := range interfaces {
		device := r.devicesByName[iface.Device]
		if device == nil {
			continue
		}
		r.interfacesByAddr[iface.Address.Addr()] = fabricEndpoint{
			device: device, interfaceName: iface.Name,
			network: iface.Network, mac: cloneMAC(device.MACAddress),
		}
	}
}

func (r *fabricRuntime) indexAttachmentRouters(topology *fabric.Topology) {
	routers := make(map[string]*fabricRouter)
	for _, iface := range topology.Interfaces {
		device := r.devicesByName[iface.Device]
		if iface.Network != r.attachmentNetwork || device == nil || device.Type != "router" {
			continue
		}
		router := &fabricRouter{
			device: device, attachmentIP: iface.Address.Addr(), attachmentInterface: iface.Name,
			mac: cloneMAC(device.MACAddress),
		}
		routers[iface.Device] = router
		r.attachmentRouters = append(r.attachmentRouters, *router)
	}
	for _, route := range topology.Routes {
		router := routers[route.Device]
		if router != nil {
			router.routes = append(router.routes, fabricRoute{
				destination: route.Destination,
				via:         route.Via,
				nextHop:     route.NextHop,
				connected:   route.Connected,
			})
		}
	}
	for i := range r.attachmentRouters {
		if router := routers[r.attachmentRouters[i].device.Name]; router != nil {
			r.attachmentRouters[i].routes = append([]fabricRoute(nil), router.routes...)
		}
	}
}

func (r *fabricRuntime) resolveIPv4(
	dst netip.Addr,
	ingressMAC net.HardwareAddr,
) (fabricResolution, bool) {
	if r == nil || !dst.Is4() {
		return fabricResolution{}, false
	}
	endpoint, exists := r.endpointForAddress(dst)
	if !exists {
		return fabricResolution{}, false
	}
	if !r.interfaceAvailable(endpoint.device, endpoint.interfaceName) {
		return fabricResolution{}, false
	}
	if endpoint.network == r.attachmentNetwork {
		return fabricResolution{
			device: endpoint.device, replySourceMAC: cloneMAC(endpoint.mac), egressNetwork: endpoint.network,
		}, true
	}

	router, route := r.routeFor(dst, ingressMAC)
	if router == nil {
		return fabricResolution{}, false
	}
	firstHopIP, found := r.interfaceAddress(
		router.device,
		router.attachmentInterface,
		router.attachmentIP,
	)
	if !found {
		return fabricResolution{}, false
	}
	return fabricResolution{
		device: endpoint.device, replySourceMAC: cloneMAC(router.mac), routed: true,
		firstHopIP:  firstHopIP,
		firstHopMAC: cloneMAC(router.mac), firstHopDevice: router.device,
		egressNetwork: endpoint.network, routeVia: route.via,
	}, true
}

func (r *fabricRuntime) notificationEgress(
	routerDevice *config.Device,
	destination netip.Addr,
) (fabricNotificationEgress, bool) {
	if r == nil || routerDevice == nil || !destination.Is4() {
		return fabricNotificationEgress{}, false
	}
	for i := range r.attachmentRouters {
		router := &r.attachmentRouters[i]
		if router.device != routerDevice ||
			!r.interfaceAvailable(router.device, router.attachmentInterface) {
			continue
		}
		for _, route := range r.routesFor(router) {
			if !route.connected || route.via != router.attachmentInterface ||
				!route.destination.Contains(destination) {
				continue
			}
			source, found := r.interfaceAddress(
				router.device,
				router.attachmentInterface,
				router.attachmentIP,
			)
			if !found {
				return fabricNotificationEgress{}, false
			}
			return fabricNotificationEgress{
				device: router.device,
				source: source,
				target: destination,
			}, true
		}
	}
	return fabricNotificationEgress{}, false
}

func (r *fabricRuntime) interfaceAddress(
	device *config.Device,
	name string,
	fallback netip.Addr,
) (netip.Addr, bool) {
	if state := r.deviceStates[device]; state != nil {
		for _, iface := range state.Snapshot().Network.Interfaces {
			if iface.Name == name {
				return iface.Address.Addr(), iface.Address.IsValid()
			}
		}
		return netip.Addr{}, false
	}
	return fallback, fallback.IsValid()
}

func (r *fabricRuntime) deviceOwnsIPv4(device *config.Device, ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	endpoint, exists := r.endpointForAddress(addr.Unmap())
	return exists && endpoint.device == device
}

func (r *fabricRuntime) resolveARP(ip net.IP) (*config.Device, bool) {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return nil, false
	}
	endpoint, exists := r.endpointForAddress(addr.Unmap())
	if !exists || endpoint.network != r.attachmentNetwork ||
		!r.interfaceAvailable(endpoint.device, endpoint.interfaceName) {
		return nil, false
	}
	return endpoint.device, true
}

func (r *fabricRuntime) endpointForAddress(address netip.Addr) (fabricEndpoint, bool) {
	if r.deviceStates == nil {
		endpoint, found := r.interfacesByAddr[address]
		return endpoint, found
	}
	for _, endpoint := range r.interfacesByAddr {
		state := r.deviceStates[endpoint.device]
		if state == nil {
			continue
		}
		for _, iface := range state.Snapshot().Network.Interfaces {
			if iface.Name == endpoint.interfaceName && iface.Address.IsValid() &&
				iface.Address.Addr() == address {
				return endpoint, true
			}
		}
	}
	return fabricEndpoint{}, false
}

func (r *fabricRuntime) routeFor(
	dst netip.Addr,
	ingressMAC net.HardwareAddr,
) (*fabricRouter, fabricRoute) {
	bestBits := -1
	var best *fabricRouter
	var bestRoute fabricRoute
	for i := range r.attachmentRouters {
		router := &r.attachmentRouters[i]
		if !bytes.Equal(router.mac, ingressMAC) ||
			!r.interfaceAvailable(router.device, router.attachmentInterface) {
			continue
		}
		for _, route := range r.routesFor(router) {
			if r.routeAvailable(router, route) &&
				route.destination.Contains(dst) && route.destination.Bits() > bestBits {
				bestBits = route.destination.Bits()
				best = router
				bestRoute = route
			}
		}
	}
	return best, bestRoute
}

func (r *fabricRuntime) routeAvailable(router *fabricRouter, route fabricRoute) bool {
	if !r.interfaceAvailable(router.device, route.via) {
		return false
	}
	if r.deviceStates != nil && !route.connected &&
		!validateStaticRoute(r.deviceStates, router.device, devicestate.Route{
			Destination: route.destination,
			Via:         route.via,
			NextHop:     route.nextHop,
		}) {
		return false
	}
	if route.connected {
		return true
	}
	nextHop, exists := r.endpointForAddress(route.nextHop)
	return exists && nextHop.device != router.device &&
		r.interfaceAvailable(nextHop.device, nextHop.interfaceName)
}

func (r *fabricRuntime) routesFor(router *fabricRouter) []fabricRoute {
	if r.deviceStates == nil {
		return router.routes
	}
	state := r.deviceStates[router.device]
	if state == nil {
		return nil
	}
	routes := state.Snapshot().Network.Routes
	result := make([]fabricRoute, 0, len(routes))
	for _, route := range routes {
		result = append(result, fabricRoute{
			destination: route.Destination,
			via:         route.Via,
			nextHop:     route.NextHop,
			connected:   route.Connected,
		})
	}
	return result
}

func (s *Stack) staticRouteValidator(device *config.Device) func(devicestate.Route) bool {
	return func(route devicestate.Route) bool {
		return validateStaticRoute(s.deviceStates, device, route)
	}
}

func validateStaticRoute(
	states map[*config.Device]*devicestate.Store,
	device *config.Device,
	route devicestate.Route,
) bool {
	state := states[device]
	if state == nil {
		return false
	}
	interfaces := make(map[string]fabric.Interface)
	for _, iface := range state.Snapshot().Network.Interfaces {
		interfaces[iface.Name] = fabric.Interface{
			Device: device.Name, Name: iface.Name, Network: iface.Network, Address: iface.Address,
		}
	}
	owners := make(map[netip.Addr]string)
	duplicates := make(map[netip.Addr]struct{})
	for candidate, candidateState := range states {
		for _, iface := range candidateState.Snapshot().Network.Interfaces {
			if iface.Address.IsValid() {
				address := iface.Address.Addr()
				if _, exists := owners[address]; exists {
					duplicates[address] = struct{}{}
				}
				owners[address] = candidate.Name
			}
		}
	}
	for address := range duplicates {
		delete(owners, address)
	}
	_, diagnostic := fabric.ValidateStaticRoute(
		fabric.StaticRouteSpec{
			Device: device.Name, Destination: route.Destination.String(),
			Via: route.Via, NextHop: route.NextHop.String(),
		},
		fabric.RouteValidationContext{Interfaces: interfaces, AddressOwners: owners},
	)
	return diagnostic == nil
}

func (r *fabricRuntime) bindDeviceStates(states map[*config.Device]*devicestate.Store) {
	r.deviceStates = states
}

func (r *fabricRuntime) interfaceAvailable(device *config.Device, name string) bool {
	if r.deviceStates == nil {
		return true
	}
	state := r.deviceStates[device]
	if state == nil {
		return false
	}
	for _, iface := range state.Snapshot().Network.Interfaces {
		if iface.Name == name {
			return iface.AdminUp && iface.OperUp
		}
	}
	return false
}

func cloneMAC(mac net.HardwareAddr) net.HardwareAddr {
	return append(net.HardwareAddr(nil), mac...)
}
