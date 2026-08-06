package devicestate

import (
	"errors"
	"maps"
	"slices"
)

// ErrCheckpointNotFound indicates that a named checkpoint does not exist.
var ErrCheckpointNotFound = errors.New("checkpoint not found")

// checkpoint is a point a scenario can return to. Active faults are part of
// what a scenario is doing, so a checkpoint that captured only configuration
// restored a device that looked right and behaved wrong.
type checkpoint struct {
	config configuration
	faults map[interfaceFaultKey]InterfaceFault
}

// SaveCheckpoint captures running configuration and active faults under name.
func (s *Store) SaveCheckpoint(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.checkpoints == nil {
		s.checkpoints = make(map[string]checkpoint)
	}
	s.checkpoints[name] = checkpoint{
		config: cloneConfiguration(s.running),
		faults: cloneFaults(s.faults),
	}
	s.version++
	s.recordEvent(EventCheckpointSaved, name)
}

// RestoreCheckpoint replaces running configuration and active faults from a
// named checkpoint. Faults raised since the save are cleared, so restoring a
// checkpoint taken while healthy returns a healthy device.
func (s *Store) RestoreCheckpoint(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	saved, ok := s.checkpoints[name]
	if !ok {
		return ErrCheckpointNotFound
	}
	s.faults = cloneFaults(saved.faults)
	s.replaceRunning(saved.config, EventCheckpointRestored, name)
	return nil
}

// cloneFaults copies the fault map so a checkpoint cannot be rewritten by
// later changes to the live one, and vice versa.
func cloneFaults(
	faults map[interfaceFaultKey]InterfaceFault,
) map[interfaceFaultKey]InterfaceFault {
	cloned := make(map[interfaceFaultKey]InterfaceFault, len(faults))
	maps.Copy(cloned, faults)
	return cloned
}

func (s *Store) recordChangedInterfaceEvents(previous []Interface) {
	byName := make(map[string]Interface, len(previous))
	for _, iface := range previous {
		byName[iface.Name] = iface
	}
	for index, current := range s.running.network.Interfaces {
		before, found := byName[current.Name]
		if found && !sameInterface(before, current) {
			s.recordInterfaceEvent(index+1, before, current)
		}
	}
}

func sameInterface(left, right Interface) bool {
	return left.Name == right.Name && left.Network == right.Network &&
		left.Address == right.Address && left.Description == right.Description &&
		slices.Equal(left.VLANs, right.VLANs) && left.AdminUp == right.AdminUp &&
		left.OperUp == right.OperUp && left.CarrierUp == right.CarrierUp
}
