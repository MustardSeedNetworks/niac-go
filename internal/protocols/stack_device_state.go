package protocols

import (
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
	if topology == nil {
		return
	}
	for device, state := range s.deviceStates {
		state.ReplaceNetwork(deviceNetworkState(topology, device))
	}
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
