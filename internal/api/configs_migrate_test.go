package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

const migrateSampleYAML = `devices:
  - name: core1
    mac: "00:11:22:33:44:55"
    ips: ["10.0.0.1"]
`

func openMigrateTestLibrary(t *testing.T) *library.Library {
	t.Helper()
	lib, err := library.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open library: %v", err)
	}
	return lib
}

func writeLegacyConfig(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	path := filepath.Join(dir, name+".yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write legacy config %s: %v", path, err)
	}
}

func TestMigrateLegacyUserConfigsMigratesNewNames(t *testing.T) {
	lib := openMigrateTestLibrary(t)
	legacyDir := t.TempDir()
	writeLegacyConfig(t, legacyDir, "legacy-net", migrateSampleYAML)

	migrated, err := migrateLegacyUserConfigsFromDirs(lib, []string{legacyDir})
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if migrated != 1 {
		t.Fatalf("migrated = %d, want 1", migrated)
	}

	doc, err := lib.ReadNetwork("legacy-net")
	if err != nil {
		t.Fatalf("read migrated network: %v", err)
	}
	if doc.Content != migrateSampleYAML {
		t.Errorf("migrated content = %q, want %q", doc.Content, migrateSampleYAML)
	}
}

func TestMigrateLegacyUserConfigsIsIdempotent(t *testing.T) {
	lib := openMigrateTestLibrary(t)
	legacyDir := t.TempDir()
	writeLegacyConfig(t, legacyDir, "legacy-net", migrateSampleYAML)

	first, err := migrateLegacyUserConfigsFromDirs(lib, []string{legacyDir})
	if err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if first != 1 {
		t.Fatalf("first migrated = %d, want 1", first)
	}

	second, err := migrateLegacyUserConfigsFromDirs(lib, []string{legacyDir})
	if err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	if second != 0 {
		t.Errorf("second migrated = %d, want 0 (idempotent)", second)
	}
}

func TestMigrateLegacyUserConfigsDoesNotTouchLegacyFiles(t *testing.T) {
	lib := openMigrateTestLibrary(t)
	legacyDir := t.TempDir()
	writeLegacyConfig(t, legacyDir, "legacy-net", migrateSampleYAML)
	legacyPath := filepath.Join(legacyDir, "legacy-net.yaml")

	before, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatalf("read legacy file before migrate: %v", err)
	}
	beforeInfo, err := os.Stat(legacyPath)
	if err != nil {
		t.Fatalf("stat legacy file before migrate: %v", err)
	}

	if _, migrateErr := migrateLegacyUserConfigsFromDirs(lib, []string{legacyDir}); migrateErr != nil {
		t.Fatalf("migrate: %v", migrateErr)
	}

	after, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatalf("legacy file missing after migrate: %v", err)
	}
	if string(before) != string(after) {
		t.Errorf("legacy file content changed: before %q, after %q", before, after)
	}
	afterInfo, err := os.Stat(legacyPath)
	if err != nil {
		t.Fatalf("stat legacy file after migrate: %v", err)
	}
	if beforeInfo.ModTime() != afterInfo.ModTime() {
		t.Errorf("legacy file mtime changed: before %v, after %v", beforeInfo.ModTime(), afterInfo.ModTime())
	}
}

func TestMigrateLegacyUserConfigsSkipsExistingLibraryNames(t *testing.T) {
	lib := openMigrateTestLibrary(t)
	if err := lib.WriteNetwork("dup-net", "devices:\n  - name: existing\n"); err != nil {
		t.Fatalf("seed library entry: %v", err)
	}

	legacyDir := t.TempDir()
	writeLegacyConfig(t, legacyDir, "dup-net", migrateSampleYAML)

	migrated, err := migrateLegacyUserConfigsFromDirs(lib, []string{legacyDir})
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if migrated != 0 {
		t.Errorf("migrated = %d, want 0 (name already exists in library)", migrated)
	}

	doc, err := lib.ReadNetwork("dup-net")
	if err != nil {
		t.Fatalf("read library entry: %v", err)
	}
	if doc.Content != "devices:\n  - name: existing\n" {
		t.Errorf("library entry was overwritten by legacy migration: %q", doc.Content)
	}
}

func TestMigrateLegacyUserConfigsSkipsInvalidContent(t *testing.T) {
	lib := openMigrateTestLibrary(t)
	legacyDir := t.TempDir()
	writeLegacyConfig(t, legacyDir, "not-a-config", "# just a comment, no devices section\n")

	migrated, err := migrateLegacyUserConfigsFromDirs(lib, []string{legacyDir})
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if migrated != 0 {
		t.Errorf("migrated = %d, want 0 (invalid content should be skipped, not migrated)", migrated)
	}
	if _, readErr := lib.ReadNetwork("not-a-config"); readErr == nil {
		t.Error("invalid legacy config should not have been written to the library")
	}
}

func TestMigrateLegacyUserConfigsSkipsMissingDirs(t *testing.T) {
	lib := openMigrateTestLibrary(t)
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	migrated, err := migrateLegacyUserConfigsFromDirs(lib, []string{missing})
	if err != nil {
		t.Fatalf("migrate over missing dir should not error: %v", err)
	}
	if migrated != 0 {
		t.Errorf("migrated = %d, want 0", migrated)
	}
}

func TestMigrateLegacyUserConfigsDedupesAcrossDirsByFirstMatch(t *testing.T) {
	lib := openMigrateTestLibrary(t)
	firstDir := t.TempDir()
	secondDir := t.TempDir()
	writeLegacyConfig(t, firstDir, "shared-name", migrateSampleYAML)
	writeLegacyConfig(t, secondDir, "shared-name", "devices:\n  - name: from-second-dir\n")

	migrated, err := migrateLegacyUserConfigsFromDirs(lib, []string{firstDir, secondDir})
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if migrated != 1 {
		t.Fatalf("migrated = %d, want 1", migrated)
	}

	doc, err := lib.ReadNetwork("shared-name")
	if err != nil {
		t.Fatalf("read migrated network: %v", err)
	}
	if doc.Content != migrateSampleYAML {
		t.Errorf("expected first-dir content to win, got %q", doc.Content)
	}
}
