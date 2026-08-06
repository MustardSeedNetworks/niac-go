package api

import (
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
	"github.com/MustardSeedNetworks/niac-go/internal/topology"
)

type simulationAPIState struct {
	stack      *protocols.Stack
	config     *config.Config
	configPath string
	iface      string
	replay     ReplayManager
}

func (s *Server) SelectSimulation(sessionID string) bool {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	state, ok := s.simulations[sessionID]
	if !ok {
		return false
	}
	s.selectedSimulation = sessionID
	s.applySimulationState(state)
	return true
}

func (s *Server) RemoveSimulation(sessionID string) {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	delete(s.simulations, sessionID)
	if s.selectedSimulation != sessionID {
		return
	}
	s.selectedSimulation = ""
	s.applySimulationState(simulationAPIState{})
}

func (s *Server) applySimulationState(state simulationAPIState) {
	s.cfg.Stack = state.stack
	s.cfg.Config = state.config
	s.cfg.ConfigPath = state.configPath
	s.cfg.Interface = state.iface
	s.cfg.Replay = state.replay
	if state.config == nil {
		s.cfg.Topology = topology.Graph{}
		return
	}
	s.cfg.Topology = topology.Build(state.config)
}
