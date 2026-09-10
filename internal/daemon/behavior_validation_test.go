package daemon

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

func TestUnavailableActionRejectedBeforeCapture(t *testing.T) {
	cfg := &config.Config{
		Devices: []config.Device{
			{
				Name:       "host",
				Type:       "server",
				SNMPConfig: config.SNMPConfig{Community: "reader"},
				STPConfig:  &config.STPConfig{Enabled: true},
			},
		},
		BehaviorTimelines: []config.BehaviorTimeline{
			{
				RepeatCount: 1,
				Phases: []config.BehaviorPhase{
					{Actions: []config.BehaviorAction{{Device: "host", Type: devicestate.ActionSTPTopologyChange}}},
				},
			},
		},
	}
	if err := protocols.ValidateConfiguredBehaviorTargets(cfg, nil); !errors.Is(
		err,
		protocols.ErrDeviceActionUnobservable,
	) {
		t.Fatalf("validation=%v", err)
	}
	engine, stack, cancel, err := startSimulationStack("no-such-interface", cfg, nil, 0, nil)
	if !errors.Is(err, protocols.ErrDeviceActionUnobservable) || engine != nil || stack != nil || cancel != nil {
		t.Fatalf("capture was attempted: %v", err)
	}
	daemon := &Daemon{}
	if _, err = daemon.startTrunkSimulationResources(
		"no-such-interface",
		200,
		cfg,
		nil,
		false,
		false,
		nil,
	); !errors.Is(
		err,
		protocols.ErrDeviceActionUnobservable,
	) {
		t.Fatalf("trunk capture attempted: %v", err)
	}
}
