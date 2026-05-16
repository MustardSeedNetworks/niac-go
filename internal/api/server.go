// Package api provides the REST API server and web UI for NIAC.
//
// The API server exposes endpoints for:
//   - Configuration management (read, update, validate)
//   - Statistics and monitoring (runtime stats, device info, topology)
//   - PCAP replay control (upload, start, stop)
//   - Alert configuration (threshold-based notifications)
//   - Simulation control in daemon mode (start, stop, status)
//
// Security features include:
//   - Bearer token authentication
//   - CSRF protection on state-changing endpoints
//   - Per-IP rate limiting (100 req/s with burst of 200)
//   - Comprehensive input validation
//   - Panic recovery middleware
//   - Security headers (CSP, X-Frame-Options, etc.)
//
// The server can operate in two modes:
//   - Standard mode: API for a running simulation
//   - Daemon mode: Full simulation lifecycle control via API
package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/library"
	"github.com/krisarmstrong/niac-go/internal/protocols"
	"github.com/krisarmstrong/niac-go/internal/storage"
)

const (
	// MaxRequestBodySize is the maximum size for API request bodies (1MB).
	MaxRequestBodySize = 1 << 20 // 1MB
	// MaxPCAPUploadSize is the maximum size for PCAP file uploads (100MB).
	MaxPCAPUploadSize = 100 << 20 // 100MB

	// MaxRateLimiterCount is the maximum number of IP addresses tracked by rate limiter.
	MaxRateLimiterCount = 10000

	// DefaultRateLimit is the default requests per second allowed per IP.
	DefaultRateLimit = 100
	// DefaultBurst is the default burst size for rate limiting.
	DefaultBurst = 200

	// UploadRateLimit controls per-endpoint limits for upload operations.
	UploadRateLimit = 5  // 5 requests per second
	UploadBurst     = 10 // Burst of 10
	// WriteRateLimit controls per-endpoint limits for write operations.
	WriteRateLimit = 20 // 20 requests per second
	WriteBurst     = 40 // Burst of 40
	// FileRateLimit controls per-endpoint limits for file listing operations.
	FileRateLimit = 30 // 30 requests per second
	FileBurst     = 60 // Burst of 60
	// WalkRateLimit controls per-endpoint limits for walk file operations.
	WalkRateLimit = 10 // 10 requests per second
	WalkBurst     = 20 // Burst of 20

	// ErrMsgRequestBodyTooLarge is the error message when HTTP request body is too large.
	ErrMsgRequestBodyTooLarge = "http: request body too large"

	// Validation and limit constants.
	requestIDBytes      = 16       // bytes for unique request ID
	csrfTokenBytes      = 32       // bytes for CSRF token
	maxURLLength        = 2048     // max webhook URL length
	maxInterfaceNameLen = 15       // Linux IFNAMSIZ limit
	maxPathLength       = 4096     // max file path length
	maxQueryParamLen    = 1024     // max query parameter length
	maxLoopMs           = 86400000 // max loop ms (24 hours)
	maxScaleFactor      = 1000.0   // max scale factor
	truncateErrorValue  = 50       // truncate value for error messages
	protocolCapacity    = 8        // initial protocol slice capacity
	historyListLimit    = 20       // limit for history listing
	maxFileEntries      = 200      // max file entries to return
	minPCAPSize         = 4        // minimum PCAP file size

	// HTTP server timeout constants.
	httpReadTimeout       = 10 // seconds
	httpWriteTimeout      = 10 // seconds
	httpIdleTimeout       = 60 // seconds
	httpReadHeaderTimeout = 5  // seconds

	// Background task interval constants.
	rateLimiterCleanupMins = 5 // minutes between rate limiter cleanup
	alertTickerSecs        = 5 // seconds between alert checks
	webhookTimeoutSecs     = 5 // seconds for webhook timeout

	// Bit shift constants for IP parsing.
	bitShift24 = 24
	bitShift16 = 16
	bitShift8  = 8

	// Base64 encoding ratio (4/3).
	base64Ratio = 3
)

// Sentinel errors for API server.
var (
	ErrAPIServerRequiresStackAndConfig = errors.New(
		"api server requires stack and config references",
	)
	ErrFileTooSmallForPCAP         = errors.New("file too small to be a valid PCAP (< 4 bytes)")
	ErrInvalidPCAPMagicNumber      = errors.New("invalid PCAP magic number")
	ErrPcapFilePathOrDataRequired  = errors.New("pcap file path or data is required")
	ErrPCAPDataExceedsSizeLimit    = errors.New("PCAP data exceeds size limit (max 100MB)")
	ErrDecodedPCAPExceedsSizeLimit = errors.New("decoded PCAP exceeds size limit (max 100MB)")
	ErrPathIsADirectory            = errors.New("path is a directory")
	ErrConfigPathNotAvailable      = errors.New("config path not available")
	ErrConfigFileNotFound          = errors.New("config file not found")
	ErrUnsupportedFileKind         = errors.New("unsupported file kind")
	ErrNotADirectory               = errors.New("path is not a directory")
	ErrPathOutsideRoot             = errors.New("path is outside allowed directory")
)

