package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/protocols/snmp"
)

// validWalkContent contains valid walk file test data.
const validWalkContent = `.1.3.6.1.2.1.1.1.0 = STRING: "Cisco IOS XE"
.1.3.6.1.2.1.1.3.0 = Timeticks: (12345) 0:02:03.45
.1.3.6.1.2.1.1.5.0 = STRING: "router-01"
`

// walkContentWithIssues contains walk file test data with issues.
const walkContentWithIssues = `.1.3.6.1.2.1.1.1.0 = STRING: "Cisco IOS XE"
.1.3.6.1.2.1.1.3.0  Timeticks: (12345)
.1.3.6.1.2.1.1.5.0 = STRING: "router-01"
`

func createTestWalkFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	walkPath := filepath.Join(dir, name)
	err := os.WriteFile(walkPath, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("write walk file: %v", err)
	}
	return walkPath
}

func TestHandleWalkValidation_ValidFile(t *testing.T) {
	server, configPath := newTestServer(t)
	configDir := filepath.Dir(configPath)

	// Create a valid walk file in the config directory
	walkPath := createTestWalkFile(t, configDir, "test.walk", validWalkContent)

	// Create validation request
	reqBody := map[string]any{
		"filename": filepath.Base(walkPath),
	}
	body, _ := json.Marshal(reqBody)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/walk/validate", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	server.handleWalkValidation(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp WalkValidationResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success=true, got false: %s", resp.Message)
	}

	if resp.Result == nil {
		t.Fatal("expected result to be non-nil")
	}

	if !resp.Result.Valid {
		t.Errorf("expected valid file, got invalid with issues: %+v", resp.Result.Issues)
	}
}

func TestHandleWalkValidation_FileWithIssues(t *testing.T) {
	server, configPath := newTestServer(t)
	configDir := filepath.Dir(configPath)

	// Create a walk file with issues
	walkPath := createTestWalkFile(t, configDir, "issues.walk", walkContentWithIssues)

	reqBody := map[string]any{
		"filename": filepath.Base(walkPath),
	}
	body, _ := json.Marshal(reqBody)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/walk/validate", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	server.handleWalkValidation(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp WalkValidationResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Result == nil {
		t.Fatal("expected result to be non-nil")
	}

	if len(resp.Result.Issues) == 0 {
		t.Error("expected issues to be found")
	}
}

