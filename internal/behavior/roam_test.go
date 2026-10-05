package behavior_test

import (
	"errors"
	"reflect"
	"testing"
	"testing/synctest"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/behavior"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

const roamingStation = "02:c0:17:a4:03:6b"

// roamEvery30s is the W1 shape: the station spends 15 s on AP-02 out of every
// 30 s, for two cycles.
func roamEvery30s() []config.BehaviorTimeline {
	return []config.BehaviorTimeline{{
		Name: "roaming", RepeatCount: 2,
		Phases: []config.BehaviorPhase{
			{Name: "on-ap-01", Duration: 15 * time.Second, Traffic: []config.BehaviorTraffic{{
				Device: "MED-AP-01", Interface: "Gi0", Utilization: 10,
			}}},
			{
				Name: "on-ap-02", StartOffset: 15 * time.Second, Duration: 15 * time.Second, Reset: true,
				Roams: []config.BehaviorRoam{{Station: roamingStation, From: "MED-AP-01", To: "MED-AP-02"}},
			},
		},
	}}
}

// TestCompileRoamsBackOnReset: reset sends the station back where it came
// from, so a repeating phase is a station going back and forth rather than a
// one-way trip that the second cycle would find already made.
func TestCompileRoamsBackOnReset(t *testing.T) {
	away := behavior.RoamAction{Station: roamingStation, From: "MED-AP-01", To: "MED-AP-02"}
	back := behavior.RoamAction{Station: roamingStation, From: "MED-AP-02", To: "MED-AP-01", Return: true}
	type roamsAt struct {
		offset time.Duration
		roams  []behavior.RoamAction
	}
	want := []roamsAt{
		{15 * time.Second, []behavior.RoamAction{away}},
		{30 * time.Second, []behavior.RoamAction{back}},
		{45 * time.Second, []behavior.RoamAction{away}},
		{60 * time.Second, []behavior.RoamAction{back}},
	}

	var got []roamsAt
	for _, transition := range behavior.Compile(roamEvery30s()) {
		if len(transition.RoamActions) > 0 {
			got = append(got, roamsAt{transition.Offset, transition.RoamActions})
		}
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("roams by offset = %+v, want %+v", got, want)
	}
}

// TestCompileCarriesTheCauseBothWays: the roam away applies the cause and the
// roam back undoes it, so both directions carry it and only the second is a
// return.
func TestCompileCarriesTheCauseBothWays(t *testing.T) {
	timelines := roamEvery30s()
	timelines[0].Phases[1].Roams[0].Cause = devicestate.RoamCauseRadioDown

	var got []behavior.RoamAction
	for _, transition := range behavior.Compile(timelines)[:3] {
		got = append(got, transition.RoamActions...)
	}

	want := []behavior.RoamAction{
		{Station: roamingStation, From: "MED-AP-01", To: "MED-AP-02", Cause: devicestate.RoamCauseRadioDown},
		{
			Station: roamingStation,
			From:    "MED-AP-02",
			To:      "MED-AP-01",
			Cause:   devicestate.RoamCauseRadioDown,
			Return:  true,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("first cycle roams = %+v, want %+v", got, want)
	}
}

type refusingRoamTarget struct{ recordingTarget }

func (*refusingRoamTarget) RoamStation(behavior.RoamAction) error {
	return errors.New("station is not associated to this device")
}

func TestRunnerRoamsInOrderAndStopsOnARefusal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		target := &recordingTarget{}
		runner := behavior.New(target, behavior.Compile(roamEvery30s()))
		runner.Start()
		time.Sleep(time.Minute)
		synctest.Wait()

		if state := runner.Status().State; state != "completed" {
			t.Fatalf("state = %s, want completed", state)
		}
		away := behavior.RoamAction{Station: roamingStation, From: "MED-AP-01", To: "MED-AP-02"}
		back := behavior.RoamAction{Station: roamingStation, From: "MED-AP-02", To: "MED-AP-01", Return: true}
		if want := []behavior.RoamAction{away, back, away, back}; !reflect.DeepEqual(target.roamActions, want) {
			t.Fatalf("roams = %+v, want %+v", target.roamActions, want)
		}
	})

	synctest.Test(t, func(t *testing.T) {
		runner := behavior.New(&refusingRoamTarget{}, behavior.Compile(roamEvery30s()))
		runner.Start()
		time.Sleep(time.Minute)
		synctest.Wait()

		if status := runner.Status(); status.State != "failed" || status.LastError == "" {
			t.Fatalf("status = %+v, want a failed run carrying the refusal", status)
		}
	})
}
