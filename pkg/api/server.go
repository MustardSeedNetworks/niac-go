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
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/krisarmstrong/niac-go/pkg/capture"
	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/errors"
	"github.com/krisarmstrong/niac-go/pkg/protocols"
	"github.com/krisarmstrong/niac-go/pkg/storage"
)

const (
	// MaxRequestBodySize is the maximum size for API request bodies (1MB)
	MaxRequestBodySize = 1 << 20 // 1MB
	// MaxPCAPUploadSize is the maximum size for PCAP file uploads (100MB)
	// SECURITY: This prevents memory exhaustion attacks via large uploads
	MaxPCAPUploadSize = 100 << 20 // 100MB

	// MaxRateLimiterCount is the maximum number of IP addresses tracked by rate limiter
	// SECURITY FIX #2.8.1: This prevents memory exhaustion from IP spoofing attacks
	MaxRateLimiterCount = 10000

	// DefaultRateLimit is the default requests per second allowed per IP
	// FEATURE #104: Allow 100 requests per second per IP with burst of 200
	DefaultRateLimit = 100
	// DefaultBurst is the default burst size for rate limiting
	DefaultBurst = 200
)

// rateLimiterEntry tracks a rate limiter with its last access time
// SECURITY FIX HIGH-2: Prevents unbounded memory growth
type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter provides per-IP rate limiting for API requests
// FEATURE #104: Prevents brute force and DoS attacks
type RateLimiter struct {
	limiters map[string]*rateLimiterEntry
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewRateLimiter creates a new rate limiter with the given rate and burst
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rateLimiterEntry),
		rate:     r,
		burst:    b,
	}
}

// GetLimiter returns the rate limiter for the given IP address
// SECURITY FIX #2.8.1: Enforce maximum count to prevent memory exhaustion
func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.limiters[ip]
	if !exists {
		// SECURITY FIX #2.8.1: Enforce max size limit
		if len(rl.limiters) >= MaxRateLimiterCount {
			// Remove oldest entry (FIFO eviction)
			var oldestIP string
			oldestTime := time.Now()
			for checkIP, checkEntry := range rl.limiters {
				if checkEntry.lastSeen.Before(oldestTime) {
					oldestTime = checkEntry.lastSeen
					oldestIP = checkIP
				}
			}
			if oldestIP != "" {
				delete(rl.limiters, oldestIP)
				log.Printf("[API] Rate limiter at capacity (%d), evicted oldest IP: %s", MaxRateLimiterCount, oldestIP)
			}
		}

		entry = &rateLimiterEntry{
			limiter:  rate.NewLimiter(rl.rate, rl.burst),
			lastSeen: time.Now(),
		}
		rl.limiters[ip] = entry
	} else {
		// Update last seen time
		entry.lastSeen = time.Now()
	}

	return entry.limiter
}

// CleanupStale removes limiters for IPs that haven't been seen recently
// SECURITY FIX HIGH-2: Aggressive cleanup to prevent memory exhaustion
// This prevents memory growth from storing limiters for millions of IPs over time
func (rl *RateLimiter) CleanupStale() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	// Remove limiters not seen in the last hour
	// This is aggressive enough to prevent memory growth while allowing
	// legitimate clients to maintain their rate limit state during normal usage
	const staleThreshold = 1 * time.Hour

	count := 0
	for ip, entry := range rl.limiters {
		if now.Sub(entry.lastSeen) > staleThreshold {
			delete(rl.limiters, ip)
			count++
		}
	}

	if count > 0 {
		log.Printf("[API] Cleaned up %d stale rate limiters (total: %d)", count, len(rl.limiters))
	}
}

// getClientIP extracts the real client IP from the request
// SECURITY FIX HIGH-1: Only trust forwarded headers from trusted proxies
func getClientIP(r *http.Request) string {
	// Get the direct connection IP first
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteIP = r.RemoteAddr
	}

	// SECURITY: Only trust X-Forwarded-For/X-Real-IP if coming from localhost/private networks
	// This prevents header spoofing attacks where clients forge these headers to bypass rate limiting
	// In production behind a reverse proxy, configure trusted proxy ranges
	if isTrustedProxy(remoteIP) {
		// Check X-Forwarded-For header (for proxies/load balancers)
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// Take the first IP in the chain (leftmost = original client)
			if idx := strings.Index(xff, ","); idx != -1 {
				clientIP := strings.TrimSpace(xff[:idx])
				// Validate it's a valid IP before trusting it
				if net.ParseIP(clientIP) != nil {
					return clientIP
				}
			} else {
				clientIP := strings.TrimSpace(xff)
				if net.ParseIP(clientIP) != nil {
					return clientIP
				}
			}
		}

		// Check X-Real-IP header
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			clientIP := strings.TrimSpace(xri)
			if net.ParseIP(clientIP) != nil {
				return clientIP
			}
		}
	}

	// Use direct connection IP (not trusted proxy or invalid forwarded IP)
	return remoteIP
}

// isTrustedProxy checks if an IP is from a trusted proxy/load balancer
// SECURITY: Prevents header spoofing by only trusting forwarded headers from known proxies
func isTrustedProxy(ip string) bool {
	// Parse the IP
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// Trust localhost (127.0.0.0/8, ::1)
	if parsedIP.IsLoopback() {
		return true
	}

	// Trust private networks (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16)
	// This is safe for internal deployments behind a reverse proxy
	// For internet-facing deployments, configure specific proxy IPs
	if parsedIP.IsPrivate() {
		return true
	}

	// TODO: Add configuration option for custom trusted proxy CIDRs
	// For now, only trust localhost and private networks
	return false
}

// generateRequestID creates a unique request ID for tracing
// FEATURE #118: Request tracing for debugging and monitoring
func generateRequestID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes) // #nosec G104 -- error logged or non-critical
	return hex.EncodeToString(bytes)
}

// csrfProtect wraps handlers that modify state and require CSRF token validation
// SECURITY FIX LOW-1: Prevents Cross-Site Request Forgery attacks
func (s *Server) csrfProtect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only check CSRF for state-changing methods
		if r.Method == http.MethodPost || r.Method == http.MethodPut ||
			r.Method == http.MethodPatch || r.Method == http.MethodDelete {

			// Skip CSRF check if no token was generated (error during startup)
			if s.csrfToken == "" {
				next(w, r)
				return
			}

			// Get CSRF token from header
			clientToken := r.Header.Get("X-CSRF-Token")
			if clientToken == "" {
				writeError(w, r, http.StatusForbidden, "csrf_token_missing",
					"CSRF token required for state-changing requests. Include X-CSRF-Token header.", nil)
				return
			}

			// Constant-time comparison to prevent timing attacks
			if subtle.ConstantTimeCompare([]byte(clientToken), []byte(s.csrfToken)) != 1 {
				writeError(w, r, http.StatusForbidden, "csrf_token_invalid",
					"Invalid CSRF token", nil)
				return
			}
		}

		next(w, r)
	}
}

// addSecurityHeaders adds security headers to all HTTP responses
// SECURITY FIX #102: Comprehensive security headers to prevent web attacks
func addSecurityHeaders(w http.ResponseWriter, r *http.Request) {
	// Prevent MIME type sniffing
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// Prevent clickjacking attacks
	w.Header().Set("X-Frame-Options", "DENY")

	// Enable XSS protection (legacy, but still useful for older browsers)
	w.Header().Set("X-XSS-Protection", "1; mode=block")

	// Content Security Policy - restrict resource loading
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; "+
			"script-src 'self' 'unsafe-inline'; "+
			"style-src 'self' 'unsafe-inline'; "+
			"img-src 'self' data:; "+
			"font-src 'self'; "+
			"connect-src 'self'; "+
			"object-src 'none'; "+
			"base-uri 'self'; "+
			"form-action 'self'")

	// Only add HSTS if connection is over TLS
	if r.TLS != nil {
		w.Header().Set("Strict-Transport-Security",
			"max-age=31536000; includeSubDomains")
	}

	// Restrict browser features
	w.Header().Set("Permissions-Policy",
		"geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=()")

	// Control referrer information
	w.Header().Set("Referrer-Policy", "no-referrer")
}

