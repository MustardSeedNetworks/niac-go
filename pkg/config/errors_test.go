package config

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestConfigError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ConfigError
		expected string
	}{
		{
			name: "with line and column",
			err: &ConfigError{
				File:    "config.yaml",
				Line:    45,
				Column:  12,
				Message: "invalid value",
			},
			expected: "config.yaml:45:12: invalid value",
		},
		{
			name: "with line only",
			err: &ConfigError{
				File:    "config.yaml",
				Line:    45,
				Message: "invalid value",
			},
			expected: "config.yaml:45: invalid value",
		},
		{
			name: "file only",
			err: &ConfigError{
				File:    "config.yaml",
				Message: "invalid value",
			},
			expected: "config.yaml: invalid value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestConfigError_Format(t *testing.T) {
	err := &ConfigError{
		File:       "config.yaml",
		Line:       45,
		Column:     12,
		Field:      "threshold",
		Message:    "Invalid value for 'threshold'",
		Expected:   "integer (0-100)",
		Got:        "string 'ninety'",
		Suggestion: "Use a number: threshold: 90",
		Severity:   SeverityError,
	}

	formatted := err.Format()

	// Check that formatted output contains key elements
	if !strings.Contains(formatted, "✗") {
		t.Error("Formatted error should contain error icon")
	}
	if !strings.Contains(formatted, "config.yaml:45:12") {
		t.Error("Formatted error should contain file location")
	}
	if !strings.Contains(formatted, "Invalid value for 'threshold'") {
		t.Error("Formatted error should contain message")
	}
	if !strings.Contains(formatted, "Expected: integer (0-100)") {
		t.Error("Formatted error should contain expected value")
	}
	if !strings.Contains(formatted, "Got: string 'ninety'") {
		t.Error("Formatted error should contain actual value")
	}
	if !strings.Contains(formatted, "💡 Suggestion") {
		t.Error("Formatted error should contain suggestion")
	}
}

func TestConfigErrorList_Add(t *testing.T) {
	list := &ConfigErrorList{File: "config.yaml", Valid: true}

	// Add an error
	err := &ConfigError{
		File:     "config.yaml",
		Message:  "error message",
		Severity: SeverityError,
	}
	list.Add(err)

	if list.Valid {
		t.Error("Valid should be false after adding error")
	}
	if len(list.Errors) != 1 {
		t.Errorf("Errors length = %d, want 1", len(list.Errors))
	}

	// Add a warning
	warn := &ConfigError{
		File:     "config.yaml",
		Message:  "warning message",
		Severity: SeverityWarning,
	}
	list.Add(warn)

	if len(list.Warnings) != 1 {
		t.Errorf("Warnings length = %d, want 1", len(list.Warnings))
	}
	// Valid should still be false
	if list.Valid {
		t.Error("Valid should remain false")
	}
}

func TestConfigErrorList_HasErrors(t *testing.T) {
	list := &ConfigErrorList{File: "config.yaml"}

	if list.HasErrors() {
		t.Error("HasErrors() should be false for empty list")
	}

	list.Add(&ConfigError{Severity: SeverityWarning, Message: "warning"})
	if list.HasErrors() {
		t.Error("HasErrors() should be false for warnings only")
	}

	list.Add(&ConfigError{Severity: SeverityError, Message: "error"})
	if !list.HasErrors() {
		t.Error("HasErrors() should be true after adding error")
	}
}

func TestConfigErrorList_HasWarnings(t *testing.T) {
	list := &ConfigErrorList{File: "config.yaml"}

	if list.HasWarnings() {
		t.Error("HasWarnings() should be false for empty list")
	}

	list.Add(&ConfigError{Severity: SeverityWarning, Message: "warning"})
	if !list.HasWarnings() {
		t.Error("HasWarnings() should be true after adding warning")
	}
}

func TestConfigErrorList_ToJSON(t *testing.T) {
	list := &ConfigErrorList{
		File:  "config.yaml",
		Valid: false,
	}
	list.Add(&ConfigError{
		File:     "config.yaml",
		Line:     45,
		Message:  "test error",
		Severity: SeverityError,
	})

	jsonStr, err := list.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// Parse JSON to verify it's valid
	var parsed ConfigErrorList
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	if parsed.File != "config.yaml" {
		t.Errorf("Parsed file = %q, want %q", parsed.File, "config.yaml")
	}
	if len(parsed.Errors) != 1 {
		t.Errorf("Parsed errors length = %d, want 1", len(parsed.Errors))
	}
	if parsed.Valid {
		t.Error("Parsed valid should be false")
	}
}

func TestNewConfigError(t *testing.T) {
	err := NewConfigError("config.yaml", "threshold", "invalid value")

	if err.File != "config.yaml" {
		t.Errorf("File = %q, want %q", err.File, "config.yaml")
	}
	if err.Field != "threshold" {
		t.Errorf("Field = %q, want %q", err.Field, "threshold")
	}
	if err.Message != "invalid value" {
		t.Errorf("Message = %q, want %q", err.Message, "invalid value")
	}
	if err.Severity != SeverityError {
		t.Errorf("Severity = %q, want %q", err.Severity, SeverityError)
	}
}

func TestNewConfigWarning(t *testing.T) {
	warn := NewConfigWarning("config.yaml", "option", "deprecated")

	if warn.Severity != SeverityWarning {
		t.Errorf("Severity = %q, want %q", warn.Severity, SeverityWarning)
	}
}

func TestConfigErrorList_Format(t *testing.T) {
	list := &ConfigErrorList{File: "config.yaml"}
	list.Add(&ConfigError{
		File:     "config.yaml",
		Message:  "error 1",
		Severity: SeverityError,
	})
	list.Add(&ConfigError{
		File:     "config.yaml",
		Message:  "warning 1",
		Severity: SeverityWarning,
	})

	formatted := list.Format()

	if !strings.Contains(formatted, "✗ Configuration errors found") {
		t.Error("Should contain error header")
	}
	if !strings.Contains(formatted, "⚠ Configuration warnings") {
		t.Error("Should contain warning header")
	}
	if !strings.Contains(formatted, "Summary: 1 error(s), 1 warning(s)") {
		t.Error("Should contain summary")
	}
}
