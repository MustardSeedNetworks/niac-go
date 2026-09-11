package api

import (
	"errors"
	"net/http"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// Checkpoints are the seam an acceptance run resets against: save a healthy
// scenario, inject a fault, assert the consumer sees it, restore, assert it is
// gone. Without an HTTP surface that sequence could only be driven from inside
// the process, so nothing could exercise the shipped binary.
//
// Restoring does not rewind consumed timeline actions (owner decision
// 2026-09-10); a full session restart remains a separate operation.

// checkpointRequest names the checkpoint a save or restore acts on.
type checkpointRequest struct {
	Name string `json:"name"`
}

func (s *Server) handleSessionCheckpoints(
	w http.ResponseWriter, r *http.Request, session sessionRuntime,
) {
	switch r.Method {
	case http.MethodGet:
		stack, ok := s.checkpointStack(w, r, session)
		if !ok {
			return
		}
		s.writeJSON(w, map[string]any{
			"sessionId":   session.id,
			"checkpoints": stack.CheckpointNames(),
		})
	case http.MethodPost:
		s.saveSessionCheckpoint(w, r, session)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed",
			"Method not allowed", nil)
	}
}

func (s *Server) handleSessionCheckpointRestore(
	w http.ResponseWriter, r *http.Request, session sessionRuntime,
) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed",
			"Method not allowed", nil)
		return
	}
	stack, name, ok := s.checkpointTarget(w, r, session)
	if !ok {
		return
	}

	if err := stack.RestoreCheckpoint(name); err != nil {
		if errors.Is(err, devicestate.ErrCheckpointNotFound) {
			writeError(w, r, http.StatusNotFound, "checkpoint_not_found",
				"No checkpoint named "+name+" on every device in this session", nil)
			return
		}
		writeError(w, r, http.StatusInternalServerError, "checkpoint_restore_failed",
			"Could not restore the checkpoint", nil)
		return
	}

	s.writeJSON(w, map[string]any{
		"sessionId":  session.id,
		"checkpoint": name,
		"restored":   true,
	})
}

func (s *Server) saveSessionCheckpoint(
	w http.ResponseWriter, r *http.Request, session sessionRuntime,
) {
	stack, name, ok := s.checkpointTarget(w, r, session)
	if !ok {
		return
	}

	devices := stack.SaveCheckpoint(name)

	w.WriteHeader(http.StatusCreated)
	s.writeJSON(w, map[string]any{
		"sessionId":  session.id,
		"checkpoint": name,
		"devices":    devices,
	})
}

// checkpointTarget resolves the stack and the checkpoint name a mutating
// request acts on, answering the client itself when either is unusable.
func (s *Server) checkpointTarget(
	w http.ResponseWriter, r *http.Request, session sessionRuntime,
) (*protocols.Stack, string, bool) {
	stack, ok := s.checkpointStack(w, r, session)
	if !ok {
		return nil, "", false
	}
	var request checkpointRequest
	if !decodeJSONStrict(w, r, &request, MaxRequestBodySize) {
		return nil, "", false
	}
	if request.Name == "" {
		writeError(w, r, http.StatusBadRequest, "validation_failed",
			"A checkpoint name is required",
			[]ErrorDetail{{Field: "name", Issue: "must not be empty"}})
		return nil, "", false
	}

	return stack, request.Name, true
}

func (s *Server) checkpointStack(
	w http.ResponseWriter, r *http.Request, session sessionRuntime,
) (*protocols.Stack, bool) {
	stack := session.stack()
	if stack == nil {
		writeError(w, r, http.StatusConflict, "session_not_running",
			"Session "+session.id+" is not serving a scenario", nil)
		return nil, false
	}

	return stack, true
}
