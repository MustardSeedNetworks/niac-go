package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// ErrorResponse represents a standardized API error response
// FEATURE #105: Consistent error format for all API endpoints.
type ErrorResponse struct {
	Error     string        `json:"error"`               // Machine-readable error code
	Message   string        `json:"message"`             // Human-readable error message
	Details   []ErrorDetail `json:"details,omitempty"`   // Optional detailed error information
	RequestID string        `json:"requestId,omitempty"` // Optional request ID for tracing
	Timestamp time.Time     `json:"timestamp"`           // When the error occurred
	Path      string        `json:"path"`                // Request path that caused the error
	Method    string        `json:"method"`              // HTTP method
}

// ErrorDetail provides detailed information about a specific error.
type ErrorDetail struct {
	Code   string `json:"code,omitempty"`   // Stable finding code, when the source defines one (fabric.DiagnosticCode)
	Field  string `json:"field,omitempty"`  // Field name that caused the error
	Issue  string `json:"issue"`            // Description of the issue
	Value  string `json:"value,omitempty"`  // The value that caused the error (sanitized)
	Line   int    `json:"line,omitempty"`   // 1-based source line, when known (e.g. YAML parse errors)
	Column int    `json:"column,omitempty"` // 1-based source column, when known
}

// writeError writes a standardized error response.
func writeError(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	errorCode, message string,
	details []ErrorDetail,
) {
	// The registrar assigns X-Request-ID; the body quotes it so a client
	// report can be joined to the log line below.
	requestID := r.Header.Get("X-Request-ID")
	response := ErrorResponse{
		Error:     errorCode,
		Message:   message,
		Details:   details,
		RequestID: requestID,
		Timestamp: time.Now(),
		Path:      r.URL.Path,
		Method:    r.Method,
	}

	if requestID != "" {
		logger := slog.Default()
		logger.ErrorContext(r.Context(),
			"[API] Error response",
			"requestID",
			requestID,
			"status",
			status,
			"errorCode",
			errorCode,
			"message",
			message,
		)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response) // HTTP write errors are non-critical
}

// writeJSON writes a JSON response with pretty-printing.
func (s *Server) writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(payload)
}
