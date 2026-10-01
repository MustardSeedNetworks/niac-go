package builtins

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ErrNotFound is returned by Load when no scenario matches the
// requested name. Callers that need to distinguish "not found" (404) from
// a read failure (500) should check for it with errors.Is.
var ErrNotFound = errors.New("built-in scenario not found")

// Find searches for a scenario by name across all scenario directories
// and returns its absolute path, or empty string if not found. It is
// used both by the HTTP content and copy handlers and by the daemon's
// StartSimulation path, which loads a scenario directly off disk so that
// include_path resolves against the scenario's own source directory.
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

// Load finds and reads a scenario by name, returning its raw content and
// resolved on-disk path.
func Load(name string) ([]byte, string, error) {
	scenarioPath := Find(name)
	if scenarioPath == "" {
		return nil, "", fmt.Errorf("%w: %s", ErrNotFound, name)
	}

	content, err := os.ReadFile(filepath.Clean(scenarioPath))
	if err != nil {
		return nil, "", errors.New("failed to read built-in scenario")
	}

	return content, scenarioPath, nil
}

// SaveConfig writes scenario content to a new config file under the
// configs/ directory. newConfigName is optional; when empty a name is
// derived from scenarioName. The resolved absolute config path is
// returned.
func SaveConfig(scenarioName, newConfigName string, content []byte) (string, error) {
	configName := newConfigName
	if configName == "" {
		configName = scenarioName + "-config"
	}

	configName = SanitizeConfigName(configName)
	configDir := "configs"

	// Use secure permissions: 0750 for directory (owner rwx, group rx)
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		return "", errors.New("failed to create configs directory")
	}

	configPath := filepath.Clean(filepath.Join(configDir, configName+".yaml"))

	// Defense-in-depth: configName has been sanitized upstream
	// (SanitizeConfigName), but make the bounded path explicit so static
	// analysers see the barrier.
	absConfigPath, absErr := filepath.Abs(configPath)
	if absErr != nil || strings.Contains(configPath, "..") {
		return "", errors.New("invalid config path")
	}
	absConfigDir, absDirErr := filepath.Abs(configDir)
	if absDirErr != nil || !strings.HasPrefix(absConfigPath, absConfigDir+string(filepath.Separator)) {
		return "", errors.New("invalid config path")
	}

	// Use secure permissions: 0600 for file (owner rw only)
	if err := os.WriteFile(absConfigPath, content, 0o600); err != nil {
		return "", errors.New("failed to write config file")
	}

	return absConfigPath, nil
}

// configNameUnsafe matches any character not allowed in a config name.
var configNameUnsafe = regexp.MustCompile(`[^a-zA-Z0-9\-_]`)

// SanitizeConfigName removes unsafe characters from a config name,
// replacing each with a dash. It is a shared config-file utility used by
// both the scenario-copy handler and the api config-create handler.
func SanitizeConfigName(name string) string {
	// Only allow alphanumeric, dash, underscore
	return configNameUnsafe.ReplaceAllString(name, "-")
}
