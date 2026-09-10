package devicestate

import (
	"cmp"
	"net/netip"
	"slices"
)

// InterfaceAddressFaultType identifies an explicitly addressed interface outcome.
type InterfaceAddressFaultType string

// FaultDuplicateIP adds a conflict responder without replacing canonical ownership.
const FaultDuplicateIP InterfaceAddressFaultType = "duplicate_ip"

// InterfaceAddressFault keeps addressed outcomes separate from numeric interface rates.
type InterfaceAddressFault struct {
	Interface string
	Type      InterfaceAddressFaultType
	Address   netip.Addr
}

type interfaceAddressFaultKey struct {
	interfaceName string
	faultType     InterfaceAddressFaultType
}

// SetInterfaceAddressFault stores one explicit interface/address outcome.
func (s *Store) SetInterfaceAddressFault(fault InterfaceAddressFault) error {
	if fault.Type != FaultDuplicateIP {
		return ErrFaultTypeInvalid
	}
	if !ValidFaultAddress(fault.Address) {
		return ErrFaultAddressInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !interfaceExists(s.running.network.Interfaces, fault.Interface) {
		return ErrInterfaceNotFound
	}
	key := interfaceAddressFaultKey{fault.Interface, fault.Type}
	if s.addressFaults[key] == fault {
		return nil
	}
	s.addressFaults[key] = fault
	s.version++
	s.recordEvent(EventFaultUpdated, fault.Interface+":"+string(fault.Type))
	return nil
}

// ClearInterfaceAddressFault clears one addressed outcome without changing other faults.
func (s *Store) ClearInterfaceAddressFault(iface string, kind InterfaceAddressFaultType) error {
	if kind != FaultDuplicateIP {
		return ErrFaultTypeInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !interfaceExists(s.running.network.Interfaces, iface) {
		return ErrInterfaceNotFound
	}
	key := interfaceAddressFaultKey{iface, kind}
	if _, exists := s.addressFaults[key]; exists {
		delete(s.addressFaults, key)
		s.version++
		s.recordEvent(EventFaultCleared, iface+":"+string(kind))
	}
	return nil
}

func sortedAddressFaults(faults map[interfaceAddressFaultKey]InterfaceAddressFault) []InterfaceAddressFault {
	result := make([]InterfaceAddressFault, 0, len(faults))
	for _, fault := range faults {
		result = append(result, fault)
	}
	slices.SortFunc(result, func(a, b InterfaceAddressFault) int {
		return cmp.Or(cmp.Compare(a.Interface, b.Interface), cmp.Compare(a.Type, b.Type))
	})
	return result
}

func importAddressFaults(faults []InterfaceAddressFault) map[interfaceAddressFaultKey]InterfaceAddressFault {
	result := make(map[interfaceAddressFaultKey]InterfaceAddressFault, len(faults))
	for _, fault := range faults {
		result[interfaceAddressFaultKey{fault.Interface, fault.Type}] = fault
	}
	return result
}

func validStateAddressFaults(faults []InterfaceAddressFault, interfaces []Interface) bool {
	seen := make(map[interfaceAddressFaultKey]bool, len(faults))
	for _, fault := range faults {
		key := interfaceAddressFaultKey{fault.Interface, fault.Type}
		if seen[key] || fault.Type != FaultDuplicateIP || !ValidFaultAddress(fault.Address) ||
			!interfaceExists(interfaces, fault.Interface) {
			return false
		}
		seen[key] = true
	}
	return true
}
