package templates

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Find searches for a template by name across all template directories
// and returns its absolute path, or empty string if not found. It is
// used both by the HTTP content/use handlers and by the daemon's
// StartSimulation path, which loads a template directly off disk so that
// include_path resolves against the template's own source directory.
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

// Load finds and reads a template by name, returning its raw content and
// resolved on-disk path.
func Load(name string) ([]byte, string, error) {
	templatePath := Find(name)
	if templatePath == "" {
		return nil, "", fmt.Errorf("template not found: %s", name)
	}

	content, err := os.ReadFile(filepath.Clean(templatePath))
	if err != nil {
		return nil, "", errors.New("failed to read template")
	}

	return content, templatePath, nil
}

// SaveConfig writes template content to a new config file under the
// configs/ directory. newConfigName is optional; when empty a name is
// derived from templateName. The resolved absolute config path is
// returned.
func SaveConfig(templateName, newConfigName string, content []byte) (string, error) {
	configName := newConfigName
	if configName == "" {
		configName = templateName + "-config"
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
// both the template-use handler and the api config-create handler.
func SanitizeConfigName(name string) string {
	// Only allow alphanumeric, dash, underscore
	return configNameUnsafe.ReplaceAllString(name, "-")
}
