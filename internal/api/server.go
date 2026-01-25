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
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	apperr "github.com/krisarmstrong/niac-go/internal/apperr"
	"github.com/krisarmstrong/niac-go/internal/capture"
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

// rateLimiterEntry tracks a rate limiter with its last access time
// SECURITY FIX HIGH-2: Prevents unbounded memory growth.
type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter provides per-IP rate limiting for API requests
// FEATURE #104: Prevents brute force and DoS attacks.
type RateLimiter struct {
	limiters map[string]*rateLimiterEntry
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	logger   *slog.Logger
}

// NewRateLimiter creates a new rate limiter with the given rate and burst.
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rateLimiterEntry),
		rate:     r,
		burst:    b,
		logger:   slog.Default(),
	}
}

// evictOldestLimiter removes the oldest rate limiter entry (FIFO eviction).
// Must be called while holding rl.mu.
func (rl *RateLimiter) evictOldestLimiter() {
	var oldestIP string

	oldestTime := time.Now()
	for checkIP, checkEntry := range rl.limiters {
		if checkEntry.lastSeen.Before(oldestTime) {
			oldestTime = checkEntry.lastSeen
			oldestIP = checkIP
		}
	}

	if oldestIP == "" {
		return
	}

	delete(rl.limiters, oldestIP)
	rl.logger.Info(
		"[API] Rate limiter at capacity, evicted oldest IP",
		"capacity",
		MaxRateLimiterCount,
		"evictedIP",
		oldestIP,
	)
}

// GetLimiter returns the rate limiter for the given IP address
// SECURITY FIX #2.8.1: Enforce maximum count to prevent memory exhaustion.
func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.limiters[ip]
	if exists {
		entry.lastSeen = time.Now()

		return entry.limiter
	}

	// SECURITY FIX #2.8.1: Enforce max size limit
	if len(rl.limiters) >= MaxRateLimiterCount {
		rl.evictOldestLimiter()
	}

	entry = &rateLimiterEntry{
		limiter:  rate.NewLimiter(rl.rate, rl.burst),
		lastSeen: time.Now(),
	}
	rl.limiters[ip] = entry

	return entry.limiter
}

// CleanupStale removes limiters for IPs that haven't been seen recently
// SECURITY FIX HIGH-2: Aggressive cleanup to prevent memory exhaustion
// This prevents memory growth from storing limiters for millions of IPs over time.
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
		rl.logger.Info(
			"[API] Cleaned up stale rate limiters",
			"cleaned",
			count,
			"total",
			len(rl.limiters),
		)
	}
}

// sseTokenEntry represents a short-lived SSE authentication token.
// FIX #283: Used instead of exposing the main API token in URL query params.
type sseTokenEntry struct {
	expiresAt time.Time
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

// SECURITY FIX #156: Per-endpoint rate limiting middleware wrappers

// uploadRateLimit applies stricter rate limiting for upload endpoints.
func (s *Server) uploadRateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r, s.cfg.TrustedProxies)
		if !s.uploadLimiter.GetLimiter(clientIP).Allow() {
			writeError(w, r, http.StatusTooManyRequests, "upload_rate_limit_exceeded",
				"Upload rate limit exceeded. Please wait before uploading again.", nil)
			s.logger.Warn(
				"[API] Upload rate limit exceeded",
				"clientIP",
				clientIP,
				"path",
				r.URL.Path,
			)
			return
		}
		next(w, r)
	}
}

// writeRateLimit applies moderate rate limiting for write operations.
func (s *Server) writeRateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only apply to mutating methods
		if r.Method == http.MethodPost || r.Method == http.MethodPut ||
			r.Method == http.MethodPatch || r.Method == http.MethodDelete {
			clientIP := getClientIP(r, s.cfg.TrustedProxies)
			if !s.writeLimiter.GetLimiter(clientIP).Allow() {
				writeError(w, r, http.StatusTooManyRequests, "write_rate_limit_exceeded",
					"Write rate limit exceeded. Please wait before making more changes.", nil)
				s.logger.Warn(
					"[API] Write rate limit exceeded",
					"clientIP",
					clientIP,
					"path",
					r.URL.Path,
				)
				return
			}
		}
		next(w, r)
	}
}

// walkRateLimit applies rate limiting for walk file operations.
func (s *Server) walkRateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r, s.cfg.TrustedProxies)
		if !s.walkLimiter.GetLimiter(clientIP).Allow() {
			writeError(w, r, http.StatusTooManyRequests, "walk_rate_limit_exceeded",
				"Walk file operation rate limit exceeded. Please wait before trying again.", nil)
			s.logger.Warn(
				"[API] Walk rate limit exceeded",
				"clientIP",
				clientIP,
				"path",
				r.URL.Path,
			)
			return
		}
		next(w, r)
	}
}

// parseValidIP extracts and validates an IP from a string.
// Returns the trimmed IP string if valid, empty string otherwise.
func parseValidIP(rawIP string) string {
	trimmedIP := strings.TrimSpace(rawIP)
	if net.ParseIP(trimmedIP) != nil {
		return trimmedIP
	}

	return ""
}

// extractXForwardedForIP extracts the original client IP from X-Forwarded-For header.
// Returns empty string if header is missing or contains invalid IP.
func extractXForwardedForIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		return ""
	}

	// Take the first IP in the chain (leftmost = original client)
	firstIP, _, _ := strings.Cut(xff, ",")

	return parseValidIP(firstIP)
}

// extractXRealIP extracts the client IP from X-Real-IP header.
// Returns empty string if header is missing or contains invalid IP.
func extractXRealIP(r *http.Request) string {
	xri := r.Header.Get("X-Real-IP")
	if xri == "" {
		return ""
	}

	return parseValidIP(xri)
}

// getClientIP extracts the real client IP from the request.
// FIX #277: Accepts trusted CIDRs for proxy trust decisions.
func getClientIP(r *http.Request, trustedCIDRs []*net.IPNet) string {
	// Get the direct connection IP first
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteIP = r.RemoteAddr
	}

	// SECURITY: Only trust X-Forwarded-For/X-Real-IP if coming from trusted proxies
	if !isTrustedProxy(remoteIP, trustedCIDRs) {
		return remoteIP
	}

	// Check X-Forwarded-For header (for proxies/load balancers)
	if ip := extractXForwardedForIP(r); ip != "" {
		return ip
	}

	// Check X-Real-IP header
	if ip := extractXRealIP(r); ip != "" {
		return ip
	}

	// Use direct connection IP (forwarded headers invalid or missing)
	return remoteIP
}

// isTrustedProxy checks if an IP is from a trusted proxy/load balancer.
// FIX #277: Uses configurable CIDRs when available, falls back to localhost+private.
func isTrustedProxy(ip string, trustedCIDRs []*net.IPNet) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// If custom trusted CIDRs are configured, use only those
	if len(trustedCIDRs) > 0 {
		for _, cidr := range trustedCIDRs {
			if cidr.Contains(parsedIP) {
				return true
			}
		}
		return false
	}

	// Default: only trust loopback when no CIDRs configured
	return parsedIP.IsLoopback()
}

// generateRequestID creates a unique request ID for tracing
// FEATURE #118: Request tracing for debugging and monitoring.
func generateRequestID() string {
	bytes := make([]byte, requestIDBytes)
	if _, err := rand.Read(bytes); err != nil {
		slog.Warn("crypto/rand read failed for request ID", "error", err)
	}

	return hex.EncodeToString(bytes)
}

// csrfProtect wraps handlers that modify state and require CSRF token validation
// SECURITY FIX LOW-1: Prevents Cross-Site Request Forgery attacks.
func (s *Server) csrfProtect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only check CSRF for state-changing methods
		if r.Method == http.MethodPost || r.Method == http.MethodPut ||
			r.Method == http.MethodPatch || r.Method == http.MethodDelete {
			s.csrfMu.RLock()
			currentToken := s.csrfToken
			prevToken := s.csrfPrevToken
			s.csrfMu.RUnlock()

			// FIX #273: Block requests if CSRF token unavailable (don't silently bypass)
			if currentToken == "" {
				writeError(w, r, http.StatusServiceUnavailable, "csrf_unavailable",
					"CSRF protection is unavailable (token generation failed at startup)", nil)

				return
			}

			// Get CSRF token from header
			clientToken := r.Header.Get("X-Csrf-Token")
			if clientToken == "" {
				writeError(
					w,
					r,
					http.StatusForbidden,
					"csrf_token_missing",
					"CSRF token required for state-changing requests. Include X-CSRF-Token header.",
					nil,
				)

				return
			}

			// FIX #293: Accept both current and previous token during rotation window
			currentMatch := subtle.ConstantTimeCompare([]byte(clientToken), []byte(currentToken)) == 1
			prevMatch := prevToken != "" &&
				subtle.ConstantTimeCompare([]byte(clientToken), []byte(prevToken)) == 1

			if !currentMatch && !prevMatch {
				writeError(w, r, http.StatusForbidden, "csrf_token_invalid",
					"Invalid CSRF token", nil)

				return
			}
		}

		next(w, r)
	}
}

