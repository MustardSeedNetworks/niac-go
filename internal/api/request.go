package api

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
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

// getClientIP extracts the real client IP from the request
// SECURITY FIX HIGH-1: Only trust forwarded headers from trusted proxies.
func getClientIP(r *http.Request) string {
	// Get the direct connection IP first
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteIP = r.RemoteAddr
	}

	// SECURITY: Only trust X-Forwarded-For/X-Real-IP if coming from localhost/private networks
	// This prevents header spoofing attacks where clients forge these headers to bypass rate limiting
	if !isTrustedProxy(remoteIP) {
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

// isTrustedProxy checks if an IP is from a trusted proxy/load balancer
// SECURITY: Prevents header spoofing by only trusting forwarded headers from known proxies.
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

	return false
}

// generateRequestID creates a unique request ID for tracing
// FEATURE #118: Request tracing for debugging and monitoring.
func generateRequestID() string {
	b := make([]byte, requestIDBytes)
	_, _ = rand.Read(b) // crypto/rand read errors will result in zero bytes, still usable

	return hex.EncodeToString(b)
}
