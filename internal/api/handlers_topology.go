package api

import (
	"fmt"
	"net/http"
)

func (s *Server) handleTopology(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, s.currentTopology())
}

func (s *Server) handleTopologyExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	// SECURITY FIX MEDIUM-3: Validate format parameter
	allowedFormats := []string{"json", "graphml", "dot"}
	if err := validateQueryParam("format", format, allowedFormats); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_parameter",
			"Invalid format parameter", []ErrorDetail{*err})

		return
	}

	topology := s.currentTopology()

	// Note: format is validated above, so only json/graphml/dot can reach here
	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=\"topology.json\"")
		s.writeJSON(w, topology)

	case "graphml":
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("Content-Disposition", "attachment; filename=\"topology.graphml\"")
		_, _ = fmt.Fprint(w, topology.ExportGraphML())

	case "dot":
		w.Header().Set("Content-Type", "text/vnd.graphviz")
		w.Header().Set("Content-Disposition", "attachment; filename=\"topology.dot\"")
		_, _ = fmt.Fprint(w, topology.ExportDOT())
	}
}

func (s *Server) currentTopology() Topology {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	return s.cfg.Topology
}