// addSecurityHeaders adds security headers to all HTTP responses
// SECURITY FIX #102: Comprehensive security headers to prevent web attacks.
func addSecurityHeaders(w http.ResponseWriter, r *http.Request) {
	// Prevent MIME type sniffing
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// Prevent clickjacking attacks
	w.Header().Set("X-Frame-Options", "DENY")

	// Enable XSS protection (legacy, but still useful for older browsers)
	w.Header().Set("X-XSS-Protection", "1; mode=block")

	// FIX #279: Content Security Policy - removed unsafe-inline for scripts
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; "+
			"script-src 'self'; "+
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; "+
			"img-src 'self' data:; "+
			"font-src 'self' https://fonts.gstatic.com; "+
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

// addCORSHeaders adds Cross-Origin Resource Sharing headers.
// FIX #267: Enables cross-origin requests for development and production.
func (s *Server) addCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return
	}

	if len(s.cfg.CORSAllowOrigins) == 0 {
		// Default: allow same-origin only (no CORS headers needed)
		return
	}

	isWildcard := false
	allowed := false
	for _, o := range s.cfg.CORSAllowOrigins {
		if o == "*" {
			isWildcard = true
			allowed = true
			break
		}
		if o == origin {
			allowed = true
			break
		}
	}

	if !allowed {
		return
	}

	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Csrf-Token, Accept")
	w.Header().Set("Access-Control-Max-Age", "86400")

	// SECURITY: Wildcard origins must NOT include credentials
	if isWildcard {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	} else {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
}

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

// validateAlertConfig validates alert configuration fields
// SECURITY FIX MEDIUM-3: Comprehensive input validation to prevent injection attacks.
func validateAlertConfig(cfg AlertConfig) []ErrorDetail {
	var errors []ErrorDetail

	// Validate packet threshold (must be reasonable)
	if cfg.PacketsThreshold > 0 && cfg.PacketsThreshold > 1000000000 {
		errors = append(errors, ErrorDetail{
			Field: "packets_threshold",
			Issue: "threshold exceeds maximum allowed value of 1 billion",
			Value: strconv.FormatUint(cfg.PacketsThreshold, 10),
		})
	}

	// Validate webhook URL if provided
	// SECURITY FIX #158: Enhanced SSRF protection for webhook URLs
	if cfg.WebhookURL != "" {
		if len(cfg.WebhookURL) > maxURLLength {
			errors = append(errors, ErrorDetail{
				Field: "webhook_url",
				Issue: "URL exceeds maximum length of 2048 characters",
				Value: "[too long]",
			})
		}
		// Basic URL format validation
		if !strings.HasPrefix(cfg.WebhookURL, "http://") &&
			!strings.HasPrefix(cfg.WebhookURL, "https://") {
			errors = append(errors, ErrorDetail{
				Field: "webhook_url",
				Issue: "URL must start with http:// or https://",
				Value: cfg.WebhookURL[:min(truncateErrorValue, len(cfg.WebhookURL))],
			})
		}
		// SSRF protection: check for internal/private addresses
		err := validateWebhookURLSSRF(cfg.WebhookURL)
		if err != nil {
			errors = append(errors, ErrorDetail{
				Field: "webhook_url",
				Issue: err.Error(),
				Value: "[redacted]",
			})
		}
	}

	return errors
}

// normalizeAndParseIP parses an IP address string, handling various edge cases.
// SECURITY FIX #168: Handle IPv6 zone identifiers, mapped addresses, and decimal notation.
func normalizeAndParseIP(host string) net.IP {
	// Strip IPv6 zone identifier (e.g., "::1%eth0" -> "::1")
	if idx := strings.Index(host, "%"); idx != -1 {
		host = host[:idx]
	}

	// Try standard parsing first
	ip := net.ParseIP(host)
	if ip != nil {
		return ip
	}

	// Try parsing as decimal IPv4 (e.g., "2130706433" = 127.0.0.1)
	if decimal, err := strconv.ParseUint(host, 10, 32); err == nil {
		return net.IPv4(
			byte(decimal>>bitShift24),
			byte(decimal>>bitShift16),
			byte(decimal>>bitShift8),
			byte(decimal),
		)
	}

	return nil
}

// isIPv6MappedLoopback checks if an IPv6 address is a mapped IPv4 loopback.
// SECURITY FIX #168: Detect ::ffff:127.0.0.1 and similar mapped loopback addresses.
func isIPv6MappedLoopback(ip net.IP) bool {
	// Check if it's an IPv6-mapped IPv4 address
	if ip4 := ip.To4(); ip4 != nil && len(ip) == net.IPv6len {
		// This is an IPv6-mapped IPv4 address, check the IPv4 part
		return ip4.IsLoopback()
	}
	return false
}

// isBlockedHostname checks if a hostname is in the blocked list.
func isBlockedHostname(host string) bool {
	// Blocked hosts for SSRF protection.
	blockedHosts := []string{
		"localhost",
		"localhost.localdomain",
		"127.0.0.1",
		"::1",
		"0.0.0.0",
		"0",
		"[::1]",
	}

	// Metadata service hosts to block.
	metadataHosts := []string{
		"169.254.169.254", // AWS/GCP/Azure metadata service
		"metadata.google.internal",
		"metadata.goog",
	}

	lowerHost := strings.ToLower(host)

	if slices.Contains(blockedHosts, lowerHost) {
		return true
	}

	// Check for short-form localhost variations
	if strings.HasPrefix(lowerHost, "127.") || lowerHost == "127" {
		return true
	}

	return slices.Contains(metadataHosts, lowerHost)
}

// isBlockedIP checks if an IP address should be blocked for SSRF.
func isBlockedIP(ip net.IP) error {
	if ip.IsLoopback() {
		return errors.New("loopback addresses not allowed")
	}

	if isIPv6MappedLoopback(ip) {
		return errors.New("loopback addresses not allowed")
	}

	if ip.IsPrivate() {
		return errors.New("private network addresses not allowed")
	}

	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return errors.New("link-local addresses not allowed")
	}

	if ip.IsUnspecified() {
		return errors.New("unspecified addresses not allowed")
	}

	// Block 169.254.0.0/16 (link-local) explicitly
	if ip4 := ip.To4(); ip4 != nil && ip4[0] == 169 && ip4[1] == 254 {
		return errors.New("link-local addresses not allowed")
	}

	return nil
}

// isBlockedIPv6Mapped checks for IPv6-mapped private addresses.
func isBlockedIPv6Mapped(ip net.IP) error {
	ip4 := ip.To4()
	if ip4 == nil || len(ip) != net.IPv6len {
		return nil
	}

	if ip4.IsPrivate() || ip4.IsLoopback() || ip4.IsLinkLocalUnicast() {
		return errors.New("private/loopback addresses not allowed (IPv6-mapped)")
	}

	return nil
}

// validateWebhookURLSSRF validates a webhook URL to prevent SSRF attacks.
// SECURITY FIX #158, #168: Prevents requests to internal/private networks.
func validateWebhookURLSSRF(urlStr string) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return errors.New("invalid URL format")
	}

	host := parsedURL.Hostname()
	if host == "" {
		return errors.New("URL must have a hostname")
	}

	if isBlockedHostname(host) {
		return errors.New("blocked hostname")
	}

	ip := normalizeAndParseIP(host)
	if ip == nil {
		// FIX #313: Hostname - resolve and validate all resolved IPs
		addrs, lookupErr := net.DefaultResolver.LookupIPAddr(context.Background(), host)
		if lookupErr != nil {
			return fmt.Errorf("cannot resolve hostname: %s", host)
		}

		for _, addr := range addrs {
			if blockedErr := isBlockedIP(addr.IP); blockedErr != nil {
				return fmt.Errorf("hostname resolves to blocked address: %w", blockedErr)
			}

			if mappedErr := isBlockedIPv6Mapped(addr.IP); mappedErr != nil {
				return fmt.Errorf("hostname resolves to blocked address: %w", mappedErr)
			}
		}

		return nil
	}

	if err = isBlockedIP(ip); err != nil {
		return err
	}

	return isBlockedIPv6Mapped(ip)
}

