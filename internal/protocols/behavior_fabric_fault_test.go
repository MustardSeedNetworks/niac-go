package protocols

import (
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// A timeline fault on a port with no address must land in a routed session
// exactly as it does in the flat one (niac-go#2482): the fabric's device state
// once held only the ports it placed, so the fault failed with "interface not
// found" and the link never went down.
func TestBehaviorInterfaceFaultAppliesOnUnaddressedFabricPort(t *testing.T) {
	const trunk = "HundredGigabitEthernet1/0/2"
	cfg := placementConfig()
	cfg.BehaviorTimelines = []config.BehaviorTimeline{{
		Name: "trunk-down", RepeatCount: 1,
		Phases: []config.BehaviorPhase{{
			Name: "down", Duration: time.Minute,
			Faults: []config.BehaviorFault{{
				Device: placementCore, Interface: trunk,
				Type: string(devicestate.FaultLinkDown), Value: 1,
			}},
		}},
	}}
	stack := NewStackWithTransport(idleTransport{}, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(compilePlacement(t, cfg))
	stack.startBehaviorTimelines()
	t.Cleanup(stack.stopBehaviorTimelines)

	deadline := time.Now().Add(5 * time.Second)
	for stack.BehaviorStatus().AppliedTransitions == 0 && stack.BehaviorStatus().State == "running" {
		if time.Now().After(deadline) {
			t.Fatal("timeline never applied its first transition")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if status := stack.BehaviorStatus(); status.LastError != "" || status.AppliedTransitions == 0 {
		t.Fatalf("BehaviorStatus() = %+v, want the fault applied", status)
	}
	if got := stack.ActiveInterfaceFaults()[placementCore][trunk][devicestate.FaultLinkDown]; got != 1 {
		t.Fatalf("active link_down on %s %s = %d, want 1", placementCore, trunk, got)
	}
}
