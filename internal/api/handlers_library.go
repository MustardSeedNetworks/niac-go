package api

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
	"github.com/MustardSeedNetworks/niac-go/internal/sanitize"
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
		s.logger.ErrorContext(r.Context(), "[API] library: list networks", "error", err)
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
		s.logger.ErrorContext(r.Context(), "[API] library: read network", "name", name, "error", err)
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
	var req libraryNetworkUploadRequest
	if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
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
		s.logger.ErrorContext(r.Context(), "[API] library: write network", "name", req.Name, "error", err)
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
	// Method gating (GET-only) is enforced declaratively by the route registry
	// on /api/v1/library/{walks,pcaps}.
	entries, err := s.library.ListFiles(kind)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "[API] library: list files", "kind", string(kind), "error", err)
		writeError(w, r, http.StatusInternalServerError, "library_list_failed",
			"Failed to list "+string(kind), nil)
		return
	}

	// Fall through to candidate directories when the library subdir
	// is empty. Without this, freshly-installed daemons return [] for
	// /api/v1/library/pcaps even with example pcaps right there in
	// the repo — and the Traffic page picker stays empty. Once an
	// operator drops files into ~/.niac/library/{walks,pcaps}/ the
	// fallback stops firing (library content wins).
	//
	// Candidates (in priority order):
	//   1. Sim-config-relative dir (legacy collectFiles path, only
	//      meaningful when a sim is running)
	//   2. examples/captures/ relative to cwd (bundled in the repo)
	//   3. examples/pcaps/ relative to cwd (older example layout)
	if len(entries) == 0 {
		if legacy, legacyErr := s.collectFiles(string(kind)); legacyErr == nil && len(legacy) > 0 {
			s.writeJSON(w, libraryEntriesFromLegacy(legacy))
			return
		}
		for _, candidate := range fallbackDirsForKind(kind) {
			// Walks live in vendor-keyed subdirectories
			// (examples/device_walks/{cisco,juniper,...}/walk.txt),
			// so a flat scan finds nothing. Pcaps are flat —
			// the per-kind switch in scanFnForKind picks the
			// right walker.
			scanner := scanFnForKind(kind)
			if extras := scanner(candidate, extensionsForKind(kind)); len(extras) > 0 {
				s.writeJSON(w, extras)
				return
			}
		}
	}
	s.writeJSON(w, entries)
}

// fallbackDirsForKind returns the ordered list of directories to scan
// when the library subdir for `kind` is empty. Order matters — first
// non-empty hit wins.
func fallbackDirsForKind(kind library.Kind) []string {
	switch kind {
	case library.KindPcaps:
		return []string{"examples/captures", "examples/pcaps"}
	case library.KindWalks:
		return []string{"examples/walks", "examples/device_walks_sanitized", "examples/device_walks"}
	case library.KindNetworks:
		// networks have their own bootstrap path (starter pack copies
		// into the library on first run) and the live editor / picker
		// flow doesn't need a cold-start fallback. Return nil so the
		// caller treats it as "no candidates" rather than panicking.
		return nil
	default:
		return nil
	}
}

// extensionsForKind is the per-kind file extension allow-list used by
// the candidate-dir scanner.
func extensionsForKind(kind library.Kind) []string {
	switch kind {
	case library.KindPcaps:
		return []string{".pcap", ".pcapng", ".cap"}
	case library.KindWalks:
		return []string{".walk", ".snmpwalk", ".txt"}
	case library.KindNetworks:
		// see fallbackDirsForKind — networks don't use this codepath.
		return nil
	default:
		return nil
	}
}

// scanFnForKind returns the directory scanner appropriate for `kind`.
// Walks need a recursive walker because the bundled
// examples/device_walks tree groups files by vendor subdirectory;
// a flat scan would return zero. Pcaps live in flat directories so
// the cheaper flat scan still applies.
func scanFnForKind(kind library.Kind) func(string, []string) []library.FileEntry {
	if kind == library.KindWalks {
		return scanDirRecursiveForExtensions
	}
	return scanDirForExtensions
}

// scanDirRecursiveForExtensions walks `dir` (any depth), returning
// entries whose extension matches one of `exts`. Vendor / model
// subdirectories are flattened — the returned Name is the basename
// only, matching the flat-scan contract callers already consume.
// Errors are swallowed (best-effort fallback content).
func scanDirRecursiveForExtensions(dir string, exts []string) []library.FileEntry {
	if _, err := os.Stat(dir); err != nil {
		return nil
	}
	allowed := make(map[string]bool, len(exts))
	for _, e := range exts {
		allowed[e] = true
	}
	var out []library.FileEntry
	_ = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil //nolint:nilerr // best-effort scan; skip unreadable subtrees
		}
		if !allowed[strings.ToLower(filepath.Ext(d.Name()))] {
			return nil
		}
		info, statErr := d.Info()
		if statErr != nil {
			return nil //nolint:nilerr // skip files we can't stat, keep walking
		}
		out = append(out, library.FileEntry{
			Name:       d.Name(),
			SizeBytes:  info.Size(),
			ModifiedAt: info.ModTime().UTC(),
			Source:     library.SourceUser,
		})
		return nil
	})
	return out
}

