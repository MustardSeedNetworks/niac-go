package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/niac-go/internal/api/ratelimit"
	"github.com/MustardSeedNetworks/niac-go/internal/api/sse"
)

// TestRoutePolicyManifest verifies the capability registry exposes every route
// via /__capabilities and records each route's policy correctly. The registry
// is the single source of truth that foundation's check-route-policy.sh enforces.
func TestRoutePolicyManifest(t *testing.T) {
	server, _, _ := newTestServerWithAuth(t)
	server.writeLimiter = ratelimit.NewRateLimiter(WriteRateLimit, WriteBurst)
	mux := server.apiHandler()

	req := httptest.NewRequest(http.MethodGet, "/__capabilities", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /__capabilities: status = %d, want 200", rec.Code)
	}

	var views []route.Policy
	if err := json.Unmarshal(rec.Body.Bytes(), &views); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if len(views) == 0 {
		t.Fatal("expected a non-empty route manifest")
	}

	byPath := make(map[string]route.Policy, len(views))
	for _, v := range views {
		byPath[v.Path] = v
	}

	// Whole-topology import is admin-scoped + CSRF-protected + write-limited.
	imp, impOK := byPath["/api/v1/config/import"]
	if !impOK || imp.Scope != scopeAdmin || !imp.CSRF || !imp.RateLimited {
		t.Errorf("/api/v1/config/import policy = %+v, want admin+csrf+rateLimited", imp)
	}
	// Whole-library install (#897 L3b) is the same admin-class shape as
	// config/import — it replaces networks/walks/pcaps content wholesale.
	inst, instOK := byPath["/api/v1/library/install"]
	if !instOK || inst.Scope != scopeAdmin || !inst.CSRF || !inst.RateLimited {
		t.Errorf("/api/v1/library/install policy = %+v, want admin+csrf+rateLimited", inst)
	}
	// A safe read carries none of those.
	rd, rdOK := byPath["/api/v1/topology"]
	if !rdOK || rd.Scope != "" || rd.CSRF || rd.RateLimited {
		t.Errorf("/api/v1/topology policy = %+v, want no admin/csrf/rateLimited", rd)
	}
}

func TestTemplateUseRoutePolicy(t *testing.T) {
	use, ok := fetchRouteManifest(t)["/api/v1/templates/use"]
	if !ok || !use.CSRF || !use.RateLimited {
		t.Errorf("/api/v1/templates/use policy = %+v, want csrf+rateLimited", use)
	}
}

func TestErrorsRoutePolicy(t *testing.T) {
	errorsRoute, ok := fetchRouteManifest(t)["/api/v1/errors"]
	if !ok || !errorsRoute.CSRF || !errorsRoute.RateLimited {
		t.Errorf("/api/v1/errors policy = %+v, want csrf+rateLimited", errorsRoute)
	}
}

