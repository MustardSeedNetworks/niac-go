package protocols

import (
	"net/netip"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// Build transient eligibility from one snapshot per device, rather than scanning
// every peer again for each port in the frequently polled fault catalog.
func (s *Stack) interfaceConflictTargets(
	states map[*config.Device]devicestate.Snapshot,
) map[*config.Device]map[string]bool {
	attached, prefix := s.conflictAttachment()
	segments := s.conflictSegments(states)
	owners := make(map[int]int)
	ownerDevices := make(map[*config.Device]bool)
	result := make(map[*config.Device]map[string]bool)
	for device, snapshot := range states {
		segment := segments[device]
		result[device] = make(map[string]bool)
		for _, iface := range snapshot.Network.Interfaces {
			if !iface.AdminUp || !iface.OperUp || (s.fabric != nil && !attached[device][iface.Name]) {
				continue
			}
			result[device][iface.Name] = true
			addressPrefix := iface.Address
			if s.fabric != nil {
				addressPrefix = prefix
			}
			ownerDevices[device] = ownerDevices[device] || config.ValidIPv4Host(addressPrefix, iface.Address.Addr())
		}
		if ownerDevices[device] {
			owners[segment]++
		}
	}
	for device := range result {
		peers := owners[segments[device]]
		if peers == 0 || (peers == 1 && ownerDevices[device]) {
			delete(result, device)
		}
	}
	return result
}

func (s *Stack) conflictAttachment() (map[*config.Device]map[string]bool, netip.Prefix) {
	attached := make(map[*config.Device]map[string]bool)
	var prefix netip.Prefix
	if s.fabric != nil {
		for _, endpoint := range s.fabric.interfacesByAddr {
			if endpoint.network == s.fabric.attachmentNetwork {
				if attached[endpoint.device] == nil {
					attached[endpoint.device] = make(map[string]bool)
				}
				attached[endpoint.device][endpoint.interfaceName] = true
			}
		}
		for _, network := range s.fabric.topology.Networks {
			if network.Name == s.fabric.attachmentNetwork {
				prefix = network.Prefix
			}
		}
	}
	return attached, prefix
}

func (s *Stack) conflictSegments(states map[*config.Device]devicestate.Snapshot) map[*config.Device]int {
	segments := make(map[*config.Device]int, len(states))
	if s.fabric != nil {
		return segments
	}
	for device := range states {
		segments[device] = device.VLAN
	}
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	if s.config != nil {
		for _, segment := range s.config.Segments {
			for index := range segment.Devices {
				segments[&segment.Devices[index]] = segment.Tag
			}
		}
	}
	return segments
}
