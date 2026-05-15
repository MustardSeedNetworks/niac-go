package api

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/krisarmstrong/niac-go/internal/safeconv"
)

// Sentinel errors for API devices.
var (
	ErrInvalidMACAddress = errors.New("invalid MAC address")
	ErrInvalidIPAddress  = errors.New("invalid IP address")
	errValidationFailed  = errors.New("validation failed")
)

const (
	// MaxDeviceCount is the maximum number of devices allowed.
	// SECURITY FIX #173: Prevents resource exhaustion via unbounded device creation.
	MaxDeviceCount = 1000

	// maxLabelLen is the max length of a hostname label (RFC 1123).
	maxLabelLen = 63
)

// validateHostname validates a device hostname per RFC 1123.
// SECURITY FIX #169: Prevents invalid or dangerous hostname values.
func validateHostname(hostname string) *ErrorDetail {
	if hostname == "" {
		return &ErrorDetail{Field: "hostname", Issue: "hostname is required"}
	}

	if len(hostname) > maxHostnameLen {
		return &ErrorDetail{
			Field: "hostname",
			Issue: fmt.Sprintf("hostname exceeds maximum length of %d characters", maxHostnameLen),
			Value: hostname[:min(truncateErrorValue, len(hostname))],
		}
	}

	if net.ParseIP(hostname) != nil {
		return &ErrorDetail{Field: "hostname", Issue: "hostname must not be an IP address", Value: hostname}
	}

	for label := range strings.SplitSeq(hostname, ".") {
		if issue := validateHostnameLabel(label); issue != "" {
			return &ErrorDetail{
				Field: "hostname",
				Issue: issue,
				Value: hostname[:min(truncateErrorValue, len(hostname))],
			}
		}
	}

	return nil
}

// validateHostnameLabel validates a single hostname label per RFC 1123.
func validateHostnameLabel(label string) string {
	if len(label) == 0 {
		return "hostname contains empty label (consecutive dots)"
	}
	if len(label) > maxLabelLen {
		return fmt.Sprintf("hostname label exceeds maximum length of %d characters", maxLabelLen)
	}
	if !isAlphanumeric(label[0]) {
		return "hostname labels must start with an alphanumeric character"
	}
	if !isAlphanumeric(label[len(label)-1]) {
		return "hostname labels must not end with a hyphen"
	}
	for _, c := range label {
		if !isAlphanumeric(safeconv.ByteFromRune(c)) && c != '-' {
			return "hostname contains invalid characters (only alphanumeric and hyphens allowed)"
		}
	}
	return ""
}

// isAlphanumeric checks if a byte is alphanumeric.
func isAlphanumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// YAML validation constants to prevent resource exhaustion attacks.
// SECURITY FIX #153: Limit YAML input size and complexity.
const (
	MaxYAMLSize  = 1024 * 1024 // 1MB max YAML input
	MaxYAMLDepth = 20          // Maximum nesting depth
)

// Device protocol collection constant.
const deviceProtocolCapacity = 10 // Expected max protocols per device

// validateYAMLInput checks YAML input for size and complexity limits.
// SECURITY FIX #153: Prevents memory exhaustion and DoS attacks.
func validateYAMLInput(input string) error {
	// Check size
	if len(input) > MaxYAMLSize {
		return fmt.Errorf("YAML input too large: %d bytes (max %d)", len(input), MaxYAMLSize)
	}

	// Check for empty input
	if strings.TrimSpace(input) == "" {
		return errors.New("YAML input is empty")
	}

	return nil
}

// checkYAMLDepth verifies parsed YAML doesn't exceed maximum nesting depth.
// SECURITY FIX #153: Prevents billion laughs and similar attacks.
func checkYAMLDepth(data any, currentDepth int) error {
	if currentDepth > MaxYAMLDepth {
		return fmt.Errorf("YAML nesting depth exceeds maximum of %d", MaxYAMLDepth)
	}

	switch v := data.(type) {
	case map[string]any:
		for _, val := range v {
			err := checkYAMLDepth(val, currentDepth+1)
			if err != nil {
				return err
			}
		}
	case []any:
		for _, item := range v {
			err := checkYAMLDepth(item, currentDepth+1)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