// isValidInterfaceChar checks if a character is valid in an interface name.
func isValidInterfaceChar(c rune) bool {
	isLowerLetter := c >= 'a' && c <= 'z'
	isUpperLetter := c >= 'A' && c <= 'Z'
	isDigit := c >= '0' && c <= '9'
	isAllowedSpecial := c == '-' || c == '_' || c == '.'

	return isLowerLetter || isUpperLetter || isDigit || isAllowedSpecial
}

// validateInterfaceName validates interface name characters and length.
func validateInterfaceName(iface string) *ErrorDetail {
	if iface == "" {
		return nil
	}

	if len(iface) > maxInterfaceNameLen {
		return &ErrorDetail{
			Field: "interface",
			Issue: "interface name exceeds 255 characters",
			Value: iface[:truncateErrorValue],
		}
	}

	for _, c := range iface {
		if !isValidInterfaceChar(c) {
			return &ErrorDetail{
				Field: "interface",
				Issue: "interface name contains invalid characters (only alphanumeric, dash, underscore, dot allowed)",
				Value: iface[:min(truncateErrorValue, len(iface))],
			}
		}
	}

	return nil
}

// validateConfigPath checks for path traversal and length.
func validateConfigPath(path string) []ErrorDetail {
	if path == "" {
		return nil
	}

	var errs []ErrorDetail

	if strings.Contains(path, "..") {
		errs = append(errs, ErrorDetail{
			Field: "config_path",
			Issue: "path traversal detected (.. not allowed)",
			Value: "[redacted]",
		})
	}

	if len(path) > maxPathLength {
		errs = append(errs, ErrorDetail{
			Field: "config_path",
			Issue: "path exceeds maximum length of 4096 characters",
			Value: "[too long]",
		})
	}

	return errs
}

// validateSimulationRequest validates simulation request fields
// SECURITY FIX MEDIUM-3: Prevent injection and resource exhaustion.
func validateSimulationRequest(req SimulationRequest) []ErrorDetail {
	var errors []ErrorDetail

	if err := validateInterfaceName(req.Interface); err != nil {
		errors = append(errors, *err)
	}

	errors = append(errors, validateConfigPath(req.ConfigPath)...)

	if req.ConfigData != "" && len(req.ConfigData) > MaxRequestBodySize {
		errors = append(errors, ErrorDetail{
			Field: "config_data",
			Issue: fmt.Sprintf("config data exceeds maximum size of %d bytes", MaxRequestBodySize),
			Value: "[too large]",
		})
	}

	return errors
}

// validateReplayRequest validates replay request fields
// SECURITY FIX MEDIUM-3: Prevent PCAP injection and resource exhaustion.
func validateReplayRequest(req ReplayRequest) []ErrorDetail {
	var errors []ErrorDetail

	// Validate inline data size
	if len(req.InlineData) > MaxPCAPUploadSize*4/base64Ratio {
		errors = append(errors, ErrorDetail{
			Field: "data",
			Issue: fmt.Sprintf(
				"inline data exceeds maximum size of %d bytes (100MB base64)",
				MaxPCAPUploadSize*4/base64Ratio,
			),
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

		if len(req.File) > maxPathLength {
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
			Value: strconv.Itoa(req.LoopMs),
		})
	}

	if req.LoopMs > maxLoopMs { // Max 24 hours
		errors = append(errors, ErrorDetail{
			Field: "loop_ms",
			Issue: "loop_ms exceeds maximum of 24 hours (86400000ms)",
			Value: strconv.Itoa(req.LoopMs),
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

	if req.Scale > maxScaleFactor {
		errors = append(errors, ErrorDetail{
			Field: "scale",
			Issue: "scale exceeds maximum of 1000x",
			Value: fmt.Sprintf("%f", req.Scale),
		})
	}

	return errors
}

// validateQueryParam validates a query parameter
// SECURITY FIX MEDIUM-3: Prevent injection via query parameters.
func validateQueryParam(name, value string, allowedValues []string) *ErrorDetail {
	if value == "" {
		return nil
	}

	// Length check
	if len(value) > maxQueryParamLen {
		return &ErrorDetail{
			Field: name,
			Issue: "parameter value exceeds maximum length of 1024 characters",
			Value: value[:truncateErrorValue],
		}
	}

	// If allowed values specified, check against whitelist
	if len(allowedValues) > 0 {
		if !slices.Contains(allowedValues, value) {
			return &ErrorDetail{
				Field: name,
				Issue: fmt.Sprintf("invalid value (allowed: %v)", allowedValues),
				Value: value[:min(truncateErrorValue, len(value))],
			}
		}
	}

	return nil
}

// recoverMiddleware recovers from panics in HTTP handlers to prevent server crashes
// SECURITY FIX #2.8.1: Add panic recovery to prevent single malformed request from crashing API.
func (s *Server) recoverMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				requestID := r.Header.Get("X-Request-ID")
				// Log panic with stack trace
				s.logger.Error("[API] PANIC recovered", "requestID", requestID, "error", err)
				s.logger.Error("[API] Stack trace", "stack", string(debug.Stack()))

				// Return 500 error to client
				writeError(w, r, http.StatusInternalServerError,
					"internal_server_error",
					"An internal error occurred. Please try again later.", nil)
			}
		}()

		next(w, r)
	}
}

// writeError writes a standardized error response.
func writeError(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	errorCode, message string,
	details []ErrorDetail,
) {
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
		logger := slog.Default()
		logger.Error(
			"[API] Error response",
			"requestID",
			requestID,
			"status",
			status,
			"errorCode",
			errorCode,
			"message",
			message,
		)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response) // HTTP write errors are non-critical
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

// generateCSRFToken generates a cryptographically secure random token
// SECURITY FIX LOW-1: CSRF protection.
func generateCSRFToken() (string, error) {
	bytes := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}

// registerAPIRoutes registers all API endpoints on the provided mux.
func (s *Server) registerAPIRoutes(mux *http.ServeMux) {
	// SECURITY FIX #2.8.1: Wrap all handlers with panic recovery middleware
	// SECURITY FIX LOW-1: CSRF token endpoint for clients to retrieve token
	mux.HandleFunc("/api/v1/csrf-token", s.recoverMiddleware(s.auth(s.handleCSRFToken)))
	mux.HandleFunc("/api/v1/stats", s.recoverMiddleware(s.auth(s.handleStats)))
	mux.HandleFunc("/api/v1/devices", s.recoverMiddleware(s.auth(s.handleDevices)))
	mux.HandleFunc("/api/v1/history", s.recoverMiddleware(s.auth(s.handleHistory)))

	// SECURITY FIX LOW-1: Protect state-changing endpoints with CSRF
	// SECURITY FIX #156: Apply write rate limiting to state-changing endpoints
	s.registerWriteProtectedRoutes(mux)
	s.registerReadOnlyRoutes(mux)
	s.registerWalkRoutes(mux)
	s.registerPcapRoutes(mux)
	s.registerTemplateRoutes(mux)
	s.registerDebugRoutes(mux)
	s.registerSSERoutes(mux)

	mux.HandleFunc("/metrics", s.recoverMiddleware(s.auth(s.handleMetrics)))
	mux.HandleFunc("/", s.recoverMiddleware(s.auth(s.serveSPA())))
}

// registerWriteProtectedRoutes registers routes that require write protection (CSRF + rate limiting).
func (s *Server) registerWriteProtectedRoutes(mux *http.ServeMux) {
	mux.HandleFunc(
		"/api/v1/config",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleConfig)))),
	)
	mux.HandleFunc(
		"/api/v1/config/devices",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleDevicesV2)))),
	)
	mux.HandleFunc(
		"/api/v1/config/devices/",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleDevicesV2)))),
	)
	mux.HandleFunc(
		"/api/v1/replay",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleReplay)))),
	)
	mux.HandleFunc(
		"/api/v1/alerts",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleAlerts)))),
	)
}

// registerReadOnlyRoutes registers routes that only require authentication.
func (s *Server) registerReadOnlyRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/config/schema", s.recoverMiddleware(s.auth(s.handleConfigSchema)))
	mux.HandleFunc("/api/v1/files", s.recoverMiddleware(s.auth(s.handleFiles)))
	mux.HandleFunc("/api/v1/topology", s.recoverMiddleware(s.auth(s.handleTopology)))
	mux.HandleFunc("/api/v1/topology/export", s.recoverMiddleware(s.auth(s.handleTopologyExport)))
	// FIX #291: Split /api/v1/errors - GET is read-only, POST/DELETE need write protection
	mux.HandleFunc("/api/v1/errors", s.recoverMiddleware(s.auth(s.handleErrorsRouter)))
	mux.HandleFunc("/api/v1/interfaces", s.recoverMiddleware(s.auth(s.handleInterfaces)))
	mux.HandleFunc("/api/v1/runtime", s.recoverMiddleware(s.auth(s.handleRuntime)))
	mux.HandleFunc(
		"/api/v1/simulation",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleSimulation)))),
	)
	mux.HandleFunc("/api/v1/version", s.recoverMiddleware(s.auth(s.handleVersion)))
	mux.HandleFunc("/api/v1/neighbors", s.recoverMiddleware(s.auth(s.handleNeighbors)))
}

