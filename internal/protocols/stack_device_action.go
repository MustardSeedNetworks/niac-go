package protocols

import (
	"errors"

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
	if kind != devicestate.ActionReboot && kind != devicestate.ActionSTPTopologyChange {
		return devicestate.ErrDeviceActionInvalid
	}
	if kind == devicestate.ActionSTPTopologyChange &&
		(device.STPConfig == nil || !device.STPConfig.Enabled) {
		return ErrDeviceActionUnobservable
	}
	if !s.snmpAgents[device].deviceActionObservable(kind, snmpEnabled(device.SNMPConfig)) {
		return ErrDeviceActionUnobservable
	}
	_, err = store.ExecuteDeviceAction(kind, id)
	return err
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
