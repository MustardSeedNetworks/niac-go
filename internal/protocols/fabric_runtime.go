package protocols

import (
	"bytes"
	"net"
	"net/netip"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

type fabricRuntime struct {
	attachmentNetwork string
	devicesByName     map[string]*config.Device
	interfacesByAddr  map[netip.Addr]fabricEndpoint
	attachmentRouters []fabricRouter
	attachmentDHCP    *config.Device
}

type fabricEndpoint struct {
	device  *config.Device
	network string
	mac     net.HardwareAddr
}

type fabricRouter struct {
	device *config.Device
	mac    net.HardwareAddr
	routes []netip.Prefix
}

type fabricResolution struct {
	device         *config.Device
	replySourceMAC net.HardwareAddr
	routed         bool
}

func newFabricRuntime(topology *fabric.Topology, cfg *config.Config) *fabricRuntime {
	if topology == nil || cfg == nil {
		return nil
	}

	runtime := &fabricRuntime{
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
			device: device, network: iface.Network, mac: cloneMAC(device.MACAddress),
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
		router := &fabricRouter{device: device, mac: cloneMAC(device.MACAddress)}
		routers[iface.Device] = router
		r.attachmentRouters = append(r.attachmentRouters, *router)
	}
	for _, route := range topology.Routes {
		router := routers[route.Device]
		if router != nil {
			router.routes = append(router.routes, route.Destination)
		}
	}
	for i := range r.attachmentRouters {
		if router := routers[r.attachmentRouters[i].device.Name]; router != nil {
			r.attachmentRouters[i].routes = append([]netip.Prefix(nil), router.routes...)
		}
	}
}

func (r *fabricRuntime) resolveIPv4(dst netip.Addr, ingressMAC net.HardwareAddr) (fabricResolution, bool) {
	if r == nil || !dst.Is4() {
		return fabricResolution{}, false
	}
	endpoint, exists := r.interfacesByAddr[dst]
	if !exists {
		return fabricResolution{}, false
	}
	if endpoint.network == r.attachmentNetwork {
		return fabricResolution{
			device: endpoint.device, replySourceMAC: cloneMAC(endpoint.mac),
		}, true
	}

	router := r.routeFor(dst, ingressMAC)
	if router == nil {
		return fabricResolution{}, false
	}
	return fabricResolution{
		device: endpoint.device, replySourceMAC: cloneMAC(router.mac), routed: true,
	}, true
}

func (r *fabricRuntime) deviceOwnsIPv4(device *config.Device, ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	endpoint, exists := r.interfacesByAddr[addr.Unmap()]
	return exists && endpoint.device == device
}

func (r *fabricRuntime) resolveARP(ip net.IP) (*config.Device, bool) {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return nil, false
	}
	endpoint, exists := r.interfacesByAddr[addr.Unmap()]
	if !exists || endpoint.network != r.attachmentNetwork {
		return nil, false
	}
	return endpoint.device, true
}

func (r *fabricRuntime) routeFor(dst netip.Addr, ingressMAC net.HardwareAddr) *fabricRouter {
	bestBits := -1
	var best *fabricRouter
	for i := range r.attachmentRouters {
		router := &r.attachmentRouters[i]
		if !bytes.Equal(router.mac, ingressMAC) {
			continue
		}
		for _, route := range router.routes {
			if route.Contains(dst) && route.Bits() > bestBits {
				bestBits = route.Bits()
				best = router
			}
		}
	}
	return best
}

func cloneMAC(mac net.HardwareAddr) net.HardwareAddr {
	return append(net.HardwareAddr(nil), mac...)
}