// TestRoutePolicyManifestMethodAndBody verifies the ADR-0002 parity additions:
// every route reports a non-zero body cap and its accepted methods, upload /
// replay routes carry the larger PCAP cap (not the 1MB default), and method-
// gated routes report the exact method set the registry enforces.
func TestRoutePolicyManifestMethodAndBody(t *testing.T) {
	byPath := fetchRouteManifest(t)

	// Every route must record a non-zero body cap (the registrar defaults 0 to
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
	// Content-bundle install carries its own (larger) cap for the same
	// base64-expansion reason as pcap/replay.
	if v := byPath["/api/v1/library/install"]; v.MaxBodyBytes != int64(MaxLibraryInstallBodySize) {
		t.Errorf("/api/v1/library/install: maxBodyBytes = %d, want MaxLibraryInstallBodySize (%d)",
			v.MaxBodyBytes, int64(MaxLibraryInstallBodySize))
	}
	for _, path := range []string{
		"/api/v1/library/drafts",
		"/api/v1/library/drafts/",
		"/api/v1/simulation/preflight",
		"/api/v1/simulation",
	} {
		if got := byPath[path].MaxBodyBytes; got != int64(MaxScenarioRequestBodySize) {
			t.Errorf("%s: maxBodyBytes = %d, want MaxScenarioRequestBodySize (%d)",
				path, got, int64(MaxScenarioRequestBodySize))
		}
	}

	// Method-gated routes report their methods: a multi-method dispatcher
	// declares its full set, a single-method route declares exactly one.
	wantMethods := map[string][]string{
		"/api/v1/config": {http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodPost},
		"/api/v1/config/devices": {
			http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
		},
		"/api/v1/replay": {
			http.MethodGet, http.MethodPost, http.MethodDelete,
		},
		"/api/v1/simulation": {
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
func fetchRouteManifest(t *testing.T) map[string]route.Policy {
	t.Helper()
	server, _, _ := newTestServerWithAuth(t)
	server.writeLimiter = ratelimit.NewRateLimiter(WriteRateLimit, WriteBurst)
	mux := server.apiHandler()

	req := httptest.NewRequest(http.MethodGet, "/__capabilities", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /__capabilities: status = %d, want 200", rec.Code)
	}

	var views []route.Policy
	if err := json.Unmarshal(rec.Body.Bytes(), &views); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	byPath := make(map[string]route.Policy, len(views))
	for _, v := range views {
		byPath[v.Path] = v
	}
	return byPath
}

// TestMethodGateRejectsWrongMethod verifies the registrar's method gate returns 405
// with an Allow header for a method outside the route's declared set, exercising
// the declarative path that replaced the in-handler guards.
func TestMethodGateRejectsWrongMethod(t *testing.T) {
	server, _, token := newTestServerWithAuth(t)
	server.writeLimiter = ratelimit.NewRateLimiter(WriteRateLimit, WriteBurst)
	mux := server.apiHandler()

	// /api/v1/topology is GET-only; a DELETE (with a read-write bearer so it
	// clears auth's scope-by-method check) must 405 with Allow: GET from the
	// registry's method gate, not pass through to the handler.
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

func TestSPAIsPublicButAPIRemainsAuthenticated(t *testing.T) {
	server, _, token := newTestServerWithAuth(t)
	server.writeLimiter = ratelimit.NewRateLimiter(WriteRateLimit, WriteBurst)
	mux := server.apiHandler()

	spaReq := httptest.NewRequest(http.MethodGet, "https://niac.example/", nil)
	spaRec := httptest.NewRecorder()
	mux.ServeHTTP(spaRec, spaReq)
	if spaRec.Code == http.StatusUnauthorized {
		t.Fatal("GET / without bearer returned 401; the SPA cannot bootstrap its auth prompt")
	}
	for _, header := range []string{
		"Content-Security-Policy",
		"Strict-Transport-Security",
		"X-Content-Type-Options",
		"X-Frame-Options",
	} {
		if spaRec.Header().Get(header) == "" {
			t.Errorf("GET / without bearer omitted %s", header)
		}
	}

	apiReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/scope", nil)
	apiRec := httptest.NewRecorder()
	mux.ServeHTTP(apiRec, apiReq)
	if apiRec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/v1/auth/scope without bearer: status = %d, want 401", apiRec.Code)
	}

	missingReq := httptest.NewRequest(http.MethodGet, "/api/v1/missing", nil)
	missingReq.Header.Set("Authorization", "Bearer "+token)
	missingRec := httptest.NewRecorder()
	mux.ServeHTTP(missingRec, missingReq)
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("GET unknown /api path with bearer: status = %d, want 404", missingRec.Code)
	}
	if got := missingRec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("GET unknown /api path Content-Type = %q, want application/json", got)
	}
}

// TestRequestIDIsOnePerRequest pins that the registrar's request ID is the
// only one: the client's X-Request-ID, the error body's requestId and every
// log line about the request (the access log and auth's refusal) name the
// same value, so an operator can join a client report to the log.
func TestRequestIDIsOnePerRequest(t *testing.T) {
	server, _, _ := newTestServerWithAuth(t)
	var logs bytes.Buffer
	server.logger = slog.New(slog.NewJSONHandler(&logs, nil))
	handler := server.apiHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/scope", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/v1/auth/scope without bearer: status = %d, want 401", rec.Code)
	}
	id := rec.Header().Get("X-Request-ID")
	if id == "" {
		t.Fatal("response carries no X-Request-ID")
	}
	var body ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if body.RequestID != id {
		t.Errorf("error body requestId = %q, want the header's %q", body.RequestID, id)
	}

	named := 0
	for line := range bytes.Lines(logs.Bytes()) {
		var entry map[string]any
		if err := json.Unmarshal(line, &entry); err != nil {
			t.Fatalf("log line is not JSON: %s", line)
		}
		for _, key := range []string{"request_id", "requestID"} {
			if got, ok := entry[key]; ok {
				named++
				if got != id {
					t.Errorf("log %q names %s = %v, want %q", entry["msg"], key, got, id)
				}
			}
		}
	}
	if named < 2 {
		t.Errorf("%d log fields named the request, want the access log and auth's refusal:\n%s", named, logs.String())
	}
}

// TestSSEStreamsThroughTheRegistrar pins that the registrar's response-writer
// wrapping keeps streaming alive: the stream's first event reaches the client
// while the handler is still running, which only happens if Flush gets through
// the access-log and recovery wrappers to net/http's writer.
func TestSSEStreamsThroughTheRegistrar(t *testing.T) {
	server, _, token := newTestServerWithAuth(t)
	server.sseHub = sse.NewHub(sse.Config{})
	go server.sseHub.Run()
	t.Cleanup(server.sseHub.Stop)
	ts := httptest.NewServer(server.apiHandler())
	t.Cleanup(ts.Close)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/v1/stream/logs", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /api/v1/stream/logs: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}
	line, err := bufio.NewReader(resp.Body).ReadString('\n')
	if err != nil {
		t.Fatalf("no event arrived before the deadline: %v", err)
	}
	if line != "event: connected\n" {
		t.Errorf("first line = %q, want the connected event", line)
	}
}
