package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

// newLibraryTestServer spins up a real library rooted in a fresh
// t.TempDir() and hooks it onto a minimal Server so the library
// handlers can be exercised directly with httptest. Returns the root
// so the test can plant fixture walks/pcaps before calling the
// handler.
func newLibraryTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("NIAC_LIBRARY_ROOT", root)

	lib, err := library.Open(root)
	if err != nil {
		t.Fatalf("open library: %v", err)
	}
	return &Server{
		logger:  slog.Default(),
		library: lib,
	}, root
}

func TestHandleLibraryWalksLists(t *testing.T) {
	server, root := newLibraryTestServer(t)

	// Drop two fixture walks; the inner cisco/ subdir mirrors what a
	// real install looks like (vendor-namespaced walk files).
	if err := os.MkdirAll(filepath.Join(root, "walks", "cisco"), 0o755); err != nil {
		t.Fatal(err)
	}
	ciscoWalk := filepath.Join(root, "walks", "cisco", "c3900.walk")
	if err := os.WriteFile(ciscoWalk, []byte("oid = STRING: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	routerWalk := filepath.Join(root, "walks", "router.walk")
	if err := os.WriteFile(routerWalk, []byte("oid = STRING: y\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/walks", nil)
	rec := httptest.NewRecorder()
	server.handleLibraryWalks(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var got []library.FileEntry
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("entry count = %d, want 2 (%v)", len(got), got)
	}
	// ListFiles sorts by name; cisco/c3900.walk comes before router.walk.
	if got[0].Name != "cisco/c3900.walk" {
		t.Errorf("entry[0].Name = %q, want cisco/c3900.walk", got[0].Name)
	}
	if got[1].Name != "router.walk" {
		t.Errorf("entry[1].Name = %q, want router.walk", got[1].Name)
	}
	if got[1].SizeBytes == 0 {
		t.Errorf("entry[1] should report nonzero size")
	}
}

func TestHandleLibraryPcapsEmpty(t *testing.T) {
	server, _ := newLibraryTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/pcaps", nil)
	rec := httptest.NewRecorder()
	server.handleLibraryPcaps(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var got []library.FileEntry
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("empty pcaps dir should return empty slice, got %v", got)
	}
}

func TestHandleLibraryWalksRejectsNonGet(t *testing.T) {
	server, _ := newLibraryTestServer(t)

	// Method gating moved from the handler to the route registry (ADR-0002);
	// exercise the methodGate wrapper register() composes for the GET-only
	// /api/v1/library/walks route.
	gated := server.methodGate([]string{http.MethodGet}, server.handleLibraryWalks)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/walks", nil)
	rec := httptest.NewRecorder()
	gated(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
	if got := rec.Header().Get("Allow"); got != "GET" {
		t.Errorf("Allow header = %q, want GET", got)
	}
}

func TestHandleLibraryWalksUnavailable(t *testing.T) {
	// Server with library=nil simulates the daemon failing to open the
	// library at startup. The handler should 503 rather than NPE.
	server := &Server{logger: slog.Default()}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/walks", nil)
	rec := httptest.NewRecorder()
	server.handleLibraryWalks(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
}

// TestValidateWalkFilePathFallsBackToLibrary verifies that #556's
// resolver tries the library walks dir as a fallback when the file
// isn't in the legacy allow-list dir. Without this, the new picker
// integrations would send library-relative names that the walk
// validator would reject.
func TestValidateWalkFilePathFallsBackToLibrary(t *testing.T) {
	server, root := newLibraryTestServer(t)

	// Plant a walk in the library/walks dir but NOT in the cfgPath
	// allow-list dir (server.configPath returns "" → falls back to
	// cwd; we don't drop the walk there).
	walkPath := filepath.Join(root, "walks", "lib-only.walk")
	if err := os.WriteFile(walkPath, []byte("oid = STRING: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := server.validateWalkFilePath("lib-only.walk")
	if err != nil {
		t.Fatalf("validateWalkFilePath: %v", err)
	}
	if got != walkPath {
		t.Errorf("resolved path = %q, want %q", got, walkPath)
	}
}

// TestValidateWalkFilePathPreservesConfigDirPriority verifies that a
// walk available in BOTH the config dir and the library resolves to
// the config-dir copy — the legacy bare-filename behaviour in YAML
// configs must keep working unchanged.
func TestValidateWalkFilePathPreservesConfigDirPriority(t *testing.T) {
	server, root := newLibraryTestServer(t)
	cfgDir := t.TempDir()
	// configPath() reads from server.cfg.ConfigPath; set it.
	server.cfg.ConfigPath = filepath.Join(cfgDir, "config.yaml")

	const filename = "shared.walk"
	if err := os.WriteFile(filepath.Join(cfgDir, filename), []byte("oid = STRING: cfg\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "walks", filename), []byte("oid = STRING: lib\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := server.validateWalkFilePath(filename)
	if err != nil {
		t.Fatalf("validateWalkFilePath: %v", err)
	}
	want := filepath.Join(cfgDir, filename)
	if got != want {
		t.Errorf("resolved path = %q, want config-dir copy %q", got, want)
	}
}

// TestValidatePcapFilePathFallsBackToLibrary mirrors the walk test:
// a pcap that lives only under <library>/pcaps/ must still resolve
// via the legacy validator surface, so the new ReplayControlPanel
// picker (#556 pcap half) doesn't need a parallel API.
func TestValidatePcapFilePathFallsBackToLibrary(t *testing.T) {
	server, root := newLibraryTestServer(t)

	pcapPath := filepath.Join(root, "pcaps", "lib-only.pcap")
	if err := os.WriteFile(pcapPath, []byte("fake-pcap-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := server.validatePcapFilePath("lib-only.pcap")
	if err != nil {
		t.Fatalf("validatePcapFilePath: %v", err)
	}
	if got != pcapPath {
		t.Errorf("resolved path = %q, want %q", got, pcapPath)
	}
}

// TestValidatePcapFilePathPreservesConfigDirPriority — same priority
// guarantee for pcaps as for walks. A pcap present in both places
// resolves to the config-dir copy.
func TestValidatePcapFilePathPreservesConfigDirPriority(t *testing.T) {
	server, root := newLibraryTestServer(t)
	cfgDir := t.TempDir()
	server.cfg.ConfigPath = filepath.Join(cfgDir, "config.yaml")

	const filename = "shared.pcap"
	if err := os.WriteFile(filepath.Join(cfgDir, filename), []byte("cfg-pcap"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pcaps", filename), []byte("lib-pcap"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := server.validatePcapFilePath(filename)
	if err != nil {
		t.Fatalf("validatePcapFilePath: %v", err)
	}
	want := filepath.Join(cfgDir, filename)
	if got != want {
		t.Errorf("resolved path = %q, want config-dir copy %q", got, want)
	}
}
