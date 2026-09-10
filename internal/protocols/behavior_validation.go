package protocols

import (
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// ValidateConfiguredBehaviorActions checks operation inventory before acquiring packet I/O.
func ValidateConfiguredBehaviorActions(cfg *config.Config) error {
	for _, timeline := range cfg.BehaviorTimelines {
		for _, phase := range timeline.Phases {
			if len(phase.Actions) > 0 {
				// Only operation-bearing configurations pay for an offline preparation:
				// captured scalar inventory must be checked before opening packet I/O.
				stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
				return stack.ValidateBehaviorActions()
			}
		}
	}
	return nil
}
