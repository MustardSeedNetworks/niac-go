package config

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ErrorSeverity represents the severity level of a configuration error.
type ErrorSeverity string

const (
	// SeverityError indicates a critical error that prevents configuration loading.
	SeverityError ErrorSeverity = "error"
	// SeverityWarning indicates a potential issue that doesn't prevent loading.
	SeverityWarning ErrorSeverity = "warning"
	// SeverityInfo indicates informational messages.
	SeverityInfo ErrorSeverity = "info"
)

// Error represents a structured configuration error with context.
type Error struct {
	File       string        `json:"file"`                 // Configuration file path
	Line       int           `json:"line"`                 // Line number (0 if unknown)
	Column     int           `json:"column"`               // Column number (0 if unknown)
	Field      string        `json:"field"`                // Field name that has error
	Message    string        `json:"message"`              // Error message
	Expected   string        `json:"expected,omitempty"`   // Expected value/type
	Got        string        `json:"got,omitempty"`        // Actual value/type received
	Suggestion string        `json:"suggestion,omitempty"` // Fix suggestion
	Severity   ErrorSeverity `json:"severity"`             // Error severity
}

// NewConfigError creates a new configuration error.
func NewConfigError(file, field, message string) *Error {
	return &Error{
		File:     file,
		Field:    field,
		Message:  message,
		Severity: SeverityError,
	}
}

// NewConfigWarning creates a new configuration warning.
func NewConfigWarning(file, field, message string) *Error {
	return &Error{
		File:     file,
		Field:    field,
		Message:  message,
		Severity: SeverityWarning,
	}
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Line > 0 && e.Column > 0 {
		return fmt.Sprintf("%s:%d:%d: %s", e.File, e.Line, e.Column, e.Message)
	}

	if e.Line > 0 {
		return fmt.Sprintf("%s:%d: %s", e.File, e.Line, e.Message)
	}

	return fmt.Sprintf("%s: %s", e.File, e.Message)
}

// Format returns a beautifully formatted error message for terminal display.
func (e *Error) Format() string {
	var b strings.Builder

	// Error icon and location
	switch e.Severity {
	case SeverityError:
		b.WriteString("✗ ")
	case SeverityWarning:
		b.WriteString("⚠ ")
	case SeverityInfo:
		b.WriteString("ℹ ")
	}

	b.WriteString("Configuration ")
	b.WriteString(string(e.Severity))

	if e.File != "" {
		b.WriteString(" in ")
		b.WriteString(e.File)
	}

	b.WriteString("\n\n")

	// Location indicator
	if e.Line > 0 {
		if e.Column > 0 {
			b.WriteString(fmt.Sprintf("%s:%d:%d\n", e.File, e.Line, e.Column))
		} else {
			b.WriteString(fmt.Sprintf("%s:%d\n", e.File, e.Line))
		}

		b.WriteString("  |\n")
		b.WriteString(fmt.Sprintf("%2d| ", e.Line))

		// Would need actual file content here to show the line
		// For now, just show field if available
		if e.Field != "" {
			b.WriteString("    ")
			b.WriteString(e.Field)
		}

		b.WriteString("\n")

		if e.Column > 0 {
			b.WriteString("  | ")
			b.WriteString(strings.Repeat(" ", e.Column))
			b.WriteString("^\n")
		}

		b.WriteString("  |\n")
	}

	// Error message
	b.WriteString(fmt.Sprintf("Error: %s\n", e.Message))

	// Expected vs Got
	if e.Expected != "" {
		b.WriteString(fmt.Sprintf("  Expected: %s\n", e.Expected))
	}

	if e.Got != "" {
		b.WriteString(fmt.Sprintf("  Got: %s\n", e.Got))
	}

	// Suggestion
	if e.Suggestion != "" {
		b.WriteString(fmt.Sprintf("\n💡 Suggestion: %s\n", e.Suggestion))
	}

	return b.String()
}

// ListError holds multiple configuration errors.
type ListError struct {
	File     string   `json:"file"`
	Errors   []*Error `json:"errors"`
	Warnings []*Error `json:"warnings,omitempty"`
	Valid    bool     `json:"valid"`
}

// Error implements the error interface for ConfigListError.
func (l *ListError) Error() string {
	if len(l.Errors) == 0 {
		return "no errors"
	}

	if len(l.Errors) == 1 {
		return l.Errors[0].Error()
	}

	return fmt.Sprintf("%d configuration errors found", len(l.Errors))
}

// Add adds an error to the list.
func (l *ListError) Add(err *Error) {
	switch err.Severity {
	case SeverityError:
		l.Errors = append(l.Errors, err)
		l.Valid = false
	case SeverityWarning:
		l.Warnings = append(l.Warnings, err)
	case SeverityInfo:
		// Info messages are not tracked in error or warning lists
	}
}

// HasErrors returns true if there are any errors (not warnings).
func (l *ListError) HasErrors() bool {
	return len(l.Errors) > 0
}

// HasWarnings returns true if there are any warnings.
func (l *ListError) HasWarnings() bool {
	return len(l.Warnings) > 0
}

// Format returns a formatted string with all errors and warnings.
func (l *ListError) Format() string {
	var b strings.Builder

	if len(l.Errors) > 0 {
		b.WriteString(fmt.Sprintf("✗ Configuration errors found: %s\n\n", l.File))

		for i, err := range l.Errors {
			if i > 0 {
				b.WriteString("\n")
			}

			b.WriteString(err.Format())
		}
	}

	if len(l.Warnings) > 0 {
		if len(l.Errors) > 0 {
			b.WriteString("\n")
		}

		b.WriteString(fmt.Sprintf("⚠ Configuration warnings: %s\n\n", l.File))

		for i, warn := range l.Warnings {
			if i > 0 {
				b.WriteString("\n")
			}

			b.WriteString(warn.Format())
		}
	}

	// Summary
	b.WriteString(fmt.Sprintf("\nSummary: %d error(s), %d warning(s)\n",
		len(l.Errors), len(l.Warnings)))

	return b.String()
}

// ToJSON converts the error list to JSON format for CI/CD integration.
func (l *ListError) ToJSON() (string, error) {
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal errors to JSON: %w", err)
	}

	return string(data), nil
}
