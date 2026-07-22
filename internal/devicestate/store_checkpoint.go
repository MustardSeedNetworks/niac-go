package devicestate

import (
	"errors"
	"slices"
)

// ErrCheckpointNotFound indicates that a named checkpoint does not exist.
var ErrCheckpointNotFound = errors.New("checkpoint not found")

// SaveCheckpoint captures running configuration under name.
func (s *Store) SaveCheckpoint(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.checkpoints == nil {
		s.checkpoints = make(map[string]configuration)
	}
	s.checkpoints[name] = cloneConfiguration(s.running)
	s.version++
	s.recordEvent(EventCheckpointSaved, name)
}

// RestoreCheckpoint replaces running configuration from a named checkpoint.
func (s *Store) RestoreCheckpoint(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	checkpoint, ok := s.checkpoints[name]
	if !ok {
		return ErrCheckpointNotFound
	}
	s.replaceRunning(checkpoint, EventCheckpointRestored, name)
	return nil
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
