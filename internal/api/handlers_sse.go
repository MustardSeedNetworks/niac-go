package api

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"
)

// sseTokenEntry represents a short-lived SSE authentication token.
// FIX #283: Used instead of exposing the main API token in URL query params.
type sseTokenEntry struct {
	expiresAt time.Time
}

// sseTokenExpiry is the validity duration for short-lived SSE tokens.
const sseTokenExpiry = 60 * time.Second

// handleCSRFToken returns the CSRF token for the client.
// FIX #293: Rotates the CSRF token if it has expired (> 1 hour).
// SECURITY FIX LOW-1: Clients must retrieve this token and include it in state-changing requests.
func (s *Server) handleCSRFToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	// FIX #293: Check if token needs rotation
	s.csrfMu.Lock()
	if time.Now().After(s.csrfExpiry) {
		newToken, err := generateCSRFToken()
		if err == nil {
			s.csrfPrevToken = s.csrfToken
			s.csrfToken = newToken
			s.csrfExpiry = time.Now().Add(1 * time.Hour)
			s.logger.Info("[API] CSRF token rotated")
		}
	}
	token := s.csrfToken
	s.csrfMu.Unlock()

	s.writeJSON(w, map[string]string{
		"token": token,
	})
}

// handleSSEToken generates a short-lived, one-time-use token for SSE connections.
// FIX #283: Prevents exposing the main API token in URL query parameters.
func (s *Server) handleSSEToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	tokenBytes := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(tokenBytes); err != nil {
		writeError(w, r, http.StatusInternalServerError, "token_generation_failed",
			"Failed to generate SSE token", nil)

		return
	}

	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Store with expiry
	s.sseTokens.Store(token, &sseTokenEntry{
		expiresAt: time.Now().Add(sseTokenExpiry),
	})

	s.writeJSON(w, map[string]string{
		"token": token,
	})
}

// trySSEToken attempts to validate a short-lived SSE token from the query string.
// Returns true if the token was valid and the request was served, false otherwise.
func (s *Server) trySSEToken(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) bool {
	queryToken := r.URL.Query().Get("token")
	if queryToken == "" {
		return false
	}

	entry, ok := s.sseTokens.LoadAndDelete(queryToken)
	if !ok {
		return false
	}

	sseEntry, entryOK := entry.(*sseTokenEntry)
	if !entryOK || !time.Now().Before(sseEntry.expiresAt) {
		return false
	}

	next(w, r)

	return true
}

// cleanupSSETokens removes expired SSE tokens from the [sync.Map].
func (s *Server) cleanupSSETokens() {
	now := time.Now()
	s.sseTokens.Range(func(key, value any) bool {
		entry, ok := value.(*sseTokenEntry)
		if !ok || !now.Before(entry.expiresAt) {
			s.sseTokens.Delete(key)
		}
		return true
	})
}
