// Package daemon provides a long-running service that can start/stop NIAC simulations dynamically
package daemon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/krisarmstrong/niac-go/internal/api"
	"github.com/krisarmstrong/niac-go/internal/capture"
	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/logging"
	"github.com/krisarmstrong/niac-go/internal/protocols"
	"github.com/krisarmstrong/niac-go/internal/storage"
)

// Sentinel errors for daemon package.
var (
	ErrInterfaceNotExist        = errors.New("interface does not exist")
	ErrConfigDataExceedsMaxSize = errors.New("config data exceeds maximum size")
	ErrConfigPathOrDataRequired = errors.New("either config_path or config_data must be provided")
	ErrNoSimulationRunning      = errors.New("no simulation running")
	ErrReplayNotImplemented     = errors.New("replay not yet implemented in daemon mode")
)

const (
	// DefaultDebugLevel is the default debug level for capture engine.
	DefaultDebugLevel = 0
)

// Config holds daemon configuration.
type Config struct {
	ListenAddr  string
	Token       string
	StoragePath string
	Version     string
	Commit      string
	BuildTime   string
	UIBuildHash string
}

// Daemon manages the NIAC simulation lifecycle.
type Daemon struct {
	cfg       Config
	apiServer *api.Server
	storage   *storage.Storage

	mu         sync.RWMutex
	simulation *Simulation
}

// Simulation represents a running NIAC simulation.
type Simulation struct {
	Interface  string
	ConfigPath string
	ConfigName string
	StartedAt  time.Time

	engine *capture.Engine
	stack  *protocols.Stack
	cfg    *config.Config
	replay api.ReplayManager
	cancel context.CancelFunc
}

// NewDaemon creates a new daemon instance.
func NewDaemon(cfg Config) (*Daemon, error) {
	daemon := &Daemon{
		cfg: cfg,
	}

	// Open storage if enabled
	if cfg.StoragePath != "" && cfg.StoragePath != "disabled" {
		storagePath := expandPath(cfg.StoragePath)

		var err error

		daemon.storage, err = storage.Open(storagePath)
		if err != nil {
			return nil, fmt.Errorf("open storage: %w", err)
		}
	}

	return daemon, nil
}