// registerWalkRoutes registers SNMP walk file validation endpoints.
func (s *Server) registerWalkRoutes(mux *http.ServeMux) {
	// SECURITY FIX #156: Apply walk-specific rate limiting
	mux.HandleFunc(
		"/api/v1/walk/validate",
		s.recoverMiddleware(s.auth(s.walkRateLimit(s.csrfProtect(s.handleWalkValidation)))),
	)
	mux.HandleFunc(
		"/api/v1/walk/fix",
		s.recoverMiddleware(s.auth(s.walkRateLimit(s.csrfProtect(s.handleWalkValidation)))),
	)
	mux.HandleFunc(
		"/api/v1/walk/list",
		s.recoverMiddleware(s.auth(s.walkRateLimit(s.handleWalkList))),
	)
	mux.HandleFunc(
		"/api/v1/walk/validate-all",
		s.recoverMiddleware(s.auth(s.walkRateLimit(s.csrfProtect(s.handleWalkBatchValidate)))),
	)
}

// registerPcapRoutes registers PCAP analysis endpoints.
func (s *Server) registerPcapRoutes(mux *http.ServeMux) {
	// SECURITY FIX #156: Apply upload-specific rate limiting for uploads
	mux.HandleFunc(
		"/api/v1/pcap/upload",
		s.recoverMiddleware(s.auth(s.uploadRateLimit(s.csrfProtect(s.handlePcapUpload)))),
	)
	mux.HandleFunc("/api/v1/pcap/", s.recoverMiddleware(s.auth(s.handlePcapAnalysis)))
}

// registerSSERoutes registers Server-Sent Events endpoints for real-time streaming.
func (s *Server) registerSSERoutes(mux *http.ServeMux) {
	// FIX #283: SSE token endpoint for short-lived authentication tokens
	mux.HandleFunc("/api/v1/stream/token", s.recoverMiddleware(s.auth(s.handleSSEToken)))
	mux.HandleFunc("/api/v1/stream/packets", s.recoverMiddleware(s.auth(s.handleSSEPackets)))
	mux.HandleFunc("/api/v1/stream/logs", s.recoverMiddleware(s.auth(s.handleSSELogs)))
	mux.HandleFunc("/api/v1/stream/stats", s.recoverMiddleware(s.auth(s.handleSSEStats)))
	mux.HandleFunc("/api/v1/stream/status", s.recoverMiddleware(s.auth(s.handleSSEStatus)))
}

// registerTemplateRoutes registers template management endpoints.
// FIX #263: Implements backend for template CRUD operations.
func (s *Server) registerTemplateRoutes(mux *http.ServeMux) {
	mux.HandleFunc(
		"/api/v1/templates",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleTemplates)))),
	)
	mux.HandleFunc(
		"/api/v1/templates/use",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleTemplateUse)))),
	)
	mux.HandleFunc(
		"/api/v1/templates/",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleTemplateByName)))),
	)
}

// registerDebugRoutes registers protocol debug level endpoints.
// FIX #264: Implements backend for debug level management.
func (s *Server) registerDebugRoutes(mux *http.ServeMux) {
	mux.HandleFunc(
		"/api/v1/debug/levels",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleDebugLevels)))),
	)
	mux.HandleFunc(
		"/api/v1/debug/levels/reset",
		s.recoverMiddleware(s.auth(s.writeRateLimit(s.csrfProtect(s.handleDebugLevelsReset)))),
	)
}

// newSecureHTTPServer creates an HTTP server with security timeouts configured.
func newSecureHTTPServer(addr string, handler http.Handler) *http.Server {
	// SECURITY FIX #99: Add HTTP timeouts to prevent slowloris attacks
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       httpReadTimeout * time.Second,
		WriteTimeout:      httpWriteTimeout * time.Second,
		IdleTimeout:       httpIdleTimeout * time.Second,
		ReadHeaderTimeout: httpReadHeaderTimeout * time.Second,
		MaxHeaderBytes:    MaxHeaderBytesSize, // 64KB - SECURITY FIX #282
	}
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

// cleanupSSETokens removes expired SSE tokens from the [sync.Map].
func (s *Server) cleanupSSETokens() {
	now := time.Now()
	s.sseTokens.Range(func(key, value any) bool {
		entry, ok := value.(*sseTokenEntry)
		if !ok || !now.Before(entry.expiresAt) {
			s.sseTokens.Delete(key)
		}
		return true
	})
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

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// FEATURE #118: Generate unique request ID for tracing
		requestID := generateRequestID()
		r.Header.Set("X-Request-ID", requestID)
		w.Header().Set("X-Request-ID", requestID)

		// FIX #267: Add CORS headers
		s.addCORSHeaders(w, r)

		// Handle CORS preflight
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Add security headers to all responses
		addSecurityHeaders(w, r)

		// FEATURE #104: Apply rate limiting per IP address
		clientIP := getClientIP(r, s.cfg.TrustedProxies)

		limiter := s.rateLimiter.GetLimiter(clientIP)
		if !limiter.Allow() {
			writeError(w, r, http.StatusTooManyRequests, "rate_limit_exceeded",
				"Rate limit exceeded. Please try again later.", nil)
			s.logger.Warn("[API] Rate limit exceeded", "requestID", requestID, "clientIP", clientIP)

			return
		}

		if s.cfg.Token == "" {
			w.Header().Set("X-Auth-Mode", "none")
			next(w, r)

			return
		}

		// Accept Authorization header
		token := r.Header.Get("Authorization")
		token = strings.TrimPrefix(token, "Bearer ")

		// FIX #283: For SSE endpoints, only accept short-lived SSE tokens (no main token fallback)
		if token == "" && strings.HasPrefix(r.URL.Path, "/api/v1/stream/") {
			if s.trySSEToken(w, r, next) {
				return
			}
		}

		// SECURITY FIX #100: Use constant-time comparison to prevent timing attacks
		// Standard string comparison (!=) could leak token information via timing
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.Token)) != 1 {
			// FEATURE #105: Use standardized error response
			writeError(w, r, http.StatusUnauthorized, "unauthorized",
				"Invalid or missing authentication token", nil)
			s.logger.Warn(
				"[API] Unauthorized request",
				"requestID",
				requestID,
				"clientIP",
				clientIP,
			)

			return
		}

		next(w, r)
	}
}

// trySSEToken attempts to validate a short-lived SSE token from the query string.
// Returns true if the token was valid and the request was served, false otherwise.
func (s *Server) trySSEToken(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) bool {
	queryToken := r.URL.Query().Get("token")
	if queryToken == "" {
		return false
	}

	entry, ok := s.sseTokens.LoadAndDelete(queryToken)
	if !ok {
		return false
	}

	sseEntry, entryOK := entry.(*sseTokenEntry)
	if !entryOK || !time.Now().Before(sseEntry.expiresAt) {
		return false
	}

	next(w, r)

	return true
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

// handleCSRFToken returns the CSRF token for the client.
// FIX #293: Rotates the CSRF token if it has expired (> 1 hour).
// SECURITY FIX LOW-1: Clients must retrieve this token and include it in state-changing requests.
func (s *Server) handleCSRFToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	// FIX #293: Check if token needs rotation
	s.csrfMu.Lock()
	if time.Now().After(s.csrfExpiry) {
		newToken, err := generateCSRFToken()
		if err == nil {
			s.csrfPrevToken = s.csrfToken
			s.csrfToken = newToken
			s.csrfExpiry = time.Now().Add(1 * time.Hour)
			s.logger.Info("[API] CSRF token rotated")
		}
	}
	token := s.csrfToken
	s.csrfMu.Unlock()

	s.writeJSON(w, map[string]string{
		"token": token,
	})
}

// sseTokenExpiry is the validity duration for short-lived SSE tokens.
const sseTokenExpiry = 60 * time.Second

// handleSSEToken generates a short-lived, one-time-use token for SSE connections.
// FIX #283: Prevents exposing the main API token in URL query parameters.
func (s *Server) handleSSEToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	tokenBytes := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(tokenBytes); err != nil {
		writeError(w, r, http.StatusInternalServerError, "token_generation_failed",
			"Failed to generate SSE token", nil)

		return
	}

	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Store with expiry
	s.sseTokens.Store(token, &sseTokenEntry{
		expiresAt: time.Now().Add(sseTokenExpiry),
	})

	s.writeJSON(w, map[string]string{
		"token": token,
	})
}

