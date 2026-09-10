package daemon

import (
	"context"
	"fmt"

	"github.com/MustardSeedNetworks/niac-go/internal/capture"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

func prepareDryRunSimulation(
	cfg *config.Config, topology *fabric.Topology, debugLevel int, restore restoreRuntimeState,
) (simulationResources, error) {
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(debugLevel))
	stack.ConfigureFabric(topology)
	if err := applyRuntimeState(stack, restore); err != nil {
		return simulationResources{}, err
	}
	if err := stack.ValidateBehaviorActions(); err != nil {
		return simulationResources{}, err
	}
	_, cancel := context.WithCancel(context.Background())
	return simulationResources{stack: stack, cancel: cancel}, nil
}

// startSimulationStack creates the capture engine and starts the protocol stack.
// Returns (engine, stack, cancel, err). Cleans up on failure.
func startSimulationStack(
	iface string, cfg *config.Config, topology *fabric.Topology, debugLevel int,
	restore restoreRuntimeState,
) (*capture.Engine, *protocols.Stack, context.CancelFunc, error) {
	if err := protocols.ValidateConfiguredBehaviorActions(cfg); err != nil {
		return nil, nil, nil, err
	}
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

	if restoreErr := applyRuntimeState(stack, restore); restoreErr != nil {
		cancel()
		engine.Close()
		return nil, nil, nil, restoreErr
	}

	if startErr := stack.Start(); startErr != nil {
		cancel()
		engine.Close()
		return nil, nil, nil, fmt.Errorf("start protocol stack: %w", startErr)
	}

	return engine, stack, cancel, nil
}

// applyRuntimeState runs the recovery restore, if there is one, between the
// compiled fabric being installed and the stack starting.
func applyRuntimeState(stack *protocols.Stack, restore restoreRuntimeState) error {
	if restore == nil {
		return nil
	}
	if err := restore(stack); err != nil {
		return fmt.Errorf("restore runtime state: %w", err)
	}
	return nil
}
