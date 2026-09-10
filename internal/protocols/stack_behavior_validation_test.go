package protocols

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestBehaviorValidationRejectsMissingScalarBeforeStart(t *testing.T) {
	device := faultTestDevice("edge")
	device.Type = "server"
	device.STPConfig = &config.STPConfig{Enabled: true}
	device.SNMPConfig.WalkFile = actionWalkWithoutSTP(t)
	cfg := &config.Config{
		Devices: []config.Device{device},
		BehaviorTimelines: []config.BehaviorTimeline{
			{
				RepeatCount: 1,
				Phases: []config.BehaviorPhase{
					{
						Actions: []config.BehaviorAction{
							{Device: device.Name, Type: devicestate.ActionSTPTopologyChange},
						},
					},
				},
			},
		},
	}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	if err := stack.ValidateBehaviorActions(); err == nil {
		t.Fatal("missing STP MIB accepted")
	}
	if err := stack.Start(); err == nil {
		stack.Stop()
		t.Fatal("invalid timeline started")
	}
	if stack.started || stack.running.Load() {
		t.Fatal("validation ran after startup")
	}
	assertConsumedActionCount(t, stack, false)
}

func TestInvalidActionReloadPreservesConfigurationAndState(t *testing.T) {
	previous := &config.Config{Devices: []config.Device{faultTestDevice("healthy")}}
	stack := NewStack(nil, previous, logging.NewDebugConfig(0))
	defer stack.Stop()
	saved := stack.ExportDeviceStates()
	device := faultTestDevice("unserved")
	device.Type = "server"
	device.STPConfig = &config.STPConfig{Enabled: true}
	device.SNMPConfig.WalkFile = actionWalkWithoutSTP(t)
	replacement := &config.Config{
		Devices: []config.Device{device},
		BehaviorTimelines: []config.BehaviorTimeline{{RepeatCount: 1, Phases: []config.BehaviorPhase{{
			Actions: []config.BehaviorAction{{Device: "unserved", Type: devicestate.ActionSTPTopologyChange}},
		}}}},
	}
	if err := stack.ReloadConfig(replacement); !errors.Is(err, ErrDeviceActionUnobservable) {
		t.Fatalf("missing captured STP inventory accepted: %v", err)
	}
	if stack.config != previous || !reflect.DeepEqual(saved, stack.ExportDeviceStates()) {
		t.Fatal("invalid reload changed existing configuration or state")
	}
}

func actionWalkWithoutSTP(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "server.walk")
	if err := os.WriteFile(path, []byte(".1.3.6.1.2.1.1.1.0 = STRING: \"Captured server\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
