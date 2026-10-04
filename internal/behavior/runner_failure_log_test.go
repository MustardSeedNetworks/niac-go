package behavior_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/behavior"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

type refusingInterfaceTarget struct{ recordingTarget }

func (*refusingInterfaceTarget) SetInterfaceFault(string, string, devicestate.FaultType, int) error {
	return devicestate.ErrInterfaceNotFound
}

// `daemon --once` has no API to read BehaviorStatus from, so a refused fault
// must reach the log, naming the phase (niac-go#2482).
func TestRunnerLogsRefusedTransition(t *testing.T) {
	var out bytes.Buffer
	logging.SetOutput(&out)
	t.Cleanup(func() { logging.SetOutput(nil) })

	runner := behavior.New(&refusingInterfaceTarget{}, []behavior.Transition{{
		StartPhases: []behavior.PhaseRef{{ID: "0/0", Label: "dbg: down"}},
		Actions: []behavior.Action{{
			Device: "LAB-EDGE-R1", Interface: "HundredGigabitEthernet0/0/2",
			Type: devicestate.FaultLinkDown, Value: 1,
		}},
	}})
	t.Cleanup(runner.Stop)
	runner.Start()
	deadline := time.Now().Add(5 * time.Second)
	for runner.Status().State == "running" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if status := runner.Status(); status.State != "failed" {
		t.Fatalf("Status() = %+v, want failed", status)
	}
	line := out.String()
	for _, want := range []string{"BEHAVIOR:", "dbg: down", devicestate.ErrInterfaceNotFound.Error()} {
		if !strings.Contains(line, want) {
			t.Fatalf("log %q does not contain %q", line, want)
		}
	}
}
