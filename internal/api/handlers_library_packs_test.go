package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api/ratelimit"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// newPackLibraryMux builds the registered mux over a server that has both auth
// and a library root, so a request crosses the same middleware a browser's
// does. The registry's per-route body cap runs before any handler, so a test
// that calls the handler directly cannot see the defect this file covers
// (#2203: the named-network route kept the 1 MiB default while every other
// YAML-bearing route carries the authored-scenario cap).
func newPackLibraryMux(t *testing.T) (*Server, http.Handler, string, string) {
	t.Helper()
	server, _, token := newTestServerWithAuth(t)
	server.writeLimiter = ratelimit.NewRateLimiter(WriteRateLimit, WriteBurst)

	root := t.TempDir()
	t.Setenv("NIAC_LIBRARY_ROOT", root)
	lib, err := library.Open(root)
	if err != nil {
		t.Fatalf("open library: %v", err)
	}
	server.library = lib
	server.logger = slog.Default()

	mux := server.apiHandler()

	return server, mux, token, root
}

func packPost(
	t *testing.T,
	server *Server,
	mux http.Handler,
	token, path string,
	payload any,
) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s request: %v", path, err)
	}

	return packPostRaw(t, server, mux, token, path, body)
}

func packPostRaw(
	t *testing.T,
	server *Server,
	mux http.Handler,
	token, path string,
	body []byte,
) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Csrf-Token", testCSRFToken(t, server, token))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	return rec
}

// TestEveryScenarioPackSavesThroughTheRegisteredNamedNetworkRoute is the
// regression for #2203. Five of the seven shipped packs generate a YAML larger
// than 1 MiB, so saving one as a named network answered 413 and the flagship
// Hospital pack could not be kept in the library the daemon starts from. The
// draft creates are asserted in the same loop because they already worked and
// must keep working: the two routes carry one YAML contract.
func TestEveryScenarioPackSavesThroughTheRegisteredNamedNetworkRoute(t *testing.T) {
	server, mux, token, root := newPackLibraryMux(t)

	packs := scenario.Packs()
	if len(packs) != 7 {
		t.Fatalf("packs = %d, want the seven shipped packs", len(packs))
	}

	for _, pack := range packs {
		t.Run(pack.ID, func(t *testing.T) {
			generated := packPost(t, server, mux, token, "/api/v1/scenario/generate", pack.Request)
			if generated.Code != http.StatusOK {
				t.Fatalf("generate %s: status = %d, want 200; body=%s",
					pack.ID, generated.Code, generated.Body.String())
			}
			var response scenarioGenerateResponse
			if err := json.NewDecoder(generated.Body).Decode(&response); err != nil {
				t.Fatalf("decode generated %s: %v", pack.ID, err)
			}

			draft := packPost(t, server, mux, token, "/api/v1/library/drafts",
				map[string]string{"name": pack.ID + "-draft", "content": response.Content})
			if draft.Code != http.StatusCreated {
				t.Fatalf("draft %s (%d bytes): status = %d, want 201; body=%s",
					pack.ID, len(response.Content), draft.Code, draft.Body.String())
			}

			saved := packPost(t, server, mux, token, "/api/v1/library/networks",
				libraryNetworkUploadRequest{Name: pack.ID, Content: response.Content})
			if saved.Code != http.StatusCreated {
				t.Fatalf("save %s (%d bytes): status = %d, want 201; body=%s",
					pack.ID, len(response.Content), saved.Code, saved.Body.String())
			}

			// The artifact on disk is what a later start loads, so prove it
			// parses rather than trusting the 201.
			onDisk, err := os.ReadFile(filepath.Join(root, "networks", pack.ID+".yaml"))
			if err != nil {
				t.Fatalf("read saved %s: %v", pack.ID, err)
			}
			if _, err = config.LoadYAMLBytes(onDisk); err != nil {
				t.Fatalf("saved %s does not load: %v", pack.ID, err)
			}
		})
	}
}

// TestLibraryNetworkUploadStillBoundsOversizeContent pins the other half of
// #2203: the cap rises to the authored-scenario contract, it does not go away.
// The content bound and the body bound are distinct limits and both are
// asserted through the mux, since only one of them is the handler's.
func TestLibraryNetworkUploadStillBoundsOversizeContent(t *testing.T) {
	server, mux, token, _ := newPackLibraryMux(t)

	oversizeContent := packPost(t, server, mux, token, "/api/v1/library/networks",
		libraryNetworkUploadRequest{
			Name:    "oversize",
			Content: strings.Repeat("a", MaxScenarioConfigSize+1),
		})
	if oversizeContent.Code != http.StatusBadRequest {
		t.Fatalf("oversize content: status = %d, want 400", oversizeContent.Code)
	}
	if !strings.Contains(oversizeContent.Body.String(), "max_size_exceeded") {
		t.Fatalf("oversize content body = %s, want max_size_exceeded",
			oversizeContent.Body.String())
	}

	// Valid JSON, so the body cap is what rejects it rather than the parser
	// tripping over the first byte.
	oversizeBody := []byte(`{"name":"oversize","content":"` +
		strings.Repeat("a", MaxScenarioRequestBodySize+1) + `"}`)
	rejected := packPostRaw(t, server, mux, token, "/api/v1/library/networks", oversizeBody)
	if rejected.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversize body: status = %d, want 413; body=%s",
			rejected.Code, rejected.Body.String())
	}
}
