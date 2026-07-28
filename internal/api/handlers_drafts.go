package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/api/templates"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

type draftCreateRequest struct {
	Name         string `json:"name"`
	Content      string `json:"content"`
	TemplateName string `json:"template_name"`
}

type draftReplaceRequest struct {
	Content string `json:"content"`
}

func (s *Server) handleLibraryDrafts(w http.ResponseWriter, r *http.Request) {
	if !s.libraryReady() {
		s.writeLibraryUnavailable(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")

	switch r.Method {
	case http.MethodGet:
		s.handleLibraryDraftList(w, r)
	case http.MethodPost:
		s.handleLibraryDraftCreate(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
	}
}

func (s *Server) handleLibraryDraftByName(w http.ResponseWriter, r *http.Request) {
	if !s.libraryReady() {
		s.writeLibraryUnavailable(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/library/drafts/")
	if name == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Draft name required", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleLibraryDraftRead(w, r, name)
	case http.MethodPut:
		s.handleLibraryDraftReplace(w, r, name)
	case http.MethodDelete:
		s.handleLibraryDraftDelete(w, r, name)
	default:
		w.Header().Set("Allow", "GET, PUT, DELETE")
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
	}
}

func (s *Server) handleLibraryDraftList(w http.ResponseWriter, r *http.Request) {
	entries, err := s.library.ListDrafts()
	if err != nil {
		s.logger.ErrorContext(r.Context(), "[API] library: list drafts", "error", err)
		writeError(w, r, http.StatusInternalServerError, "draft_list_failed", "Failed to list drafts", nil)
		return
	}
	s.writeJSON(w, entries)
}

func (s *Server) handleLibraryDraftCreate(w http.ResponseWriter, r *http.Request) {
	var req draftCreateRequest
	if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
		return
	}
	content, ok := s.prepareDraftContent(w, r, req)
	if !ok {
		return
	}

	draft, err := s.library.CreateDraft(req.Name, content)
	if err != nil {
		s.writeDraftStoreError(w, r, "create", req.Name, err)
		return
	}
	w.Header().Set("Location", "/api/v1/library/drafts/"+draft.Name)
	s.writeDraft(w, http.StatusCreated, draft)
}

func (s *Server) prepareDraftContent(
	w http.ResponseWriter, r *http.Request, req draftCreateRequest,
) (string, bool) {
	hasContent := strings.TrimSpace(req.Content) != ""
	hasTemplate := strings.TrimSpace(req.TemplateName) != ""
	if hasContent == hasTemplate {
		writeError(w, r, http.StatusBadRequest, "validation_failed",
			"Exactly one of content or templateName is required", nil)
		return "", false
	}
	if hasContent {
		return req.Content, s.validateDraftContent(w, r, req.Content)
	}

	_, templatePath, err := templates.Load(req.TemplateName)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Template not found", nil)
		return "", false
	}
	cfg, _, err := config.LoadYAMLManaged(templatePath, templates.Dirs())
	if err != nil {
		s.logger.ErrorContext(r.Context(), "[API] Draft template validation failed", "error", err)
		writeError(w, r, http.StatusBadRequest, "config_invalid", "Template configuration is invalid", nil)
		return "", false
	}
	if !s.authorizeConfigEntitlements(w, r, cfg) {
		return "", false
	}

	// Loading resolves include_path and every SNMP resource to absolute paths.
	// Preserve that resolved base when moving the YAML into draft storage so
	// inline preflight can enforce that each resource remains inside it.
	if cfg.CapturePlayback != nil && !filepath.IsAbs(cfg.CapturePlayback.FileName) {
		cfg.CapturePlayback.FileName = filepath.Join(filepath.Dir(templatePath), cfg.CapturePlayback.FileName)
	}
	materialized, err := config.MarshalConfigYAML(cfg)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "[API] Draft template materialization failed", "error", err)
		writeError(w, r, http.StatusInternalServerError, "draft_materialization_failed",
			"Failed to prepare template draft", nil)
		return "", false
	}
	return string(materialized), true
}

func (s *Server) handleLibraryDraftRead(w http.ResponseWriter, r *http.Request, name string) {
	draft, err := s.library.ReadDraft(name)
	if err != nil {
		s.writeDraftStoreError(w, r, "read", name, err)
		return
	}
	s.writeDraft(w, http.StatusOK, draft)
}

func (s *Server) handleLibraryDraftReplace(w http.ResponseWriter, r *http.Request, name string) {
	revision, ok := requireDraftIfMatch(w, r)
	if !ok {
		return
	}
	var req draftReplaceRequest
	if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
		return
	}
	if !s.validateDraftContent(w, r, req.Content) {
		return
	}

	draft, err := s.library.ReplaceDraft(name, revision, req.Content)
	if err != nil {
		s.writeDraftStoreError(w, r, "replace", name, err)
		return
	}
	s.writeDraft(w, http.StatusOK, draft)
}

func (s *Server) handleLibraryDraftDelete(w http.ResponseWriter, r *http.Request, name string) {
	revision, ok := requireDraftIfMatch(w, r)
	if !ok {
		return
	}
	if err := s.library.DeleteDraft(name, revision); err != nil {
		s.writeDraftStoreError(w, r, "delete", name, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) validateDraftContent(w http.ResponseWriter, r *http.Request, content string) bool {
	if strings.TrimSpace(content) == "" {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "Draft content is required", nil)
		return false
	}
	cfg, ok := s.validateConfigContent(w, r, content)
	return ok && s.authorizeConfigEntitlements(w, r, cfg)
}

func quoteDraftETag(revision string) string {
	return `"` + revision + `"`
}

func requireDraftIfMatch(w http.ResponseWriter, r *http.Request) (string, bool) {
	raw := strings.TrimSpace(r.Header.Get("If-Match"))
	if raw == "" {
		writeError(w, r, http.StatusPreconditionRequired, "revision_required",
			"If-Match with the current draft ETag is required", nil)
		return "", false
	}
	if len(raw) < 3 || raw[0] != '"' || raw[len(raw)-1] != '"' || strings.Contains(raw[1:len(raw)-1], `"`) {
		writeError(w, r, http.StatusBadRequest, "invalid_revision", "If-Match must contain one strong ETag", nil)
		return "", false
	}
	return raw[1 : len(raw)-1], true
}

func (s *Server) writeDraft(w http.ResponseWriter, status int, draft *library.Draft) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("ETag", quoteDraftETag(draft.Revision))
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(draft)
}

func (s *Server) writeDraftStoreError(
	w http.ResponseWriter,
	r *http.Request,
	operation string,
	name string,
	err error,
) {
	switch {
	case errors.Is(err, library.ErrInvalidName):
		writeError(w, r, http.StatusBadRequest, "invalid_name", "Draft name is invalid", nil)
	case errors.Is(err, library.ErrEmptyContent):
		writeError(w, r, http.StatusBadRequest, "validation_failed", "Draft content is required", nil)
	case errors.Is(err, library.ErrAlreadyExists):
		writeError(w, r, http.StatusConflict, "draft_exists", "Draft already exists", nil)
	case errors.Is(err, library.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "not_found", "Draft not found", nil)
	case errors.Is(err, library.ErrRevisionMismatch):
		writeError(w, r, http.StatusPreconditionFailed, "revision_mismatch",
			"Draft changed since it was read", nil)
	default:
		s.logger.ErrorContext(r.Context(), "[API] library: draft operation failed",
			"operation", operation, "name", name, "error", err)
		writeError(w, r, http.StatusInternalServerError, "draft_operation_failed",
			"Draft operation failed", nil)
	}
}
