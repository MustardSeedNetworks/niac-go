package config

import (
	"cmp"
	"errors"
	"fmt"
	"slices"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

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
		if len(phase.Traffic) == 0 && len(phase.Faults) == 0 && len(phase.Actions) == 0 {
			return fmt.Errorf("%w: %q", ErrBehaviorPhaseEmpty, phase.Name)
		}
		for _, traffic := range phase.Traffic {
			if err := validateBehaviorTarget(targets, traffic.Device, traffic.Interface); err != nil {
				return fmt.Errorf("phase %q traffic: %w", phase.Name, err)
			}
		}
		for _, fault := range phase.Faults {
			if err := validateBehaviorFault(targets, fault); err != nil {
				return fmt.Errorf("phase %q fault: %w", phase.Name, err)
			}
		}
		if err := validateBehaviorActions(targets, phase.Actions); err != nil {
			return fmt.Errorf("phase %q action: %w", phase.Name, err)
		}
	}
	return nil
}

// ErrBehaviorActionInvalid indicates an unknown device operation.
var ErrBehaviorActionInvalid = errors.New("invalid behavior action")

// ErrBehaviorActionDuplicate indicates a repeated device operation in one phase.
var ErrBehaviorActionDuplicate = errors.New("duplicate behavior action")

// ErrBehaviorActionUnavailable indicates disabled protocols required by an operation.
var ErrBehaviorActionUnavailable = errors.New(
	"device operation requires served SNMP and enabled STP for topology changes",
)

func validateBehaviorActions(targets map[string]behaviorTarget, actions []BehaviorAction) error {
	seen := make(map[BehaviorAction]struct{}, len(actions))
	for _, action := range actions {
		if action.Type != devicestate.ActionReboot && action.Type != devicestate.ActionSTPTopologyChange {
			return fmt.Errorf("%w: %q", ErrBehaviorActionInvalid, action.Type)
		}
		if err := validateBehaviorDevice(targets, action.Device); err != nil {
			return err
		}
		device := targets[action.Device].device
		if (!SNMPv2Enabled(device.SNMPConfig) && !SNMPv3Enabled(device.SNMPv3Config)) ||
			(action.Type == devicestate.ActionSTPTopologyChange && (device.STPConfig == nil || !device.STPConfig.Enabled)) {
			return fmt.Errorf("%w: %s", ErrBehaviorActionUnavailable, action.Device)
		}
		if _, exists := seen[action]; exists {
			return fmt.Errorf("%w: %s %s", ErrBehaviorActionDuplicate, action.Device, action.Type)
		}
		seen[action] = struct{}{}
	}
	return nil
}
