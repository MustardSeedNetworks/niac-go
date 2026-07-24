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

	"github.com/gopacket/gopacket"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/api/templates"
	"github.com/MustardSeedNetworks/niac-go/internal/api/tokenstore"
	"github.com/MustardSeedNetworks/niac-go/internal/capture"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
	"github.com/MustardSeedNetworks/niac-go/internal/replay"
	"github.com/MustardSeedNetworks/niac-go/internal/storage"
)

// Sentinel errors for daemon package.
var (
	ErrInterfaceNotExist        = errors.New("interface does not exist")
	ErrConfigDataExceedsMaxSize = errors.New("config data exceeds maximum size")
	ErrConfigPathOrDataRequired = errors.New("either config_path, config_data, or template_name must be provided")
	ErrNoSimulationRunning      = errors.New("no simulation running")
	ErrTemplateNotFound         = errors.New("template not found")
	ErrUnsafeTopology           = errors.New("routed topology failed preflight")
	ErrInvalidSimulationConfig  = errors.New("simulation configuration failed semantic validation")
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
	e2eDryRunEnv      = "NIAC_E2E_DRY_RUN_SIMULATION"
)

// Config holds daemon configuration.
type Config struct {
	ListenAddr string
	// Token is the primary single bearer token read from NIAC_API_TOKEN.
	// Forwarded to api.ServerConfig.Token.
	Token string
	// TokenFile is the Wave 2 multi-token JSON file path. Forwarded to
	// api.ServerConfig.TokenFile. When non-empty, the daemon's SIGHUP
	// handler re-reads this path on each signal.
	TokenFile    string
	StoragePath  string
	RecoveryPath string
	Version      string
	Commit       string
	BuildTime    string
	ReleaseTrain string
	UIBuildHash  string
	// WebhookAllowedHosts restricts the alert webhook destination to an
	// admin-managed set of hostnames. Empty = no allowlist (the existing
	// private-IP / blocked-hostname filters still apply).
	WebhookAllowedHosts []string
	// CertDir is the directory holding the self-signed cert+key when
	// CertFile/KeyFile are not explicitly set.
	CertDir string
	// CertFile / KeyFile are explicit PEM paths; empty triggers
	// auto-generation under CertDir.
	CertFile string
	KeyFile  string
	// AttachmentPolicies are operator-owned permissions for routed physical bindings.
	AttachmentPolicies []fabric.PhysicalAttachmentPolicy
}

// Daemon manages the NIAC simulation lifecycle.
type Daemon struct {
	cfg       Config
	apiServer *api.Server
	storage   *storage.Storage

	mu         sync.RWMutex
	simulation *Simulation
	recovery   *api.SimulationRecovery
	// startSimulation is injectable so replacement failure and cleanup are deterministic in tests.
	startSimulation simulationStarter
	// capture is the optional standalone packet-capture session that
	// runs without a simulation. Mutually exclusive with simulation
	// because both want exclusive ownership of the same libpcap
	// interface handle; see internal/daemon/capture.go.
	capture                *standaloneCapture
	captureLastError       string
	captureInterfaceExists func(string) bool
	newCaptureEngine       func(string, int) (captureEngine, error)
	captureRunner          func(context.Context, captureEngine, func(gopacket.Packet)) error
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
	fabric *fabric.Topology
	replay api.ReplayManager
	cancel context.CancelFunc
}

type simulationResources struct {
	engine *capture.Engine
	stack  *protocols.Stack
	replay api.ReplayManager
	cancel context.CancelFunc
}

type simulationStarter func(
	string,
	*config.Config,
	*fabric.Topology,
	bool,
) (simulationResources, error)

