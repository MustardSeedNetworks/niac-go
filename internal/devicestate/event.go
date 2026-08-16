package devicestate

import "time"

const maxEventHistory = 1024

// EventKind identifies an authoritative state transition.
type EventKind string

// Event kinds recorded in a device's authoritative state history.
const (
	EventNetworkInstalled   EventKind = "network.installed"
	EventIdentityUpdated    EventKind = "identity.updated"
	EventInterfaceUpdated   EventKind = "interface.updated"
	EventStartupSaved       EventKind = "startup.saved"
	EventStartupReloaded    EventKind = "startup.reloaded"
	EventStartupErased      EventKind = "startup.erased"
	EventAuthoredReset      EventKind = "authored.reset"
	EventCheckpointSaved    EventKind = "checkpoint.saved"
	EventCheckpointRestored EventKind = "checkpoint.restored"
	EventVLANUpdated        EventKind = "vlan.updated"
	EventRouterUpdated      EventKind = "router.updated"
	EventRouteUpdated       EventKind = "route.updated"
	EventFaultUpdated       EventKind = "fault.updated"
	EventFaultCleared       EventKind = "fault.cleared"
)

// Event records one committed state transition.
type Event struct {
	Version           uint64
	Kind              EventKind
	Target            string
	Interface         *Interface
	PreviousInterface *Interface
	InterfaceIndex    int
	Timestamp         time.Time
}

// Events returns an ordered copy of committed state transitions.
func (s *Store) Events() []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return cloneEvents(s.events)
}

// EventsAfter returns retained transitions newer than version. Complete is
// false when the requested cursor predates the bounded history window.
func (s *Store) EventsAfter(version uint64) ([]Event, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.events) > 0 && version+1 < s.events[0].Version {
		return cloneEvents(s.events), false
	}
	for index, event := range s.events {
		if event.Version > version {
			return cloneEvents(s.events[index:]), true
		}
	}
	return nil, true
}

// Changes is signaled after one or more transitions commit. Consumers should
// call EventsAfter to retrieve every transition; signals are intentionally
// coalesced so a slow consumer cannot block configuration changes.
func (s *Store) Changes() <-chan struct{} { return s.changes }

func (s *Store) recordEvent(kind EventKind, target string) {
	event := Event{Version: s.version, Kind: kind, Target: target, Timestamp: s.now().UTC()}
	s.appendEvent(event)
	s.signalChange()
}

func (s *Store) recordInterfaceEvent(index int, previous, current Interface) {
	previousCopy := cloneInterface(previous)
	currentCopy := cloneInterface(current)
	s.appendEvent(Event{
		Version: s.version, Kind: EventInterfaceUpdated, Target: current.Name,
		Interface: &currentCopy, PreviousInterface: &previousCopy, InterfaceIndex: index,
		Timestamp: s.now().UTC(),
	})
	s.signalChange()
}

func (s *Store) appendEvent(event Event) {
	if len(s.events) == maxEventHistory {
		copy(s.events, s.events[1:])
		s.events[len(s.events)-1] = event
		return
	}
	s.events = append(s.events, event)
}

func (s *Store) signalChange() {
	if s.changeObserver != nil {
		s.changeObserver(s.snapshot(s.running))
	}
	select {
	case s.changes <- struct{}{}:
	default:
	}
	if s.changeSignal != nil {
		select {
		case s.changeSignal <- struct{}{}:
		default:
		}
	}
}

func cloneEvents(events []Event) []Event {
	result := make([]Event, len(events))
	copy(result, events)
	for index := range result {
		if result[index].Interface != nil {
			cloned := cloneInterface(*result[index].Interface)
			result[index].Interface = &cloned
		}
		if result[index].PreviousInterface != nil {
			cloned := cloneInterface(*result[index].PreviousInterface)
			result[index].PreviousInterface = &cloned
		}
	}
	return result
}
