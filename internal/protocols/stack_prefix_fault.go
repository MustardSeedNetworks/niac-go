package protocols

import (
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func (s *Stack) validateMaskBinding(fault config.BehaviorFault) error {
	device, store, err := s.interfaceFaultTarget(fault.Device)
	if err != nil {
		return err
	}
	if !stateHasInterface(store.Snapshot(), fault.Interface) {
		return devicestate.ErrInterfaceNotFound
	}
	if s.fabric == nil {
		return nil
	}
	for _, endpoint := range s.fabric.interfacesByAddr {
		if endpoint.device == device && endpoint.interfaceName == fault.Interface &&
			endpoint.network == s.fabric.attachmentNetwork {
			return nil
		}
	}
	return devicestate.ErrInterfaceNotFound
}

// ActiveInterfacePrefixFaults returns mask outcomes keyed by device name.
func (s *Stack) ActiveInterfacePrefixFaults() map[string][]devicestate.InterfacePrefixFault {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	result := make(map[string][]devicestate.InterfacePrefixFault)
	for device, store := range s.deviceStates {
		if faults := store.Snapshot().PrefixFaults; len(faults) > 0 {
			result[device.Name] = faults
		}
	}
	return result
}

// SetInterfacePrefixFault changes host routing without changing canonical ownership.
func (s *Stack) SetInterfacePrefixFault(
	target string,
	fault devicestate.InterfacePrefixFault,
) error {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	device, store, err := s.interfaceFaultTarget(target)
	if err != nil {
		return err
	}
	for _, iface := range store.Snapshot().Network.Interfaces {
		if iface.Name == fault.Interface && s.maskInterfaceEligible(device, iface) {
			return store.SetInterfacePrefixFault(fault)
		}
	}
	return devicestate.ErrInterfaceNotFound
}

func (s *Stack) maskInterfaceEligible(device *config.Device, iface devicestate.Interface) bool {
	if !hostMaskRole(device) || !iface.AdminUp || !iface.OperUp ||
		!devicestate.ValidFaultAddress(iface.Address.Addr()) {
		return false
	}
	if s.fabric == nil {
		return true
	}
	for _, endpoint := range s.fabric.interfacesByAddr {
		if endpoint.device == device && endpoint.interfaceName == iface.Name &&
			endpoint.network == s.fabric.attachmentNetwork {
			return true
		}
	}
	return false
}

// ClearInterfacePrefixFault removes a mask even when its interface is now unavailable.
func (s *Stack) ClearInterfacePrefixFault(
	target, iface string,
	kind devicestate.InterfacePrefixFaultType,
) error {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	_, store, err := s.interfaceFaultTarget(target)
	if err != nil {
		return err
	}
	return store.ClearInterfacePrefixFault(iface, kind)
}
