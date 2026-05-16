package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/protocols/snmp"
)

// SECURITY FIX #154, #166: Secure path validation for walk files
// validateWalkFilePath ensures the file path is safe and doesn't traverse outside allowed directories.
// Walks are resolved in two passes: first the legacy config-dir / cwd
// allow-list (kept for backward compatibility with bare filenames in
// the running config), then the unified library walks dir
// (~/.niac/library/walks/) so the new picker integrations (#556) can
// send library-relative names like "cisco/c3900.walk".
func (s *Server) validateWalkFilePath(filename string) (string, error) {
	if filename == "" {
		return "", errors.New("filename cannot be empty")
	}

	cleanPath := filepath.Clean(filename)
	if strings.ContainsRune(cleanPath, 0) {
		return "", errors.New("filename contains invalid characters")
	}

	allowedDirs, err := s.resolveWalkAllowedDirs()
	if err != nil {
		return "", err
	}

	// Try each allowed dir in order. First one that produces a path
	// passing all checks wins. If a dir produces a path that fails on
	// "file not found", fall through to the next dir; any other error
	// (extension, traversal) short-circuits.
	var lastNotFound error
	for _, allowedDir := range allowedDirs {
		absPath := cleanPath
		if !filepath.IsAbs(cleanPath) {
			absPath = filepath.Join(allowedDir, cleanPath)
		}
		absPath = filepath.Clean(absPath)

		if traversalErr := ensureWalkPathWithin(absPath, allowedDir, filename); traversalErr != nil {
			// "access denied" against THIS dir doesn't preclude finding
			// the file under a different dir. Keep going.
			lastNotFound = traversalErr
			continue
		}
		if statErr := checkWalkFileExists(absPath, filename); statErr != nil {
			lastNotFound = statErr
			continue
		}
		if extErr := validateWalkExtension(absPath); extErr != nil {
			// Extension errors aren't dir-specific — surface immediately.
			return "", extErr
		}
		return absPath, nil
	}

	if lastNotFound != nil {
		return "", lastNotFound
	}
	return "", fmt.Errorf("walk file not found: %s", filename)
}

// resolveWalkAllowedDirs returns the ordered list of directories walk
// files may live under. The first entry is the legacy config-dir / cwd
// surface; the second is the library walks/ subdir when the library
// opened successfully. Order matters — config-dir wins so a walk
// referenced relative to the running config behaves the same as
// before this change.
func (s *Server) resolveWalkAllowedDirs() ([]string, error) {
	var dirs []string

	cfgPath := s.configPath()
	if cfgPath != "" {
		dirs = append(dirs, filepath.Dir(cfgPath))
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("cannot determine allowed directory: %w", err)
		}
		dirs = append(dirs, cwd)
	}

	// Library walks dir — only added when the library opened cleanly.
	// Library failure is non-fatal for the daemon as a whole, and
	// silently omitting it here matches that posture: clients hitting
	// /api/v1/library/walks already see 503, so this is just one more
	// place that surface is unavailable.
	if s.library != nil {
		dirs = append(dirs, s.library.SubDir("walks"))
	}

	return dirs, nil
}

// ensureWalkPathWithin verifies that absPath resolves under allowedDir (following symlinks).
func ensureWalkPathWithin(absPath, allowedDir, filename string) error {
	// Inline barrier on the EvalSymlinks/Stat sinks below so static analysers
	// see the path is bounded before any filesystem access.
	if !filepath.IsAbs(absPath) || strings.Contains(absPath, "..") {
		return errors.New("walk file path failed safety check")
	}
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("walk file not found: %s", filename)
		}
		realPath = absPath
	}

	realAllowedDir, err := filepath.EvalSymlinks(allowedDir)
	if err != nil {
		realAllowedDir = allowedDir
	}

	if !strings.HasPrefix(realPath, realAllowedDir+string(filepath.Separator)) && realPath != realAllowedDir {
		return fmt.Errorf("access denied: file must be within %s", allowedDir)
	}
	return nil
}

// checkWalkFileExists stats the walk path and ensures it is a regular file.
func checkWalkFileExists(absPath, filename string) error {
	if !filepath.IsAbs(absPath) || strings.Contains(absPath, "..") {
		return errors.New("walk file path failed safety check")
	}
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("walk file not found: %s", filename)
		}
		return fmt.Errorf("cannot access walk file: %w", err)
	}
	if info.IsDir() {
		return errors.New("path is a directory, not a file")
	}
	return nil
}

