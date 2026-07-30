package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api/ratelimit"
	"github.com/MustardSeedNetworks/niac-go/internal/api/tokenstore"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

func attachDraftLibrary(t *testing.T, server *Server) *library.Library {
	t.Helper()
	lib, err := library.Open(t.TempDir())
	if err != nil {
		t.Fatalf("library.Open() error = %v", err)
	}
	server.library = lib
	return lib
}

func draftRequest(method, target, body, revision string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if revision != "" {
		req.Header.Set("If-Match", quoteDraftETag(revision))
	}
	return req
}

func decodeDraftResponse(t *testing.T, rec *httptest.ResponseRecorder) library.Draft {
	t.Helper()
	var draft library.Draft
	if err := json.NewDecoder(rec.Body).Decode(&draft); err != nil {
		t.Fatalf("decode draft response: %v; body=%s", err, rec.Body.String())
	}
	return draft
}

func TestDraftHandlersLifecycleAndRuntimeIsolation(t *testing.T) {
	server, configPath := newTestServer(t)
	attachDraftLibrary(t, server)
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read active config: %v", err)
	}
	applied := false
	server.cfg.ApplyConfig = func(_ *config.Config) error {
		applied = true
		return nil
	}

	createBody := fmt.Sprintf(`{"name":"branch-office","content":%s}`, strconvJSON(baseConfigYAML))
	createRec := httptest.NewRecorder()
	server.handleLibraryDrafts(createRec, draftRequest(http.MethodPost, "/api/v1/library/drafts", createBody, ""))
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body=%s", createRec.Code, createRec.Body.String())
	}
	created := decodeDraftResponse(t, createRec)
	if got := createRec.Header().Get("ETag"); got != quoteDraftETag(created.Revision) {
		t.Fatalf("create ETag = %q, want %q", got, quoteDraftETag(created.Revision))
	}
	listRec := httptest.NewRecorder()
	server.handleLibraryDrafts(listRec, draftRequest(http.MethodGet, "/api/v1/library/drafts", "", ""))
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body=%s", listRec.Code, listRec.Body.String())
	}
	var entries []library.DraftEntry
	if decodeErr := json.NewDecoder(listRec.Body).Decode(&entries); decodeErr != nil {
		t.Fatalf("decode list response: %v", decodeErr)
	}
	if len(entries) != 1 || entries[0].Revision != created.Revision {
		t.Fatalf("list response = %+v", entries)
	}

	missingRevisionRec := httptest.NewRecorder()
	server.handleLibraryDraftByName(missingRevisionRec, draftRequest(
		http.MethodPut,
		"/api/v1/library/drafts/branch-office",
		fmt.Sprintf(`{"content":%s}`, strconvJSON(updatedConfigYAML)),
		"",
	))
	if missingRevisionRec.Code != http.StatusPreconditionRequired {
		t.Fatalf("replace without If-Match status = %d, want 428", missingRevisionRec.Code)
	}

	staleRec := httptest.NewRecorder()
	server.handleLibraryDraftByName(staleRec, draftRequest(
		http.MethodPut,
		"/api/v1/library/drafts/branch-office",
		fmt.Sprintf(`{"content":%s}`, strconvJSON(updatedConfigYAML)),
		"stale",
	))
	if staleRec.Code != http.StatusPreconditionFailed {
		t.Fatalf("stale replace status = %d, want 412", staleRec.Code)
	}

	replaceRec := httptest.NewRecorder()
	server.handleLibraryDraftByName(replaceRec, draftRequest(
		http.MethodPut,
		"/api/v1/library/drafts/branch-office",
		fmt.Sprintf(`{"content":%s}`, strconvJSON(updatedConfigYAML)),
		created.Revision,
	))
	if replaceRec.Code != http.StatusOK {
		t.Fatalf("replace status = %d, body=%s", replaceRec.Code, replaceRec.Body.String())
	}
	replaced := decodeDraftResponse(t, replaceRec)
	if replaced.Revision == created.Revision {
		t.Fatal("replace did not advance revision")
	}

	readRec := httptest.NewRecorder()
	server.handleLibraryDraftByName(readRec, draftRequest(
		http.MethodGet, "/api/v1/library/drafts/branch-office", "", "",
	))
	if readRec.Code != http.StatusOK {
		t.Fatalf("read status = %d, body=%s", readRec.Code, readRec.Body.String())
	}
	if read := decodeDraftResponse(t, readRec); read.Revision != replaced.Revision {
		t.Fatalf("read revision = %q, want %q", read.Revision, replaced.Revision)
	}

	deleteRec := httptest.NewRecorder()
	server.handleLibraryDraftByName(deleteRec, draftRequest(
		http.MethodDelete, "/api/v1/library/drafts/branch-office", "", replaced.Revision,
	))
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, body=%s", deleteRec.Code, deleteRec.Body.String())
	}

	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read active config after draft operations: %v", err)
	}
	if applied || string(after) != string(before) || server.configPath() != configPath {
		t.Fatal("draft operations changed or applied the running configuration")
	}
}

