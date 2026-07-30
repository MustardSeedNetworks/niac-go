// Package behavior compiles and runs deterministic saved behavior timelines.
package behavior

import (
	"cmp"
	"fmt"
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

// PhaseRef identifies one compiled phase while retaining its authored label.
type PhaseRef struct {
	ID    string
	Label string
}

// Transition groups every action that occurs at one offset from runtime start.
type Transition struct {
	Offset      time.Duration
	StartPhases []PhaseRef
	EndPhases   []PhaseRef
	Actions     []Action
}

type scheduledTransition struct {
	offset  time.Duration
	phase   PhaseRef
	end     bool
	actions []Action
}

// Compile produces a stable transition sequence for every finite repetition.
func Compile(timelines []config.BehaviorTimeline) []Transition {
	scheduled := make([]scheduledTransition, 0)
	for timelineIndex, timeline := range timelines {
		cycleDuration := behaviorCycleDuration(timeline.Phases)
		for repetition := range timeline.RepeatCount {
			cycleStart := timeline.StartOffset + time.Duration(repetition)*cycleDuration
			for phaseIndex, phase := range timeline.Phases {
				actions := behaviorActions(phase)
				phaseRef := PhaseRef{
					ID:    fmt.Sprintf("%d:%d:%d", timelineIndex, repetition, phaseIndex),
					Label: timeline.Name + ": " + phase.Name,
				}
				scheduled = append(scheduled, scheduledTransition{
					offset: cycleStart + phase.StartOffset, phase: phaseRef, actions: actions,
				})
				end := scheduledTransition{
					offset: cycleStart + phase.StartOffset + phase.Duration,
					phase:  phaseRef, end: true,
				}
				if phase.Reset {
					end.actions = resetActions(actions)
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

func behaviorActions(phase config.BehaviorPhase) []Action {
	actions := make([]Action, 0, len(phase.Traffic)+len(phase.Faults))
	for _, traffic := range phase.Traffic {
		actions = append(actions, Action{
			Device: traffic.Device, Interface: traffic.Interface,
			Type: devicestate.FaultUtilization, Value: traffic.Utilization,
		})
	}
	for _, fault := range phase.Faults {
		actions = append(actions, Action{
			Device: fault.Device, Interface: fault.Interface,
			Type: devicestate.FaultType(fault.Type), Value: fault.Value,
		})
	}
	return actions
}

func resetActions(actions []Action) []Action {
	result := make([]Action, len(actions))
	copy(result, actions)
	for index := range result {
		result[index].Value = 0
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
	}
	return result
}
