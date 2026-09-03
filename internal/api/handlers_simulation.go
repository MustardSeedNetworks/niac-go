package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

var (
	// ErrSimulationDeviceLimitExceeded indicates that a config exceeds NIAC's absolute device ceiling.
	ErrSimulationDeviceLimitExceeded = errors.New("simulation exceeds the absolute device limit")
	// ErrSimulationSessionConflict indicates the session targets a physical
	// VLAN or interface binding another active session already holds.
	ErrSimulationSessionConflict = errors.New(
		"simulation session conflicts with an active physical binding",
	)
	// ErrSimulationSessionIDRequired means a session operation was called with an empty session ID.
	ErrSimulationSessionIDRequired = errors.New("simulation session ID is required")
	// ErrSimulationSessionNotFound means the named session ID has no registered simulation state.
	ErrSimulationSessionNotFound = errors.New("simulation session was not found")
	// ErrSimulationSessionCapacity and ErrSimulationDeviceCapacity are daemon-wide
	// technical capacity limits, not entitlements. A per-config check cannot
	// enforce them because several sessions run at once.
	ErrSimulationSessionCapacity = errors.New("daemon session capacity exceeded")
	// ErrSimulationDeviceCapacity is documented alongside ErrSimulationSessionCapacity above.
	ErrSimulationDeviceCapacity = errors.New("daemon device capacity exceeded")
)

// DaemonCapacity reports what the daemon is currently carrying against its
// aggregate safety budgets, so an operator can see how close a start is to
// being refused before it is.
type DaemonCapacity struct {
	Sessions    int `json:"sessions"`
	MaxSessions int `json:"maxSessions"`
	Devices     int `json:"devices"`
	MaxDevices  int `json:"maxDevices"`
}

// ValidateConfigDeviceCount enforces NIAC's absolute per-config device ceiling,
// used by both simulation starts and whole-config replacement. It is a
// technical limit on what one config may carry, not an entitlement; the
// daemon-wide budgets in internal/daemon/admission.go bound everything running
// at once.
func ValidateConfigDeviceCount(cfg *config.Config) error {
	if cfg.DeviceCount() > MaxDeviceCount {
		return ErrSimulationDeviceLimitExceeded
	}
	return nil
}

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
	case http.MethodPut:
		s.handleSimulationSelect(w, r)
	case http.MethodDelete:
		s.handleSimulationStop(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSimulationSelect(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID string `json:"sessionId"`
	}
	if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
		return
	}
	if !ValidSessionID(req.SessionID) {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "Scenario ID is invalid", nil)
		return
	}
	if err := s.daemon.SelectSimulation(req.SessionID); err != nil {
		writeError(w, r, http.StatusNotFound, "simulation_session_not_found",
			"The selected scenario is not running", nil)
		return
	}
	s.writeJSON(w, s.daemon.GetStatus())
}

// handleSimulationPreflight compiles a simulation request without changing runtime state.
func (s *Server) handleSimulationPreflight(w http.ResponseWriter, r *http.Request) {
	if s.daemon == nil {
		writeError(
			w,
			r,
			http.StatusNotImplemented,
			"daemon_required",
			"Daemon mode is required",
			nil,
		)
		return
	}
	var req SimulationRequest
	if !decodeJSONStrict(w, r, &req, MaxScenarioRequestBodySize) {
		return
	}
	if validationErrors := validateSimulationForPreflight(req); len(validationErrors) > 0 {
		writeError(
			w,
			r,
			http.StatusBadRequest,
			"validation_failed",
			"Validation failed",
			validationErrors,
		)
		return
	}
	report, err := s.daemon.PreflightSimulation(req)
	if err != nil {
		if writeManagedConfigPathError(w, r, err) {
			return
		}
		s.logger.ErrorContext(r.Context(), "[API] Simulation preflight failed", "error", err)
		// Return what went wrong, not just that something did. The daemon
		// already knows ("SNMPv1/v2c requires an explicit community"), and the
		// CLI prints it; collapsing it to a fixed string left the one button
		// whose whole job is diagnosis unable to diagnose anything (D5).
		writeError(
			w,
			r,
			http.StatusBadRequest,
			"preflight_failed",
			"Simulation preflight failed",
			preflightErrorDetails(err),
		)
		return
	}
	s.writeJSON(w, report)
}

// preflightErrorDetails turns a preflight error into per-issue details so the
// client can show the operator what to fix. Semantic validation aggregates its
// findings into one error separated by newlines or "; ", so split on both.
func preflightErrorDetails(err error) []ErrorDetail {
	if err == nil {
		return nil
	}
	// A validation failure carries a structured error per finding, complete with
	// the offending field path. ListError.Error() collapses those to a bare
	// count once there is more than one, so unwrap before falling back to text.
	var listErr *config.ListError
	if errors.As(err, &listErr) && len(listErr.Errors) > 0 {
		return validationErrorDetails(listErr)
	}
	raw := strings.NewReplacer("\n", "; ").Replace(err.Error())
	var details []ErrorDetail
	for part := range strings.SplitSeq(raw, "; ") {
		issue := strings.TrimSpace(part)
		if issue == "" {
			continue
		}
		details = append(details, ErrorDetail{Issue: issue})
	}
	if len(details) == 0 {
		return nil
	}

	return details
}

