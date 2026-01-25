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
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/protocols"
	"github.com/krisarmstrong/niac-go/internal/storage"
)

const (
	// MaxRequestBodySize is the maximum size for API request bodies (1MB).
	MaxRequestBodySize = 1 << 20 // 1MB
	// MaxPCAPUploadSize is the maximum size for PCAP file uploads (100MB)
	// SECURITY: This prevents memory exhaustion attacks via large uploads.
	MaxPCAPUploadSize = 100 << 20 // 100MB

	// MaxRateLimiterCount is the maximum number of IP addresses tracked by rate limiter
	// SECURITY FIX #2.8.1: This prevents memory exhaustion from IP spoofing attacks.
	MaxRateLimiterCount = 10000

	// DefaultRateLimit is the default requests per second allowed per IP
	// FEATURE #104: Allow 100 requests per second per IP with burst of 200.
	DefaultRateLimit = 100
	// DefaultBurst is the default burst size for rate limiting.
	DefaultBurst = 200

	// UploadRateLimit controls per-endpoint limits for upload operations.
	UploadRateLimit = 5  // 5 requests per second
	UploadBurst     = 10 // Burst of 10
	// WriteRateLimit controls per-endpoint limits for write operations.
	WriteRateLimit = 20 // 20 requests per second
	WriteBurst     = 40 // Burst of 40
	// WalkRateLimit controls per-endpoint limits for walk file operations.
	WalkRateLimit = 10 // 10 requests per second
	WalkBurst     = 20 // Burst of 20

	// ErrMsgRequestBodyTooLarge is the error message when HTTP request body is too large.
	ErrMsgRequestBodyTooLarge = "http: request body too large"

	// Validation and limit constants.
	requestIDBytes      = 16       // bytes for unique request ID
	csrfTokenBytes      = 32       // bytes for CSRF token
	maxURLLength        = 2048     // max webhook URL length
	maxInterfaceNameLen = 255      // max interface name length
	maxPathLength       = 4096     // max file path length
	maxQueryParamLen    = 1024     // max query parameter length
	maxLoopMs           = 86400000 // max loop ms (24 hours)
	maxScaleFactor      = 1000.0   // max scale factor
	truncateErrorValue  = 50       // truncate value for error messages
	protocolCapacity    = 8        // initial protocol slice capacity
	historyListLimit    = 20       // limit for history listing
	maxFileEntries      = 200      // max file entries to return
	minPCAPSize         = 4        // minimum PCAP file size

	// MaxHeaderBytesSize is the maximum size for HTTP request headers (64KB).
	// SECURITY FIX #282: Reduced from 1MB to prevent header-based resource exhaustion.
	MaxHeaderBytesSize = 64 << 10 // 64KB

	// HTTP server timeout constants.
	httpReadTimeout       = 10 // seconds
	httpWriteTimeout      = 10 // seconds
	httpIdleTimeout       = 60 // seconds
	httpReadHeaderTimeout = 5  // seconds

	// Background task interval constants.
	rateLimiterCleanupMins = 5  // minutes between rate limiter cleanup
	sseTokenCleanupSecs    = 60 // seconds between SSE token cleanup
	alertTickerSecs        = 5  // seconds between alert checks
	webhookTimeoutSecs     = 5  // seconds for webhook timeout

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

// ErrorResponse represents a standardized API error response
// FEATURE #105: Consistent error format for all API endpoints.
type ErrorResponse struct {
	Error     string        `json:"error"`                // Machine-readable error code
	Message   string        `json:"message"`              // Human-readable error message
	Details   []ErrorDetail `json:"details,omitempty"`    // Optional detailed error information
	RequestID string        `json:"request_id,omitempty"` // Optional request ID for tracing
	Timestamp time.Time     `json:"timestamp"`            // When the error occurred
	Path      string        `json:"path"`                 // Request path that caused the error
	Method    string        `json:"method"`               // HTTP method
}

// ErrorDetail provides detailed information about a specific error.
type ErrorDetail struct {
	Field string `json:"field,omitempty"` // Field name that caused the error
	Issue string `json:"issue"`           // Description of the issue
	Value string `json:"value,omitempty"` // The value that caused the error (sanitized)
}

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

// FileEntry represents a discovered file (pcap, walk, etc.).
type FileEntry struct {
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	SizeBytes int64     `json:"size_bytes"`
	Modified  time.Time `json:"modified_at"`
}

// ReplayManager controls PCAP playback from the API server.
type ReplayManager interface {
	Status() ReplayState
	Start(ReplayRequest) (ReplayState, error)
	Stop() (ReplayState, error)
}

// ServerConfig defines API server options.
type ServerConfig struct {
	Addr             string
	MetricsAddr      string
	Token            string
	Stack            *protocols.Stack
	Config           *config.Config
	ConfigPath       string
	Storage          *storage.Storage
	Interface        string
	Version          string
	Topology         Topology
	Alert            AlertConfig
	ApplyConfig      func(*config.Config) error
	Replay           ReplayManager
	TrustedProxies   []*net.IPNet // FIX #277: Configurable trusted proxy CIDRs
	CORSAllowOrigins []string     // FIX #267: Configurable CORS allowed origins
	TemplatesDir     string       // FIX #263: Directory for template files
	PcapCacheMemory  int64        // FIX #290: Configurable PCAP cache memory (bytes), default 100MB
}

// SimulationRequest represents a request to start a simulation.
type SimulationRequest struct {
	Interface  string `json:"interface"`
	ConfigPath string `json:"config_path,omitempty"`
	ConfigData string `json:"config_data,omitempty"`
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
	cfg           ServerConfig
	logger        *slog.Logger
	httpServer    *http.Server
	metricsServer *http.Server
	alertStop     chan struct{}
	bgDone        chan struct{} // FIX #271: Signal background goroutines to stop
	lastAlert     uint64
	alertMu       sync.RWMutex
	configMu      sync.RWMutex
	daemon        DaemonController // Optional: only set in daemon mode
	startTime     time.Time        // Track server start time for uptime
	rateLimiter   *RateLimiter     // FEATURE #104: Per-IP rate limiting
	csrfMu        sync.RWMutex     // Protects csrfToken, csrfPrevToken, csrfExpiry
	csrfToken     string           // SECURITY FIX LOW-1: CSRF protection token
	csrfPrevToken string           // FIX #293: Previous CSRF token for rotation window
	csrfExpiry    time.Time        // FIX #293: CSRF token expiry time
	sseHub        *SSEHub          // SSE hub for real-time streaming
	sseTokens     sync.Map         // FIX #283: Short-lived SSE tokens (map[string]*sseTokenEntry)
	pcapCache     *pcapCache       // FIX #280: Per-server PCAP cache (not global)
	templateStore *templateStore   // FIX #263: Template management store
	// SECURITY FIX #156: Per-endpoint rate limiters for sensitive operations
	uploadLimiter *RateLimiter // Stricter limits for upload endpoints
	writeLimiter  *RateLimiter // Moderate limits for write operations
	walkLimiter   *RateLimiter // Limits for walk file operations
}

// NewServer returns a configured API server.
func NewServer(cfg ServerConfig) *Server {
	logger := slog.Default()

	// FIX #273: Log critical warning if CSRF token generation fails
	csrfToken, csrfErr := generateCSRFToken()
	if csrfErr != nil {
		logger.Error("[API] CRITICAL: Failed to generate CSRF token - CSRF protection disabled",
			"error", csrfErr)
	}

	return &Server{
		cfg:           cfg,
		logger:        logger,
		startTime:     time.Now(),
		rateLimiter:   NewRateLimiter(DefaultRateLimit, DefaultBurst),
		csrfToken:     csrfToken,
		csrfExpiry:    time.Now().Add(1 * time.Hour), // FIX #293: Token valid for 1 hour
		sseHub:        NewSSEHub(),
		bgDone:        make(chan struct{}),                        // FIX #271: Background goroutine stop signal
		pcapCache:     newPcapCacheWithLimit(cfg.PcapCacheMemory), // FIX #290: Configurable cache
		templateStore: newTemplateStore(cfg.TemplatesDir),         // FIX #263: Template store
		// SECURITY FIX #156: Initialize per-endpoint rate limiters
		uploadLimiter: NewRateLimiter(UploadRateLimit, UploadBurst),
		writeLimiter:  NewRateLimiter(WriteRateLimit, WriteBurst),
		walkLimiter:   NewRateLimiter(WalkRateLimit, WalkBurst),
	}
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
}

// ClearSimulation clears simulation components (for daemon mode).
func (s *Server) ClearSimulation() {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	s.cfg.Stack = nil
	s.cfg.Config = nil
	s.cfg.Replay = nil
}

// startBackgroundTasks starts the rate limiter cleanup and SSE hub goroutines.
// FIX #271: Background goroutines now respect bgDone channel for clean shutdown.
func (s *Server) startBackgroundTasks() {
	go func() {
		ticker := time.NewTicker(rateLimiterCleanupMins * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.rateLimiter.CleanupStale()
			case <-s.bgDone:
				return
			}
		}
	}()

	// FIX #301: Periodically clean up expired SSE tokens
	go func() {
		ticker := time.NewTicker(sseTokenCleanupSecs * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.cleanupSSETokens()
			case <-s.bgDone:
				return
			}
		}
	}()

	// Start SSE hub for real-time streaming
	go s.sseHub.Run()
	s.logger.Info("[SSE] Server-Sent Events hub started")
}

