package behavior_test

import (
	"net/netip"
	"reflect"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/behavior"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestCompileInterfaceAddressFaultRetainsScopeAndReset(t *testing.T) {
	address := netip.MustParseAddr("192.0.2.20")
	got := behavior.Compile([]config.BehaviorTimeline{{
		Name: "conflict", RepeatCount: 1,
		Phases: []config.BehaviorPhase{{
			Name: "duplicate", Duration: time.Second, Reset: true,
			Faults: []config.BehaviorFault{{Device: "host", Interface: "eth0", Type: "duplicate_ip", Address: address}},
		}},
	}})
	if len(got) != 2 || len(got[0].AddressActions) != 1 || len(got[1].AddressActions) != 1 {
		t.Fatalf("wrong transitions: %+v", got)
	}
	first, reset := got[0].AddressActions[0], got[1].AddressActions[0]
	if first.Device != "host" || first.Fault.Interface != "eth0" || first.Fault.Type != devicestate.FaultDuplicateIP ||
		first.Fault.Address != address ||
		first.Clear {
		t.Fatalf("wrong apply: %+v", first)
	}
	if !reset.Clear || reset.Fault.Address.IsValid() || reset.Fault.Interface != "eth0" || reset.Device != "host" {
		t.Fatalf("wrong reset: %+v", reset)
	}
	if len(got[0].Actions) != 0 || len(got[0].DeviceActions) != 0 {
		t.Fatal("address fault leaked into numeric actions")
	}
}

func TestRunnerInterfaceAddressFaultUsesTypedCalls(t *testing.T) {
	target := new(recordingTarget)
	apply := behavior.InterfaceAddressAction{Device: "host", Fault: devicestate.InterfaceAddressFault{
		Interface: "eth0", Type: devicestate.FaultDuplicateIP, Address: netip.MustParseAddr("192.0.2.20"),
	}}
	reset := behavior.InterfaceAddressAction{Device: "host", Clear: true, Fault: devicestate.InterfaceAddressFault{
		Interface: "eth0", Type: devicestate.FaultDuplicateIP,
	}}
	runner := behavior.New(target, []behavior.Transition{
		{AddressActions: []behavior.InterfaceAddressAction{apply}},
		{AddressActions: []behavior.InterfaceAddressAction{reset}},
	})
	runner.Start()
	t.Cleanup(runner.Stop)
	waitForAppliedTransitions(t, runner, 2)
	target.mu.Lock()
	defer target.mu.Unlock()
	if !reflect.DeepEqual(target.addressActions, []behavior.InterfaceAddressAction{apply, reset}) ||
		len(target.actions) != 0 || len(target.deviceActions) != 0 {
		t.Fatal("runner lost typed payload or used numeric setter")
	}
}
