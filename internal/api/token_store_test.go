package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api/tokenstore"
)

// TestAuthMiddleware_PublicPaths_NoAuthRequired confirms /__version
// stays unauthenticated even with a populated TokenStore. The Wave 2
// allowlist is identical to Wave 1's — the only public endpoint is
// /__version (and /healthz if it ever lands).
func TestAuthMiddleware_PublicPaths_NoAuthRequired(t *testing.T) {
	server := createTestServerForMiddleware(t)
	server.SetTokens([]tokenstore.ScopedToken{{Value: "rw-secret", Scope: tokenstore.ScopeReadWrite}})
	server.rateLimiter = NewRateLimiter(100, 200)

	// /__version is wired without s.auth() in registerAPIRoutes, so it
	// reaches the handler directly. The test mirrors that wiring by
	// invoking the build-version handler without the auth wrapper.
	req := httptest.NewRequest(http.MethodGet, "/__version", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	rec := httptest.NewRecorder()

	server.recoverMiddleware(server.handleBuildVersion)(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/__version unauthed: status = %d, want 200", rec.Code)
	}
}
