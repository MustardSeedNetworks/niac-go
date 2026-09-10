package devicestate

import (
	"errors"
	"slices"
	"time"
)

const (
	maxConsumedDeviceActions = 100_000
	maxDeviceActionIDLength  = 128
)

// Device action errors distinguish invalid input, identity reuse and exhausted history.
var (
	ErrDeviceActionInvalid  = errors.New("device action is invalid")
	ErrDeviceActionConflict = errors.New("device action identity already has a different type")
	ErrDeviceActionLimit    = errors.New("device action history limit reached")
)

// DeviceTelemetry holds the effects of one-shot actions, not pending commands.
type DeviceTelemetry struct {
	RebootedAt   time.Time
	STPChanges   uint32
	STPChangedAt time.Time
}

// ConsumedDeviceAction prevents a committed timeline action from being replayed.
// Its enclosing runtime record supplies the simulation generation.
type ConsumedDeviceAction struct {
	ID   string
	Type DeviceActionType
}

// ExecuteDeviceAction atomically records an action's effect and consumption.
// Repeated delivery of the same identity returns false without emitting an event.
func (s *Store) ExecuteDeviceAction(kind DeviceActionType, id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !validDeviceAction(kind) || id == "" || len(id) > maxDeviceActionIDLength {
		return false, ErrDeviceActionInvalid
	}
	if previous, found := s.consumedActions[id]; found {
		if previous != kind {
			return false, ErrDeviceActionConflict
		}
		return false, nil
	}
	if len(s.consumedActions) >= maxConsumedDeviceActions {
		return false, ErrDeviceActionLimit
	}
	now := s.now().UTC()
	var eventKind EventKind
	switch kind {
	case ActionReboot:
		s.telemetry.RebootedAt = now
		eventKind = EventDeviceRebooted
	case ActionSTPTopologyChange:
		s.telemetry.STPChanges++
		s.telemetry.STPChangedAt = now
		eventKind = EventSTPTopologyChanged
	}
	if s.consumedActions == nil {
		s.consumedActions = make(map[string]DeviceActionType)
	}
	s.consumedActions[id] = kind
	s.version++
	s.appendEvent(Event{Version: s.version, Kind: eventKind, Target: id, Timestamp: now})
	s.signalChange()
	return true, nil
}

// DeviceTelemetry returns a consistent copy without cloning network inventory.
func (s *Store) DeviceTelemetry() DeviceTelemetry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.telemetry
}

func validDeviceAction(kind DeviceActionType) bool {
	return kind == ActionReboot || kind == ActionSTPTopologyChange
}

func exportConsumedActions(consumed map[string]DeviceActionType) []ConsumedDeviceAction {
	ids := make([]string, 0, len(consumed))
	for id := range consumed {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	result := make([]ConsumedDeviceAction, 0, len(ids))
	for _, id := range ids {
		result = append(result, ConsumedDeviceAction{ID: id, Type: consumed[id]})
	}
	return result
}

func importConsumedActions(consumed []ConsumedDeviceAction) map[string]DeviceActionType {
	result := make(map[string]DeviceActionType, len(consumed))
	for _, action := range consumed {
		result[action.ID] = action.Type
	}
	return result
}

func validActionState(state State) bool {
	if !validDeviceTelemetry(state.Telemetry) ||
		len(state.ConsumedActions) > maxConsumedDeviceActions {
		return false
	}
	seen := make(map[string]DeviceActionType, len(state.ConsumedActions))
	counts := make(map[DeviceActionType]uint32)
	for _, action := range state.ConsumedActions {
		if seen[action.ID] != "" || action.ID == "" || len(action.ID) > maxDeviceActionIDLength ||
			!validDeviceAction(action.Type) {
			return false
		}
		seen[action.ID] = action.Type
		counts[action.Type]++
	}
	if !telemetryHasConsumedActions(state.Telemetry, counts) {
		return false
	}
	for _, point := range state.Checkpoints {
		if !telemetryHasConsumedActions(point.Telemetry, counts) {
			return false
		}
	}
	for _, event := range state.Events {
		if event.Kind == EventDeviceRebooted && seen[event.Target] != ActionReboot ||
			event.Kind == EventSTPTopologyChanged && seen[event.Target] != ActionSTPTopologyChange {
			return false
		}
	}
	return true
}

func telemetryHasConsumedActions(
	telemetry DeviceTelemetry,
	counts map[DeviceActionType]uint32,
) bool {
	return (telemetry.RebootedAt.IsZero() || counts[ActionReboot] > 0) &&
		(telemetry.STPChangedAt.IsZero() || counts[ActionSTPTopologyChange] > 0) &&
		telemetry.STPChanges <= counts[ActionSTPTopologyChange]
}

func validDeviceTelemetry(telemetry DeviceTelemetry) bool {
	if telemetry.STPChangedAt.IsZero() && telemetry.STPChanges != 0 {
		return false
	}
	for _, timestamp := range []time.Time{telemetry.RebootedAt, telemetry.STPChangedAt} {
		if !timestamp.IsZero() && timestamp.Before(time.Unix(0, 0)) {
			return false
		}
	}
	return true
}
