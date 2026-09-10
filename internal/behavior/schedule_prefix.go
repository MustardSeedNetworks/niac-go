package behavior

import (
	"context"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// InterfacePrefixAction separates an active /0 prefix from an explicit clear.
type InterfacePrefixAction struct {
	Device string
	Fault  devicestate.InterfacePrefixFault
	Clear  bool
}

func (r *Runner) applyPrefixActions(ctx context.Context, actions []InterfacePrefixAction) bool {
	for _, action := range actions {
		if ctx.Err() != nil {
			r.finish("stopped", "")
			return false
		}
		if err := r.applyInterfacePrefixFault(action); err != nil {
			r.finish("failed", err.Error())
			return false
		}
	}
	return true
}

func interfacePrefixActions(phase config.BehaviorPhase, clearFault bool) []InterfacePrefixAction {
	var result []InterfacePrefixAction
	for _, fault := range phase.Faults {
		if fault.Type != string(devicestate.FaultBadMask) {
			continue
		}
		result = append(result, InterfacePrefixAction{
			Device: fault.Device, Clear: clearFault,
			Fault: devicestate.InterfacePrefixFault{
				Interface:  fault.Interface,
				Type:       devicestate.FaultBadMask,
				PrefixBits: fault.PrefixBits,
			},
		})
	}
	return result
}

func (r *Runner) applyInterfacePrefixFault(action InterfacePrefixAction) error {
	if action.Clear {
		return r.target.ClearInterfacePrefixFault(
			action.Device,
			action.Fault.Interface,
			action.Fault.Type,
		)
	}
	return r.target.SetInterfacePrefixFault(action.Device, action.Fault)
}
