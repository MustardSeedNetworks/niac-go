package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Template represents a configuration template.
type Template struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	DeviceCount int      `json:"device_count"`
	Type        string   `json:"type"`
	Tags        []string `json:"tags,omitempty"`
	CreatedAt   string   `json:"created_at,omitempty"`
	ModifiedAt  string   `json:"modified_at,omitempty"`
}

// TemplateContent represents the full content of a template.
type TemplateContent struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Format  string `json:"format"`
}

// UseTemplateRequest represents a request to apply a template.
type UseTemplateRequest struct {
	TemplateName  string `json:"template_name"`
	NewConfigName string `json:"new_config_name,omitempty"`
}

// UseTemplateResponse represents the response from applying a template.
type UseTemplateResponse struct {
	Success    bool   `json:"success"`
	ConfigPath string `json:"config_path"`
	Message    string `json:"message"`
}

// UploadTemplateRequest represents a request to upload a new template.
type UploadTemplateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
	Type        string `json:"type,omitempty"`
}

// UploadTemplateResponse represents the response from uploading a template.
type UploadTemplateResponse struct {
	Success  bool     `json:"success"`
	Template Template `json:"template"`
	Message  string   `json:"message"`
}

// templateStore manages in-memory template storage with optional file persistence.
type templateStore struct {
	mu        sync.RWMutex
	templates map[string]*templateEntry
	dir       string // directory for file-backed templates (optional)
}

type templateEntry struct {
	meta    Template
	content string
}

const (
	maxTemplateName    = 128
	maxTemplateSize    = 1 << 20 // 1MB
	templateTimeFormat = time.RFC3339
)

// isValidTemplateName checks that the name is safe for use as a filename.
// Rejects path traversal, special characters, and empty names.
func isValidTemplateName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	for _, ch := range name {
		switch {
		case ch >= 'a' && ch <= 'z':
		case ch >= 'A' && ch <= 'Z':
		case ch >= '0' && ch <= '9':
		case ch == '-' || ch == '_' || ch == '.':
		default:
			return false
		}
	}
	// Reject names that start with a dot (hidden files)
	if name[0] == '.' {
		return false
	}
	return true
}

func newTemplateStore(dir string) *templateStore {
	ts := &templateStore{
		templates: make(map[string]*templateEntry),
		dir:       dir,
	}
	if dir != "" {
		ts.loadFromDir()
	}
	return ts
}

func (ts *templateStore) loadFromDir() {
	entries, err := os.ReadDir(ts.dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ext)
		data, readErr := os.ReadFile(filepath.Join(ts.dir, entry.Name()))
		if readErr != nil {
			continue
		}
		info, _ := entry.Info()
		meta := Template{
			Name:        name,
			Type:        "custom",
			DeviceCount: countDevicesInYAML(data),
		}
		if info != nil {
			meta.CreatedAt = info.ModTime().Format(templateTimeFormat)
			meta.ModifiedAt = info.ModTime().Format(templateTimeFormat)
		}
		ts.templates[name] = &templateEntry{
			meta:    meta,
			content: string(data),
		}
	}
}

