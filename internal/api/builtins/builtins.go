// Package builtins finds a built-in scenario by name in the on-disk scenario
// directories: the hand-authored YAML under cmd/niac/templates in a source
// checkout, and the installed, per-user and NIAC_TEMPLATES_DIR copies of that
// tree. The daemon's StartSimulation path asks it first and falls back to the
// library, which is what an installed host has (#2131).
package builtins

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Dirs returns the directories to scan for scenarios. It checks multiple
// locations for compatibility with both development and installed
// deployments. A NIAC_TEMPLATES_DIR override, when set, takes precedence.
func Dirs() []string {
	dirs := []string{
		// Development paths (relative to working directory)
		"cmd/niac/templates",
		"examples",
		// System-wide installed paths
		"/usr/share/niac/templates",
		"/var/lib/niac/templates",
		// User-specific paths
		os.ExpandEnv("$HOME/.niac/templates"),
	}

	// Add custom directory from environment variable
	if customDir := os.Getenv("NIAC_TEMPLATES_DIR"); customDir != "" {
		dirs = append([]string{customDir}, dirs...)
	}

	return dirs
}

// Find searches for a scenario by name across all scenario directories
// and returns its path, or empty string if not found. The daemon loads the
// file in place so that include_path resolves against the scenario's own
// source directory.
func Find(name string) string {
	for _, dir := range Dirs() {
		var found string
		_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil //nolint:nilerr // WalkDir: continue on errors
			}

			baseName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			if strings.EqualFold(baseName, name) {
				found = path
				return filepath.SkipAll
			}
			return nil
		})
		if found != "" {
			return found
		}
	}
	return ""
}