func (s *Server) handleStats(w http.ResponseWriter, _ *http.Request) {
	// SECURITY FIX #161: Thread-safe access to all config fields
	s.configMu.RLock()
	stack := s.cfg.Stack
	cfg := s.cfg.Config
	iface := s.cfg.Interface
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

	payload := map[string]any{
		"timestamp":    time.Now().UTC(),
		"interface":    iface,
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

// getDeviceProtocols extracts the list of enabled protocols for a device.
func getDeviceProtocols(dev *config.Device) []string {
	protos := make([]string, 0, protocolCapacity)

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

	return protos
}

// ipAddressesToStrings converts a slice of [net.IP] to string representations.
func ipAddressesToStrings(ips []net.IP) []string {
	result := make([]string, 0, len(ips))
	for _, ip := range ips {
		result = append(result, ip.String())
	}

	return result
}

func (s *Server) handleDevices(w http.ResponseWriter, _ *http.Request) {
	cfg := s.currentConfig()
	if cfg == nil {
		s.writeJSON(w, []map[string]any{})

		return
	}

	devices := make([]map[string]any, 0, len(cfg.Devices))
	for i := range cfg.Devices {
		dev := &cfg.Devices[i]
		devices = append(devices, map[string]any{
			"name":      dev.Name,
			"type":      dev.Type,
			"ips":       ipAddressesToStrings(dev.IPAddresses),
			"protocols": getDeviceProtocols(dev),
		})
	}

	s.writeJSON(w, devices)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Storage == nil {
		s.writeJSON(w, []storage.RunRecord{})

		return
	}

	history, err := s.cfg.Storage.ListRuns(historyListLimit)
	if err != nil {
		// SECURITY FIX MEDIUM-6: Don't expose internal error details
		s.logger.Error("[API] Failed to list run history", "error", err)
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
		s.logger.Error("[API] Failed to read config", "error", err)
		writeError(w, r, status, "config_read_failed",
			"Failed to read configuration", nil)

		return
	}

	s.writeJSON(w, doc)
}

func (s *Server) handleConfigUpdate(w http.ResponseWriter, r *http.Request) {
	// SECURITY FIX #111: Enforce request body size limit
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

	// SECURITY FIX #161: Thread-safe access to ConfigPath
	if s.configPath() == "" {
		http.Error(w, "config path not available", http.StatusBadRequest)

		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err.Error() == ErrMsgRequestBodyTooLarge {
			writeError(
				w,
				r,
				http.StatusRequestEntityTooLarge,
				"request_too_large",
				fmt.Sprintf(
					"Request body exceeds maximum size of %d bytes",
					MaxRequestBodySize,
				),
				nil,
			)

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
		s.logger.Error("[API] Config validation failed", "error", err)
		writeError(w, r, http.StatusBadRequest, "config_invalid",
			"Configuration validation failed", nil)

		return
	}

	prevCfg := s.currentConfig()
	if s.cfg.ApplyConfig != nil {
		applyErr := s.cfg.ApplyConfig(newCfg)
		if applyErr != nil {
			// SECURITY FIX MEDIUM-6: Don't expose internal error details
			s.logger.Error("[API] Failed to apply config", "error", applyErr)
			writeError(w, r, http.StatusInternalServerError, "config_apply_failed",
				"Failed to apply configuration", nil)

			return
		}
	}

	writeErr := s.writeConfigFile(req.Content)
	if writeErr != nil {
		if s.cfg.ApplyConfig != nil && prevCfg != nil {
			// Attempt rollback to previous config to avoid divergence.
			if rollbackErr := s.cfg.ApplyConfig(prevCfg); rollbackErr != nil {
				s.logger.Error("[API] CRITICAL: config rollback failed", "error", rollbackErr)
			}
		}
		// SECURITY FIX MEDIUM-6: Don't expose file paths
		s.logger.Error("[API] Failed to write config file", "error", writeErr)
		writeError(w, r, http.StatusInternalServerError, "config_write_failed",
			"Failed to save configuration", nil)

		return
	}

	s.replaceConfig(newCfg)

	doc, status, err := s.readConfigDocument()
	if err != nil {
		// SECURITY FIX MEDIUM-6: Don't expose internal error details
		s.logger.Error("[API] Failed to read updated config", "error", err)
		writeError(w, r, status, "config_read_failed",
			"Configuration updated but failed to retrieve", nil)

		return
	}

	s.writeJSON(w, doc)
}

func (s *Server) handleReplay(w http.ResponseWriter, r *http.Request) {
	// FEATURE #132: Graceful degradation when replay engine is unavailable
	if s.cfg.Replay == nil {
		writeError(
			w,
			r,
			http.StatusServiceUnavailable,
			"replay_unavailable",
			"PCAP replay functionality is not available in this mode. Start niac with a configuration to enable replay.",
			nil,
		)

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
			if err.Error() == ErrMsgRequestBodyTooLarge {
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
			s.logger.Error("[API] Failed to stop replay", "error", err)
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
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			if err.Error() == ErrMsgRequestBodyTooLarge {
				writeError(
					w,
					r,
					http.StatusRequestEntityTooLarge,
					"request_too_large",
					fmt.Sprintf(
						"Request body exceeds maximum size of %d bytes",
						MaxRequestBodySize,
					),
					nil,
				)

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

func (s *Server) handleTopology(w http.ResponseWriter, _ *http.Request) {
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
		_, _ = fmt.Fprint(w, topology.ExportGraphML())

	case "dot":
		w.Header().Set("Content-Type", "text/vnd.graphviz")
		w.Header().Set("Content-Disposition", "attachment; filename=\"topology.dot\"")
		_, _ = fmt.Fprint(w, topology.ExportDOT())
	}
}

func (s *Server) handleVersion(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, map[string]string{"version": s.cfg.Version})
}

func (s *Server) handleNeighbors(w http.ResponseWriter, _ *http.Request) {
	// SECURITY FIX #161: Thread-safe access to Stack
	stack := s.currentStack()
	if stack == nil {
		http.Error(w, "no simulation running", http.StatusServiceUnavailable)

		return
	}

	neighbors := stack.GetNeighbors()
	if neighbors == nil {
		neighbors = []protocols.NeighborRecord{}
	}

	s.writeJSON(w, neighbors)
}

// errorInjectionRequest represents a request to inject an error.
type errorInjectionRequest struct {
	DeviceIP  string `json:"device_ip"`
	Interface string `json:"interface"`
	ErrorType string `json:"error_type"`
	Value     int    `json:"value"`
}

// getKnownErrorTypes returns the set of valid error types for injection.
func getKnownErrorTypes() map[string]bool {
	return map[string]bool{
		"FCS Errors":       true,
		"Packet Discards":  true,
		"Interface Errors": true,
		"High Utilization": true,
		"High CPU":         true,
		"High Memory":      true,
		"High Disk":        true,
	}
}

// validateErrorInjectionRequest validates the error injection request fields.
func (req *errorInjectionRequest) validate(w http.ResponseWriter, r *http.Request) bool {
	if req.DeviceIP == "" {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "device_ip is required", nil)
		return false
	}

	// Validate DeviceIP is a valid IP address
	if net.ParseIP(req.DeviceIP) == nil {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "device_ip must be a valid IP address", nil)
		return false
	}

	if req.Interface == "" {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "interface is required", nil)
		return false
	}

	if req.ErrorType == "" {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "error_type is required", nil)
		return false
	}

	// Validate ErrorType against known types
	if !getKnownErrorTypes()[req.ErrorType] {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "unknown error_type", nil)
		return false
	}

	if req.Value < 0 || req.Value > 100 {
		writeError(
			w,
			r,
			http.StatusBadRequest,
			"validation_failed",
			"value must be between 0 and 100",
			nil,
		)
		return false
	}

	return true
}

// availableErrorTypes returns the list of available error types for injection.
func availableErrorTypes() []map[string]string {
	return []map[string]string{
		{"type": "FCS Errors", "description": "Frame Check Sequence errors (0-100)"},
		{"type": "Packet Discards", "description": "Dropped packets (0-100)"},
		{"type": "Interface Errors", "description": "Generic interface errors (0-100)"},
		{"type": "High Utilization", "description": "Interface bandwidth saturation (0-100%)"},
		{"type": "High CPU", "description": "Device CPU load (0-100%)"},
		{"type": "High Memory", "description": "Device memory usage (0-100%)"},
		{"type": "High Disk", "description": "Device disk usage (0-100%)"},
	}
}

