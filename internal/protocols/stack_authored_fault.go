package protocols

import (
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// applyAuthoredFaults arms the conditions the configuration says the scenario
// starts in. It runs after the device stores hold their interfaces and before
// anything is served, so an authored fault is true at the first poll rather
// than arriving with a behavior phase.
//
// It goes through the stack's own setters, not the stores', so an authored
// fault meets exactly the guards an injected one does: a counter fault on an
// interface no agent reports, a PoE fault on a port supplying no power, or a
// DNS fault on a device serving no DNS is refused here the same way the API
// refuses it. A refusal is logged and skipped rather than failing construction:
// the configuration already validated, so this is the eligibility check that
// only the assembled stack can make, and a scenario that loses one fault is
// still worth serving.
func (s *Stack) applyAuthoredFaults(cfg *config.Config) {
	if cfg == nil {
		return
	}
	for index := range cfg.Devices {
		device := &cfg.Devices[index]
		for _, fault := range device.Faults {
			s.armAuthoredFault(device.Name, "", fault.Type, fault.Value)
		}
		for _, iface := range device.Interfaces {
			for _, fault := range iface.Faults {
				s.armAuthoredFault(device.Name, iface.Name, fault.Type, fault.Value)
			}
		}
	}
}

func (s *Stack) armAuthoredFault(device, interfaceName, faultType string, value int) {
	var err error
	if interfaceName == "" {
		err = s.setDeviceFaultNoLock(device, devicestate.DeviceFaultType(faultType), value)
	} else {
		err = s.setInterfaceFaultNoLock(
			device, interfaceName, devicestate.FaultType(faultType), value, time.Now(),
		)
	}
	if err != nil {
		logging.Warningf(
			"authored fault %s on %s %s not armed: %v", faultType, device, interfaceName, err,
		)
	}
}
