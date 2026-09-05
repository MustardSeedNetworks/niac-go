package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveConfigSourceFileWins(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeTestFile(t, path, []byte(minimalConfig))

	source, err := resolveConfigSource(path)
	if err != nil {
		t.Fatalf("resolveConfigSource(file): %v", err)
	}
	if source.label != path || source.path != path {
		t.Fatalf("label = %q, path = %q, want both %q", source.label, source.path, path)
	}
	if string(source.data) != minimalConfig {
		t.Fatalf("data = %q, want the file's contents", source.data)
	}
}

// `run` resolved a bare scenario name against the built-in templates; `--once`
// inherits that, or the Java parity matrix's "run named demo scenario" row
// regresses.
func TestResolveConfigSourceTemplate(t *testing.T) {
	source, err := resolveConfigSource("basic-network")
	if err != nil {
		t.Fatalf("resolveConfigSource(template): %v", err)
	}
	if source.label != "template:basic-network" {
		t.Fatalf("label = %q, want template:basic-network", source.label)
	}
	if !strings.Contains(string(source.data), "devices:") {
		t.Fatalf("template document has no devices: %q", source.data)
	}
	// A template is not a file, so nothing may resolve relative paths against it.
	if source.path != "" {
		t.Fatalf("path = %q, want empty for a template", source.path)
	}
}

func TestResolveConfigSourceUnknownName(t *testing.T) {
	if _, err := resolveConfigSource("no-such-scenario-anywhere"); err == nil {
		t.Fatal("resolveConfigSource(unknown) = nil error, want a not-found error")
	}
}