// ErrorResponse represents a standardized API error response
// FEATURE #105: Consistent error format for all API endpoints
type ErrorResponse struct {
	Error     string        `json:"error"`                // Machine-readable error code
	Message   string        `json:"message"`              // Human-readable error message
	Details   []ErrorDetail `json:"details,omitempty"`    // Optional detailed error information
	RequestID string        `json:"request_id,omitempty"` // Optional request ID for tracing
	Timestamp time.Time     `json:"timestamp"`            // When the error occurred
	Path      string        `json:"path"`                 // Request path that caused the error
	Method    string        `json:"method"`               // HTTP method
}

// ErrorDetail provides detailed information about a specific error
type ErrorDetail struct {
	Field string `json:"field,omitempty"` // Field name that caused the error
	Issue string `json:"issue"`           // Description of the issue
	Value string `json:"value,omitempty"` // The value that caused the error (sanitized)
}

// validateAlertConfig validates alert configuration fields
// SECURITY FIX MEDIUM-3: Comprehensive input validation to prevent injection attacks
func validateAlertConfig(cfg AlertConfig) []ErrorDetail {
	var errors []ErrorDetail

	// Validate packet threshold (must be reasonable)
	if cfg.PacketsThreshold > 0 && cfg.PacketsThreshold > 1000000000 {
		errors = append(errors, ErrorDetail{
			Field: "packets_threshold",
			Issue: "threshold exceeds maximum allowed value of 1 billion",
			Value: fmt.Sprintf("%d", cfg.PacketsThreshold),
		})
	}

	// Validate webhook URL if provided
	if cfg.WebhookURL != "" {
		if len(cfg.WebhookURL) > 2048 {
			errors = append(errors, ErrorDetail{
				Field: "webhook_url",
				Issue: "URL exceeds maximum length of 2048 characters",
				Value: "[too long]",
			})
		}
		// Basic URL format validation
		if !strings.HasPrefix(cfg.WebhookURL, "http://") && !strings.HasPrefix(cfg.WebhookURL, "https://") {
			errors = append(errors, ErrorDetail{
				Field: "webhook_url",
				Issue: "URL must start with http:// or https://",
				Value: cfg.WebhookURL[:min(50, len(cfg.WebhookURL))],
			})
		}
	}

	return errors
}

// validateSimulationRequest validates simulation request fields
// SECURITY FIX MEDIUM-3: Prevent injection and resource exhaustion
func validateSimulationRequest(req SimulationRequest) []ErrorDetail {
	var errors []ErrorDetail

	// Validate interface name (alphanumeric + limited special chars only)
	if req.Interface != "" {
		if len(req.Interface) > 255 {
			errors = append(errors, ErrorDetail{
				Field: "interface",
				Issue: "interface name exceeds 255 characters",
				Value: req.Interface[:50],
			})
		}
		// Allow letters, numbers, dash, underscore, dot (common in interface names)
		for _, c := range req.Interface {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
				(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.') {
				errors = append(errors, ErrorDetail{
					Field: "interface",
					Issue: "interface name contains invalid characters (only alphanumeric, dash, underscore, dot allowed)",
					Value: req.Interface[:min(50, len(req.Interface))],
				})
				break
			}
		}
	}

	// Validate config path (prevent path traversal)
	if req.ConfigPath != "" {
		if strings.Contains(req.ConfigPath, "..") {
			errors = append(errors, ErrorDetail{
				Field: "config_path",
				Issue: "path traversal detected (.. not allowed)",
				Value: "[redacted]",
			})
		}
		if len(req.ConfigPath) > 4096 {
			errors = append(errors, ErrorDetail{
				Field: "config_path",
				Issue: "path exceeds maximum length of 4096 characters",
				Value: "[too long]",
			})
		}
	}

	// Validate config data size
	if req.ConfigData != "" {
		if len(req.ConfigData) > MaxRequestBodySize {
			errors = append(errors, ErrorDetail{
				Field: "config_data",
				Issue: fmt.Sprintf("config data exceeds maximum size of %d bytes", MaxRequestBodySize),
				Value: "[too large]",
			})
		}
	}

	return errors
}

// validateReplayRequest validates replay request fields
// SECURITY FIX MEDIUM-3: Prevent PCAP injection and resource exhaustion
func validateReplayRequest(req ReplayRequest) []ErrorDetail {
	var errors []ErrorDetail

	// Validate inline data size
	if len(req.InlineData) > MaxPCAPUploadSize*4/3 {
		errors = append(errors, ErrorDetail{
			Field: "data",
			Issue: fmt.Sprintf("inline data exceeds maximum size of %d bytes (100MB base64)", MaxPCAPUploadSize*4/3),
			Value: "[too large]",
		})
	}

	// Validate file path (prevent path traversal)
	if req.File != "" {
		if strings.Contains(req.File, "..") {
			errors = append(errors, ErrorDetail{
				Field: "file",
				Issue: "path traversal detected (.. not allowed)",
				Value: "[redacted]",
			})
		}
		if len(req.File) > 4096 {
			errors = append(errors, ErrorDetail{
				Field: "file",
				Issue: "file path exceeds maximum length of 4096 characters",
				Value: "[too long]",
			})
		}
	}

	// Validate loop milliseconds (prevent extreme values)
	if req.LoopMs < 0 {
		errors = append(errors, ErrorDetail{
			Field: "loop_ms",
			Issue: "loop_ms cannot be negative",
			Value: fmt.Sprintf("%d", req.LoopMs),
		})
	}
	if req.LoopMs > 86400000 { // Max 24 hours
		errors = append(errors, ErrorDetail{
			Field: "loop_ms",
			Issue: "loop_ms exceeds maximum of 24 hours (86400000ms)",
			Value: fmt.Sprintf("%d", req.LoopMs),
		})
	}

	// Validate scale (prevent extreme values)
	if req.Scale < 0 {
		errors = append(errors, ErrorDetail{
			Field: "scale",
			Issue: "scale cannot be negative",
			Value: fmt.Sprintf("%f", req.Scale),
		})
	}
	if req.Scale > 1000.0 {
		errors = append(errors, ErrorDetail{
			Field: "scale",
			Issue: "scale exceeds maximum of 1000x",
			Value: fmt.Sprintf("%f", req.Scale),
		})
	}

	return errors
}

// validateQueryParam validates a query parameter
// SECURITY FIX MEDIUM-3: Prevent injection via query parameters
func validateQueryParam(name, value string, allowedValues []string) *ErrorDetail {
	if value == "" {
		return nil
	}

	// Length check
	if len(value) > 1024 {
		return &ErrorDetail{
			Field: name,
			Issue: "parameter value exceeds maximum length of 1024 characters",
			Value: value[:50],
		}
	}

	// If allowed values specified, check against whitelist
	if len(allowedValues) > 0 {
		valid := false
		for _, allowed := range allowedValues {
			if value == allowed {
				valid = true
				break
			}
		}
		if !valid {
			return &ErrorDetail{
				Field: name,
				Issue: fmt.Sprintf("invalid value (allowed: %v)", allowedValues),
				Value: value[:min(50, len(value))],
			}
		}
	}

	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// recoverMiddleware recovers from panics in HTTP handlers to prevent server crashes
// SECURITY FIX #2.8.1: Add panic recovery to prevent single malformed request from crashing API
func (s *Server) recoverMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				requestID := r.Header.Get("X-Request-ID")
				// Log panic with stack trace
				log.Printf("[API] [%s] PANIC recovered: %v", requestID, err)
				log.Printf("[API] Stack trace: %s", debug.Stack())

				// Return 500 error to client
				writeError(w, r, http.StatusInternalServerError,
					"internal_server_error",
					"An internal error occurred. Please try again later.", nil)
			}
		}()
		next(w, r)
	}
}

