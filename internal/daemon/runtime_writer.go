package daemon

import (
	"context"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// Persistence failures leave replay running; report the durability problem
// without turning a full disk into a network outage.
func runRuntimeStateWriter(
	ctx context.Context,
	sessionID string,
	stack *protocols.Stack,
	write func(string, *protocols.Stack) error,
	interval time.Duration,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var written uint64
	var reported bool
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			version := stack.RuntimeStateVersion()
			if version == written {
				continue
			}
			if err := write(sessionID, stack); err != nil {
				if !reported {
					logging.Warningf("Runtime state for session %s is not being persisted: %v", sessionID, err)
					reported = true
				}
				continue
			}
			written = version
			reported = false
		}
	}
}

func (d *Daemon) startRuntimeStateWriter(sim *Simulation) {
	if sim == nil || sim.stack == nil || d.runtimeStateDir() == "" {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	sim.runtimeCancel = cancel
	sim.runtimeDone = done
	go func() {
		defer close(done)
		write := func(sessionID string, stack *protocols.Stack) error {
			return d.writeRuntimeState(sessionID, sim.runtimeGeneration, stack)
		}
		runRuntimeStateWriter(ctx, sim.SessionID, sim.stack, write, runtimeStateInterval)
	}()
}

func stopRuntimeStateWriter(sim *Simulation) {
	if sim == nil || sim.runtimeCancel == nil {
		return
	}
	sim.runtimeCancel()
	if sim.runtimeDone != nil {
		<-sim.runtimeDone
	}
	sim.runtimeCancel = nil
	sim.runtimeDone = nil
}

// settleRuntimeState runs after the writer and protocol producers have stopped.
func (d *Daemon) settleRuntimeState(sim *Simulation, clearIntent bool) {
	if sim == nil {
		return
	}
	stopRuntimeStateWriter(sim)
	if clearIntent {
		d.clearRuntimeState(sim.SessionID, sim.runtimeGeneration)
		return
	}
	if writeErr := d.writeRuntimeState(sim.SessionID, sim.runtimeGeneration, sim.stack); writeErr != nil {
		logging.Warningf("Runtime state for session %s was not persisted at shutdown: %v", sim.SessionID, writeErr)
	}
}
