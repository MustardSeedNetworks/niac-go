package api

import (
	"errors"
	"net/http"
)

// CaptureRequest starts a standalone packet capture session that runs
// independently of any simulation. The Packet Capture page in the UI
// uses this when the user wants to sniff an interface without booting
// the protocol stack on it.
type CaptureRequest struct {
	Interface string `json:"interface"`
	// Filter is an optional libpcap BPF expression. Applied after the
	// capture handle opens. Empty means "no filter — capture everything".
	Filter string `json:"filter,omitempty"`
}

// CaptureStatus is the response body for the GET /api/v1/capture
// endpoint and is also returned from POST on success.
type CaptureStatus struct {
	Running   bool   `json:"running"`
	Interface string `json:"interface,omitempty"`
	Filter    string `json:"filter,omitempty"`
	StartedAt string `json:"startedAt,omitempty"`
	// Packets is the running count of packets the capture has observed
	// since it started. Updated by the capture loop.
	Packets uint64 `json:"packets"`
}

// CaptureController is the daemon-side surface the API server uses to
// drive standalone capture. Kept separate from DaemonController so the
// existing simulation-only flows keep their tight interface.
type CaptureController interface {
	StartCapture(req CaptureRequest) error
	StopCapture() error
	GetCaptureStatus() CaptureStatus
}

// SetCaptureController wires a CaptureController into the server. Like
// SetDaemonController this is called by the daemon at startup; in tests
// the controller is nil and the endpoints return 501.
func (s *Server) SetCaptureController(c CaptureController) {
	s.captureController = c
}

// handleStandaloneCapture routes the /api/v1/capture lifecycle.
//
//	GET    → current status
//	POST   → start (idempotent: returns 409 if already running)
//	DELETE → stop  (idempotent: returns 200 even when not running)
func (s *Server) handleStandaloneCapture(w http.ResponseWriter, r *http.Request) {
	if s.captureController == nil {
		writeError(w, r, http.StatusNotImplemented, "capture_unavailable",
			"Standalone packet capture is only available in daemon mode.", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.writeJSON(w, s.captureController.GetCaptureStatus())
	case http.MethodPost:
		s.handleStandaloneCaptureStart(w, r)
	case http.MethodDelete:
		s.handleStandaloneCaptureStop(w, r)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed",
			"Method not allowed", nil)
	}
}

func (s *Server) handleStandaloneCaptureStart(w http.ResponseWriter, r *http.Request) {
	var req CaptureRequest
	if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
		return
	}

	if req.Interface == "" {
		writeError(w, r, http.StatusBadRequest, "validation_failed",
			"Validation failed",
			[]ErrorDetail{{Field: "interface", Issue: "interface is required"}})
		return
	}

	if err := s.captureController.StartCapture(req); err != nil {
		// Surface conflict-with-sim and not-found errors with their
		// natural status codes; everything else is a 500. The detail
		// string is the daemon's error message, which is operator-
		// readable (it's already used in the sim flow's daemon log).
		s.logger.ErrorContext(r.Context(), "[API] Failed to start standalone capture", "error", err)
		switch {
		case errors.Is(err, ErrCaptureAlreadyRunning):
			writeError(w, r, http.StatusConflict, "capture_already_running",
				err.Error(), nil)
		case errors.Is(err, ErrCaptureConflictsWithSim):
			writeError(w, r, http.StatusConflict, "capture_conflicts_with_sim",
				err.Error(), nil)
		case errors.Is(err, ErrCaptureInterfaceNotFound):
			writeError(w, r, http.StatusNotFound, "interface_not_found",
				err.Error(), nil)
		default:
			writeError(w, r, http.StatusInternalServerError, "capture_start_failed",
				"Failed to start capture", nil)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	s.writeJSON(w, s.captureController.GetCaptureStatus())
}

func (s *Server) handleStandaloneCaptureStop(w http.ResponseWriter, r *http.Request) {
	if err := s.captureController.StopCapture(); err != nil {
		s.logger.ErrorContext(r.Context(), "[API] Failed to stop standalone capture", "error", err)
		writeError(w, r, http.StatusInternalServerError, "capture_stop_failed",
			"Failed to stop capture", nil)
		return
	}
	s.writeJSON(w, map[string]string{"status": "stopped"})
}

// Capture-control sentinel errors. Exported so the daemon can wrap them
// (errors.Is checks downstream) and so test code can match without
// scraping message strings.
var (
	ErrCaptureAlreadyRunning    = errors.New("standalone capture already running")
	ErrCaptureConflictsWithSim  = errors.New("cannot start standalone capture while a simulation is running")
	ErrCaptureInterfaceNotFound = errors.New("capture interface not found")
)
