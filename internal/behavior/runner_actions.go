package behavior

import (
	"context"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func (r *Runner) applyTransition(ctx context.Context, transition Transition) bool {
	for _, action := range transition.Actions {
		if err := r.target.SetInterfaceFault(action.Device, action.Interface, action.Type, action.Value); err != nil {
			r.finish("failed", err.Error())
			return false
		}
	}
	for _, action := range transition.DeviceActions {
		if err := r.applyDeviceFault(action); err != nil {
			r.finish("failed", err.Error())
			return false
		}
	}
	for _, action := range transition.AddressActions {
		if ctx.Err() != nil {
			r.finish("stopped", "")
			return false
		}
		if err := r.applyInterfaceAddressFault(action); err != nil {
			r.finish("failed", err.Error())
			return false
		}
	}
	if !r.applyPrefixActions(ctx, transition.PrefixActions) {
		return false
	}
	for _, action := range transition.OneShotActions {
		if ctx.Err() != nil {
			r.finish("stopped", "")
			return false
		}
		if err := r.target.ExecuteDeviceAction(action.Device, action.Type, action.ID); err != nil {
			r.finish("failed", err.Error())
			return false
		}
	}
	return true
}

func (r *Runner) applyDeviceFault(action DeviceAction) error {
	if action.Clear {
		return r.target.ClearDeviceFault(action.Device, action.Type)
	}
	if action.Type == devicestate.FaultDuplicateDHCPOffer {
		return r.target.SetDeviceAddressFault(action.Device, action.Type, action.Address)
	}
	return r.target.SetDeviceFault(action.Device, action.Type, action.Value)
}