// scanDirForExtensions walks dir non-recursively, returning entries
// whose extension matches one of `exts`. Errors (missing dir, perm
// issues) are swallowed — the caller treats this as best-effort
// fallback content.
func scanDirForExtensions(dir string, exts []string) []library.FileEntry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	allowed := make(map[string]bool, len(exts))
	for _, e := range exts {
		allowed[e] = true
	}
	var out []library.FileEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !allowed[strings.ToLower(filepath.Ext(e.Name()))] {
			continue
		}
		info, statErr := e.Info()
		if statErr != nil {
			continue
		}
		out = append(out, library.FileEntry{
			Name:       e.Name(),
			SizeBytes:  info.Size(),
			ModifiedAt: info.ModTime().UTC(),
			Source:     library.SourceUser,
		})
	}
	return out
}

// libraryEntriesFromLegacy converts the legacy collectFiles result
// shape (FileEntry with path / name / sizeBytes / modifiedAt) into
// the library.FileEntry shape (name / sizeBytes / modifiedAt /
// source). The legacy entries get source="user" since they originated
// from the operator's filesystem layout, not the library bootstrap.
func libraryEntriesFromLegacy(in []FileEntry) []library.FileEntry {
	out := make([]library.FileEntry, len(in))
	for i, e := range in {
		out[i] = library.FileEntry{
			Name:       e.Name,
			SizeBytes:  e.SizeBytes,
			ModifiedAt: e.Modified,
			Source:     library.SourceUser,
		}
	}
	return out
}

// handleLibraryWalks is the registered route handler for /api/v1/library/walks.
func (s *Server) handleLibraryWalks(w http.ResponseWriter, r *http.Request) {
	s.handleLibraryFiles(w, r, library.KindWalks)
}

// handleLibraryPcaps is the registered route handler for /api/v1/library/pcaps.
func (s *Server) handleLibraryPcaps(w http.ResponseWriter, r *http.Request) {
	s.handleLibraryFiles(w, r, library.KindPcaps)
}

// libraryWalkRevertRequest is the body for POST /api/v1/library/walks/revert.
type libraryWalkRevertRequest struct {
	Name string `json:"name"`
}

// handleLibraryWalkRevert handles POST /api/v1/library/walks/revert.
// It restores a walk from its preserved pristine ".orig" copy (see
// library.PreserveOriginal) and removes the sidecar, returning the
// walk to a clean "no edits" state. Responds with the refreshed
// FileEntry (edited=false) on success.
func (s *Server) handleLibraryWalkRevert(w http.ResponseWriter, r *http.Request) {
	if !s.libraryReady() {
		s.writeLibraryUnavailable(w, r)
		return
	}
	// Method gating (POST-only) is enforced declaratively by the route registry.
	var req libraryWalkRevertRequest
	if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Walk name required", nil)
		return
	}

	if err := s.library.RevertToOriginal(library.KindWalks, name); err != nil {
		if errors.Is(err, library.ErrNoOriginal) {
			writeError(w, r, http.StatusNotFound, "no_original", "No preserved original to revert to", nil)
			return
		}
		if errors.Is(err, library.ErrInvalidName) {
			writeError(w, r, http.StatusBadRequest, "invalid_name", err.Error(), nil)
			return
		}
		s.logger.ErrorContext(r.Context(), "[API] library: revert walk", "name", name, "error", err)
		writeError(w, r, http.StatusInternalServerError, "revert_failed", "Failed to revert walk", nil)
		return
	}

	entry, err := s.library.FileEntryByName(library.KindWalks, name)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "[API] library: reload reverted walk", "name", name, "error", err)
		writeError(w, r, http.StatusInternalServerError, "library_read_failed",
			"Reverted but failed to reload the entry", nil)
		return
	}
	s.writeJSON(w, entry)
}

// maxSanitizeBatch caps a single sanitize-batch request, mirroring the
// per-request bulk-operation cap devices batch-delete uses (MaxDeviceCount).
const maxSanitizeBatch = MaxDeviceCount

// sanitizeWalkResult reports the outcome of sanitizing a single walk as
// part of a single or batch sanitize request.
type sanitizeWalkResult struct {
	Name                 string `json:"name"`
	Success              bool   `json:"success"`
	Error                string `json:"error,omitempty"`
	IPsTransformed       int    `json:"ipsTransformed,omitempty"`
	HostnamesTransformed int    `json:"hostnamesTransformed,omitempty"`
}

