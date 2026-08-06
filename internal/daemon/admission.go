package daemon

import (
	"fmt"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// Safety bounds for the whole daemon, not for one session. Per-config limits
// multiply once several sessions run at once: a 1000-device ceiling checked
// against each config in turn permits 1000 devices per session, which is not a
// ceiling at all. These are technical capacity limits, never an entitlement.
const (
	// maxActiveSessions bounds concurrent runtimes on one daemon. Each owns a
	// protocol stack, goroutines and an ingress queue, so the count itself is a
	// resource.
	maxActiveSessions = 16
)

// admitSessionLocked decides whether a session may start given everything
// already running. sessionID names the session being started; if it is already
// active its own devices are excluded, because a replacement swaps a session
// rather than adding one. Caller holds d.mu.
func (d *Daemon) admitSessionLocked(sessionID string, cfg *config.Config) error {
	if cfg == nil || d.sessions == nil {
		return nil
	}
	incoming := cfg.DeviceCount()

	active, devices := 0, 0
	for id, sim := range d.sessions.sessions {
		if id == sessionID {
			// Replaced, not added.
			continue
		}
		active++
		if sim != nil && sim.cfg != nil {
			devices += sim.cfg.DeviceCount()
		}
	}

	if _, replacing := d.sessions.sessions[sessionID]; !replacing && active+1 > maxActiveSessions {
		return fmt.Errorf("%w: %d sessions already running, limit is %d",
			api.ErrSimulationSessionCapacity, active, maxActiveSessions)
	}
	if total := devices + incoming; total > api.MaxDeviceCount {
		return fmt.Errorf(
			"%w: %d devices already running plus %d requested exceeds the %d-device daemon limit",
			api.ErrSimulationDeviceCapacity, devices, incoming, api.MaxDeviceCount)
	}
	return nil
}

// aggregateUsageLocked reports what the daemon is currently carrying, so an
// operator can see how close to the budgets they are before a start is refused.
// Caller holds d.mu.
func (d *Daemon) aggregateUsageLocked() api.DaemonCapacity {
	capacity := api.DaemonCapacity{
		MaxSessions: maxActiveSessions,
		MaxDevices:  api.MaxDeviceCount,
	}
	if d.sessions == nil {
		return capacity
	}
	for _, sim := range d.sessions.sessions {
		capacity.Sessions++
		if sim != nil && sim.cfg != nil {
			capacity.Devices += sim.cfg.DeviceCount()
		}
	}
	return capacity
}
