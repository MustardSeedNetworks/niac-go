package daemon

import (
	"context"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/supervise"

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

// runtimeWriterRestarts is how many times a faulted writer is restarted. A
// restart is safe: the writer rewrites the current state on its next tick.
const runtimeWriterRestarts = 3

// superviseRuntimeStateWriter starts the writer as a supervised worker and
// returns the group that stops it.
func superviseRuntimeStateWriter(
	sessionID string,
	stack *protocols.Stack,
	write func(string, *protocols.Stack) error,
	interval time.Duration,
) *supervise.Group {
	workers := supervise.New(nil)
	workers.Add("runtime-state-writer", supervise.RestartN(runtimeWriterRestarts), func(ctx context.Context) error {
		runRuntimeStateWriter(ctx, sessionID, stack, write, interval)
		return nil
	})
	workers.Start(context.Background())
	return workers
}

func (d *Daemon) startRuntimeStateWriter(sim *Simulation) {
	if sim == nil || sim.stack == nil || d.runtimeStateDir() == "" {
		return
	}
	write := func(sessionID string, stack *protocols.Stack) error {
		return d.writeRuntimeState(sessionID, sim.runtimeGeneration, stack)
	}
	sim.runtimeWriter = superviseRuntimeStateWriter(sim.SessionID, sim.stack, write, runtimeStateInterval)
}

// stopRuntimeStateWriter waits for an in-flight write: the settle that follows
// must be the only owner of the record.
func stopRuntimeStateWriter(sim *Simulation) {
	if sim == nil || sim.runtimeWriter == nil {
		return
	}
	_ = sim.runtimeWriter.Stop(context.Background())
	sim.runtimeWriter = nil
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
