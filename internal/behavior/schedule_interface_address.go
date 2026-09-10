package behavior

import (
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// InterfaceAddressAction applies or clears one explicitly addressed interface outcome.
type InterfaceAddressAction struct {
	Device string
	Fault  devicestate.InterfaceAddressFault
	Clear  bool
}

func interfaceAddressActions(phase config.BehaviorPhase, clearFault bool) []InterfaceAddressAction {
	var result []InterfaceAddressAction
	for _, fault := range phase.Faults {
		if fault.Type != string(devicestate.FaultDuplicateIP) {
			continue
		}
		action := InterfaceAddressAction{
			Device: fault.Device, Clear: clearFault,
			Fault: devicestate.InterfaceAddressFault{Interface: fault.Interface, Type: devicestate.FaultDuplicateIP},
		}
		if !clearFault {
			action.Fault.Address = fault.Address
		}
		result = append(result, action)
	}
	return result
}

func (r *Runner) applyInterfaceAddressFault(action InterfaceAddressAction) error {
	if action.Clear {
		return r.target.ClearInterfaceAddressFault(action.Device, action.Fault.Interface, action.Fault.Type)
	}
	return r.target.SetInterfaceAddressFault(action.Device, action.Fault)
}
