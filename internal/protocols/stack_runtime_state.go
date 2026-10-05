package protocols

import (
	"errors"
	"fmt"
	"slices"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// ErrDeviceStateNotFound indicates that the scenario has no device by a name.
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

// DiscardedDeviceState is one durable record recovery could not use.
type DiscardedDeviceState struct {
	Err    error
	Device string
}

// RestoreDeviceStates restores every record that still fits its device and
// returns the ones it discarded. Records are per device, and an upgrade can
// change what a scenario authors: a record that names a device the scenario no
// longer has, or no longer validates against its device, is discarded and that
// device starts from the scenario, so one stale record does not cost the whole
// session (#2481). A device with no record starts from the scenario too; it
// has no runtime state to lose. Every record is validated before any store
// changes.
func (s *Stack) RestoreDeviceStates(states map[string]devicestate.State) ([]DiscardedDeviceState, error) {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	byName := make(map[string]*devicestate.Store, len(s.deviceStates))
	for device, store := range s.deviceStates {
		byName[device.Name] = store
	}
	names := make([]string, 0, len(states))
	for name := range states {
		names = append(names, name)
	}
	slices.Sort(names)
	var discarded []DiscardedDeviceState
	restorable := names[:0]
	for _, name := range names {
		store := byName[name]
		if store == nil {
			discarded = append(discarded, DiscardedDeviceState{Device: name, Err: ErrDeviceStateNotFound})
			continue
		}
		if err := store.ValidateState(states[name]); err != nil {
			discarded = append(discarded, DiscardedDeviceState{Device: name, Err: err})
			continue
		}
		restorable = append(restorable, name)
	}
	for _, name := range restorable {
		if err := byName[name].RestoreState(states[name]); err != nil {
			return discarded, fmt.Errorf("restore %s: %w", name, err)
		}
	}
	s.notifications.skipRestoredHistory()
	return discarded, nil
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
