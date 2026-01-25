package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/krisarmstrong/niac-go/internal/protocols/snmp"
)

// SECURITY FIX #154, #166: Secure path validation for walk files
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

	// SECURITY FIX #161: Thread-safe access to ConfigPath
	// SECURITY FIX #166: Get config directory as the allowed base directory
	cfgPath := s.configPath()
	var allowedDir string
	if cfgPath != "" {
		allowedDir = filepath.Dir(cfgPath)
	} else {
		// If no config path, use current working directory
		var err error
		allowedDir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("cannot determine allowed directory: %w", err)
		}
	}

	// Resolve to absolute path
	var absPath string
	if !filepath.IsAbs(cleanPath) {
		absPath = filepath.Join(allowedDir, cleanPath)
	} else {
		absPath = cleanPath
	}

	// Clean the absolute path
	absPath = filepath.Clean(absPath)

	// SECURITY FIX #166: Resolve symlinks to prevent symlink-based traversal attacks
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If symlink resolution fails, file may not exist
		if os.IsNotExist(err) {
			return "", fmt.Errorf("walk file not found: %s", filename)
		}
		realPath = absPath
	}

	// Resolve allowed directory to real path for comparison
	realAllowedDir, err := filepath.EvalSymlinks(allowedDir)
	if err != nil {
		realAllowedDir = allowedDir
	}

	// SECURITY FIX #166: Verify the file is within the allowed directory (prevent path traversal)
	if !strings.HasPrefix(realPath, realAllowedDir+string(filepath.Separator)) && realPath != realAllowedDir {
		return "", fmt.Errorf("access denied: file must be within %s", allowedDir)
	}

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

	// Parse request body with size limit
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

	var req WalkValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)

		return
	}

	// SECURITY FIX #154: Validate and sanitize the file path
	validatedPath, pathErr := s.validateWalkFilePath(req.Filename)
	if pathErr != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_path", pathErr.Error(), nil)
		return
	}

	// Log the validation request with sanitized path
	s.logger.Info("[API] Walk file validation request", "filename", validatedPath, "autoFix", isAutoFix || req.AutoFix)

	var (
		result *snmp.ValidationResult
		err    error
	)

	if isAutoFix || req.AutoFix {
		// Validate and auto-fix using validated path
		result, err = snmp.AutoFixWalkFile(validatedPath, "")
		if err != nil {
			s.logger.Error("[API] Walk file auto-fix error", "error", err, "filename", validatedPath)
			writeError(w, r, http.StatusInternalServerError, "fix_failed",
				"Walk file auto-fix failed", nil)

			return
		}

		s.logger.Info("[API] Walk file auto-fixed", "fixedCount", result.FixedCount, "filename", validatedPath)
	} else {
		// Validate only using validated path
		result, err = snmp.ValidateWalkFile(validatedPath)
		if err != nil {
			s.logger.Error("[API] Walk file validation error", "error", err, "filename", validatedPath)
			writeError(w, r, http.StatusInternalServerError, "validation_failed",
				"Walk file validation failed", nil)

			return
		}

		s.logger.Info(
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

// walkBatchValidationResponse holds the batch validation response data.
type walkBatchValidationResponse struct {
	Success      bool                              `json:"success"`
	Message      string                            `json:"message"`
	TotalFiles   int                               `json:"total_files"`
	InvalidFiles int                               `json:"invalid_files"`
	TotalIssues  int                               `json:"total_issues"`
	Results      map[string]*snmp.ValidationResult `json:"results"`
}

// collectConfiguredWalkFiles extracts all unique walk files from device configurations.
func (s *Server) collectConfiguredWalkFiles() map[string]bool {
	walkFiles := make(map[string]bool)

	for _, device := range s.cfg.Config.Devices {
		if device.SNMPConfig.WalkFile != "" {
			walkFiles[device.SNMPConfig.WalkFile] = true
		}
		for _, wf := range device.SNMPConfig.WalkFiles {
			if wf != "" {
				walkFiles[wf] = true
			}
		}
	}

	return walkFiles
}

// validateSingleWalkFile validates a single walk file and returns the result.
func (s *Server) validateSingleWalkFile(file string) (*snmp.ValidationResult, error) {
	validatedPath, pathErr := s.validateWalkFilePath(file)
	if pathErr != nil {
		return &snmp.ValidationResult{
			Filename: file,
			Valid:    false,
			Issues: []snmp.ValidationIssue{
				{
					Line:     0,
					Severity: "error",
					Message:  "Invalid file path",
				},
			},
		}, pathErr
	}

	result, err := snmp.ValidateWalkFile(validatedPath)
	if err != nil {
		s.logger.Error("[API] Walk file batch validation error", "error", err, "filename", validatedPath)
		return &snmp.ValidationResult{
			Filename: file,
			Valid:    false,
			Issues: []snmp.ValidationIssue{
				{
					Line:     0,
					Severity: "error",
					Message:  "Validation failed",
				},
			},
		}, err
	}

	return result, nil
}

// buildBatchValidationResponse constructs the response from validation results.
func buildBatchValidationResponse(
	results map[string]*snmp.ValidationResult,
	totalFiles int,
) *walkBatchValidationResponse {
	var totalIssues, invalidCount int

	for _, result := range results {
		totalIssues += len(result.Issues)
		if !result.Valid {
			invalidCount++
		}
	}

	resp := &walkBatchValidationResponse{
		Success:      invalidCount == 0,
		TotalFiles:   totalFiles,
		InvalidFiles: invalidCount,
		TotalIssues:  totalIssues,
		Results:      results,
	}

	if invalidCount == 0 {
		resp.Message = fmt.Sprintf("All %d walk files are valid", totalFiles)
	} else {
		resp.Message = fmt.Sprintf("%d of %d walk files have issues", invalidCount, totalFiles)
	}

	return resp
}

// handleWalkBatchValidate validates multiple walk files at once
// POST /api/v1/walk/validate-all - Validate all configured walk files.
func (s *Server) handleWalkBatchValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
		return
	}

	if s.cfg.Config == nil {
		writeError(w, r, http.StatusInternalServerError, "config_unavailable",
			"Server configuration not available", nil)
		return
	}

	walkFiles := s.collectConfiguredWalkFiles()
	results := make(map[string]*snmp.ValidationResult)

	for file := range walkFiles {
		result, _ := s.validateSingleWalkFile(file)
		results[file] = result
	}

	response := buildBatchValidationResponse(results, len(walkFiles))

	s.logger.Info("[API] Batch walk file validation",
		"totalFiles", response.TotalFiles,
		"invalidFiles", response.InvalidFiles,
		"totalIssues", response.TotalIssues)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
