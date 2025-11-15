package api

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/base64"
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

	// FEATURE #104: Rate limiting defaults
	// Allow 100 requests per second per IP with burst of 200
	DefaultRateLimit = 100
	DefaultBurst     = 200
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
func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.limiters[ip]
	if !exists {
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

// logRequest logs HTTP requests for debugging
func logRequest(r *http.Request) {
	log.Printf("[API] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
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
}

// NewServer returns a configured API server.
func NewServer(cfg ServerConfig) *Server {
	return &Server{
		cfg:         cfg,
		startTime:   time.Now(),
		rateLimiter: NewRateLimiter(DefaultRateLimit, DefaultBurst),
	}
}

// Start boots the HTTP listeners.
// SECURITY FIX #98: Goroutines will properly exit when Shutdown() is called
// The ListenAndServe calls run in goroutines and will terminate when Shutdown()
// is invoked, preventing goroutine leaks. Always call Shutdown() to cleanup.
func (s *Server) Start() error {
	if s.cfg.Stack == nil || s.cfg.Config == nil {
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
		mux.HandleFunc("/api/v1/stats", s.auth(s.handleStats))
		mux.HandleFunc("/api/v1/devices", s.auth(s.handleDevices))
		mux.HandleFunc("/api/v1/history", s.auth(s.handleHistory))
		mux.HandleFunc("/api/v1/config", s.auth(s.handleConfig))
		mux.HandleFunc("/api/v1/replay", s.auth(s.handleReplay))
		mux.HandleFunc("/api/v1/alerts", s.auth(s.handleAlerts))
		mux.HandleFunc("/api/v1/files", s.auth(s.handleFiles))
		mux.HandleFunc("/api/v1/topology", s.auth(s.handleTopology))
		mux.HandleFunc("/api/v1/topology/export", s.auth(s.handleTopologyExport))
		mux.HandleFunc("/api/v1/errors", s.auth(s.handleErrors))
		mux.HandleFunc("/api/v1/interfaces", s.auth(s.handleInterfaces))
		mux.HandleFunc("/api/v1/runtime", s.auth(s.handleRuntime))
		mux.HandleFunc("/api/v1/simulation", s.auth(s.handleSimulation))
		mux.HandleFunc("/api/v1/version", s.auth(s.handleVersion))
		mux.HandleFunc("/api/v1/neighbors", s.auth(s.handleNeighbors))
		mux.HandleFunc("/metrics", s.handleMetrics)
		mux.HandleFunc("/", s.auth(s.serveSPA()))

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
		mux.HandleFunc("/metrics", s.handleMetrics)

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
		// Add security headers to all responses
		addSecurityHeaders(w, r)

		// FEATURE #104: Apply rate limiting per IP address
		clientIP := getClientIP(r)
		limiter := s.rateLimiter.GetLimiter(clientIP)
		if !limiter.Allow() {
			// FEATURE #105: Use standardized error response
			writeError(w, r, http.StatusTooManyRequests, "rate_limit_exceeded",
				"Rate limit exceeded. Please try again later.", nil)
			log.Printf("[API] Rate limit exceeded for IP: %s", clientIP)
			return
		}

		if s.cfg.Token == "" {
			next(w, r)
			return
		}

		// Only accept Authorization header (not query parameters for security)
		token := r.Header.Get("Authorization")
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		}

		// SECURITY FIX #100: Use constant-time comparison to prevent timing attacks
		// Standard string comparison (!=) could leak token information via timing
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.Token)) != 1 {
			// FEATURE #105: Use standardized error response
			writeError(w, r, http.StatusUnauthorized, "unauthorized",
				"Invalid or missing authentication token", nil)
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
	payload := map[string]interface{}{
		"timestamp":    time.Now().UTC(),
		"interface":    s.cfg.Interface,
		"version":      s.cfg.Version,
		"device_count": deviceCount,
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), status)
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
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}

	newCfg, err := config.LoadYAMLBytes([]byte(req.Content))
	if err != nil {
		http.Error(w, fmt.Sprintf("config validation failed: %v", err), http.StatusBadRequest)
		return
	}

	prevCfg := s.currentConfig()
	if s.cfg.ApplyConfig != nil {
		if err := s.cfg.ApplyConfig(newCfg); err != nil {
			http.Error(w, fmt.Sprintf("failed to apply config: %v", err), http.StatusInternalServerError)
			return
		}
	}

	if err := s.writeConfigFile(req.Content); err != nil {
		if s.cfg.ApplyConfig != nil && prevCfg != nil {
			// Attempt rollback to previous config to avoid divergence.
			_ = s.cfg.ApplyConfig(prevCfg)
		}
		http.Error(w, fmt.Sprintf("failed to write config: %v", err), http.StatusInternalServerError)
		return
	}

	s.replaceConfig(newCfg)

	doc, status, err := s.readConfigDocument()
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	s.writeJSON(w, doc)
}

