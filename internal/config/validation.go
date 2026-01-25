package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/krisarmstrong/niac-go/internal/converter"
)

// Sentinel errors for configuration validation.
var (
	ErrInvalidDeviceDeclaration = errors.New("invalid device declaration")
	ErrNoDevicesDefined         = errors.New("no devices defined in configuration")
	ErrInvalidMapToIP           = errors.New("invalid map_to_ip")
	ErrInvalidTTLIP             = errors.New("invalid ttl ip")
	ErrInvalidTTLMask           = errors.New("invalid ttl mask")
	ErrInvalidIPAddress         = errors.New("invalid IP address")
	ErrInvalidSNMPAddr          = errors.New("invalid snmp_addr")
	ErrDNSTTLNegative           = errors.New("DNS record TTL cannot be negative")
	ErrDNSTTLExceedsMax         = errors.New("DNS record TTL exceeds maximum (2147483647)")
	ErrInsufficientFields       = errors.New("insufficient fields")
	ErrPathTraversalDetected    = errors.New("path traversal detected")
	ErrPathOutsideBaseDir       = errors.New("path outside base directory")
	ErrWalkFileNotFound         = errors.New("walk file not found")
	ErrWalkFileIsSymlink        = errors.New("walk file is a symlink (not allowed)")
	ErrWalkFileNotRegular       = errors.New("walk file is not a regular file")
)

// validateYAMLConfig validates a YAML configuration.
func validateYAMLConfig(yamlConfig *converter.Config) (*converter.Config, error) {
	err := converter.ValidateConfig(yamlConfig)
	if err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return yamlConfig, nil
}

// validateWalkFilePath validates and resolves SNMP walk file paths
// Prevents path traversal attacks and ensures file exists.
func validateWalkFilePath(basePath, walkFile, deviceName string) (string, error) {
	// Clean the path to normalize it FIRST
	cleanPath := filepath.Clean(walkFile)

	// Security: Check for traversal AFTER cleaning
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("device %s: %w: %s", deviceName, ErrPathTraversalDetected, walkFile)
	}

	// Build full path
	var fullPath string

	switch {
	case filepath.IsAbs(cleanPath):
		fullPath = cleanPath
	case basePath != "":
		fullPath = filepath.Join(basePath, cleanPath)
	default:
		fullPath = cleanPath
	}

	// CRITICAL: Verify resolved path stays within base directory
	if basePath != "" {
		absBase, err := filepath.Abs(basePath)
		if err != nil {
			return "", fmt.Errorf("device %s: invalid base path: %w", deviceName, err)
		}

		absFull, err := filepath.Abs(fullPath)
		if err != nil {
			return "", fmt.Errorf("device %s: invalid file path: %w", deviceName, err)
		}

		// Ensure path starts with base (add separator to prevent partial match)
		if !strings.HasPrefix(absFull+string(filepath.Separator), absBase+string(filepath.Separator)) {
			return "", fmt.Errorf("device %s: %w: %s", deviceName, ErrPathOutsideBaseDir, walkFile)
		}
	}

	// Use Lstat to detect symlinks (doesn't follow them)
	info, err := os.Lstat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("device %s: %w: %s", deviceName, ErrWalkFileNotFound, fullPath)
		}

		return "", fmt.Errorf("device %s: cannot access walk file %s: %w", deviceName, fullPath, err)
	}

	// Reject symlinks for security
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("device %s: %w: %s", deviceName, ErrWalkFileIsSymlink, fullPath)
	}

	// Verify it's a regular file, not a directory or device
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("device %s: %w: %s", deviceName, ErrWalkFileNotRegular, fullPath)
	}

	return fullPath, nil
}

// validateDNSTTL validates and returns the DNS TTL value.
func validateDNSTTL(ttl int, deviceName string) (uint32, error) {
	if ttl <= 0 {
		return uint32(DefaultDNSTTL), nil
	}

	if ttl < 0 {
		return 0, fmt.Errorf("device %s: %w: %d", deviceName, ErrDNSTTLNegative, ttl)
	}

	if ttl > maxDNSTTL {
		return 0, fmt.Errorf("device %s: %w: %d", deviceName, ErrDNSTTLExceedsMax, ttl)
	}

	return uint32(ttl), nil
}
