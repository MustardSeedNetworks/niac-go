package catalogsync_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/catalogsync"
)

const testCommit = "0123456789abcdef0123456789abcdef01234567"

type sourceManifest struct {
	Repository string `json:"repository"`
	Commit     string `json:"commit"`
	Files      []struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	} `json:"files"`
}

func TestRunSyncWritesDeterministicManifestAndDetectsDrift(t *testing.T) {
	catalog := writeCatalogFixture(t, validScenario, validWalk)
	first := filepath.Join(t.TempDir(), "examples")
	second := filepath.Join(t.TempDir(), "examples")
	options := catalogsync.Options{
		Mode:        catalogsync.ModeSync,
		CatalogDir:  catalog,
		ExamplesDir: first,
		Repository:  "git@example.test:catalog.git",
		Commit:      testCommit,
	}

	if runErr := catalogsync.Run(options); runErr != nil {
		t.Fatalf("Run(sync) error = %v", runErr)
	}
	options.ExamplesDir = second
	if runErr := catalogsync.Run(options); runErr != nil {
		t.Fatalf("second Run(sync) error = %v", runErr)
	}

	firstManifest, firstReadErr := os.ReadFile(filepath.Join(first, catalogsync.ManifestName))
	if firstReadErr != nil {
		t.Fatal(firstReadErr)
	}
	secondManifest, secondReadErr := os.ReadFile(filepath.Join(second, catalogsync.ManifestName))
	if secondReadErr != nil {
		t.Fatal(secondReadErr)
	}
	if string(firstManifest) != string(secondManifest) {
		t.Fatalf("manifest is not deterministic:\n%s\n%s", firstManifest, secondManifest)
	}
	var got sourceManifest
	if decodeErr := json.Unmarshal(firstManifest, &got); decodeErr != nil {
		t.Fatalf("decode manifest: %v", decodeErr)
	}
	if got.Commit != testCommit || got.Repository != options.Repository || len(got.Files) == 0 {
		t.Fatalf("manifest = %#v", got)
	}

	options.Mode = catalogsync.ModeCheck
	options.ExamplesDir = first
	if runErr := catalogsync.Run(options); runErr != nil {
		t.Fatalf("Run(check) error = %v", runErr)
	}
	if writeErr := os.WriteFile(filepath.Join(first, "lab.yaml"), []byte("changed"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	if driftErr := catalogsync.Run(options); driftErr == nil || !strings.Contains(driftErr.Error(), "differs") {
		t.Fatalf("Run(check drift) error = %v", driftErr)
	}
}

func TestRunRejectsInvalidScenarioBeforeSync(t *testing.T) {
	catalog := writeCatalogFixture(t, "devices: [", validWalk)
	examples := filepath.Join(t.TempDir(), "examples")

	runErr := catalogsync.Run(catalogsync.Options{
		Mode:        catalogsync.ModeSync,
		CatalogDir:  catalog,
		ExamplesDir: examples,
		Repository:  "git@example.test:catalog.git",
		Commit:      testCommit,
	})
	if runErr == nil || !strings.Contains(runErr.Error(), "load scenario") {
		t.Fatalf("Run() error = %v", runErr)
	}
	if _, examplesErr := os.Stat(examples); !errors.Is(examplesErr, os.ErrNotExist) {
		t.Fatalf("examples created despite invalid scenario: %v", examplesErr)
	}
}

func TestRunRejectsInvalidWalkBeforeSync(t *testing.T) {
	catalog := writeCatalogFixture(t, validScenario, "invalid walk line")
	examples := filepath.Join(t.TempDir(), "examples")

	runErr := catalogsync.Run(catalogsync.Options{
		Mode:        catalogsync.ModeSync,
		CatalogDir:  catalog,
		ExamplesDir: examples,
		Repository:  "git@example.test:catalog.git",
		Commit:      testCommit,
	})
	if runErr == nil || !strings.Contains(runErr.Error(), "walk validation failed") {
		t.Fatalf("Run() error = %v", runErr)
	}
	if _, examplesErr := os.Stat(examples); !errors.Is(examplesErr, os.ErrNotExist) {
		t.Fatalf("examples created despite invalid walk: %v", examplesErr)
	}
}

func TestRunRejectsMutableRevisionIdentifier(t *testing.T) {
	runErr := catalogsync.Run(catalogsync.Options{
		Mode:        catalogsync.ModeSync,
		CatalogDir:  t.TempDir(),
		ExamplesDir: filepath.Join(t.TempDir(), "examples"),
		Repository:  "git@example.test:catalog.git",
		Commit:      "main",
	})
	if runErr == nil || !strings.Contains(runErr.Error(), "full 40- or 64-character") {
		t.Fatalf("Run() error = %v", runErr)
	}
}

func writeCatalogFixture(t *testing.T, scenario, walk string) string {
	t.Helper()
	root := t.TempDir()
	directories := []string{
		"scenarios/go-yaml",
		"walks/raw",
		"walks/sanitized",
		"captures/shared",
		"captures/go-extra",
		"tools/walk-scripts/go",
		"docs/imported/go-examples",
	}
	for _, directory := range directories {
		if mkdirErr := os.MkdirAll(filepath.Join(root, directory), 0o750); mkdirErr != nil {
			t.Fatal(mkdirErr)
		}
	}
	writeFixtureFile(t, filepath.Join(root, "scenarios/go-yaml/lab.yaml"), scenario)
	writeFixtureFile(t, filepath.Join(root, "walks/raw/device.walk"), walk)
	writeFixtureFile(t, filepath.Join(root, "walks/sanitized/device.walk"), walk)
	writeFixtureFile(t, filepath.Join(root, "captures/shared/capture.pcap"), "capture")
	writeFixtureFile(t, filepath.Join(root, "captures/go-extra/extra.pcap"), "extra")
	writeFixtureFile(t, filepath.Join(root, "tools/walk-scripts/go/run.sh"), "#!/bin/sh\n")
	writeFixtureFile(t, filepath.Join(root, "docs/imported/go-examples/README.md"), "demo\n")
	return root
}

func writeFixtureFile(t *testing.T, path, content string) {
	t.Helper()
	if writeErr := os.WriteFile(path, []byte(content), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
}

const validScenario = `devices:
  - name: test-device
    type: switch
    mac: "00:11:22:33:44:55"
    ips: ["192.0.2.10"]
`

const validWalk = `.1.3.6.1.2.1.1.5.0 = STRING: "test-device"
`
