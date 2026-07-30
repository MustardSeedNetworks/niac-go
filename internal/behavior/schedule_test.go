package behavior_test

import (
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/behavior"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestCompileProducesStableRepeatedTransitions(t *testing.T) {
	timelines := []config.BehaviorTimeline{{
		Name: "uplink degradation", StartOffset: time.Second, RepeatCount: 2,
		Phases: []config.BehaviorPhase{{
			Name: "congested", StartOffset: 2 * time.Second, Duration: 3 * time.Second, Reset: true,
			Traffic: []config.BehaviorTraffic{{Device: "access-1", Interface: "Gi0/48", Utilization: 85}},
			Faults: []config.BehaviorFault{{
				Device: "access-1", Interface: "Gi0/48", Type: "packet_discards", Value: 12,
			}},
		}},
	}}

	transitions := behavior.Compile(timelines)
	if len(transitions) != 4 {
		t.Fatalf("Compile() transition count = %d, want 4", len(transitions))
	}
	wantOffsets := []time.Duration{3 * time.Second, 6 * time.Second, 8 * time.Second, 11 * time.Second}
	for index, want := range wantOffsets {
		if transitions[index].Offset != want {
			t.Errorf("transition %d offset = %s, want %s", index, transitions[index].Offset, want)
		}
	}
	if transitions[0].Actions[0].Type != devicestate.FaultUtilization ||
		transitions[0].Actions[0].Value != 85 {
		t.Fatalf("traffic action = %+v", transitions[0].Actions[0])
	}
	for _, transition := range []behavior.Transition{transitions[1], transitions[3]} {
		for _, action := range transition.Actions {
			if action.Value != 0 {
				t.Fatalf("reset action = %+v, want zero value", action)
			}
		}
	}
}

func TestCompileClearsBeforeApplyingAtSharedBoundary(t *testing.T) {
	timelines := []config.BehaviorTimeline{{
		Name: "handoff", RepeatCount: 1,
		Phases: []config.BehaviorPhase{
			{
				Name: "warning", Duration: time.Second, Reset: true,
				Faults: []config.BehaviorFault{{
					Device: "switch-1", Interface: "Gi0/1", Type: "fcs_errors", Value: 5,
				}},
			},
			{
				Name: "critical", StartOffset: time.Second, Duration: time.Second, Reset: true,
				Faults: []config.BehaviorFault{{
					Device: "switch-1", Interface: "Gi0/1", Type: "fcs_errors", Value: 25,
				}},
			},
		},
	}}

	transitions := behavior.Compile(timelines)
	if len(transitions) != 3 {
		t.Fatalf("Compile() transition count = %d, want 3", len(transitions))
	}
	boundary := transitions[1]
	if boundary.Offset != time.Second || len(boundary.Actions) != 2 ||
		boundary.Actions[0].Value != 0 || boundary.Actions[1].Value != 25 {
		t.Fatalf("shared boundary = %+v", boundary)
	}
}
