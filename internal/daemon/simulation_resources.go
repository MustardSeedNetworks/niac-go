package daemon

import (
	"context"
	"errors"
	"fmt"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/capture"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// Building and tearing down the resources one simulation session needs: the
// capture engine or trunk transport, the protocol stack, and the cancel that
// stops them. Split out of daemon.go, which owns the daemon's own lifecycle.

func (d *Daemon) startResourcesForRequest(
	req api.SimulationRequest,
	cfg *config.Config,
	compiled *fabric.Topology,
	dryRun bool,
	replacing bool,
	restore restoreRuntimeState,
) (simulationResources, error) {
	// Access-mode sessions join an existing shared capture in the native
	// slot; a competing direct capture would receive the same frames twice.
	if req.AttachmentMode == fabric.ModeTrunk {
		return d.startTrunkSimulationResources(
			req.Interface, req.AccessVLAN, cfg, compiled, dryRun, replacing, restore,
		)
	}
	if req.AttachmentMode == fabric.ModeAccess && !dryRun && d.trunks[req.Interface] != nil {
		return d.startTrunkSimulationResources(
			req.Interface, nativeVLANKey, cfg, compiled, dryRun, replacing, restore,
		)
	}
	return d.startSimulation(req.Interface, cfg, compiled, dryRun, d.cfg.DebugLevel, restore)
}

func startSimulationResources(
	iface string,
	cfg *config.Config,
	topology *fabric.Topology,
	dryRun bool,
	debugLevel int,
	restore restoreRuntimeState,
) (simulationResources, error) {
	if dryRun {
		return prepareDryRunSimulation(cfg, topology, debugLevel, restore)
	}

	engine, stack, cancel, err := startSimulationStack(iface, cfg, topology, debugLevel, restore)
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
	restore restoreRuntimeState,
) (simulationResources, error) {
	if dryRun {
		return prepareDryRunSimulation(cfg, topology, d.cfg.DebugLevel, restore)
	}

	if err := protocols.ValidateConfiguredBehaviorTargets(cfg, topology); err != nil {
		return simulationResources{}, err
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
	releaseTransport := func() {
		if previous != nil {
			managed.capture.restore(vlan, transport, previous)
		} else {
			managed.capture.unregister(vlan, transport)
		}
		d.closeUnusedTrunk(iface)
	}
	stack.ConfigureFabric(topology)
	if err = applyRuntimeState(stack, restore); err != nil {
		releaseTransport()
		return simulationResources{}, err
	}
	if err = stack.Start(); err != nil {
		releaseTransport()
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
