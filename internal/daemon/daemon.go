// Package daemon provides a long-running service that can start/stop NIAC simulations dynamically
package daemon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/krisarmstrong/niac-go/internal/api"
	"github.com/krisarmstrong/niac-go/internal/capture"
	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/logging"
	"github.com/krisarmstrong/niac-go/internal/protocols"
	"github.com/krisarmstrong/niac-go/internal/replay"
	"github.com/krisarmstrong/niac-go/internal/storage"
)

// Sentinel errors for daemon package.
var (
	ErrInterfaceNotExist        = errors.New("interface does not exist")
	ErrConfigDataExceedsMaxSize = errors.New("config data exceeds maximum size")
	ErrConfigPathOrDataRequired = errors.New("either config_path, config_data, or template_name must be provided")
	ErrNoSimulationRunning      = errors.New("no simulation running")
	ErrTemplateNotFound         = errors.New("template not found")
)

const (
	// DefaultDebugLevel is the default debug level for the capture
	// engine + protocol stack. Set to 1 (DebugLevelBasic) so the
	// per-protocol "Starting periodic advertisements" lines and
	// neighbor-discovery info messages reach the Debug Console out
	// of the box. Without this the SSE log tee (PR #574) is wired
	// correctly but operators see an empty Debug Console because
	// every ProtocolLogf call gates on debugLevel >= 1. Per-packet
	// verbose logging still requires raising the level explicitly.
	DefaultDebugLevel = 1
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
	// WebhookAllowedHosts restricts the alert webhook destination to an
	// admin-managed set of hostnames. Empty = no allowlist (the existing
	// private-IP / blocked-hostname filters still apply).
	WebhookAllowedHosts []string
	// EnableTLS controls whether the daemon's API listener serves HTTPS.
	// Wave 1 default: true.
	EnableTLS bool
	// CertDir is the directory holding the self-signed cert+key when
	// CertFile/KeyFile are not explicitly set.
	CertDir string
	// CertFile / KeyFile are explicit PEM paths; empty triggers
	// auto-generation under CertDir.
	CertFile string
	KeyFile  string
	// HTTPRedirectAddr is the address of the small HTTP listener that
	// 308-redirects to the HTTPS service when TLS is on. Empty disables.
	HTTPRedirectAddr string
	// TLSPort is the destination HTTPS port the redirect handler
	// advertises. Zero falls back to api.defaultTLSPort.
	TLSPort int
}