// writeError writes a standardized error response
func writeError(w http.ResponseWriter, r *http.Request, status int, errorCode, message string, details []ErrorDetail) {
	response := ErrorResponse{
		Error:     errorCode,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
		Path:      r.URL.Path,
		Method:    r.Method,
	}

	// FEATURE #118: Include request ID in error logging
	requestID := r.Header.Get("X-Request-ID")
	if requestID != "" {
		log.Printf("[API] [%s] Error %d: %s - %s", requestID, status, errorCode, message)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response) // #nosec G104 -- error logged or non-critical
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
	StartedAt time.Time `json:"started_at,omitempty"`
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
	Addr        string
	MetricsAddr string
	Token       string
	Stack       *protocols.Stack
	Config      *config.Config
	ConfigPath  string
	Storage     *storage.Storage
	Interface   string
	Version     string
	Topology    Topology
	Alert       AlertConfig
	ApplyConfig func(*config.Config) error
	Replay      ReplayManager
}

// SimulationRequest represents a request to start a simulation
type SimulationRequest struct {
	Interface  string `json:"interface"`
	ConfigPath string `json:"config_path,omitempty"`
	ConfigData string `json:"config_data,omitempty"`
}

// SimulationStatus represents the current simulation status
type SimulationStatus struct {
	Running       bool      `json:"running"`
	Interface     string    `json:"interface,omitempty"`
	ConfigPath    string    `json:"config_path,omitempty"`
	ConfigName    string    `json:"config_name,omitempty"`
	DeviceCount   int       `json:"device_count"`
	StartedAt     time.Time `json:"started_at,omitempty"`
	UptimeSeconds float64   `json:"uptime_seconds"`
}

// DaemonController interface for daemon mode operations
type DaemonController interface {
	StartSimulation(req SimulationRequest) error
	StopSimulation() error
	GetStatus() SimulationStatus
}

// Server exposes the REST API, metrics endpoint, and Web UI.
type Server struct {
	cfg           ServerConfig
	httpServer    *http.Server
	metricsServer *http.Server
	alertStop     chan struct{}
	lastAlert     uint64
	alertMu       sync.RWMutex
	configMu      sync.RWMutex
	daemon        DaemonController // Optional: only set in daemon mode
	startTime     time.Time        // Track server start time for uptime
	rateLimiter   *RateLimiter     // FEATURE #104: Per-IP rate limiting
	csrfToken     string           // SECURITY FIX LOW-1: CSRF protection token
}

// generateCSRFToken generates a cryptographically secure random token
// SECURITY FIX LOW-1: CSRF protection
func generateCSRFToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// NewServer returns a configured API server.
func NewServer(cfg ServerConfig) *Server {
	// Generate CSRF token (ignore errors, fallback to empty which disables CSRF check)
	csrfToken, _ := generateCSRFToken()

	return &Server{
		cfg:         cfg,
		startTime:   time.Now(),
		rateLimiter: NewRateLimiter(DefaultRateLimit, DefaultBurst),
		csrfToken:   csrfToken,
	}
}

// Start boots the HTTP listeners.
// SECURITY FIX #98: Goroutines will properly exit when Shutdown() is called
// The ListenAndServe calls run in goroutines and will terminate when Shutdown()
// is invoked, preventing goroutine leaks. Always call Shutdown() to cleanup.
func (s *Server) Start() error {
	// In daemon mode, Stack and Config can be nil initially (set later when simulation starts)
	// In non-daemon mode, they must be set before Start()
	if s.daemon == nil && (s.cfg.Stack == nil || s.cfg.Config == nil) {
		return fmt.Errorf("api server requires stack and config references")
	}

	// SECURITY FIX #107: Warn if API is running without authentication
	if s.cfg.Token == "" && s.cfg.Addr != "" {
		log.Println("⚠️  WARNING: API server running WITHOUT authentication!")
		log.Println("    All endpoints are publicly accessible without any access control.")
		log.Println("    Set NIAC_API_TOKEN environment variable to enable authentication.")
		log.Println("    Example: export NIAC_API_TOKEN=$(openssl rand -base64 32)")
	}

	if s.cfg.Addr != "" {
		mux := http.NewServeMux()
		// SECURITY FIX #2.8.1: Wrap all handlers with panic recovery middleware
		// SECURITY FIX LOW-1: CSRF token endpoint for clients to retrieve token
		mux.HandleFunc("/api/v1/csrf-token", s.recoverMiddleware(s.auth(s.handleCSRFToken)))
		mux.HandleFunc("/api/v1/stats", s.recoverMiddleware(s.auth(s.handleStats)))
		mux.HandleFunc("/api/v1/devices", s.recoverMiddleware(s.auth(s.handleDevices)))
		mux.HandleFunc("/api/v1/history", s.recoverMiddleware(s.auth(s.handleHistory)))
		// SECURITY FIX LOW-1: Protect state-changing endpoints with CSRF
		mux.HandleFunc("/api/v1/config", s.recoverMiddleware(s.auth(s.csrfProtect(s.handleConfig))))
		mux.HandleFunc("/api/v1/replay", s.recoverMiddleware(s.auth(s.csrfProtect(s.handleReplay))))
		mux.HandleFunc("/api/v1/alerts", s.recoverMiddleware(s.auth(s.csrfProtect(s.handleAlerts))))
		mux.HandleFunc("/api/v1/files", s.recoverMiddleware(s.auth(s.handleFiles)))
		mux.HandleFunc("/api/v1/topology", s.recoverMiddleware(s.auth(s.handleTopology)))
		mux.HandleFunc("/api/v1/topology/export", s.recoverMiddleware(s.auth(s.handleTopologyExport)))
		mux.HandleFunc("/api/v1/errors", s.recoverMiddleware(s.auth(s.handleErrors)))
		mux.HandleFunc("/api/v1/interfaces", s.recoverMiddleware(s.auth(s.handleInterfaces)))
		mux.HandleFunc("/api/v1/runtime", s.recoverMiddleware(s.auth(s.handleRuntime)))
		mux.HandleFunc("/api/v1/simulation", s.recoverMiddleware(s.auth(s.handleSimulation)))
		mux.HandleFunc("/api/v1/version", s.recoverMiddleware(s.auth(s.handleVersion)))
		mux.HandleFunc("/api/v1/neighbors", s.recoverMiddleware(s.auth(s.handleNeighbors)))
		mux.HandleFunc("/metrics", s.recoverMiddleware(s.handleMetrics))
		mux.HandleFunc("/", s.recoverMiddleware(s.auth(s.serveSPA())))

		// SECURITY FIX #99: Add HTTP timeouts to prevent slowloris attacks
		s.httpServer = &http.Server{
			Addr:              s.cfg.Addr,
			Handler:           mux,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       60 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			MaxHeaderBytes:    1 << 20, // 1MB
		}

		go func() {
			if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("api server stopped: %v", err)
			}
		}()
	}

	if s.cfg.MetricsAddr != "" && s.cfg.MetricsAddr != s.cfg.Addr {
		mux := http.NewServeMux()
		mux.HandleFunc("/metrics", s.recoverMiddleware(s.handleMetrics))

		// SECURITY FIX #99: Add HTTP timeouts to metrics server too
		s.metricsServer = &http.Server{
			Addr:              s.cfg.MetricsAddr,
			Handler:           mux,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       60 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			MaxHeaderBytes:    1 << 20, // 1MB
		}

		go func() {
			if err := s.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("metrics server stopped: %v", err)
			}
		}()
	}

	// FEATURE #104: Start periodic cleanup of stale rate limiters
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			s.rateLimiter.CleanupStale()
		}
	}()

	s.updateAlertConfig(s.cfg.Alert)
	return nil
}

