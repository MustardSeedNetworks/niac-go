package api

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

// getUserConfigDirs returns the legacy directories the pre-#897-L4 API
// server scanned for user-saved YAML configs (the removed
// /api/v1/configs surface — see internal/api/configs.go's git history).
// migrateLegacyUserConfigs is the only remaining caller: the directories
// are still consulted, read-only, once at startup, so nothing already on
// an operator's disk goes missing when they upgrade.
// NIAC_CONFIGS_DIR replaces the built-in locations rather than sitting in
// front of them: an operator who names a configs directory is naming the only
// one, and searching $HOME behind it migrated another install's networks into
// a deliberately isolated library (P1-16).
func getUserConfigDirs() []string {
	if customDir := os.Getenv("NIAC_CONFIGS_DIR"); customDir != "" {
		return []string{customDir}
	}
	return []string{
		// Working directory configs
		"configs",
		// System-wide user configs
		"/var/lib/niac/configs",
		// User-specific configs
		os.ExpandEnv("$HOME/.niac/configs"),
	}
}

// isDaemonInlineConfig reports whether a legacy filename is the daemon's own
// materialisation of a POSTed config — `_running.inline.yaml`, or
// `_running.<session>.inline.yaml` for a named session (see
// internal/daemon's inlineConfigName). Those are scratch files, not something
// an operator authored, so migrating them published a running scenario under
// a reserved name.
func isDaemonInlineConfig(name string) bool {
	return strings.HasPrefix(name, "_running.") && strings.HasSuffix(name, "inline")
}

// legacyConfigCandidate is one YAML file found under a legacy config
// directory, pending a name-collision check against the library.
type legacyConfigCandidate struct {
	path string
	name string
}

// collectLegacyConfigCandidates walks dir (one of getUserConfigDirs'
// entries) and returns every *.yaml/*.yml file found. It never reads file
// contents itself — that happens afterward, outside the WalkDir callback,
// so a slow or malicious legacy directory can't stall directory traversal
// with file I/O interleaved (gosec G122). A dir that doesn't exist yields
// no candidates and no error, matching the pre-#897-L4 scan behavior.
func collectLegacyConfigCandidates(dir string) ([]legacyConfigCandidate, error) {
	var candidates []legacyConfigCandidate
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil //nolint:nilerr // legacy dirs may not exist; skip and keep walking
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if isDaemonInlineConfig(name) {
			return nil
		}
		candidates = append(candidates, legacyConfigCandidate{path: path, name: name})
		return nil
	})
	return candidates, err
}

// migrateLegacyUserConfigs copies every legacy user config YAML that
// isn't already present in the library's networks/ store into it. It is:
//
//   - Non-destructive: legacy files are only ever read, never modified or
//     removed, so they remain a safety net on disk.
//   - Idempotent: a name already in the library (case-insensitive, same
//     rule findUserConfig used to apply) is left alone, so a second call
//     — e.g. the next daemon restart — migrates nothing new.
//
// Returns the number of configs migrated so the caller can log a summary.
func migrateLegacyUserConfigs(lib *library.Library) (int, error) {
	return migrateLegacyUserConfigsFromDirs(lib, getUserConfigDirs())
}

// migrateLegacyUserConfigsFromDirs is migrateLegacyUserConfigs with an
// explicit dir list, so tests can exercise the migration against fixture
// directories instead of the real $HOME / /var/lib/niac paths.
func migrateLegacyUserConfigsFromDirs(lib *library.Library, dirs []string) (int, error) {
	existing, err := lib.ListNetworks()
	if err != nil {
		return 0, fmt.Errorf("list library networks: %w", err)
	}
	have := make(map[string]bool, len(existing))
	for _, entry := range existing {
		have[strings.ToLower(entry.Name)] = true
	}

	migrated := 0
	for _, dir := range dirs {
		candidates, walkErr := collectLegacyConfigCandidates(dir)
		if walkErr != nil {
			return migrated, fmt.Errorf("walk legacy config dir %s: %w", dir, walkErr)
		}
		for _, candidate := range candidates {
			if have[strings.ToLower(candidate.name)] {
				continue
			}
			if !migrateOneLegacyConfig(lib, candidate) {
				continue
			}
			have[strings.ToLower(candidate.name)] = true
			migrated++
		}
	}

	return migrated, nil
}

// migrateOneLegacyConfig reads and writes a single candidate, reporting
// success. Read/parse/validation failures are best-effort skips — the
// legacy file is left in place and the migration moves on to the rest.
func migrateOneLegacyConfig(lib *library.Library, candidate legacyConfigCandidate) bool {
	// #nosec G304 -- candidate.path came from collectLegacyConfigCandidates
	// walking one of the fixed legacy config directories, not user input.
	content, readErr := os.ReadFile(candidate.path)
	if readErr != nil {
		return false
	}
	if writeErr := lib.WriteNetwork(candidate.name, string(content)); writeErr != nil {
		// e.g. an invalid name or content missing "devices:".
		return false
	}
	return true
}