// Start boots the HTTP listeners.
// SECURITY FIX #98: Goroutines will properly exit when Shutdown() is called
// The ListenAndServe calls run in goroutines and will terminate when Shutdown()
// is invoked, preventing goroutine leaks. Always call Shutdown() to cleanup.
func (s *Server) Start() error {
	// In daemon mode, Stack and Config can be nil initially (set later when simulation starts)
	// In non-daemon mode, they must be set before Start()
	// SECURITY FIX #161: Thread-safe access to Stack and Config
	s.configMu.RLock()
	hasStack := s.cfg.Stack != nil
	hasConfig := s.cfg.Config != nil
	s.configMu.RUnlock()

	if s.daemon == nil && (!hasStack || !hasConfig) {
		return ErrAPIServerRequiresStackAndConfig
	}

	// SECURITY FIX #107: Warn if API is running without authentication
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
		mux.HandleFunc("/metrics", s.recoverMiddleware(s.handleMetrics))
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
// FIX #271, #274: Stops background goroutines and SSE hub.
func (s *Server) Shutdown(ctx context.Context) error {
	// FIX #271: Stop background goroutines
	close(s.bgDone)

	// FIX #274: Stop SSE hub and wait for it to finish
	if s.sseHub != nil {
		s.sseHub.Stop()
		<-s.sseHub.Done()
	}

	// Acquire lock before closing channel to prevent race with updateAlertConfig
	s.alertMu.Lock()

	if s.alertStop != nil {
		close(s.alertStop)
		s.alertStop = nil
	}

	s.alertMu.Unlock()

	var errs []error

	// Shutdown metrics server first (less critical)
	if s.metricsServer != nil {
		if err := s.metricsServer.Shutdown(ctx); err != nil {
			s.logger.ErrorContext(ctx, "Error shutting down metrics server", "error", err)
			errs = append(errs, err)
		}
	}

	// Shutdown main HTTP server
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			s.logger.ErrorContext(ctx, "Error shutting down HTTP server", "error", err)
			errs = append(errs, err)
		}
	}

	// Return first error encountered, if any
	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}
