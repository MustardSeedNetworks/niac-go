package library_test

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

const updatedDraftYAML = `devices:
  - name: r2
    type: router
`

func TestDraftLifecycle(t *testing.T) {
	lib := openTempLibrary(t)

	created, err := lib.CreateDraft("branch-office", sampleYAML)
	if err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}
	if created.Name != "branch-office" || created.Content != sampleYAML {
		t.Fatalf("CreateDraft() = %+v", created)
	}
	if len(created.Revision) != 64 {
		t.Fatalf("revision length = %d, want 64", len(created.Revision))
	}

	read, err := lib.ReadDraft("branch-office")
	if err != nil {
		t.Fatalf("ReadDraft() error = %v", err)
	}
	if read.Revision != created.Revision || read.Content != created.Content {
		t.Fatalf("ReadDraft() = %+v, want revision/content from create", read)
	}

	entries, err := lib.ListDrafts()
	if err != nil {
		t.Fatalf("ListDrafts() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name != created.Name || entries[0].Revision != created.Revision {
		t.Fatalf("ListDrafts() = %+v", entries)
	}

	replaced, err := lib.ReplaceDraft("branch-office", created.Revision, updatedDraftYAML)
	if err != nil {
		t.Fatalf("ReplaceDraft() error = %v", err)
	}
	if replaced.Revision == created.Revision || replaced.Content != updatedDraftYAML {
		t.Fatalf("ReplaceDraft() = %+v", replaced)
	}

	if deleteErr := lib.DeleteDraft("branch-office", replaced.Revision); deleteErr != nil {
		t.Fatalf("DeleteDraft() error = %v", deleteErr)
	}
	if _, readErr := lib.ReadDraft("branch-office"); !errors.Is(readErr, library.ErrNotFound) {
		t.Fatalf("ReadDraft() after delete error = %v, want ErrNotFound", readErr)
	}
}

func TestCreateDraftRejectsDuplicateAndInvalidInput(t *testing.T) {
	lib := openTempLibrary(t)
	if _, err := lib.CreateDraft("valid", sampleYAML); err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}
	if _, err := lib.CreateDraft("valid", updatedDraftYAML); !errors.Is(err, library.ErrAlreadyExists) {
		t.Fatalf("duplicate CreateDraft() error = %v, want ErrAlreadyExists", err)
	}
	if _, err := lib.CreateDraft("../escape", sampleYAML); !errors.Is(err, library.ErrInvalidName) {
		t.Fatalf("traversal CreateDraft() error = %v, want ErrInvalidName", err)
	}
	if _, err := lib.CreateDraft("empty", ""); !errors.Is(err, library.ErrEmptyContent) {
		t.Fatalf("empty CreateDraft() error = %v, want ErrEmptyContent", err)
	}
}

func TestReplaceAndDeleteDraftRejectStaleRevision(t *testing.T) {
	lib := openTempLibrary(t)
	created, err := lib.CreateDraft("protected", sampleYAML)
	if err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}

	if _, replaceErr := lib.ReplaceDraft("protected", "stale", updatedDraftYAML); !errors.Is(
		replaceErr,
		library.ErrRevisionMismatch,
	) {
		t.Fatalf("ReplaceDraft() error = %v, want ErrRevisionMismatch", replaceErr)
	}
	if deleteErr := lib.DeleteDraft("protected", "stale"); !errors.Is(
		deleteErr,
		library.ErrRevisionMismatch,
	) {
		t.Fatalf("DeleteDraft() error = %v, want ErrRevisionMismatch", deleteErr)
	}

	read, err := lib.ReadDraft("protected")
	if err != nil {
		t.Fatalf("ReadDraft() error = %v", err)
	}
	if read.Revision != created.Revision || read.Content != sampleYAML {
		t.Fatalf("stale operations changed draft: %+v", read)
	}
}

func TestConcurrentDraftReplacementAllowsOneWriter(t *testing.T) {
	lib := openTempLibrary(t)
	created, err := lib.CreateDraft("shared", sampleYAML)
	if err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}

	contents := []string{updatedDraftYAML, `devices:
  - name: r3
    type: router
`}
	second, err := library.Open(lib.Root())
	if err != nil {
		t.Fatalf("open second library instance: %v", err)
	}
	libraries := []*library.Library{lib, second}
	errs := make(chan error, len(contents))
	var wg sync.WaitGroup
	for index, content := range contents {
		wg.Go(func() {
			_, replaceErr := libraries[index].ReplaceDraft("shared", created.Revision, content)
			errs <- replaceErr
		})
	}
	wg.Wait()
	close(errs)

	successes := 0
	mismatches := 0
	for replaceErr := range errs {
		switch {
		case replaceErr == nil:
			successes++
		case errors.Is(replaceErr, library.ErrRevisionMismatch):
			mismatches++
		default:
			t.Fatalf("ReplaceDraft() unexpected error = %v", replaceErr)
		}
	}
	if successes != 1 || mismatches != 1 {
		t.Fatalf("concurrent replacements: successes=%d mismatches=%d", successes, mismatches)
	}
}