// Shutdown stops the HTTP listeners.
// Shutdown gracefully shuts down the API and metrics servers
// SECURITY FIX #98: Proper server shutdown to prevent goroutine leaks
func (s *Server) Shutdown(ctx context.Context) error {
	// Acquire lock before closing channel to prevent race with updateAlertConfig
	s.alertMu.Lock()
	if s.alertStop != nil {
		close(s.alertStop)
		s.alertStop = nil
	}
	s.alertMu.Unlock()

	var firstErr error

	// Shutdown metrics server first (less critical)
	if s.metricsServer != nil {
		if err := s.metricsServer.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down metrics server: %v", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	// Shutdown main HTTP server
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down HTTP server: %v", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	// Return first error encountered, if any
	return firstErr
}

// SetDaemonController sets the daemon controller (for daemon mode)
func (s *Server) SetDaemonController(daemon DaemonController) {
	s.daemon = daemon
}

// UpdateSimulation updates the server with simulation components (for daemon mode)
func (s *Server) UpdateSimulation(stack *protocols.Stack, cfg *config.Config, configPath string, iface string, replay ReplayManager) {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	s.cfg.Stack = stack
	s.cfg.Config = cfg
	s.cfg.ConfigPath = configPath
	s.cfg.Interface = iface
	s.cfg.Replay = replay
	s.cfg.Topology = BuildTopology(cfg)
}

// ClearSimulation clears simulation components (for daemon mode)
func (s *Server) ClearSimulation() {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	s.cfg.Stack = nil
	s.cfg.Config = nil
	s.cfg.Replay = nil
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// FEATURE #118: Generate unique request ID for tracing
		requestID := generateRequestID()
		r.Header.Set("X-Request-ID", requestID)
		w.Header().Set("X-Request-ID", requestID)

		// Add security headers to all responses
		addSecurityHeaders(w, r)

		// FEATURE #104: Apply rate limiting per IP address
		clientIP := getClientIP(r)
		limiter := s.rateLimiter.GetLimiter(clientIP)
		if !limiter.Allow() {
			// FEATURE #105: Use standardized error response
			writeError(w, r, http.StatusTooManyRequests, "rate_limit_exceeded",
				"Rate limit exceeded. Please try again later.", nil)
			log.Printf("[API] [%s] Rate limit exceeded for IP: %s", requestID, clientIP)
			return
		}

		if s.cfg.Token == "" {
			next(w, r)
			return
		}

		// Only accept Authorization header (not query parameters for security)
		token := r.Header.Get("Authorization")
		token = strings.TrimPrefix(token, "Bearer ")

		// SECURITY FIX #100: Use constant-time comparison to prevent timing attacks
		// Standard string comparison (!=) could leak token information via timing
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.Token)) != 1 {
			// FEATURE #105: Use standardized error response
			writeError(w, r, http.StatusUnauthorized, "unauthorized",
				"Invalid or missing authentication token", nil)
			log.Printf("[API] [%s] Unauthorized request from %s", requestID, clientIP)
			return
		}

		next(w, r)
	}
}

func (s *Server) serveSPA() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}

		requestPath := strings.TrimPrefix(r.URL.Path, "/")
		if requestPath == "" || strings.HasSuffix(r.URL.Path, "/") {
			requestPath = "index.html"
		}
		if strings.Contains(requestPath, "..") {
			http.NotFound(w, r)
			return
		}

		lookupPath := path.Join("ui", requestPath)
		data, err := fs.ReadFile(uiFS, lookupPath)
		if err != nil {
			data, err = fs.ReadFile(uiFS, path.Join("ui", "index.html"))
			if err != nil {
				http.NotFound(w, r)
				return
			}
			requestPath = "index.html"
		}

		if ctype := mime.TypeByExtension(filepath.Ext(requestPath)); ctype != "" {
			w.Header().Set("Content-Type", ctype)
		} else if strings.HasSuffix(requestPath, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		} else if strings.HasSuffix(requestPath, ".css") {
			w.Header().Set("Content-Type", "text/css")
		}

		http.ServeContent(w, r, requestPath, time.Time{}, bytes.NewReader(data))
	}
}