// NewDaemon creates a new daemon instance.
func NewDaemon(cfg Config) (*Daemon, error) {
	daemon := &Daemon{
		cfg:                    cfg,
		startSimulation:        startSimulationResources,
		captureInterfaceExists: capture.InterfaceExists,
		newCaptureEngine: func(name string, debugLevel int) (captureEngine, error) {
			return capture.New(name, debugLevel)
		},
		captureRunner: func(
			ctx context.Context,
			engine captureEngine,
			handler func(gopacket.Packet),
		) error {
			return engine.StartCaptureContext(ctx, handler)
		},
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
		Addr:                           d.cfg.ListenAddr,
		Token:                          d.cfg.Token,
		TokenFile:                      d.cfg.TokenFile,
		Version:                        d.cfg.Version,
		Commit:                         d.cfg.Commit,
		BuildTime:                      d.cfg.BuildTime,
		ReleaseTrain:                   d.cfg.ReleaseTrain,
		UIBuildHash:                    d.cfg.UIBuildHash,
		Storage:                        d.storage,
		WebhookAllowedHosts:            d.cfg.WebhookAllowedHosts,
		CertDir:                        d.cfg.CertDir,
		CertFile:                       d.cfg.CertFile,
		KeyFile:                        d.cfg.KeyFile,
		SuppressUnauthenticatedWarning: e2eDryRunSimulation(),
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

	d.recoverActiveSimulation(d.apiServer.SimulationEntitlements())

	return nil
}

// ReloadTokens re-reads the configured token source (file or env) and
// publishes the new token set to the API server. Returns the rotated
// token count and any error encountered while loading. On error the
// previously-active tokens stay in effect — the caller (typically the
// SIGHUP handler in cmd/niac) should log and keep serving.
//
// Sources, in precedence order — same as initialTokenStore:
//  1. TokenFile non-empty: re-read the JSON file.
//  2. TokenFile empty, Token non-empty: republish the single token as
//     ScopeReadWrite. This is the back-compat path: an operator can
//     `export NIAC_API_TOKEN=<new>; kill -HUP <pid>` and the daemon
//     picks up the new value without restarting. cfg.Token is
//     re-read from the daemon's env on every call so the rotation
//     actually picks up the new value.
//  3. Neither set: clear the store. The non-loopback gate is a
//     startup-time check, not a per-request check, so this does not
//     change the listen address; new requests simply fall through to
//     the unauthed branch (matching Wave 1's loopback default).
func (d *Daemon) ReloadTokens() (int, error) {
	if d.apiServer == nil {
		return 0, errors.New("api server not started")
	}
	if d.cfg.TokenFile != "" {
		return d.apiServer.ReloadTokensFromFile()
	}
	envToken := os.Getenv("NIAC_API_TOKEN")
	switch {
	case envToken != "":
		d.apiServer.SetTokens([]tokenstore.ScopedToken{{Value: envToken, Scope: tokenstore.ScopeReadWrite}})
		return 1, nil
	case d.cfg.Token != "":
		// Fall back to the daemon's startup-captured token when the
		// env var has been unset between starts (rare but possible).
		d.apiServer.SetTokens([]tokenstore.ScopedToken{{Value: d.cfg.Token, Scope: tokenstore.ScopeReadWrite}})
		return 1, nil
	default:
		d.apiServer.SetTokens(nil)
		return 0, nil
	}
}

// TokenScopeCounts returns the number of active (read-only,
// read-write, admin) tokens in the API server's token store. Used by
// the SIGHUP handler to surface a useful audit line without leaking
// token values. Third return is the admin-token count (#743).
func (d *Daemon) TokenScopeCounts() (int, int, int) {
	if d.apiServer == nil {
		return 0, 0, 0
	}
	return d.apiServer.TokenScopeCounts()
}

// Shutdown gracefully shuts down the daemon.
func (d *Daemon) Shutdown(ctx context.Context) error {
	// Preserve active launch intent across process and host restarts.
	d.mu.Lock()
	if d.simulation != nil {
		if stopErr := d.stopSimulationLocked(false); stopErr != nil {
			logging.Errorf("Error stopping simulation: %v", stopErr)
		}
	}
	d.mu.Unlock()

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
func loadSimulationConfig(req api.SimulationRequest, persistInline bool) (*config.Config, string, error) {
	switch {
	case req.TemplateName != "":
		// Loading templates by name preserves the template's own
		// directory as the include_path base — needed for vendor
		// templates with `include_path: ".."` plus relative walk_file
		// refs that resolve to sibling directories (e.g. examples/
		// device_walks_sanitized/...). Fetching the YAML text and
		// POSTing it as ConfigData would lose that context.
		templatePath := templates.Find(req.TemplateName)
		if templatePath == "" {
			return nil, "", fmt.Errorf("%w: %s", ErrTemplateNotFound, req.TemplateName)
		}
		cfg, managedPath, err := config.LoadYAMLManaged(templatePath, simulationConfigRoots())
		if err != nil {
			return nil, "", fmt.Errorf("load template %q: %w", req.TemplateName, err)
		}
		return cfg, managedPath, nil
	case req.ConfigData != "":
		if len(req.ConfigData) > maxSimulationConfigSize {
			return nil, "", fmt.Errorf(
				"%w: %d bytes (got %d bytes)",
				ErrConfigDataExceedsMaxSize,
				maxSimulationConfigSize,
				len(req.ConfigData),
			)
		}
		configDir, err := inlineConfigDir()
		if err != nil {
			return nil, "", fmt.Errorf("load configuration: %w", err)
		}
		cfg, err := config.LoadYAMLBytesManaged(
			[]byte(req.ConfigData),
			configDir,
			simulationConfigRoots(),
		)
		if err != nil {
			return nil, "", fmt.Errorf("load configuration: %w", err)
		}
		if !persistInline {
			return cfg, "", nil
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
		roots := simulationConfigRoots()
		cfg, managedPath, err := config.LoadYAMLManaged(req.ConfigPath, roots)
		if err != nil {
			return nil, "", fmt.Errorf("load configuration: %w", err)
		}
		return cfg, managedPath, nil
	default:
		return nil, "", ErrConfigPathOrDataRequired
	}
}

func simulationConfigRoots() []string {
	roots := []string{
		filepath.Join(library.DefaultRoot(), string(library.KindNetworks)),
		"configs",
		"/var/lib/niac/configs",
		os.ExpandEnv("$HOME/.niac/configs"),
	}
	if custom := os.Getenv("NIAC_CONFIGS_DIR"); custom != "" {
		roots = append([]string{custom}, roots...)
	}
	return append(roots, templates.Dirs()...)
}

func loadValidSimulationConfig(
	req api.SimulationRequest,
	persistInline bool,
) (*config.Config, string, error) {
	cfg, path, err := loadSimulationConfig(req, persistInline)
	if err != nil {
		return nil, "", err
	}
	result := config.NewValidator(path).Validate(cfg)
	if result.HasErrors() {
		return nil, "", fmt.Errorf("%w: %w", ErrInvalidSimulationConfig, result)
	}
	return cfg, path, nil
}

// PreflightSimulation compiles a routed request without opening capture or changing runtime state.
func (d *Daemon) PreflightSimulation(req api.SimulationRequest) (fabric.Report, error) {
	if diagnostic := simulationInterfaceDiagnostic(req.Interface, e2eDryRunSimulation()); diagnostic != nil {
		return fabric.Report{Diagnostics: []fabric.Diagnostic{*diagnostic}}, nil
	}
	cfg, _, err := loadValidSimulationConfig(req, false)
	if err != nil {
		return fabric.Report{}, err
	}
	if !usesRoutedFabric(cfg) {
		return fabric.Report{
			Safe: true,
			Topology: fabric.Topology{Binding: fabric.CompiledBinding{
				Binding: d.bindingFromRequest(req),
			}},
		}, nil
	}
	return fabric.Compile(cfg, d.bindingFromRequest(req)), nil
}

func usesRoutedFabric(cfg *config.Config) bool {
	return len(cfg.Networks) > 0 || len(cfg.Attachments) > 0
}

func (d *Daemon) bindingFromRequest(req api.SimulationRequest) fabric.Binding {
	binding := fabric.Binding{
		Attachment: req.Attachment,
		Interface:  req.Interface,
		Mode:       req.AttachmentMode,
		AccessVLAN: req.AccessVLAN,
	}
	for _, policy := range d.cfg.AttachmentPolicies {
		if policy.Approves(binding) {
			binding.PolicyApproved = true
			break
		}
	}
	return binding
}

type compiledSimulationFabric struct {
	topology *fabric.Topology
}

func (d *Daemon) compileSimulationFabric(
	cfg *config.Config,
	req api.SimulationRequest,
) (compiledSimulationFabric, error) {
	if !usesRoutedFabric(cfg) {
		return compiledSimulationFabric{}, nil
	}
	report := fabric.Compile(cfg, d.bindingFromRequest(req))
	if !report.Safe {
		return compiledSimulationFabric{}, fmt.Errorf("%w: %v", ErrUnsafeTopology, report.Diagnostics)
	}
	return compiledSimulationFabric{topology: &report.Topology}, nil
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
	cleanDir, dirErr := inlineConfigDir()
	if dirErr != nil {
		return "", dirErr
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

func inlineConfigDir() (string, error) {
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
	return cleanDir, nil
}

func e2eDryRunSimulation() bool {
	return strings.EqualFold(os.Getenv(e2eDryRunEnv), "1") ||
		strings.EqualFold(os.Getenv(e2eDryRunEnv), "true")
}

func simulationInterfaceDiagnostic(interfaceName string, dryRun bool) *fabric.Diagnostic {
	if dryRun || capture.InterfaceExists(interfaceName) {
		return nil
	}
	return &fabric.Diagnostic{
		Code:    fabric.CodeHostInterfaceUnavailable,
		Field:   "interface",
		Message: "host interface does not exist",
	}
}

// StartSimulation starts a new simulation.
func (d *Daemon) StartSimulation(req api.SimulationRequest, entitlements api.SimulationEntitlements) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	dryRun := e2eDryRunSimulation()
	if simulationInterfaceDiagnostic(req.Interface, dryRun) != nil {
		return fmt.Errorf("%w: %s", ErrInterfaceNotExist, req.Interface)
	}

	cfg, configPath, err := loadAuthorizedSimulationConfig(req, entitlements)
	if err != nil {
		return err
	}
	compiledFabric, err := d.compileSimulationFabric(cfg, req)
	if err != nil {
		return err
	}
	compiled := compiledFabric.topology
	resources, err := d.startSimulation(req.Interface, cfg, compiled, dryRun)
	if err != nil {
		resources.stop()
		return err
	}
	if req.ConfigData != "" {
		configPath, err = persistInlineConfig(req.ConfigData)
		if err != nil {
			resources.stop()
			return fmt.Errorf("persist inline config: %w", err)
		}
	}

	replacement := &Simulation{
		Interface:  req.Interface,
		ConfigPath: configPath,
		ConfigName: filepath.Base(configPath),
		StartedAt:  time.Now(),
		engine:     resources.engine,
		stack:      resources.stack,
		cfg:        cfg,
		fabric:     compiled,
		replay:     resources.replay,
		cancel:     resources.cancel,
	}
	intent := req
	intent.ConfigPath = configPath
	intent.ConfigData = ""
	intent.TemplateName = ""
	if persistErr := d.persistActiveSimulation(intent); persistErr != nil {
		resources.stop()
		return fmt.Errorf("persist active simulation: %w", persistErr)
	}

	active := d.simulation
	d.simulation = replacement
	d.apiServer.UpdateSimulation(resources.stack, cfg, configPath, req.Interface, resources.replay)
	if active != nil {
		d.stopSimulation(active)
	}

	logging.Successf("✓ Simulation started on %s with %d devices", req.Interface, len(cfg.Devices))

	// Auto-start PCAP playback if configured. The playback is best-effort —
	// a failed playback start logs but doesn't fail the simulation, since
	// the protocol stack is already up and serving devices.
	if !dryRun && resources.replay != nil &&
		cfg.CapturePlayback != nil &&
		strings.TrimSpace(cfg.CapturePlayback.FileName) != "" {
		fileName := resolvePlaybackPath(cfg.CapturePlayback.FileName, configPath)
		_, replayErr := resources.replay.Start(api.ReplayRequest{
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

func startSimulationResources(
	iface string,
	cfg *config.Config,
	topology *fabric.Topology,
	dryRun bool,
) (simulationResources, error) {
	if dryRun {
		stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(DefaultDebugLevel))
		stack.ConfigureFabric(topology)
		_, cancel := context.WithCancel(context.Background())
		return simulationResources{stack: stack, cancel: cancel}, nil
	}

	engine, stack, cancel, err := startSimulationStack(iface, cfg, topology)
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

func (resources simulationResources) stop() {
	if resources.replay != nil {
		_, _ = resources.replay.Stop()
	}
	if resources.cancel != nil {
		resources.cancel()
	}
	if resources.stack != nil {
		resources.stack.Stop()
	}
	if resources.engine != nil {
		resources.engine.Close()
	}
}

func loadAuthorizedSimulationConfig(
	req api.SimulationRequest,
	entitlements api.SimulationEntitlements,
) (*config.Config, string, error) {
	cfg, configPath, err := loadValidSimulationConfig(req, false)
	if err != nil {
		return nil, "", err
	}
	if entitlementErr := api.ValidateConfigEntitlements(cfg, entitlements); entitlementErr != nil {
		return nil, "", entitlementErr
	}
	if runtimeErr := config.ValidateRuntimeRequirements(cfg); runtimeErr != nil {
		return nil, "", runtimeErr
	}
	return cfg, configPath, nil
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
	iface string, cfg *config.Config, topology *fabric.Topology,
) (*capture.Engine, *protocols.Stack, context.CancelFunc, error) {
	engine, err := capture.New(iface, DefaultDebugLevel)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create capture engine: %w", err)
	}

	stack := protocols.NewStack(engine, cfg, logging.NewDebugConfig(DefaultDebugLevel))
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

// StopSimulation stops the current simulation.
func (d *Daemon) StopSimulation() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.stopSimulationLocked(true)
}

func (d *Daemon) stopSimulationLocked(clearIntent bool) error {
	if d.simulation == nil {
		return ErrNoSimulationRunning
	}
	if clearIntent {
		if clearErr := d.clearActiveSimulation(); clearErr != nil {
			return fmt.Errorf("clear active simulation: %w", clearErr)
		}
	}

	sim := d.simulation
	d.simulation = nil
	d.apiServer.ClearSimulation()
	d.stopSimulation(sim)

	logging.Infof("Simulation stopped")

	return nil
}

func (d *Daemon) stopSimulation(sim *Simulation) {
	simulationResources{
		engine: sim.engine,
		stack:  sim.stack,
		replay: sim.replay,
		cancel: sim.cancel,
	}.stop()
	if d.storage != nil && sim.stack != nil && sim.cfg != nil {
		stats := sim.stack.GetStats()
		_ = d.storage.AddRun(storage.RunRecord{
			StartedAt:       sim.StartedAt,
			Duration:        time.Since(sim.StartedAt),
			Interface:       sim.Interface,
			ConfigName:      sim.ConfigName,
			DeviceCount:     len(sim.cfg.Devices),
			PacketsSent:     stats.PacketsSent,
			PacketsReceived: stats.PacketsReceived,
			Errors:          stats.Errors,
		})
	}
}

// GetStatus returns the current simulation status.
func (d *Daemon) GetStatus() api.SimulationStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()

	status := api.SimulationStatus{
		Running:  d.simulation != nil,
		Recovery: d.recovery,
	}

	if d.simulation != nil {
		status.Interface = d.simulation.Interface
		status.ConfigPath = d.simulation.ConfigPath
		status.ConfigName = d.simulation.ConfigName
		status.StartedAt = d.simulation.StartedAt

		status.UptimeSeconds = time.Since(d.simulation.StartedAt).Seconds()
		if d.simulation.cfg != nil {
			status.DeviceCount = d.simulation.cfg.DeviceCount()
		}
		if d.simulation.fabric != nil && d.simulation.stack != nil {
			stats := d.simulation.stack.GetStats()
			topology, ok := d.simulation.stack.RuntimeFabricTopology()
			if !ok {
				topology = *d.simulation.fabric
			}
			status.Fabric = &api.SimulationFabricStatus{
				Topology:    topology,
				Forwarded:   stats.FabricForwarded,
				Drops:       stats.FabricDrops,
				Received:    stats.PacketsReceived,
				Transmitted: stats.PacketsSent,
			}
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
