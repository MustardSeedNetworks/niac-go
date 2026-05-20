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
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
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

// errNonLoopbackRequiresToken refuses startup when --listen targets a
// non-loopback address without an API token. The message must be helpful
// because operators will hit this the first time they expose the API
// beyond 127.0.0.1 — see task #88 part 1.
var errNonLoopbackRequiresToken = errors.New(
	"niac refuses to bind a non-loopback address without an API token.\n" +
		"Set NIAC_API_TOKEN=<value> or pass --api-token=<value>.\n" +
		"If you want loopback-only (no auth), set --listen=127.0.0.1")

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
	// EnableTLS controls whether the API listener uses HTTPS. When true,
	// CertFile/KeyFile (or, when both are empty, the auto-generated
	// self-signed pair under CertDir) are loaded and the listener serves
	// TLS 1.3. See internal/api/tls.go.
	EnableTLS bool
	// CertDir is the directory the self-signed cert+key default to when
	// CertFile and KeyFile are not explicitly set. Empty falls back to
	// `certs/` relative to the working directory.
	CertDir string
	// CertFile is the path to a PEM-encoded server certificate. Empty
	// triggers auto-generation under CertDir.
	CertFile string
	// KeyFile is the path to the PEM-encoded private key for CertFile.
	// Empty triggers auto-generation under CertDir.
	KeyFile string
	// HTTPRedirectAddr, when non-empty and EnableTLS is true, starts a
	// tiny HTTP listener on that address that 308-redirects to the
	// HTTPS service. Wave 1 default is `:8044`.
	HTTPRedirectAddr string
	// TLSPort is the destination HTTPS port the redirect handler advertises.
	// Defaults to defaultTLSPort (8445) when zero.
	TLSPort int
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
	redirectServer    *http.Server
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
	tlsFingerprint    tlsFingerprintCache
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

	// Default-secure gate (task #88 part 1): when binding to a
	// non-loopback address we refuse to start without an API token.
	// Loopback binds are still allowed without one but log a warning.
	if s.cfg.Addr != "" {
		nonLoopback, parseErr := addrIsNonLoopback(s.cfg.Addr)
		if parseErr != nil {
			return fmt.Errorf("parse listen address: %w", parseErr)
		}
		if nonLoopback && s.cfg.Token == "" {
			return errNonLoopbackRequiresToken
		}
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

		// Bind before launching the serve goroutine so a fatal bind error
		// (e.g. permission denied, or all fallback ports taken) surfaces
		// synchronously instead of disappearing into a background log.
		// See #69 — busy 8080 / 8445 falls back to +1..+9 with a WARN.
		apiLn, apiAddr, bindErr := bindWithFallback(context.Background(), s.logger, s.cfg.Addr)
		if bindErr != nil {
			return fmt.Errorf("API server bind: %w", bindErr)
		}
		s.httpServer.Addr = apiAddr

		if s.cfg.EnableTLS {
			certFile, keyFile, certErr := s.resolveTLSCertPaths()
			if certErr != nil {
				_ = apiLn.Close()
				return fmt.Errorf("resolve TLS cert: %w", certErr)
			}
			s.httpServer.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS13}
			go func() {
				if err := s.httpServer.ServeTLS(apiLn, certFile, keyFile); err != nil &&
					!errors.Is(err, http.ErrServerClosed) {
					s.logger.Error("API server (TLS) stopped", "error", err)
				}
			}()
		} else {
			go func() {
				if err := s.httpServer.Serve(apiLn); err != nil &&
					!errors.Is(err, http.ErrServerClosed) {
					s.logger.Error("API server stopped", "error", err)
				}
			}()
		}

		if s.cfg.EnableTLS && s.cfg.HTTPRedirectAddr != "" {
			s.startHTTPRedirect()
		}
	}

	if s.cfg.MetricsAddr != "" && s.cfg.MetricsAddr != s.cfg.Addr {
		mux := http.NewServeMux()
		mux.HandleFunc("/metrics", s.recoverMiddleware(s.auth(s.handleMetrics)))
		s.metricsServer = newSecureHTTPServer(s.cfg.MetricsAddr, mux)

		metricsLn, metricsAddr, bindErr := bindWithFallback(context.Background(), s.logger, s.cfg.MetricsAddr)
		if bindErr != nil {
			return fmt.Errorf("metrics server bind: %w", bindErr)
		}
		s.metricsServer.Addr = metricsAddr

		go func() {
			if err := s.metricsServer.Serve(metricsLn); err != nil &&
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

	if s.redirectServer != nil {
		if err := s.redirectServer.Shutdown(ctx); err != nil {
			s.logger.ErrorContext(ctx, "Error shutting down HTTP redirect server", "error", err)
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

// resolveTLSCertPaths returns the cert+key paths the listener should
// load. When the explicit ServerConfig.CertFile/KeyFile are set we use
// them as-is; otherwise we fall back to the auto-generated pair under
// CertDir (creating it on first start).
func (s *Server) resolveTLSCertPaths() (string, string, error) {
	if s.cfg.CertFile != "" && s.cfg.KeyFile != "" {
		return s.cfg.CertFile, s.cfg.KeyFile, nil
	}
	certPath, keyPath := DefaultCertPaths(s.cfg.CertDir)
	c, k, err := ensureSelfSignedCert(certPath, keyPath)
	if err != nil {
		return "", "", err
	}
	// Populate ServerConfig.CertFile/KeyFile so /__version's fingerprint
	// lookup hits the same path the listener loaded.
	s.cfg.CertFile = c
	s.cfg.KeyFile = k
	return c, k, nil
}

// startHTTPRedirect spawns the tiny HTTP listener that 308-redirects to
// the HTTPS service. The redirect target port defaults to defaultTLSPort
// when ServerConfig.TLSPort is zero. Errors during bind are logged and
// non-fatal: the API itself is still up via the TLS listener.
func (s *Server) startHTTPRedirect() {
	port := s.cfg.TLSPort
	if port == 0 {
		port = defaultTLSPort
	}
	handler := httpToHTTPSRedirectHandler(port)

	const redirectReadWriteTimeoutSec = 5
	srv := &http.Server{
		Addr:              s.cfg.HTTPRedirectAddr,
		Handler:           handler,
		ReadTimeout:       redirectReadWriteTimeoutSec * time.Second,
		WriteTimeout:      redirectReadWriteTimeoutSec * time.Second,
		ReadHeaderTimeout: redirectReadWriteTimeoutSec * time.Second,
	}

	ln, boundAddr, bindErr := bindWithFallback(context.Background(), s.logger, s.cfg.HTTPRedirectAddr)
	if bindErr != nil {
		s.logger.Error("HTTP→HTTPS redirect listener bind failed",
			"addr", s.cfg.HTTPRedirectAddr, "error", bindErr)
		return
	}
	srv.Addr = boundAddr
	s.redirectServer = srv
	s.logger.Info("Starting HTTP→HTTPS redirect server", "addr", boundAddr, "https_port", port)

	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("HTTP redirect server stopped", "error", err)
		}
	}()
}

// addrIsNonLoopback reports whether the host portion of a "host:port"
// listen address resolves to a non-loopback IP. An empty host, "0.0.0.0",
// "::", or any host that resolves to a non-loopback IP returns true so the
// caller can enforce the API-token requirement (task #88 part 1).
func addrIsNonLoopback(addr string) (bool, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false, fmt.Errorf("split host/port %q: %w", addr, err)
	}
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		// Empty host means "bind all interfaces" — definitely non-loopback.
		return true, nil
	case "localhost":
		return false, nil
	}
	// Literal IP fast path.
	if ip := net.ParseIP(host); ip != nil {
		return !ip.IsLoopback(), nil
	}
	// Hostname — resolve and treat as non-loopback if any returned IP
	// is non-loopback. Unresolvable hosts surface the lookup error to the
	// caller so the operator sees the DNS failure and fixes it; we no
	// longer silently treat them as non-loopback.
	//
	// The lookup is bounded by a short context so a slow/broken resolver
	// doesn't hang server startup. noctx rule requires the resolver form
	// (net.LookupIP wraps the same call but elides the context.)
	ctx, cancel := context.WithTimeout(context.Background(), addrLookupTimeout)
	defer cancel()
	ips, lookupErr := net.DefaultResolver.LookupIPAddr(ctx, host)
	if lookupErr != nil {
		return false, fmt.Errorf("lookup %q: %w", host, lookupErr)
	}
	for _, ipAddr := range ips {
		if !ipAddr.IP.IsLoopback() {
			return true, nil
		}
	}
	return false, nil
}

// addrLookupTimeout bounds the DNS lookup inside addrIsNonLoopback so
// startup cannot hang on a slow resolver. Two seconds is plenty for the
// localhost-vs-real-hostname decision the gate makes.
const addrLookupTimeout = 2 * time.Second

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
