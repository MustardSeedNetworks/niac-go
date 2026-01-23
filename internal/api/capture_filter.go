package api

import (
	"encoding/json"
	"net/http"
)

// CaptureFilterResponse represents the current BPF capture filter state.
type CaptureFilterResponse struct {
	Active bool   `json:"active"`
	Filter string `json:"filter"`
}

// CaptureFilterRequest represents a request to set a BPF capture filter.
type CaptureFilterRequest struct {
	Filter string `json:"filter"`
}

// handleCaptureFilter routes GET/PUT/DELETE for /api/v1/capture/filter.
func (s *Server) handleCaptureFilter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleCaptureFilterGet(w)
	case http.MethodPut:
		s.handleCaptureFilterSet(w, r)
	case http.MethodDelete:
		s.handleCaptureFilterClear(w)
	default:
		w.Header().Set("Allow", "GET, PUT, DELETE")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleCaptureFilterGet returns the current filter state.
func (s *Server) handleCaptureFilterGet(w http.ResponseWriter) {
	stack := s.currentStack()
	if stack == nil {
		s.writeJSON(w, CaptureFilterResponse{Active: false, Filter: ""})

		return
	}

	filter := stack.CaptureFilter()
	s.writeJSON(w, CaptureFilterResponse{
		Active: filter != "",
		Filter: filter,
	})
}

// handleCaptureFilterSet validates and applies a BPF filter.
func (s *Server) handleCaptureFilterSet(w http.ResponseWriter, r *http.Request) {
	stack := s.currentStack()
	if stack == nil {
		http.Error(w, "no simulation running", http.StatusServiceUnavailable)

		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

	var req CaptureFilterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request",
			"Failed to parse request body", nil)

		return
	}

	if err := stack.SetCaptureFilter(req.Filter); err != nil {
		s.logger.Error("[API] Failed to set capture filter", "filter", req.Filter, "error", err)
		writeError(w, r, http.StatusBadRequest, "invalid_filter",
			"Invalid BPF filter expression", nil)

		return
	}

	s.logger.Info("[API] Capture filter set", "filter", req.Filter)
	s.writeJSON(w, CaptureFilterResponse{
		Active: req.Filter != "",
		Filter: req.Filter,
	})
}

// handleCaptureFilterClear removes the active BPF filter.
func (s *Server) handleCaptureFilterClear(w http.ResponseWriter) {
	stack := s.currentStack()
	if stack == nil {
		http.Error(w, "no simulation running", http.StatusServiceUnavailable)

		return
	}

	if err := stack.SetCaptureFilter(""); err != nil {
		s.logger.Error("[API] Failed to clear capture filter", "error", err)
		writeError(w, nil, http.StatusInternalServerError, "filter_clear_failed",
			"Failed to clear capture filter", nil)

		return
	}

	s.logger.Info("[API] Capture filter cleared")
	s.writeJSON(w, CaptureFilterResponse{Active: false, Filter: ""})
}
