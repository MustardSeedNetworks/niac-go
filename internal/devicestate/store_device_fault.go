package devicestate

import (
	"cmp"
	"errors"
	"maps"
	"slices"
)

// ErrDeviceFaultTypeInvalid indicates that a fault is not a device-service outcome.
var ErrDeviceFaultTypeInvalid = errors.New("invalid device fault type")

// Device fault types. These are service outcomes rather than interface
// counters: they change what a device's protocol handlers answer, not what
// its interface telemetry reports, so they are keyed by device and not by
// interface. A non-zero value arms the fault the way link-down does.
const (
	FaultDHCPNoOffer FaultType = "dhcp_no_offer"
	FaultDNSNXDomain FaultType = "dns_nxdomain"
	FaultDNSTimeout  FaultType = "dns_timeout"
)

// DeviceFault is one active service outcome on a simulated device.
type DeviceFault struct {
	Type  FaultType
	Value int
}

func deviceFaultDefinitions() []FaultDefinition {
	return []FaultDefinition{
		{Type: FaultDHCPNoOffer, Label: "DHCP No Offer"},
		{Type: FaultDNSNXDomain, Label: "DNS NXDOMAIN"},
		{Type: FaultDNSTimeout, Label: "DNS Timeout"},
	}
}

// DeviceFaultDefinitions returns the supported device-fault catalog.
func DeviceFaultDefinitions() []FaultDefinition {
	return deviceFaultDefinitions()
}

// SetDeviceFault arms one device-service fault. A zero value clears only that
// fault type.
func (s *Store) SetDeviceFault(faultType FaultType, value int) error {
	if !validDeviceFaultType(faultType) {
		return ErrDeviceFaultTypeInvalid
	}
	if value < 0 || value > 100 {
		return ErrFaultValueInvalid
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	current, exists := s.deviceFaults[faultType]
	if value == 0 {
		if !exists {
			return nil
		}
		delete(s.deviceFaults, faultType)
		s.version++
		s.recordEvent(EventDeviceFaultCleared, string(faultType))
		return nil
	}
	if exists && current.Value == value {
		return nil
	}
	s.deviceFaults[faultType] = DeviceFault{Type: faultType, Value: value}
	s.version++
	s.recordEvent(EventDeviceFaultUpdated, string(faultType))
	return nil
}

// ClearDeviceFaults clears every active device-service fault.
func (s *Store) ClearDeviceFaults() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.deviceFaults) == 0 {
		return
	}
	clear(s.deviceFaults)
	s.version++
	s.recordEvent(EventDeviceFaultCleared, "*")
}

// DeviceFaultActive reports whether one device fault is armed. Protocol
// handlers ask this per request, so it reads the map directly rather than
// cloning a snapshot.
func (s *Store) DeviceFaultActive(faultType FaultType) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fault, active := s.deviceFaults[faultType]
	return active && fault.Value > 0
}

func validDeviceFaultType(faultType FaultType) bool {
	return slices.ContainsFunc(deviceFaultDefinitions(), func(definition FaultDefinition) bool {
		return definition.Type == faultType
	})
}

func cloneDeviceFaults(faults map[FaultType]DeviceFault) map[FaultType]DeviceFault {
	cloned := make(map[FaultType]DeviceFault, len(faults))
	maps.Copy(cloned, faults)
	return cloned
}

func sortedDeviceFaults(faults map[FaultType]DeviceFault) []DeviceFault {
	result := make([]DeviceFault, 0, len(faults))
	for _, fault := range faults {
		result = append(result, fault)
	}
	slices.SortFunc(result, func(left, right DeviceFault) int {
		return cmp.Compare(left.Type, right.Type)
	})
	return result
}
