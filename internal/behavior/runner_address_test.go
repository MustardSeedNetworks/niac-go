package behavior_test

import (
	"fmt"
	"net/netip"
	"slices"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/behavior"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

type addressTarget struct {
	recordingTarget

	calls []string
}

func (target *addressTarget) SetDeviceFault(device string, kind devicestate.DeviceFaultType, value int) error {
	target.mu.Lock()
	defer target.mu.Unlock()
	target.calls = append(target.calls, fmt.Sprintf("number %s %s %d", device, kind, value))
	return nil
}

func (target *addressTarget) SetDeviceAddressFault(
	device string,
	kind devicestate.DeviceFaultType,
	address netip.Addr,
) error {
	target.mu.Lock()
	defer target.mu.Unlock()
	target.calls = append(target.calls, fmt.Sprintf("address %s %s %s", device, kind, address))
	return nil
}

func (target *addressTarget) ClearDeviceFault(device string, kind devicestate.DeviceFaultType) error {
	target.mu.Lock()
	defer target.mu.Unlock()
	target.calls = append(target.calls, fmt.Sprintf("clear %s %s", device, kind))
	return nil
}

func TestRunnerAddressFaultUsesTypedSetterAndExplicitClear(t *testing.T) {
	target := new(addressTarget)
	transitions := behavior.Compile([]config.BehaviorTimeline{{
		Name: "address", RepeatCount: 1,
		Phases: []config.BehaviorPhase{{
			Name: "conflict", Duration: time.Millisecond, Reset: true,
			Faults: []config.BehaviorFault{
				{Device: "server", Type: "duplicate_dhcp_offer", Address: netip.MustParseAddr("192.0.2.20")},
				{Device: "server", Type: "latency", Value: 50},
			},
		}},
	}})
	runner := behavior.New(target, transitions)
	runner.Start()
	t.Cleanup(runner.Stop)
	deadline := time.Now().Add(time.Second)
	for runner.Status().State == "running" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if status := runner.Status(); status.State != "completed" || status.AppliedTransitions != 2 {
		t.Fatalf("runner status = %+v", status)
	}
	target.mu.Lock()
	defer target.mu.Unlock()
	want := []string{
		"address server duplicate_dhcp_offer 192.0.2.20", "number server latency 50",
		"clear server duplicate_dhcp_offer", "clear server latency",
	}
	if !slices.Equal(target.calls, want) {
		t.Fatalf("calls = %v, want %v", target.calls, want)
	}
}
