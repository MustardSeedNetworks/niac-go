package api

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api/csrf"
	"github.com/MustardSeedNetworks/niac-go/internal/api/ratelimit"
	"github.com/MustardSeedNetworks/niac-go/internal/api/tokenstore"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

// libraryTarballEntry mirrors internal/content's test helper (unexported
// there, so duplicated here) — a single file or directory to bake into a
// synthesised gzip-tar bundle.
type libraryTarballEntry struct {
	name string
	body string
	dir  bool
}

func buildLibraryTarball(t *testing.T, entries []libraryTarballEntry) []byte {
	t.Helper()
	var raw bytes.Buffer
	gz := gzip.NewWriter(&raw)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		hdr := &tar.Header{Name: e.name, Mode: 0o644, Size: int64(len(e.body))}
		if e.dir {
			hdr.Typeflag = tar.TypeDir
			hdr.Size = 0
		} else {
			hdr.Typeflag = tar.TypeReg
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if !e.dir {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatalf("write tar body: %v", err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gz: %v", err)
	}
	return raw.Bytes()
}

func validBundleJSONBody(t *testing.T) []byte {
	t.Helper()
	tarball := buildLibraryTarball(t, []libraryTarballEntry{
		{name: "walks/switch1.walk", body: "1.3.6.1.2.1.1.1.0 = STRING: test\n"},
		{name: "networks/lab.yaml", body: "devices: []\n"},
		{name: "pcaps/sample.pcap", body: "not-really-a-pcap-but-extract-doesnt-care"},
	})
	body, err := json.Marshal(LibraryInstallRequest{
		Filename: "bundle.tar.gz",
		Data:     base64.StdEncoding.EncodeToString(tarball),
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	return body
}

// newLibraryInstallServer wires a real mux (registerAPIRoutes) with an
// opened library rooted at a tmp dir and the write rate limiter the
// registry chain needs before it reaches csrf/admin checks — the same
// setup TestCSRFWiring_TemplatesAndLibraryNetworks uses. scope selects the
// bearer token's scope so tests can exercise both the admin-token success
// path and the non-admin rejection path.
func newLibraryInstallServer(t *testing.T, scope tokenstore.TokenScope) (*Server, *http.ServeMux, string) {
	t.Helper()
	server, tmpDir := newTestServer(t)
	server.rateLimiter = ratelimit.NewRateLimiter(DefaultRateLimit, DefaultBurst)
	server.csrf = csrf.NewManager()
	t.Cleanup(server.csrf.Stop)
	server.writeLimiter = ratelimit.NewRateLimiter(WriteRateLimit, WriteBurst)

	token := generateTestToken()
	server.cfg.Token = token
	server.SetTokens([]tokenstore.ScopedToken{{Value: token, Scope: scope}})

	lib, err := library.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open library: %v", err)
	}
	server.library = lib

	mux := http.NewServeMux()
	server.registerAPIRoutes(mux)
	_ = tmpDir
	return server, mux, token
}

func libraryInstallRequest(t *testing.T, server *Server, token string, body []byte) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/install", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Csrf-Token", testCSRFToken(t, server, token))
	return req
}

// TestLibraryInstall_ValidBundle drives the real mux with an admin-scoped
// token end-to-end: extraction lands on disk and the response manifest
// matches what was in the bundle.
func TestLibraryInstall_ValidBundle(t *testing.T) {
	server, mux, token := newLibraryInstallServer(t, tokenstore.ScopeAdmin)

	req := libraryInstallRequest(t, server, token, validBundleJSONBody(t))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	var resp LibraryInstallResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.Success || resp.Files != 3 {
		t.Fatalf("response = %+v, want success with 3 files", resp)
	}
	if resp.PerKind[library.KindWalks] != 1 || resp.PerKind[library.KindNetworks] != 1 ||
		resp.PerKind[library.KindPcaps] != 1 {
		t.Fatalf("perKind = %+v, want one entry per kind", resp.PerKind)
	}
}

// TestLibraryInstall_NonAdminRejected proves a read-write (non-admin) token
// is rejected — this is the whole-library-replace endpoint and must stay
// admin-class like /api/v1/config/import.
func TestLibraryInstall_NonAdminRejected(t *testing.T) {
	server, mux, token := newLibraryInstallServer(t, tokenstore.ScopeReadWrite)

	req := libraryInstallRequest(t, server, token, validBundleJSONBody(t))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 for non-admin token; body=%s", rec.Code, rec.Body.String())
	}
}

// TestLibraryInstall_MalformedBundle proves a non-gzip body comes back as a
// structured JSON error, not a 500 dump.
func TestLibraryInstall_MalformedBundle(t *testing.T) {
	server, mux, token := newLibraryInstallServer(t, tokenstore.ScopeAdmin)

	body, err := json.Marshal(LibraryInstallRequest{
		Filename: "bad.tar.gz",
		Data:     base64.StdEncoding.EncodeToString([]byte("not a gzip stream")),
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := libraryInstallRequest(t, server, token, body)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if unmarshalErr := json.Unmarshal(rec.Body.Bytes(), &payload); unmarshalErr != nil {
		t.Fatalf("response is not structured JSON: %v (body=%s)", unmarshalErr, rec.Body.String())
	}
	if payload["error"] != "bundle_invalid" {
		t.Fatalf("error code = %v, want bundle_invalid", payload["error"])
	}
}

// TestLibraryInstall_BodyTooLarge proves handleLibraryInstall's own
// decodeJSONStrict(..., MaxLibraryInstallBodySize) call rejects a body over
// its cap with a structured 413, the same mechanism route_test.go's
// TestRoutePolicyManifestMethodAndBody proves the registry wires at
// MaxLibraryInstallBodySize. Exercised at a scaled-down cap here (rather
// than transferring 700MB of filler) — decodeJSONStrict takes maxSize as a
// plain parameter, so the enforcement path is identical at any size.
func TestLibraryInstall_BodyTooLarge(t *testing.T) {
	const tinyCap = 64

	filler := bytes.Repeat([]byte{'A'}, tinyCap+16)
	var body bytes.Buffer
	body.WriteString(`{"filename":"oversized.tar.gz","data":"`)
	body.Write(filler)
	body.WriteString(`"}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/install", &body)
	rec := httptest.NewRecorder()

	var req2 LibraryInstallRequest
	ok := decodeJSONStrict(rec, req, &req2, tinyCap)

	if ok {
		t.Fatal("decodeJSONStrict = true, want false for an oversized body")
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413; body=%s", rec.Code, rec.Body.String())
	}
}