func TestDraftCreateAcceptsGeneratedFleetBody(t *testing.T) {
	server, _ := newTestServer(t)
	attachDraftLibrary(t, server)
	content := baseConfigYAML + strings.Repeat("# generated fleet padding\n", 50_000)
	body := fmt.Sprintf(`{"name":"generated-fleet","content":%s}`, strconvJSON(content))
	if len(body) <= MaxRequestBodySize {
		t.Fatalf("request body = %d bytes, want more than legacy limit %d", len(body), MaxRequestBodySize)
	}
	rec := httptest.NewRecorder()

	server.handleLibraryDrafts(rec, draftRequest(
		http.MethodPost, "/api/v1/library/drafts", body, "",
	))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if draft := decodeDraftResponse(t, rec); draft.Content != content {
		t.Fatalf("draft content = %d bytes, want %d", len(draft.Content), len(content))
	}
}

func TestDraftCreateRejectsOversizedScenarioContent(t *testing.T) {
	server, _ := newTestServer(t)
	attachDraftLibrary(t, server)
	content := baseConfigYAML + strings.Repeat("# x\n", MaxScenarioConfigSize/4+1)
	body := fmt.Sprintf(`{"name":"oversized","content":%s}`, strconvJSON(content))
	rec := httptest.NewRecorder()

	server.handleLibraryDrafts(rec, draftRequest(
		http.MethodPost, "/api/v1/library/drafts", body, "",
	))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestDraftBehaviorReplacementPersistsValidatedTimeline(t *testing.T) {
	server, _ := newTestServer(t)
	lib := attachDraftLibrary(t, server)
	content := `devices:
  - name: access-1
    type: switch
    vendor: cisco
    ips: [192.0.2.10]
    interfaces:
      - name: Gi0/48
        speed: 10000
`
	draft, err := lib.CreateDraft("behavior-lab", content)
	if err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}
	body := `{"timelines":[{"name":"uplink degradation","startOffsetMs":1000,"repeatCount":2,"phases":[{"name":"congested","startOffsetMs":0,"durationMs":3000,"reset":true,"traffic":[{"device":"access-1","interface":"Gi0/48","utilization":85}],"faults":[{"device":"access-1","interface":"Gi0/48","type":"packet_discards","value":12}]}]}]}`
	rec := httptest.NewRecorder()
	server.handleLibraryDraftByName(rec, draftRequest(
		http.MethodPut, "/api/v1/library/drafts/behavior-lab/behaviors", body, draft.Revision,
	))
	if rec.Code != http.StatusOK {
		t.Fatalf("replace behaviors status = %d, body=%s", rec.Code, rec.Body.String())
	}
	updated := decodeDraftResponse(t, rec)
	cfg, err := config.LoadYAMLBytes([]byte(updated.Content))
	if err != nil {
		t.Fatalf("LoadYAMLBytes() error = %v\n%s", err, updated.Content)
	}
	if len(cfg.BehaviorTimelines) != 1 || cfg.BehaviorTimelines[0].RepeatCount != 2 ||
		cfg.BehaviorTimelines[0].Phases[0].Traffic[0].Utilization != 85 {
		t.Fatalf("saved behavior timelines = %+v", cfg.BehaviorTimelines)
	}
}

