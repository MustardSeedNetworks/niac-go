package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/krisarmstrong/niac-go/internal/library"
)

// libraryReady returns true if the on-disk library opened cleanly; the
// handlers below short-circuit with 503 when it didn't, leaving the
// rest of the API serviceable.
func (s *Server) libraryReady() bool { return s.library != nil }

func (s *Server) writeLibraryUnavailable(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusServiceUnavailable, "library_unavailable",
		"Content library failed to open at startup", nil)
}

// handleLibraryNetworks handles GET (list) and POST (upload) on
// /api/v1/library/networks.
func (s *Server) handleLibraryNetworks(w http.ResponseWriter, r *http.Request) {
	if !s.libraryReady() {
		s.writeLibraryUnavailable(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleLibraryNetworksList(w, r)
	case http.MethodPost:
		s.handleLibraryNetworkUpload(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
	}
}

// handleLibraryNetworkByName handles GET (read) and DELETE on
// /api/v1/library/networks/{name}.
func (s *Server) handleLibraryNetworkByName(w http.ResponseWriter, r *http.Request) {
	if !s.libraryReady() {
		s.writeLibraryUnavailable(w, r)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/library/networks/")
	if name == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Network name required", nil)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleLibraryNetworkRead(w, r, name)
	case http.MethodDelete:
		s.handleLibraryNetworkDelete(w, r, name)
	default:
		w.Header().Set("Allow", "GET, DELETE")
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
	}
}

func (s *Server) handleLibraryNetworksList(w http.ResponseWriter, r *http.Request) {
	entries, err := s.library.ListNetworks()
	if err != nil {
		s.logger.Error("[API] library: list networks", "error", err)
		writeError(w, r, http.StatusInternalServerError, "library_list_failed",
			"Failed to list networks", nil)
		return
	}
	s.writeJSON(w, entries)
}

func (s *Server) handleLibraryNetworkRead(w http.ResponseWriter, r *http.Request, name string) {
	doc, err := s.library.ReadNetwork(name)
	if err != nil {
		if errors.Is(err, library.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Network not found", nil)
			return
		}
		if errors.Is(err, library.ErrInvalidName) {
			writeError(w, r, http.StatusBadRequest, "invalid_name", err.Error(), nil)
			return
		}
		s.logger.Error("[API] library: read network", "name", name, "error", err)
		writeError(w, r, http.StatusInternalServerError, "library_read_failed",
			"Failed to read network", nil)
		return
	}
	s.writeJSON(w, doc)
}

type libraryNetworkUploadRequest struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func (s *Server) handleLibraryNetworkUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)
	var req libraryNetworkUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON body", nil)
		return
	}
	if err := s.library.WriteNetwork(req.Name, req.Content); err != nil {
		if errors.Is(err, library.ErrInvalidName) {
			writeError(w, r, http.StatusBadRequest, "invalid_name", err.Error(), nil)
			return
		}
		if errors.Is(err, library.ErrEmptyContent) {
			writeError(w, r, http.StatusBadRequest, "invalid_content", err.Error(), nil)
			return
		}
		s.logger.Error("[API] library: write network", "name", req.Name, "error", err)
		writeError(w, r, http.StatusInternalServerError, "library_write_failed",
			"Failed to save network", nil)
		return
	}
	w.WriteHeader(http.StatusCreated)
	s.writeJSON(w, map[string]any{
		"success": true,
		"name":    req.Name,
	})
}

func (s *Server) handleLibraryNetworkDelete(w http.ResponseWriter, r *http.Request, name string) {
	if err := s.library.DeleteNetwork(name); err != nil {
		if errors.Is(err, library.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Network not found", nil)
			return
		}
		if errors.Is(err, library.ErrInvalidName) {
			writeError(w, r, http.StatusBadRequest, "invalid_name", err.Error(), nil)
			return
		}
		// "starter networks cannot be deleted" — surface as 400 with a
		// human-readable message rather than a 500, since it's the
		// caller asking us to do something we're not going to do.
		writeError(w, r, http.StatusBadRequest, "delete_refused", err.Error(), nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleLibraryFiles serves GET on /api/v1/library/{walks,pcaps},
// returning a flat list of file entries with name, size, and mtime.
// PR 3 ships the browser pages on top of this; uploads + deletes for
// walks/pcaps are deferred to a follow-up so this PR stays focused
// on the read path (which is what the picker integrations also need).
func (s *Server) handleLibraryFiles(w http.ResponseWriter, r *http.Request, kind library.Kind) {
	if !s.libraryReady() {
		s.writeLibraryUnavailable(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
		return
	}
	entries, err := s.library.ListFiles(kind)
	if err != nil {
		s.logger.Error("[API] library: list files", "kind", string(kind), "error", err)
		writeError(w, r, http.StatusInternalServerError, "library_list_failed",
			"Failed to list "+string(kind), nil)
		return
	}
	s.writeJSON(w, entries)
}

// handleLibraryWalks is the registered route handler for /api/v1/library/walks.
func (s *Server) handleLibraryWalks(w http.ResponseWriter, r *http.Request) {
	s.handleLibraryFiles(w, r, library.KindWalks)
}

// handleLibraryPcaps is the registered route handler for /api/v1/library/pcaps.
func (s *Server) handleLibraryPcaps(w http.ResponseWriter, r *http.Request) {
	s.handleLibraryFiles(w, r, library.KindPcaps)
}

// Drained body helper — kept for symmetry with the imports the upload
// endpoints (networks today, walks/pcaps in a follow-up) already use.
var _ = io.Discard
