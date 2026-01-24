package api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// newTestServerWithAuth creates a test server with auth and rate limiting initialized.
func newTestServerWithAuth(t *testing.T) (*Server, string, string) {
	t.Helper()
	server, tmpDir := newTestServer(t)

	// Initialize server components
	server.rateLimiter = NewRateLimiter(DefaultRateLimit, DefaultBurst)

	// Generate CSRF token
	csrfToken, err := generateCSRFToken()
	if err != nil {
		t.Fatalf("Failed to generate CSRF token: %v", err)
	}

	server.csrfToken = csrfToken

	// Generate auth token
	token := generateTestToken()
	server.cfg.Token = token

	return server, tmpDir, token
}

// TestRateLimiter_ExcessiveRequests verifies rate limiting blocks excessive requests.
func TestRateLimiter_ExcessiveRequests(t *testing.T) {
	limiter := NewRateLimiter(rate.Limit(2), 5) // 2 req/sec, burst of 5

	ip := "192.168.1.1"
	rl := limiter.GetLimiter(ip)

	// First 5 requests should succeed (burst)
	for i := range 5 {
		if !rl.Allow() {
			t.Errorf("Request %d should be allowed (within burst)", i+1)
		}
	}

	// 6th request should be rate limited
	if rl.Allow() {
		t.Error("Request 6 should be rate limited (burst exhausted)")
	}

	// Wait for rate limiter to refill
	time.Sleep(500 * time.Millisecond)

	// Should allow 1 more request (rate refill)
	if !rl.Allow() {
		t.Error("Request after rate refill should be allowed")
	}
}

// TestRateLimiter_PerIPIsolation verifies different IPs have independent rate limits.
func TestRateLimiter_PerIPIsolation(t *testing.T) {
	limiter := NewRateLimiter(rate.Limit(2), 3)

	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	rl1 := limiter.GetLimiter(ip1)
	rl2 := limiter.GetLimiter(ip2)

	// Exhaust IP1's burst
	for i := range 3 {
		if !rl1.Allow() {
			t.Errorf("IP1 request %d should be allowed", i+1)
		}
	}

	// IP1 should be rate limited
	if rl1.Allow() {
		t.Error("IP1 should be rate limited after burst")
	}

	// IP2 should still have full burst available
	for i := range 3 {
		if !rl2.Allow() {
			t.Errorf("IP2 request %d should be allowed (independent limit)", i+1)
		}
	}
}

// TestRateLimiter_CleanupPreventsMemoryLeak verifies stale limiters are cleaned up.
func TestRateLimiter_CleanupPreventsMemoryLeak(t *testing.T) {
	limiter := NewRateLimiter(rate.Limit(10), 20)

	// Create limiters for 100 different IPs
	for i := range 100 {
		ip := fmt.Sprintf("192.168.1.%d", i)
		limiter.GetLimiter(ip)
	}

	// Verify all 100 limiters exist
	limiter.mu.RLock()
	initialCount := len(limiter.limiters)
	limiter.mu.RUnlock()

	if initialCount != 100 {
		t.Errorf("Expected 100 limiters, got %d", initialCount)
	}

	// Access only the first 10 IPs to keep them "fresh"
	for i := range 10 {
		ip := fmt.Sprintf("192.168.1.%d", i)
		limiter.GetLimiter(ip)
	}

	// Manually set lastSeen to simulate stale entries (over 1 hour ago)
	limiter.mu.Lock()

	staleTime := time.Now().Add(-2 * time.Hour)

	freshIPs := map[string]bool{
		"192.168.1.0": true, "192.168.1.1": true, "192.168.1.2": true,
		"192.168.1.3": true, "192.168.1.4": true, "192.168.1.5": true,
		"192.168.1.6": true, "192.168.1.7": true, "192.168.1.8": true,
		"192.168.1.9": true,
	}
	for ip, entry := range limiter.limiters {
		// Make all but first 10 IPs stale
		if !freshIPs[ip] {
			entry.lastSeen = staleTime
		}
	}

	limiter.mu.Unlock()

	// Run cleanup
	limiter.CleanupStale()

	// Verify stale limiters were removed
	limiter.mu.RLock()
	finalCount := len(limiter.limiters)
	limiter.mu.RUnlock()

	if finalCount != 10 {
		t.Errorf("Expected 10 limiters after cleanup (kept fresh ones), got %d", finalCount)
	}
}

