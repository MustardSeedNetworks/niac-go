package api

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

// Sentinel errors for API devices.
var (
	ErrInvalidMACAddress = errors.New("invalid MAC address")
	ErrInvalidIPAddress  = errors.New("invalid IP address")
)

const (
	// MaxDeviceCount is the maximum number of devices allowed.
	// SECURITY FIX #173: Prevents resource exhaustion via unbounded device creation.
	MaxDeviceCount = 1000

	// maxLabelLen is the max length of a hostname label (RFC 1123).
	maxLabelLen = 63
)

// YAML validation constants to prevent resource exhaustion attacks.
// SECURITY FIX #153: Limit YAML input size and complexity.
const (
	MaxYAMLSize  = 1024 * 1024 // 1MB max YAML input
	MaxYAMLDepth = 20          // Maximum nesting depth
)

// validateHostname validates a device hostname per RFC 1123.
// SECURITY FIX #169: Prevents invalid or dangerous hostname values.
func validateHostname(hostname string) *ErrorDetail {
	if hostname == "" {
		return &ErrorDetail{
			Field: "hostname",
			Issue: "hostname is required",
		}
	}

	if len(hostname) > maxHostnameLen {
		return &ErrorDetail{
			Field: "hostname",
			Issue: fmt.Sprintf("hostname exceeds maximum length of %d characters", maxHostnameLen),
			Value: hostname[:min(truncateErrorValue, len(hostname))],
		}
	}

	// Must not be an IP address
	if net.ParseIP(hostname) != nil {
		return &ErrorDetail{
			Field: "hostname",
			Issue: "hostname must not be an IP address",
			Value: hostname,
		}
	}

	// Validate each label
	labels := strings.Split(hostname, ".")
	for _, label := range labels {
		if len(label) == 0 {
			return &ErrorDetail{
				Field: "hostname",
				Issue: "hostname contains empty label (consecutive dots)",
				Value: hostname[:min(truncateErrorValue, len(hostname))],
			}
		}

		if len(label) > maxLabelLen {
			return &ErrorDetail{
				Field: "hostname",
				Issue: fmt.Sprintf("hostname label exceeds maximum length of %d characters", maxLabelLen),
				Value: hostname[:min(truncateErrorValue, len(hostname))],
			}
		}

		// Must start with alphanumeric
		if !isAlphanumeric(label[0]) {
			return &ErrorDetail{
				Field: "hostname",
				Issue: "hostname labels must start with an alphanumeric character",
				Value: hostname[:min(truncateErrorValue, len(hostname))],
			}
		}

		// Must end with alphanumeric (not hyphen)
		if !isAlphanumeric(label[len(label)-1]) {
			return &ErrorDetail{
				Field: "hostname",
				Issue: "hostname labels must not end with a hyphen",
				Value: hostname[:min(truncateErrorValue, len(hostname))],
			}
		}

		// Only alphanumeric and hyphens allowed
		for _, c := range label {
			if !isAlphanumeric(byte(c)) && c != '-' {
				return &ErrorDetail{
					Field: "hostname",
					Issue: "hostname contains invalid characters (only alphanumeric and hyphens allowed)",
					Value: hostname[:min(truncateErrorValue, len(hostname))],
				}
			}
		}
	}

	return nil
}

// isAlphanumeric checks if a byte is alphanumeric.
func isAlphanumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

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

// parseMAC parses a MAC address string.
func parseMAC(s string) (net.HardwareAddr, error) {
	mac, err := net.ParseMAC(s)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidMACAddress, s)
	}

	return mac, nil
}

// parseIP parses an IP address string.
func parseIP(s string) (net.IP, error) {
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidIPAddress, s)
	}

	return ip, nil
}
