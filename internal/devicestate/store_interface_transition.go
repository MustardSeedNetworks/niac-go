package devicestate

import (
	"slices"
	"time"
)

type interfaceTransition struct {
	operUp    bool
	changedAt time.Time
}

// InterfaceLastChange returns when the interface last entered its effective
// operational state, or zero if unchanged since installation or recovery.
func (s *Store) InterfaceLastChange(name string) time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.interfaceTransitions[name].changedAt
}

// updateInterfaceTransitions runs under the store lock before notifying readers.
// Configuration banks and the bounded event journal do not own this lifetime.
func (s *Store) updateInterfaceTransitions() {
	interfaces := slices.Clone(s.running.network.Interfaces)
	applyCarrierFaults(interfaces, s.faults)
	next := make(map[string]interfaceTransition, len(interfaces))
	var changedAt time.Time
	for _, iface := range interfaces {
		previous, exists := s.interfaceTransitions[iface.Name]
		if exists && previous.operUp != iface.OperUp {
			if changedAt.IsZero() {
				changedAt = s.now()
			}
			previous.changedAt = changedAt
		}
		previous.operUp = iface.OperUp
		next[iface.Name] = previous
	}
	s.interfaceTransitions = next
}
