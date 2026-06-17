package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/api/templates"
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

	// Handle /api/v1/templates/use separately. Applying a template is
	// gated behind the config_templates feature; listing / reading /
	// deleting templates stays open.
	if name == "use" {
		if s.license != nil && !s.license.HasFeature("config_templates") {
			s.writeFeatureGate(w, r, "config_templates",
				"Applying a config template requires the Pro tier. "+
					"Start a 14-day Pro trial with `niac license trial`.")
			return
		}
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
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
		return
	}

	req, err := parseUseTemplateRequest(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}

	content, _, err := templates.Load(req.TemplateName)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", err.Error(), nil)
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
