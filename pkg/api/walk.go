package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/krisarmstrong/niac-go/pkg/snmp"
)

// WalkValidationRequest is the request body for walk file validation
type WalkValidationRequest struct {
	Filename string `json:"filename"`
	AutoFix  bool   `json:"auto_fix,omitempty"`
}

// WalkValidationResponse wraps the validation result
type WalkValidationResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message,omitempty"`
	Result  *snmp.ValidationResult `json:"result,omitempty"`
}

// handleWalkValidation handles walk file validation requests
// POST /api/v1/walk/validate - Validate a walk file
// POST /api/v1/walk/fix - Validate and auto-fix a walk file
func (s *Server) handleWalkValidation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
		return
	}

	// Determine action from URL path
	path := r.URL.Path
	isAutoFix := strings.HasSuffix(path, "/fix")

	// Parse request body
	var req WalkValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", fmt.Sprintf("Invalid request body: %v", err), nil)
		return
	}

	if req.Filename == "" {
		writeError(w, r, http.StatusBadRequest, "missing_filename", "Filename is required", nil)
		return
	}

	// Security: validate the filename is within allowed paths
	// Only allow files in the config directory or walk file directories
	cleanPath := filepath.Clean(req.Filename)
	if strings.Contains(cleanPath, "..") {
		writeError(w, r, http.StatusBadRequest, "invalid_path", "Invalid filename: directory traversal not allowed", nil)
		return
	}

	// Log the validation request
	log.Printf("[API] Walk file validation request: %s (auto-fix: %v)", req.Filename, isAutoFix || req.AutoFix)

	var result *snmp.ValidationResult
	var err error

	if isAutoFix || req.AutoFix {
		// Validate and auto-fix
		result, err = snmp.AutoFixWalkFile(req.Filename, "")
		if err != nil {
			log.Printf("[API] Walk file auto-fix error: %v", err)
			writeError(w, r, http.StatusInternalServerError, "fix_failed", fmt.Sprintf("Failed to fix walk file: %v", err), nil)
			return
		}
		log.Printf("[API] Walk file auto-fixed: %d issues fixed in %s", result.FixedCount, req.Filename)
	} else {
		// Validate only
		result, err = snmp.ValidateWalkFile(req.Filename)
		if err != nil {
			log.Printf("[API] Walk file validation error: %v", err)
			writeError(w, r, http.StatusInternalServerError, "validation_failed", fmt.Sprintf("Failed to validate walk file: %v", err), nil)
			return
		}
		log.Printf("[API] Walk file validated: %s (valid: %v, issues: %d)", req.Filename, result.Valid, len(result.Issues))
	}

	response := WalkValidationResponse{
		Success: true,
		Result:  result,
	}

	if isAutoFix || req.AutoFix {
		response.Message = fmt.Sprintf("Walk file validated and %d issues fixed", result.FixedCount)
	} else {
		if result.Valid {
			response.Message = "Walk file is valid"
		} else {
			response.Message = fmt.Sprintf("Walk file has %d issues", len(result.Issues))
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleWalkList lists available walk files
// GET /api/v1/walk/list - List walk files in the config directory
func (s *Server) handleWalkList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
		return
	}

	// Get the config from the server's config
	if s.cfg.Config == nil {
		writeError(w, r, http.StatusInternalServerError, "config_unavailable", "Server configuration not available", nil)
		return
	}

	// Find walk files referenced in device configs
	walkFiles := make(map[string]bool)
	for _, device := range s.cfg.Config.Devices {
		// Add single WalkFile if specified
		if device.SNMPConfig.WalkFile != "" {
			walkFiles[device.SNMPConfig.WalkFile] = true
		}
		// Add multiple WalkFiles if specified
		for _, wf := range device.SNMPConfig.WalkFiles {
			if wf != "" {
				walkFiles[wf] = true
			}
		}
	}

	// Convert to list
	var files []string
	for file := range walkFiles {
		files = append(files, file)
	}

	response := struct {
		Success bool     `json:"success"`
		Files   []string `json:"files"`
	}{
		Success: true,
		Files:   files,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleWalkBatchValidate validates multiple walk files at once
// POST /api/v1/walk/validate-all - Validate all configured walk files
func (s *Server) handleWalkBatchValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
		return
	}

	if s.cfg.Config == nil {
		writeError(w, r, http.StatusInternalServerError, "config_unavailable", "Server configuration not available", nil)
		return
	}

	// Find all unique walk files
	walkFiles := make(map[string]bool)
	for _, device := range s.cfg.Config.Devices {
		// Add single WalkFile if specified
		if device.SNMPConfig.WalkFile != "" {
			walkFiles[device.SNMPConfig.WalkFile] = true
		}
		// Add multiple WalkFiles if specified
		for _, wf := range device.SNMPConfig.WalkFiles {
			if wf != "" {
				walkFiles[wf] = true
			}
		}
	}

	// Validate each file
	results := make(map[string]*snmp.ValidationResult)
	var totalIssues int
	var invalidCount int

	for file := range walkFiles {
		result, err := snmp.ValidateWalkFile(file)
		if err != nil {
			// Create an error result
			results[file] = &snmp.ValidationResult{
				Filename: file,
				Valid:    false,
				Issues: []snmp.ValidationIssue{
					{
						Line:     0,
						Severity: "error",
						Message:  err.Error(),
					},
				},
			}
			invalidCount++
		} else {
			results[file] = result
			totalIssues += len(result.Issues)
			if !result.Valid {
				invalidCount++
			}
		}
	}

	response := struct {
		Success      bool                              `json:"success"`
		Message      string                            `json:"message"`
		TotalFiles   int                               `json:"total_files"`
		InvalidFiles int                               `json:"invalid_files"`
		TotalIssues  int                               `json:"total_issues"`
		Results      map[string]*snmp.ValidationResult `json:"results"`
	}{
		Success:      invalidCount == 0,
		TotalFiles:   len(walkFiles),
		InvalidFiles: invalidCount,
		TotalIssues:  totalIssues,
		Results:      results,
	}

	if invalidCount == 0 {
		response.Message = fmt.Sprintf("All %d walk files are valid", len(walkFiles))
	} else {
		response.Message = fmt.Sprintf("%d of %d walk files have issues", invalidCount, len(walkFiles))
	}

	log.Printf("[API] Batch walk file validation: %d files, %d invalid, %d total issues",
		len(walkFiles), invalidCount, totalIssues)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
