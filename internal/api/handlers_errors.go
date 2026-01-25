package api

import (
	"encoding/json"
	"net"
	"net/http"

	apperr "github.com/krisarmstrong/niac-go/internal/apperr"
)

// errorInjectionRequest represents a request to inject an error.
type errorInjectionRequest struct {
	DeviceIP  string `json:"device_ip"`
	Interface string `json:"interface"`
	ErrorType string `json:"error_type"`
	Value     int    `json:"value"`
}

// getKnownErrorTypes returns the set of valid error types for injection.
func getKnownErrorTypes() map[string]bool {
	return map[string]bool{
		"FCS Errors":       true,
		"Packet Discards":  true,
		"Interface Errors": true,
		"High Utilization": true,
		"High CPU":         true,
		"High Memory":      true,
		"High Disk":        true,
	}
}

// validateErrorInjectionRequest validates the error injection request fields.
func (req *errorInjectionRequest) validate(w http.ResponseWriter, r *http.Request) bool {
	if req.DeviceIP == "" {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "device_ip is required", nil)
		return false
	}

	// Validate DeviceIP is a valid IP address
	if net.ParseIP(req.DeviceIP) == nil {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "device_ip must be a valid IP address", nil)
		return false
	}

	if req.Interface == "" {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "interface is required", nil)
		return false
	}

	if req.ErrorType == "" {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "error_type is required", nil)
		return false
	}

	// Validate ErrorType against known types
	if !getKnownErrorTypes()[req.ErrorType] {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "unknown error_type", nil)
		return false
	}

	if req.Value < 0 || req.Value > 100 {
		writeError(
			w,
			r,
			http.StatusBadRequest,
			"validation_failed",
			"value must be between 0 and 100",
			nil,
		)
		return false
	}

	return true
}

// availableErrorTypes returns the list of available error types for injection.
func availableErrorTypes() []map[string]string {
	return []map[string]string{
		{"type": "FCS Errors", "description": "Frame Check Sequence errors (0-100)"},
		{"type": "Packet Discards", "description": "Dropped packets (0-100)"},
		{"type": "Interface Errors", "description": "Generic interface errors (0-100)"},
		{"type": "High Utilization", "description": "Interface bandwidth saturation (0-100%)"},
		{"type": "High CPU", "description": "Device CPU load (0-100%)"},
		{"type": "High Memory", "description": "Device memory usage (0-100%)"},
		{"type": "High Disk", "description": "Device disk usage (0-100%)"},
	}
}

// handleErrorsRouter routes /api/v1/errors - GET is read-only, mutations need write protection.
// FIX #291: Prevents write rate limiting on read-only GET /errors endpoint.
func (s *Server) handleErrorsRouter(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Read-only: no write rate limit or CSRF needed
		s.handleErrors(w, r)

		return
	}

	// Mutating methods: apply write rate limit and CSRF protection
	s.writeRateLimit(s.csrfProtect(s.handleErrors))(w, r)
}

func (s *Server) handleErrors(w http.ResponseWriter, r *http.Request) {
	s.configMu.RLock()
	stack := s.cfg.Stack
	s.configMu.RUnlock()

	if stack == nil {
		http.Error(w, "no simulation running", http.StatusServiceUnavailable)
		return
	}

	errorMgr := stack.GetErrorManager()
	if errorMgr == nil {
		http.Error(w, "error manager not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.writeJSON(w, map[string]any{
			"available_types": availableErrorTypes(),
			"active_errors":   errorMgr.GetAllStates(),
		})

	case http.MethodPost, http.MethodPut:
		s.handleErrorInjection(w, r, errorMgr)

	case http.MethodDelete:
		s.handleErrorClear(w, r, errorMgr)

	default:
		w.Header().Set("Allow", "GET, POST, PUT, DELETE")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleErrorInjection handles POST/PUT requests to inject errors.
func (s *Server) handleErrorInjection(
	w http.ResponseWriter,
	r *http.Request,
	errorMgr *apperr.StateManager,
) {
	// SECURITY FIX #111: Enforce request body size limit
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

	var req errorInjectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// SECURITY FIX MEDIUM-6: Don't expose internal error details
		writeError(
			w,
			r,
			http.StatusBadRequest,
			"invalid_request",
			"Failed to parse request body",
			nil,
		)
		return
	}

	if !req.validate(w, r) {
		return
	}

	errorMgr.SetError(req.DeviceIP, req.Interface, apperr.ErrorType(req.ErrorType), req.Value)

	s.writeJSON(w, map[string]any{
		"success":    true,
		"message":    "error injected successfully",
		"device_ip":  req.DeviceIP,
		"interface":  req.Interface,
		"error_type": req.ErrorType,
		"value":      req.Value,
	})
}

// handleErrorClear handles DELETE requests to clear errors.
func (s *Server) handleErrorClear(
	w http.ResponseWriter,
	r *http.Request,
	errorMgr *apperr.StateManager,
) {
	query := r.URL.Query()
	deviceIP := query.Get("device_ip")
	iface := query.Get("interface")

	switch {
	case deviceIP == "" && iface == "":
		errorMgr.ClearAll()
		s.writeJSON(w, map[string]any{"success": true, "message": "all errors cleared"})
	case deviceIP != "" && iface != "":
		errorMgr.ClearError(deviceIP, iface)
		s.writeJSON(
			w,
			map[string]any{
				"success":   true,
				"message":   "error cleared",
				"device_ip": deviceIP,
				"interface": iface,
			},
		)
	default:
		http.Error(
			w,
			"both device_ip and interface are required, or omit both to clear all",
			http.StatusBadRequest,
		)
	}
}
