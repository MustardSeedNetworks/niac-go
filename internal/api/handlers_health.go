package api

import "net/http"

// LivenessResponse is the body of GET /health/live.
type LivenessResponse struct {
	Status string `json:"status"`
}

// ReadinessCheck is one dependency's result inside a ReadinessResponse.
type ReadinessCheck struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// ReadinessResponse is the body of GET /health/ready.
type ReadinessResponse struct {
	Status string                    `json:"status"`
	Checks map[string]ReadinessCheck `json:"checks"`
}

// handleHealthLive is the liveness probe: the process is up and serving.
func (s *Server) handleHealthLive(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, LivenessResponse{Status: "ok"})
}

// handleHealthReady is the readiness probe: the daemon has the configuration
// and the content library it serves from. It answers 503 until both are loaded.
func (s *Server) handleHealthReady(w http.ResponseWriter, _ *http.Request) {
	checks := map[string]ReadinessCheck{
		"config":  readiness(s.currentConfig() != nil, "configuration not loaded"),
		"library": readiness(s.library != nil, "content library not opened"),
	}

	resp := ReadinessResponse{Status: "ready", Checks: checks}
	code := http.StatusOK
	for _, c := range checks {
		if c.Status != "ok" {
			resp.Status = "not_ready"
			code = http.StatusServiceUnavailable
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	s.writeJSON(w, resp)
}

func readiness(ok bool, failure string) ReadinessCheck {
	if ok {
		return ReadinessCheck{Status: "ok"}
	}
	return ReadinessCheck{Status: "error", Error: failure}
}
