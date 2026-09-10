package devicestate

import "math"

// ValidateState checks a durable record against the authored interface identities
// without changing the store. Runtime addresses and hostnames may differ.
func (s *Store) ValidateState(state State) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return validateState(state, s.authored.network.Interfaces)
}

func validateState(state State, authored []Interface) error {
	if !validStateHistory(state) || !validStateInterfaces(state.Startup.Network.Interfaces, authored) {
		return ErrStateInvalid
	}
	current := Checkpoint{
		Configuration:   state.Running,
		InterfaceFaults: state.InterfaceFaults,
		DeviceFaults:    state.DeviceFaults,
	}
	if !validStateCheckpoint(current, authored) {
		return ErrStateInvalid
	}
	seen := make(map[string]bool, len(state.Checkpoints))
	for _, point := range state.Checkpoints {
		if seen[point.Name] || !validStateCheckpoint(point, authored) {
			return ErrStateInvalid
		}
		seen[point.Name] = true
	}
	return nil
}

func validStateHistory(state State) bool {
	if state.Version == 0 || state.Version == math.MaxUint64 || len(state.Events) > maxEventHistory {
		return false
	}
	var previous uint64
	for _, event := range state.Events {
		if event.Version == 0 || event.Version < previous || event.Version > state.Version {
			return false
		}
		previous = event.Version
	}
	return true
}

func validStateInterfaces(interfaces, authored []Interface) bool {
	if len(interfaces) != len(authored) {
		return false
	}
	seen := make(map[string]bool, len(interfaces))
	for index, iface := range interfaces {
		if seen[iface.Name] || iface.Name != authored[index].Name {
			return false
		}
		seen[iface.Name] = true
	}
	return true
}

func validStateCheckpoint(point Checkpoint, authored []Interface) bool {
	return validStateInterfaces(point.Configuration.Network.Interfaces, authored) &&
		validStateInterfaceFaults(point.InterfaceFaults, authored) &&
		validStateDeviceFaults(point.DeviceFaults)
}

func validStateInterfaceFaults(faults []InterfaceFault, interfaces []Interface) bool {
	seen := make(map[interfaceFaultKey]bool, len(faults))
	for _, fault := range faults {
		key := interfaceFaultKey{interfaceName: fault.Interface, faultType: fault.Type}
		if seen[key] || !validFaultType(fault.Type) || fault.Value <= 0 || fault.Value > faultRateMax ||
			!interfaceExists(interfaces, fault.Interface) {
			return false
		}
		seen[key] = true
	}
	return true
}

func validStateDeviceFaults(faults []DeviceFault) bool {
	seen := make(map[DeviceFaultType]bool, len(faults))
	for _, fault := range faults {
		definition, known := deviceFaultDefinition(fault.Type)
		if seen[fault.Type] || !known || fault.Value <= 0 || fault.Value > definition.MaxValue {
			return false
		}
		seen[fault.Type] = true
	}
	return true
}
