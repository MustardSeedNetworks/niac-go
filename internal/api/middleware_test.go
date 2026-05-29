package api

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/logging"
	"github.com/krisarmstrong/niac-go/internal/protocols"
)

// createTestServerForMiddleware creates a minimal server for middleware tests.
func createTestServerForMiddleware(t *testing.T) *Server {
	t.Helper()
	cfg, err := config.LoadYAMLBytes([]byte(`
devices:
  - name: test-router
    mac: "00:11:22:33:44:55"
    ips: ["10.0.0.1"]
    type: router
`))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))

	return &Server{
		cfg: ServerConfig{
			Stack:  stack,
			Config: cfg,
		},
		logger: slog.Default(),
		// auth() applies per-IP rate limiting before the token check, so
		// any test that exercises auth() needs a non-nil limiter.
		rateLimiter: NewRateLimiter(DefaultRateLimit, DefaultBurst),
	}
}

func TestRecoverMiddlewareReturnsInternalError(t *testing.T) {
	server := createTestServerForMiddleware(t)

	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("test panic message")
	})

	wrapped := server.recoverMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Should not panic - middleware catches it
	wrapped(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestRecoverMiddlewareNoPanic(t *testing.T) {
	server := createTestServerForMiddleware(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	wrapped := server.recoverMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestCSRFProtectionGETAllowed(t *testing.T) {
	server := createTestServerForMiddleware(t)
	server.csrf = NewCSRFManager()
	t.Cleanup(server.csrf.Stop)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := server.csrfProtect(handler)

	// GET should always be allowed
	for _, method := range []string{"GET", "HEAD", "OPTIONS"} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", nil)
			rec := httptest.NewRecorder()

			wrapped(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("%s status = %d, want %d", method, rec.Code, http.StatusOK)
			}
		})
	}
}

func TestCSRFProtectionStateMethods(t *testing.T) {
	server := createTestServerForMiddleware(t)
	server.csrf = NewCSRFManager()
	t.Cleanup(server.csrf.Stop)

	// Mint the per-session token for the loopback bypass key (no
	// Authorization header => csrfLoopbackSessionKey).
	validToken, err := server.csrf.Generate(SessionKey(""))
	if err != nil {
		t.Fatalf("mint CSRF: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := server.csrfProtect(handler)

	// State-changing methods require CSRF token
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		t.Run(method+"_without_token", func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", nil)
			rec := httptest.NewRecorder()

			wrapped(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Errorf("%s without token: status = %d, want %d", method, rec.Code, http.StatusForbidden)
			}
		})

		t.Run(method+"_with_valid_token", func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", nil)
			req.Header.Set("X-Csrf-Token", validToken)
			rec := httptest.NewRecorder()

			wrapped(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("%s with valid token: status = %d, want %d", method, rec.Code, http.StatusOK)
			}
		})

		t.Run(method+"_with_invalid_token", func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", nil)
			req.Header.Set("X-Csrf-Token", "wrong-token")
			rec := httptest.NewRecorder()

			wrapped(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Errorf("%s with invalid token: status = %d, want %d", method, rec.Code, http.StatusForbidden)
			}
		})
	}
}