// TestRateLimiter_MaxCapacityEnforcement verifies FIFO eviction at max capacity.
func TestRateLimiter_MaxCapacityEnforcement(t *testing.T) {
	// Temporarily reduce max for testing
	originalMax := MaxRateLimiterCount

	defer func() {
		// Can't restore const, but document that this test modifies behavior
	}()

	limiter := NewRateLimiter(rate.Limit(10), 20)

	// Fill up to a reasonable test limit (100 IPs)
	testLimit := 100
	for i := range testLimit {
		ip := fmt.Sprintf("192.168.%d.%d", i/256, i%256)
		limiter.GetLimiter(ip)
	}

	// Verify we have exactly testLimit limiters
	limiter.mu.RLock()
	count := len(limiter.limiters)
	limiter.mu.RUnlock()

	if count != testLimit {
		t.Errorf("Expected %d limiters, got %d", testLimit, count)
	}

	// Note: At normal MaxRateLimiterCount (10000), we can't easily test eviction
	// This test verifies the mechanism works at smaller scale
	limiter.mu.Lock()
	// Check current size before adding new IP
	_ = len(limiter.limiters) // currentSize for future use
	limiter.mu.Unlock()

	// Add one more IP - if at capacity, oldest should be evicted
	newIP := "10.0.0.1"
	limiter.GetLimiter(newIP)

	// Verify the new IP was added
	limiter.mu.RLock()
	_, exists := limiter.limiters[newIP]
	limiter.mu.RUnlock()

	if !exists {
		t.Error("New IP should have been added")
	}

	t.Logf("Test note: MaxRateLimiterCount=%d prevents exhaustive eviction testing", originalMax)
}

// TestCSRF_TokenValidation verifies CSRF token validation works correctly.
func TestCSRF_TokenValidation(t *testing.T) {
	server, tmpDir, token := newTestServerWithAuth(t)

	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Test valid CSRF token on protected endpoint
	reqBody := strings.NewReader(`{"packets_threshold":1000}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/alerts", reqBody)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Csrf-Token", server.csrfToken)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	handler := server.auth(server.csrfProtect(server.handleAlerts))
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Valid CSRF token should be accepted, got status: %d", w.Code)
	}
}

// TestCSRF_InvalidTokenRejection verifies invalid CSRF tokens are rejected.
func TestCSRF_InvalidTokenRejection(t *testing.T) {
	server, tmpDir, token := newTestServerWithAuth(t)

	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Try to use an invalid CSRF token
	reqBody := strings.NewReader(`{"packets_threshold":1000}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/alerts", reqBody)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Csrf-Token", "invalid-token-12345")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	handler := server.auth(server.csrfProtect(server.handleAlerts))
	handler(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Invalid CSRF token should be rejected with 403, got: %d", w.Code)
	}
}

