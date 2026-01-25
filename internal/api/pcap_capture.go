package api

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// handlePcapUpload handles POST /api/v1/pcap/upload.
func (s *Server) handlePcapUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed",
			"Only POST method is allowed", nil)

		return
	}

	// Limit request size
	r.Body = http.MaxBytesReader(w, r.Body, MaxPCAPUploadSize)

	var req PcapUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err.Error() == ErrMsgRequestBodyTooLarge {
			writeError(w, r, http.StatusRequestEntityTooLarge, "request_too_large",
				"PCAP file too large (max 100MB)", nil)

			return
		}

		writeError(w, r, http.StatusBadRequest, "invalid_request",
			"Invalid JSON request body", nil)

		return
	}

	// Validate request
	if req.Data == "" {
		writeError(w, r, http.StatusBadRequest, "missing_data",
			"PCAP data is required", nil)

		return
	}

	// Decode base64 data
	pcapData, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_base64",
			"Invalid base64 encoded data", nil)

		return
	}

	// Validate PCAP magic number
	if validateErr := validatePCAPMagic(pcapData); validateErr != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_pcap",
			validateErr.Error(), nil)

		return
	}

	// Generate analysis ID from content hash
	hash := sha256.Sum256(pcapData)
	analysisID := hex.EncodeToString(hash[:8])

	// Check if we already have this analysis cached
	if existing, ok := s.pcapCache.Get(analysisID); ok {
		s.writeJSON(w, PcapUploadResponse{
			Success:    true,
			AnalysisID: analysisID,
			Message:    fmt.Sprintf("Analysis retrieved from cache (%d packets)", len(existing.Packets)),
		})

		return
	}

	// Analyze the PCAP
	result, err := analyzePcapData(pcapData, req.Filename)
	if err != nil {
		s.logger.Error("[API] PCAP analysis failed", "error", err)
		writeError(w, r, http.StatusInternalServerError, "analysis_failed",
			"PCAP analysis failed", nil)

		return
	}

	result.ID = analysisID
	result.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	// Cache the result
	s.pcapCache.Set(analysisID, result)

	s.writeJSON(w, PcapUploadResponse{
		Success:    true,
		AnalysisID: analysisID,
		Message:    fmt.Sprintf("Successfully analyzed %d packets", len(result.Packets)),
	})
}

// handlePcapAnalysis handles GET /api/v1/pcap/{id}.
func (s *Server) handlePcapAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed",
			"Only GET method is allowed", nil)

		return
	}

	// Extract analysis ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/pcap/")
	analysisID := strings.TrimSuffix(path, "/")

	if analysisID == "" {
		writeError(w, r, http.StatusBadRequest, "missing_id",
			"Analysis ID is required", nil)

		return
	}

	// FIX #319: Validate analysis ID format (hex-encoded SHA256 prefix = 16 hex chars)
	if len(analysisID) != 16 || !isHexString(analysisID) {
		writeError(w, r, http.StatusBadRequest, "invalid_id",
			"Invalid analysis ID format", nil)

		return
	}

	// Look up in cache
	result, ok := s.pcapCache.Get(analysisID)
	if !ok {
		writeError(w, r, http.StatusNotFound, "not_found",
			"Analysis not found. It may have expired or was never created.", nil)

		return
	}

	s.writeJSON(w, result)
}
