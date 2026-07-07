package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
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
	imp, impOK := byPath["/api/v1/config/import"]
	if !impOK || !imp.Admin || !imp.CSRF || !imp.RateLimited {
		t.Errorf("/api/v1/config/import policy = %+v, want admin+csrf+rateLimited", imp)
	}
	// A safe read carries none of those.
	rd, rdOK := byPath["/api/v1/topology"]
	if !rdOK || rd.Admin || rd.CSRF || rd.RateLimited {
		t.Errorf("/api/v1/topology policy = %+v, want no admin/csrf/rateLimited", rd)
	}
}

// TestRoutePolicyManifestMethodAndBody verifies the ADR-0002 parity additions:
// every route reports a non-zero body cap and its accepted methods, upload /
// replay routes carry the larger PCAP cap (not the 1MB default), and method-
// gated routes report the exact method set the registry enforces.
func TestRoutePolicyManifestMethodAndBody(t *testing.T) {
	byPath := fetchRouteManifest(t)

	// Every route must record a non-zero body cap (register() defaults 0 to
	// MaxRequestBodySize) and must declare its accepted methods.
	for _, v := range byPath {
		if v.MaxBodyBytes == 0 {
			t.Errorf("%s: maxBodyBytes = 0, want a non-zero cap (default %d)",
				v.Path, int64(MaxRequestBodySize))
		}
		if len(v.Methods) == 0 {
			t.Errorf("%s: methods empty, want declared HTTP methods", v.Path)
		}
	}

	// Upload / replay routes accept inline PCAP payloads and MUST carry the
	// larger cap, not the 1MB default — a regression here silently truncates
	// valid captures.
	for _, p := range []string{"/api/v1/pcap/upload", "/api/v1/replay"} {
		if v := byPath[p]; v.MaxBodyBytes != int64(MaxPCAPUploadBodySize) {
			t.Errorf("%s: maxBodyBytes = %d, want MaxPCAPUploadBodySize (%d)",
				p, v.MaxBodyBytes, int64(MaxPCAPUploadBodySize))
		}
	}

	// Method-gated routes report their methods: a multi-method dispatcher
	// declares its full set, a single-method route declares exactly one.
	wantMethods := map[string][]string{
		"/api/v1/config": {http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodPost},
		"/api/v1/config/devices": {
			http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
		},
		"/api/v1/topology": {http.MethodGet},
	}
	for p, want := range wantMethods {
		if got := byPath[p].Methods; !slices.Equal(got, want) {
			t.Errorf("%s methods = %v, want %v", p, got, want)
		}
	}
}

// fetchRouteManifest registers the full route table and returns the
// /__capabilities manifest keyed by path.
func fetchRouteManifest(t *testing.T) map[string]routePolicyView {
	t.Helper()
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
	byPath := make(map[string]routePolicyView, len(views))
	for _, v := range views {
		byPath[v.Path] = v
	}
	return byPath
}

// TestMethodGateRejectsWrongMethod verifies register()'s methodGate returns 405
// with an Allow header for a method outside the route's declared set, exercising
// the declarative path that replaced the in-handler guards.
func TestMethodGateRejectsWrongMethod(t *testing.T) {
	server, _, token := newTestServerWithAuth(t)
	server.writeLimiter = ratelimit.NewRateLimiter(WriteRateLimit, WriteBurst)
	mux := http.NewServeMux()
	server.registerAPIRoutes(mux)

	// /api/v1/topology is GET-only; a DELETE (with a read-write bearer so it
	// clears auth's scope-by-method check) must 405 with Allow: GET from the
	// registry's methodGate, not pass through to the handler.
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/topology", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("DELETE /api/v1/topology: status = %d, want 405", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); allow != http.MethodGet {
		t.Errorf("Allow header = %q, want %q", allow, http.MethodGet)
	}
}