func (s *Server) handleReplay(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Replay == nil {
		http.Error(w, "replay control unavailable", http.StatusNotImplemented)
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
				http.Error(w, "PCAP file too large (max 100MB)", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
			return
		}
		prepared, err := s.prepareReplayRequest(req)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid replay request: %v", err), http.StatusBadRequest)
			return
		}
		state, err := s.cfg.Replay.Start(prepared)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.writeJSON(w, state)
	case http.MethodDelete:
		state, err := s.cfg.Replay.Stop()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
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
		var req AlertConfig
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
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
	entries, err := s.collectFiles(kind)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

	topology := s.currentTopology()

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

	default:
		http.Error(w, fmt.Sprintf("unsupported format: %s (supported: json, graphml, dot)", format), http.StatusBadRequest)
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
			http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
			return
		}

		// Validate inputs
		if req.DeviceIP == "" {
			http.Error(w, "device_ip is required", http.StatusBadRequest)
			return
		}
		if req.Interface == "" {
			http.Error(w, "interface is required", http.StatusBadRequest)
			return
		}
		if req.ErrorType == "" {
			http.Error(w, "error_type is required", http.StatusBadRequest)
			return
		}
		if req.Value < 0 || req.Value > 100 {
			http.Error(w, "value must be between 0 and 100", http.StatusBadRequest)
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
		http.Error(w, fmt.Sprintf("Failed to list interfaces: %v", err), http.StatusInternalServerError)
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
		// Add request body size limit
		r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

		var req SimulationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
			return
		}

		// Validate input
		if req.Interface == "" {
			http.Error(w, "interface is required", http.StatusBadRequest)
			return
		}

		if req.ConfigPath == "" && req.ConfigData == "" {
			http.Error(w, "either config_path or config_data must be provided", http.StatusBadRequest)
			return
		}

		if err := s.daemon.StartSimulation(req); err != nil {
			http.Error(w, fmt.Sprintf("failed to start simulation: %v", err), http.StatusInternalServerError)
			return
		}

		status := s.daemon.GetStatus()
		w.WriteHeader(http.StatusCreated)
		s.writeJSON(w, status)

	case http.MethodDelete:
		// Stop simulation
		if err := s.daemon.StopSimulation(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to stop simulation: %v", err), http.StatusInternalServerError)
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
		resp.Body.Close()
	}
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
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create upload dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "upload-*.pcap")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer tmp.Close()
	if _, err := tmp.Write(data); err != nil {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("write upload: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		os.Remove(tmp.Name())
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
	s.alertMu.Lock()
	if s.alertStop != nil {
		close(s.alertStop)
		s.alertStop = nil
	}
	s.cfg.Alert = cfg
	s.lastAlert = 0
	var stopChan chan struct{}
	if cfg.PacketsThreshold > 0 {
		stopChan = make(chan struct{})
		s.alertStop = stopChan
	}
	s.alertMu.Unlock()

	if stopChan != nil {
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
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".niac-config-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	defer os.Remove(tmpPath)

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Chmod(tmpPath, 0o644); err != nil {
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
