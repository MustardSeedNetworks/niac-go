package protocols

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

func (s *Stack) registerDeviceState(device *config.Device) {
	s.deviceStates[device] = devicestate.NewStore(deviceIdentity(device))
}

func deviceIdentity(device *config.Device) devicestate.Identity {
	hostname := device.SNMPConfig.SysName
	if hostname == "" {
		hostname = device.Properties["sysName"]
	}
	if hostname == "" {
		hostname = device.Name
	}

	return devicestate.Identity{Hostname: hostname}
}

func (s *Stack) deviceHostname(device *config.Device) string {
	if state := s.deviceStates[device]; state != nil {
		return state.Snapshot().Identity.Hostname
	}
	return device.Name
}

func (s *Stack) configureDeviceStates(topology *fabric.Topology) {
	for device, state := range s.deviceStates {
		if topology == nil {
			state.ReplaceNetwork(flatDeviceNetworkState(device))
			continue
		}
		state.ReplaceNetwork(deviceNetworkState(topology, device))
	}
}

func flatDeviceNetworkState(device *config.Device) devicestate.Network {
	interfaces := make([]devicestate.Interface, 0, max(len(device.Interfaces), len(device.IPAddresses)))
	routes := make([]devicestate.Route, 0, len(device.Routes)+len(device.IPAddresses))
	for index, authored := range device.Interfaces {
		address := parseFlatPrefix(authored.Address)
		if !address.IsValid() && index < len(device.IPAddresses) {
			address = deviceIPPrefix(device.IPAddresses[index])
		}
		interfaces = append(interfaces, flatDeviceInterfaceState(authored, address))
		if address.IsValid() {
			routes = append(routes, connectedRoute(address, authored.Name))
		}
	}
	for index := len(device.Interfaces); index < len(device.IPAddresses); index++ {
		address := deviceIPPrefix(device.IPAddresses[index])
		if !address.IsValid() {
			continue
		}
		name := fmt.Sprintf("eth%d", index)
		if index == 0 {
			name = "Management"
		}
		interfaces = append(interfaces, flatDeviceInterfaceState(config.Interface{Name: name}, address))
		routes = append(routes, connectedRoute(address, name))
	}
	for _, authored := range device.Routes {
		destination := parseFlatPrefix(authored.Destination)
		if !destination.IsValid() {
			continue
		}
		route := devicestate.Route{Destination: destination, Via: authored.Via}
		if authored.NextHop != "" {
			route.NextHop, _ = netip.ParseAddr(authored.NextHop)
		}
		routes = append(routes, route)
	}

	return devicestate.Network{Interfaces: interfaces, Routes: routes}
}

func flatDeviceInterfaceState(authored config.Interface, address netip.Prefix) devicestate.Interface {
	adminUp := statusUp(authored.AdminStatus)
	carrierUp := statusUp(authored.OperStatus)
	return devicestate.Interface{
		Name: authored.Name, Network: authored.Network, Address: address,
		Description: authored.Description, VLANs: authored.VLANs,
		AdminUp: adminUp, OperUp: adminUp && carrierUp, CarrierUp: carrierUp,
	}
}

func parseFlatPrefix(value string) netip.Prefix {
	prefix, _ := netip.ParsePrefix(strings.TrimSpace(value))
	return prefix
}

func deviceIPPrefix(value []byte) netip.Prefix {
	address, ok := netip.AddrFromSlice(value)
	if !ok {
		return netip.Prefix{}
	}
	return netip.PrefixFrom(address.Unmap(), address.Unmap().BitLen())
}

func connectedRoute(address netip.Prefix, via string) devicestate.Route {
	return devicestate.Route{Destination: address.Masked(), Via: via, Connected: true}
}

func deviceNetworkState(topology *fabric.Topology, device *config.Device) devicestate.Network {
	return devicestate.Network{
		Interfaces: deviceInterfaceStates(topology.Interfaces, device),
		Routes:     deviceRouteStates(topology.Routes, device.Name),
	}
}

func deviceInterfaceStates(compiled []fabric.Interface, device *config.Device) []devicestate.Interface {
	result := make([]devicestate.Interface, 0, len(device.Interfaces))
	for _, iface := range compiled {
		if iface.Device == device.Name {
			result = append(result, deviceInterfaceState(iface, findConfigInterface(device, iface.Name)))
		}
	}

	return result
}

func deviceInterfaceState(compiled fabric.Interface, authored config.Interface) devicestate.Interface {
	adminUp := statusUp(authored.AdminStatus)
	carrierUp := statusUp(authored.OperStatus)
	return devicestate.Interface{
		Name: compiled.Name, Network: compiled.Network, Address: compiled.Address,
		Description: authored.Description, VLANs: authored.VLANs,
		AdminUp: adminUp, OperUp: adminUp && carrierUp, CarrierUp: carrierUp,
	}
}

func findConfigInterface(device *config.Device, name string) config.Interface {
	for _, iface := range device.Interfaces {
		if iface.Name == name {
			return iface
		}
	}

	return config.Interface{}
}

func statusUp(status string) bool {
	return !strings.EqualFold(strings.TrimSpace(status), "down")
}

func deviceRouteStates(compiled []fabric.Route, device string) []devicestate.Route {
	result := make([]devicestate.Route, 0)
	for _, route := range compiled {
		if route.Device == device {
			result = append(result, devicestate.Route{
				Destination: route.Destination, Via: route.Via,
				NextHop: route.NextHop, Connected: route.Connected,
			})
		}
	}

	return result
}