func TestCSRFProtectionRejectsWhenNoToken(t *testing.T) {
	server := createTestServerForMiddleware(t)
	// #1257: nil csrf manager => server misconfigured.
	server.csrf = nil

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := server.csrfProtect(handler)

	// POST must be rejected when csrfToken is empty (fail-closed security)
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("POST without csrfToken configured: status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestAuthMiddlewareWithoutToken(t *testing.T) {
	server := createTestServerForMiddleware(t)
	// Wave 2: tokens live in TokenStore. The Wave 1 single-token model
	// is preserved by seeding the store with one ScopeReadWrite entry —
	// behaviourally identical to setting cfg.Token previously.
	server.SetTokens([]ScopedToken{{Value: "secret-api-token", Scope: ScopeReadWrite}})
	server.rateLimiter = NewRateLimiter(100, 200)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := server.auth(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	rec := httptest.NewRecorder()

	wrapped(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("request without token: status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddlewareWithValidToken(t *testing.T) {
	server := createTestServerForMiddleware(t)
	server.SetTokens([]ScopedToken{{Value: "secret-api-token", Scope: ScopeReadWrite}})
	server.rateLimiter = NewRateLimiter(100, 200)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := server.auth(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	req.Header.Set("Authorization", "Bearer secret-api-token")
	rec := httptest.NewRecorder()

	wrapped(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("request with valid token: status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthMiddlewareWithInvalidToken(t *testing.T) {
	server := createTestServerForMiddleware(t)
	server.SetTokens([]ScopedToken{{Value: "secret-api-token", Scope: ScopeReadWrite}})
	server.rateLimiter = NewRateLimiter(100, 200)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := server.auth(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	req.Header.Set("Authorization", "Bearer wrong-token")
	rec := httptest.NewRecorder()

	wrapped(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("request with invalid token: status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddlewareNoTokenConfigured(t *testing.T) {
	server := createTestServerForMiddleware(t)
	// Wave 2: empty TokenStore is the canonical "auth disabled" shape.
	server.SetTokens(nil)
	server.rateLimiter = NewRateLimiter(100, 200)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := server.auth(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	rec := httptest.NewRecorder()

	wrapped(rec, req)

	// When no API token is configured, auth should be skipped
	if rec.Code != http.StatusOK {
		t.Errorf("request with no token configured: status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// TestAuthMiddleware_ReadOnlyTokenOnGet_200 asserts a read-only token
// is sufficient for GET requests (Wave 2 scope enforcement).
func TestAuthMiddleware_ReadOnlyTokenOnGet_200(t *testing.T) {
	server := createTestServerForMiddleware(t)
	server.SetTokens([]ScopedToken{{Value: "ro-token", Scope: ScopeReadOnly}})
	server.rateLimiter = NewRateLimiter(100, 200)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := server.auth(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	req.Header.Set("Authorization", "Bearer ro-token")
	rec := httptest.NewRecorder()

	wrapped(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("read-only token on GET: status = %d, want 200", rec.Code)
	}
}

// TestAuthMiddleware_ReadOnlyTokenOnPost_403 asserts a read-only token
// is rejected on state-changing methods with 403 (not 401) — the token
// is valid, just under-privileged.
func TestAuthMiddleware_ReadOnlyTokenOnPost_403(t *testing.T) {
	server := createTestServerForMiddleware(t)
	server.SetTokens([]ScopedToken{{Value: "ro-token", Scope: ScopeReadOnly}})
	server.rateLimiter = NewRateLimiter(100, 200)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := server.auth(handler)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/config", nil)
			req.RemoteAddr = "192.168.1.1:1234"
			req.Header.Set("Authorization", "Bearer ro-token")
			rec := httptest.NewRecorder()

			wrapped(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Errorf("read-only token on %s: status = %d, want 403", method, rec.Code)
			}
		})
	}
}

// TestAuthMiddleware_ReadWriteTokenOnPost_200 asserts a read-write
// token is accepted on state-changing methods.
func TestAuthMiddleware_ReadWriteTokenOnPost_200(t *testing.T) {
	server := createTestServerForMiddleware(t)
	server.SetTokens([]ScopedToken{{Value: "rw-token", Scope: ScopeReadWrite}})
	server.rateLimiter = NewRateLimiter(100, 200)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := server.auth(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/config", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	req.Header.Set("Authorization", "Bearer rw-token")
	rec := httptest.NewRecorder()

	wrapped(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("read-write token on POST: status = %d, want 200", rec.Code)
	}
}

// TestAuthMiddleware_UnknownToken_401 asserts a token not in the store
// is rejected with 401 regardless of HTTP method.
func TestAuthMiddleware_UnknownToken_401(t *testing.T) {
	server := createTestServerForMiddleware(t)
	server.SetTokens([]ScopedToken{{Value: "known-token", Scope: ScopeReadWrite}})
	server.rateLimiter = NewRateLimiter(100, 200)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := server.auth(handler)

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/stats", nil)
			req.RemoteAddr = "192.168.1.1:1234"
			req.Header.Set("Authorization", "Bearer not-the-token")
			rec := httptest.NewRecorder()

			wrapped(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("unknown token on %s: status = %d, want 401", method, rec.Code)
			}
		})
	}
}

// TestAuthMiddleware_BackCompat_SingleNIACAPIToken asserts the Wave 1
// contract: setting only ServerConfig.Token (the env-var-backed single
// token) yields a read-write token via initialTokenStore. Operators
// who set just NIAC_API_TOKEN must see no behaviour change.
func TestAuthMiddleware_BackCompat_SingleNIACAPIToken(t *testing.T) {
	store := initialTokenStore(ServerConfig{Token: "legacy-token"})
	if store.Len() != 1 {
		t.Fatalf("Wave 1 single-token seed: store size = %d, want 1", store.Len())
	}
	scope, ok := store.Lookup("legacy-token")
	if !ok {
		t.Fatal("Wave 1 single-token seed: token not found in store")
	}
	if scope != ScopeReadWrite {
		t.Errorf("Wave 1 single-token seed: scope = %v, want ScopeReadWrite", scope)
	}
}

// TestAuth_EmptyStore_LoopbackBypasses confirms the documented local-dev
// path: an empty token store on a loopback (or unbound) listener lets
// requests through without auth (#739).
func TestAuth_EmptyStore_LoopbackBypasses(t *testing.T) {
	server := createTestServerForMiddleware(t)
	server.tokens = NewTokenStore(nil) // empty store
	server.cfg.Addr = "127.0.0.1:8445"

	called := false
	wrapped := server.auth(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	rec := httptest.NewRecorder()
	wrapped(rec, req)

	if !called || rec.Code != http.StatusOK {
		t.Errorf("loopback + empty store should bypass auth: called=%v code=%d", called, rec.Code)
	}
}

// TestAuth_EmptyStore_NonLoopbackFailsClosed is the #739 regression test:
// an empty token store on a NON-loopback listener, without an explicit
// opt-out, must fail closed (401) instead of silently bypassing auth —
// even though the startup gate should have prevented this bind.
func TestAuth_EmptyStore_NonLoopbackFailsClosed(t *testing.T) {
	server := createTestServerForMiddleware(t)
	server.tokens = NewTokenStore(nil) // empty store
	server.cfg.Addr = "0.0.0.0:8445"   // non-loopback

	called := false
	wrapped := server.auth(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	rec := httptest.NewRecorder()
	wrapped(rec, req)

	if called {
		t.Error("non-loopback + empty store must NOT reach the handler (silent auth bypass)")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for non-loopback unauth bind, got %d", rec.Code)
	}
}

// TestAuth_EmptyStore_NonLoopbackExplicitOptOut verifies the operator can
// still run unauthenticated on a non-loopback bind, but only by setting
// NIAC_AUTH_DISABLED=true explicitly (#739).
func TestAuth_EmptyStore_NonLoopbackExplicitOptOut(t *testing.T) {
	t.Setenv("NIAC_AUTH_DISABLED", "true")
	server := createTestServerForMiddleware(t)
	server.tokens = NewTokenStore(nil)
	server.cfg.Addr = "0.0.0.0:8445"

	called := false
	wrapped := server.auth(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	rec := httptest.NewRecorder()
	wrapped(rec, req)

	if !called || rec.Code != http.StatusOK {
		t.Errorf("explicit NIAC_AUTH_DISABLED=true should bypass: called=%v code=%d", called, rec.Code)
	}
}
