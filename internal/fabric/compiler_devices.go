package fabric

import (
	"fmt"
	"net"
	"net/netip"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

const (
	bitsPerByte                     = 8
	maxUsableEndpointPrefixLen      = 30
	fullByteMask               byte = 0xff
)

func (c *scenarioCompiler) compileInterfaces(device *config.Device) map[string]Interface {
	compiled := make(map[string]Interface)
	names := make(map[string]struct{})
	for i, source := range device.Interfaces {
		if source.Network == "" && source.Address == "" {
			continue
		}
		field := fmt.Sprintf("devices[%s].interfaces[%d]", device.Name, i)
		if _, exists := names[source.Name]; exists {
			c.add(CodeDuplicateInterface, field+".name", "interface name must be unique per device")
			continue
		}
		names[source.Name] = struct{}{}
		iface, ok := c.compileInterface(device.Name, source, field)
		if !ok {
			continue
		}
		compiled[iface.Name] = iface
		c.report.Topology.Interfaces = append(c.report.Topology.Interfaces, iface)
		if device.Type == "router" {
			c.report.Topology.Routes = append(c.report.Topology.Routes, Route{
				Device: device.Name, Destination: c.networks[iface.Network].Prefix,
				Via: iface.Name, Connected: true,
			})
		}
	}
	return compiled
}

func (c *scenarioCompiler) compileInterface(
	device string,
	source config.Interface,
	field string,
) (Interface, bool) {
	network, exists := c.networks[source.Network]
	if !exists {
		c.add(CodeUnknownNetwork, field+".network", "interface references an unknown network")
		return Interface{}, false
	}
	address, err := netip.ParsePrefix(source.Address)
	if err != nil || !address.Addr().Is4() {
		c.add(CodeInvalidInterfaceAddress, field+".address", "address must be an IPv4 prefix")
		return Interface{}, false
	}
	if !network.Prefix.Contains(address.Addr()) {
		c.add(CodeAddressOutsideNetwork, field+".address", "address is outside its network")
		return Interface{}, false
	}
	if address.Bits() != network.Prefix.Bits() {
		c.add(
			CodeInterfacePrefixMismatch,
			field+".address",
			"address prefix length must match its network",
		)
		return Interface{}, false
	}
	if isReservedEndpoint(network.Prefix, address.Addr()) {
		c.add(
			CodeReservedInterfaceAddr,
			field+".address",
			"interface cannot use the network or broadcast address",
		)
		return Interface{}, false
	}
	if owner, assigned := c.addresses[address.Addr()]; assigned {
		c.add(
			CodeDuplicateInterfaceAddr,
			field+".address",
			fmt.Sprintf("interface address is already assigned to device %s", owner),
		)
		return Interface{}, false
	}
	c.addresses[address.Addr()] = device
	return Interface{
		Device:  device,
		Name:    source.Name,
		Network: source.Network,
		Address: address,
	}, true
}

func (c *scenarioCompiler) compileRoutes(device *config.Device, interfaces map[string]Interface) {
	for i, source := range device.Routes {
		field := fmt.Sprintf("devices[%s].routes[%d]", device.Name, i)
		route, diagnostic := ValidateStaticRoute(
			StaticRouteSpec{
				Device: device.Name, Destination: source.Destination,
				Via: source.Via, NextHop: source.NextHop,
			},
			RouteValidationContext{Interfaces: interfaces, AddressOwners: c.addresses},
		)
		if diagnostic != nil {
			c.add(diagnostic.Code, field+"."+diagnostic.Field, diagnostic.Message)
			continue
		}
		c.report.Topology.Routes = append(c.report.Topology.Routes, route)
	}
}

func (c *scenarioCompiler) compileDHCP(device *config.Device, interfaces map[string]Interface) {
	if device.DHCPConfig == nil || !hasDHCPPool(device.DHCPConfig) {
		return
	}
	start, startOK := ipToAddr(device.DHCPConfig.PoolStart)
	end, endOK := ipToAddr(device.DHCPConfig.PoolEnd)
	if !startOK || !endOK {
		c.add(
			CodeDHCPPoolOutsideNetwork,
			"devices."+device.Name+".dhcp",
			"DHCP pool must be inside its network",
		)
		return
	}
	candidates := make(map[string]Interface)
	for _, iface := range interfaces {
		network := c.networks[iface.Network]
		if network.Prefix.Contains(start) && network.Prefix.Contains(end) {
			candidates[iface.Network] = iface
		}
	}
	if len(candidates) == 0 {
		c.add(
			CodeDHCPPoolOutsideNetwork,
			"devices."+device.Name+".dhcp",
			"DHCP pool must be inside one routed network",
		)
		return
	}
	if len(candidates) > 1 {
		c.add(
			CodeDHCPNetworkAmbiguous,
			"devices."+device.Name+".dhcp",
			"DHCP pool must identify exactly one routed network",
		)
		return
	}
	for _, iface := range candidates {
		c.appendDHCPScope(device, iface)
	}
}

func hasDHCPPool(cfg *config.DHCPConfig) bool {
	return cfg.PoolStart != nil || cfg.PoolEnd != nil
}

func (c *scenarioCompiler) appendDHCPScope(device *config.Device, iface Interface) {
	network := c.networks[iface.Network]
	start, startOK := ipToAddr(device.DHCPConfig.PoolStart)
	end, endOK := ipToAddr(device.DHCPConfig.PoolEnd)
	if !startOK || !endOK || !network.Prefix.Contains(start) || !network.Prefix.Contains(end) {
		c.add(
			CodeDHCPPoolOutsideNetwork,
			"devices."+device.Name+".dhcp",
			"DHCP pool must be inside its network",
		)
		return
	}
	if start.Compare(end) > 0 {
		c.add(CodeInvalidDHCPRange, "devices."+device.Name+".dhcp", "DHCP pool start must not follow end")
		return
	}
	if isReservedEndpoint(network.Prefix, start) || isReservedEndpoint(network.Prefix, end) {
		c.add(
			CodeReservedDHCPAddress,
			"devices."+device.Name+".dhcp",
			"DHCP pool cannot include the network or broadcast address",
		)
		return
	}
	router, routerOK := ipToAddr(device.DHCPConfig.Router)
	if !routerOK || !network.Prefix.Contains(router) || isReservedEndpoint(network.Prefix, router) {
		c.add(
			CodeInvalidDHCPRouter,
			"devices."+device.Name+".dhcp.router",
			"DHCP router must be a usable address inside its network",
		)
		return
	}
	for address, owner := range c.addresses {
		if address.Compare(start) >= 0 && address.Compare(end) <= 0 {
			c.add(
				CodeDHCPAddressCollision,
				"devices."+device.Name+".dhcp",
				fmt.Sprintf("DHCP pool contains an interface address assigned to device %s", owner),
			)
			return
		}
	}
	for _, scope := range c.report.Topology.DHCPScopes {
		if scope.Network == iface.Network && start.Compare(scope.End) <= 0 && end.Compare(scope.Start) >= 0 {
			c.add(
				CodeDHCPAddressCollision,
				"devices."+device.Name+".dhcp",
				fmt.Sprintf("DHCP pool overlaps the pool assigned to device %s", scope.Device),
			)
			return
		}
	}
	c.report.Topology.DHCPScopes = append(c.report.Topology.DHCPScopes, DHCPScope{
		Device: device.Name, Network: iface.Network, Start: start, End: end, Router: router,
	})
}

func isReservedEndpoint(prefix netip.Prefix, address netip.Addr) bool {
	if prefix.Bits() > maxUsableEndpointPrefixLen {
		return false
	}
	return address == prefix.Addr() || address == broadcastAddress(prefix)
}

func broadcastAddress(prefix netip.Prefix) netip.Addr {
	bytes := prefix.Addr().As4()
	wholeBytes := prefix.Bits() / bitsPerByte
	remainingBits := prefix.Bits() % bitsPerByte
	if remainingBits != 0 {
		bytes[wholeBytes] |= fullByteMask >> uint(remainingBits)
		wholeBytes++
	}
	for i := wholeBytes; i < len(bytes); i++ {
		bytes[i] = fullByteMask
	}
	return netip.AddrFrom4(bytes)
}

func ipToAddr(ip net.IP) (netip.Addr, bool) {
	addr, ok := netip.AddrFromSlice(ip)
	return addr.Unmap(), ok
}
