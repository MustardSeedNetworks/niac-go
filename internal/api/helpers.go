package api

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/krisarmstrong/niac-go/internal/config"
)

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

// generateCSRFToken generates a cryptographically secure random token
// SECURITY FIX LOW-1: CSRF protection.
func generateCSRFToken() (string, error) {
	bytes := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	return hex.EncodeToString(bytes), nil
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

// writeJSON writes a JSON response to the [http.ResponseWriter].
func (s *Server) writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		s.logger.Debug("[API] JSON encode/write error", "error", err)
	}
}

// serveSPA serves the Single Page Application for the web UI.
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
