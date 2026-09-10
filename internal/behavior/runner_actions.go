package behavior

import "context"

func (r *Runner) applyTransition(ctx context.Context, transition Transition) bool {
	for _, action := range transition.Actions {
		if err := r.target.SetInterfaceFault(action.Device, action.Interface, action.Type, action.Value); err != nil {
			r.finish("failed", err.Error())
			return false
		}
	}
	for _, action := range transition.DeviceActions {
		if err := r.target.SetDeviceFault(action.Device, action.Type, action.Value); err != nil {
			r.finish("failed", err.Error())
			return false
		}
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
