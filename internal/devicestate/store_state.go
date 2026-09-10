package devicestate

import (
	"errors"
	"slices"
)

// ErrStateInvalid indicates that a durable state record cannot be restored.
var ErrStateInvalid = errors.New("device state record is invalid")

// Configuration is one serialisable configuration bank -- running or startup.
type Configuration struct {
	Identity Identity
	Network  Network
}

// Checkpoint is one saved scenario point in a durable state record.
type Checkpoint struct {
	Telemetry       DeviceTelemetry
	Name            string
	Configuration   Configuration
	InterfaceFaults []InterfaceFault
	DeviceFaults    []DeviceFault
}

// State is everything a store holds that the compiled configuration cannot
// reproduce: the two configuration banks as they stand now, the faults armed
// on both axes, saved checkpoints and the authoritative event history.
//
// Authored configuration is deliberately absent. It is the compiled scenario,
// so it comes back from the configuration file on the next start; restoring a
// stale copy of it would make ResetAuthored return a device to a scenario the
// file no longer describes.
type State struct {
	Telemetry       DeviceTelemetry
	ConsumedActions []ConsumedDeviceAction
	Running         Configuration
	Startup         Configuration
	InterfaceFaults []InterfaceFault
	DeviceFaults    []DeviceFault
	Checkpoints     []Checkpoint
	Events          []Event
	Version         uint64
}

// Version returns the store's current transaction counter. It is the cheap
// way to ask whether anything changed since a durable record was written,
// without cloning the configuration a full export copies.
func (s *Store) Version() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.version
}

// ExportState returns a durable copy of runtime state.
//
// The configuration banks are exported raw rather than through Snapshot():
// a snapshot projects carrier faults onto the interfaces it returns, so a
// snapshot taken while link_down was armed and restored as running state
// would latch OperUp and CarrierUp false with no fault left to clear.
func (s *Store) ExportState() State {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return State{
		Telemetry:       s.telemetry,
		ConsumedActions: exportConsumedActions(s.consumedActions),
		Running:         exportConfiguration(s.running),
		Startup:         exportConfiguration(s.startup),
		InterfaceFaults: sortedInterfaceFaults(s.faults),
		DeviceFaults:    sortedDeviceFaults(s.deviceFaults),
		Checkpoints:     exportCheckpoints(s.checkpoints),
		Events:          cloneEvents(s.events),
		Version:         s.version,
	}
}

// RestoreState replaces runtime state with a durable record.
//
// The event history is restored verbatim rather than appended to: it is the
// sequence a consuming NMS already saw, and the events recovery itself raises
// while reinstalling an identical compiled network are not transitions the
// replayed device lived through. Nothing is re-emitted -- observers are
// notified of the restored state, but past events do not become new traps.
func (s *Store) RestoreState(state State) error {
	s.mu.Lock()
	if err := validateState(state, s.authored.network.Interfaces); err != nil {
		s.mu.Unlock()
		return err
	}
	s.running = importConfiguration(state.Running)
	s.startup = importConfiguration(state.Startup)
	s.faults = importInterfaceFaults(state.InterfaceFaults)
	s.deviceFaults = importDeviceFaults(state.DeviceFaults)
	s.telemetry = state.Telemetry
	s.consumedActions = importConsumedActions(state.ConsumedActions)
	s.checkpoints = importCheckpoints(state.Checkpoints)
	s.events = cloneEvents(state.Events)
	s.version = state.Version
	// Recovery starts a new management lifetime, without replaying transitions.
	s.interfaceTransitions = nil
	s.updateInterfaceTransitions()
	observer := s.changeObserver
	restored := s.snapshot(s.running)
	s.mu.Unlock()

	// The observer maintains the stack's address index and the DHCP server's
	// derived address, so a restore that skipped it would answer for the
	// compiled addresses while reporting the restored ones.
	if observer != nil {
		observer(restored)
	}
	return nil
}

func exportConfiguration(source configuration) Configuration {
	return Configuration{Identity: source.identity, Network: cloneNetwork(source.network)}
}

func importConfiguration(source Configuration) configuration {
	return configuration{identity: source.Identity, network: cloneNetwork(source.Network)}
}

func exportCheckpoints(saved map[string]checkpoint) []Checkpoint {
	names := make([]string, 0, len(saved))
	for name := range saved {
		names = append(names, name)
	}
	slices.Sort(names)
	result := make([]Checkpoint, 0, len(names))
	for _, name := range names {
		point := saved[name]
		result = append(result, Checkpoint{
			Telemetry:       point.telemetry,
			Name:            name,
			Configuration:   exportConfiguration(point.config),
			InterfaceFaults: sortedInterfaceFaults(point.faults),
			DeviceFaults:    sortedDeviceFaults(point.deviceFaults),
		})
	}
	return result
}

func importCheckpoints(saved []Checkpoint) map[string]checkpoint {
	result := make(map[string]checkpoint, len(saved))
	for _, point := range saved {
		result[point.Name] = checkpoint{
			telemetry:    point.Telemetry,
			config:       importConfiguration(point.Configuration),
			faults:       importInterfaceFaults(point.InterfaceFaults),
			deviceFaults: importDeviceFaults(point.DeviceFaults),
		}
	}
	return result
}

func importInterfaceFaults(faults []InterfaceFault) map[interfaceFaultKey]InterfaceFault {
	result := make(map[interfaceFaultKey]InterfaceFault, len(faults))
	for _, fault := range faults {
		result[interfaceFaultKey{interfaceName: fault.Interface, faultType: fault.Type}] = fault
	}
	return result
}

func importDeviceFaults(faults []DeviceFault) map[DeviceFaultType]DeviceFault {
	result := make(map[DeviceFaultType]DeviceFault, len(faults))
	for _, fault := range faults {
		result[fault.Type] = fault
	}
	return result
}
