package daemon

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPersistInlineSessionConfig_RejectsSessionIDsThatEscapeTheDirectory is why
// the session id is validated here rather than trusted from the caller.
//
// The id becomes part of the inline config's filename. The HTTP handler checks
// it, but the crash-recovery path calls StartSimulation with a request read back
// from active-simulation.json and never reaches that validator, so the guarantee
// has to live at the sink. Each id below either escapes the configs directory or
// puts a character in the filename that has no business there; none may reach
// the filesystem.
func TestPersistInlineSessionConfig_RejectsSessionIDsThatEscapeTheDirectory(t *testing.T) {
	ids := []string{
		"../../etc/evil",
		"..",
		"a/../../b",
		"/absolute",
		"has space",
		"UPPER",
		"-leading-hyphen",
		"trailing-hyphen-",
		"",
		strings.Repeat("a", 41),
	}

	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("NIAC_CONFIGS_DIR", dir)

			path, err := persistInlineSessionConfig("devices: []\n", id)
			if err == nil {
				t.Fatalf("persistInlineSessionConfig(%q) wrote %q, want a refusal", id, path)
			}
			if !errors.Is(err, errInvalidInlineSessionID) {
				t.Errorf("error = %v, want errInvalidInlineSessionID", err)
			}
			if path != "" {
				t.Errorf("path = %q, want empty alongside the error", path)
			}

			assertNothingWritten(t, dir)
		})
	}
}

// TestPersistInlineSessionConfig_WritesValidSessions is the positive half: the
// ids the API accepts still produce a per-session file, and the unnamed session
// keeps the fixed filename.
func TestPersistInlineSessionConfig_WritesValidSessions(t *testing.T) {
	tests := []struct {
		sessionID string
		wantName  string
	}{
		{defaultSessionID, inlineConfigName},
		{"lab-01", "_running.lab-01.inline.yaml"},
		{"a", "_running.a.inline.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.sessionID, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("NIAC_CONFIGS_DIR", dir)

			const content = "devices: []\n"

			path, err := persistInlineSessionConfig(content, tt.sessionID)
			if err != nil {
				t.Fatalf("persistInlineSessionConfig(%q): %v", tt.sessionID, err)
			}

			if got := filepath.Base(path); got != tt.wantName {
				t.Errorf("filename = %q, want %q", got, tt.wantName)
			}
			if !filepath.IsAbs(path) {
				t.Errorf("path = %q, want an absolute path", path)
			}

			written, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("read back %s: %v", path, readErr)
			}
			if string(written) != content {
				t.Errorf("file contains %q, want %q", written, content)
			}
		})
	}
}

// assertNothingWritten fails if the configs directory gained any file, which is
// what distinguishes a refusal from a write that merely reported an error.
func assertNothingWritten(t *testing.T, dir string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read configs dir: %v", err)
	}

	if len(entries) != 0 {
		names := make([]string, len(entries))
		for i, entry := range entries {
			names[i] = entry.Name()
		}

		t.Errorf("configs dir contains %v, want nothing written", names)
	}
}
