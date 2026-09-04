package api

import (
	"errors"

	"github.com/MustardSeedNetworks/niac-go/internal/capturering"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
	"github.com/MustardSeedNetworks/niac-go/internal/topology"
)

// ErrSessionNotFound is returned when a request names a session that is not
// running. Naming a session that does not exist is a client error, not an
// empty result: silently falling back to some other session is how one
// browser tab ends up driving another tab's scenario.
var ErrSessionNotFound = errors.New("simulation session was not found")

// sessionRuntime is one running session's state, addressed explicitly. Every
// runtime read and mutation goes through a lookup by session ID rather than a
// process-wide "selected" pointer.
type sessionRuntime struct {
	id    string
	state simulationAPIState
}

// session looks up one running session. Callers must handle the miss; there is
// deliberately no fallback to a default or most-recent session.
func (s *Server) session(sessionID string) (sessionRuntime, error) {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	state, ok := s.simulations[sessionID]
	if !ok {
		return sessionRuntime{}, ErrSessionNotFound
	}
	return sessionRuntime{id: sessionID, state: state}, nil
}

// sessionIDs lists running sessions in a stable order.
func (s *Server) sessionIDs() []string {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	ids := make([]string, 0, len(s.simulations))
	for id := range s.simulations {
		ids = append(ids, id)
	}
	return ids
}

func (r sessionRuntime) config() *config.Config { return r.state.config }

func (r sessionRuntime) stack() *protocols.Stack { return r.state.stack }

func (r sessionRuntime) configPath() string { return r.state.configPath }

func (r sessionRuntime) iface() string { return r.state.iface }

// captureFrames returns the frames the session's ring retains, newest last.
// A session registered without a ring — every test server built by hand, and
// any future caller of RegisterSimulation that predates it — reports an empty
// capture rather than a nil dereference.
func (r sessionRuntime) captureFrames(last int) []capturering.Frame {
	if r.state.capture == nil {
		return nil
	}

	return r.state.capture.Snapshot(last)
}

// topology prefers what the stack is actually running over what was authored,
// matching the projection the global accessor used.
func (r sessionRuntime) topology() topology.Graph {
	if r.state.stack != nil {
		return r.state.stack.RuntimeTopology()
	}
	if r.state.config == nil {
		return topology.Graph{}
	}
	return topology.Build(r.state.config)
}
