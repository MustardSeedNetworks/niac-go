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
	server.csrfToken = "test-token"

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
	server.csrfToken = "valid-csrf-token"

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
			req.Header.Set("X-Csrf-Token", "valid-csrf-token")
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
	server.csrfToken = "" // No CSRF token set

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
	server.cfg.Token = "secret-api-token"
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
	server.cfg.Token = "secret-api-token"
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
	server.cfg.Token = "secret-api-token"
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
	server.cfg.Token = "" // No API token configured
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
