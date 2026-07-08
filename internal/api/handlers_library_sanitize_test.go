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

	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

const sanitizeFixtureWalk = `SNMPv2-MIB::sysName.0 = STRING: real-switch-01
SNMPv2-MIB::sysContact.0 = STRING: admin@realcorp.com
.1.3.6.1.2.1.4.20.1.1.192.168.1.1 = IpAddress: 192.168.1.1
`

// sanitizeRequest builds a POST /api/v1/library/walks/sanitize request with
// the given walk name as the JSON body.
func sanitizeRequest(t *testing.T, name string) *http.Request {
	t.Helper()
	body, err := json.Marshal(map[string]string{"name": name})
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewRequest(http.MethodPost, "/api/v1/library/walks/sanitize", bytes.NewReader(body))
}

// sanitizeBatchRequest builds a POST /api/v1/library/walks/sanitize-batch
// request with the given walk names as the JSON body.
func sanitizeBatchRequest(t *testing.T, names []string) *http.Request {
	t.Helper()
	body, err := json.Marshal(map[string][]string{"names": names})
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewRequest(http.MethodPost, "/api/v1/library/walks/sanitize-batch", bytes.NewReader(body))
}

func TestHandleLibraryWalkSanitizeSuccess(t *testing.T) {
	server, root := newLibraryTestServer(t)
	walkPath := filepath.Join(root, "walks", "router.walk")
	if err := os.WriteFile(walkPath, []byte(sanitizeFixtureWalk), 0o644); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	server.handleLibraryWalkSanitize(rec, sanitizeRequest(t, "router.walk"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var got library.FileEntry
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.Edited {
		t.Errorf("edited = false after sanitize, want true (original preserved)")
	}

	content, err := os.ReadFile(walkPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "real-switch-01") {
		t.Errorf("sanitized content still contains original hostname: %s", content)
	}
	if strings.Contains(string(content), "192.168.1.1") {
		t.Errorf("sanitized content still contains original IP: %s", content)
	}
	if !strings.Contains(string(content), "niac-core-") {
		t.Errorf("sanitized content missing branded hostname: %s", content)
	}

	// Original must be untouched and recoverable.
	orig, err := os.ReadFile(walkPath + ".orig")
	if err != nil {
		t.Fatalf("read .orig: %v", err)
	}
	if string(orig) != sanitizeFixtureWalk {
		t.Errorf(".orig content = %q, want pristine original", orig)
	}

	if revertErr := server.library.RevertToOriginal(library.KindWalks, "router.walk"); revertErr != nil {
		t.Fatalf("revert: %v", revertErr)
	}
	reverted, err := os.ReadFile(walkPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(reverted) != sanitizeFixtureWalk {
		t.Errorf("content after revert = %q, want pristine original", reverted)
	}
}

func TestHandleLibraryWalkSanitizeIdempotentPreserve(t *testing.T) {
	// A second sanitize call must not clobber the .orig sidecar with
	// already-sanitized content — PreserveOriginal is preserve-once.
	server, root := newLibraryTestServer(t)
	walkPath := filepath.Join(root, "walks", "router.walk")
	if err := os.WriteFile(walkPath, []byte(sanitizeFixtureWalk), 0o644); err != nil {
		t.Fatal(err)
	}

	rec1 := httptest.NewRecorder()
	server.handleLibraryWalkSanitize(rec1, sanitizeRequest(t, "router.walk"))
	if rec1.Code != http.StatusOK {
		t.Fatalf("first sanitize status = %d, want 200", rec1.Code)
	}

	rec2 := httptest.NewRecorder()
	server.handleLibraryWalkSanitize(rec2, sanitizeRequest(t, "router.walk"))
	if rec2.Code != http.StatusOK {
		t.Fatalf("second sanitize status = %d, want 200", rec2.Code)
	}

	orig, err := os.ReadFile(walkPath + ".orig")
	if err != nil {
		t.Fatalf("read .orig: %v", err)
	}
	if string(orig) != sanitizeFixtureWalk {
		t.Errorf(".orig content after re-sanitize = %q, want unchanged pristine original", orig)
	}
}

func TestHandleLibraryWalkSanitizeNotFound(t *testing.T) {
	server, _ := newLibraryTestServer(t)

	rec := httptest.NewRecorder()
	server.handleLibraryWalkSanitize(rec, sanitizeRequest(t, "missing.walk"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleLibraryWalkSanitizeInvalidName(t *testing.T) {
	server, _ := newLibraryTestServer(t)

	rec := httptest.NewRecorder()
	server.handleLibraryWalkSanitize(rec, sanitizeRequest(t, "../../etc/passwd"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleLibraryWalkSanitizeEmptyName(t *testing.T) {
	server, _ := newLibraryTestServer(t)

	rec := httptest.NewRecorder()
	server.handleLibraryWalkSanitize(rec, sanitizeRequest(t, ""))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleLibraryWalkSanitizeUnavailable(t *testing.T) {
	// Server with library=nil simulates the daemon failing to open the
	// library at startup — must 503, not NPE.
	server := &Server{logger: slog.Default()}

	rec := httptest.NewRecorder()
	server.handleLibraryWalkSanitize(rec, sanitizeRequest(t, "router.walk"))

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
}

func TestHandleLibraryWalkSanitizeRejectsNonPost(t *testing.T) {
	server, _ := newLibraryTestServer(t)

	gated := server.methodGate([]string{http.MethodPost}, server.handleLibraryWalkSanitize)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/walks/sanitize", nil)
	rec := httptest.NewRecorder()
	gated(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
	if got := rec.Header().Get("Allow"); got != http.MethodPost {
		t.Errorf("Allow header = %q, want POST", got)
	}
}

func TestHandleLibraryWalkSanitizeBatchSuccess(t *testing.T) {
	server, root := newLibraryTestServer(t)
	for _, name := range []string{"router1.walk", "router2.walk"} {
		if err := os.WriteFile(filepath.Join(root, "walks", name), []byte(sanitizeFixtureWalk), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	rec := httptest.NewRecorder()
	server.handleLibraryWalkSanitizeBatch(rec, sanitizeBatchRequest(t, []string{"router1.walk", "router2.walk"}))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var got libraryWalkSanitizeBatchResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Sanitized != 2 || got.Failed != 0 {
		t.Errorf("Sanitized=%d Failed=%d, want 2/0", got.Sanitized, got.Failed)
	}
	for _, result := range got.Results {
		if !result.Success {
			t.Errorf("result for %s: success = false, want true", result.Name)
		}
	}
}

func TestHandleLibraryWalkSanitizeBatchPartialFailure(t *testing.T) {
	server, root := newLibraryTestServer(t)
	walkPath := filepath.Join(root, "walks", "router1.walk")
	if err := os.WriteFile(walkPath, []byte(sanitizeFixtureWalk), 0o644); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	server.handleLibraryWalkSanitizeBatch(rec, sanitizeBatchRequest(t, []string{"router1.walk", "missing.walk"}))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var got libraryWalkSanitizeBatchResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Sanitized != 1 || got.Failed != 1 {
		t.Errorf("Sanitized=%d Failed=%d, want 1/1", got.Sanitized, got.Failed)
	}
}

func TestHandleLibraryWalkSanitizeBatchEmptyNames(t *testing.T) {
	server, _ := newLibraryTestServer(t)

	rec := httptest.NewRecorder()
	server.handleLibraryWalkSanitizeBatch(rec, sanitizeBatchRequest(t, []string{}))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleLibraryWalkSanitizeBatchUnavailable(t *testing.T) {
	server := &Server{logger: slog.Default()}

	rec := httptest.NewRecorder()
	server.handleLibraryWalkSanitizeBatch(rec, sanitizeBatchRequest(t, []string{"router.walk"}))

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
}