// handleCSRFToken returns the CSRF token for the client
// SECURITY FIX LOW-1: Clients must retrieve this token and include it in state-changing requests
func (s *Server) handleCSRFToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.writeJSON(w, map[string]string{
		"token": s.csrfToken,
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	s.configMu.RLock()
	stack := s.cfg.Stack
	cfg := s.cfg.Config
	s.configMu.RUnlock()

	if stack == nil {
		http.Error(w, "no simulation running", http.StatusServiceUnavailable)
		return
	}

	stats := stack.GetStats()
	deviceCount := 0
	if cfg != nil {
		deviceCount = len(cfg.Devices)
	}

	// FEATURE #119: Include goroutine count for debugging and monitoring
	goroutineCount := runtime.NumGoroutine()

	payload := map[string]interface{}{
		"timestamp":    time.Now().UTC(),
		"interface":    s.cfg.Interface,
		"version":      s.cfg.Version,
		"device_count": deviceCount,
		"goroutines":   goroutineCount, // FEATURE #119: Monitor goroutine count
		"stack": map[string]uint64{
			"packets_sent":     stats.PacketsSent,
			"packets_received": stats.PacketsReceived,
			"arp_requests":     stats.ARPRequests,
			"arp_replies":      stats.ARPReplies,
			"icmp_requests":    stats.ICMPRequests,
			"icmp_replies":     stats.ICMPReplies,
			"dns_queries":      stats.DNSQueries,
			"dhcp_requests":    stats.DHCPRequests,
			"snmp_queries":     stats.SNMPQueries,
			"errors":           stats.Errors,
		},
	}
	s.writeJSON(w, payload)
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	cfg := s.currentConfig()
	if cfg == nil {
		s.writeJSON(w, []map[string]interface{}{})
		return
	}

	devices := make([]map[string]interface{}, 0, len(cfg.Devices))
	for _, dev := range cfg.Devices {
		ips := make([]string, 0, len(dev.IPAddresses))
		for _, ip := range dev.IPAddresses {
			ips = append(ips, ip.String())
		}

		protos := make([]string, 0, 8)
		if dev.SNMPConfig.Community != "" || dev.SNMPConfig.WalkFile != "" {
			protos = append(protos, "SNMP")
		}
		if dev.DHCPConfig != nil {
			protos = append(protos, "DHCP")
		}
		if dev.DNSConfig != nil {
			protos = append(protos, "DNS")
		}
		if dev.HTTPConfig != nil {
			protos = append(protos, "HTTP")
		}
		if dev.FTPConfig != nil {
			protos = append(protos, "FTP")
		}
		if dev.LLDPConfig != nil && dev.LLDPConfig.Enabled {
			protos = append(protos, "LLDP")
		}
		if dev.CDPConfig != nil && dev.CDPConfig.Enabled {
			protos = append(protos, "CDP")
		}

		devices = append(devices, map[string]interface{}{
			"name":      dev.Name,
			"type":      dev.Type,
			"ips":       ips,
			"protocols": protos,
		})
	}
	s.writeJSON(w, devices)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Storage == nil {
		s.writeJSON(w, []storage.RunRecord{})
		return
	}
	history, err := s.cfg.Storage.ListRuns(20)
	if err != nil {
		// SECURITY FIX MEDIUM-6: Don't expose internal error details
		log.Printf("[API] Failed to list run history: %v", err)
		writeError(w, r, http.StatusInternalServerError, "storage_error",
			"Failed to retrieve run history", nil)
		return
	}
	s.writeJSON(w, history)
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleConfigGet(w, r)
	case http.MethodPut, http.MethodPatch, http.MethodPost:
		s.handleConfigUpdate(w, r)
	default:
		w.Header().Set("Allow", "GET, PUT, PATCH, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleConfigGet(w http.ResponseWriter, r *http.Request) {
	doc, status, err := s.readConfigDocument()
	if err != nil {
		// SECURITY FIX MEDIUM-6: Don't expose internal error details
		log.Printf("[API] Failed to read config: %v", err)
		writeError(w, r, status, "config_read_failed",
			"Failed to read configuration", nil)
		return
	}
	s.writeJSON(w, doc)
}

func (s *Server) handleConfigUpdate(w http.ResponseWriter, r *http.Request) {
	// SECURITY FIX #111: Enforce request body size limit
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

	if s.cfg.ConfigPath == "" {
		http.Error(w, "config path not available", http.StatusBadRequest)
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err.Error() == "http: request body too large" {
			writeError(w, r, http.StatusRequestEntityTooLarge, "request_too_large",
				fmt.Sprintf("Request body exceeds maximum size of %d bytes", MaxRequestBodySize), nil)
			return
		}
		// SECURITY FIX MEDIUM-6: Don't expose internal error details
		writeError(w, r, http.StatusBadRequest, "invalid_request",
			"Failed to parse request body", nil)
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		writeError(w, r, http.StatusBadRequest, "validation_failed",
			"Configuration content is required", nil)
		return
	}

	newCfg, err := config.LoadYAMLBytes([]byte(req.Content))
	if err != nil {
		// SECURITY FIX MEDIUM-6: Log details server-side, return generic message
		log.Printf("[API] Config validation failed: %v", err)
		writeError(w, r, http.StatusBadRequest, "config_invalid",
			"Configuration validation failed", nil)
		return
	}

	prevCfg := s.currentConfig()
	if s.cfg.ApplyConfig != nil {
		if err := s.cfg.ApplyConfig(newCfg); err != nil {
			// SECURITY FIX MEDIUM-6: Don't expose internal error details
			log.Printf("[API] Failed to apply config: %v", err)
			writeError(w, r, http.StatusInternalServerError, "config_apply_failed",
				"Failed to apply configuration", nil)
			return
		}
	}

	if err := s.writeConfigFile(req.Content); err != nil {
		if s.cfg.ApplyConfig != nil && prevCfg != nil {
			// Attempt rollback to previous config to avoid divergence.
			_ = s.cfg.ApplyConfig(prevCfg)
		}
		// SECURITY FIX MEDIUM-6: Don't expose file paths
		log.Printf("[API] Failed to write config file: %v", err)
		writeError(w, r, http.StatusInternalServerError, "config_write_failed",
			"Failed to save configuration", nil)
		return
	}

	s.replaceConfig(newCfg)

	doc, status, err := s.readConfigDocument()
	if err != nil {
		// SECURITY FIX MEDIUM-6: Don't expose internal error details
		log.Printf("[API] Failed to read updated config: %v", err)
		writeError(w, r, status, "config_read_failed",
			"Configuration updated but failed to retrieve", nil)
		return
	}
	s.writeJSON(w, doc)
}

func (s *Server) handleReplay(w http.ResponseWriter, r *http.Request) {
	// FEATURE #132: Graceful degradation when replay engine is unavailable
	if s.cfg.Replay == nil {
		writeError(w, r, http.StatusServiceUnavailable, "replay_unavailable",
			"PCAP replay functionality is not available in this mode. Start niac with a configuration to enable replay.", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.writeJSON(w, s.cfg.Replay.Status())
	case http.MethodPost:
		// SECURITY FIX #97: Enforce request body size limit for PCAP uploads
		r.Body = http.MaxBytesReader(w, r.Body, MaxPCAPUploadSize)

		var req ReplayRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if err.Error() == "http: request body too large" {
				writeError(w, r, http.StatusRequestEntityTooLarge, "request_too_large",
					"PCAP file too large (max 100MB)", nil)
				return
			}
			writeError(w, r, http.StatusBadRequest, "invalid_request",
				"Failed to parse request body", nil)
			return
		}

		// SECURITY FIX MEDIUM-3: Comprehensive validation
		if validationErrors := validateReplayRequest(req); len(validationErrors) > 0 {
			writeError(w, r, http.StatusBadRequest, "validation_failed",
				"Replay request validation failed", validationErrors)
			return
		}

		prepared, err := s.prepareReplayRequest(req)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "replay_preparation_failed",
				"Failed to prepare replay request", nil)
			return
		}
		state, err := s.cfg.Replay.Start(prepared)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "replay_start_failed",
				"Failed to start replay", nil)
			return
		}
		s.writeJSON(w, state)
	case http.MethodDelete:
		state, err := s.cfg.Replay.Stop()
		if err != nil {
			// SECURITY FIX MEDIUM-6: Don't expose internal error details
			log.Printf("[API] Failed to stop replay: %v", err)
			writeError(w, r, http.StatusInternalServerError, "replay_stop_failed",
				"Failed to stop replay", nil)
			return
		}
		s.writeJSON(w, state)
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.writeJSON(w, s.getAlertConfig())
	case http.MethodPut, http.MethodPost:
		// SECURITY FIX MEDIUM-3: Add request body size limit
		r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

		var req AlertConfig
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if err.Error() == "http: request body too large" {
				writeError(w, r, http.StatusRequestEntityTooLarge, "request_too_large",
					fmt.Sprintf("Request body exceeds maximum size of %d bytes", MaxRequestBodySize), nil)
				return
			}
			writeError(w, r, http.StatusBadRequest, "invalid_request",
				"Failed to parse request body", nil)
			return
		}

		// SECURITY FIX MEDIUM-3: Validate input fields
		if validationErrors := validateAlertConfig(req); len(validationErrors) > 0 {
			writeError(w, r, http.StatusBadRequest, "validation_failed",
				"Alert configuration validation failed", validationErrors)
			return
		}

		s.updateAlertConfig(req)
		s.writeJSON(w, s.getAlertConfig())
	default:
		w.Header().Set("Allow", "GET, PUT, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("kind")

	// SECURITY FIX MEDIUM-3: Validate query parameter
	allowedKinds := []string{"", "snmp", "config", "pcap", "walks", "pcaps"}
	if err := validateQueryParam("kind", kind, allowedKinds); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_parameter",
			"Invalid query parameter", []ErrorDetail{*err})
		return
	}

	entries, err := s.collectFiles(kind)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "file_collection_failed",
			"Failed to collect files", nil)
		return
	}
	s.writeJSON(w, entries)
}

func (s *Server) handleTopology(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, s.currentTopology())
}

func (s *Server) handleTopologyExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	// SECURITY FIX MEDIUM-3: Validate format parameter
	allowedFormats := []string{"json", "graphml", "dot"}
	if err := validateQueryParam("format", format, allowedFormats); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_parameter",
			"Invalid format parameter", []ErrorDetail{*err})
		return
	}

	topology := s.currentTopology()

	// Note: format is validated above, so only json/graphml/dot can reach here
	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=\"topology.json\"")
		s.writeJSON(w, topology)

	case "graphml":
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("Content-Disposition", "attachment; filename=\"topology.graphml\"")
		fmt.Fprint(w, topology.ExportGraphML())

	case "dot":
		w.Header().Set("Content-Type", "text/vnd.graphviz")
		w.Header().Set("Content-Disposition", "attachment; filename=\"topology.dot\"")
		fmt.Fprint(w, topology.ExportDOT())
	}
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, map[string]string{"version": s.cfg.Version})
}

func (s *Server) handleNeighbors(w http.ResponseWriter, r *http.Request) {
	neighbors := s.cfg.Stack.GetNeighbors()
	if neighbors == nil {
		neighbors = []protocols.NeighborRecord{}
	}
	s.writeJSON(w, neighbors)
}

