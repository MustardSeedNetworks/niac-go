package api

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
)

// registerSessionRoutes registers the session-scoped runtime surface. Both
// entries go through the shared register() path, so they pick up auth, CSRF
// and rate limiting the same way every other route does.
//
// The read surfaces beneath /api/v1/sessions/{id}/ are GET-only and enforce
// that per resource. They still carry csrf/rlWrite at the route entry because
// this subtree will also carry mutating resources (replay, capture filter,
// errors, debug) as they migrate; a GET is CSRF-exempt in practice, so the
// stricter tuple costs reads nothing and cannot be forgotten later.
func (s *Server) registerSessionRoutes(mux *http.ServeMux) {
	s.registerAll(mux, []apiRoute{
		{
			path:    "/api/v1/sessions",
			handler: s.handleSessions,
			methods: []string{http.MethodGet},
		},
		{
			path:    "/api/v1/sessions/",
			handler: s.dispatchSessionSubpath,
			methods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
			rl:      rlWrite,
			csrf:    true,
		},
	})
}

// sessionSummary is one entry in GET /api/v1/sessions.
type sessionSummary struct {
	SessionID   string `json:"sessionId"`
	Interface   string `json:"interface,omitempty"`
	ConfigPath  string `json:"configPath,omitempty"`
	DeviceCount int    `json:"deviceCount"`
}

// handleSessions lists the running sessions a client may address. Without it a
// client would have to guess session IDs to use the session-scoped routes.
func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
		return
	}
	ids := s.sessionIDs()
	slices.Sort(ids)
	summaries := make([]sessionSummary, 0, len(ids))
	for _, id := range ids {
		session, err := s.session(id)
		if err != nil {
			continue
		}
		summary := sessionSummary{
			SessionID:  id,
			Interface:  session.iface(),
			ConfigPath: session.configPath(),
		}
		if cfg := session.config(); cfg != nil {
			summary.DeviceCount = cfg.DeviceCount()
		}
		summaries = append(summaries, summary)
	}
	s.writeJSON(w, summaries)
}

// dispatchSessionSubpath routes /api/v1/sessions/{id}/{resource}. The session
// is resolved once here and passed to each handler, so no runtime handler can
// read a process-wide "selected" session.
//
// Every resource under this prefix shares one route policy entry. A resource
// needing stricter policy than its siblings must be registered as its own
// literal path instead, the way /api/v1/config/import is split from
// /api/v1/config.
func (s *Server) dispatchSessionSubpath(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/v1/sessions/")
	sessionID, resource, found := strings.Cut(rest, "/")
	if sessionID == "" {
		writeError(w, r, http.StatusNotFound, "not_found", "A session ID is required", nil)
		return
	}
	if !ValidSessionID(sessionID) {
		writeError(w, r, http.StatusBadRequest, "validation_failed",
			"Invalid session ID", []ErrorDetail{{Field: "sessionId", Issue: "invalid session ID"}})
		return
	}
	if !found || resource == "" {
		if r.Method == http.MethodDelete {
			// Resolved here, not left to the daemon, so an unknown ID answers
			// the same 404 the read resources under this prefix answer.
			if _, err := s.session(sessionID); err != nil {
				s.writeSessionLookupError(w, r, sessionID, err)
				return
			}
			s.stopSessionByID(w, r, sessionID)
			return
		}
		writeError(w, r, http.StatusNotFound, "not_found",
			"A session resource is required", nil)
		return
	}

	session, err := s.session(sessionID)
	if err != nil {
		s.writeSessionLookupError(w, r, sessionID, err)
		return
	}

	handler, ok := s.sessionResourceHandler(resource)
	if !ok {
		writeError(w, r, http.StatusNotFound, "not_found",
			fmt.Sprintf("Unknown session resource: %s", resource), nil)
		return
	}
	handler(w, r, session)
}

type sessionHandler func(http.ResponseWriter, *http.Request, sessionRuntime)

// sessionResourceHandler maps a resource segment to its handler. Adding a
// runtime surface means adding it here, which is also the list a reviewer
// checks to confirm nothing still reads global state.
func (s *Server) sessionResourceHandler(resource string) (sessionHandler, bool) {
	handlers := map[string]sessionHandler{
		"topology":   s.handleSessionTopology,
		"devices":    s.handleSessionDevices,
		"interfaces": s.handleSessionInterfaces,
		"segments":   s.handleSessionSegments,
		"neighbors":  s.handleSessionNeighbors,
		"stats":      s.handleSessionStats,
		"runtime":    s.handleSessionRuntime,
	}
	handler, ok := handlers[resource]
	return handler, ok
}

func (s *Server) writeSessionLookupError(
	w http.ResponseWriter, r *http.Request, sessionID string, err error,
) {
	if errors.Is(err, ErrSessionNotFound) {
		writeError(w, r, http.StatusNotFound, "session_not_found",
			fmt.Sprintf("No running session named %q", sessionID), nil)
		return
	}
	writeError(w, r, http.StatusInternalServerError, "session_lookup_failed",
		"Could not resolve the session", nil)
}

func requireGet(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet {
		return true
	}
	w.Header().Set("Allow", http.MethodGet)
	writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
	return false
}
