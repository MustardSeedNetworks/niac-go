package protocols

import "log/slog"

func (s *stackDatagramSender) currentNeighborProbe(
	key notificationNeighborKey,
	resolution *notificationNeighborResolution,
) (pendingNotification, bool) {
	s.mu.Lock()
	if s.pending[key] != resolution {
		s.mu.Unlock()
		return pendingNotification{}, false
	}
	if len(resolution.notifications) != 0 {
		probe := resolution.notifications[0]
		s.mu.Unlock()
		return probe, true
	}
	packet := resolution.hostPackets[0]
	s.mu.Unlock()
	s.stack.reloadMu.RLock()
	route, handled, err := s.stack.hostPacketRoute(packet)
	s.stack.reloadMu.RUnlock()
	s.mu.Lock()
	if s.pending[key] != resolution {
		s.mu.Unlock()
		return pendingNotification{}, false
	}
	if len(resolution.notifications) != 0 {
		probe := resolution.notifications[0]
		s.mu.Unlock()
		return probe, true
	}
	if err == nil && handled && route.target == key.address {
		s.mu.Unlock()
		return pendingNotification{
			egressDevice:   packet.generatedHost,
			vlan:           key.vlan,
			neighborSource: route.iface.Address.Addr(),
		}, true
	}
	packets := resolution.hostPackets
	delete(s.pending, key)
	if resolution.timer != nil {
		resolution.timer.Stop()
	}
	s.mu.Unlock()
	for _, queued := range packets {
		if sendErr := s.stack.send(queued); sendErr != nil {
			slog.Debug("obsolete host resolution packet rejected", "error", sendErr)
		}
	}
	return pendingNotification{}, false
}
