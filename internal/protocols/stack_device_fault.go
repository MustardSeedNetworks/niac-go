package protocols

import (
	"cmp"
	"errors"
	"slices"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// ErrFaultServiceAbsent means the target device does not run the service the
// fault suppresses, so arming it would change nothing observable.
var ErrFaultServiceAbsent = errors.New("fault target does not run the affected service")

// DeviceFaultTarget describes one device's device-scoped fault surface.
type DeviceFaultTarget struct {
	Device  string
	Address string
	Faults  []devicestate.DeviceFaultType
}

// SetDeviceFault arms one device-service fault on the stack-owned device state.
func (s *Stack) SetDeviceFault(
	deviceTarget string,
	faultType devicestate.DeviceFaultType,
	value int,
) error {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	device, store, err := s.interfaceFaultTarget(deviceTarget)
	if err != nil {
		return err
	}
	// A zero value clears, and clearing a fault the device could never serve
	// is harmless; only arming one has to be refused. An unknown type has no
	// label and belongs to the store to reject, not to this guard: otherwise a
	// typo would report as a missing service.
	if value != 0 && faultType.Label() != "" && !s.deviceServesFault(device, faultType) {
		return ErrFaultServiceAbsent
	}
	return store.SetDeviceFault(faultType, value)
}

// ClearDeviceFaults clears every device-service fault on one device.
func (s *Stack) ClearDeviceFaults(deviceTarget string) error {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	_, store, err := s.interfaceFaultTarget(deviceTarget)
	if err != nil {
		return err
	}
	store.ClearDeviceFaults()
	return nil
}

// ClearAllDeviceFaults clears every device-service fault in the stack.
func (s *Stack) ClearAllDeviceFaults() {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	for _, store := range s.deviceStates {
		store.ClearDeviceFaults()
	}
}

// ActiveDeviceFaults returns a JSON-ready snapshot keyed by device name.
func (s *Stack) ActiveDeviceFaults() map[string]map[devicestate.DeviceFaultType]int {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	result := make(map[string]map[devicestate.DeviceFaultType]int)
	for device, store := range s.deviceStates {
		faults := store.Snapshot().DeviceFaults
		if len(faults) == 0 {
			continue
		}
		byType := make(map[devicestate.DeviceFaultType]int, len(faults))
		for _, fault := range faults {
			byType[fault.Type] = fault.Value
		}
		result[device.Name] = byType
	}
	return result
}

// DeviceFaultTargets returns the devices that run a faultable service, with
// the fault types each one can serve.
func (s *Stack) DeviceFaultTargets() []DeviceFaultTarget {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	result := make([]DeviceFaultTarget, 0, len(s.deviceStates))
	for device, store := range s.deviceStates {
		faults := s.servableDeviceFaults(device)
		if len(faults) == 0 {
			continue
		}
		result = append(result, DeviceFaultTarget{
			Device:  device.Name,
			Address: firstDeviceAddress(store.Snapshot()),
			Faults:  faults,
		})
	}
	slices.SortFunc(result, func(left, right DeviceFaultTarget) int {
		return cmp.Compare(left.Device, right.Device)
	})
	return result
}

// deviceFaultActive reports whether a device-service fault is armed on device.
// Protocol handlers call this per request, so it must stay cheap.
func (s *Stack) deviceFaultActive(
	device *config.Device, faultType devicestate.DeviceFaultType,
) bool {
	store := s.deviceStates[device]
	return store != nil && store.DeviceFaultActive(faultType)
}

// deviceFaultValue returns the armed value of a device-service fault, or zero
// when it is not armed. Handlers call it per request, so it must stay cheap.
func (s *Stack) deviceFaultValue(
	device *config.Device, faultType devicestate.DeviceFaultType,
) int {
	store := s.deviceStates[device]
	if store == nil {
		return 0
	}

	return store.DeviceFaultValue(faultType)
}

// deviceServesFault reports whether the device runs the service a fault
// suppresses. Arming a DHCP fault on a device with no DHCP server would look
// applied and do nothing, which is exactly the failure ErrFaultUnobservable
// prevents on the interface axis.
func (s *Stack) deviceServesFault(device *config.Device, faultType devicestate.DeviceFaultType) bool {
	switch faultType {
	case devicestate.FaultCaptivePortal:
		return device.HTTPConfig != nil && device.HTTPConfig.Enabled
	case devicestate.FaultCPUPercent, devicestate.FaultMemoryPercent, devicestate.FaultDiskPercent:
		return s.snmpAgents[device].resourceFaultObservable(faultType, config.SNMPv2Enabled(device.SNMPConfig))
	case devicestate.FaultLatency:
		// Latency suppresses no service: every simulated device answers the
		// echo requests addressed to it, so every device can be made slow.
		return true
	case devicestate.FaultDHCPNoOffer:
		return device.DHCPConfig != nil
	case devicestate.FaultDNSNXDomain, devicestate.FaultDNSTimeout:
		// NXDOMAIN must change an otherwise successful lookup; an empty
		// authored DNS service already returns NXDOMAIN for every name.
		return device.DNSConfig != nil &&
			(len(device.DNSConfig.ForwardRecords) > 0 || len(device.DNSConfig.ReverseRecords) > 0)
	}

	return false
}

func (s *Stack) servableDeviceFaults(device *config.Device) []devicestate.DeviceFaultType {
	result := make([]devicestate.DeviceFaultType, 0, len(devicestate.DeviceFaultDefinitions()))
	for _, definition := range devicestate.DeviceFaultDefinitions() {
		if s.deviceServesFault(device, definition.Type) {
			result = append(result, definition.Type)
		}
	}
	return result
}

func firstDeviceAddress(snapshot devicestate.Snapshot) string {
	for _, iface := range snapshot.Network.Interfaces {
		if iface.Address.IsValid() {
			return iface.Address.Addr().Unmap().String()
		}
	}
	return ""
}
