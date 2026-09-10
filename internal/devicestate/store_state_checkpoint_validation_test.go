package devicestate_test

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestRestoreStateChecksEveryCheckpointFault(t *testing.T) {
	for name, mutate := range invalidCheckpointFaults() {
		t.Run(name, func(t *testing.T) {
			store := newStateTestStore(t)
			store.SaveCheckpoint("first")
			store.SaveCheckpoint("second")
			state := store.ExportState()
			mutate(&state.Checkpoints[1])
			if err := store.RestoreState(state); !errors.Is(err, devicestate.ErrStateInvalid) {
				t.Fatalf("RestoreState() = %v, want ErrStateInvalid", err)
			}
		})
	}
}

func invalidCheckpointFaults() map[string]func(*devicestate.Checkpoint) {
	return map[string]func(*devicestate.Checkpoint){
		"unknown interface": func(p *devicestate.Checkpoint) {
			p.InterfaceFaults = []devicestate.InterfaceFault{
				{Interface: "missing", Type: devicestate.FaultFCS, Value: 1},
			}
		},
		"unknown interface fault type": func(p *devicestate.Checkpoint) {
			p.InterfaceFaults = []devicestate.InterfaceFault{{Interface: "eth0", Type: "invalid", Value: 1}}
		},
		"zero interface fault": func(p *devicestate.Checkpoint) {
			p.InterfaceFaults = []devicestate.InterfaceFault{{Interface: "eth0", Type: devicestate.FaultFCS}}
		},
		"duplicate interface fault": func(p *devicestate.Checkpoint) {
			fault := devicestate.InterfaceFault{Interface: "eth0", Type: devicestate.FaultFCS, Value: 1}
			p.InterfaceFaults = []devicestate.InterfaceFault{fault, fault}
		},
		"unknown device fault type": func(p *devicestate.Checkpoint) {
			p.DeviceFaults = []devicestate.DeviceFault{{Type: "invalid", Value: 1}}
		},
		"zero device fault": func(p *devicestate.Checkpoint) {
			p.DeviceFaults = []devicestate.DeviceFault{{Type: devicestate.FaultDNSTimeout}}
		},
		"duplicate device fault": func(p *devicestate.Checkpoint) {
			fault := devicestate.DeviceFault{Type: devicestate.FaultLatency, Value: 1}
			p.DeviceFaults = []devicestate.DeviceFault{fault, fault}
		},
	}
}
