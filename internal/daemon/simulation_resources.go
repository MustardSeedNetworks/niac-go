package daemon

import (
	"context"
	"errors"
	"fmt"

	"github.com/MustardSeedNetworks/niac-go/internal/capture"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// Building and tearing down the resources one simulation session needs: the
// capture engine or trunk transport, the protocol stack, and the cancel that
// stops them. Split out of daemon.go, which owns the daemon's own lifecycle.

func startSimulationResources(
	iface string,
	cfg *config.Config,
	topology *fabric.Topology,
	dryRun bool,
	debugLevel int,
) (simulationResources, error) {
	if dryRun {
		stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(debugLevel))
		stack.ConfigureFabric(topology)
		_, cancel := context.WithCancel(context.Background())
		return simulationResources{stack: stack, cancel: cancel}, nil
	}

	engine, stack, cancel, err := startSimulationStack(iface, cfg, topology, debugLevel)
	if err != nil {
		return simulationResources{}, err
	}
	return simulationResources{
		engine: engine,
		stack:  stack,
		replay: newReplayController(engine, stack.GetDebugLevel()),
		cancel: cancel,
	}, nil
}

func (d *Daemon) startTrunkSimulationResources(
	iface string,
	vlan uint16,
	cfg *config.Config,
	topology *fabric.Topology,
	dryRun bool,
	replacing bool,
) (simulationResources, error) {
	if dryRun {
		stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(d.cfg.DebugLevel))
		stack.ConfigureFabric(topology)
		_, cancel := context.WithCancel(context.Background())
		return simulationResources{stack: stack, cancel: cancel}, nil
	}

	managed := d.trunks[iface]
	if managed == nil {
		engine, err := capture.New(iface, d.cfg.DebugLevel)
		if err != nil {
			return simulationResources{}, fmt.Errorf("create trunk capture engine: %w", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		managed = &managedTrunkCapture{capture: newTrunkCapture(engine), cancel: cancel}
		d.trunks[iface] = managed
		go func() {
			captureErr := managed.capture.run(ctx)
			if captureErr == nil || errors.Is(captureErr, context.Canceled) {
				return
			}
			// Every session on this interface is now deaf and mute. Record it
			// so their status says so, instead of reporting running while no
			// frame can reach them.
			logging.Errorf("Trunk capture on %s stopped: %v", iface, captureErr)
			managed.capture.fail(captureErr)
		}()
	}

	transport, previous, err := acquireTrunkTransport(managed.capture, vlan, replacing)
	if err != nil {
		return simulationResources{}, err
	}
	stack := protocols.NewStackWithTransport(
		transport,
		cfg,
		logging.NewDebugConfig(d.cfg.DebugLevel),
	)
	stack.ConfigureFabric(topology)
	if err = stack.Start(); err != nil {
		if previous != nil {
			managed.capture.restore(vlan, transport, previous)
		} else {
			managed.capture.unregister(vlan, transport)
		}
		d.closeUnusedTrunk(iface)
		return simulationResources{}, fmt.Errorf("start protocol stack: %w", err)
	}
	_, cancel := context.WithCancel(context.Background())
	return simulationResources{
		stack:  stack,
		replay: newReplayController(&trunkReplaySender{transport: transport, vlan: vlan}, stack.GetDebugLevel()),
		cancel: cancel,
		close: func() {
			managed.capture.unregister(vlan, transport)
			d.closeUnusedTrunk(iface)
		},
		rollback: func() {
			if previous != nil {
				managed.capture.restore(vlan, transport, previous)
			}
		},
	}, nil
}

func acquireTrunkTransport(
	capture *trunkCapture,
	vlan uint16,
	replacing bool,
) (*trunkSessionTransport, *trunkSessionTransport, error) {
	if replacing {
		replacement, previous := capture.replace(vlan)
		return replacement, previous, nil
	}
	transport, err := capture.register(vlan)
	return transport, nil, err
}

func (d *Daemon) closeUnusedTrunk(iface string) {
	managed := d.trunks[iface]
	if managed == nil {
		return
	}
	managed.capture.mu.RLock()
	active := len(managed.capture.sessions)
	managed.capture.mu.RUnlock()
	if active != 0 {
		return
	}
	delete(d.trunks, iface)
	managed.cancel()
	managed.capture.close()
}

func (resources simulationResources) stop() {
	if resources.replay != nil {
		_, _ = resources.replay.Stop()
	}
	if resources.cancel != nil {
		resources.cancel()
	}
	if resources.close != nil {
		resources.close()
	}
	if resources.stack != nil {
		resources.stack.Stop()
	}
	if resources.engine != nil {
		resources.engine.Close()
	}
}

func (resources simulationResources) abort() {
	if resources.rollback != nil {
		resources.rollback()
	}
	resources.stop()
}

// startSimulationStack creates the capture engine and starts the protocol stack.
// Returns (engine, stack, cancel, err). Cleans up on failure.
func startSimulationStack(
	iface string, cfg *config.Config, topology *fabric.Topology, debugLevel int,
) (*capture.Engine, *protocols.Stack, context.CancelFunc, error) {
	engine, err := capture.New(iface, debugLevel)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create capture engine: %w", err)
	}

	stack := protocols.NewStack(engine, cfg, logging.NewDebugConfig(debugLevel))
	stack.ConfigureFabric(topology)

	// Lifecycle cancel used by StopSimulation. Stack.Start() does not accept a context,
	// so the stop signal flows via Stack.Stop() and engine.Close(). The cancel is
	// retained for future context plumbing.
	_, cancel := context.WithCancel(context.Background())

	if startErr := stack.Start(); startErr != nil {
		cancel()
		engine.Close()
		return nil, nil, nil, fmt.Errorf("start protocol stack: %w", startErr)
	}

	return engine, stack, cancel, nil
}
