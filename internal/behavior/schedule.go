// Package behavior compiles and runs deterministic saved behavior timelines.
package behavior

import (
	"cmp"
	"fmt"
	"net/netip"
	"slices"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// Action is one authoritative interface-fault update.
type Action struct {
	Device    string
	Interface string
	Type      devicestate.FaultType
	Value     int
}

// DeviceAction is one authoritative device-service fault update. It is a
// separate type from Action rather than an Action with an empty interface:
// the two axes have separate setters, and a compiled action that could carry
// either type would put the choice back at apply time.
type DeviceAction struct {
	Device  string
	Type    devicestate.DeviceFaultType
	Value   int
	Address netip.Addr
	Clear   bool
}

// OneShotAction carries a stable identity for a device operation within a simulation generation.
type OneShotAction struct {
	Device string
	Type   devicestate.DeviceActionType
	ID     string
}

// PhaseRef identifies one compiled phase while retaining its authored label.
type PhaseRef struct {
	ID    string
	Label string
}

// Transition groups every action that occurs at one offset from runtime start.
type Transition struct {
	Offset         time.Duration
	StartPhases    []PhaseRef
	EndPhases      []PhaseRef
	Actions        []Action
	DeviceActions  []DeviceAction
	AddressActions []InterfaceAddressAction
	PrefixActions  []InterfacePrefixAction
	OneShotActions []OneShotAction
}

type scheduledTransition struct {
	offset         time.Duration
	phase          PhaseRef
	end            bool
	actions        []Action
	deviceActions  []DeviceAction
	addressActions []InterfaceAddressAction
	prefixActions  []InterfacePrefixAction
	oneShotActions []OneShotAction
}

// Compile produces a stable transition sequence for every finite repetition.
func Compile(timelines []config.BehaviorTimeline) []Transition {
	scheduled := make([]scheduledTransition, 0)
	for timelineIndex, timeline := range timelines {
		cycleDuration := behaviorCycleDuration(timeline.Phases)
		for repetition := range timeline.RepeatCount {
			cycleStart := timeline.StartOffset + time.Duration(repetition)*cycleDuration
			for phaseIndex, phase := range timeline.Phases {
				actions, deviceActions := behaviorActions(phase)
				phaseRef := PhaseRef{
					ID:    fmt.Sprintf("%d:%d:%d", timelineIndex, repetition, phaseIndex),
					Label: timeline.Name + ": " + phase.Name,
				}
				scheduled = append(scheduled, scheduledTransition{
					offset: cycleStart + phase.StartOffset, phase: phaseRef,
					actions: actions, deviceActions: deviceActions,
					oneShotActions: compileOneShotActions(phase.Actions, phaseRef.ID),
					addressActions: interfaceAddressActions(phase, false),
					prefixActions:  interfacePrefixActions(phase, false),
				})
				end := scheduledTransition{
					offset: cycleStart + phase.StartOffset + phase.Duration,
					phase:  phaseRef, end: true,
				}
				if phase.Reset {
					end.actions = resetActions(actions)
					end.deviceActions = resetDeviceActions(deviceActions)
					end.addressActions = interfaceAddressActions(phase, true)
					end.prefixActions = interfacePrefixActions(phase, true)
				}
				scheduled = append(scheduled, end)
			}
		}
	}
	slices.SortStableFunc(scheduled, func(left, right scheduledTransition) int {
		if order := cmp.Compare(left.offset, right.offset); order != 0 {
			return order
		}
		if left.end == right.end {
			return 0
		}
		if left.end {
			return -1
		}
		return 1
	})
	return groupTransitions(scheduled)
}

func behaviorCycleDuration(phases []config.BehaviorPhase) time.Duration {
	var duration time.Duration
	for _, phase := range phases {
		duration = max(duration, phase.StartOffset+phase.Duration)
	}
	return duration
}

// behaviorActions splits a phase across the two fault axes. An authored fault
// with no interface is device-scoped; config validation has already refused
// the mismatches, so the interface alone decides here.
func behaviorActions(phase config.BehaviorPhase) ([]Action, []DeviceAction) {
	actions := make([]Action, 0, len(phase.Traffic)+len(phase.Faults))
	deviceActions := make([]DeviceAction, 0, len(phase.Faults))
	for _, traffic := range phase.Traffic {
		actions = append(actions, Action{
			Device: traffic.Device, Interface: traffic.Interface,
			Type: devicestate.FaultUtilization, Value: traffic.Utilization,
		})
	}
	for _, fault := range phase.Faults {
		if fault.Type == string(devicestate.FaultDuplicateIP) || fault.Type == string(devicestate.FaultBadMask) {
			continue
		}
		if fault.Interface == "" {
			deviceActions = append(deviceActions, DeviceAction{
				Device: fault.Device,
				Type:   devicestate.DeviceFaultType(fault.Type), Value: fault.Value, Address: fault.Address,
			})
			continue
		}
		actions = append(actions, Action{
			Device: fault.Device, Interface: fault.Interface,
			Type: devicestate.FaultType(fault.Type), Value: fault.Value,
		})
	}
	return actions, deviceActions
}

func resetActions(actions []Action) []Action {
	result := make([]Action, len(actions))
	copy(result, actions)
	for index := range result {
		result[index].Value = 0
	}
	return result
}

func resetDeviceActions(actions []DeviceAction) []DeviceAction {
	result := make([]DeviceAction, len(actions))
	copy(result, actions)
	for index := range result {
		result[index].Value = 0
		result[index].Address = netip.Addr{}
		result[index].Clear = true
	}
	return result
}

func groupTransitions(scheduled []scheduledTransition) []Transition {
	result := make([]Transition, 0, len(scheduled))
	for _, current := range scheduled {
		if len(result) == 0 || result[len(result)-1].Offset != current.offset {
			result = append(result, Transition{Offset: current.offset})
		}
		transition := &result[len(result)-1]
		if current.end {
			transition.EndPhases = append(transition.EndPhases, current.phase)
		} else {
			transition.StartPhases = append(transition.StartPhases, current.phase)
		}
		transition.Actions = append(transition.Actions, current.actions...)
		transition.DeviceActions = append(transition.DeviceActions, current.deviceActions...)
		transition.AddressActions = append(transition.AddressActions, current.addressActions...)
		transition.PrefixActions = append(transition.PrefixActions, current.prefixActions...)
		transition.OneShotActions = append(transition.OneShotActions, current.oneShotActions...)
	}
	return result
}

func compileOneShotActions(actions []config.BehaviorAction, phaseID string) []OneShotAction {
	result := make([]OneShotAction, len(actions))
	for index, action := range actions {
		result[index] = OneShotAction{
			Device: action.Device, Type: action.Type, ID: fmt.Sprintf("%s:%d", phaseID, index),
		}
	}
	return result
}
