package devicestate_test

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestRestoreStateRejectsMalformedBanksWithoutMutation(t *testing.T) {
	for name, mutate := range invalidStateBanks() {
		t.Run(name, func(t *testing.T) {
			store := newStateTestStore(t)
			store.SaveCheckpoint("baseline")
			before := store.ExportState()
			state := store.ExportState()
			mutate(&state)
			if err := store.RestoreState(state); !errors.Is(err, devicestate.ErrStateInvalid) {
				t.Fatalf("RestoreState() = %v, want ErrStateInvalid", err)
			}
			if !reflect.DeepEqual(before, store.ExportState()) {
				t.Fatal("invalid record changed the store")
			}
		})
	}
}

func invalidStateBanks() map[string]func(*devicestate.State) {
	return map[string]func(*devicestate.State){
		"missing running interface": func(s *devicestate.State) { s.Running.Network.Interfaces = nil },
		"renamed startup interface": func(s *devicestate.State) { s.Startup.Network.Interfaces[0].Name = "eth1" },
		"duplicate running interface": func(s *devicestate.State) {
			s.Running.Network.Interfaces = append(s.Running.Network.Interfaces, s.Running.Network.Interfaces[0])
		},
		"checkpoint renamed interface": func(s *devicestate.State) {
			s.Checkpoints[0].Configuration.Network.Interfaces[0].Name = "eth1"
		},
		"duplicate checkpoint": func(s *devicestate.State) { s.Checkpoints = append(s.Checkpoints, s.Checkpoints[0]) },
		"unknown fault interface": func(s *devicestate.State) {
			s.InterfaceFaults = []devicestate.InterfaceFault{{Interface: "eth1", Type: devicestate.FaultFCS, Value: 1}}
		},
		"duplicate interface fault": func(s *devicestate.State) {
			fault := devicestate.InterfaceFault{Interface: "eth0", Type: devicestate.FaultFCS, Value: 1}
			s.InterfaceFaults = []devicestate.InterfaceFault{fault, fault}
		},
		"duplicate device fault": func(s *devicestate.State) {
			fault := devicestate.DeviceFault{Type: devicestate.FaultLatency, Value: 1}
			s.DeviceFaults = []devicestate.DeviceFault{fault, fault}
		},
		"checkpoint bad interface fault": func(s *devicestate.State) {
			s.Checkpoints[0].InterfaceFaults = []devicestate.InterfaceFault{{
				Interface: "eth0", Type: devicestate.FaultFCS, Value: 101,
			}}
		},
		"checkpoint bad device fault": func(s *devicestate.State) {
			s.Checkpoints[0].DeviceFaults = []devicestate.DeviceFault{{Type: devicestate.FaultLatency, Value: 60001}}
		},
	}
}

func TestRestoreStateRejectsMalformedHistory(t *testing.T) {
	for name, mutate := range invalidStateHistory() {
		t.Run(name, func(t *testing.T) {
			store := newStateTestStore(t)
			store.SaveStartup()
			state := store.ExportState()
			mutate(&state)
			if err := store.RestoreState(state); !errors.Is(err, devicestate.ErrStateInvalid) {
				t.Fatalf("RestoreState() = %v, want ErrStateInvalid", err)
			}
		})
	}
}

func invalidStateHistory() map[string]func(*devicestate.State) {
	return map[string]func(*devicestate.State){
		"exhausted version":  func(s *devicestate.State) { s.Version = math.MaxUint64 },
		"zero event version": func(s *devicestate.State) { s.Events[0].Version = 0 },
		"future event":       func(s *devicestate.State) { s.Events[0].Version = s.Version + 1 },
		"unordered history":  func(s *devicestate.State) { s.Events[0], s.Events[1] = s.Events[1], s.Events[0] },
		"oversized history": func(s *devicestate.State) {
			for len(s.Events) <= 1024 {
				s.Events = append(s.Events, s.Events[len(s.Events)-1])
			}
		},
	}
}
