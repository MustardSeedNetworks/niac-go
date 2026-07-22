package devicestate

import "errors"

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
	s.running = cloneConfiguration(checkpoint)
	s.version++
	s.recordEvent(EventCheckpointRestored, name)
	return nil
}
