package devicestate

import (
	"cmp"
	"errors"
	"slices"
)

var (
	// ErrFaultTypeInvalid indicates that a fault is not supported by interface telemetry.
	ErrFaultTypeInvalid = errors.New("invalid interface fault type")
	// ErrFaultValueInvalid indicates that a fault rate is outside 0 through 100.
	ErrFaultValueInvalid = errors.New("interface fault value must be between 0 and 100")
)

// FaultType identifies one SNMP-observable interface condition.
type FaultType string

// Interface fault types injectable via the fault API and interactive CLI.
const (
	FaultFCS         FaultType = "fcs_errors"
	FaultDiscards    FaultType = "packet_discards"
	FaultInterface   FaultType = "interface_errors"
	FaultUtilization FaultType = "high_utilization"
	// FaultLinkDown drops the interface's carrier. Unlike the rates above it
	// is an outcome rather than a counter: the interface reports operationally
	// down, stops forwarding, and disappears from neighbour discovery.
	FaultLinkDown FaultType = "link_down"
	// FaultPoELoss cuts the power the port supplies. Like link_down it is an
	// outcome rather than a counter, and it takes the carrier with it -- a
	// powered device with no power stops linking -- but it is distinguishable
	// from an unplugged cable: POWER-ETHERNET-MIB reports the port faulted and
	// the chassis consumption drops by what that device was drawing.
	FaultPoELoss FaultType = "poe_loss"
)

// FaultDefinition is one supported interface fault and its operator-facing label.
type FaultDefinition struct {
	Type  FaultType
	Label string
}

func interfaceFaultDefinitions() []FaultDefinition {
	return []FaultDefinition{
		{Type: FaultFCS, Label: "FCS Errors"},
		{Type: FaultDiscards, Label: "Packet Discards"},
		{Type: FaultInterface, Label: "Interface Errors"},
		{Type: FaultUtilization, Label: "High Utilization"},
		{Type: FaultLinkDown, Label: "Link Down"},
		{Type: FaultPoELoss, Label: "PoE Loss"},
	}
}

// InterfaceFaultDefinitions returns the supported interface-fault catalog.
func InterfaceFaultDefinitions() []FaultDefinition {
	return interfaceFaultDefinitions()
}

// Label returns the operator-facing fault name.
func (f FaultType) Label() string {
	for _, definition := range interfaceFaultDefinitions() {
		if definition.Type == f {
			return definition.Label
		}
	}
	return ""
}

// ParseFaultLabel returns the supported interface fault type for a label.
func ParseFaultLabel(label string) (FaultType, bool) {
	for _, definition := range interfaceFaultDefinitions() {
		if definition.Label == label {
			return definition.Type, true
		}
	}
	return "", false
}

// InterfaceFault is one active condition on a simulated interface.
type InterfaceFault struct {
	Interface string
	Type      FaultType
	Value     int
}

type interfaceFaultKey struct {
	interfaceName string
	faultType     FaultType
}

// SetInterfaceFault sets one fault rate. A zero value clears only that fault type.
func (s *Store) SetInterfaceFault(interfaceName string, faultType FaultType, value int) error {
	if !validFaultType(faultType) {
		return ErrFaultTypeInvalid
	}
	if value < 0 || value > 100 {
		return ErrFaultValueInvalid
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !interfaceExists(s.running.network.Interfaces, interfaceName) {
		return ErrInterfaceNotFound
	}
	key := interfaceFaultKey{interfaceName: interfaceName, faultType: faultType}
	current, exists := s.faults[key]
	if value == 0 {
		if !exists {
			return nil
		}
		delete(s.faults, key)
		s.version++
		s.recordEvent(EventFaultCleared, interfaceName+":"+string(faultType))
		return nil
	}
	if exists && current.Value == value {
		return nil
	}
	s.faults[key] = InterfaceFault{Interface: interfaceName, Type: faultType, Value: value}
	s.version++
	s.recordEvent(EventFaultUpdated, interfaceName+":"+string(faultType))
	return nil
}

// ClearInterfaceFaults clears every active fault on one authored interface.
func (s *Store) ClearInterfaceFaults(interfaceName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !interfaceExists(s.running.network.Interfaces, interfaceName) {
		return ErrInterfaceNotFound
	}
	changed := false
	for key := range s.addressFaults {
		if key.interfaceName == interfaceName {
			delete(s.addressFaults, key)
			changed = true
		}
	}
	for key := range s.faults {
		if key.interfaceName == interfaceName {
			delete(s.faults, key)
			changed = true
		}
	}
	if changed {
		s.version++
		s.recordEvent(EventFaultCleared, interfaceName)
	}
	return nil
}

// ClearAllFaults clears every active interface fault.
func (s *Store) ClearAllFaults() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.faults) == 0 && len(s.addressFaults) == 0 {
		return
	}
	clear(s.faults)
	clear(s.addressFaults)
	s.version++
	s.recordEvent(EventFaultCleared, "*")
}

func validFaultType(faultType FaultType) bool {
	return slices.ContainsFunc(interfaceFaultDefinitions(), func(definition FaultDefinition) bool {
		return definition.Type == faultType
	})
}

func interfaceExists(interfaces []Interface, name string) bool {
	return slices.ContainsFunc(interfaces, func(iface Interface) bool { return iface.Name == name })
}

func sortedInterfaceFaults(faults map[interfaceFaultKey]InterfaceFault) []InterfaceFault {
	result := make([]InterfaceFault, 0, len(faults))
	for _, fault := range faults {
		result = append(result, fault)
	}
	slices.SortFunc(result, func(left, right InterfaceFault) int {
		if order := cmp.Compare(left.Interface, right.Interface); order != 0 {
			return order
		}
		return cmp.Compare(left.Type, right.Type)
	})
	return result
}

// isCarrierFault reports whether a fault takes the carrier away rather than
// moving a counter. A cut cable and a cut power budget both end with the port
// not linking; what separates them is POWER-ETHERNET-MIB, not the interface.
func isCarrierFault(faultType FaultType) bool {
	switch faultType {
	case FaultLinkDown, FaultPoELoss:
		return true
	case FaultFCS, FaultDiscards, FaultInterface, FaultUtilization:
		return false
	}

	return false
}

// applyCarrierFaults projects active carrier-taking faults onto an interface
// list. The carrier is what such a fault takes away; operational state follows
// from it, the same way it does when an operator shuts a port.
func applyCarrierFaults(interfaces []Interface, faults map[interfaceFaultKey]InterfaceFault) {
	for key, fault := range faults {
		if !isCarrierFault(key.faultType) || fault.Value <= 0 {
			continue
		}
		for index := range interfaces {
			if interfaces[index].Name == key.interfaceName {
				interfaces[index].CarrierUp = false
				interfaces[index].OperUp = false
			}
		}
	}
}
