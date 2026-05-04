package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteErrorBasic(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)

	writeError(rec, req, http.StatusBadRequest, "bad_request", "Invalid input", nil)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	// Check Content-Type
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Error != "bad_request" {
		t.Errorf("error = %q, want %q", resp.Error, "bad_request")
	}
	if resp.Message != "Invalid input" {
		t.Errorf("message = %q, want %q", resp.Message, "Invalid input")
	}
	if resp.Path != "/api/v1/test" {
		t.Errorf("path = %q, want %q", resp.Path, "/api/v1/test")
	}
	if resp.Method != http.MethodGet {
		t.Errorf("method = %q, want %q", resp.Method, "GET")
	}
	if resp.Timestamp.IsZero() {
		t.Error("timestamp should not be zero")
	}
}

func TestWriteErrorWithValidationDetails(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", nil)

	details := []ErrorDetail{
		{Field: "name", Issue: "required field missing"},
		{Field: "ip", Issue: "invalid format", Value: "xyz"},
	}

	writeError(rec, req, http.StatusUnprocessableEntity, "validation_error", "Validation failed", details)

	var resp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Details) != 2 {
		t.Fatalf("details count = %d, want 2", len(resp.Details))
	}

	if resp.Details[0].Field != "name" {
		t.Errorf("first detail field = %q, want %q", resp.Details[0].Field, "name")
	}
	if resp.Details[1].Value != "xyz" {
		t.Errorf("second detail value = %q, want %q", resp.Details[1].Value, "xyz")
	}
}

func TestWriteErrorWithRequestID(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("X-Request-ID", "test-request-123")

	writeError(rec, req, http.StatusInternalServerError, "internal_error", "Something went wrong", nil)

	// Error should still be written correctly even with request ID
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestWriteErrorVariousStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		errorCode  string
	}{
		{"Unauthorized", http.StatusUnauthorized, "unauthorized"},
		{"Forbidden", http.StatusForbidden, "forbidden"},
		{"Not Found", http.StatusNotFound, "not_found"},
		{"Method Not Allowed", http.StatusMethodNotAllowed, "method_not_allowed"},
		{"Conflict", http.StatusConflict, "conflict"},
		{"Gone", http.StatusGone, "gone"},
		{"Too Many Requests", http.StatusTooManyRequests, "rate_limit_exceeded"},
		{"Internal Server Error", http.StatusInternalServerError, "internal_error"},
		{"Service Unavailable", http.StatusServiceUnavailable, "service_unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			writeError(rec, req, tt.statusCode, tt.errorCode, "Test message", nil)

			if rec.Code != tt.statusCode {
				t.Errorf("status = %d, want %d", rec.Code, tt.statusCode)
			}

			var resp ErrorResponse
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			if resp.Error != tt.errorCode {
				t.Errorf("error = %q, want %q", resp.Error, tt.errorCode)
			}
		})
	}
}

func TestErrorResponseStructure(t *testing.T) {
	// Verify ErrorResponse JSON tags
	resp := ErrorResponse{
		Error:     "test_error",
		Message:   "Test message",
		RequestID: "req-123",
		Path:      "/test",
		Method:    "GET",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// Unmarshal into map to verify JSON field names
	var m map[string]any
	if err = json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	expectedFields := []string{"error", "message", "request_id", "path", "method", "timestamp"}
	for _, field := range expectedFields {
		if _, ok := m[field]; !ok {
			t.Errorf("missing expected JSON field: %s", field)
		}
	}
}

func TestErrorDetailStructure(t *testing.T) {
	detail := ErrorDetail{
		Field: "email",
		Issue: "invalid format",
		Value: "not-an-email",
	}

	data, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var m map[string]any
	if err = json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if m["field"] != "email" {
		t.Errorf("field = %v, want %q", m["field"], "email")
	}
	if m["issue"] != "invalid format" {
		t.Errorf("issue = %v, want %q", m["issue"], "invalid format")
	}
	if m["value"] != "not-an-email" {
		t.Errorf("value = %v, want %q", m["value"], "not-an-email")
	}
}

func TestErrorDetailOmitEmpty(t *testing.T) {
	detail := ErrorDetail{
		Issue: "something wrong",
	}

	data, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var m map[string]any
	if err = json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	// Field and Value should be omitted when empty
	if _, ok := m["field"]; ok {
		t.Error("empty field should be omitted")
	}
	if _, ok := m["value"]; ok {
		t.Error("empty value should be omitted")
	}
}
