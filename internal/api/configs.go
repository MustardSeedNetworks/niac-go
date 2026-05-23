package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// UserConfig represents a user-uploaded configuration file.
type UserConfig struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	DeviceCount int       `json:"deviceCount"`
	ModifiedAt  time.Time `json:"modifiedAt"`
	Size        int64     `json:"size"`
}

// UserConfigContent represents the full content of a user config.
type UserConfigContent struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Format  string `json:"format"`
}

// UploadConfigRequest is the request to upload a new user config.
type UploadConfigRequest struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// UploadConfigResponse is the response after uploading a config.
type UploadConfigResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Path    string `json:"path"`
}

// getUserConfigDirs returns the directories to scan for user configs.
func getUserConfigDirs() []string {
	dirs := []string{
		// Working directory configs
		"configs",
		// System-wide user configs
		"/var/lib/niac/configs",
		// User-specific configs
		os.ExpandEnv("$HOME/.niac/configs"),
	}

	// Add custom directory from environment variable
	if customDir := os.Getenv("NIAC_CONFIGS_DIR"); customDir != "" {
		dirs = append([]string{customDir}, dirs...)
	}

	return dirs
}

// getDefaultConfigDir returns the default directory for storing new user configs.
func getDefaultConfigDir() string {
	// Check environment variable first
	if customDir := os.Getenv("NIAC_CONFIGS_DIR"); customDir != "" {
		return customDir
	}

	// Use working directory configs folder
	return "configs"
}

// handleUserConfigs handles GET /api/v1/configs (list) and POST (upload).
func (s *Server) handleUserConfigs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleUserConfigsList(w, r)
	case http.MethodPost:
		s.handleUserConfigUpload(w, r)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
	}
}

// handleUserConfigByName handles GET /api/v1/configs/{name} and DELETE.
func (s *Server) handleUserConfigByName(w http.ResponseWriter, r *http.Request) {
	// Extract config name from path: /api/v1/configs/{name}
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/configs/")

	if name == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Config name required", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleUserConfigContent(w, r, name)
	case http.MethodDelete:
		s.handleUserConfigDelete(w, r, name)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
	}
}

// handleUserConfigsList returns a list of user-uploaded configs.
func (s *Server) handleUserConfigsList(w http.ResponseWriter, r *http.Request) {
	configs := []UserConfig{}

	for _, dir := range getUserConfigDirs() {
		dirConfigs, err := scanUserConfigDir(dir)
		if err != nil {
			s.logger.WarnContext(r.Context(), fmt.Sprintf("Failed to scan config dir %s: %v", dir, err))
			continue
		}
		configs = append(configs, dirConfigs...)
	}

	s.writeJSON(w, map[string]any{
		"configs": configs,
	})
}

// scanUserConfigDir scans a directory for user YAML config files.
func scanUserConfigDir(baseDir string) ([]UserConfig, error) {
	var configs []UserConfig

	err := filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, walkErr error) error {
		// Skip entries with errors
		if walkErr != nil || d.IsDir() {
			return nil //nolint:nilerr // WalkDir: skip errors, continue walking
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			return nil //nolint:nilerr // WalkDir: skip files we can't stat
		}

		config := parseUserConfigFile(path, info)
		configs = append(configs, config)
		return nil
	})

	return configs, err
}

// parseUserConfigFile extracts config metadata from a file.
func parseUserConfigFile(path string, info fs.FileInfo) UserConfig {
	// Extract name from filename without extension
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

	// Count devices in the YAML file
	deviceCount := countDevicesInFile(path)

	return UserConfig{
		Name:        name,
		Path:        path,
		DeviceCount: deviceCount,
		ModifiedAt:  info.ModTime(),
		Size:        info.Size(),
	}
}

// handleUserConfigContent returns the content of a specific user config.
func (s *Server) handleUserConfigContent(w http.ResponseWriter, r *http.Request, name string) {
	// Security: validate name doesn't contain path traversal
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		writeError(w, r, http.StatusBadRequest, "invalid_name", "Invalid config name", nil)
		return
	}

	// Search for config in all user config directories
	foundPath := findUserConfig(name)

	if foundPath == "" {
		writeError(w, r, http.StatusNotFound, "not_found", "Config not found: "+name, nil)
		return
	}

	content, err := os.ReadFile(foundPath)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "read_error", "Failed to read config", nil)
		return
	}

	response := UserConfigContent{
		Name:    name,
		Content: string(content),
		Format:  "yaml",
	}

	s.writeJSON(w, response)
}