// AlertConfig controls basic threshold-based alerting.
type AlertConfig struct {
	PacketsThreshold uint64 `json:"packets_threshold"`
	WebhookURL       string `json:"webhook_url"`
}

// ReplayRequest represents a packet replay request.
type ReplayRequest struct {
	File       string  `json:"file"`
	LoopMs     int     `json:"loop_ms"`
	Scale      float64 `json:"scale"`
	InlineData string  `json:"data,omitempty"`
	Uploaded   bool    `json:"-"`
}

// ReplayState reports the current replay status.
type ReplayState struct {
	Running   bool      `json:"running"`
	File      string    `json:"file"`
	LoopMs    int       `json:"loop_ms"`
	Scale     float64   `json:"scale"`
	StartedAt time.Time `json:"started_at,omitzero"`
}

// ReplayManager controls PCAP playback from the API server.
type ReplayManager interface {
	Status() ReplayState
	Start(ReplayRequest) (ReplayState, error)
	Stop() (ReplayState, error)
}

// ServerConfig defines API server options.
type ServerConfig struct {
	Addr        string
	MetricsAddr string
	Token       string
	Stack       *protocols.Stack
	Config      *config.Config
	ConfigPath  string
	Storage     *storage.Storage
	Interface   string
	Version     string
	Commit      string
	BuildTime   string
	UIBuildHash string
	Topology    Topology
	Alert       AlertConfig
	// WebhookAllowedHosts is an admin-side allowlist of hostnames that the
	// alert webhook is permitted to dispatch to. Empty = no allowlist (the
	// existing private-IP / blocked-hostname filters in
	// validateWebhookURLSSRF still apply). Non-empty = strict allowlist:
	// the parsed URL's hostname must match one of these exactly.
	WebhookAllowedHosts []string
	ApplyConfig         func(*config.Config) error
	Replay              ReplayManager
	// LibraryRoot is the on-disk root the unified library reads from
	// (~/.niac/library by default, /var/lib/niac/library when packaged).
	// If empty, the daemon picks a sensible default via library.DefaultRoot().
	LibraryRoot string
}

// SimulationRequest represents a request to start a simulation.
type SimulationRequest struct {
	Interface  string `json:"interface"`
	ConfigPath string `json:"config_path,omitempty"`
	ConfigData string `json:"config_data,omitempty"`
	// TemplateName, when set, tells the daemon to load a built-in
	// template directly from disk by name. This preserves the template's
	// own directory as the include_path base, which matters for templates
	// like vendors/paloalto-firewall.yaml that reference walk files via
	// `include_path: ".."` plus a relative `walk_file:` path. Fetching the
	// template content and POSTing it as ConfigData would lose that
	// directory context and trip the walk-file path-traversal guard.
	TemplateName string `json:"template_name,omitempty"`
}

// SimulationStatus represents the current simulation status.
type SimulationStatus struct {
	Running       bool      `json:"running"`
	Interface     string    `json:"interface,omitempty"`
	ConfigPath    string    `json:"config_path,omitempty"`
	ConfigName    string    `json:"config_name,omitempty"`
	DeviceCount   int       `json:"device_count"`
	StartedAt     time.Time `json:"started_at,omitzero"`
	UptimeSeconds float64   `json:"uptime_seconds"`
}

// DaemonController interface for daemon mode operations.
type DaemonController interface {
	StartSimulation(req SimulationRequest) error
	StopSimulation() error
	GetStatus() SimulationStatus
}

// Server exposes the REST API, metrics endpoint, and Web UI.
type Server struct {
	cfg               ServerConfig
	logger            *slog.Logger
	httpServer        *http.Server
	metricsServer     *http.Server
	alertStop         chan struct{}
	lastAlert         uint64
	alertMu           sync.RWMutex
	configMu          sync.RWMutex
	daemon            DaemonController
	captureController CaptureController
	startTime         time.Time
	rateLimiter       *RateLimiter
	csrfToken         string
	sseHub            *SSEHub
	uploadLimiter     *RateLimiter
	writeLimiter      *RateLimiter
	walkLimiter       *RateLimiter
	fileLimiter       *RateLimiter
	bgStop            chan struct{}
	bgStopOnce        sync.Once
	library           *library.Library
}

