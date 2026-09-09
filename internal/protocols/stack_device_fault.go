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
	Faults  []devicestate.FaultType
}

// SetDeviceFault arms one device-service fault on the stack-owned device state.
func (s *Stack) SetDeviceFault(
	deviceTarget string,
	faultType devicestate.FaultType,
	value int,
) error {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	device, store, err := s.interfaceFaultTarget(deviceTarget)
	if err != nil {
		return err
	}
	// The store owns type validity; ask it first so an interface fault sent to
	// this axis reports as the wrong type rather than as a missing service.
	// A zero value clears, and clearing a fault the device could never serve
	// is harmless; only arming one has to be refused.
	if value != 0 && isDeviceFaultType(faultType) && !deviceServesFault(device, faultType) {
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
func (s *Stack) ActiveDeviceFaults() map[string]map[devicestate.FaultType]int {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	result := make(map[string]map[devicestate.FaultType]int)
	for device, store := range s.deviceStates {
		faults := store.Snapshot().DeviceFaults
		if len(faults) == 0 {
			continue
		}
		byType := make(map[devicestate.FaultType]int, len(faults))
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
		faults := servableDeviceFaults(device)
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
	device *config.Device, faultType devicestate.FaultType,
) bool {
	store := s.deviceStates[device]
	return store != nil && store.DeviceFaultActive(faultType)
}

// deviceServesFault reports whether the device runs the service a fault
// suppresses. Arming a DHCP fault on a device with no DHCP server would look
// applied and do nothing, which is exactly the failure ErrFaultUnobservable
// prevents on the interface axis.
func deviceServesFault(device *config.Device, faultType devicestate.FaultType) bool {
	switch faultType {
	case devicestate.FaultDHCPNoOffer:
		return device.DHCPConfig != nil
	case devicestate.FaultDNSNXDomain, devicestate.FaultDNSTimeout:
		// A non-nil DNSConfig proves nothing: parseDNSConfig hands every
		// YAML-loaded device an empty one, so a workstation that authored no
		// `dns:` block would otherwise look like a DNS server. Records are
		// what make a device answer queries.
		return device.DNSConfig != nil &&
			(len(device.DNSConfig.ForwardRecords) > 0 || len(device.DNSConfig.ReverseRecords) > 0)
	default:
		return false
	}
}

func isDeviceFaultType(faultType devicestate.FaultType) bool {
	return slices.ContainsFunc(
		devicestate.DeviceFaultDefinitions(),
		func(definition devicestate.FaultDefinition) bool { return definition.Type == faultType },
	)
}

func servableDeviceFaults(device *config.Device) []devicestate.FaultType {
	result := make([]devicestate.FaultType, 0, len(devicestate.DeviceFaultDefinitions()))
	for _, definition := range devicestate.DeviceFaultDefinitions() {
		if deviceServesFault(device, definition.Type) {
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
