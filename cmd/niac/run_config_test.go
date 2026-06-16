package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigOrScenarioFileWins(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	writeTestFile(t, configPath, []byte(`devices:
  - name: file-router
    mac: "00:11:22:33:44:55"
    ips:
      - "192.0.2.10"
`))

	cfg, resolved, err := loadConfigOrScenario(configPath)
	if err != nil {
		t.Fatalf("loadConfigOrScenario(file): %v", err)
	}
	if resolved != configPath {
		t.Fatalf("resolved = %q, want %q", resolved, configPath)
	}
	if len(cfg.Devices) != 1 || cfg.Devices[0].Name != "file-router" {
		t.Fatalf("unexpected config devices: %+v", cfg.Devices)
	}
}

func TestLoadConfigOrScenarioTemplate(t *testing.T) {
	cfg, resolved, err := loadConfigOrScenario("basic-network")
	if err != nil {
		t.Fatalf("loadConfigOrScenario(template): %v", err)
	}
	if resolved != "template:basic-network" {
		t.Fatalf("resolved = %q, want template:basic-network", resolved)
	}
	if len(cfg.Devices) == 0 {
		t.Fatal("expected template devices")
	}
}

func TestLoadConfigOrScenarioLibraryNetwork(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NIAC_LIBRARY_ROOT", root)
	writeTestFile(t, filepath.Join(root, "networks", "demo.yaml"), []byte(`devices:
  - name: library-router
    mac: "00:11:22:33:44:66"
    ips:
      - "192.0.2.11"
`))

	cfg, resolved, err := loadConfigOrScenario("demo")
	if err != nil {
		t.Fatalf("loadConfigOrScenario(library): %v", err)
	}
	if resolved != "library:demo" {
		t.Fatalf("resolved = %q, want library:demo", resolved)
	}
	if len(cfg.Devices) != 1 || cfg.Devices[0].Name != "library-router" {
		t.Fatalf("unexpected config devices: %+v", cfg.Devices)
	}
}

func TestLoadConfigOrScenarioMissing(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NIAC_LIBRARY_ROOT", root)

	_, _, err := loadConfigOrScenario("does-not-exist")
	if err == nil {
		t.Fatal("expected missing scenario error")
	}
	if !strings.Contains(err.Error(), "config file or scenario not found") {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, statErr := os.Stat(filepath.Join(root, "networks")); statErr != nil {
		t.Fatalf("library layout should be created: %v", statErr)
	}
}