func (s *Server) handleErrors(w http.ResponseWriter, r *http.Request) {
	s.configMu.RLock()
	stack := s.cfg.Stack
	s.configMu.RUnlock()

	if stack == nil {
		http.Error(w, "no simulation running", http.StatusServiceUnavailable)
		return
	}

	errorMgr := stack.GetErrorManager()
	if errorMgr == nil {
		http.Error(w, "error manager not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// List available error types and current active errors
		errorTypes := []map[string]string{
			{"type": "FCS Errors", "description": "Frame Check Sequence errors (0-100)"},
			{"type": "Packet Discards", "description": "Dropped packets (0-100)"},
			{"type": "Interface Errors", "description": "Generic interface errors (0-100)"},
			{"type": "High Utilization", "description": "Interface bandwidth saturation (0-100%)"},
			{"type": "High CPU", "description": "Device CPU load (0-100%)"},
			{"type": "High Memory", "description": "Device memory usage (0-100%)"},
			{"type": "High Disk", "description": "Device disk usage (0-100%)"},
		}

		activeErrors := errorMgr.GetAllStates()
		s.writeJSON(w, map[string]interface{}{
			"available_types": errorTypes,
			"active_errors":   activeErrors,
		})

	case http.MethodPost, http.MethodPut:
		// SECURITY FIX #111: Enforce request body size limit
		r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

		// Inject or update error
		var req struct {
			DeviceIP  string `json:"device_ip"`
			Interface string `json:"interface"`
			ErrorType string `json:"error_type"`
			Value     int    `json:"value"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// SECURITY FIX MEDIUM-6: Don't expose internal error details
			writeError(w, r, http.StatusBadRequest, "invalid_request",
				"Failed to parse request body", nil)
			return
		}

		// Validate inputs
		if req.DeviceIP == "" {
			writeError(w, r, http.StatusBadRequest, "validation_failed",
				"device_ip is required", nil)
			return
		}
		if req.Interface == "" {
			writeError(w, r, http.StatusBadRequest, "validation_failed",
				"interface is required", nil)
			return
		}
		if req.ErrorType == "" {
			writeError(w, r, http.StatusBadRequest, "validation_failed",
				"error_type is required", nil)
			return
		}
		if req.Value < 0 || req.Value > 100 {
			writeError(w, r, http.StatusBadRequest, "validation_failed",
				"value must be between 0 and 100", nil)
			return
		}

		// Inject error
		errorMgr.SetError(req.DeviceIP, req.Interface, errors.ErrorType(req.ErrorType), req.Value)

		s.writeJSON(w, map[string]interface{}{
			"success":    true,
			"message":    "error injected successfully",
			"device_ip":  req.DeviceIP,
			"interface":  req.Interface,
			"error_type": req.ErrorType,
			"value":      req.Value,
		})

	case http.MethodDelete:
		// Clear errors
		query := r.URL.Query()
		deviceIP := query.Get("device_ip")
		iface := query.Get("interface")

		if deviceIP == "" && iface == "" {
			// Clear all errors
			errorMgr.ClearAll()
			s.writeJSON(w, map[string]interface{}{
				"success": true,
				"message": "all errors cleared",
			})
		} else if deviceIP != "" && iface != "" {
			// Clear specific device/interface error
			errorMgr.ClearError(deviceIP, iface)
			s.writeJSON(w, map[string]interface{}{
				"success":   true,
				"message":   "error cleared",
				"device_ip": deviceIP,
				"interface": iface,
			})
		} else {
			http.Error(w, "both device_ip and interface are required, or omit both to clear all", http.StatusBadRequest)
		}

	default:
		w.Header().Set("Allow", "GET, POST, PUT, DELETE")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleInterfaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get available network interfaces from pcap
	ifaces, err := capture.GetAllInterfaces()
	if err != nil {
		// SECURITY FIX MEDIUM-6: Don't expose internal error details
		log.Printf("[API] Failed to list interfaces: %v", err)
		writeError(w, r, http.StatusInternalServerError, "interface_list_failed",
			"Failed to retrieve network interfaces", nil)
		return
	}

	type interfaceInfo struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Addresses   []string `json:"addresses"`
		Current     bool     `json:"current"`
	}

	result := make([]interfaceInfo, 0, len(ifaces))
	for _, iface := range ifaces {
		addrs := make([]string, 0, len(iface.Addresses))
		for _, addr := range iface.Addresses {
			addrs = append(addrs, addr.IP.String())
		}
		result = append(result, interfaceInfo{
			Name:        iface.Name,
			Description: iface.Description,
			Addresses:   addrs,
			Current:     iface.Name == s.cfg.Interface,
		})
	}

	s.writeJSON(w, map[string]interface{}{
		"interfaces":        result,
		"current_interface": s.cfg.Interface,
	})
}

func (s *Server) handleRuntime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if stack is available
	s.configMu.RLock()
	stack := s.cfg.Stack
	cfg := s.cfg.Config
	s.configMu.RUnlock()

	if stack == nil {
		http.Error(w, "no simulation running", http.StatusServiceUnavailable)
		return
	}

	stats := stack.GetStats()

	runtime := map[string]interface{}{
		"running":          true, // API server is running
		"interface":        s.cfg.Interface,
		"config_path":      s.cfg.ConfigPath,
		"version":          s.cfg.Version,
		"device_count":     0,
		"packets_sent":     stats.PacketsSent,
		"packets_received": stats.PacketsReceived,
		"uptime_seconds":   time.Since(s.startTime).Seconds(),
	}

	if cfg != nil {
		runtime["device_count"] = len(cfg.Devices)
		runtime["config_name"] = filepath.Base(s.cfg.ConfigPath)
	}

	s.writeJSON(w, runtime)
}

func (s *Server) handleSimulation(w http.ResponseWriter, r *http.Request) {
	if s.daemon == nil {
		http.Error(w, "Simulation control is only available in daemon mode. Start NIAC with 'niac daemon' command.", http.StatusNotImplemented)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get simulation status
		status := s.daemon.GetStatus()
		s.writeJSON(w, status)

	case http.MethodPost:
		// Start simulation
		// SECURITY FIX MEDIUM-3: Request body size limit
		r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

		var req SimulationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if err.Error() == "http: request body too large" {
				writeError(w, r, http.StatusRequestEntityTooLarge, "request_too_large",
					fmt.Sprintf("Request body exceeds maximum size of %d bytes", MaxRequestBodySize), nil)
				return
			}
			writeError(w, r, http.StatusBadRequest, "invalid_request",
				"Failed to parse request body", nil)
			return
		}

		// Validate required fields
		if req.Interface == "" {
			writeError(w, r, http.StatusBadRequest, "validation_failed",
				"Validation failed", []ErrorDetail{{Field: "interface", Issue: "interface is required"}})
			return
		}

		if req.ConfigPath == "" && req.ConfigData == "" {
			writeError(w, r, http.StatusBadRequest, "validation_failed",
				"Validation failed", []ErrorDetail{{Field: "config", Issue: "either config_path or config_data must be provided"}})
			return
		}

		// SECURITY FIX MEDIUM-3: Comprehensive input validation
		if validationErrors := validateSimulationRequest(req); len(validationErrors) > 0 {
			writeError(w, r, http.StatusBadRequest, "validation_failed",
				"Simulation request validation failed", validationErrors)
			return
		}

		if err := s.daemon.StartSimulation(req); err != nil {
			writeError(w, r, http.StatusInternalServerError, "simulation_start_failed",
				"Failed to start simulation", nil)
			return
		}

		status := s.daemon.GetStatus()
		w.WriteHeader(http.StatusCreated)
		s.writeJSON(w, status)

	case http.MethodDelete:
		// Stop simulation
		if err := s.daemon.StopSimulation(); err != nil {
			// SECURITY FIX MEDIUM-6: Don't expose internal error details
			log.Printf("[API] Failed to stop simulation: %v", err)
			writeError(w, r, http.StatusInternalServerError, "simulation_stop_failed",
				"Failed to stop simulation", nil)
			return
		}

		s.writeJSON(w, map[string]string{"status": "stopped"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	s.configMu.RLock()
	stack := s.cfg.Stack
	cfg := s.cfg.Config
	s.configMu.RUnlock()

	if stack == nil {
		http.Error(w, "no simulation running", http.StatusServiceUnavailable)
		return
	}

	stats := stack.GetStats()
	deviceCount := 0
	if cfg != nil {
		deviceCount = len(cfg.Devices)
	}

	// Get system metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	// Existing basic metrics
	fmt.Fprintf(w, "# HELP niac_packets_sent_total Total packets sent\n")
	fmt.Fprintf(w, "# TYPE niac_packets_sent_total counter\n")
	fmt.Fprintf(w, "niac_packets_sent_total %d\n", stats.PacketsSent)

	fmt.Fprintf(w, "# HELP niac_packets_received_total Total packets received\n")
	fmt.Fprintf(w, "# TYPE niac_packets_received_total counter\n")
	fmt.Fprintf(w, "niac_packets_received_total %d\n", stats.PacketsReceived)

	fmt.Fprintf(w, "# HELP niac_snmp_queries_total Total SNMP queries processed\n")
	fmt.Fprintf(w, "# TYPE niac_snmp_queries_total counter\n")
	fmt.Fprintf(w, "niac_snmp_queries_total %d\n", stats.SNMPQueries)

	fmt.Fprintf(w, "# HELP niac_errors_total Total errors\n")
	fmt.Fprintf(w, "# TYPE niac_errors_total counter\n")
	fmt.Fprintf(w, "niac_errors_total %d\n", stats.Errors)

	fmt.Fprintf(w, "# HELP niac_devices_total Number of simulated devices\n")
	fmt.Fprintf(w, "# TYPE niac_devices_total gauge\n")
	fmt.Fprintf(w, "niac_devices_total %d\n", deviceCount)

	// Protocol-specific metrics
	fmt.Fprintf(w, "# HELP niac_arp_requests_total Total ARP requests sent\n")
	fmt.Fprintf(w, "# TYPE niac_arp_requests_total counter\n")
	fmt.Fprintf(w, "niac_arp_requests_total %d\n", stats.ARPRequests)

	fmt.Fprintf(w, "# HELP niac_arp_replies_total Total ARP replies sent\n")
	fmt.Fprintf(w, "# TYPE niac_arp_replies_total counter\n")
	fmt.Fprintf(w, "niac_arp_replies_total %d\n", stats.ARPReplies)

	fmt.Fprintf(w, "# HELP niac_icmp_requests_total Total ICMP requests sent\n")
	fmt.Fprintf(w, "# TYPE niac_icmp_requests_total counter\n")
	fmt.Fprintf(w, "niac_icmp_requests_total %d\n", stats.ICMPRequests)

	fmt.Fprintf(w, "# HELP niac_icmp_replies_total Total ICMP replies sent\n")
	fmt.Fprintf(w, "# TYPE niac_icmp_replies_total counter\n")
	fmt.Fprintf(w, "niac_icmp_replies_total %d\n", stats.ICMPReplies)

	fmt.Fprintf(w, "# HELP niac_dns_queries_total Total DNS queries processed\n")
	fmt.Fprintf(w, "# TYPE niac_dns_queries_total counter\n")
	fmt.Fprintf(w, "niac_dns_queries_total %d\n", stats.DNSQueries)

	fmt.Fprintf(w, "# HELP niac_dhcp_requests_total Total DHCP requests processed\n")
	fmt.Fprintf(w, "# TYPE niac_dhcp_requests_total counter\n")
	fmt.Fprintf(w, "niac_dhcp_requests_total %d\n", stats.DHCPRequests)

	// System performance metrics
	fmt.Fprintf(w, "# HELP niac_uptime_seconds Server uptime in seconds\n")
	fmt.Fprintf(w, "# TYPE niac_uptime_seconds gauge\n")
	fmt.Fprintf(w, "niac_uptime_seconds %d\n", int64(time.Since(s.startTime).Seconds()))

	fmt.Fprintf(w, "# HELP niac_goroutines_total Number of goroutines\n")
	fmt.Fprintf(w, "# TYPE niac_goroutines_total gauge\n")
	fmt.Fprintf(w, "niac_goroutines_total %d\n", runtime.NumGoroutine())

	fmt.Fprintf(w, "# HELP niac_memory_usage_bytes Memory usage in bytes\n")
	fmt.Fprintf(w, "# TYPE niac_memory_usage_bytes gauge\n")
	fmt.Fprintf(w, "niac_memory_usage_bytes %d\n", memStats.Alloc)

	fmt.Fprintf(w, "# HELP niac_memory_sys_bytes Total memory obtained from OS in bytes\n")
	fmt.Fprintf(w, "# TYPE niac_memory_sys_bytes gauge\n")
	fmt.Fprintf(w, "niac_memory_sys_bytes %d\n", memStats.Sys)

	fmt.Fprintf(w, "# HELP niac_gc_runs_total Total number of GC runs\n")
	fmt.Fprintf(w, "# TYPE niac_gc_runs_total counter\n")
	fmt.Fprintf(w, "niac_gc_runs_total %d\n", memStats.NumGC)
}

func (s *Server) writeJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(payload)
}

func (s *Server) alertLoop(stop <-chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cfg := s.getAlertConfig()
			if cfg.PacketsThreshold == 0 {
				continue
			}

			s.configMu.RLock()
			stack := s.cfg.Stack
			s.configMu.RUnlock()

			if stack == nil {
				continue
			}

			stats := stack.GetStats()
			total := stats.PacketsSent + stats.PacketsReceived
			if total >= cfg.PacketsThreshold {
				s.alertMu.Lock()
				if total != s.lastAlert {
					s.lastAlert = total
					go s.sendAlert(total)
				}
				s.alertMu.Unlock()
			}
		case <-stop:
			return
		}
	}
}

func (s *Server) sendAlert(total uint64) {
	log.Printf("alert: packet threshold exceeded (total=%d)", total)
	cfg := s.getAlertConfig()
	if cfg.WebhookURL == "" {
		return
	}

	body, _ := json.Marshal(map[string]interface{}{
		"type":        "packet_threshold",
		"threshold":   cfg.PacketsThreshold,
		"total":       total,
		"interface":   s.cfg.Interface,
		"triggeredAt": time.Now().UTC(),
	})

	req, err := http.NewRequest(http.MethodPost, cfg.WebhookURL, strings.NewReader(string(body)))
	if err != nil {
		log.Printf("alert webhook error: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	if resp, err := client.Do(req); err != nil {
		log.Printf("alert webhook request failed: %v", err)
	} else {
		resp.Body.Close() // #nosec G104 -- error logged or non-critical
	}
}

// validatePCAPMagic validates that the file begins with a valid PCAP magic number
// SECURITY FIX LOW-2: Prevents processing of non-PCAP files that could exploit parser bugs
func validatePCAPMagic(data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("file too small to be a valid PCAP (< 4 bytes)")
	}

	// Check for valid PCAP magic numbers
	// 0xa1b2c3d4 = standard pcap (microsecond precision, big-endian)
	// 0xd4c3b2a1 = standard pcap (microsecond precision, little-endian)
	// 0xa1b23c4d = pcap with nanosecond precision (big-endian)
	// 0x4d3cb2a1 = pcap with nanosecond precision (little-endian)
	// 0x0a0d0d0a = pcapng format
	magic := uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])

	validMagics := []uint32{
		0xa1b2c3d4, // pcap microsecond BE
		0xd4c3b2a1, // pcap microsecond LE
		0xa1b23c4d, // pcap nanosecond BE
		0x4d3cb2a1, // pcap nanosecond LE
		0x0a0d0d0a, // pcapng
	}

	for _, validMagic := range validMagics {
		if magic == validMagic {
			return nil
		}
	}

	return fmt.Errorf("invalid PCAP magic number: 0x%08x (expected pcap or pcapng format)", magic)
}

func (s *Server) prepareReplayRequest(req ReplayRequest) (ReplayRequest, error) {
	if strings.TrimSpace(req.File) == "" && req.InlineData == "" {
		return req, fmt.Errorf("pcap file path or data is required")
	}

	if req.InlineData != "" {
		// SECURITY FIX #97: Additional check on base64 encoded data size
		// Base64 encoding increases size by ~4/3, so check before decode
		if len(req.InlineData) > MaxPCAPUploadSize*4/3 {
			return req, fmt.Errorf("PCAP data exceeds size limit (max 100MB)")
		}

		data, err := base64.StdEncoding.DecodeString(req.InlineData)
		if err != nil {
			return req, fmt.Errorf("decode replay data: %w", err)
		}

		// Double-check decoded size
		if len(data) > MaxPCAPUploadSize {
			return req, fmt.Errorf("decoded PCAP exceeds size limit (max 100MB)")
		}

		// SECURITY FIX LOW-2: Validate PCAP file magic number
		if err := validatePCAPMagic(data); err != nil {
			return req, fmt.Errorf("invalid PCAP file: %w", err)
		}

		path, err := s.writeUploadedFile(data)
		if err != nil {
			return req, err
		}
		req.File = path
		req.Uploaded = true
		req.InlineData = ""
		return req, nil
	}

	abs, err := filepath.Abs(req.File)
	if err != nil {
		return req, fmt.Errorf("resolve path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return req, fmt.Errorf("stat %s: %w", abs, err)
	}
	if info.IsDir() {
		return req, fmt.Errorf("%s is a directory", abs)
	}
	req.File = abs
	return req, nil
}

func (s *Server) writeUploadedFile(data []byte) (string, error) {
	dir := filepath.Join(os.TempDir(), "niac-replay")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("create upload dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "upload-*.pcap")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer tmp.Close() // #nosec G104 -- deferred close
	if _, err := tmp.Write(data); err != nil {
		os.Remove(tmp.Name()) // #nosec G104 -- error logged or non-critical
		return "", fmt.Errorf("write upload: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		os.Remove(tmp.Name()) // #nosec G104 -- error logged or non-critical
		return "", fmt.Errorf("sync upload: %w", err)
	}
	return tmp.Name(), nil
}

func (s *Server) getAlertConfig() AlertConfig {
	s.alertMu.RLock()
	defer s.alertMu.RUnlock()
	return s.cfg.Alert
}

func (s *Server) updateAlertConfig(cfg AlertConfig) {
	// SECURITY FIX #2.8.1: Prevent race condition on alertStop channel
	s.alertMu.Lock()
	defer s.alertMu.Unlock()

	// Stop existing alert loop if running
	if s.alertStop != nil {
		// Check if channel is already closed to prevent panic
		select {
		case <-s.alertStop:
			// Already closed
		default:
			// Safe to close
			close(s.alertStop)
		}
		s.alertStop = nil
	}

	s.cfg.Alert = cfg
	s.lastAlert = 0

	// Start new alert loop if threshold is set
	if cfg.PacketsThreshold > 0 {
		stopChan := make(chan struct{})
		s.alertStop = stopChan
		// Start goroutine while still holding lock to prevent race
		go s.alertLoop(stopChan)
	}
}

type configDocument struct {
	Path        string    `json:"path"`
	Filename    string    `json:"filename"`
	ModifiedAt  time.Time `json:"modified_at"`
	SizeBytes   int64     `json:"size_bytes"`
	DeviceCount int       `json:"device_count"`
	Content     string    `json:"content"`
}

func (s *Server) readConfigDocument() (*configDocument, int, error) {
	if s.cfg.ConfigPath == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("config path not available")
	}

	data, err := os.ReadFile(s.cfg.ConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, http.StatusNotFound, fmt.Errorf("config file %s not found", s.cfg.ConfigPath)
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("reading config: %w", err)
	}

	info, err := os.Stat(s.cfg.ConfigPath)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("stat config: %w", err)
	}

	cfg := s.currentConfig()
	deviceCount := 0
	if cfg != nil {
		deviceCount = len(cfg.Devices)
	}

	return &configDocument{
		Path:        s.cfg.ConfigPath,
		Filename:    filepath.Base(s.cfg.ConfigPath),
		ModifiedAt:  info.ModTime().UTC(),
		SizeBytes:   info.Size(),
		DeviceCount: deviceCount,
		Content:     string(data),
	}, http.StatusOK, nil
}

func (s *Server) writeConfigFile(content string) error {
	dir := filepath.Dir(s.cfg.ConfigPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".niac-config-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	defer os.Remove(tmpPath)

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close() // #nosec G104 -- error logged or non-critical
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close() // #nosec G104 -- error logged or non-critical
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}

	if err := os.Rename(tmpPath, s.cfg.ConfigPath); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}

func (s *Server) currentConfig() *config.Config {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.cfg.Config
}

func (s *Server) currentTopology() Topology {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.cfg.Topology
}

func (s *Server) replaceConfig(cfg *config.Config) {
	s.configMu.Lock()
	s.cfg.Config = cfg
	s.cfg.Topology = BuildTopology(cfg)
	s.configMu.Unlock()
}

func (s *Server) collectFiles(kind string) ([]FileEntry, error) {
	var root string
	var exts []string

	switch kind {
	case "walks":
		root = s.resolveIncludePath()
		exts = []string{".walk"}
	case "pcaps":
		if s.cfg.ConfigPath != "" {
			root = filepath.Dir(s.cfg.ConfigPath)
		}
		exts = []string{".pcap", ".pcapng"}
	default:
		return nil, fmt.Errorf("unsupported file kind: %s", kind)
	}

	if root == "" {
		return []FileEntry{}, nil
	}

	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return []FileEntry{}, nil
	}

	// Resolve canonical root path to prevent path traversal attacks
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve root path: %w", err)
	}
	// Resolve symlinks in root path
	rootReal, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		// If symlink resolution fails, use absolute path
		rootReal = rootAbs
	}

	var entries []FileEntry
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		match := false
		for _, allowed := range exts {
			if ext == allowed {
				match = true
				break
			}
		}
		if !match {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil
		}

		// SECURITY FIX #95: Validate path stays within root directory
		// Resolve symlinks to prevent symlink attacks (#96)
		realPath, err := filepath.EvalSymlinks(absPath)
		if err != nil {
			// If symlink resolution fails, skip this file
			return nil
		}

		// Ensure resolved path is within the allowed root directory
		if !strings.HasPrefix(realPath, rootReal+string(os.PathSeparator)) && realPath != rootReal {
			// Path is outside allowed directory, skip it
			return nil
		}

		entries = append(entries, FileEntry{
			Path:      absPath,
			Name:      filepath.Base(path),
			SizeBytes: info.Size(),
			Modified:  info.ModTime().UTC(),
		})
		if len(entries) >= 200 {
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil && err != filepath.SkipDir {
		return nil, err
	}
	return entries, nil
}

func (s *Server) resolveIncludePath() string {
	cfg := s.currentConfig()
	if cfg == nil || cfg.IncludePath == "" {
		return ""
	}

	includePath := cfg.IncludePath
	if !filepath.IsAbs(includePath) && s.cfg.ConfigPath != "" {
		includePath = filepath.Join(filepath.Dir(s.cfg.ConfigPath), includePath)
	}

	if abs, err := filepath.Abs(includePath); err == nil {
		return abs
	}
	return includePath
}
