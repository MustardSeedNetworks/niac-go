package protocols

import (
	"net"
	"net/netip"
	"slices"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// ActiveInterfaceAddressFaults returns addressed outcomes keyed by unique device name.
func (s *Stack) ActiveInterfaceAddressFaults() map[string][]devicestate.InterfaceAddressFault {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	result := make(map[string][]devicestate.InterfaceAddressFault)
	for device, store := range s.deviceStates {
		if faults := store.Snapshot().AddressFaults; len(faults) != 0 {
			result[device.Name] = faults
		}
	}
	return result
}

// SetInterfaceAddressFault arms an additional responder on one explicit interface.
func (s *Stack) SetInterfaceAddressFault(target string, fault devicestate.InterfaceAddressFault) error {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	device, store, err := s.interfaceFaultTarget(target)
	if err != nil {
		return err
	}
	if fault.Type != devicestate.FaultDuplicateIP {
		return devicestate.ErrFaultTypeInvalid
	}
	if !stateHasInterface(store.Snapshot(), fault.Interface) {
		return devicestate.ErrInterfaceNotFound
	}
	if !s.conflictInterfaceAvailable(device, fault.Interface) || !s.activeConflictPeer(device, fault.Address) {
		return ErrFaultConflictAbsent
	}
	return store.SetInterfaceAddressFault(fault)
}

// ClearInterfaceAddressFault removes one explicitly addressed interface outcome.
func (s *Stack) ClearInterfaceAddressFault(target, iface string, kind devicestate.InterfaceAddressFaultType) error {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	_, store, err := s.interfaceFaultTarget(target)
	if err != nil {
		return err
	}
	return store.ClearInterfaceAddressFault(iface, kind)
}

func (s *Stack) conflictInterfaceAvailable(device *config.Device, name string) bool {
	for _, iface := range s.deviceStates[device].Snapshot().Network.Interfaces {
		if iface.Name != name || !iface.AdminUp || !iface.OperUp {
			continue
		}
		if s.fabric == nil {
			return true
		}
		for _, endpoint := range s.fabric.interfacesByAddr {
			if endpoint.device == device && endpoint.interfaceName == name &&
				endpoint.network == s.fabric.attachmentNetwork {
				return true
			}
		}
	}
	return false
}

func (s *Stack) appendConflictResponders(devices []*config.Device, target net.IP, vlan int) []*config.Device {
	address, ok := netip.AddrFromSlice(target)
	if !ok || len(devices) == 0 {
		return devices
	}
	address = address.Unmap()
	for device, store := range s.deviceStates {
		if s.discoveryVLAN(device) != vlan || slices.Contains(devices, device) {
			continue
		}
		for _, fault := range store.Snapshot().AddressFaults {
			if fault.Type == devicestate.FaultDuplicateIP && fault.Address == address &&
				s.conflictInterfaceAvailable(device, fault.Interface) && s.activeConflictPeer(device, address) {
				devices = append(devices, device)
				break
			}
		}
	}
	return devices
}