// handleErrorsRouter routes /api/v1/errors - GET is read-only, mutations need write protection.
// FIX #291: Prevents write rate limiting on read-only GET /errors endpoint.
func (s *Server) handleErrorsRouter(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Read-only: no write rate limit or CSRF needed
		s.handleErrors(w, r)

		return
	}

	// Mutating methods: apply write rate limit and CSRF protection
	s.writeRateLimit(s.csrfProtect(s.handleErrors))(w, r)
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
		s.writeJSON(w, map[string]any{
			"available_types": availableErrorTypes(),
			"active_errors":   errorMgr.GetAllStates(),
		})

	case http.MethodPost, http.MethodPut:
		s.handleErrorInjection(w, r, errorMgr)

	case http.MethodDelete:
		s.handleErrorClear(w, r, errorMgr)

	default:
		w.Header().Set("Allow", "GET, POST, PUT, DELETE")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleErrorInjection handles POST/PUT requests to inject errors.
func (s *Server) handleErrorInjection(
	w http.ResponseWriter,
	r *http.Request,
	errorMgr *apperr.StateManager,
) {
	// SECURITY FIX #111: Enforce request body size limit
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

	var req errorInjectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// SECURITY FIX MEDIUM-6: Don't expose internal error details
		writeError(
			w,
			r,
			http.StatusBadRequest,
			"invalid_request",
			"Failed to parse request body",
			nil,
		)
		return
	}

	if !req.validate(w, r) {
		return
	}

	errorMgr.SetError(req.DeviceIP, req.Interface, apperr.ErrorType(req.ErrorType), req.Value)

	s.writeJSON(w, map[string]any{
		"success":    true,
		"message":    "error injected successfully",
		"device_ip":  req.DeviceIP,
		"interface":  req.Interface,
		"error_type": req.ErrorType,
		"value":      req.Value,
	})
}

// handleErrorClear handles DELETE requests to clear errors.
func (s *Server) handleErrorClear(
	w http.ResponseWriter,
	r *http.Request,
	errorMgr *apperr.StateManager,
) {
	query := r.URL.Query()
	deviceIP := query.Get("device_ip")
	iface := query.Get("interface")

	switch {
	case deviceIP == "" && iface == "":
		errorMgr.ClearAll()
		s.writeJSON(w, map[string]any{"success": true, "message": "all errors cleared"})
	case deviceIP != "" && iface != "":
		errorMgr.ClearError(deviceIP, iface)
		s.writeJSON(
			w,
			map[string]any{
				"success":   true,
				"message":   "error cleared",
				"device_ip": deviceIP,
				"interface": iface,
			},
		)
	default:
		http.Error(
			w,
			"both device_ip and interface are required, or omit both to clear all",
			http.StatusBadRequest,
		)
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
		s.logger.Error("[API] Failed to list interfaces", "error", err)
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

	// SECURITY FIX #161: Thread-safe access to Interface
	currentIface := s.currentInterface()

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
			Current:     iface.Name == currentIface,
		})
	}

	s.writeJSON(w, map[string]any{
		"interfaces":        result,
		"current_interface": currentIface,
	})
}

func (s *Server) handleRuntime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	// SECURITY FIX #161: Thread-safe access to all config fields
	s.configMu.RLock()
	stack := s.cfg.Stack
	cfg := s.cfg.Config
	iface := s.cfg.Interface
	cfgPath := s.cfg.ConfigPath
	s.configMu.RUnlock()

	if stack == nil {
		http.Error(w, "no simulation running", http.StatusServiceUnavailable)

		return
	}

	stats := stack.GetStats()

	runtime := map[string]any{
		"running":          true, // API server is running
		"interface":        iface,
		"config_path":      filepath.Base(cfgPath),
		"version":          s.cfg.Version,
		"device_count":     0,
		"packets_sent":     stats.PacketsSent,
		"packets_received": stats.PacketsReceived,
		"uptime_seconds":   time.Since(s.startTime).Seconds(),
	}

	if cfg != nil {
		runtime["device_count"] = len(cfg.Devices)
		runtime["config_name"] = filepath.Base(cfgPath)
	}

	s.writeJSON(w, runtime)
}

func (s *Server) handleSimulation(w http.ResponseWriter, r *http.Request) {
	if s.daemon == nil {
		http.Error(
			w,
			"Simulation control is only available in daemon mode. Start NIAC with 'niac daemon' command.",
			http.StatusNotImplemented,
		)
		return
	}

	switch r.Method {
	case http.MethodGet:
		status := s.daemon.GetStatus()
		s.writeJSON(w, status)
	case http.MethodPost:
		s.handleSimulationStart(w, r)
	case http.MethodDelete:
		s.handleSimulationStop(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSimulationStart processes POST requests to start a simulation.
func (s *Server) handleSimulationStart(w http.ResponseWriter, r *http.Request) {
	// SECURITY FIX MEDIUM-3: Request body size limit
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

	var req SimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err.Error() == ErrMsgRequestBodyTooLarge {
			writeError(w, r, http.StatusRequestEntityTooLarge, "request_too_large",
				fmt.Sprintf("Request body exceeds maximum size of %d bytes", MaxRequestBodySize), nil)
			return
		}
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Failed to parse request body", nil)
		return
	}

	if err := validateSimulationStartRequest(req); err != nil {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "Validation failed", err)
		return
	}

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

	w.WriteHeader(http.StatusCreated)
	s.writeJSON(w, s.daemon.GetStatus())
}

// handleSimulationStop processes DELETE requests to stop a simulation.
func (s *Server) handleSimulationStop(w http.ResponseWriter, r *http.Request) {
	if err := s.daemon.StopSimulation(); err != nil {
		// SECURITY FIX MEDIUM-6: Don't expose internal error details
		s.logger.Error("[API] Failed to stop simulation", "error", err)
		writeError(w, r, http.StatusInternalServerError, "simulation_stop_failed",
			"Failed to stop simulation", nil)
		return
	}
	s.writeJSON(w, map[string]string{"status": "stopped"})
}

// validateSimulationStartRequest validates required fields for simulation start.
func validateSimulationStartRequest(req SimulationRequest) []ErrorDetail {
	var errors []ErrorDetail
	if req.Interface == "" {
		errors = append(errors, ErrorDetail{Field: "interface", Issue: "interface is required"})
	}
	if req.ConfigPath == "" && req.ConfigData == "" {
		errors = append(errors, ErrorDetail{
			Field: "config",
			Issue: "either config_path or config_data must be provided",
		})
	}
	return errors
}

// writePrometheusMetric writes a single Prometheus metric with help text, type, and value.
func writePrometheusMetric(w io.Writer, name, help, metricType string, value any) {
	_, _ = fmt.Fprintf(w, "# HELP %s %s\n", name, help)
	_, _ = fmt.Fprintf(w, "# TYPE %s %s\n", name, metricType)
	_, _ = fmt.Fprintf(w, "%s %v\n", name, value)
}

// writeBasicMetrics writes packet and error metrics.
func writeBasicMetrics(w io.Writer, stats *protocols.Statistics, deviceCount int) {
	writePrometheusMetric(
		w,
		"niac_packets_sent_total",
		"Total packets sent",
		"counter",
		stats.PacketsSent,
	)
	writePrometheusMetric(
		w,
		"niac_packets_received_total",
		"Total packets received",
		"counter",
		stats.PacketsReceived,
	)
	writePrometheusMetric(
		w,
		"niac_snmp_queries_total",
		"Total SNMP queries processed",
		"counter",
		stats.SNMPQueries,
	)
	writePrometheusMetric(w, "niac_errors_total", "Total errors", "counter", stats.Errors)
	writePrometheusMetric(
		w,
		"niac_devices_total",
		"Number of simulated devices",
		"gauge",
		deviceCount,
	)
}

// writeProtocolMetrics writes protocol-specific metrics (ARP, ICMP, DNS, DHCP).
func writeProtocolMetrics(w io.Writer, stats *protocols.Statistics) {
	writePrometheusMetric(
		w,
		"niac_arp_requests_total",
		"Total ARP requests sent",
		"counter",
		stats.ARPRequests,
	)
	writePrometheusMetric(
		w,
		"niac_arp_replies_total",
		"Total ARP replies sent",
		"counter",
		stats.ARPReplies,
	)
	writePrometheusMetric(
		w,
		"niac_icmp_requests_total",
		"Total ICMP requests sent",
		"counter",
		stats.ICMPRequests,
	)
	writePrometheusMetric(
		w,
		"niac_icmp_replies_total",
		"Total ICMP replies sent",
		"counter",
		stats.ICMPReplies,
	)
	writePrometheusMetric(
		w,
		"niac_dns_queries_total",
		"Total DNS queries processed",
		"counter",
		stats.DNSQueries,
	)
	writePrometheusMetric(
		w,
		"niac_dhcp_requests_total",
		"Total DHCP requests processed",
		"counter",
		stats.DHCPRequests,
	)
}

// writeSystemMetrics writes runtime and memory metrics.
func writeSystemMetrics(w io.Writer, startTime time.Time, memStats *runtime.MemStats) {
	writePrometheusMetric(
		w,
		"niac_uptime_seconds",
		"Server uptime in seconds",
		"gauge",
		int64(time.Since(startTime).Seconds()),
	)
	writePrometheusMetric(
		w,
		"niac_goroutines_total",
		"Number of goroutines",
		"gauge",
		runtime.NumGoroutine(),
	)
	writePrometheusMetric(
		w,
		"niac_memory_usage_bytes",
		"Memory usage in bytes",
		"gauge",
		memStats.Alloc,
	)
	writePrometheusMetric(
		w,
		"niac_memory_sys_bytes",
		"Total memory obtained from OS in bytes",
		"gauge",
		memStats.Sys,
	)
	writePrometheusMetric(
		w,
		"niac_gc_runs_total",
		"Total number of GC runs",
		"counter",
		memStats.NumGC,
	)
}

func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	s.configMu.RLock()
	stk := s.cfg.Stack
	cfg := s.cfg.Config
	s.configMu.RUnlock()

	if stk == nil {
		http.Error(w, "no simulation running", http.StatusServiceUnavailable)
		return
	}

	stats := stk.GetStats()

	deviceCount := 0
	if cfg != nil {
		deviceCount = len(cfg.Devices)
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	writeBasicMetrics(w, &stats, deviceCount)
	writeProtocolMetrics(w, &stats)
	writeSystemMetrics(w, s.startTime, &memStats)
}

func (s *Server) writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		s.logger.Debug("[API] JSON encode/write error", "error", err)
	}
}