// Daemon manages the NIAC simulation lifecycle.
type Daemon struct {
	cfg       Config
	apiServer *api.Server
	storage   *storage.Storage

	mu         sync.RWMutex
	simulation *Simulation
	// capture is the optional standalone packet-capture session that
	// runs without a simulation. Mutually exclusive with simulation
	// because both want exclusive ownership of the same libpcap
	// interface handle; see internal/daemon/capture.go.
	capture *standaloneCapture
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
		// SECURITY: Check for path traversal in the INPUT before filepath.Clean
		// strips it. Cleaning first would let "/etc/../etc/passwd" slip through.
		if strings.Contains(cfg.StoragePath, "..") {
			return nil, fmt.Errorf("storage path must not contain '..' components: %s", cfg.StoragePath)
		}

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
		Addr:                d.cfg.ListenAddr,
		Token:               d.cfg.Token,
		Version:             d.cfg.Version,
		Commit:              d.cfg.Commit,
		BuildTime:           d.cfg.BuildTime,
		UIBuildHash:         d.cfg.UIBuildHash,
		Storage:             d.storage,
		WebhookAllowedHosts: d.cfg.WebhookAllowedHosts,
		EnableTLS:           d.cfg.EnableTLS,
		CertDir:             d.cfg.CertDir,
		CertFile:            d.cfg.CertFile,
		KeyFile:             d.cfg.KeyFile,
		HTTPRedirectAddr:    d.cfg.HTTPRedirectAddr,
		TLSPort:             d.cfg.TLSPort,
		// Stack, Config, etc. will be nil until simulation starts
	}

	d.apiServer = api.NewServer(serverCfg)

	// Set daemon controller on the API server
	// This allows the API to call our Start/Stop/Status methods
	d.apiServer.SetDaemonController(d)
	d.apiServer.SetCaptureController(d)

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

	// Stop standalone capture if running. StopCapture is idempotent so
	// the no-capture case is a fast nil return.
	if captureErr := d.StopCapture(); captureErr != nil {
		logging.Errorf("Error stopping standalone capture: %v", captureErr)
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

// maxSimulationConfigSize limits the size of inline-posted simulation configs.
const maxSimulationConfigSize = 10 * 1024 * 1024 // 10MB limit

// loadSimulationConfig resolves either inline ConfigData or a cleaned ConfigPath into a Config.
//
// Inline data is written to a deterministic file under the user-configs
// directory so the rest of the daemon — GET /api/v1/config, the running-
// config YAML editor, "Download YAML" — has a real path to read from.
// Without this, those surfaces returned config_read_failed because they
// did a file Stat on the literal string "<inline>".
func loadSimulationConfig(req api.SimulationRequest) (*config.Config, string, error) {
	switch {
	case req.TemplateName != "":
		// Loading templates by name preserves the template's own
		// directory as the include_path base — needed for vendor
		// templates with `include_path: ".."` plus relative walk_file
		// refs that resolve to sibling directories (e.g. examples/
		// device_walks_sanitized/...). Fetching the YAML text and
		// POSTing it as ConfigData would lose that context.
		templatePath := api.FindTemplateOnDisk(req.TemplateName)
		if templatePath == "" {
			return nil, "", fmt.Errorf("%w: %s", ErrTemplateNotFound, req.TemplateName)
		}
		cfg, err := config.Load(templatePath)
		if err != nil {
			return nil, "", fmt.Errorf("load template %q: %w", req.TemplateName, err)
		}
		return cfg, templatePath, nil
	case req.ConfigData != "":
		if len(req.ConfigData) > maxSimulationConfigSize {
			return nil, "", fmt.Errorf(
				"%w: %d bytes (got %d bytes)",
				ErrConfigDataExceedsMaxSize,
				maxSimulationConfigSize,
				len(req.ConfigData),
			)
		}
		cfg, err := config.LoadYAMLBytes([]byte(req.ConfigData))
		if err != nil {
			return nil, "", fmt.Errorf("load configuration: %w", err)
		}
		path, err := persistInlineConfig(req.ConfigData)
		if err != nil {
			// Persistence failure isn't fatal — the sim can still run on
			// the parsed Config — but the downstream "view running YAML"
			// flows will fail. Log via the returned error chain so the
			// API surface sees it.
			return cfg, "", fmt.Errorf("persist inline config: %w", err)
		}
		return cfg, path, nil
	case req.ConfigPath != "":
		cleanedPath := filepath.Clean(req.ConfigPath)
		if strings.Contains(cleanedPath, "..") {
			return nil, "", fmt.Errorf("config path must not contain '..' components: %s", req.ConfigPath)
		}
		cfg, err := config.Load(cleanedPath)
		if err != nil {
			return nil, "", fmt.Errorf("load configuration: %w", err)
		}
		return cfg, cleanedPath, nil
	default:
		return nil, "", ErrConfigPathOrDataRequired
	}
}

// inlineConfigName is the deterministic filename used to materialise inline
// (uploaded / template-derived) config data on disk. Overwritten on every
// start so the daemon doesn't accumulate stale files; the simulation-stop
// path leaves it in place so the user can still download the YAML after.
const inlineConfigName = "_running.inline.yaml"

// persistInlineConfig writes the inline YAML to disk so the rest of the
// daemon has a real configPath to operate on. Returns the absolute path
// it was written to.
func persistInlineConfig(content string) (string, error) {
	rawDir := os.Getenv("NIAC_CONFIGS_DIR")
	if rawDir == "" {
		// Prefer $HOME/.niac/configs over CWD so daemon restarts find it.
		if home, err := os.UserHomeDir(); err == nil {
			rawDir = filepath.Join(home, ".niac", "configs")
		} else {
			rawDir = "configs"
		}
	}
	// Clean dir + reject any traversal so a hostile NIAC_CONFIGS_DIR
	// like "../../etc" can't escape its intended scope. The path is
	// always operator-controlled (daemon env) but the explicit barrier
	// lets static analysers see the check.
	cleanDir := filepath.Clean(rawDir)
	if strings.Contains(cleanDir, "..") {
		return "", fmt.Errorf("configs dir must not contain '..' components: %s", rawDir)
	}
	if err := os.MkdirAll(cleanDir, 0o750); err != nil {
		return "", fmt.Errorf("create configs dir: %w", err)
	}
	path := filepath.Join(cleanDir, inlineConfigName)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", fmt.Errorf("write inline config: %w", err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path, nil //nolint:nilerr // best-effort abs path; relative is fine too
	}
	return abs, nil
}

