package config

import (
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func behaviorTimelinesToYAML(timelines []BehaviorTimeline) []converter.BehaviorTimeline {
	result := make([]converter.BehaviorTimeline, len(timelines))
	for timelineIndex, timeline := range timelines {
		result[timelineIndex] = converter.BehaviorTimeline{
			Name: timeline.Name, StartOffsetMS: int(timeline.StartOffset / time.Millisecond),
			RepeatCount: timeline.RepeatCount, Phases: make([]converter.BehaviorPhase, len(timeline.Phases)),
		}
		for phaseIndex, phase := range timeline.Phases {
			result[timelineIndex].Phases[phaseIndex] = converter.BehaviorPhase{
				Name: phase.Name, StartOffsetMS: int(phase.StartOffset / time.Millisecond),
				DurationMS: int(phase.Duration / time.Millisecond), Reset: phase.Reset,
				Traffic: behaviorTrafficToYAML(phase.Traffic), Faults: behaviorFaultsToYAML(phase.Faults),
				Actions: behaviorActionsToYAML(phase.Actions),
			}
		}
	}
	return result
}

func behaviorTrafficToYAML(traffic []BehaviorTraffic) []converter.BehaviorTraffic {
	result := make([]converter.BehaviorTraffic, len(traffic))
	for index, action := range traffic {
		result[index] = converter.BehaviorTraffic(action)
	}
	return result
}

func behaviorFaultsToYAML(faults []BehaviorFault) []converter.BehaviorFault {
	result := make([]converter.BehaviorFault, len(faults))
	for index, action := range faults {
		result[index] = converter.BehaviorFault(action)
	}
	return result
}

func convertBehaviorTimelines(authored []converter.BehaviorTimeline) []BehaviorTimeline {
	result := make([]BehaviorTimeline, len(authored))
	for timelineIndex, timeline := range authored {
		result[timelineIndex] = BehaviorTimeline{
			Name: timeline.Name, StartOffset: time.Duration(timeline.StartOffsetMS) * time.Millisecond,
			RepeatCount: timeline.RepeatCount, Phases: make([]BehaviorPhase, len(timeline.Phases)),
		}
		for phaseIndex, phase := range timeline.Phases {
			result[timelineIndex].Phases[phaseIndex] = BehaviorPhase{
				Name: phase.Name, StartOffset: time.Duration(phase.StartOffsetMS) * time.Millisecond,
				Duration: time.Duration(phase.DurationMS) * time.Millisecond, Reset: phase.Reset,
				Traffic: convertBehaviorTraffic(phase.Traffic), Faults: convertBehaviorFaults(phase.Faults),
				Actions: convertBehaviorActions(phase.Actions),
			}
		}
	}
	return result
}

func convertBehaviorTraffic(authored []converter.BehaviorTraffic) []BehaviorTraffic {
	result := make([]BehaviorTraffic, len(authored))
	for index, traffic := range authored {
		result[index] = BehaviorTraffic(traffic)
	}
	return result
}

func convertBehaviorFaults(authored []converter.BehaviorFault) []BehaviorFault {
	result := make([]BehaviorFault, len(authored))
	for index, fault := range authored {
		result[index] = BehaviorFault(fault)
	}
	return result
}

func behaviorActionsToYAML(actions []BehaviorAction) []converter.BehaviorAction {
	result := make([]converter.BehaviorAction, len(actions))
	for i, action := range actions {
		result[i] = converter.BehaviorAction{Device: action.Device, Type: string(action.Type)}
	}
	return result
}

func convertBehaviorActions(actions []converter.BehaviorAction) []BehaviorAction {
	result := make([]BehaviorAction, len(actions))
	for i, action := range actions {
		result[i] = BehaviorAction{Device: action.Device, Type: devicestate.DeviceActionType(action.Type)}
	}
	return result
}
