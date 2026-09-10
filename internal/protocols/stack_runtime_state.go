package protocols

import (
	"errors"
	"fmt"
	"slices"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// ErrDeviceStateNotFound indicates that no simulated device carries a name.
var ErrDeviceStateNotFound = errors.New("device state not found")

// ExportDeviceStates returns durable runtime state for every simulated device,
// keyed by authored device name. The name is the key rather than the address
// because addresses are themselves runtime state: a device whose interface was
// renumbered while it ran must still be recognised after a restart.
func (s *Stack) ExportDeviceStates() map[string]devicestate.State {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	result := make(map[string]devicestate.State, len(s.deviceStates))
	for device, store := range s.deviceStates {
		if store == nil {
			continue
		}
		result[device.Name] = store.ExportState()
	}
	return result
}

// RestoreDeviceStates validates the complete record before restoring any store.
// Every authored device must be present, and no unknown device is accepted.
func (s *Stack) RestoreDeviceStates(states map[string]devicestate.State) error {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	byName := make(map[string]*devicestate.Store, len(s.deviceStates))
	for device, store := range s.deviceStates {
		byName[device.Name] = store
	}
	if len(states) != len(byName) {
		return fmt.Errorf("%w: runtime record does not match scenario devices", ErrDeviceStateNotFound)
	}
	names := make([]string, 0, len(states))
	for name := range states {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		store := byName[name]
		if store == nil {
			return fmt.Errorf("%w: %s", ErrDeviceStateNotFound, name)
		}
		if err := store.ValidateState(states[name]); err != nil {
			return fmt.Errorf("validate %s: %w", name, err)
		}
	}
	for _, name := range names {
		if err := byName[name].RestoreState(states[name]); err != nil {
			return fmt.Errorf("restore %s: %w", name, err)
		}
	}
	s.notifications.skipRestoredHistory()
	return nil
}

// RuntimeStateVersion sums every device's transaction counter. A durable
// writer polls this rather than exporting on a timer: an export deep-copies
// each device's network, and most ticks have nothing to write.
func (s *Stack) RuntimeStateVersion() uint64 {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	var total uint64
	for _, store := range s.deviceStates {
		if store != nil {
			total += store.Version()
		}
	}
	return total
}
