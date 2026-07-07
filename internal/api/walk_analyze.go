package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MustardSeedNetworks/niac-go/internal/walkanalysis"
)

// WalkAnalyzeRequest is the request body for walk analysis. The filename is
// resolved the same way as walk validation: a library-relative name
// ("cisco/c3900.walk") or a legacy config-dir path.
type WalkAnalyzeRequest struct {
	Filename string `json:"filename"`
}

// WalkAnalyzeResponse wraps the extracted device/interface/neighbor analysis.
type WalkAnalyzeResponse struct {
	Success bool                   `json:"success"`
	Result  *walkanalysis.Analysis `json:"result,omitempty"`
}

// handleWalkAnalyze parses an SNMP walk into device identity, interface
// inventory, and LLDP/CDP neighbors. Route: POST /api/v1/walk/analyze.
func (s *Server) handleWalkAnalyze(w http.ResponseWriter, r *http.Request) {
	var req WalkAnalyzeRequest
	if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
		return
	}

	validatedPath, pathErr := s.validateWalkFilePath(req.Filename)
	if pathErr != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_path", pathErr.Error(), nil)
		return
	}

	analysis, err := walkanalysis.AnalyzeFile(validatedPath)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "[API] Walk analysis error", "error", err)
		writeError(w, r, http.StatusInternalServerError, "analysis_failed",
			fmt.Sprintf("Failed to analyze walk file: %v", err), nil)
		return
	}

	s.logger.InfoContext(r.Context(), "[API] Walk file analyzed",
		"filename", validatedPath,
		"interfaces", analysis.Statistics.TotalInterfaces,
		"neighbors", analysis.Statistics.TotalNeighbors,
	)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(WalkAnalyzeResponse{Success: true, Result: analysis})
}
