package daemon

import "github.com/MustardSeedNetworks/niac-go/internal/protocols"

// SessionStats reports what a running session has put on and taken off the
// wire. The one-shot runtime needs it for its summary: the status struct
// carries no counters, and reading them back out of run history would make the
// summary depend on storage being enabled.
func (d *Daemon) SessionStats(sessionID string) (protocols.Statistics, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.sessions == nil {
		return protocols.Statistics{}, false
	}
	sim := d.sessions.get(sessionID)
	if sim == nil || sim.stack == nil {
		return protocols.Statistics{}, false
	}

	return sim.stack.GetStats(), true
}
