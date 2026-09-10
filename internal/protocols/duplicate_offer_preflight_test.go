package protocols

import (
	"errors"
	"net/netip"
	"reflect"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func duplicateOfferTimeline(device, address string) []config.BehaviorTimeline {
	return []config.BehaviorTimeline{{Name: "conflict", RepeatCount: 1, Phases: []config.BehaviorPhase{{
		Name: "offer", Duration: time.Second,
		Faults: []config.BehaviorFault{{
			Device: device, Type: string(devicestate.FaultDuplicateDHCPOffer),
			Address: netip.MustParseAddr(address),
		}},
	}}}}
}

func TestDuplicateOfferConfiguredPreflightUsesSelectedTopology(t *testing.T) {
	stack, cfg := isolationRoutedDHCP(t)
	for _, tc := range []struct {
		device, address string
		wantError       bool
	}{
		{"a", "10.10.200.3", false},
		{"remote", "10.20.0.1", true},
	} {
		cfg.BehaviorTimelines = duplicateOfferTimeline(tc.device, tc.address)
		err := ValidateConfiguredBehaviorTargets(cfg, &stack.fabric.topology)
		if tc.wantError && !errors.Is(err, ErrFaultConflictAbsent) || !tc.wantError && err != nil {
			t.Errorf("device %s: preflight error = %v, want error=%v", tc.device, err, tc.wantError)
		}
	}
}

func TestDuplicateOfferRejectedReloadPreservesRuntime(t *testing.T) {
	stack, cfg := isolationRoutedDHCP(t)
	if err := stack.SetDeviceAddressFault(
		"a",
		devicestate.FaultDuplicateDHCPOffer,
		netip.MustParseAddr("10.10.200.3"),
	); err != nil {
		t.Fatal(err)
	}
	before := stack.ExportDeviceStates()
	priorFabric, priorRunner := stack.fabric, stack.behaviorRunner
	next := *cfg
	next.BehaviorTimelines = duplicateOfferTimeline("remote", "10.20.0.1")
	if err := stack.ReloadConfig(&next); !errors.Is(err, ErrFaultConflictAbsent) {
		t.Fatalf("reload error = %v, want conflict absent", err)
	}
	if stack.config != cfg || stack.fabric != priorFabric || stack.behaviorRunner != priorRunner ||
		!reflect.DeepEqual(before, stack.ExportDeviceStates()) {
		t.Fatal("rejected reload replaced configuration, fabric, runner or durable state")
	}
}

func TestDuplicateOfferRecoveryAllowsTemporarilyUnavailablePeer(t *testing.T) {
	stack, cfg := isolationPair(t)
	cfg.BehaviorTimelines = duplicateOfferTimeline("a", "10.0.0.3")
	if err := stack.SetDeviceAddressFault(
		"a",
		devicestate.FaultDuplicateDHCPOffer,
		netip.MustParseAddr("10.0.0.3"),
	); err != nil {
		t.Fatal(err)
	}
	peer := stack.deviceStates[&cfg.Devices[1]]
	if err := peer.SetInterfaceFault(
		peer.Snapshot().Network.Interfaces[0].Name,
		devicestate.FaultLinkDown,
		1,
	); err != nil {
		t.Fatal(err)
	}
	restored := NewStack(nil, cfg, logging.NewDebugConfig(0))
	if err := restored.RestoreDeviceStates(stack.ExportDeviceStates()); err != nil {
		t.Fatal(err)
	}
	if err := restored.ValidateBehaviorTargets(); err != nil {
		t.Fatalf("recovered temporary peer outage blocks session startup: %v", err)
	}
}