func (s *Server) alertLoop(stop <-chan struct{}) {
	ticker := time.NewTicker(alertTickerSecs * time.Second)
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
	s.logger.Warn("Alert: packet threshold exceeded", "total", total)

	cfg := s.getAlertConfig()
	if cfg.WebhookURL == "" {
		return
	}

	// SECURITY FIX #161: Thread-safe access to Interface
	body, _ := json.Marshal(map[string]any{
		"type":        "packet_threshold",
		"threshold":   cfg.PacketsThreshold,
		"total":       total,
		"interface":   s.currentInterface(),
		"triggeredAt": time.Now().UTC(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), webhookTimeoutSecs*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		cfg.WebhookURL,
		strings.NewReader(string(body)),
	)
	if err != nil {
		s.logger.Error("Alert webhook error", "error", err)

		return
	}

	req.Header.Set("Content-Type", "application/json")

	// FIX #314, #315: Custom transport to validate resolved IPs and deny redirects
	dialer := &net.Dialer{Timeout: webhookTimeoutSecs * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, splitErr := net.SplitHostPort(addr)
			if splitErr != nil {
				return nil, fmt.Errorf("invalid address: %s", addr)
			}

			addrs, resolveErr := net.DefaultResolver.LookupIPAddr(ctx, host)
			if resolveErr != nil {
				return nil, fmt.Errorf("cannot resolve host: %s", host)
			}

			for _, addr := range addrs {
				if blockedErr := isBlockedIP(addr.IP); blockedErr != nil {
					return nil, fmt.Errorf("blocked destination: %w", blockedErr)
				}
			}

			// Connect to first allowed IP
			return dialer.DialContext(ctx, network, net.JoinHostPort(addrs[0].IP.String(), port))
		},
	}

	client := &http.Client{
		Timeout:   webhookTimeoutSecs * time.Second,
		Transport: transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errors.New("redirects not allowed for webhook URLs")
		},
	}

	resp, doErr := client.Do(req)
	if doErr != nil {
		s.logger.Error("Alert webhook request failed", "error", doErr)

		return
	}

	_ = resp.Body.Close()
}

// validatePCAPMagic validates that the file begins with a valid PCAP magic number
// SECURITY FIX LOW-2: Prevents processing of non-PCAP files that could exploit parser bugs.
func validatePCAPMagic(data []byte) error {
	if len(data) < minPCAPSize {
		return ErrFileTooSmallForPCAP
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

	if slices.Contains(validMagics, magic) {
		return nil
	}

	return fmt.Errorf(
		"%w: 0x%08x (expected pcap or pcapng format)",
		ErrInvalidPCAPMagicNumber,
		magic,
	)
}

// processInlineData decodes and validates inline PCAP data, writes it to a temp file.
// Returns the file path and any error encountered.
func (s *Server) processInlineData(inlineData string) (string, error) {
	// SECURITY FIX #97: Additional check on base64 encoded data size
	// Base64 encoding increases size by ~4/3, so check before decode
	if len(inlineData) > MaxPCAPUploadSize*4/base64Ratio {
		return "", ErrPCAPDataExceedsSizeLimit
	}

	data, decodeErr := base64.StdEncoding.DecodeString(inlineData)
	if decodeErr != nil {
		return "", fmt.Errorf("decode replay data: %w", decodeErr)
	}

	// Double-check decoded size
	if len(data) > MaxPCAPUploadSize {
		return "", ErrDecodedPCAPExceedsSizeLimit
	}

	// SECURITY FIX LOW-2: Validate PCAP file magic number
	if magicErr := validatePCAPMagic(data); magicErr != nil {
		return "", fmt.Errorf("invalid PCAP file: %w", magicErr)
	}

	return s.writeUploadedFile(data)
}

func (s *Server) prepareReplayRequest(req ReplayRequest) (ReplayRequest, error) {
	if strings.TrimSpace(req.File) == "" && req.InlineData == "" {
		return req, ErrPcapFilePathOrDataRequired
	}

	if req.InlineData == "" {
		// SECURITY FIX #162: Validate PCAP file path to prevent arbitrary file access
		validatedPath, err := s.validatePcapFilePath(req.File)
		if err != nil {
			return req, err
		}

		req.File = validatedPath

		return req, nil
	}

	path, err := s.processInlineData(req.InlineData)
	if err != nil {
		return req, err
	}

	req.File = path
	req.Uploaded = true
	req.InlineData = ""

	return req, nil
}

// SECURITY FIX #162: validatePcapFilePath ensures the file path is safe and doesn't traverse outside allowed directories.
func (s *Server) validatePcapFilePath(filename string) (string, error) {
	// Empty filename is invalid
	if filename == "" {
		return "", errors.New("filename cannot be empty")
	}

	// Clean the path to normalize it
	cleanPath := filepath.Clean(filename)

	// Reject paths containing null bytes (potential bypass attempt)
	if strings.ContainsRune(cleanPath, 0) {
		return "", errors.New("filename contains invalid characters")
	}

	// Get config directory as the allowed base directory
	cfgPath := s.configPath()
	var allowedDir string
	if cfgPath != "" {
		allowedDir = filepath.Dir(cfgPath)
	} else {
		// If no config path, use current working directory
		var err error
		allowedDir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("cannot determine allowed directory: %w", err)
		}
	}

	// Resolve to absolute path
	var absPath string
	if !filepath.IsAbs(cleanPath) {
		absPath = filepath.Join(allowedDir, cleanPath)
	} else {
		absPath = cleanPath
	}

	// Clean the absolute path and resolve symlinks for security
	absPath = filepath.Clean(absPath)
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If symlink resolution fails, file may not exist yet or symlink is broken
		realPath = absPath
	}

	// Resolve allowed directory to real path for comparison
	realAllowedDir, err := filepath.EvalSymlinks(allowedDir)
	if err != nil {
		realAllowedDir = allowedDir
	}

	// Verify the file is within the allowed directory (prevent path traversal)
	if !strings.HasPrefix(realPath, realAllowedDir+string(filepath.Separator)) &&
		realPath != realAllowedDir {
		return "", fmt.Errorf("access denied: file must be within %s", allowedDir)
	}

	// Verify the path exists and is a file (not a directory)
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("pcap file not found: %s", filename)
		}
		return "", fmt.Errorf("cannot access pcap file: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrPathIsADirectory, absPath)
	}

	// Verify file has a valid pcap extension
	ext := strings.ToLower(filepath.Ext(absPath))
	validExts := map[string]bool{".pcap": true, ".pcapng": true, ".cap": true}
	if !validExts[ext] {
		return "", fmt.Errorf(
			"invalid pcap file extension: %s (allowed: .pcap, .pcapng, .cap)",
			ext,
		)
	}

	return absPath, nil
}

