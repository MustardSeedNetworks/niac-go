package config

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
)

var (
	ErrBehaviorTargetNotFound  = errors.New("behavior target not found")
	ErrBehaviorTargetAmbiguous = errors.New("behavior target is ambiguous")
	ErrBehaviorPhaseEmpty      = errors.New("behavior phase has no traffic or faults")
	ErrBehaviorPhaseOverlap    = errors.New("behavior phases overlap")
)

type behaviorTarget struct {
	count      int
	interfaces map[string]struct{}
}

func validateBehaviorTargets(cfg *Config) error {
	targets := behaviorTargets(cfg)
	for _, timeline := range cfg.BehaviorTimelines {
		if err := validateBehaviorTimeline(timeline, targets); err != nil {
			return fmt.Errorf("behavior timeline %q: %w", timeline.Name, err)
		}
	}
	return nil
}

func behaviorTargets(cfg *Config) map[string]behaviorTarget {
	result := make(map[string]behaviorTarget)
	for _, segment := range cfg.NormalizedSegments() {
		for _, device := range segment.Devices {
			target := result[device.Name]
			target.count++
			if target.interfaces == nil {
				target.interfaces = make(map[string]struct{}, len(device.Interfaces))
			}
			for _, iface := range device.Interfaces {
				target.interfaces[iface.Name] = struct{}{}
			}
			result[device.Name] = target
		}
	}
	return result
}

func validateBehaviorTimeline(timeline BehaviorTimeline, targets map[string]behaviorTarget) error {
	phases := append([]BehaviorPhase(nil), timeline.Phases...)
	slices.SortFunc(phases, func(left, right BehaviorPhase) int {
		return cmp.Compare(left.StartOffset, right.StartOffset)
	})
	var previousEnd int64
	for index, phase := range phases {
		if index > 0 && phase.StartOffset.Nanoseconds() < previousEnd {
			return fmt.Errorf("%w: phase %q", ErrBehaviorPhaseOverlap, phase.Name)
		}
		previousEnd = (phase.StartOffset + phase.Duration).Nanoseconds()
		if len(phase.Traffic) == 0 && len(phase.Faults) == 0 {
			return fmt.Errorf("%w: %q", ErrBehaviorPhaseEmpty, phase.Name)
		}
		for _, traffic := range phase.Traffic {
			if err := validateBehaviorTarget(targets, traffic.Device, traffic.Interface); err != nil {
				return fmt.Errorf("phase %q traffic: %w", phase.Name, err)
			}
		}
		for _, fault := range phase.Faults {
			if err := validateBehaviorTarget(targets, fault.Device, fault.Interface); err != nil {
				return fmt.Errorf("phase %q fault: %w", phase.Name, err)
			}
		}
	}
	return nil
}

func validateBehaviorTarget(targets map[string]behaviorTarget, device, iface string) error {
	target, found := targets[device]
	if !found {
		return fmt.Errorf("%w: device %q", ErrBehaviorTargetNotFound, device)
	}
	if target.count != 1 {
		return fmt.Errorf("%w: device %q", ErrBehaviorTargetAmbiguous, device)
	}
	if _, found = target.interfaces[iface]; !found {
		return fmt.Errorf("%w: interface %q on device %q", ErrBehaviorTargetNotFound, iface, device)
	}
	return nil
}
