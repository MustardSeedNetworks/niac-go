package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Template type constants matching the UI interface.
const (
	templateTypeBasic       = "basic"
	templateTypeRouter      = "router"
	templateTypeSwitch      = "switch"
	templateTypeAccessPoint = "access-point"
	templateTypeServer      = "server"
	templateTypeComplete    = "complete"
	templateTypeCustom      = "custom"
)

// Template represents a configuration template.
//
// Name is the stable filename-derived identifier ("minimal", "router")
// used by the /api/v1/templates/{name} and /api/v1/templates/use APIs.
// DisplayName is the human-readable label optionally provided via the
// front-matter "# Display: ..." line; clients should prefer it for UI
// rendering and fall back to Name when absent.
type Template struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName,omitempty"`
	Description string   `json:"description"`
	DeviceCount int      `json:"deviceCount"`
	Type        string   `json:"type"`
	Tags        []string `json:"tags,omitempty"`
}

// TemplateContent represents the full content of a template.
type TemplateContent struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Format  string `json:"format"`
}

// UseTemplateRequest is the request to create a config from a template.
type UseTemplateRequest struct {
	TemplateName  string `json:"templateName"`
	NewConfigName string `json:"newConfigName,omitempty"`
}

// UseTemplateResponse is the response after creating a config from a template.
type UseTemplateResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	ConfigPath string `json:"configPath"`
}

// getTemplateDirs returns the directories to scan for templates.
// Checks multiple locations for compatibility with both development and installed deployments.
func getTemplateDirs() []string {
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

// handleTemplates handles GET /api/v1/templates (list) and POST (upload).
func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleTemplatesList(w, r)
	case http.MethodPost:
		s.handleTemplateUpload(w, r)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
	}
}

// handleTemplateByName handles GET /api/v1/templates/{name} and DELETE.
func (s *Server) handleTemplateByName(w http.ResponseWriter, r *http.Request) {
	// Extract template name from path: /api/v1/templates/{name}
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/templates/")

	// Handle /api/v1/templates/use separately
	if name == "use" {
		s.handleTemplateUse(w, r)
		return
	}

	if name == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Template name required", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleTemplateContent(w, r, name)
	case http.MethodDelete:
		s.handleTemplateDelete(w, r, name)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
	}
}

// handleTemplatesList returns a list of available templates.
func (s *Server) handleTemplatesList(w http.ResponseWriter, _ *http.Request) {
	templates := []Template{}

	for _, dir := range getTemplateDirs() {
		dirTemplates, err := scanTemplateDir(dir)
		if err != nil {
			s.logger.Warn(fmt.Sprintf("Failed to scan template dir %s: %v", dir, err))
			continue
		}
		templates = append(templates, dirTemplates...)
	}

	s.writeJSON(w, templates)
}

// scanTemplateDir scans a directory for YAML template files.
func scanTemplateDir(baseDir string) ([]Template, error) {
	var templates []Template

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

		template := parseTemplateFile(path, info)
		templates = append(templates, template)
		return nil
	})

	return templates, err
}

// parseTemplateFile extracts template metadata from a file.
//
// Templates can opt-in to a small front-matter block at the very top of
// the YAML for richer UI metadata. The parser walks consecutive comment
// lines and recognises:
//
//	# Display: Enterprise Router
//	# Description: A full enterprise router persona with LLDP, CDP, SNMP, ...
//
// When present these win over the legacy "first non-empty comment line"
// heuristic. The keys are case-insensitive and the rest of the file is
// untouched.
func parseTemplateFile(path string, _ fs.FileInfo) Template {
	// Extract name from filename without extension
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

	meta := extractFrontMatter(path)

	displayName := meta["display"]

	description := meta["description"]
	if description == "" {
		// Fall back to the legacy first-comment heuristic for templates
		// that haven't been migrated yet.
		description = extractDescription(path)
	}
	if description == "" {
		base := displayName
		if base == "" {
			base = name
		}
		description = base + " configuration template"
	}

	// Count devices in the YAML file
	deviceCount := countDevicesInFile(path)

	// Determine type: front-matter wins, otherwise infer from name+path.
	templateType := strings.ToLower(strings.TrimSpace(meta["type"]))
	if templateType == "" {
		templateType = determineTemplateType(path, name)
	}

	// Tags: front-matter "tags:" comma-separated wins, otherwise infer.
	tags := splitTags(meta["tags"])
	if len(tags) == 0 {
		tags = generateTags(path)
	}

	return Template{
		Name:        name,
		DisplayName: displayName,
		Description: description,
		DeviceCount: deviceCount,
		Type:        templateType,
		Tags:        tags,
	}
}