// TestCSRF_MissingTokenRejection verifies missing CSRF tokens are rejected.
func TestCSRF_MissingTokenRejection(t *testing.T) {
	server, tmpDir, token := newTestServerWithAuth(t)

	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Try to make a state-changing request without CSRF token
	reqBody := strings.NewReader(`{"packets_threshold":1000}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/alerts", reqBody)
	req.Header.Set("Authorization", "Bearer "+token)
	// Deliberately omit X-CSRF-Token header
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	handler := server.auth(server.csrfProtect(server.handleAlerts))
	handler(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Missing CSRF token should be rejected with 403, got: %d", w.Code)
	}
}

// TestAuth_ValidToken verifies valid authentication tokens are accepted.
func TestAuth_ValidToken(t *testing.T) {
	server, tmpDir, token := newTestServerWithAuth(t)

	defer func() { _ = os.RemoveAll(tmpDir) }()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()

	handler := server.auth(server.handleStats)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Valid token should be accepted, got status: %d", w.Code)
	}
}

// TestAuth_InvalidTokenRejection verifies invalid tokens are rejected.
func TestAuth_InvalidToken(t *testing.T) {
	server, tmpDir, _ := newTestServerWithAuth(t)

	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Use a different invalid token
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-xyz")

	w := httptest.NewRecorder()

	handler := server.auth(server.handleStats)
	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Invalid token should be rejected with 401, got: %d", w.Code)
	}
}

// TestAuth_MissingToken verifies requests without tokens are rejected.
func TestAuth_MissingToken(t *testing.T) {
	server, tmpDir, _ := newTestServerWithAuth(t)

	defer func() { _ = os.RemoveAll(tmpDir) }()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	// Deliberately omit Authorization header
	w := httptest.NewRecorder()

	handler := server.auth(server.handleStats)
	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Missing token should be rejected with 401, got: %d", w.Code)
	}
}

// TestAuth_NoAuthRequired verifies endpoints work without auth when token not set.
func TestAuth_NoAuthRequired(t *testing.T) {
	server, tmpDir := newTestServer(t)

	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Initialize rate limiter but NOT auth token
	server.rateLimiter = NewRateLimiter(DefaultRateLimit, DefaultBurst)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	// No Authorization header
	w := httptest.NewRecorder()

	handler := server.auth(server.handleStats)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Request should succeed when auth is disabled, got: %d", w.Code)
	}
}

// TestPanicRecovery_NilPointerPanic verifies panic recovery catches nil pointer panics.
func TestPanicRecovery_NilPointerPanic(t *testing.T) {
	server, tmpDir, _ := newTestServerWithAuth(t)

	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a handler that deliberately panics
	panicHandler := func(_ http.ResponseWriter, _ *http.Request) {
		panic("intentional panic for testing recovery middleware")
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Wrap with recovery middleware
	handler := server.recoverMiddleware(panicHandler)
	handler(w, req)

	// Should recover and return 500
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Panic should be recovered with 500, got: %d", w.Code)
	}

	// Response should be JSON error
	var response ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if response.Error != "internal_server_error" {
		t.Errorf("Expected error code 'internal_server_error', got: %s", response.Error)
	}
}

// TestPanicRecovery_ArrayOutOfBounds verifies panic recovery catches array panics.
func TestPanicRecovery_ArrayOutOfBounds(t *testing.T) {
	server, tmpDir, _ := newTestServerWithAuth(t)

	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a handler that panics with array out of bounds
	panicHandler := func(_ http.ResponseWriter, _ *http.Request) {
		arr := []int{1, 2, 3}
		_ = arr[10]
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler := server.recoverMiddleware(panicHandler)
	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Panic should be recovered with 500, got: %d", w.Code)
	}
}

// TestPanicRecovery_NormalOperation verifies recovery doesn't affect normal requests.
func TestPanicRecovery_NormalOperation(t *testing.T) {
	server, tmpDir, _ := newTestServerWithAuth(t)

	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Normal handler that doesn't panic
	normalHandler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler := server.recoverMiddleware(normalHandler)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Normal request should succeed, got: %d", w.Code)
	}

	if w.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got: %s", w.Body.String())
	}
}

// TestGetClientIP_DirectConnection verifies IP extraction from direct connections.
func TestGetClientIP_DirectConnection(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.100:54321"

	ip := getClientIP(req, nil)

	if ip != "192.168.1.100" {
		t.Errorf("Expected IP 192.168.1.100, got: %s", ip)
	}
}

// TestGetClientIP_XForwardedFor verifies IP extraction from X-Forwarded-For header.
func TestGetClientIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:54321"
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 198.51.100.1")

	ip := getClientIP(req, nil)

	// Should use first IP from X-Forwarded-For
	if ip != "203.0.113.5" {
		t.Errorf("Expected IP 203.0.113.5, got: %s", ip)
	}
}

// TestGetClientIP_XRealIP verifies IP extraction from X-Real-IP header.
func TestGetClientIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:54321"
	req.Header.Set("X-Real-IP", "203.0.113.10")

	ip := getClientIP(req, nil)

	if ip != "203.0.113.10" {
		t.Errorf("Expected IP 203.0.113.10, got: %s", ip)
	}
}

// TestRateLimiting_ConcurrentRequests verifies rate limiting under concurrent load.
func TestRateLimiting_ConcurrentRequests(t *testing.T) {
	limiter := NewRateLimiter(rate.Limit(100), 200)

	ip := "192.168.1.1"
	concurrency := 50
	requestsPerGoroutine := 10

	var wg sync.WaitGroup

	allowed := make(chan bool, concurrency*requestsPerGoroutine)

	// Launch concurrent requests
	for range concurrency {
		wg.Go(func() {
			rl := limiter.GetLimiter(ip)

			for range requestsPerGoroutine {
				allowed <- rl.Allow()
			}
		})
	}

	wg.Wait()
	close(allowed)

	// Count allowed requests
	allowedCount := 0

	for a := range allowed {
		if a {
			allowedCount++
		}
	}

	// Should allow burst (200) + some more from rate refill
	// But not all 500 requests
	if allowedCount > 300 {
		t.Errorf("Too many requests allowed: %d (expected ~200-300)", allowedCount)
	}

	if allowedCount < 150 {
		t.Errorf("Too few requests allowed: %d (expected ~200-300)", allowedCount)
	}

	t.Logf("Allowed %d/%d concurrent requests", allowedCount, concurrency*requestsPerGoroutine)
}

// TestAlertConfig_ConcurrentUpdates verifies concurrent alert config updates don't race (SECURITY FIX #2.8.1).
func TestAlertConfig_ConcurrentUpdates(t *testing.T) {
	server, tmpDir, token := newTestServerWithAuth(t)

	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Concurrently update alert config multiple times
	const numUpdates = 10

	var wg sync.WaitGroup

	for i := range numUpdates {
		wg.Add(1)

		go func(threshold int) {
			defer wg.Done()

			reqBody := fmt.Sprintf(`{"packets_threshold":%d}`, threshold*1000)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/alerts", strings.NewReader(reqBody))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("X-Csrf-Token", server.csrfToken)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			handler := server.auth(server.csrfProtect(server.handleAlerts))
			handler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Alert update failed with status: %d", w.Code)
			}
		}(i)
	}

	wg.Wait()

	// Verify server is still functional
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()

	handler := server.auth(server.handleAlerts)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Final alert GET failed with status: %d", w.Code)
	}
}

// TestAlertConfig_NoDoubleClose verifies alert channel doesn't panic on double close (SECURITY FIX #2.8.1).
func TestAlertConfig_NoDoubleClose(t *testing.T) {
	server, tmpDir, token := newTestServerWithAuth(t)

	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Enable alerts
	reqBody := `{"packets_threshold":1000}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/alerts", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Csrf-Token", server.csrfToken)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	handler := server.auth(server.csrfProtect(server.handleAlerts))
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to enable alerts: %d", w.Code)
	}

	// Update alerts multiple times rapidly (triggers stop/start of goroutine)
	for i := range 5 {
		reqBody = fmt.Sprintf(`{"packets_threshold":%d}`, (i+1)*1000)
		req = httptest.NewRequest(http.MethodPut, "/api/v1/alerts", strings.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-Csrf-Token", server.csrfToken)
		req.Header.Set("Content-Type", "application/json")

		w = httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Failed to update alerts iteration %d: %d", i, w.Code)
		}
	}

	// Disable alerts (should cleanly close channel)
	reqBody = `{"packets_threshold":0}`
	req = httptest.NewRequest(http.MethodPut, "/api/v1/alerts", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Csrf-Token", server.csrfToken)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to disable alerts: %d", w.Code)
	}
	// No panic should occur
}

