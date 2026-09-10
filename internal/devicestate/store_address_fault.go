package devicestate

import (
	"errors"
	"net/netip"
)

// FaultDuplicateDHCPOffer advertises an explicitly conflicting IPv4 address.
const FaultDuplicateDHCPOffer DeviceFaultType = "duplicate_dhcp_offer"

// ErrFaultAddressInvalid indicates an unusable IPv4 address payload.
var ErrFaultAddressInvalid = errors.New("fault address must be a unicast IPv4 address")

// SetDeviceAddressFault arms an address-bearing fault without numeric coercion.
func (s *Store) SetDeviceAddressFault(kind DeviceFaultType, address netip.Addr) error {
	if kind != FaultDuplicateDHCPOffer {
		return ErrDeviceFaultTypeInvalid
	}
	if !validFaultAddress(address) {
		return ErrFaultAddressInvalid
	}
	s.setDeviceFault(DeviceFault{Type: kind, Address: address})
	return nil
}

// DeviceFaultAddress returns the active address payload, or an invalid address.
func (s *Store) DeviceFaultAddress(kind DeviceFaultType) netip.Addr {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.deviceFaults[kind].Address
}

// ClearDeviceFault clears a single known fault without a sentinel payload.
func (s *Store) ClearDeviceFault(kind DeviceFaultType) error {
	if _, known := deviceFaultDefinition(kind); !known && kind != FaultDuplicateDHCPOffer {
		return ErrDeviceFaultTypeInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.deviceFaults[kind]; exists {
		delete(s.deviceFaults, kind)
		s.version++
		s.recordEvent(EventDeviceFaultCleared, string(kind))
	}
	return nil
}

func (s *Store) setDeviceFault(fault DeviceFault) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if current, exists := s.deviceFaults[fault.Type]; exists && current == fault {
		return
	}
	s.deviceFaults[fault.Type] = fault
	s.version++
	s.recordEvent(EventDeviceFaultUpdated, string(fault.Type))
}

func validFaultAddress(address netip.Addr) bool {
	return address.Is4() && !address.IsUnspecified() && !address.IsMulticast() &&
		address != netip.AddrFrom4([4]byte{255, 255, 255, 255})
}

func validDeviceFault(fault DeviceFault) bool {
	if fault.Type == FaultDuplicateDHCPOffer {
		return fault.Value == 0 && validFaultAddress(fault.Address)
	}
	definition, known := deviceFaultDefinition(fault.Type)
	return known && fault.Address == (netip.Addr{}) && fault.Value > 0 && fault.Value <= definition.MaxValue
}
