package daemon

import (
	"fmt"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// Session registry errors, re-exported from internal/api so callers that
// only import daemon don't need an api import for errors.Is comparisons.
var (
	// ErrSessionIDRequired means a session operation was called with an empty session ID.
	ErrSessionIDRequired = api.ErrSimulationSessionIDRequired
	// ErrPhysicalVLANInUse means the requested VLAN is already bound to another active session.
	ErrPhysicalVLANInUse = api.ErrSimulationSessionConflict
	// ErrInterfaceInUse means the requested physical interface is already bound to another active session.
	ErrInterfaceInUse = api.ErrSimulationSessionConflict
)

type sessionRegistry struct {
	sessions map[string]*Simulation
}

func newSessionRegistry() *sessionRegistry {
	return &sessionRegistry{sessions: make(map[string]*Simulation)}
}

func (r *sessionRegistry) validateReplacement(sessionID string, binding fabric.Binding) error {
	if sessionID == "" {
		return ErrSessionIDRequired
	}
	for id, active := range r.sessions {
		if id == sessionID || active.Binding.Interface != binding.Interface {
			continue
		}
		if active.Binding.Mode != fabric.ModeTrunk || binding.Mode != fabric.ModeTrunk {
			return fmt.Errorf("%w: %s", ErrInterfaceInUse, binding.Interface)
		}
		if active.Binding.AccessVLAN == binding.AccessVLAN {
			return fmt.Errorf(
				"%w: VLAN %d on %s",
				ErrPhysicalVLANInUse,
				binding.AccessVLAN,
				binding.Interface,
			)
		}
	}
	return nil
}

func (r *sessionRegistry) replace(sessionID string, simulation *Simulation) *Simulation {
	previous := r.sessions[sessionID]
	r.sessions[sessionID] = simulation
	return previous
}

func (r *sessionRegistry) remove(sessionID string) *Simulation {
	active := r.sessions[sessionID]
	delete(r.sessions, sessionID)
	return active
}

func (r *sessionRegistry) get(sessionID string) *Simulation {
	return r.sessions[sessionID]
}

func (r *sessionRegistry) len() int {
	return len(r.sessions)
}

func (r *sessionRegistry) first() *Simulation {
	for _, simulation := range r.sessions {
		return simulation
	}
	return nil
}