// splitTags parses a comma-separated tags string into a slice. Empty tags
// are skipped and whitespace is trimmed.
func splitTags(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// extractFrontMatter pulls "# Key: Value" lines from the top of a YAML
// file into a map. Stops at the first non-comment, non-blank line. Keys
// are lowercased; values are trimmed.
func extractFrontMatter(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	meta := make(map[string]string)
	for line := range strings.SplitSeq(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		comment, isComment := strings.CutPrefix(trimmed, "#")
		if !isComment {
			break
		}
		comment = strings.TrimSpace(comment)
		key, value, ok := strings.Cut(comment, ":")
		if !ok {
			continue
		}
		k := strings.ToLower(strings.TrimSpace(key))
		v := strings.TrimSpace(value)
		if k != "" && v != "" {
			meta[k] = v
		}
	}
	return meta
}

// countDevicesInFile counts the number of devices defined in a YAML config.
func countDevicesInFile(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 1 // Default to 1 if can't read
	}

	// Simple heuristic: count lines that start with "  - name:" (device entries)
	count := 0
	for line := range strings.SplitSeq(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- name:") {
			count++
		}
	}

	if count == 0 {
		return 1 // At least 1 device assumed
	}
	return count
}

// determineTemplateType determines the template type from its path and name.
func determineTemplateType(path, name string) string {
	pathLower := strings.ToLower(path)
	nameLower := strings.ToLower(name)

	// Check filename first for specific types
	switch {
	case strings.Contains(nameLower, "router"):
		return templateTypeRouter
	case strings.Contains(nameLower, "switch"):
		return templateTypeSwitch
	case strings.Contains(nameLower, "ap") || strings.Contains(nameLower, "wireless") ||
		strings.Contains(nameLower, "access-point"):
		return templateTypeAccessPoint
	case strings.Contains(nameLower, "server"):
		return templateTypeServer
	case strings.Contains(nameLower, "complete") || strings.Contains(nameLower, "full"):
		return templateTypeComplete
	}

	// Check path for type hints
	switch {
	case strings.Contains(pathLower, "/services/"):
		return templateTypeServer
	case strings.Contains(pathLower, "/combinations/"):
		return templateTypeComplete
	case strings.Contains(pathLower, "/vendors/"):
		return templateTypeCustom
	default:
		return templateTypeBasic
	}
}

// extractDescription extracts description from first comment line of YAML file.
func extractDescription(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if desc, found := strings.CutPrefix(line, "#"); found {
			desc = strings.TrimSpace(desc)
			if desc != "" && !strings.HasPrefix(strings.ToLower(desc), "yaml") {
				return desc
			}
		} else if line != "" {
			// First non-comment, non-empty line - stop
			break
		}
	}
	return ""
}

// generateTags generates tags from the file path.
func generateTags(path string) []string {
	tags := []string{}
	pathLower := strings.ToLower(path)

	// Add tags based on path components
	tagPatterns := map[string]string{
		"router":     "router",
		"switch":     "switch",
		"wireless":   "wireless",
		"ap":         "wireless",
		"snmp":       "snmp",
		"firewall":   "security",
		"security":   "security",
		"datacenter": "datacenter",
		"enterprise": "enterprise",
		"campus":     "enterprise",
	}

	for pattern, tag := range tagPatterns {
		if strings.Contains(pathLower, pattern) && !slices.Contains(tags, tag) {
			tags = append(tags, tag)
		}
	}

	// Add vendor tags
	vendors := []string{
		"cisco", "juniper", "arista", "dell", "aruba",
		"meraki", "fortinet", "paloalto", "extreme", "foundry",
	}
	for _, vendor := range vendors {
		if strings.Contains(pathLower, vendor) {
			tags = append(tags, vendor)
		}
	}

	return tags
}