// sanitizeWalk preserves name's pristine original (idempotent — a no-op if
// already preserved), reads the current content, runs it through
// internal/sanitize with the CLI's default options, and overwrites the
// entry with the sanitized bytes. The original stays recoverable via
// RevertToOriginal for as long as the .orig sidecar exists.
func (s *Server) sanitizeWalk(name string) (library.FileEntry, sanitize.Stats, error) {
	if err := s.library.PreserveOriginal(library.KindWalks, name); err != nil {
		return library.FileEntry{}, sanitize.Stats{}, err
	}

	content, err := s.library.ReadFile(library.KindWalks, name)
	if err != nil {
		return library.FileEntry{}, sanitize.Stats{}, err
	}

	sanitized, stats, err := sanitize.Content(content, nil, sanitize.DefaultOptions())
	if err != nil {
		return library.FileEntry{}, sanitize.Stats{}, err
	}

	if writeErr := s.library.WriteFile(library.KindWalks, name, sanitized); writeErr != nil {
		return library.FileEntry{}, sanitize.Stats{}, writeErr
	}

	entry, err := s.library.FileEntryByName(library.KindWalks, name)
	if err != nil {
		return library.FileEntry{}, sanitize.Stats{}, err
	}
	return entry, stats, nil
}

// libraryWalkSanitizeRequest is the body for POST /api/v1/library/walks/sanitize.
type libraryWalkSanitizeRequest struct {
	Name string `json:"name"`
}

// handleLibraryWalkSanitize handles POST /api/v1/library/walks/sanitize. It
// preserves the walk's pristine original (if not already preserved) and
// overwrites the entry with a sanitized copy, responding with the refreshed
// FileEntry (edited=true) on success — mirroring handleLibraryWalkRevert's
// response shape.
func (s *Server) handleLibraryWalkSanitize(w http.ResponseWriter, r *http.Request) {
	if !s.libraryReady() {
		s.writeLibraryUnavailable(w, r)
		return
	}
	// Method gating (POST-only) is enforced declaratively by the route registry.
	var req libraryWalkSanitizeRequest
	if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Walk name required", nil)
		return
	}

	entry, _, err := s.sanitizeWalk(name)
	if err != nil {
		if errors.Is(err, library.ErrInvalidName) {
			writeError(w, r, http.StatusBadRequest, "invalid_name", err.Error(), nil)
			return
		}
		if errors.Is(err, library.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Walk not found", nil)
			return
		}
		s.logger.ErrorContext(r.Context(), "[API] library: sanitize walk", "name", name, "error", err)
		writeError(w, r, http.StatusInternalServerError, "sanitize_failed", "Failed to sanitize walk", nil)
		return
	}
	s.writeJSON(w, entry)
}

// libraryWalkSanitizeBatchRequest is the body for
// POST /api/v1/library/walks/sanitize-batch.
type libraryWalkSanitizeBatchRequest struct {
	Names []string `json:"names"`
}

// libraryWalkSanitizeBatchResponse reports per-walk outcomes rather than
// failing the whole request when some walks don't exist, mirroring
// handleDeviceBatchDelete's shape.
type libraryWalkSanitizeBatchResponse struct {
	Results   []sanitizeWalkResult `json:"results"`
	Sanitized int                  `json:"sanitized"`
	Failed    int                  `json:"failed"`
}

// handleLibraryWalkSanitizeBatch handles POST /api/v1/library/walks/sanitize-batch.
func (s *Server) handleLibraryWalkSanitizeBatch(w http.ResponseWriter, r *http.Request) {
	if !s.libraryReady() {
		s.writeLibraryUnavailable(w, r)
		return
	}
	// Method gating (POST-only) is enforced declaratively by the route registry.
	var req libraryWalkSanitizeBatchRequest
	if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
		return
	}
	if len(req.Names) == 0 {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "names must not be empty", nil)
		return
	}
	if len(req.Names) > maxSanitizeBatch {
		writeError(w, r, http.StatusBadRequest, "validation_failed",
			fmt.Sprintf("names must not exceed %d entries", maxSanitizeBatch), nil)
		return
	}

	results := make([]sanitizeWalkResult, 0, len(req.Names))
	sanitized, failed := 0, 0

	for _, rawName := range req.Names {
		name := strings.TrimSpace(rawName)
		entry, stats, err := s.sanitizeWalk(name)
		if err != nil {
			failed++
			results = append(results, sanitizeWalkResult{Name: name, Success: false, Error: err.Error()})
			continue
		}
		sanitized++
		results = append(results, sanitizeWalkResult{
			Name:                 entry.Name,
			Success:              true,
			IPsTransformed:       stats.IPsTransformed,
			HostnamesTransformed: stats.HostnamesTransformed,
		})
	}

	s.writeJSON(w, libraryWalkSanitizeBatchResponse{
		Results:   results,
		Sanitized: sanitized,
		Failed:    failed,
	})
}
