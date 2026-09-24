package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/foundation/pkg/csrf"

	"github.com/MustardSeedNetworks/niac-go/internal/api/ratelimit"

	"github.com/MustardSeedNetworks/niac-go/internal/api/tokenstore"
)

// TestAuthMiddleware_PublicPaths_NoAuthRequired confirms /__version
// stays unauthenticated even with a populated TokenStore. The Wave 2
// allowlist is identical to Wave 1's — the only public endpoint is
// /__version (and /healthz if it ever lands).
func TestAuthMiddleware_PublicPaths_NoAuthRequired(t *testing.T) {
	server := createTestServerForMiddleware(t)
	server.SetTokens([]tokenstore.ScopedToken{{Value: "rw-secret", Scope: tokenstore.ScopeReadWrite}})
	server.rateLimiter = ratelimit.NewRateLimiter(100, 200)
	server.csrf = csrf.NewManager()
	t.Cleanup(server.csrf.Stop)

	// /__version is registered without Auth, so a request carrying no bearer
	// reaches the handler through the real route table.
	req := httptest.NewRequest(http.MethodGet, "/__version", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	rec := httptest.NewRecorder()

	server.apiHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/__version unauthed: status = %d, want 200", rec.Code)
	}
}