// NewServer returns a configured API server.
func NewServer(cfg ServerConfig) *Server {
	csrfToken, err := generateCSRFToken()
	if err != nil {
		slog.Error("[API] Failed to generate CSRF token, server cannot start securely", "error", err)

		return nil
	}

	libraryRoot := cfg.LibraryRoot
	if libraryRoot == "" {
		libraryRoot = library.DefaultRoot()
	}
	lib, err := library.Open(libraryRoot)
	if err != nil {
		// Library failure is non-fatal — log loudly, leave Server.library
		// nil, and the library endpoints will return 503. The rest of
		// the daemon (sim engine, /api/v1/devices, etc.) keeps working.
		slog.Error("[API] Failed to open content library, /api/v1/library endpoints disabled",
			"root", libraryRoot, "error", err)
	} else {
		slog.Info("[API] Content library opened", "root", lib.Root())
	}

	return &Server{
		cfg:           cfg,
		logger:        slog.Default(),
		startTime:     time.Now(),
		rateLimiter:   NewRateLimiter(DefaultRateLimit, DefaultBurst),
		csrfToken:     csrfToken,
		sseHub:        NewSSEHub(SSEConfig{}),
		bgStop:        make(chan struct{}),
		uploadLimiter: NewRateLimiter(UploadRateLimit, UploadBurst),
		writeLimiter:  NewRateLimiter(WriteRateLimit, WriteBurst),
		walkLimiter:   NewRateLimiter(WalkRateLimit, WalkBurst),
		fileLimiter:   NewRateLimiter(FileRateLimit, FileBurst),
		library:       lib,
	}
}

// Start boots the HTTP listeners.
func (s *Server) Start() error {
	s.configMu.RLock()
	hasStack := s.cfg.Stack != nil
	hasConfig := s.cfg.Config != nil
	s.configMu.RUnlock()

	if s.daemon == nil && (!hasStack || !hasConfig) {
		return ErrAPIServerRequiresStackAndConfig
	}

	if s.cfg.Token == "" && s.cfg.Addr != "" {
		s.logger.Warn(
			"API server running WITHOUT authentication - all endpoints are publicly accessible",
		)
		s.logger.Warn("Set NIAC_API_TOKEN environment variable to enable authentication")
		s.logger.Info("Example: export NIAC_API_TOKEN=$(openssl rand -base64 32)")
	}

	if s.cfg.Addr != "" {
		mux := http.NewServeMux()
		s.registerAPIRoutes(mux)
		s.httpServer = newSecureHTTPServer(s.cfg.Addr, mux)

		go func() {
			if err := s.httpServer.ListenAndServe(); err != nil &&
				!errors.Is(err, http.ErrServerClosed) {
				s.logger.Error("API server stopped", "error", err)
			}
		}()
	}

	if s.cfg.MetricsAddr != "" && s.cfg.MetricsAddr != s.cfg.Addr {
		mux := http.NewServeMux()
		mux.HandleFunc("/metrics", s.recoverMiddleware(s.auth(s.handleMetrics)))
		s.metricsServer = newSecureHTTPServer(s.cfg.MetricsAddr, mux)

		go func() {
			if err := s.metricsServer.ListenAndServe(); err != nil &&
				!errors.Is(err, http.ErrServerClosed) {
				s.logger.Error("Metrics server stopped", "error", err)
			}
		}()
	}

	s.startBackgroundTasks()
	s.updateAlertConfig(s.cfg.Alert)

	return nil
}

// Shutdown gracefully shuts down the API and metrics servers.
func (s *Server) Shutdown(ctx context.Context) error {
	s.alertMu.Lock()

	if s.alertStop != nil {
		close(s.alertStop)
		s.alertStop = nil
	}

	s.alertMu.Unlock()

	// Stop background goroutines (rate limiter cleanup).
	s.bgStopOnce.Do(func() {
		if s.bgStop != nil {
			close(s.bgStop)
		}
	})

	// Stop SSE hub to release its goroutine and clients.
	if s.sseHub != nil {
		s.sseHub.Stop()
	}

	var errs []error

	if s.metricsServer != nil {
		if err := s.metricsServer.Shutdown(ctx); err != nil {
			s.logger.ErrorContext(ctx, "Error shutting down metrics server", "error", err)
			errs = append(errs, err)
		}
	}

	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			s.logger.ErrorContext(ctx, "Error shutting down HTTP server", "error", err)
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

// SetDaemonController sets the daemon controller (for daemon mode).
func (s *Server) SetDaemonController(daemon DaemonController) {
	s.daemon = daemon
}

// UpdateSimulation updates the server with simulation components (for daemon mode).
func (s *Server) UpdateSimulation(
	stack *protocols.Stack,
	cfg *config.Config,
	configPath string,
	iface string,
	replay ReplayManager,
) {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	s.cfg.Stack = stack
	s.cfg.Config = cfg
	s.cfg.ConfigPath = configPath
	s.cfg.Interface = iface
	s.cfg.Replay = replay
	s.cfg.Topology = BuildTopology(cfg)

	// Wire the stack's packet stream into the SSE hub so the
	// /api/v1/stream/packets subscribers actually see frames.
	// Previously the hub had BroadcastPacket defined but it was never
	// called, leaving the Packet Capture page perpetually empty even
	// while a simulation was clearly handling traffic.
	if stack != nil && s.sseHub != nil {
		stack.AddPacketObserver(&sseHubPacketObserver{hub: s.sseHub})
	}
}

// ClearSimulation clears simulation components (for daemon mode).
func (s *Server) ClearSimulation() {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	s.cfg.Stack = nil
	s.cfg.Config = nil
	s.cfg.Replay = nil
}