func TestHandleWalkValidation_MethodNotAllowed(t *testing.T) {
	server, _ := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/walk/validate", nil)

	server.handleWalkValidation(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleWalkValidation_InvalidJSON(t *testing.T) {
	server, _ := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/walk/validate",
		strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")

	server.handleWalkValidation(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleWalkValidation_InvalidPath(t *testing.T) {
	server, _ := newTestServer(t)

	// Try path traversal attack
	reqBody := map[string]any{
		"filename": "../../../etc/passwd",
	}
	body, _ := json.Marshal(reqBody)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/walk/validate", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	server.handleWalkValidation(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for path traversal, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleWalkList_WithWalkFiles(t *testing.T) {
	server, _ := newTestServer(t)

	// Configure devices with walk files
	server.cfg.Config = &config.Config{
		Devices: []config.Device{
			{
				Name: "device1",
				SNMPConfig: config.SNMPConfig{
					WalkFile: "device1.walk",
				},
			},
			{
				Name: "device2",
				SNMPConfig: config.SNMPConfig{
					WalkFiles: []string{"device2a.walk", "device2b.walk"},
				},
			},
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/walk/list", nil)

	server.handleWalkList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool     `json:"success"`
		Files   []string `json:"files"`
	}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !resp.Success {
		t.Error("expected success=true")
	}

	if len(resp.Files) != 3 {
		t.Errorf("expected 3 unique walk files, got %d: %v", len(resp.Files), resp.Files)
	}
}

func TestHandleWalkList_MethodNotAllowed(t *testing.T) {
	server, _ := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/walk/list", nil)

	server.handleWalkList(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleWalkList_NoConfig(t *testing.T) {
	server, _ := newTestServer(t)
	server.cfg.Config = nil

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/walk/list", nil)

	server.handleWalkList(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestHandleWalkBatchValidate_Success(t *testing.T) {
	server, configPath := newTestServer(t)
	configDir := filepath.Dir(configPath)

	// Create valid walk files
	walkPath1 := createTestWalkFile(t, configDir, "device1.walk", validWalkContent)
	walkPath2 := createTestWalkFile(t, configDir, "device2.walk", validWalkContent)

	// Configure devices with walk files (use relative names)
	server.cfg.Config = &config.Config{
		Devices: []config.Device{
			{
				Name: "device1",
				SNMPConfig: config.SNMPConfig{
					WalkFile: filepath.Base(walkPath1),
				},
			},
			{
				Name: "device2",
				SNMPConfig: config.SNMPConfig{
					WalkFile: filepath.Base(walkPath2),
				},
			},
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/walk/validate-all", nil)

	server.handleWalkBatchValidate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp walkBatchValidationResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success=true, got false: %s", resp.Message)
	}

	if resp.TotalFiles != 2 {
		t.Errorf("expected 2 total files, got %d", resp.TotalFiles)
	}

	if resp.InvalidFiles != 0 {
		t.Errorf("expected 0 invalid files, got %d", resp.InvalidFiles)
	}
}

func TestHandleWalkBatchValidate_MethodNotAllowed(t *testing.T) {
	server, _ := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/walk/validate-all", nil)

	server.handleWalkBatchValidate(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleWalkBatchValidate_NoConfig(t *testing.T) {
	server, _ := newTestServer(t)
	server.cfg.Config = nil

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/walk/validate-all", nil)

	server.handleWalkBatchValidate(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestCollectConfiguredWalkFiles(t *testing.T) {
	server, _ := newTestServer(t)

	// Configure devices with various walk file configurations
	server.cfg.Config = &config.Config{
		Devices: []config.Device{
			{
				Name: "device1",
				SNMPConfig: config.SNMPConfig{
					WalkFile: "single.walk",
				},
			},
			{
				Name: "device2",
				SNMPConfig: config.SNMPConfig{
					WalkFiles: []string{"multi1.walk", "multi2.walk"},
				},
			},
			{
				Name: "device3",
				SNMPConfig: config.SNMPConfig{
					WalkFile:  "both.walk",
					WalkFiles: []string{"both_multi.walk"},
				},
			},
			{
				Name: "device4",
				SNMPConfig: config.SNMPConfig{
					WalkFile: "single.walk", // Duplicate
				},
			},
		},
	}

	files := server.collectConfiguredWalkFiles()

	// Should have 5 unique files (single.walk appears twice but is deduplicated)
	expected := 5
	if len(files) != expected {
		t.Errorf("expected %d unique walk files, got %d: %v", expected, len(files), files)
	}

	// Verify specific files are present
	expectedFiles := []string{"single.walk", "multi1.walk", "multi2.walk", "both.walk", "both_multi.walk"}
	for _, f := range expectedFiles {
		if !files[f] {
			t.Errorf("expected file %s to be in collection", f)
		}
	}
}

func TestValidateSingleWalkFile_Success(t *testing.T) {
	server, configPath := newTestServer(t)
	configDir := filepath.Dir(configPath)

	walkPath := createTestWalkFile(t, configDir, "valid.walk", validWalkContent)

	result, err := server.validateSingleWalkFile(filepath.Base(walkPath))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected result to be non-nil")
	}

	if !result.Valid {
		t.Errorf("expected valid result, got invalid with issues: %+v", result.Issues)
	}
}

func TestValidateSingleWalkFile_InvalidPath(t *testing.T) {
	server, _ := newTestServer(t)

	result, err := server.validateSingleWalkFile("../../../etc/passwd")

	// Should return result with error and also an error
	if err == nil {
		t.Error("expected error for invalid path")
	}

	if result == nil {
		t.Fatal("expected result to be non-nil even on error")
	}

	if result.Valid {
		t.Error("expected invalid result")
	}

	if len(result.Issues) == 0 {
		t.Error("expected issues to be populated")
	}
}

func TestValidateSingleWalkFile_FileNotFound(t *testing.T) {
	server, _ := newTestServer(t)

	result, err := server.validateSingleWalkFile("nonexistent.walk")

	if err == nil {
		t.Error("expected error for nonexistent file")
	}

	if result == nil {
		t.Fatal("expected result to be non-nil even on error")
	}

	if result.Valid {
		t.Error("expected invalid result")
	}
}

func TestBuildBatchValidationResponse_AllValid(t *testing.T) {
	results := map[string]*snmp.ValidationResult{
		"file1.walk": {Filename: "file1.walk", Valid: true, Issues: nil},
		"file2.walk": {Filename: "file2.walk", Valid: true, Issues: nil},
	}

	resp := buildBatchValidationResponse(results, 2)

	if !resp.Success {
		t.Error("expected success=true when all files valid")
	}

	if resp.TotalFiles != 2 {
		t.Errorf("expected 2 total files, got %d", resp.TotalFiles)
	}

	if resp.InvalidFiles != 0 {
		t.Errorf("expected 0 invalid files, got %d", resp.InvalidFiles)
	}

	if resp.TotalIssues != 0 {
		t.Errorf("expected 0 total issues, got %d", resp.TotalIssues)
	}

	if !strings.Contains(resp.Message, "All 2 walk files are valid") {
		t.Errorf("unexpected message: %s", resp.Message)
	}
}

func TestBuildBatchValidationResponse_SomeInvalid(t *testing.T) {
	results := map[string]*snmp.ValidationResult{
		"file1.walk": {Filename: "file1.walk", Valid: true, Issues: nil},
		"file2.walk": {
			Filename: "file2.walk",
			Valid:    false,
			Issues: []snmp.ValidationIssue{
				{Line: 1, Severity: "error", Message: "test error"},
			},
		},
		"file3.walk": {
			Filename: "file3.walk",
			Valid:    true,
			Issues: []snmp.ValidationIssue{
				{Line: 2, Severity: "warning", Message: "test warning"},
			},
		},
	}

	resp := buildBatchValidationResponse(results, 3)

	if resp.Success {
		t.Error("expected success=false when some files invalid")
	}

	if resp.InvalidFiles != 1 {
		t.Errorf("expected 1 invalid file, got %d", resp.InvalidFiles)
	}

	if resp.TotalIssues != 2 {
		t.Errorf("expected 2 total issues, got %d", resp.TotalIssues)
	}

	if !strings.Contains(resp.Message, "1 of 3 walk files have issues") {
		t.Errorf("unexpected message: %s", resp.Message)
	}
}

func TestValidateWalkFilePath_EmptyFilename(t *testing.T) {
	server, _ := newTestServer(t)

	_, err := server.validateWalkFilePath("")

	if err == nil {
		t.Error("expected error for empty filename")
	}

	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected error to mention empty, got: %v", err)
	}
}

func TestValidateWalkFilePath_NullBytes(t *testing.T) {
	server, _ := newTestServer(t)

	_, err := server.validateWalkFilePath("test\x00.walk")

	if err == nil {
		t.Error("expected error for filename with null bytes")
	}
}

func TestValidateWalkFilePath_InvalidExtension(t *testing.T) {
	server, configPath := newTestServer(t)
	configDir := filepath.Dir(configPath)

	// Create a file with invalid extension
	invalidPath := filepath.Join(configDir, "test.exe")
	err := os.WriteFile(invalidPath, []byte("test"), 0o600)
	if err != nil {
		t.Fatalf("write test file: %v", err)
	}

	_, pathErr := server.validateWalkFilePath(filepath.Base(invalidPath))

	if pathErr == nil {
		t.Error("expected error for invalid extension")
	}

	if !strings.Contains(pathErr.Error(), "extension") {
		t.Errorf("expected error to mention extension, got: %v", pathErr)
	}
}

func TestValidateWalkFilePath_ValidExtensions(t *testing.T) {
	server, configPath := newTestServer(t)
	configDir := filepath.Dir(configPath)

	validExtensions := []string{".walk", ".snmpwalk", ".txt"}

	for _, ext := range validExtensions {
		filename := "test" + ext
		filePath := filepath.Join(configDir, filename)
		err := os.WriteFile(filePath, []byte("test content"), 0o600)
		if err != nil {
			t.Fatalf("write test file: %v", err)
		}

		_, pathErr := server.validateWalkFilePath(filename)
		if pathErr != nil {
			t.Errorf("expected no error for extension %s, got: %v", ext, pathErr)
		}
	}
}
