package api

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/api/capture"
)

// PcapUploadRequest is the request body for uploading a PCAP.
type PcapUploadRequest struct {
	Filename string `json:"filename"`
	Data     string `json:"data"` // Base64 encoded PCAP data
}

// PcapUploadResponse is returned after successful upload.
type PcapUploadResponse struct {
	Success    bool   `json:"success"`
	AnalysisID string `json:"analysisId"`
	Message    string `json:"message"`
}

// ============================================================================
// PCAP Analysis Handlers
// ============================================================================

// decodePcapUpload parses the JSON body, decodes the base64 payload, and validates PCAP magic.
// Returns the raw PCAP bytes, the parsed request, and true on success.
func decodePcapUpload(w http.ResponseWriter, r *http.Request) ([]byte, PcapUploadRequest, bool) {
	var req PcapUploadRequest
	if !decodeJSONStrict(w, r, &req, MaxPCAPUploadBodySize) {
		return nil, req, false
	}

	if req.Data == "" {
		writeError(w, r, http.StatusBadRequest, "missing_data",
			"PCAP data is required", nil)
		return nil, req, false
	}

	pcapData, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_base64",
			"Invalid base64 encoded data", nil)
		return nil, req, false
	}

	// The shipped examples/captures/*.pcap files are mislabelled snoop
	// (RFC 1761) captures. Transparently convert to classic pcap so the
	// rest of the analyser — which is built around libpcap-style records
	// via gopacket — stays unchanged.
	if capture.HasSnoopMagic(pcapData) {
		converted, convErr := capture.SnoopToPCAP(pcapData)
		if convErr != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_snoop",
				fmt.Sprintf("snoop capture could not be converted: %v", convErr), nil)
			return nil, req, false
		}
		pcapData = converted
	}

	if validateErr := capture.ValidateStructure(pcapData); validateErr != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_pcap",
			validateErr.Error(), nil)
		return nil, req, false
	}

	return pcapData, req, true
}

// handlePcapUpload handles POST /api/v1/pcap/upload.
func (s *Server) handlePcapUpload(w http.ResponseWriter, r *http.Request) {
	// Method gating (POST-only) is enforced declaratively by the route registry.
	pcapData, req, ok := decodePcapUpload(w, r)
	if !ok {
		return
	}

	// Generate analysis ID from content hash
	hash := sha256.Sum256(pcapData)
	analysisID := hex.EncodeToString(hash[:8])

	// Check if we already have this analysis cached
	if existing, found := s.pcapCache.Get(analysisID); found {
		s.writeJSON(w, PcapUploadResponse{
			Success:    true,
			AnalysisID: analysisID,
			Message:    fmt.Sprintf("Analysis retrieved from cache (%d packets)", len(existing.Packets)),
		})
		return
	}

	result, err := capture.Analyze(pcapData, req.Filename)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "[API] PCAP analysis failed", "error", err, "filename", req.Filename)
		writeError(w, r, http.StatusInternalServerError, "analysis_failed",
			"Failed to analyze PCAP file", nil)
		return
	}

	result.ID = analysisID
	result.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	s.pcapCache.Set(analysisID, result)

	s.writeJSON(w, PcapUploadResponse{
		Success:    true,
		AnalysisID: analysisID,
		Message:    fmt.Sprintf("Successfully analyzed %d packets", len(result.Packets)),
	})
}

// handlePcapAnalysis handles GET /api/v1/pcap/{id}.
func (s *Server) handlePcapAnalysis(w http.ResponseWriter, r *http.Request) {
	// Method gating (GET-only) is enforced declaratively by the route registry.
	// Extract analysis ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/pcap/")
	analysisID := strings.TrimSuffix(path, "/")

	if analysisID == "" {
		writeError(w, r, http.StatusBadRequest, "missing_id",
			"Analysis ID is required", nil)

		return
	}

	// Look up in cache
	result, ok := s.pcapCache.Get(analysisID)
	if !ok {
		writeError(w, r, http.StatusNotFound, "not_found",
			"Analysis not found. It may have expired or was never created.", nil)

		return
	}

	s.writeJSON(w, pageResult(result, r.URL.Query()))
}

// pageResult applies ?offset= and ?limit= to the packet list.
//
// Both default to "everything retained", so a client that does not ask for a
// page sees exactly what it saw before. A capture at the retention cap is
// 50,000 rows, which is more than a browser wants in one response but not so
// many that slicing them here is expensive.
func pageResult(result *capture.AnalysisResult, query url.Values) *capture.AnalysisResult {
	offset := queryInt(query, "offset", 0)
	limit := queryInt(query, "limit", len(result.Packets))
	if offset <= 0 && limit >= len(result.Packets) {
		return result
	}

	offset = max(offset, 0)
	offset = min(offset, len(result.Packets))
	end := len(result.Packets)
	if limit >= 0 && offset+limit < end {
		end = offset + limit
	}

	// Copy the header, replace the window: the cached result must not be
	// mutated, or the next reader gets one client's page.
	paged := *result
	paged.Packets = result.Packets[offset:end]

	return &paged
}

// queryInt reads a non-negative integer query parameter, falling back to def
// for anything absent or unparseable. A malformed page request reads the
// default rather than failing: the caller wanted packets, not an error page.
func queryInt(query url.Values, name string, def int) int {
	raw := query.Get(name)
	if raw == "" {
		return def
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return def
	}

	return value
}
