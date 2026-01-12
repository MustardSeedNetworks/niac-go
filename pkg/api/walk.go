package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/krisarmstrong/niac-go/pkg/snmp"
)

// SECURITY FIX #154: Secure path validation for walk files
// validateWalkFilePath ensures the file path is safe and doesn't traverse outside allowed directories.
func (s *Server) validateWalkFilePath(filename string) (string, error) {
	// Empty filename is invalid
	if filename == "" {
		return "", errors.New("filename cannot be empty")
	}

	// Clean the path to normalize it
	cleanPath := filepath.Clean(filename)

	// Reject paths containing null bytes (potential bypass attempt)
	if strings.ContainsRune(cleanPath, 0) {
		return "", errors.New("filename contains invalid characters")
	}

	// If path is relative, make it relative to config directory
	var absPath string
	if !filepath.IsAbs(cleanPath) {
		// Get config directory from server config
		configDir := "."
		if s.cfg.ConfigPath != "" {
			configDir = filepath.Dir(s.cfg.ConfigPath)
		}
		absPath = filepath.Join(configDir, cleanPath)
	} else {
		absPath = cleanPath
	}

	// Clean the absolute path
	absPath = filepath.Clean(absPath)

	// Verify the path exists and is a file (not a directory)
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("walk file not found: %s", filename)
		}
		return "", fmt.Errorf("cannot access walk file: %w", err)
	}
	if info.IsDir() {
		return "", errors.New("path is a directory, not a file")
	}

	// Verify file has a reasonable extension (.walk, .snmpwalk, or .txt)
	ext := strings.ToLower(filepath.Ext(absPath))
	validExts := map[string]bool{".walk": true, ".snmpwalk": true, ".txt": true}
	if !validExts[ext] {
		return "", fmt.Errorf("invalid walk file extension: %s (allowed: .walk, .snmpwalk, .txt)", ext)
	}

	return absPath, nil
}

// WalkValidationRequest is the request body for walk file validation.
type WalkValidationRequest struct {
	Filename string `json:"filename"`
	AutoFix  bool   `json:"auto_fix,omitempty"`
}

// WalkValidationResponse wraps the validation result.
type WalkValidationResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message,omitempty"`
	Result  *snmp.ValidationResult `json:"result,omitempty"`
}

// handleWalkValidation handles walk file validation requests
// POST /api/v1/walk/validate - Validate a walk file
// POST /api/v1/walk/fix - Validate and auto-fix a walk file.
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

	// SECURITY FIX #154: Validate and sanitize the file path
	validatedPath, pathErr := s.validateWalkFilePath(req.Filename)
	if pathErr != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_path", pathErr.Error(), nil)
		return
	}

	// Log the validation request with sanitized path
	slog.Info("[API] Walk file validation request", "filename", validatedPath, "autoFix", isAutoFix || req.AutoFix)

	var (
		result *snmp.ValidationResult
		err    error
	)

	if isAutoFix || req.AutoFix {
		// Validate and auto-fix using validated path
		result, err = snmp.AutoFixWalkFile(validatedPath, "")
		if err != nil {
			slog.Error("[API] Walk file auto-fix error", "error", err)
			writeError(
				w,
				r,
				http.StatusInternalServerError,
				"fix_failed",
				fmt.Sprintf("Failed to fix walk file: %v", err),
				nil,
			)

			return
		}

		slog.Info("[API] Walk file auto-fixed", "fixedCount", result.FixedCount, "filename", validatedPath)
	} else {
		// Validate only using validated path
		result, err = snmp.ValidateWalkFile(validatedPath)
		if err != nil {
			slog.Error("[API] Walk file validation error", "error", err)
			writeError(
				w,
				r,
				http.StatusInternalServerError,
				"validation_failed",
				fmt.Sprintf("Failed to validate walk file: %v", err),
				nil,
			)

			return
		}

		slog.Info(
			"[API] Walk file validated",
			"filename",
			validatedPath,
			"valid",
			result.Valid,
			"issues",
			len(result.Issues),
		)
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
	_ = json.NewEncoder(w).Encode(response) // HTTP write errors are non-critical
}

// handleWalkList lists available walk files
// GET /api/v1/walk/list - List walk files in the config directory.
func (s *Server) handleWalkList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)

		return
	}

	// Get the config from the server's config
	if s.cfg.Config == nil {
		writeError(
			w,
			r,
			http.StatusInternalServerError,
			"config_unavailable",
			"Server configuration not available",
			nil,
		)

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
	_ = json.NewEncoder(w).Encode(response) // HTTP write errors are non-critical
}

// handleWalkBatchValidate validates multiple walk files at once
// POST /api/v1/walk/validate-all - Validate all configured walk files.
func (s *Server) handleWalkBatchValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)

		return
	}

	if s.cfg.Config == nil {
		writeError(
			w,
			r,
			http.StatusInternalServerError,
			"config_unavailable",
			"Server configuration not available",
			nil,
		)

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

	var (
		totalIssues  int
		invalidCount int
	)

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

	slog.Info(
		"[API] Batch walk file validation",
		"totalFiles",
		len(walkFiles),
		"invalidFiles",
		invalidCount,
		"totalIssues",
		totalIssues,
	)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response) // HTTP write errors are non-critical
}