// validationErrorDetails maps each semantic-validation finding onto a detail,
// keeping the field path and source position the validator recorded.
func validationErrorDetails(listErr *config.ListError) []ErrorDetail {
	details := make([]ErrorDetail, 0, len(listErr.Errors))
	for _, e := range listErr.Errors {
		details = append(details, ErrorDetail{
			Field:  e.Field,
			Issue:  e.Message,
			Line:   e.Line,
			Column: e.Column,
		})
	}

	return details
}

// handleSimulationStart processes POST requests to start a simulation.
func (s *Server) handleSimulationStart(w http.ResponseWriter, r *http.Request) {
	var req SimulationRequest
	if !decodeJSONStrict(w, r, &req, MaxScenarioRequestBodySize) {
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
		s.handleSimulationStartError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	s.writeJSON(w, s.daemon.GetStatus())
}

func (s *Server) handleSimulationStartError(w http.ResponseWriter, r *http.Request, err error) {
	var (
		listErr     *config.ListError
		topologyErr *fabric.UnsafeTopologyError
	)
	switch {
	case errors.Is(err, ErrSimulationDeviceLimitExceeded):
		writeError(w, r, http.StatusBadRequest, "device_limit_reached",
			"Simulation exceeds the maximum supported device count", nil)
	case errors.Is(err, ErrSimulationSessionConflict):
		writeError(
			w,
			r,
			http.StatusConflict,
			"simulation_session_conflict",
			"The selected interface or physical VLAN is already assigned to another active scenario",
			nil,
		)
	case errors.Is(err, ErrSimulationSessionIDRequired):
		writeError(w, r, http.StatusBadRequest, "validation_failed",
			"A scenario ID is required for a shared trunk", nil)
	case errors.Is(err, ErrSimulationSessionCapacity):
		writeError(w, r, http.StatusConflict, "session_capacity_reached",
			"The daemon is already running its maximum number of scenarios", nil)
	case errors.Is(err, ErrSimulationDeviceCapacity):
		writeError(w, r, http.StatusConflict, "device_capacity_reached",
			"Running this scenario would exceed the daemon's total device capacity", nil)
	case errors.Is(err, config.ErrSSHPasswordUnavailable):
		writeError(w, r, http.StatusBadRequest, "runtime_requirements_unmet",
			"Configuration runtime requirements are not met",
			[]ErrorDetail{{Field: "ssh.passwordEnv", Issue: err.Error()}})
	case writeManagedConfigPathError(w, r, err):
	case errors.As(err, &listErr):
		// Semantic validation already carries a finding per field. Collapsing
		// it to a 500 left the operator with nothing to fix.
		writeError(w, r, http.StatusBadRequest, "validation_failed", "Validation failed",
			validationErrorDetails(listErr))
	case errors.As(err, &topologyErr):
		// Preflight answers this exact config with this exact list; start
		// answered "Failed to start simulation" and a 500 (P1b-4).
		writeError(w, r, http.StatusBadRequest, "preflight_failed",
			"Simulation preflight failed", diagnosticDetails(topologyErr.Diagnostics))
	default:
		// The error may include configuration-derived secrets. Record a stable
		// failure code without writing the error text to logs or the response.
		s.logger.ErrorContext(
			r.Context(),
			"[API] Failed to start simulation",
			"error_code",
			"simulation_start_failed",
		)
		writeError(w, r, http.StatusInternalServerError, "simulation_start_failed",
			"Failed to start simulation", nil)
	}
}

func writeManagedConfigPathError(w http.ResponseWriter, r *http.Request, err error) bool {
	if !errors.Is(err, config.ErrPathOutsideManagedRoots) {
		return false
	}
	writeError(
		w,
		r,
		http.StatusBadRequest,
		"validation_failed",
		"Validation failed",
		[]ErrorDetail{{
			Field: "config_path", Issue: "configuration must be selected from NIAC-managed storage",
			Value: "[redacted]",
		}},
	)
	return true
}

// handleSimulationStop processes DELETE requests to stop a simulation.
func (s *Server) handleSimulationStop(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionId")
	if sessionID != "" && !ValidSessionID(sessionID) {
		writeError(
			w,
			r,
			http.StatusBadRequest,
			"validation_failed",
			"Validation failed",
			[]ErrorDetail{{
				Field: "sessionId", Issue: "session ID is invalid", Value: sessionID,
			}},
		)
		return
	}
	s.stopSessionByID(w, r, sessionID)
}

// stopSessionByID stops one session, or every session when sessionID is empty.
// It backs both spellings of stop: DELETE /api/v1/simulation?sessionId=<id>
// and DELETE /api/v1/sessions/<id>. Callers validate the ID first.
func (s *Server) stopSessionByID(w http.ResponseWriter, r *http.Request, sessionID string) {
	if s.daemon == nil {
		writeError(w, r, http.StatusNotImplemented, "daemon_required",
			"Simulation control is only available in daemon mode. "+
				"Start NIAC with the 'niac daemon' command.", nil)
		return
	}
	if err := s.daemon.StopSimulation(sessionID); err != nil {
		if errors.Is(err, ErrSimulationSessionNotFound) {
			writeError(w, r, http.StatusNotFound, "simulation_session_not_found",
				"The selected scenario is not running", nil)
			return
		}
		// SECURITY FIX MEDIUM-6: Don't expose internal error details
		s.logger.ErrorContext(r.Context(), "[API] Failed to stop simulation", "error", err)
		writeError(w, r, http.StatusInternalServerError, "simulation_stop_failed",
			"Failed to stop simulation", nil)
		return
	}
	s.writeJSON(w, map[string]string{"status": "stopped"})
}