// handleTemplateContent returns the content of a specific template.
func (s *Server) handleTemplateContent(w http.ResponseWriter, r *http.Request, name string) {
	// Security: validate name doesn't contain path traversal
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		writeError(w, r, http.StatusBadRequest, "invalid_name", "Invalid template name", nil)
		return
	}

	// Search for template using recursive search
	foundPath := findTemplateRecursive(name)

	if foundPath == "" {
		writeError(w, r, http.StatusNotFound, "not_found", "Template not found: "+name, nil)
		return
	}

	content, err := os.ReadFile(foundPath)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "read_error", "Failed to read template", nil)
		return
	}

	response := TemplateContent{
		Name:    name,
		Content: string(content),
		Format:  "yaml",
	}

	s.writeJSON(w, response)
}

// findTemplateRecursive searches for a template by name in all template directories.
func findTemplateRecursive(name string) string {
	for _, dir := range getTemplateDirs() {
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

// handleTemplateUse handles POST /api/v1/templates/use.
func (s *Server) handleTemplateUse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
		return
	}

	req, err := parseUseTemplateRequest(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}

	content, templatePath, err := loadTemplate(req.TemplateName)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", err.Error(), nil)
		return
	}

	configPath, err := saveConfigFromTemplate(req, content)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "write_error", err.Error(), nil)
		return
	}

	_ = templatePath // Used for logging if needed
	response := UseTemplateResponse{
		Success:    true,
		Message:    fmt.Sprintf("Configuration created from template '%s'", req.TemplateName),
		ConfigPath: configPath,
	}

	s.writeJSON(w, response)
}

// parseUseTemplateRequest parses and validates the use template request.
func parseUseTemplateRequest(r *http.Request) (*UseTemplateRequest, error) {
	var req UseTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.New("invalid JSON body")
	}

	if req.TemplateName == "" {
		return nil, errors.New("templateName is required")
	}

	return &req, nil
}

// loadTemplate finds and reads a template by name.
func loadTemplate(name string) ([]byte, string, error) {
	templatePath := findTemplateRecursive(name)
	if templatePath == "" {
		return nil, "", fmt.Errorf("template not found: %s", name)
	}

	content, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, "", errors.New("failed to read template")
	}

	return content, templatePath, nil
}

// saveConfigFromTemplate creates a config file from template content.
func saveConfigFromTemplate(req *UseTemplateRequest, content []byte) (string, error) {
	configName := req.NewConfigName
	if configName == "" {
		configName = req.TemplateName + "-config"
	}

	configName = sanitizeConfigName(configName)
	configDir := "configs"

	// Use secure permissions: 0750 for directory (owner rwx, group rx)
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		return "", errors.New("failed to create configs directory")
	}

	configPath := filepath.Clean(filepath.Join(configDir, configName+".yaml"))

	// Defense-in-depth: configName has been sanitized upstream
	// (sanitizeConfigName), but make the bounded path explicit so static
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

// sanitizeConfigName removes unsafe characters from config name.
func sanitizeConfigName(name string) string {
	// Only allow alphanumeric, dash, underscore
	re := regexp.MustCompile(`[^a-zA-Z0-9\-_]`)
	return re.ReplaceAllString(name, "-")
}

// handleTemplateUpload handles POST /api/v1/templates (upload new template).
func (s *Server) handleTemplateUpload(w http.ResponseWriter, r *http.Request) {
	// For now, return not implemented
	writeError(w, r, http.StatusNotImplemented, "not_implemented", "Template upload not yet implemented", nil)
}

// handleTemplateDelete handles DELETE /api/v1/templates/{name}.
func (s *Server) handleTemplateDelete(w http.ResponseWriter, r *http.Request, _ string) {
	// For now, return not implemented (don't want to delete built-in templates)
	writeError(w, r, http.StatusNotImplemented, "not_implemented", "Template deletion not yet implemented", nil)
}
