package config

import (
	"errors"
	"fmt"
)

// ErrAuthoredFaultValue means an authored fault's value exceeds the ceiling its
// own type carries. The type tags cannot express this: latency is milliseconds
// and stops at 60000 while every rate stops at 100, so one numeric bound on the
// field would either cap latency at a tenth of a second or let a rate reach
// 600 percent.
var ErrAuthoredFaultValue = errors.New("authored fault value exceeds its type's ceiling")

// validateAuthoredFaults checks the conditions a scenario starts in. The
// ceilings are read from the runtime's own catalogs rather than restated, so a
// fault type whose ceiling changes needs no edit here.
func validateAuthoredFaults(cfg *Config) error {
	for _, device := range cfg.Devices {
		for _, fault := range device.Faults {
			if err := checkAuthoredCeiling(device.Name, fault.Type, fault.Value); err != nil {
				return err
			}
		}
		for _, iface := range device.Interfaces {
			for _, fault := range iface.Faults {
				if err := checkAuthoredCeiling(device.Name, fault.Type, fault.Value); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func checkAuthoredCeiling(device, faultType string, value int) error {
	ceiling := interfaceFaultMax
	if definition, isDeviceFault := deviceFaultDefinition(faultType); isDeviceFault {
		ceiling = definition.MaxValue
	}
	if value < 1 || value > ceiling {
		return fmt.Errorf(
			"%w: %s %q accepts at most %d, got %d",
			ErrAuthoredFaultValue, device, faultType, ceiling, value,
		)
	}

	return nil
}
