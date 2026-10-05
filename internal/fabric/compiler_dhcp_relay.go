package fabric

import (
	"fmt"
	"maps"
	"net/netip"
	"slices"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/deviceclass"
)

// dhcpRelaySource is one authored `dhcp_relay` on a compiled interface.
type dhcpRelaySource struct {
	device *config.Device
	iface  Interface
	target string
	field  string
	// address is target, parsed once the relay is validated.
	address netip.Addr
}

// compileRelayedDHCP binds every `dhcp.scopes` pool to the router interface
// that relays its network to the server. It runs after every device's
// interfaces are known, because the relay and the server are usually
// different devices.
func (c *scenarioCompiler) compileRelayedDHCP(interfacesByDevice []map[string]Interface) {
	relays := c.validDHCPRelays()
	used := make([]bool, len(relays))
	for i := range c.cfg.Devices {
		device := &c.cfg.Devices[i]
		if interfacesByDevice[i] == nil || device.DHCPConfig == nil {
			continue
		}
		for index, scope := range device.DHCPConfig.Scopes {
			field := fmt.Sprintf("devices.%s.dhcp.scopes[%d]", device.Name, index)
			c.compileRelayedScope(device, interfacesByDevice[i], scope, field, relays, used)
		}
	}
	for i, relay := range relays {
		if !used[i] {
			c.add(
				CodeInvalidDHCPRelay,
				relay.field,
				"DHCP relay target holds no DHCP scope on this interface's network",
			)
		}
	}
}

func (c *scenarioCompiler) validDHCPRelays() []dhcpRelaySource {
	valid := make([]dhcpRelaySource, 0, len(c.dhcpRelays))
	for _, relay := range c.dhcpRelays {
		if !deviceclass.RoutesIP(deviceclass.Parse(relay.device.Type)) {
			c.add(CodeInvalidDHCPRelay, relay.field, "only a routing device relays DHCP")
			continue
		}
		target, err := netip.ParseAddr(relay.target)
		if err != nil || !target.Is4() {
			c.add(CodeInvalidDHCPRelay, relay.field, "DHCP relay target must be an IPv4 address")
			continue
		}
		relay.address = target
		valid = append(valid, relay)
	}
	return valid
}

func (c *scenarioCompiler) compileRelayedScope(
	device *config.Device,
	interfaces map[string]Interface,
	scope config.DHCPScope,
	field string,
	relays []dhcpRelaySource,
	used []bool,
) {
	start, startOK := ipToAddr(scope.PoolStart)
	end, endOK := ipToAddr(scope.PoolEnd)
	network, found := c.networkContaining(start)
	if !startOK || !endOK || !found || !network.Prefix.Contains(end) {
		c.add(CodeDHCPPoolOutsideNetwork, field, "DHCP scope must be inside one routed network")
		return
	}
	if start.Compare(end) > 0 {
		c.add(CodeInvalidDHCPRange, field, "DHCP pool start must not follow end")
		return
	}
	if isReservedEndpoint(network.Prefix, start) || isReservedEndpoint(network.Prefix, end) {
		c.add(CodeReservedDHCPAddress, field, "DHCP pool cannot include the network or broadcast address")
		return
	}
	router, routerOK := ipToAddr(scope.Router)
	if !routerOK || !network.Prefix.Contains(router) || isReservedEndpoint(network.Prefix, router) {
		c.add(CodeInvalidDHCPRouter, field+".router", "DHCP router must be a usable address inside its network")
		return
	}
	serverAddresses := make(map[netip.Addr]bool, len(interfaces))
	for _, iface := range interfaces {
		if iface.Network == network.Name {
			c.add(
				CodeInvalidDHCPRelay,
				field,
				"DHCP scope is on a network this server is attached to; author it as the server's own pool",
			)
			return
		}
		serverAddresses[iface.Address.Addr()] = true
	}
	relaysToServer := func(relay dhcpRelaySource) bool {
		return relay.iface.Network == network.Name && serverAddresses[relay.address]
	}
	relayIndex := slices.IndexFunc(relays, relaysToServer)
	if relayIndex < 0 {
		c.add(
			CodeInvalidDHCPRelay,
			field,
			fmt.Sprintf("no router interface on network %s relays DHCP to this server", network.Name),
		)
		return
	}
	if !c.validateDHCPPoolCollisions(device, Interface{Network: network.Name}, start, end) {
		return
	}
	for i, relay := range relays {
		if relaysToServer(relay) {
			used[i] = true
		}
	}
	relay := relays[relayIndex]
	c.report.Topology.DHCPScopes = append(c.report.Topology.DHCPScopes, DHCPScope{
		Device: device.Name, Network: network.Name, Start: start, End: end, Router: router,
		Relay: DHCPRelay{
			Device: relay.device.Name, Interface: relay.iface.Name,
			Address: relay.iface.Address.Addr(), Server: relay.address,
		},
	})
}

func (c *scenarioCompiler) networkContaining(address netip.Addr) (Network, bool) {
	for _, name := range slices.Sorted(maps.Keys(c.networks)) {
		if network := c.networks[name]; network.Prefix.Contains(address) {
			return network, true
		}
	}
	return Network{}, false
}