func countDevicesInYAML(data []byte) int {
	var raw struct {
		Devices []any `yaml:"devices"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return 0
	}
	return len(raw.Devices)
}

func (ts *templateStore) list() []Template {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	result := make([]Template, 0, len(ts.templates))
	for _, entry := range ts.templates {
		result = append(result, entry.meta)
	}
	return result
}

func (ts *templateStore) get(name string) (*templateEntry, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	entry, ok := ts.templates[name]
	return entry, ok
}

func (ts *templateStore) add(name string, entry *templateEntry) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.templates[name] = entry
	if ts.dir != "" {
		filePath := filepath.Join(ts.dir, name+".yaml")
		_ = os.WriteFile(filePath, []byte(entry.content), 0o600)
	}
}

func (ts *templateStore) remove(name string) bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if _, ok := ts.templates[name]; !ok {
		return false
	}
	delete(ts.templates, name)
	if ts.dir != "" {
		filePath := filepath.Join(ts.dir, name+".yaml")
		_ = os.Remove(filePath)
	}
	return true
}

// handleTemplates routes template requests by method.
func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleTemplatesList(w, r)
	case http.MethodPost:
		s.handleTemplateUpload(w, r)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed",
			"Method not allowed", nil)
	}
}

// handleTemplateByName routes individual template requests.
func (s *Server) handleTemplateByName(w http.ResponseWriter, r *http.Request) {
	// Extract template name from URL: /api/v1/templates/{name}
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/templates/")
	if name == "" {
		writeError(w, r, http.StatusBadRequest, "missing_name",
			"Template name is required", nil)
		return
	}

	// Validate name length
	if len(name) > maxTemplateName {
		writeError(w, r, http.StatusBadRequest, "name_too_long",
			"Template name exceeds maximum length", nil)
		return
	}

	// SECURITY: Validate name to prevent path traversal
	if !isValidTemplateName(name) {
		writeError(w, r, http.StatusBadRequest, "invalid_name",
			"Template name contains invalid characters (alphanumeric, hyphens, underscores only)", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleTemplateGet(w, r, name)
	case http.MethodDelete:
		s.handleTemplateDelete(w, r, name)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed",
			"Method not allowed", nil)
	}
}

// handleTemplateUse applies a template.
func (s *Server) handleTemplateUse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed",
			"Method not allowed", nil)
		return
	}

	var req UseTemplateRequest
	body := http.MaxBytesReader(w, r.Body, MaxRequestBodySize)
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json",
			"Invalid JSON request body", nil)
		return
	}

	if req.TemplateName == "" {
		writeError(w, r, http.StatusBadRequest, "missing_template_name",
			"template_name is required", nil)
		return
	}

	entry, ok := s.templateStore.get(req.TemplateName)
	if !ok {
		writeError(w, r, http.StatusNotFound, "template_not_found",
			fmt.Sprintf("Template %q not found", req.TemplateName), nil)
		return
	}

	// Determine output config name
	configName := req.NewConfigName
	if configName == "" {
		configName = req.TemplateName + "-config"
	}

	// Write config to the config directory
	configPath := ""
	s.configMu.RLock()
	if s.cfg.ConfigPath != "" {
		configPath = filepath.Join(filepath.Dir(s.cfg.ConfigPath), configName+".yaml")
	}
	s.configMu.RUnlock()

	if configPath == "" {
		writeError(w, r, http.StatusInternalServerError, "no_config_path",
			"Config path not available", nil)
		return
	}

	if err := os.WriteFile(configPath, []byte(entry.content), 0o600); err != nil {
		writeError(w, r, http.StatusInternalServerError, "write_failed",
			"Failed to write config from template", nil)
		return
	}

	s.writeJSON(w, UseTemplateResponse{
		Success:    true,
		ConfigPath: configPath,
		Message:    fmt.Sprintf("Template %q applied as %s", req.TemplateName, configName),
	})
}

func (s *Server) handleTemplatesList(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, s.templateStore.list())
}

func (s *Server) handleTemplateGet(w http.ResponseWriter, r *http.Request, name string) {
	entry, ok := s.templateStore.get(name)
	if !ok {
		writeError(w, r, http.StatusNotFound, "template_not_found",
			fmt.Sprintf("Template %q not found", name), nil)
		return
	}

	s.writeJSON(w, TemplateContent{
		Name:    name,
		Content: entry.content,
		Format:  "yaml",
	})
}

func (s *Server) handleTemplateUpload(w http.ResponseWriter, r *http.Request) {
	var req UploadTemplateRequest
	body := http.MaxBytesReader(w, r.Body, MaxRequestBodySize)
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json",
			"Invalid JSON request body", nil)
		return
	}

	if req.Name == "" {
		writeError(w, r, http.StatusBadRequest, "missing_name",
			"Template name is required", nil)
		return
	}
	if len(req.Name) > maxTemplateName {
		writeError(w, r, http.StatusBadRequest, "name_too_long",
			"Template name exceeds maximum length", nil)
		return
	}
	// SECURITY: Validate name to prevent path traversal
	if !isValidTemplateName(req.Name) {
		writeError(w, r, http.StatusBadRequest, "invalid_name",
			"Template name contains invalid characters (alphanumeric, hyphens, underscores only)", nil)
		return
	}
	if req.Content == "" {
		writeError(w, r, http.StatusBadRequest, "missing_content",
			"Template content is required", nil)
		return
	}
	if len(req.Content) > maxTemplateSize {
		writeError(w, r, http.StatusBadRequest, "content_too_large",
			"Template content exceeds maximum size (1MB)", nil)
		return
	}

	templateType := req.Type
	if templateType == "" {
		templateType = "custom"
	}

	now := time.Now().Format(templateTimeFormat)
	meta := Template{
		Name:        req.Name,
		Description: req.Description,
		Type:        templateType,
		DeviceCount: countDevicesInYAML([]byte(req.Content)),
		CreatedAt:   now,
		ModifiedAt:  now,
	}

	s.templateStore.add(req.Name, &templateEntry{
		meta:    meta,
		content: req.Content,
	})

	w.WriteHeader(http.StatusCreated)
	s.writeJSON(w, UploadTemplateResponse{
		Success:  true,
		Template: meta,
		Message:  fmt.Sprintf("Template %q created", req.Name),
	})
}

func (s *Server) handleTemplateDelete(w http.ResponseWriter, r *http.Request, name string) {
	if !s.templateStore.remove(name) {
		writeError(w, r, http.StatusNotFound, "template_not_found",
			fmt.Sprintf("Template %q not found", name), nil)
		return
	}

	s.writeJSON(w, struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}{
		Success: true,
		Message: fmt.Sprintf("Template %q deleted", name),
	})
}
