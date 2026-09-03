package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/api/templates"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

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

// handleTemplates handles GET /api/v1/templates (list).
func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	s.handleTemplatesList(w, r)
}

// handleTemplateByName handles GET /api/v1/templates/{name}.
func (s *Server) handleTemplateByName(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/templates/")
	if name == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Template name required", nil)
		return
	}
	s.handleTemplateContent(w, r, name)
}

// handleTemplatesList returns a list of available templates.
func (s *Server) handleTemplatesList(w http.ResponseWriter, r *http.Request) {
	list := []templates.Template{}

	for _, dir := range templates.Dirs() {
		dirTemplates, err := templates.Scan(dir)
		if err != nil {
			s.logger.WarnContext(r.Context(), fmt.Sprintf("Failed to scan template dir %s: %v", dir, err))
			continue
		}
		list = append(list, dirTemplates...)
	}

	s.writeJSON(w, list)
}

// handleTemplateContent returns the content of a specific template.
func (s *Server) handleTemplateContent(w http.ResponseWriter, r *http.Request, name string) {
	// Security: validate name doesn't contain path traversal
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		writeError(w, r, http.StatusBadRequest, "invalid_name", "Invalid template name", nil)
		return
	}

	// Search for template using recursive search
	foundPath := templates.Find(name)

	if foundPath == "" {
		writeError(w, r, http.StatusNotFound, "not_found", "Template not found: "+name, nil)
		return
	}

	content, err := os.ReadFile(filepath.Clean(foundPath))
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "read_error", "Failed to read template", nil)
		return
	}

	response := templates.Content{
		Name:    name,
		Content: string(content),
		Format:  "yaml",
	}

	s.writeJSON(w, response)
}

// handleTemplateUse handles POST /api/v1/templates/use.
func (s *Server) handleTemplateUse(w http.ResponseWriter, r *http.Request) {
	var req UseTemplateRequest
	if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
		return
	}
	if req.TemplateName == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "templateName is required", nil)
		return
	}

	content, _, err := templates.Load(req.TemplateName)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", err.Error(), nil)
		return
	}

	cfg, err := config.LoadYAMLBytes(content)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "invalid_template",
			"Template contains an invalid configuration", nil)
		return
	}
	if !s.authorizeConfigEntitlements(w, r, cfg) {
		return
	}

	configPath, err := templates.SaveConfig(req.TemplateName, req.NewConfigName, content)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "write_error", err.Error(), nil)
		return
	}

	response := UseTemplateResponse{
		Success:    true,
		Message:    fmt.Sprintf("Configuration created from template '%s'", req.TemplateName),
		ConfigPath: configPath,
	}

	s.writeJSON(w, response)
}
