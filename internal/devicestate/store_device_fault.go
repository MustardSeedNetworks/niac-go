package devicestate

import (
	"cmp"
	"errors"
	"maps"
	"slices"
)

// ErrDeviceFaultTypeInvalid indicates that a fault is not a device-service outcome.
var ErrDeviceFaultTypeInvalid = errors.New("invalid device fault type")

// DeviceFaultType identifies one device-service outcome. It is a separate
// type from FaultType rather than more constants in it: a service outage has
// no interface to be keyed by, so the two axes must not be assignable to each
// other's setters in the first place.
type DeviceFaultType string

// Device fault types. These change what a device's protocol handlers answer,
// not what its interface telemetry reports. A non-zero value arms the fault
// the way link-down does.
const (
	FaultDHCPNoOffer DeviceFaultType = "dhcp_no_offer"
	FaultDNSNXDomain DeviceFaultType = "dns_nxdomain"
	FaultDNSTimeout  DeviceFaultType = "dns_timeout"
)

// DeviceFault is one active service outcome on a simulated device.
type DeviceFault struct {
	Type  DeviceFaultType
	Value int
}

// DeviceFaultDefinition is one supported device fault and its label.
type DeviceFaultDefinition struct {
	Type  DeviceFaultType
	Label string
}

func deviceFaultDefinitions() []DeviceFaultDefinition {
	return []DeviceFaultDefinition{
		{Type: FaultDHCPNoOffer, Label: "DHCP No Offer"},
		{Type: FaultDNSNXDomain, Label: "DNS NXDOMAIN"},
		{Type: FaultDNSTimeout, Label: "DNS Timeout"},
	}
}

// DeviceFaultDefinitions returns the supported device-fault catalog.
func DeviceFaultDefinitions() []DeviceFaultDefinition {
	return deviceFaultDefinitions()
}

// Label returns the operator-facing device-fault name.
func (f DeviceFaultType) Label() string {
	for _, definition := range deviceFaultDefinitions() {
		if definition.Type == f {
			return definition.Label
		}
	}
	return ""
}

// ParseDeviceFaultLabel returns the device fault type for an operator-facing label.
func ParseDeviceFaultLabel(label string) (DeviceFaultType, bool) {
	for _, definition := range deviceFaultDefinitions() {
		if definition.Label == label {
			return definition.Type, true
		}
	}
	return "", false
}

// SetDeviceFault arms one device-service fault. A zero value clears only that
// fault type.
func (s *Store) SetDeviceFault(faultType DeviceFaultType, value int) error {
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
func (s *Store) DeviceFaultActive(faultType DeviceFaultType) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fault, active := s.deviceFaults[faultType]
	return active && fault.Value > 0
}

func validDeviceFaultType(faultType DeviceFaultType) bool {
	return slices.ContainsFunc(deviceFaultDefinitions(), func(definition DeviceFaultDefinition) bool {
		return definition.Type == faultType
	})
}

func cloneDeviceFaults(faults map[DeviceFaultType]DeviceFault) map[DeviceFaultType]DeviceFault {
	cloned := make(map[DeviceFaultType]DeviceFault, len(faults))
	maps.Copy(cloned, faults)
	return cloned
}

func sortedDeviceFaults(faults map[DeviceFaultType]DeviceFault) []DeviceFault {
	result := make([]DeviceFault, 0, len(faults))
	for _, fault := range faults {
		result = append(result, fault)
	}
	slices.SortFunc(result, func(left, right DeviceFault) int {
		return cmp.Compare(left.Type, right.Type)
	})
	return result
}