// validateWalkExtension rejects walk paths without a .walk/.snmpwalk/.txt extension.
func validateWalkExtension(absPath string) error {
	ext := strings.ToLower(filepath.Ext(absPath))
	validExts := map[string]bool{".walk": true, ".snmpwalk": true, ".txt": true}
	if !validExts[ext] {
		return fmt.Errorf("invalid walk file extension: %s (allowed: .walk, .snmpwalk, .txt)", ext)
	}
	return nil
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
// runWalkValidation invokes validate-or-fix on the walk file, writing errors and returning ok.
func (s *Server) runWalkValidation(
	w http.ResponseWriter, r *http.Request, validatedPath string, autoFix bool,
) (*snmp.ValidationResult, bool) {
	if autoFix {
		result, err := snmp.AutoFixWalkFile(validatedPath, "")
		if err != nil {
			s.logger.Error("[API] Walk file auto-fix error", "error", err)
			writeError(w, r, http.StatusInternalServerError, "fix_failed",
				fmt.Sprintf("Failed to fix walk file: %v", err), nil)
			return nil, false
		}
		s.logger.Info("[API] Walk file auto-fixed", "fixedCount", result.FixedCount, "filename", validatedPath)
		return result, true
	}

	result, err := snmp.ValidateWalkFile(validatedPath)
	if err != nil {
		s.logger.Error("[API] Walk file validation error", "error", err)
		writeError(w, r, http.StatusInternalServerError, "validation_failed",
			fmt.Sprintf("Failed to validate walk file: %v", err), nil)
		return nil, false
	}
	s.logger.Info(
		"[API] Walk file validated",
		"filename", validatedPath,
		"valid", result.Valid,
		"issues", len(result.Issues),
	)
	return result, true
}

// buildWalkValidationResponse returns the WalkValidationResponse with a contextual message.
func buildWalkValidationResponse(result *snmp.ValidationResult, autoFix bool) WalkValidationResponse {
	response := WalkValidationResponse{Success: true, Result: result}
	switch {
	case autoFix:
		response.Message = fmt.Sprintf("Walk file validated and %d issues fixed", result.FixedCount)
	case result.Valid:
		response.Message = "Walk file is valid"
	default:
		response.Message = fmt.Sprintf("Walk file has %d issues", len(result.Issues))
	}
	return response
}

func (s *Server) handleWalkValidation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
		return
	}

	isAutoFix := strings.HasSuffix(r.URL.Path, "/fix")
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

	var req WalkValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", fmt.Sprintf("Invalid request body: %v", err), nil)
		return
	}

	validatedPath, pathErr := s.validateWalkFilePath(req.Filename)
	if pathErr != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_path", pathErr.Error(), nil)
		return
	}

	s.logger.Info("[API] Walk file validation request", "filename", validatedPath, "autoFix", isAutoFix || req.AutoFix)

	result, ok := s.runWalkValidation(w, r, validatedPath, isAutoFix || req.AutoFix)
	if !ok {
		return
	}

	response := buildWalkValidationResponse(result, isAutoFix || req.AutoFix)

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

	// SECURITY FIX: Thread-safe access to config via currentConfig()
	cfg := s.currentConfig()
	if cfg == nil {
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

	for _, device := range cfg.Devices {
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
// collectConfigWalkFiles returns the set of walk-file paths referenced by any device.
func collectConfigWalkFiles(devices []config.Device) map[string]bool {
	walkFiles := make(map[string]bool)
	for _, device := range devices {
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

// validateWalkFileBatch validates every walk file in the set, returning results and counters.
func (s *Server) validateWalkFileBatch(
	walkFiles map[string]bool,
) (map[string]*snmp.ValidationResult, int, int) {
	results := make(map[string]*snmp.ValidationResult, len(walkFiles))
	var totalIssues, invalidCount int

	for file := range walkFiles {
		if _, err := s.validateWalkFilePath(file); err != nil {
			results[file] = makeWalkErrorResult(file, err)
			invalidCount++
			continue
		}

		result, err := snmp.ValidateWalkFile(file)
		if err != nil {
			results[file] = makeWalkErrorResult(file, err)
			invalidCount++
			continue
		}

		results[file] = result
		totalIssues += len(result.Issues)
		if !result.Valid {
			invalidCount++
		}
	}
	return results, totalIssues, invalidCount
}

// makeWalkErrorResult wraps an error into a ValidationResult for the batch response.
func makeWalkErrorResult(file string, err error) *snmp.ValidationResult {
	return &snmp.ValidationResult{
		Filename: file,
		Valid:    false,
		Issues: []snmp.ValidationIssue{
			{Line: 0, Severity: "error", Message: err.Error()},
		},
	}
}

// buildBatchWalkResponse assembles the JSON response body for handleWalkBatchValidate.
func buildBatchWalkResponse(
	walkFiles map[string]bool,
	results map[string]*snmp.ValidationResult,
	totalIssues, invalidCount int,
) struct {
	Success      bool                              `json:"success"`
	Message      string                            `json:"message"`
	TotalFiles   int                               `json:"total_files"`
	InvalidFiles int                               `json:"invalid_files"`
	TotalIssues  int                               `json:"total_issues"`
	Results      map[string]*snmp.ValidationResult `json:"results"`
} {
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
	return response
}

func (s *Server) handleWalkBatchValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
		return
	}

	cfg := s.currentConfig()
	if cfg == nil {
		writeError(w, r, http.StatusInternalServerError, "config_unavailable",
			"Server configuration not available", nil)
		return
	}

	walkFiles := collectConfigWalkFiles(cfg.Devices)
	results, totalIssues, invalidCount := s.validateWalkFileBatch(walkFiles)

	response := buildBatchWalkResponse(walkFiles, results, totalIssues, invalidCount)

	s.logger.Info(
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
