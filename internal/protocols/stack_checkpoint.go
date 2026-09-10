package protocols

import (
	"fmt"
	"slices"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// SaveCheckpoint captures every device's running configuration and active
// faults under name, and reports how many devices it covered.
//
// A scenario is the unit an acceptance run resets to, not a device: saving one
// device and mutating another would restore a state the scenario never had.
func (s *Stack) SaveCheckpoint(name string) int {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()

	for _, store := range s.deviceStates {
		store.SaveCheckpoint(name)
	}

	return len(s.deviceStates)
}

// RestoreCheckpoint returns every device to a named checkpoint. It restores
// nothing unless every device holds that name, so an interrupted restore
// cannot leave half a scenario at the checkpoint and half where it was.
//
// Restoring does not rewind the generation's consumed timeline actions; a
// scenario that has already rebooted a device stays rebooted in its history.
func (s *Stack) RestoreCheckpoint(name string) error {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()

	var missing []string
	for device, store := range s.deviceStates {
		if !store.HasCheckpoint(name) {
			missing = append(missing, device.Name)
		}
	}
	switch {
	case len(missing) == len(s.deviceStates):
		return devicestate.ErrCheckpointNotFound
	case len(missing) > 0:
		slices.Sort(missing)
		return fmt.Errorf("%w on %v", devicestate.ErrCheckpointNotFound, missing)
	}

	for _, store := range s.deviceStates {
		if err := store.RestoreCheckpoint(name); err != nil {
			return err
		}
	}

	return nil
}

// CheckpointNames lists the checkpoints every device can restore.
func (s *Stack) CheckpointNames() []string {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()

	counts := make(map[string]int)
	for _, store := range s.deviceStates {
		for _, name := range store.CheckpointNames() {
			counts[name]++
		}
	}

	names := make([]string, 0, len(counts))
	for name, held := range counts {
		if held == len(s.deviceStates) {
			names = append(names, name)
		}
	}
	slices.Sort(names)

	return names
}