// findUserConfig searches for a user config by name in all config directories.
func findUserConfig(name string) string {
	for _, dir := range getUserConfigDirs() {
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

// handleUserConfigUpload handles POST /api/v1/configs (upload new config).
func (s *Server) handleUserConfigUpload(w http.ResponseWriter, r *http.Request) {
	// Enforce request body size limit
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

	var req UploadConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err.Error() == ErrMsgRequestBodyTooLarge {
			writeError(w, r, http.StatusRequestEntityTooLarge, "request_too_large",
				"Config file too large", nil)
			return
		}
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON body", nil)
		return
	}

	// Validate request
	if req.Name == "" {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "Config name is required", nil)
		return
	}
	if req.Content == "" {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "Config content is required", nil)
		return
	}

	// Sanitize and validate name
	safeName := sanitizeConfigName(req.Name)
	if safeName == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_name", "Invalid config name", nil)
		return
	}

	// Check if config already exists
	existingPath := findUserConfig(safeName)
	if existingPath != "" {
		writeError(w, r, http.StatusConflict, "already_exists",
			fmt.Sprintf("Config '%s' already exists", safeName), nil)
		return
	}

	// Save the config file
	configPath, err := saveUserConfig(safeName, []byte(req.Content))
	if err != nil {
		s.logger.ErrorContext(r.Context(), "[API] Failed to save user config", "error", err)
		writeError(w, r, http.StatusInternalServerError, "write_error", "Failed to save config", nil)
		return
	}

	response := UploadConfigResponse{
		Success: true,
		Message: fmt.Sprintf("Config '%s' uploaded successfully", safeName),
		Path:    configPath,
	}

	s.writeJSON(w, response)
}

// saveUserConfig saves a user config file to the default config directory.
func saveUserConfig(name string, content []byte) (string, error) {
	configDir := getDefaultConfigDir()

	// Use secure permissions: 0750 for directory (owner rwx, group rx)
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		return "", errors.New("failed to create configs directory")
	}

	configPath := filepath.Clean(filepath.Join(configDir, name+".yaml"))

	// Defense-in-depth: name has been sanitized upstream (sanitizeConfigName),
	// but make the bounded path explicit so static analysers see the barrier.
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

// handleUserConfigDelete handles DELETE /api/v1/configs/{name}.
func (s *Server) handleUserConfigDelete(w http.ResponseWriter, r *http.Request, name string) {
	// Security: validate name doesn't contain path traversal
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		writeError(w, r, http.StatusBadRequest, "invalid_name", "Invalid config name", nil)
		return
	}

	// Find the config
	configPath := findUserConfig(name)
	if configPath == "" {
		writeError(w, r, http.StatusNotFound, "not_found", "Config not found: "+name, nil)
		return
	}

	// Check that the config is in a user-writable location (not in /usr/share or similar)
	// Only allow deletion from user config directories, not system directories
	if !isUserWritableConfigPath(configPath) {
		writeError(w, r, http.StatusForbidden, "forbidden",
			"Cannot delete configs from read-only locations", nil)
		return
	}

	// Delete the file
	if err := os.Remove(configPath); err != nil {
		s.logger.ErrorContext(r.Context(), "[API] Failed to delete user config", "error", err, "path", configPath)
		writeError(w, r, http.StatusInternalServerError, "delete_error", "Failed to delete config", nil)
		return
	}

	s.writeJSON(w, map[string]any{
		"success": true,
		"message": fmt.Sprintf("Config '%s' deleted successfully", name),
	})
}

// isUserWritableConfigPath checks if a config path is in a user-writable location.
func isUserWritableConfigPath(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// Allow deletion from these writable locations
	writablePrefixes := []string{
		"/var/lib/niac/configs",
		os.ExpandEnv("$HOME/.niac/configs"),
	}

	// Also allow the working directory configs folder
	cwd, _ := os.Getwd()
	if cwd != "" {
		writablePrefixes = append(writablePrefixes, filepath.Join(cwd, "configs"))
	}

	// Check environment variable
	if customDir := os.Getenv("NIAC_CONFIGS_DIR"); customDir != "" {
		writablePrefixes = append(writablePrefixes, customDir)
	}

	for _, prefix := range writablePrefixes {
		absPrefix, absErr := filepath.Abs(prefix)
		if absErr != nil {
			continue
		}
		if strings.HasPrefix(absPath, absPrefix) {
			return true
		}
	}

	return false
}