func TestDraftBehaviorReplacementRejectsUnknownTarget(t *testing.T) {
	server, _ := newTestServer(t)
	lib := attachDraftLibrary(t, server)
	draft, err := lib.CreateDraft("behavior-lab", baseConfigYAML)
	if err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}
	body := `{"timelines":[{"name":"invalid","startOffsetMs":0,"repeatCount":1,"phases":[{"name":"phase","startOffsetMs":0,"durationMs":1000,"reset":true,"traffic":[{"device":"missing","interface":"Gi0/1","utilization":85}],"faults":[]}]}]}`
	rec := httptest.NewRecorder()
	server.handleLibraryDraftByName(rec, draftRequest(
		http.MethodPut, "/api/v1/library/drafts/behavior-lab/behaviors", body, draft.Revision,
	))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid behavior status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	stored, err := lib.ReadDraft("behavior-lab")
	if err != nil {
		t.Fatalf("ReadDraft() error = %v", err)
	}
	if stored.Revision != draft.Revision {
		t.Fatal("invalid behavior replacement changed the draft")
	}
}

func TestDraftCreateValidatesConfigAndEntitlementsBeforePersistence(t *testing.T) {
	server, _ := newTestServer(t)
	lib := attachDraftLibrary(t, server)

	invalidRec := httptest.NewRecorder()
	server.handleLibraryDrafts(invalidRec, draftRequest(
		http.MethodPost,
		"/api/v1/library/drafts",
		`{"name":"invalid","content":"devices: ["}`,
		"",
	))
	if invalidRec.Code != http.StatusBadRequest {
		t.Fatalf("invalid config status = %d, want 400", invalidRec.Code)
	}
	if _, err := lib.ReadDraft("invalid"); !errors.Is(err, library.ErrNotFound) {
		t.Fatalf("invalid config persisted, ReadDraft() error = %v", err)
	}

	var oversized strings.Builder
	oversized.WriteString("devices:\n")
	const deviceYAML = "  - name: device-%d\n" +
		"    mac: \"02:00:00:00:00:%02x\"\n" +
		"    ips: [\"192.0.2.%d\"]\n"
	for index := range FreeTierDeviceCount + 1 {
		fmt.Fprintf(&oversized, deviceYAML, index, index, index+1)
	}
	licensedRec := httptest.NewRecorder()
	server.handleLibraryDrafts(licensedRec, draftRequest(
		http.MethodPost,
		"/api/v1/library/drafts",
		fmt.Sprintf(`{"name":"too-large","content":%s}`, strconvJSON(oversized.String())),
		"",
	))
	if licensedRec.Code != http.StatusPaymentRequired {
		t.Fatalf("unlicensed config status = %d, want 402; body=%s", licensedRec.Code, licensedRec.Body.String())
	}
	if _, err := lib.ReadDraft("too-large"); !errors.Is(err, library.ErrNotFound) {
		t.Fatalf("unlicensed config persisted, ReadDraft() error = %v", err)
	}
}