func TestConcurrentDraftCreateAcrossLibraryInstancesDoesNotOverwrite(t *testing.T) {
	first := openTempLibrary(t)
	second, err := library.Open(first.Root())
	if err != nil {
		t.Fatalf("open second library instance: %v", err)
	}

	libraries := []*library.Library{first, second}
	contents := []string{sampleYAML, updatedDraftYAML}
	errs := make(chan error, len(libraries))
	var wg sync.WaitGroup
	for index, lib := range libraries {
		wg.Go(func() {
			_, createErr := lib.CreateDraft("shared-create", contents[index])
			errs <- createErr
		})
	}
	wg.Wait()
	close(errs)

	successes := 0
	conflicts := 0
	for createErr := range errs {
		switch {
		case createErr == nil:
			successes++
		case errors.Is(createErr, library.ErrAlreadyExists):
			conflicts++
		default:
			t.Fatalf("CreateDraft() unexpected error = %v", createErr)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent creates: successes=%d conflicts=%d", successes, conflicts)
	}
}

func TestConcurrentDraftReadAndReplaceAcrossLibraryInstancesIsCoherent(t *testing.T) {
	reader := openTempLibrary(t)
	created, err := reader.CreateDraft("changing", sampleYAML)
	if err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}
	writer, err := library.Open(reader.Root())
	if err != nil {
		t.Fatalf("open writer library instance: %v", err)
	}

	errs := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Go(func() {
		revision := created.Revision
		for index := range 100 {
			content := sampleYAML
			if index%2 == 0 {
				content = updatedDraftYAML
			}
			replaced, replaceErr := writer.ReplaceDraft("changing", revision, content)
			if replaceErr != nil {
				errs <- replaceErr
				return
			}
			revision = replaced.Revision
		}
	})
	wg.Go(func() {
		for range 200 {
			draft, readErr := reader.ReadDraft("changing")
			if readErr != nil {
				errs <- readErr
				return
			}
			wantRevision := fmt.Sprintf("%x", sha256.Sum256([]byte(draft.Content)))
			if draft.SizeBytes != int64(len(draft.Content)) || draft.Revision != wantRevision {
				errs <- fmt.Errorf("incoherent draft read: %+v", draft)
				return
			}
		}
	})
	wg.Wait()
	close(errs)
	for operationErr := range errs {
		t.Fatalf("concurrent draft operation: %v", operationErr)
	}
}

func TestOpenRejectsDraftsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ordinary Windows users may not have symlink permission")
	}
	root := t.TempDir()
	if err := os.Symlink(t.TempDir(), filepath.Join(root, "drafts")); err != nil {
		t.Fatalf("create drafts symlink: %v", err)
	}
	if _, err := library.Open(root); err == nil || !strings.Contains(err.Error(), "real directory") {
		t.Fatalf("library.Open() error = %v, want drafts symlink rejection", err)
	}
}

func TestOpenRejectsDraftsFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "drafts"), []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("create drafts file: %v", err)
	}
	if _, err := library.Open(root); err == nil || !strings.Contains(err.Error(), "real directory") {
		t.Fatalf("library.Open() error = %v, want drafts file rejection", err)
	}
}

func TestOpenTightensExistingDraftsDirectoryPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose Unix directory permission bits")
	}
	root := t.TempDir()
	draftsDir := filepath.Join(root, "drafts")
	if err := os.Mkdir(draftsDir, 0o755); err != nil {
		t.Fatalf("create drafts directory: %v", err)
	}
	if _, err := library.Open(root); err != nil {
		t.Fatalf("library.Open() error = %v", err)
	}
	info, err := os.Stat(draftsDir)
	if err != nil {
		t.Fatalf("stat drafts directory: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Fatalf("drafts directory mode = %o, want 700", got)
	}
}

func TestDraftPersistenceUsesPrivateFilesAndNoTemporaryResidue(t *testing.T) {
	lib := openTempLibrary(t)
	created, err := lib.CreateDraft("private", sampleYAML)
	if err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}
	if _, replaceErr := lib.ReplaceDraft("private", created.Revision, updatedDraftYAML); replaceErr != nil {
		t.Fatalf("ReplaceDraft() error = %v", replaceErr)
	}

	draftsDir := filepath.Join(lib.Root(), "drafts")
	entries, err := os.ReadDir(draftsDir)
	if err != nil {
		t.Fatalf("ReadDir(%s) error = %v", draftsDir, err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".draft-") {
			t.Fatalf("temporary draft residue = %s", entry.Name())
		}
	}
	info, err := os.Stat(filepath.Join(draftsDir, "private.yaml"))
	if err != nil {
		t.Fatalf("draft Info() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("draft mode = %o, want 600", got)
	}
}
