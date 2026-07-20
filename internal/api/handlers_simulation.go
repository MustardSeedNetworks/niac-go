package api

import (
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

// handleSimulationPreflight compiles a simulation request without changing runtime state.
func (s *Server) handleSimulationPreflight(w http.ResponseWriter, r *http.Request) {
	if s.daemon == nil {
		writeError(w, r, http.StatusNotImplemented, "daemon_required", "Daemon mode is required", nil)
		return
	}
	var req SimulationRequest
	if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
		return
	}
	if validationErrors := validateSimulationForPreflight(req); len(validationErrors) > 0 {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "Validation failed", validationErrors)
		return
	}
	report, err := s.daemon.PreflightSimulation(req)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "[API] Simulation preflight failed", "error", err)
		writeError(w, r, http.StatusBadRequest, "preflight_failed", "Simulation preflight failed", nil)
		return
	}
	s.writeJSON(w, report)
}

// handleSimulationStart processes POST requests to start a simulation.
func (s *Server) handleSimulationStart(w http.ResponseWriter, r *http.Request) {
	var req SimulationRequest
	if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
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
