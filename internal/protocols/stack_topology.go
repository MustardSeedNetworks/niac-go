package protocols

import (
	"maps"
	"slices"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/topology"
)

// RuntimeTopology projects the current device state into the UI topology model.
func (s *Stack) RuntimeTopology() topology.Graph {
	s.reloadMu.Lock()
	defer s.reloadMu.Unlock()

	cfg := s.currentConfig()
	if cfg == nil {
		return topology.Graph{}
	}
	projected := *cfg
	projected.Devices = append([]config.Device(nil), cfg.Devices...)
	for index := range projected.Devices {
		source := &cfg.Devices[index]
		device := &projected.Devices[index]
		device.Interfaces = append([]config.Interface(nil), source.Interfaces...)
		state := s.deviceStates[source]
		if state == nil {
			continue
		}
		snapshot := state.Snapshot()
		faults := make(map[string][]devicestate.InterfaceFault)
		for _, fault := range snapshot.Faults {
			faults[fault.Interface] = append(faults[fault.Interface], fault)
		}
		for interfaceIndex := range device.Interfaces {
			for _, current := range snapshot.Network.Interfaces {
				if current.Name != device.Interfaces[interfaceIndex].Name {
					continue
				}
				if current.Address.IsValid() {
					device.Interfaces[interfaceIndex].Address = current.Address.String()
				} else {
					device.Interfaces[interfaceIndex].Address = ""
				}
				device.Interfaces[interfaceIndex].AdminStatus = interfaceStatus(current.AdminUp)
				device.Interfaces[interfaceIndex].OperStatus = interfaceStatus(current.OperUp)
				projectInterfaceFaults(&device.Interfaces[interfaceIndex], faults[current.Name])
				break
			}
		}
	}
	return topology.Build(&projected)
}

func projectInterfaceFaults(iface *config.Interface, faults []devicestate.InterfaceFault) {
	for _, fault := range faults {
		if fault.Value <= 0 {
			continue
		}
		if iface.OperStatus != "down" {
			iface.OperStatus = "degraded"
		}
		if fault.Type == devicestate.FaultUtilization {
			iface.InUtilization = max(iface.InUtilization, float64(fault.Value))
			iface.OutUtilization = max(iface.OutUtilization, float64(fault.Value))
		}
	}
}

// RuntimeFabricTopology projects authoritative interface and route state into
// the active compiled fabric contract.
func (s *Stack) RuntimeFabricTopology() (fabric.Topology, bool) {
	s.reloadMu.Lock()
	defer s.reloadMu.Unlock()

	cfg := s.currentConfig()
	if cfg == nil || s.fabric == nil {
		return fabric.Topology{}, false
	}
	result := s.fabric.topology
	result.Interfaces = append([]fabric.Interface(nil), result.Interfaces...)
	result.Routes = make([]fabric.Route, 0, len(result.Routes))
	interfaces := make(map[string]map[string]int, len(result.Interfaces))
	for index, current := range result.Interfaces {
		if interfaces[current.Device] == nil {
			interfaces[current.Device] = make(map[string]int)
		}
		interfaces[current.Device][current.Name] = index
	}
	result.Routes = result.Routes[:0]
	devices := make(map[string]*config.Device, len(s.deviceStates))
	for device := range s.deviceStates {
		devices[device.Name] = device
	}
	for _, name := range slices.Sorted(maps.Keys(devices)) {
		device := devices[name]
		state := s.deviceStates[device]
		snapshot := state.Snapshot()
		for _, current := range snapshot.Network.Interfaces {
			index, found := interfaces[device.Name][current.Name]
			if found {
				result.Interfaces[index].Address = current.Address
			}
		}
		for _, current := range snapshot.Network.Routes {
			result.Routes = append(result.Routes, fabric.Route{
				Device: device.Name, Destination: current.Destination,
				Via: current.Via, NextHop: current.NextHop, Connected: current.Connected,
			})
		}
	}
	return result, true
}

func interfaceStatus(up bool) string {
	if up {
		return "up"
	}
	return "down"
}