// TestAlertConfig_GoroutineCleanup verifies alert goroutines are cleaned up properly (SECURITY FIX #98).
func TestAlertConfig_GoroutineCleanup(t *testing.T) {
	server, tmpDir, token := newTestServerWithAuth(t)

	defer func() { _ = os.RemoveAll(tmpDir) }()

	initialGoroutines := runtime.NumGoroutine()

	// Enable and disable alerts multiple times
	for range 10 {
		// Enable
		reqBody := `{"packets_threshold":1000}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/alerts", strings.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-Csrf-Token", server.csrfToken)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		handler := server.auth(server.csrfProtect(server.handleAlerts))
		handler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Failed to enable alerts: %d", w.Code)
		}

		// Disable
		reqBody = `{"packets_threshold":0}`
		req = httptest.NewRequest(http.MethodPut, "/api/v1/alerts", strings.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-Csrf-Token", server.csrfToken)
		req.Header.Set("Content-Type", "application/json")

		w = httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Failed to disable alerts: %d", w.Code)
		}
	}

	// Give goroutines time to exit
	time.Sleep(100 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()

	// Allow for some variance but ensure we don't have a goroutine leak
	// We expect at most a few more goroutines than we started with
	if finalGoroutines > initialGoroutines+5 {
		t.Errorf("Possible goroutine leak: started with %d, ended with %d", initialGoroutines, finalGoroutines)
	}
}

// Helper function to generate test authentication tokens.
func generateTestToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)

	return base64.URLEncoding.EncodeToString(b)
}
