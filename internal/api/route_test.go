package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api/ratelimit"
)

// TestRoutePolicyManifest verifies the capability registry exposes every route
// via /__capabilities and records each route's policy correctly. The registry
// is the single source of truth that scripts/check-route-policy.sh enforces.
func TestRoutePolicyManifest(t *testing.T) {
	server, _, _ := newTestServerWithAuth(t)
	server.writeLimiter = ratelimit.NewRateLimiter(WriteRateLimit, WriteBurst)
	mux := http.NewServeMux()
	server.registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/__capabilities", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /__capabilities: status = %d, want 200", rec.Code)
	}

	var views []routePolicyView
	if err := json.Unmarshal(rec.Body.Bytes(), &views); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if len(views) == 0 {
		t.Fatal("expected a non-empty route manifest")
	}

	byPath := make(map[string]routePolicyView, len(views))
	for _, v := range views {
		byPath[v.Path] = v
	}

	// Whole-topology import is admin-scoped + CSRF-protected + write-limited.
	imp, ok := byPath["/api/v1/config/import"]
	if !ok || !imp.Admin || !imp.CSRF || !imp.RateLimited {
		t.Errorf("/api/v1/config/import policy = %+v, want admin+csrf+rateLimited", imp)
	}
	// A safe read carries none of those.
	rd, ok := byPath["/api/v1/topology"]
	if !ok || rd.Admin || rd.CSRF || rd.RateLimited {
		t.Errorf("/api/v1/topology policy = %+v, want no admin/csrf/rateLimited", rd)
	}
}
