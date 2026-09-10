package devicestate

import (
	"cmp"
	"errors"
	"net/netip"
	"slices"
)

// InterfacePrefixFaultType identifies a host-local prefix outcome.
type InterfacePrefixFaultType string

// FaultBadMask overrides a host's local-network mask without moving its address.
const FaultBadMask InterfacePrefixFaultType = "bad_mask"

// ErrFaultPrefixInvalid indicates an invalid IPv4 mask length or target.
var ErrFaultPrefixInvalid = errors.New("prefix fault requires an IPv4 interface and mask length 0 through 32")

// InterfacePrefixFault retains mask bits independently of the canonical address.
type InterfacePrefixFault struct {
	Interface  string
	Type       InterfacePrefixFaultType
	PrefixBits int
}

type interfacePrefixFaultKey struct {
	interfaceName string
	faultType     InterfacePrefixFaultType
}

// HasInterfacePrefixFaults avoids cloning network state on healthy packet paths.
func (s *Store) HasInterfacePrefixFaults() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.prefixFaults) != 0
}

// SetInterfacePrefixFault arms a mask outcome; zero bits does not clear it.
func (s *Store) SetInterfacePrefixFault(fault InterfacePrefixFault) error {
	if fault.Type != FaultBadMask {
		return ErrFaultTypeInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index := slices.IndexFunc(
		s.running.network.Interfaces,
		func(iface Interface) bool { return iface.Name == fault.Interface },
	)
	if index < 0 {
		return ErrInterfaceNotFound
	}
	address := s.running.network.Interfaces[index].Address
	if !address.IsValid() || !ValidFaultAddress(address.Addr()) || fault.PrefixBits < 0 || fault.PrefixBits > 32 {
		return ErrFaultPrefixInvalid
	}
	key := interfacePrefixFaultKey{fault.Interface, fault.Type}
	if current, exists := s.prefixFaults[key]; exists && current == fault {
		return nil
	}
	s.prefixFaults[key] = fault
	s.version++
	s.recordEvent(EventFaultUpdated, fault.Interface+":"+string(fault.Type))
	return nil
}

// ClearInterfacePrefixFault clears only the selected mask outcome.
func (s *Store) ClearInterfacePrefixFault(iface string, kind InterfacePrefixFaultType) error {
	if kind != FaultBadMask {
		return ErrFaultTypeInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !interfaceExists(s.running.network.Interfaces, iface) {
		return ErrInterfaceNotFound
	}
	key := interfacePrefixFaultKey{iface, kind}
	if _, exists := s.prefixFaults[key]; exists {
		delete(s.prefixFaults, key)
		s.version++
		s.recordEvent(EventFaultCleared, iface+":"+string(kind))
	}
	return nil
}

// EffectiveHostNetwork projects mask faults for host consumers, never ownership or fabric routing.
func (s Snapshot) EffectiveHostNetwork() Network {
	network := cloneNetwork(s.Network)
	for _, fault := range s.PrefixFaults {
		for index, iface := range network.Interfaces {
			if iface.Name != fault.Interface || !iface.Address.IsValid() || !iface.Address.Addr().Is4() {
				continue
			}
			iface.Address = netip.PrefixFrom(iface.Address.Addr(), fault.PrefixBits)
			network.Interfaces[index] = iface
			network.Routes = reconcileConnectedRoute(network.Routes, iface)
		}
	}
	return network
}

func sortedPrefixFaults(faults map[interfacePrefixFaultKey]InterfacePrefixFault) []InterfacePrefixFault {
	result := make([]InterfacePrefixFault, 0, len(faults))
	for _, fault := range faults {
		result = append(result, fault)
	}
	slices.SortFunc(result, func(a, b InterfacePrefixFault) int {
		if order := cmp.Compare(a.Interface, b.Interface); order != 0 {
			return order
		}
		return cmp.Compare(a.Type, b.Type)
	})
	return result
}

func importPrefixFaults(faults []InterfacePrefixFault) map[interfacePrefixFaultKey]InterfacePrefixFault {
	result := make(map[interfacePrefixFaultKey]InterfacePrefixFault, len(faults))
	for _, fault := range faults {
		result[interfacePrefixFaultKey{fault.Interface, fault.Type}] = fault
	}
	return result
}

func validStatePrefixFaults(faults []InterfacePrefixFault, interfaces []Interface) bool {
	seen := make(map[interfacePrefixFaultKey]bool, len(faults))
	for _, fault := range faults {
		key := interfacePrefixFaultKey{fault.Interface, fault.Type}
		if seen[key] || fault.Type != FaultBadMask || fault.PrefixBits < 0 || fault.PrefixBits > 32 ||
			!interfaceExists(interfaces, fault.Interface) {
			return false
		}
		seen[key] = true
	}
	return true
}
