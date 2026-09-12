package protocols

import (
	"cmp"
	"errors"
	"fmt"
	"slices"

	"github.com/MustardSeedNetworks/niac-go/internal/config"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// ErrDeviceActionUnobservable means the target cannot publish the action's effect.
var ErrDeviceActionUnobservable = errors.New("device does not support this action")

// ExecuteDeviceAction commits a one-shot event with a generation-local identity.
func (s *Stack) ExecuteDeviceAction(
	deviceTarget string,
	kind devicestate.DeviceActionType,
	id string,
) error {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	device, store, err := s.interfaceFaultTarget(deviceTarget)
	if err != nil {
		return err
	}
	if err = s.validateDeviceAction(device, kind); err != nil {
		return err
	}
	_, err = store.ExecuteDeviceAction(kind, id)
	return err
}

func (s *Stack) validateDeviceAction(device *config.Device, kind devicestate.DeviceActionType) error {
	if !slices.Contains(devicestate.DeviceActionTypes(), kind) {
		return devicestate.ErrDeviceActionInvalid
	}
	if kind == devicestate.ActionSTPTopologyChange &&
		(device.STPConfig == nil || !device.STPConfig.Enabled) {
		return ErrDeviceActionUnobservable
	}
	if !s.snmpAgents[device].deviceActionObservable(kind, config.SNMPv2Enabled(device.SNMPConfig)) {
		return ErrDeviceActionUnobservable
	}
	return nil
}

// ValidateBehaviorTargets verifies observable targets without applying behavior.
func (s *Stack) ValidateBehaviorTargets() error {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	if s.config == nil {
		return nil
	}
	for _, timeline := range s.config.BehaviorTimelines {
		for _, phase := range timeline.Phases {
			if err := s.validateBehaviorAddressFaults(phase.Faults); err != nil {
				return fmt.Errorf("timeline %s phase %s: %w", timeline.Name, phase.Name, err)
			}
			for _, action := range phase.Actions {
				device, _, err := s.interfaceFaultTarget(action.Device)
				if err == nil {
					err = s.validateDeviceAction(device, action.Type)
				}
				if err != nil {
					return fmt.Errorf(
						"timeline %s phase %s device %s: %w",
						timeline.Name,
						phase.Name,
						action.Device,
						err,
					)
				}
			}
		}
	}
	return nil
}

func (g *snmpAgentGroup) deviceActionObservable(
	kind devicestate.DeviceActionType,
	v2Enabled bool,
) bool {
	if g == nil {
		return false
	}
	if g.v3Agent != nil && g.v3Agent.DeviceActionObservable(kind) {
		return true
	}
	if v2Enabled {
		for _, agent := range g.agents {
			if agent.DeviceActionObservable(kind) {
				return true
			}
		}
	}
	return false
}

// DeviceActionTarget is one device and the operations it can actually publish.
// A topology change needs STP enabled and every action needs an agent that can
// report its effect, so the set differs per device the way the fault sets do.
type DeviceActionTarget struct {
	Device  string
	Address string
	Actions []devicestate.DeviceActionType
}

// DeviceActionTargets lists the devices an operator can act on, with the
// operations each one supports.
func (s *Stack) DeviceActionTargets() []DeviceActionTarget {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	result := make([]DeviceActionTarget, 0, len(s.deviceStates))
	for device, store := range s.deviceStates {
		actions := make([]devicestate.DeviceActionType, 0, len(devicestate.DeviceActionTypes()))
		for _, kind := range devicestate.DeviceActionTypes() {
			if s.validateDeviceAction(device, kind) == nil {
				actions = append(actions, kind)
			}
		}
		if len(actions) == 0 {
			continue
		}
		result = append(result, DeviceActionTarget{
			Device: device.Name, Address: firstDeviceAddress(store.Snapshot()), Actions: actions,
		})
	}
	slices.SortFunc(result, func(left, right DeviceActionTarget) int {
		return cmp.Compare(left.Device, right.Device)
	})

	return result
}
