package api

import (
	"errors"
	"net/http"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// errorInjectionRequest represents a request to inject an error.
type errorInjectionRequest struct {
	Device    string `json:"device"`
	Interface string `json:"interface"`
	ErrorType string `json:"errorType"`
	Value     int    `json:"value"`
}

// validate validates the error injection request fields.
func (req *errorInjectionRequest) validate(w http.ResponseWriter, r *http.Request) bool {
	message := req.validationMessage()
	if message == "" {
		return true
	}
	writeError(w, r, http.StatusBadRequest, "validation_failed", message, nil)
	return false
}

func (s *Server) handleErrors(w http.ResponseWriter, r *http.Request) {
	s.configMu.RLock()
	stack := s.cfg.Stack
	s.configMu.RUnlock()

	if stack == nil {
		http.Error(w, "no simulation running", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.writeJSON(w, map[string]any{
			"available_types": availableErrorTypes(),
			"active_errors":   interfaceFaultResponse(stack.ActiveInterfaceFaults()),
			"targets":         interfaceFaultTargetsResponse(stack.InterfaceFaultTargets()),
			"info":            "Fault injection updates SNMP interface counters",
		})
	case http.MethodPost, http.MethodPut:
		s.handleErrorInjection(w, r, stack)
	case http.MethodDelete:
		s.handleErrorClear(w, r, stack)
	default:
		w.Header().Set("Allow", "GET, POST, PUT, DELETE")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleErrorInjection handles POST/PUT requests to inject errors.
func (s *Server) handleErrorInjection(
	w http.ResponseWriter,
	r *http.Request,
	stack *protocols.Stack,
) {
	var req errorInjectionRequest
	if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
		return
	}

	if !req.validate(w, r) {
		return
	}

	faultType, err := parseInterfaceFaultType(req.ErrorType)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "validation_failed", err.Error(), nil)
		return
	}
	if err = stack.SetInterfaceFault(req.Device, req.Interface, faultType, req.Value); err != nil {
		writeInterfaceFaultError(w, r, err)
		return
	}

	s.writeJSON(w, map[string]any{
		"success":   true,
		"message":   "error injected successfully",
		"device":    req.Device,
		"interface": req.Interface,
		"errorType": req.ErrorType,
		"value":     req.Value,
	})
}

// handleErrorClear handles DELETE requests to clear errors.
func (s *Server) handleErrorClear(
	w http.ResponseWriter,
	r *http.Request,
	stack *protocols.Stack,
) {
	query := r.URL.Query()
	device := query.Get("device")
	iface := query.Get("interface")
	errorType := query.Get("errorType")

	switch {
	case device == "" && iface == "" && errorType == "":
		stack.ClearAllInterfaceFaults()
		s.writeJSON(w, map[string]any{"success": true, "message": "all errors cleared"})
	case device != "" && iface != "":
		var err error
		if errorType == "" {
			err = stack.ClearInterfaceFaults(device, iface)
		} else {
			var faultType devicestate.FaultType
			faultType, err = parseInterfaceFaultType(errorType)
			if err == nil {
				err = stack.SetInterfaceFault(device, iface, faultType, 0)
			}
		}
		if err != nil {
			writeInterfaceFaultError(w, r, err)
			return
		}
		s.writeJSON(
			w,
			map[string]any{
				"success":   true,
				"message":   "error cleared",
				"device":    device,
				"interface": iface,
				"errorType": errorType,
			},
		)
	default:
		http.Error(
			w,
			"device and interface are required together; errorType is optional",
			http.StatusBadRequest,
		)
	}
}

func writeInterfaceFaultError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusBadRequest
	code := "fault_invalid"
	switch {
	case errors.Is(err, protocols.ErrFaultDeviceNotFound):
		status, code = http.StatusNotFound, "device_not_found"
	case errors.Is(err, protocols.ErrFaultDeviceAmbiguous):
		status, code = http.StatusConflict, "device_ambiguous"
	case errors.Is(err, protocols.ErrFaultUnobservable):
		code = "fault_not_observable"
	case errors.Is(err, devicestate.ErrInterfaceNotFound):
		code = "interface_not_found"
	}
	writeError(w, r, status, code, err.Error(), nil)
}
