package protocols

import (
	"errors"
	"fmt"

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
	if kind != devicestate.ActionReboot && kind != devicestate.ActionSTPTopologyChange {
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

// ValidateBehaviorActions verifies served scalar inventory without executing operations.
func (s *Stack) ValidateBehaviorActions() error {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	if s.config == nil {
		return nil
	}
	for _, timeline := range s.config.BehaviorTimelines {
		for _, phase := range timeline.Phases {
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
