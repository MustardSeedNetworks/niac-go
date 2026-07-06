package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

// revertRequest builds a POST /api/v1/library/walks/revert request
// with the given walk name as the JSON body.
func revertRequest(t *testing.T, name string) *http.Request {
	t.Helper()
	body, err := json.Marshal(map[string]string{"name": name})
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewRequest(http.MethodPost, "/api/v1/library/walks/revert", bytes.NewReader(body))
}

func TestHandleLibraryWalkRevertSuccess(t *testing.T) {
	server, root := newLibraryTestServer(t)
	walkPath := filepath.Join(root, "walks", "router.walk")
	if err := os.WriteFile(walkPath, []byte("pristine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := server.library.PreserveOriginal(library.KindWalks, "router.walk"); err != nil {
		t.Fatalf("preserve: %v", err)
	}
	if err := os.WriteFile(walkPath, []byte("edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	server.handleLibraryWalkRevert(rec, revertRequest(t, "router.walk"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var got library.FileEntry
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Edited {
		t.Errorf("edited = true in response after revert, want false")
	}
	if got.Name != "router.walk" {
		t.Errorf("name = %q, want router.walk", got.Name)
	}

	content, err := os.ReadFile(walkPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "pristine\n" {
		t.Errorf("content after revert = %q, want pristine", content)
	}
	if _, statErr := os.Stat(walkPath + ".orig"); !os.IsNotExist(statErr) {
		t.Errorf(".orig should be removed after revert, stat err = %v", statErr)
	}
}

func TestHandleLibraryWalkRevertNoOriginal(t *testing.T) {
	server, root := newLibraryTestServer(t)
	if err := os.WriteFile(filepath.Join(root, "walks", "router.walk"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	server.handleLibraryWalkRevert(rec, revertRequest(t, "router.walk"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleLibraryWalkRevertInvalidName(t *testing.T) {
	server, _ := newLibraryTestServer(t)

	rec := httptest.NewRecorder()
	server.handleLibraryWalkRevert(rec, revertRequest(t, "../../etc/passwd"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleLibraryWalkRevertEmptyName(t *testing.T) {
	server, _ := newLibraryTestServer(t)

	rec := httptest.NewRecorder()
	server.handleLibraryWalkRevert(rec, revertRequest(t, ""))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleLibraryWalkRevertUnavailable(t *testing.T) {
	// Server with library=nil simulates the daemon failing to open the
	// library at startup — must 503, not NPE.
	server := &Server{logger: slog.Default()}

	rec := httptest.NewRecorder()
	server.handleLibraryWalkRevert(rec, revertRequest(t, "router.walk"))

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
}

func TestHandleLibraryWalkRevertRejectsNonPost(t *testing.T) {
	server, _ := newLibraryTestServer(t)

	// Method gating moved from the handler to the route registry (ADR-0002);
	// exercise the methodGate wrapper register() composes for the POST-only
	// /api/v1/library/walks/revert route.
	gated := server.methodGate([]string{http.MethodPost}, server.handleLibraryWalkRevert)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/walks/revert", nil)
	rec := httptest.NewRecorder()
	gated(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
	if got := rec.Header().Get("Allow"); got != http.MethodPost {
		t.Errorf("Allow header = %q, want POST", got)
	}
}

// TestHandleLibraryWalksReportsEditedAfterPreserveAndRevert exercises the
// walks list's `edited` flag end to end through the list handler: it
// flips true right after PreserveOriginal and back to false right after
// RevertToOriginal — the field the UI badge/button gate on.
func TestHandleLibraryWalksReportsEditedAfterPreserveAndRevert(t *testing.T) {
	server, root := newLibraryTestServer(t)
	walkPath := filepath.Join(root, "walks", "router.walk")
	if err := os.WriteFile(walkPath, []byte("pristine\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	listOnce := func() library.FileEntry {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/library/walks", nil)
		rec := httptest.NewRecorder()
		server.handleLibraryWalks(rec, req)
		var got []library.FileEntry
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("entry count = %d, want 1 (%v)", len(got), got)
		}
		return got[0]
	}

	if entry := listOnce(); entry.Edited {
		t.Errorf("edited = true before any edit, want false")
	}

	if err := server.library.PreserveOriginal(library.KindWalks, "router.walk"); err != nil {
		t.Fatalf("preserve: %v", err)
	}
	if entry := listOnce(); !entry.Edited {
		t.Errorf("edited = false after PreserveOriginal, want true")
	}

	if err := server.library.RevertToOriginal(library.KindWalks, "router.walk"); err != nil {
		t.Fatalf("revert: %v", err)
	}
	if entry := listOnce(); entry.Edited {
		t.Errorf("edited = true after RevertToOriginal, want false")
	}
}
