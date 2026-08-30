package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// writeFile creates a file with the given contents and returns its path.
func writeFile(t *testing.T, path string) string {
	t.Helper()

	if err := os.WriteFile(path, []byte("devices: []\n"), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}

	return path
}

// TestResolveManagedConfigPath_AcceptsFileInsideRoot is the positive case the
// rejections below have to leave working.
func TestResolveManagedConfigPath_AcceptsFileInsideRoot(t *testing.T) {
	root := t.TempDir()
	path := writeFile(t, filepath.Join(root, "sim.yaml"))

	got, err := config.ResolveManagedConfigPath(path, []string{root})
	if err != nil {
		t.Fatalf("ResolveManagedConfigPath: %v", err)
	}

	// The returned path is symlink-resolved, so compare against the resolved
	// form rather than the literal one (macOS /var is a symlink to /private/var).
	want, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	if got != want {
		t.Errorf("resolved path = %q, want %q", got, want)
	}
}

// TestResolveManagedConfigPath_Rejects covers the paths that must never
// resolve, each with the reason it is refused.
func TestResolveManagedConfigPath_Rejects(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()

	writeFile(t, filepath.Join(root, "sim.yaml"))
	outsideFile := writeFile(t, filepath.Join(outside, "secret.yaml"))

	tests := []struct {
		name string
		path string
	}{
		{"traversal component", filepath.Join(root, "..", "escape.yaml")},
		{"plainly outside the roots", outsideFile},
		{"a directory rather than a file", root},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := config.ResolveManagedConfigPath(tt.path, []string{root})
			if err == nil {
				t.Fatalf("ResolveManagedConfigPath(%q) resolved to %q, want a refusal", tt.path, got)
			}
			if !errors.Is(err, config.ErrPathOutsideManagedRoots) {
				t.Errorf("error = %v, want ErrPathOutsideManagedRoots", err)
			}
		})
	}
}

// TestResolveManagedConfigPath_SymlinkEscapeIsRefusedBeforeStat is why
// containment is re-checked immediately after symlink resolution.
//
// The literal path sits inside a managed root, so the pre-resolution check
// passes; only the symlink target is outside. Until the post-resolution check
// runs, the resolved path is still untrusted — statting it first both touches a
// path outside the roots and lets the outcome describe what is there. The
// assertion is therefore not just "an error" but "the plain containment error",
// with nothing said about what the escaped path turned out to be.
func TestResolveManagedConfigPath_SymlinkEscapeIsRefusedBeforeStat(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()

	// A directory outside the roots. A directory rather than a file precisely
	// because that is what the stat-first order would have reported on: it
	// would fail with "configuration must be a regular file", which only a
	// caller who could reach the target would learn.
	escapeTarget := filepath.Join(outside, "not-a-config")
	if err := os.Mkdir(escapeTarget, 0o700); err != nil {
		t.Fatalf("mkdir escape target: %v", err)
	}

	link := filepath.Join(root, "sim.yaml")
	if err := os.Symlink(escapeTarget, link); err != nil {
		t.Skipf("symlinks unsupported on this platform: %v", err)
	}

	// Confirm the premise: the literal path really is inside the root, so the
	// pre-resolution containment check cannot be what rejects it.
	if !strings.HasPrefix(link, root) {
		t.Fatalf("premise broken: %q is not inside %q", link, root)
	}

	got, err := config.ResolveManagedConfigPath(link, []string{root})
	if err == nil {
		t.Fatalf("ResolveManagedConfigPath resolved a symlink escape to %q", got)
	}
	if !errors.Is(err, config.ErrPathOutsideManagedRoots) {
		t.Fatalf("error = %v, want ErrPathOutsideManagedRoots", err)
	}
	if strings.Contains(err.Error(), "regular file") {
		t.Errorf("error = %q; the refusal describes the escaped path, so containment "+
			"was checked after the stat rather than before it", err)
	}
}

// TestResolveManagedConfigPath_SymlinkInsideRootStillResolves keeps the check
// from being a blanket ban on symlinks: one that stays inside a managed root is
// a legitimate layout and must still work.
func TestResolveManagedConfigPath_SymlinkInsideRootStillResolves(t *testing.T) {
	root := t.TempDir()

	target := writeFile(t, filepath.Join(root, "real.yaml"))
	link := filepath.Join(root, "sim.yaml")

	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unsupported on this platform: %v", err)
	}

	got, err := config.ResolveManagedConfigPath(link, []string{root})
	if err != nil {
		t.Fatalf("ResolveManagedConfigPath on an in-root symlink: %v", err)
	}

	want, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	if got != want {
		t.Errorf("resolved path = %q, want %q", got, want)
	}
}
