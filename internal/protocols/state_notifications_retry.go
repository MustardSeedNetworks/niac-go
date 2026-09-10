package protocols

import (
	"log/slog"
	"net"
	"net/netip"
)

type refreshedNotification struct {
	notification pendingNotification
	target       netip.Addr
	mac          net.HardwareAddr
}

func (s *stackDatagramSender) retryNeighborResolution(
	key notificationNeighborKey, resolution *notificationNeighborResolution,
) {
	s.stack.reloadMu.RLock()
	defer s.stack.reloadMu.RUnlock()
	s.refreshNeighborNotifications(key, resolution)
	s.sendNeighborProbe(key, resolution, true)
}

func (s *stackDatagramSender) refreshNeighborNotifications(
	key notificationNeighborKey, resolution *notificationNeighborResolution,
) {
	s.mu.Lock()
	if s.pending[key] != resolution {
		s.mu.Unlock()
		return
	}
	queued := append([]pendingNotification(nil), resolution.notifications...)
	version := resolution.refreshVersion
	s.mu.Unlock()
	if len(queued) == 0 {
		return
	}
	var retained []pendingNotification
	var redirected []refreshedNotification
	for _, notification := range queued {
		if notification.originDevice == nil {
			retained = append(retained, notification)
			continue
		}
		current, target, mac, err := s.prepareNotificationIntent(notification)
		if err != nil {
			continue
		}
		if target == key.address && mac == nil {
			retained = append(retained, current)
		} else {
			redirected = append(redirected, refreshedNotification{notification: current, target: target, mac: mac})
		}
	}
	s.mu.Lock()
	if s.pending[key] != resolution || resolution.refreshVersion != version {
		s.mu.Unlock()
		return
	}
	resolution.notifications = append(retained, resolution.notifications[len(queued):]...)
	resolution.refreshVersion++
	if len(resolution.notifications) == 0 && len(resolution.hostPackets) == 0 {
		delete(s.pending, key)
	}
	s.mu.Unlock()
	for _, current := range redirected {
		if err := s.dispatchNotification(current.notification, current.target, current.mac, true); err != nil {
			slog.Debug("changed notification route rejected", "error", err)
		}
	}
}
