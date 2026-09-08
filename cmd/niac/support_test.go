package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// supportLibrary builds a library holding one scenario, so a round trip has
// something to prove it moved.
func supportLibrary(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	networks := filepath.Join(root, "networks")
	if err := os.MkdirAll(networks, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", networks, err)
	}
	if err := os.WriteFile(filepath.Join(networks, "office.yaml"),
		[]byte("devices:\n  - name: sw1\n"), 0o600); err != nil {
		t.Fatalf("write scenario: %v", err)
	}
	return root
}

func TestBackupThenRestoreThroughTheCLI(t *testing.T) {
	source := supportLibrary(t)
	archive := filepath.Join(t.TempDir(), "library.tar.gz")

	if err := runBackup(archive, &supportOptions{libraryRoot: source}); err != nil {
		t.Fatalf("runBackup: %v", err)
	}

	target := filepath.Join(t.TempDir(), "restored")
	if err := runRestore(archive, &supportOptions{libraryRoot: target}); err != nil {
		t.Fatalf("runRestore: %v", err)
	}

	restored, err := os.ReadFile(filepath.Join(target, "networks", "office.yaml"))
	if err != nil {
		t.Fatalf("read restored scenario: %v", err)
	}
	if !strings.Contains(string(restored), "sw1") {
		t.Errorf("restored scenario is %q", restored)
	}
}

// Restore replaces the whole library, so a populated target is refused until
// the operator says they meant it.
func TestRestoreRefusesAPopulatedLibraryWithoutForce(t *testing.T) {
	source := supportLibrary(t)
	archive := filepath.Join(t.TempDir(), "library.tar.gz")
	if err := runBackup(archive, &supportOptions{libraryRoot: source}); err != nil {
		t.Fatalf("runBackup: %v", err)
	}

	target := supportLibrary(t)
	err := runRestore(archive, &supportOptions{libraryRoot: target})
	if err == nil {
		t.Fatal("restore replaced a populated library without --force")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("error does not name the way forward: %v", err)
	}

	if forceErr := runRestore(archive, &supportOptions{libraryRoot: target, force: true}); forceErr != nil {
		t.Fatalf("runRestore --force: %v", forceErr)
	}
}

// A backup must not overwrite an archive already on disk -- the common
// mistake is re-running the command and losing the earlier backup.
func TestBackupRefusesAnExistingArchive(t *testing.T) {
	source := supportLibrary(t)
	archive := filepath.Join(t.TempDir(), "library.tar.gz")
	if err := os.WriteFile(archive, []byte("existing"), 0o600); err != nil {
		t.Fatalf("seed archive: %v", err)
	}

	if err := runBackup(archive, &supportOptions{libraryRoot: source}); err == nil {
		t.Fatal("backup overwrote an existing archive")
	}
	content, err := os.ReadFile(archive)
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	if string(content) != "existing" {
		t.Error("backup overwrote the existing archive's content")
	}
}

func TestLibraryScenariosListsOnlyScenarioFiles(t *testing.T) {
	root := supportLibrary(t)
	networks := filepath.Join(root, "networks")
	for _, name := range []string{"notes.txt", "branch.yml", "sub"} {
		path := filepath.Join(networks, name)
		if name == "sub" {
			if err := os.Mkdir(path, 0o755); err != nil {
				t.Fatalf("mkdir %s: %v", path, err)
			}
			continue
		}
		if err := os.WriteFile(path, []byte("devices: []\n"), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	scenarios, err := libraryScenarios(root)
	if err != nil {
		t.Fatalf("libraryScenarios: %v", err)
	}
	want := []string{
		filepath.Join(networks, "branch.yml"),
		filepath.Join(networks, "office.yaml"),
	}
	if len(scenarios) != len(want) {
		t.Fatalf("listed %v, want %v", scenarios, want)
	}
	for i, path := range want {
		if scenarios[i] != path {
			t.Errorf("scenario %d is %s, want %s", i, scenarios[i], path)
		}
	}
}

// A library that has never been populated still produces a bundle: the
// manifest and the interface inventory are the part support asks for first.
func TestLibraryScenariosOnAMissingLibrary(t *testing.T) {
	scenarios, err := libraryScenarios(filepath.Join(t.TempDir(), "absent"))
	if err != nil {
		t.Fatalf("libraryScenarios: %v", err)
	}
	if len(scenarios) != 0 {
		t.Errorf("listed %v for a missing library", scenarios)
	}
}
