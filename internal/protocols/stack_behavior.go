package protocols

import (
	"github.com/MustardSeedNetworks/niac-go/internal/behavior"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func (s *Stack) configureBehaviorTimelines(cfg *config.Config) {
	var timelines []config.BehaviorTimeline
	if cfg != nil {
		timelines = cfg.BehaviorTimelines
	}
	runner := behavior.New(s, behavior.Compile(timelines))
	s.behaviorMu.Lock()
	s.behaviorRunner = runner
	s.behaviorMu.Unlock()
}

func (s *Stack) startBehaviorTimelines() {
	s.behaviorMu.RLock()
	runner := s.behaviorRunner
	s.behaviorMu.RUnlock()
	if runner != nil {
		runner.Start()
	}
}

func (s *Stack) stopBehaviorTimelines() {
	s.behaviorMu.RLock()
	runner := s.behaviorRunner
	s.behaviorMu.RUnlock()
	if runner != nil {
		runner.Stop()
	}
}

// BehaviorStatus returns the current saved-timeline replay state.
func (s *Stack) BehaviorStatus() behavior.Status {
	s.behaviorMu.RLock()
	runner := s.behaviorRunner
	s.behaviorMu.RUnlock()
	if runner == nil {
		return behavior.Status{State: "idle"}
	}
	return runner.Status()
}