func (s *Server) writeUploadedFile(data []byte) (string, error) {
	// SECURITY FIX #167: Use restrictive permissions for temp directory (owner-only)
	dir := filepath.Join(os.TempDir(), "niac-replay")

	mkdirErr := os.MkdirAll(dir, 0o700)
	if mkdirErr != nil {
		return "", fmt.Errorf("create upload dir: %w", mkdirErr)
	}

	// Ensure directory permissions are correct even if it already exists.
	// 0o700 = owner rwx only, restrictive for upload directories.
	const secureDirPerm = 0o700

	chmodErr := os.Chmod(dir, secureDirPerm)
	if chmodErr != nil {
		return "", fmt.Errorf("secure upload dir: %w", chmodErr)
	}

	tmp, createErr := os.CreateTemp(dir, "upload-*.pcap")
	if createErr != nil {
		return "", fmt.Errorf("create temp file: %w", createErr)
	}

	tmpPath := tmp.Name()

	// SECURITY FIX #167: Write data while file is still open (no race window)
	_, writeErr := tmp.Write(data)
	if writeErr != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)

		return "", fmt.Errorf("write upload: %w", writeErr)
	}

	syncErr := tmp.Sync()
	if syncErr != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)

		return "", fmt.Errorf("sync upload: %w", syncErr)
	}

	// Close file but don't use defer - we need to return path on success
	closeErr := tmp.Close()
	if closeErr != nil {
		_ = os.Remove(tmpPath)

		return "", fmt.Errorf("close temp file: %w", closeErr)
	}

	return tmpPath, nil
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
	// SECURITY FIX #161: Thread-safe access to ConfigPath
	cfgPath := s.configPath()
	if cfgPath == "" {
		return nil, http.StatusBadRequest, ErrConfigPathNotAvailable
	}

	// #nosec G304 -- cfgPath is validated by configPath() which ensures thread-safe
	// access and path validation before use
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, http.StatusNotFound, fmt.Errorf("%w: %s", ErrConfigFileNotFound, cfgPath)
		}

		return nil, http.StatusInternalServerError, fmt.Errorf("reading config: %w", err)
	}

	info, err := os.Stat(cfgPath)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("stat config: %w", err)
	}

	cfg := s.currentConfig()

	deviceCount := 0
	if cfg != nil {
		deviceCount = len(cfg.Devices)
	}

	return &configDocument{
		Path:        cfgPath,
		Filename:    filepath.Base(cfgPath),
		ModifiedAt:  info.ModTime().UTC(),
		SizeBytes:   info.Size(),
		DeviceCount: deviceCount,
		Content:     string(data),
	}, http.StatusOK, nil
}

func (s *Server) writeConfigFile(content string) error {
	// SECURITY FIX #161: Thread-safe access to ConfigPath
	cfgPath := s.configPath()
	dir := filepath.Dir(cfgPath)

	mkdirErr := os.MkdirAll(dir, 0o750)
	if mkdirErr != nil {
		return fmt.Errorf("create config directory: %w", mkdirErr)
	}

	tmp, createErr := os.CreateTemp(dir, ".niac-config-*")
	if createErr != nil {
		return fmt.Errorf("create temp file: %w", createErr)
	}

	tmpPath := tmp.Name()

	defer func() { _ = os.Remove(tmpPath) }()

	_, writeErr := tmp.WriteString(content)
	if writeErr != nil {
		_ = tmp.Close()

		return fmt.Errorf("write temp file: %w", writeErr)
	}

	syncErr := tmp.Sync()
	if syncErr != nil {
		_ = tmp.Close()

		return fmt.Errorf("sync temp file: %w", syncErr)
	}

	closeErr := tmp.Close()
	if closeErr != nil {
		return fmt.Errorf("close temp file: %w", closeErr)
	}

	chmodErr := os.Chmod(tmpPath, 0o600)
	if chmodErr != nil {
		return fmt.Errorf("chmod temp file: %w", chmodErr)
	}

	renameErr := os.Rename(tmpPath, cfgPath)
	if renameErr != nil {
		return fmt.Errorf("replace config: %w", renameErr)
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

// SECURITY FIX #161: Thread-safe access to ConfigPath to prevent race conditions.
func (s *Server) configPath() string {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	return s.cfg.ConfigPath
}

// SECURITY FIX #161: Thread-safe access to Stack to prevent race conditions.
func (s *Server) currentStack() *protocols.Stack {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	return s.cfg.Stack
}

// SECURITY FIX #161: Thread-safe access to Interface to prevent race conditions.
func (s *Server) currentInterface() string {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	return s.cfg.Interface
}

// getFileKindConfig returns root path and extensions for a file kind.
func (s *Server) getFileKindConfig(kind string) (string, []string, error) {
	switch kind {
	case "walks":
		return s.resolveIncludePath(), []string{".walk"}, nil
	case "pcaps":
		if cfgPath := s.configPath(); cfgPath != "" {
			return filepath.Dir(cfgPath), []string{".pcap", ".pcapng"}, nil
		}

		return "", []string{".pcap", ".pcapng"}, nil
	default:
		return "", nil, fmt.Errorf("%w: %s", ErrUnsupportedFileKind, kind)
	}
}

// resolveSecureRoot validates and resolves the root path for file collection.
func resolveSecureRoot(root string) (string, error) {
	rootInfo, statErr := os.Stat(root)
	if statErr != nil {
		return "", statErr
	}

	if !rootInfo.IsDir() {
		return "", ErrNotADirectory
	}

	rootAbs, absErr := filepath.Abs(root)
	if absErr != nil {
		return "", fmt.Errorf("failed to resolve root path: %w", absErr)
	}

	// Fall back to absolute path if symlink resolution fails
	rootReal, _ := filepath.EvalSymlinks(rootAbs)
	if rootReal == "" {
		return rootAbs, nil
	}

	return rootReal, nil
}

// processFileEntry validates and creates a FileEntry for a collected file.
func processFileEntry(path string, d fs.DirEntry, rootReal string) (*FileEntry, error) {
	info, infoErr := d.Info()
	if infoErr != nil {
		return nil, fmt.Errorf("failed to get file info: %w", infoErr)
	}

	absPath, absErr := filepath.Abs(path)
	if absErr != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", absErr)
	}

	// SECURITY FIX #95: Validate path stays within root directory
	realPath, evalErr := filepath.EvalSymlinks(absPath)
	if evalErr != nil {
		return nil, fmt.Errorf("failed to evaluate symlinks: %w", evalErr)
	}

	// Ensure resolved path is within the allowed root directory
	if !strings.HasPrefix(realPath, rootReal+string(os.PathSeparator)) && realPath != rootReal {
		return nil, ErrPathOutsideRoot
	}

	return &FileEntry{
		Path:      absPath,
		Name:      filepath.Base(path),
		SizeBytes: info.Size(),
		Modified:  info.ModTime().UTC(),
	}, nil
}

func (s *Server) collectFiles(kind string) ([]FileEntry, error) {
	root, exts, err := s.getFileKindConfig(kind)
	if err != nil {
		return nil, err
	}

	if root == "" {
		return []FileEntry{}, nil
	}

	rootReal, err := resolveSecureRoot(root)
	if err != nil {
		if errors.Is(err, ErrNotADirectory) {
			return []FileEntry{}, nil
		}

		return []FileEntry{}, fmt.Errorf("failed to stat root: %w", err)
	}

	var entries []FileEntry

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !slices.Contains(exts, ext) {
			return nil
		}

		entry, procErr := processFileEntry(path, d, rootReal)
		if procErr != nil {
			if errors.Is(procErr, ErrPathOutsideRoot) {
				return nil // Skip files outside allowed directory
			}

			return procErr
		}

		entries = append(entries, *entry)
		if len(entries) >= maxFileEntries {
			return filepath.SkipDir
		}

		return nil
	})
	if err != nil && !errors.Is(err, filepath.SkipDir) {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return entries, nil
}

func (s *Server) resolveIncludePath() string {
	cfg := s.currentConfig()
	if cfg == nil || cfg.IncludePath == "" {
		return ""
	}

	includePath := cfg.IncludePath
	// SECURITY FIX #161: Thread-safe access to ConfigPath
	if cfgPath := s.configPath(); !filepath.IsAbs(includePath) && cfgPath != "" {
		includePath = filepath.Join(filepath.Dir(cfgPath), includePath)
	}

	if abs, err := filepath.Abs(includePath); err == nil {
		return abs
	}

	return includePath
}
