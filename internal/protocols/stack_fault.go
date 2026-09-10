package protocols

import (
	"errors"
	"net/netip"
	"slices"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

var (
	// ErrFaultDeviceNotFound means the fault target names no device in the running simulation.
	ErrFaultDeviceNotFound = errors.New("fault target device not found")
	// ErrFaultDeviceAmbiguous means the fault target address resolves to more than one device.
	ErrFaultDeviceAmbiguous = errors.New("fault target address matches multiple devices")
	// ErrFaultUnobservable means the target device exposes no SNMP interface
	// counters a fault could visibly perturb.
	ErrFaultUnobservable = errors.New("fault target has no observable SNMP interface counters")
	// ErrFaultNoPSEPort means a PoE fault names a port that supplies no power,
	// so cutting it would be indistinguishable from unplugging the cable.
	ErrFaultNoPSEPort = errors.New("fault target port supplies no PoE")
)

// InterfaceFaultTarget describes one device's current fault-injection surface.
type InterfaceFaultTarget struct {
	Device     string
	Address    string
	Interfaces []string
	ErrorTypes map[string][]string
}

// SetInterfaceFault applies one fault to the stack-owned device state.
func (s *Stack) SetInterfaceFault(
	deviceIP, interfaceName string,
	faultType devicestate.FaultType,
	value int,
) error {
	return s.setInterfaceFaultAt(deviceIP, interfaceName, faultType, value, time.Now())
}

func (s *Stack) setInterfaceFaultAt(
	deviceIP, interfaceName string,
	faultType devicestate.FaultType,
	value int,
	now time.Time,
) error {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	device, store, err := s.interfaceFaultTarget(deviceIP)
	if err != nil {
		return err
	}
	if !stateHasInterface(store.Snapshot(), interfaceName) {
		return devicestate.ErrInterfaceNotFound
	}
	if !s.snmpAgents[device].interfaceFaultObservable(interfaceName) {
		return ErrFaultUnobservable
	}
	if faultType == devicestate.FaultPoELoss &&
		!s.snmpAgents[device].poeFaultObservable(interfaceName) {
		return ErrFaultNoPSEPort
	}
	s.advanceDeviceFaultTelemetry(device, now)
	return store.SetInterfaceFault(interfaceName, faultType, value)
}

// ClearInterfaceFaults clears every active fault on one simulated interface.
func (s *Stack) ClearInterfaceFaults(deviceIP, interfaceName string) error {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	device, store, err := s.interfaceFaultTarget(deviceIP)
	if err != nil {
		return err
	}
	s.advanceDeviceFaultTelemetry(device, time.Now())
	return store.ClearInterfaceFaults(interfaceName)
}

// ClearAllInterfaceFaults clears every active fault in the stack.
func (s *Stack) ClearAllInterfaceFaults() {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	now := time.Now()
	for device, store := range s.deviceStates {
		s.advanceDeviceFaultTelemetry(device, now)
		store.ClearAllFaults()
	}
}

// ActiveInterfaceFaults returns a JSON-ready snapshot keyed by unique device name.
func (s *Stack) ActiveInterfaceFaults() map[string]map[string]map[devicestate.FaultType]int {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	result := make(map[string]map[string]map[devicestate.FaultType]int)
	for device, store := range s.deviceStates {
		addActiveInterfaceFaults(result, device.Name, store.Snapshot().Faults)
	}
	return result
}

// InterfaceFaultTargets returns current device identities, addresses, and interfaces.
func (s *Stack) InterfaceFaultTargets() []InterfaceFaultTarget {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	states := make(map[*config.Device]devicestate.Snapshot, len(s.deviceStates))
	for device, store := range s.deviceStates {
		states[device] = store.Snapshot()
	}
	conflicts := s.interfaceConflictTargets(states)
	result := make([]InterfaceFaultTarget, 0, len(s.deviceStates))
	for device, snapshot := range states {
		target := InterfaceFaultTarget{
			Device:     device.Name,
			Interfaces: make([]string, 0, len(snapshot.Network.Interfaces)),
			ErrorTypes: make(map[string][]string),
		}
		for _, iface := range snapshot.Network.Interfaces {
			kinds := s.interfaceFaultCapabilities(device, iface.Name, conflicts[device][iface.Name])
			if len(kinds) != 0 {
				target.Interfaces = append(target.Interfaces, iface.Name)
				target.ErrorTypes[iface.Name] = kinds
			}
			if target.Address == "" && iface.Address.IsValid() {
				target.Address = iface.Address.Addr().Unmap().String()
			}
		}
		slices.Sort(target.Interfaces)
		result = append(result, target)
	}
	slices.SortFunc(result, func(a, b InterfaceFaultTarget) int {
		if a.Device < b.Device {
			return -1
		}
		if a.Device > b.Device {
			return 1
		}
		return 0
	})
	return result
}

func (s *Stack) interfaceFaultCapabilities(device *config.Device, iface string, conflict bool) []string {
	kinds := make([]string, 0)
	if s.snmpAgents[device].interfaceFaultObservable(iface) {
		for _, definition := range devicestate.InterfaceFaultDefinitions() {
			if definition.Type != devicestate.FaultPoELoss || s.snmpAgents[device].poeFaultObservable(iface) {
				kinds = append(kinds, definition.Label)
			}
		}
	}
	if conflict {
		kinds = append(kinds, "Duplicate IP")
	}
	return kinds
}

// InterfaceFaultTarget returns the first observable interface from a device's current state.
func (s *Stack) InterfaceFaultTarget(device *config.Device) (string, string, bool) {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	store := s.deviceStates[device]
	if store == nil {
		return "", "", false
	}
	for _, iface := range store.Snapshot().Network.Interfaces {
		if !s.snmpAgents[device].interfaceFaultObservable(iface.Name) {
			continue
		}
		address := ""
		if iface.Address.IsValid() {
			address = iface.Address.Addr().Unmap().String()
		}
		return address, iface.Name, true
	}
	return "", "", false
}

func addActiveInterfaceFaults(
	result map[string]map[string]map[devicestate.FaultType]int,
	deviceIP string,
	faults []devicestate.InterfaceFault,
) {
	for _, fault := range faults {
		if result[deviceIP] == nil {
			result[deviceIP] = make(map[string]map[devicestate.FaultType]int)
		}
		if result[deviceIP][fault.Interface] == nil {
			result[deviceIP][fault.Interface] = make(map[devicestate.FaultType]int)
		}
		result[deviceIP][fault.Interface][fault.Type] = fault.Value
	}
}

func (s *Stack) interfaceFaultTarget(
	deviceTarget string,
) (*config.Device, *devicestate.Store, error) {
	for device, store := range s.deviceStates {
		if device.Name == deviceTarget {
			return device, store, nil
		}
	}
	address, err := netip.ParseAddr(deviceTarget)
	if err != nil {
		return nil, nil, ErrFaultDeviceNotFound
	}
	var matched *config.Device
	for device, store := range s.deviceStates {
		if stateHasAddress(store.Snapshot(), address.Unmap()) {
			if matched != nil {
				return nil, nil, ErrFaultDeviceAmbiguous
			}
			matched = device
		}
	}
	if matched == nil {
		return nil, nil, ErrFaultDeviceNotFound
	}
	return matched, s.deviceStates[matched], nil
}

func stateHasAddress(snapshot devicestate.Snapshot, address netip.Addr) bool {
	for _, iface := range snapshot.Network.Interfaces {
		if iface.Address.IsValid() && iface.Address.Addr().Unmap() == address {
			return true
		}
	}
	return false
}

func stateHasInterface(snapshot devicestate.Snapshot, name string) bool {
	return slices.ContainsFunc(snapshot.Network.Interfaces, func(iface devicestate.Interface) bool {
		return iface.Name == name
	})
}

func (s *Stack) advanceDeviceFaultTelemetry(device *config.Device, now time.Time) {
	group := s.snmpAgents[device]
	store := s.deviceStates[device]
	if group == nil || store == nil {
		return
	}
	group.telemetry.AdvanceInterfaceFaults(
		now,
		store.Snapshot().Faults,
		interfaceFaultSpeeds(device),
	)
}

func interfaceFaultSpeeds(device *config.Device) map[string]int {
	result := make(map[string]int, len(device.Interfaces))
	for _, iface := range device.Interfaces {
		result[iface.Name] = iface.Speed
	}
	return result
}
