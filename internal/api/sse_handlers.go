// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"encoding/json"
	"net/http"

	"github.com/MustardSeedNetworks/niac-go/internal/api/sse"
)

// handleSSEPackets streams live capture packets.
func (s *Server) handleSSEPackets(w http.ResponseWriter, r *http.Request) {
	sse.Serve(s.sseHub, sse.StreamPackets, w, r, s.logger, simpleErr)
}

// handleSSELogs streams server logs.
func (s *Server) handleSSELogs(w http.ResponseWriter, r *http.Request) {
	sse.Serve(s.sseHub, sse.StreamLogs, w, r, s.logger, simpleErr)
}

// handleSSEStats streams periodic stats.
func (s *Server) handleSSEStats(w http.ResponseWriter, r *http.Request) {
	sse.Serve(s.sseHub, sse.StreamStats, w, r, s.logger, simpleErr)
}

// handleSSEStatus returns SSE hub status: running state, per-stream client
// counts, and the configured caps. Used by the SSE Status page in the UI to
// confirm the hub is healthy and to surface client churn.
func (s *Server) handleSSEStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
		return
	}

	var status map[string]any
	if s.sseHub != nil {
		st := s.sseHub.Status()
		status = map[string]any{"running": st.Running, "clients": st.Clients, "limits": st.Limits}
	} else {
		status = map[string]any{
			"running": false,
			"clients": map[string]int{"packets": 0, "logs": 0, "stats": 0, "total": 0},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status) // HTTP write errors are non-critical
}