// Start starts the daemon's API server.
func (d *Daemon) Start() error {
	// Create API server
	serverCfg := api.ServerConfig{
		Addr:        d.cfg.ListenAddr,
		Token:       d.cfg.Token,
		Version:     d.cfg.Version,
		Commit:      d.cfg.Commit,
		BuildTime:   d.cfg.BuildTime,
		UIBuildHash: d.cfg.UIBuildHash,
		Storage:     d.storage,
		// Stack, Config, etc. will be nil until simulation starts
	}

	d.apiServer = api.NewServer(serverCfg)

	// Set daemon controller on the API server
	// This allows the API to call our Start/Stop/Status methods
	d.apiServer.SetDaemonController(d)

	err := d.apiServer.Start()
	if err != nil {
		if d.storage != nil {
			closeErr := d.storage.Close()
			if closeErr != nil {
				logging.Errorf("Error closing storage during cleanup: %v", closeErr)
			}
		}

		return fmt.Errorf("start API server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the daemon.
func (d *Daemon) Shutdown(ctx context.Context) error {
	// Stop simulation if running
	stopErr := d.StopSimulation()
	if stopErr != nil {
		logging.Errorf("Error stopping simulation: %v", stopErr)
	}

	// Shutdown API server
	if d.apiServer != nil {
		shutdownErr := d.apiServer.Shutdown(ctx)
		if shutdownErr != nil {
			return fmt.Errorf("shutdown API server: %w", shutdownErr)
		}
	}

	// Close storage
	if d.storage != nil {
		closeErr := d.storage.Close()
		if closeErr != nil {
			logging.Errorf("Error closing storage: %v", closeErr)
		}
	}

	return nil
}

// StartSimulation starts a new simulation.
func (d *Daemon) StartSimulation(req api.SimulationRequest) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Stop existing simulation if running
	if d.simulation != nil {
		err := d.stopSimulationLocked()
		if err != nil {
			return fmt.Errorf("stop existing simulation: %w", err)
		}
	}

	// Validate interface
	if !capture.InterfaceExists(req.Interface) {
		return fmt.Errorf("%w: %s", ErrInterfaceNotExist, req.Interface)
	}

	// Load configuration
	var (
		cfg        *config.Config
		configPath string
		err        error
	)

	// SECURITY FIX #2.8.1: Validate config data size to prevent memory exhaustion
	const maxConfigSize = 10 * 1024 * 1024 // 10MB limit

	switch {
	case req.ConfigData != "":
		if len(req.ConfigData) > maxConfigSize {
			return fmt.Errorf(
				"%w: %d bytes (got %d bytes)",
				ErrConfigDataExceedsMaxSize,
				maxConfigSize,
				len(req.ConfigData),
			)
		}

		// Parse inline YAML
		cfg, err = config.LoadYAMLBytes([]byte(req.ConfigData))
		configPath = "<inline>"
	case req.ConfigPath != "":
		// Load from file
		cfg, err = config.Load(req.ConfigPath)
		configPath = req.ConfigPath
	default:
		return ErrConfigPathOrDataRequired
	}

	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	// Config is already validated during Load/LoadYAMLBytes

	// Create capture engine
	engine, err := capture.New(req.Interface, DefaultDebugLevel)
	if err != nil {
		return fmt.Errorf("create capture engine: %w", err)
	}

	// Create protocol stack with nil debug config (uses defaults)
	stack := protocols.NewStack(engine, cfg, nil)

	// Create context for lifecycle management
	ctx, cancel := context.WithCancel(context.Background())
	_ = ctx // context reserved for future use

	// Start protocol stack
	startErr := stack.Start()
	if startErr != nil {
		cancel()
		engine.Close()

		return fmt.Errorf("start protocol stack: %w", startErr)
	}

	// Create replay manager
	replay := newReplayController(engine, stack.GetDebugLevel())

	// Create simulation
	d.simulation = &Simulation{
		Interface:  req.Interface,
		ConfigPath: configPath,
		ConfigName: filepath.Base(configPath),
		StartedAt:  time.Now(),
		engine:     engine,
		stack:      stack,
		cfg:        cfg,
		replay:     replay,
		cancel:     cancel,
	}

	// Update API server with simulation components
	d.apiServer.UpdateSimulation(stack, cfg, configPath, req.Interface, replay)

	logging.Successf("✓ Simulation started on %s with %d devices", req.Interface, len(cfg.Devices))

	return nil
}

// StopSimulation stops the current simulation.
func (d *Daemon) StopSimulation() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.stopSimulationLocked()
}

func (d *Daemon) stopSimulationLocked() error {
	if d.simulation == nil {
		return ErrNoSimulationRunning
	}

	sim := d.simulation

	// Stop replay if running
	if sim.replay != nil {
		_, _ = sim.replay.Stop()
	}

	// Cancel context first to signal shutdown
	if sim.cancel != nil {
		sim.cancel()
	}

	// Stop stack
	if sim.stack != nil {
		sim.stack.Stop()
	}

	// Close engine
	if sim.engine != nil {
		sim.engine.Close()
	}

	// Save run history
	if d.storage != nil && sim.stack != nil {
		stats := sim.stack.GetStats()
		record := storage.RunRecord{
			StartedAt:       sim.StartedAt,
			Duration:        time.Since(sim.StartedAt),
			Interface:       sim.Interface,
			ConfigName:      sim.ConfigName,
			DeviceCount:     len(sim.cfg.Devices),
			PacketsSent:     stats.PacketsSent,
			PacketsReceived: stats.PacketsReceived,
			Errors:          stats.Errors,
		}
		_ = d.storage.AddRun(record)
	}

	d.simulation = nil

	// Clear simulation from API server
	d.apiServer.ClearSimulation()

	logging.Infof("Simulation stopped")

	return nil
}

// GetStatus returns the current simulation status.
func (d *Daemon) GetStatus() api.SimulationStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()

	status := api.SimulationStatus{
		Running: d.simulation != nil,
	}

	if d.simulation != nil {
		status.Interface = d.simulation.Interface
		status.ConfigPath = d.simulation.ConfigPath
		status.ConfigName = d.simulation.ConfigName
		status.StartedAt = d.simulation.StartedAt

		status.UptimeSeconds = time.Since(d.simulation.StartedAt).Seconds()
		if d.simulation.cfg != nil {
			status.DeviceCount = len(d.simulation.cfg.Devices)
		}
	}

	return status
}

func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[1:])
		}
	}
	// Clean the path to remove any .. or . elements
	return filepath.Clean(path)
}

// newReplayController is copied from runtime_services.go for now
// TODO: Refactor to share code.
type replayController struct {
	engine     *capture.Engine
	debugLevel int
	mu         sync.Mutex
	state      api.ReplayState
}

func newReplayController(engine *capture.Engine, debugLevel int) *replayController {
	return &replayController{
		engine:     engine,
		debugLevel: debugLevel,
	}
}

// Status returns the current replay state.
func (rc *replayController) Status() api.ReplayState {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	return rc.state
}

// Start begins PCAP replay with the given request.
func (rc *replayController) Start(_ api.ReplayRequest) (api.ReplayState, error) {
	// Implementation same as in runtime_services.go
	// Simplified for now
	return rc.state, ErrReplayNotImplemented
}

// Stop halts the current PCAP replay.
func (rc *replayController) Stop() (api.ReplayState, error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.state.Running = false

	return rc.state, nil
}
