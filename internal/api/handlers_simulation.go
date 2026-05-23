package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (s *Server) handleSimulation(w http.ResponseWriter, r *http.Request) {
	if s.daemon == nil {
		http.Error(
			w,
			"Simulation control is only available in daemon mode. Start NIAC with 'niac daemon' command.",
			http.StatusNotImplemented,
		)
		return
	}

	switch r.Method {
	case http.MethodGet:
		status := s.daemon.GetStatus()
		s.writeJSON(w, status)
	case http.MethodPost:
		s.handleSimulationStart(w, r)
	case http.MethodDelete:
		s.handleSimulationStop(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSimulationStart processes POST requests to start a simulation.
func (s *Server) handleSimulationStart(w http.ResponseWriter, r *http.Request) {
	// SECURITY FIX MEDIUM-3: Request body size limit
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

	var req SimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err.Error() == ErrMsgRequestBodyTooLarge {
			writeError(w, r, http.StatusRequestEntityTooLarge, "request_too_large",
				fmt.Sprintf("Request body exceeds maximum size of %d bytes", MaxRequestBodySize), nil)
			return
		}
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Failed to parse request body", nil)
		return
	}

	if err := validateSimulationStartRequest(req); err != nil {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "Validation failed", err)
		return
	}

	if validationErrors := validateSimulationRequest(req); len(validationErrors) > 0 {
		writeError(w, r, http.StatusBadRequest, "validation_failed",
			"Simulation request validation failed", validationErrors)
		return
	}

	if err := s.daemon.StartSimulation(req); err != nil {
		// SECURITY: Don't leak internal error details to the client, but
		// log them server-side so the daemon log can be used to diagnose
		// why a start failed (config parse, capture engine, walk file…).
		s.logger.ErrorContext(r.Context(), "[API] Failed to start simulation", "error", err)
		writeError(w, r, http.StatusInternalServerError, "simulation_start_failed",
			"Failed to start simulation", nil)
		return
	}

	w.WriteHeader(http.StatusCreated)
	s.writeJSON(w, s.daemon.GetStatus())
}

// handleSimulationStop processes DELETE requests to stop a simulation.
func (s *Server) handleSimulationStop(w http.ResponseWriter, r *http.Request) {
	if err := s.daemon.StopSimulation(); err != nil {
		// SECURITY FIX MEDIUM-6: Don't expose internal error details
		s.logger.ErrorContext(r.Context(), "[API] Failed to stop simulation", "error", err)
		writeError(w, r, http.StatusInternalServerError, "simulation_stop_failed",
			"Failed to stop simulation", nil)
		return
	}
	s.writeJSON(w, map[string]string{"status": "stopped"})
}