func TestDraftCreateFromTemplateMaterializesRelativeResources(t *testing.T) {
	server, _ := newTestServer(t)
	lib := attachDraftLibrary(t, server)
	templateDir := t.TempDir()
	t.Setenv("NIAC_TEMPLATES_DIR", templateDir)
	walkPath := filepath.Join(templateDir, "switch.walk")
	if err := os.WriteFile(walkPath, []byte("SNMPv2-MIB::sysName.0 = STRING: access-1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	templateYAML := `include_path: "."
devices:
  - name: access-1
    type: switch
    mac: "02:00:00:00:00:01"
    snmp_agent:
      community: "public"
      walk_file: "switch.walk"
`
	if err := os.WriteFile(filepath.Join(templateDir, "relative.yaml"), []byte(templateYAML), 0o600); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	server.handleLibraryDrafts(rec, draftRequest(
		http.MethodPost,
		"/api/v1/library/drafts",
		`{"name":"from-template","templateName":"relative"}`,
		"",
	))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body=%s", rec.Code, rec.Body.String())
	}
	draft, err := lib.ReadDraft("from-template")
	if err != nil {
		t.Fatalf("ReadDraft() error = %v", err)
	}
	resolvedTemplateDir, err := filepath.EvalSymlinks(templateDir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(draft.Content, "include_path: "+resolvedTemplateDir) {
		t.Fatalf(
			"materialized draft does not contain resolved include path %q:\n%s",
			resolvedTemplateDir,
			draft.Content,
		)
	}
	resolvedWalkPath := filepath.Join(resolvedTemplateDir, filepath.Base(walkPath))
	if !strings.Contains(draft.Content, resolvedWalkPath) {
		t.Fatalf(
			"materialized draft does not contain resolved walk path %q:\n%s",
			resolvedWalkPath,
			draft.Content,
		)
	}
	if _, loadErr := config.LoadYAMLBytes([]byte(draft.Content)); loadErr != nil {
		t.Fatalf("materialized draft is not independently loadable: %v", loadErr)
	}
	if _, loadErr := config.LoadYAMLBytesManaged(
		[]byte(draft.Content), t.TempDir(), []string{templateDir},
	); loadErr != nil {
		t.Fatalf("materialized draft does not survive managed inline loading: %v", loadErr)
	}
}

func TestDraftRoutesEnforceAuthScopeCSRFAndWriteRatePolicy(t *testing.T) {
	server, _, rwToken := newTestServerWithAuth(t)
	attachDraftLibrary(t, server)
	server.writeLimiter = ratelimit.NewRateLimiter(WriteRateLimit, WriteBurst)
	mux := http.NewServeMux()
	server.registerAPIRoutes(mux)
	body := fmt.Sprintf(`{"name":"secure","content":%s}`, strconvJSON(baseConfigYAML))

	unauthRec := httptest.NewRecorder()
	mux.ServeHTTP(unauthRec, draftRequest(http.MethodPost, "/api/v1/library/drafts", body, ""))
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401", unauthRec.Code)
	}

	const roToken = "read-only-draft-token"
	server.SetTokens([]tokenstore.ScopedToken{
		{Value: roToken, Scope: tokenstore.ScopeReadOnly},
		{Value: rwToken, Scope: tokenstore.ScopeReadWrite},
	})
	readOnlyReq := draftRequest(http.MethodPost, "/api/v1/library/drafts", body, "")
	readOnlyReq.Header.Set("Authorization", "Bearer "+roToken)
	readOnlyReq.Header.Set("X-Csrf-Token", testCSRFToken(t, server, roToken))
	readOnlyRec := httptest.NewRecorder()
	mux.ServeHTTP(readOnlyRec, readOnlyReq)
	if readOnlyRec.Code != http.StatusForbidden {
		t.Fatalf("read-only mutation status = %d, want 403", readOnlyRec.Code)
	}

	missingCSRFReq := draftRequest(http.MethodPost, "/api/v1/library/drafts", body, "")
	missingCSRFReq.Header.Set("Authorization", "Bearer "+rwToken)
	missingCSRFRec := httptest.NewRecorder()
	mux.ServeHTTP(missingCSRFRec, missingCSRFReq)
	if missingCSRFRec.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF status = %d, want 403", missingCSRFRec.Code)
	}

	validReq := draftRequest(http.MethodPost, "/api/v1/library/drafts", body, "")
	validReq.Header.Set("Authorization", "Bearer "+rwToken)
	validReq.Header.Set("X-Csrf-Token", testCSRFToken(t, server, rwToken))
	validRec := httptest.NewRecorder()
	mux.ServeHTTP(validRec, validReq)
	if validRec.Code != http.StatusCreated {
		t.Fatalf("authorized status = %d, want 201; body=%s", validRec.Code, validRec.Body.String())
	}

	policy := fetchRouteManifest(t)["/api/v1/library/drafts"]
	if !policy.CSRF || !policy.RateLimited || policy.Admin {
		t.Fatalf("draft route policy = %+v, want csrf+rateLimited without admin", policy)
	}
}