// StartSimulation starts a new simulation.
func (d *Daemon) StartSimulation(req api.SimulationRequest) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.simulation != nil {
		if err := d.stopSimulationLocked(); err != nil {
			return fmt.Errorf("stop existing simulation: %w", err)
		}
	}

	if !capture.InterfaceExists(req.Interface) {
		return fmt.Errorf("%w: %s", ErrInterfaceNotExist, req.Interface)
	}

	cfg, configPath, err := loadSimulationConfig(req)
	if err != nil {
		return err
	}

	engine, stack, cancel, err := startSimulationStack(req.Interface, cfg)
	if err != nil {
		return err
	}

	replay := newReplayController(engine, stack.GetDebugLevel())

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

	d.apiServer.UpdateSimulation(stack, cfg, configPath, req.Interface, replay)

	logging.Successf("✓ Simulation started on %s with %d devices", req.Interface, len(cfg.Devices))

	// Auto-start PCAP playback if configured. The playback is best-effort —
	// a failed playback start logs but doesn't fail the simulation, since
	// the protocol stack is already up and serving devices.
	if cfg.CapturePlayback != nil && strings.TrimSpace(cfg.CapturePlayback.FileName) != "" {
		fileName := resolvePlaybackPath(cfg.CapturePlayback.FileName, configPath)
		_, replayErr := replay.Start(api.ReplayRequest{
			File:   fileName,
			LoopMs: cfg.CapturePlayback.LoopTime,
			Scale:  cfg.CapturePlayback.ScaleTime,
		})
		if replayErr != nil {
			logging.Warningf("PCAP playback auto-start failed: %v", replayErr)
		} else {
			logging.Successf("✓ PCAP playback auto-started: %s", fileName)
		}
	}

	return nil
}

// resolvePlaybackPath turns a (possibly relative) capture_playbacks file_name
// into an absolute path. Relative paths resolve against the directory holding
// the config file — same convention as the legacy CLI's include_path handling.
func resolvePlaybackPath(fileName, configPath string) string {
	if filepath.IsAbs(fileName) {
		return fileName
	}
	if configPath == "" {
		return fileName
	}
	return filepath.Join(filepath.Dir(configPath), fileName)
}

// startSimulationStack creates the capture engine and starts the protocol stack.
// Returns (engine, stack, cancel, err). Cleans up on failure.
func startSimulationStack(
	iface string, cfg *config.Config,
) (*capture.Engine, *protocols.Stack, context.CancelFunc, error) {
	engine, err := capture.New(iface, DefaultDebugLevel)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create capture engine: %w", err)
	}

	stack := protocols.NewStack(engine, cfg, logging.NewDebugConfig(DefaultDebugLevel))

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

// newReplayController returns a replay controller. The implementation lives
// in internal/replay so the daemon and the legacy CLI's runtime services
// share one canonical copy — until #494 this was a duplicated stub here and
// a working version in cmd/niac/runtime_services.go.
func newReplayController(engine *capture.Engine, debugLevel int) *replay.Controller {
	return replay.New(engine, debugLevel)
}
