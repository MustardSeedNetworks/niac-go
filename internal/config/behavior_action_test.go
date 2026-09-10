package config

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestOneShotScheduleBudgetCountsResetOnce(t *testing.T) {
	for _, reset := range []bool{false, true} {
		phase := BehaviorPhase{Reset: reset, Actions: make([]BehaviorAction, 100)}
		timelines := []BehaviorTimeline{{RepeatCount: 1000, Phases: []BehaviorPhase{phase}}}
		if err := validateBehaviorScheduleSize(timelines); err != nil {
			t.Fatal(err)
		}
		timelines[0].Phases[0].Actions = append(timelines[0].Phases[0].Actions, BehaviorAction{})
		if err := validateBehaviorScheduleSize(timelines); !errors.Is(err, ErrBehaviorScheduleTooLarge) {
			t.Fatalf("action expansion reset=%v: %v", reset, err)
		}
	}
}

func TestOneShotScheduleBudgetIncludesFaultResets(t *testing.T) {
	phase := BehaviorPhase{Reset: true, Faults: make([]BehaviorFault, 49), Actions: make([]BehaviorAction, 2)}
	timelines := []BehaviorTimeline{{RepeatCount: 1000, Phases: []BehaviorPhase{phase}}}
	if err := validateBehaviorScheduleSize(timelines); err != nil {
		t.Fatal(err)
	}
	timelines[0].Phases[0].Actions = append(timelines[0].Phases[0].Actions, BehaviorAction{})
	if err := validateBehaviorScheduleSize(timelines); !errors.Is(err, ErrBehaviorScheduleTooLarge) {
		t.Fatal(err)
	}
}

func TestBehaviorActionValidation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		actions []BehaviorAction
		want    error
	}{
		{"unknown", []BehaviorAction{{Device: "switch", Type: "bad"}}, ErrBehaviorActionInvalid},
		{"missing", []BehaviorAction{{Device: "missing", Type: devicestate.ActionReboot}}, ErrBehaviorTargetNotFound},
		{"ambiguous", []BehaviorAction{{Device: "ambiguous", Type: devicestate.ActionReboot}}, ErrBehaviorTargetAmbiguous},
		{"duplicate", []BehaviorAction{{Device: "switch", Type: devicestate.ActionReboot}, {Device: "switch", Type: devicestate.ActionReboot}}, ErrBehaviorActionDuplicate},
		{"distinct", []BehaviorAction{{Device: "switch", Type: devicestate.ActionReboot}, {Device: "switch", Type: devicestate.ActionSTPTopologyChange}}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			targets := map[string]behaviorTarget{"switch": {count: 1}, "ambiguous": {count: 2}}
			if err := validateBehaviorActions(targets, tc.actions); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}
