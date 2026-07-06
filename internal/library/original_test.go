package library_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

// TestPreserveOriginalIsPreserveOnce proves the core invariant: the
// FIRST PreserveOriginal call snapshots the pristine content, and every
// subsequent call — no matter how many further edits happen in
// between — leaves the .orig sidecar holding that same pristine
// content. A rolling-backup implementation would fail this test
// because the second PreserveOriginal call would capture "edited-once"
// instead of leaving "pristine" alone.
func TestPreserveOriginalIsPreserveOnce(t *testing.T) {
	lib := openTempLibrary(t)
	root := lib.Root()
	walkPath := filepath.Join(root, "walks", "router.walk")
	if err := os.WriteFile(walkPath, []byte("pristine\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := lib.PreserveOriginal(library.KindWalks, "router.walk"); err != nil {
		t.Fatalf("first PreserveOriginal: %v", err)
	}
	origPath := walkPath + ".orig"
	assertFileContent(t, origPath, "pristine\n")

	// Edit the live file, then preserve again — the .orig must NOT change.
	if err := os.WriteFile(walkPath, []byte("edited-once\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := lib.PreserveOriginal(library.KindWalks, "router.walk"); err != nil {
		t.Fatalf("second PreserveOriginal: %v", err)
	}
	assertFileContent(t, origPath, "pristine\n")

	// A third edit + preserve — still must not change.
	if err := os.WriteFile(walkPath, []byte("edited-twice\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := lib.PreserveOriginal(library.KindWalks, "router.walk"); err != nil {
		t.Fatalf("third PreserveOriginal: %v", err)
	}
	assertFileContent(t, origPath, "pristine\n")

	// The live file itself keeps the latest edit — PreserveOriginal
	// never touches the working copy, only the sidecar.
	assertFileContent(t, walkPath, "edited-twice\n")
}

func TestPreserveOriginalNoopsWhenSourceMissing(t *testing.T) {
	lib := openTempLibrary(t)
	if err := lib.PreserveOriginal(library.KindWalks, "does-not-exist.walk"); err != nil {
		t.Fatalf("PreserveOriginal on missing source: %v", err)
	}
	if lib.HasOriginal(library.KindWalks, "does-not-exist.walk") {
		t.Errorf("HasOriginal = true, want false (nothing was preserved)")
	}
}

func TestHasOriginal(t *testing.T) {
	lib := openTempLibrary(t)
	root := lib.Root()
	walkPath := filepath.Join(root, "walks", "router.walk")
	if err := os.WriteFile(walkPath, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if lib.HasOriginal(library.KindWalks, "router.walk") {
		t.Errorf("HasOriginal = true before any PreserveOriginal, want false")
	}
	if err := lib.PreserveOriginal(library.KindWalks, "router.walk"); err != nil {
		t.Fatal(err)
	}
	if !lib.HasOriginal(library.KindWalks, "router.walk") {
		t.Errorf("HasOriginal = false after PreserveOriginal, want true")
	}
}

func TestRevertToOriginalRestoresAndRemovesSidecar(t *testing.T) {
	lib := openTempLibrary(t)
	root := lib.Root()
	walkPath := filepath.Join(root, "walks", "router.walk")
	if err := os.WriteFile(walkPath, []byte("pristine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := lib.PreserveOriginal(library.KindWalks, "router.walk"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(walkPath, []byte("edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := lib.RevertToOriginal(library.KindWalks, "router.walk"); err != nil {
		t.Fatalf("revert: %v", err)
	}

	assertFileContent(t, walkPath, "pristine\n")
	if _, err := os.Stat(walkPath + ".orig"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf(".orig should be removed after revert, stat err = %v", err)
	}
	if lib.HasOriginal(library.KindWalks, "router.walk") {
		t.Errorf("HasOriginal = true after revert, want false")
	}
}

func TestRevertToOriginalNoOriginalReturnsSentinel(t *testing.T) {
	lib := openTempLibrary(t)
	root := lib.Root()
	if err := os.WriteFile(filepath.Join(root, "walks", "router.walk"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := lib.RevertToOriginal(library.KindWalks, "router.walk")
	if !errors.Is(err, library.ErrNoOriginal) {
		t.Errorf("want ErrNoOriginal, got %v", err)
	}
}

func TestPreserveOriginalRejectsTraversal(t *testing.T) {
	lib := openTempLibrary(t)
	if err := lib.PreserveOriginal(library.KindWalks, "../escape.walk"); !errors.Is(err, library.ErrInvalidName) {
		t.Errorf("want ErrInvalidName, got %v", err)
	}
}

func TestRevertToOriginalRejectsTraversal(t *testing.T) {
	lib := openTempLibrary(t)
	if err := lib.RevertToOriginal(library.KindWalks, "../escape.walk"); !errors.Is(err, library.ErrInvalidName) {
		t.Errorf("want ErrInvalidName, got %v", err)
	}
}

func TestHasOriginalRejectsTraversal(t *testing.T) {
	lib := openTempLibrary(t)
	if lib.HasOriginal(library.KindWalks, "../escape.walk") {
		t.Errorf("HasOriginal = true for a traversal name, want false")
	}
}

func TestOriginalMethodsRejectNetworksKind(t *testing.T) {
	lib := openTempLibrary(t)
	if err := lib.PreserveOriginal(library.KindNetworks, "foo"); !errors.Is(err, library.ErrUnsupportedKind) {
		t.Errorf("PreserveOriginal: want ErrUnsupportedKind, got %v", err)
	}
	if lib.HasOriginal(library.KindNetworks, "foo") {
		t.Errorf("HasOriginal: want false for unsupported kind")
	}
	if err := lib.RevertToOriginal(library.KindNetworks, "foo"); !errors.Is(err, library.ErrUnsupportedKind) {
		t.Errorf("RevertToOriginal: want ErrUnsupportedKind, got %v", err)
	}
}

func TestListFilesExcludesOriginalAndBackupSidecars(t *testing.T) {
	lib := openTempLibrary(t)
	root := lib.Root()
	walkPath := filepath.Join(root, "walks", "router.walk")
	if err := os.WriteFile(walkPath, []byte("pristine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(walkPath+".orig", []byte("pristine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(walkPath+".bak", []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := lib.ListFiles(library.KindWalks)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "router.walk" {
		t.Errorf("ListFiles = %v, want exactly [router.walk] (.orig/.bak excluded)", entries)
	}
}

func TestListFilesReportsEdited(t *testing.T) {
	lib := openTempLibrary(t)
	root := lib.Root()
	walkPath := filepath.Join(root, "walks", "router.walk")
	if err := os.WriteFile(walkPath, []byte("pristine\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := lib.ListFiles(library.KindWalks)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %v, want 1", entries)
	}
	if entries[0].Edited {
		t.Errorf("Edited = true before any PreserveOriginal, want false")
	}

	if preserveErr := lib.PreserveOriginal(library.KindWalks, "router.walk"); preserveErr != nil {
		t.Fatal(preserveErr)
	}
	entries, err = lib.ListFiles(library.KindWalks)
	if err != nil {
		t.Fatal(err)
	}
	if !entries[0].Edited {
		t.Errorf("Edited = false after PreserveOriginal, want true")
	}
}

func TestFileEntryByNameReflectsEdited(t *testing.T) {
	lib := openTempLibrary(t)
	root := lib.Root()
	walkPath := filepath.Join(root, "walks", "router.walk")
	if err := os.WriteFile(walkPath, []byte("pristine\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	entry, err := lib.FileEntryByName(library.KindWalks, "router.walk")
	if err != nil {
		t.Fatalf("FileEntryByName: %v", err)
	}
	if entry.Edited {
		t.Errorf("Edited = true before PreserveOriginal, want false")
	}

	if preserveErr := lib.PreserveOriginal(library.KindWalks, "router.walk"); preserveErr != nil {
		t.Fatal(preserveErr)
	}
	entry, err = lib.FileEntryByName(library.KindWalks, "router.walk")
	if err != nil {
		t.Fatalf("FileEntryByName: %v", err)
	}
	if !entry.Edited {
		t.Errorf("Edited = false after PreserveOriginal, want true")
	}
}

func TestFileEntryByNameMissingReturnsNotFound(t *testing.T) {
	lib := openTempLibrary(t)
	_, err := lib.FileEntryByName(library.KindWalks, "does-not-exist.walk")
	if !errors.Is(err, library.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

// assertFileContent fails the test if path's content doesn't equal want.
func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != want {
		t.Errorf("%s content = %q, want %q", path, got, want)
	}
}
