// Package config provides configuration file loading and parsing for network device simulation
package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Load reads and parses a configuration file
// Automatically detects format based on file extension:
// - .yaml -> YAML format (converted from Java DSL)
// - .cfg, .conf, or other -> legacy key-value format.
func Load(filename string) (*Config, error) {
	ext := filepath.Ext(filename)

	// Route to YAML loader for .yaml files
	if ext == ".yaml" || ext == ".yml" {
		return LoadYAML(filename)
	}

	// Route to legacy format loader
	return LoadLegacy(filename)
}

// GetDeviceByMAC finds a device by MAC address.
func (c *Config) GetDeviceByMAC(mac net.HardwareAddr) *Device {
	for i := range c.Devices {
		if c.Devices[i].MACAddress.String() == mac.String() {
			return &c.Devices[i]
		}
	}

	return nil
}

// GetDeviceByIP finds a device by IP address.
func (c *Config) GetDeviceByIP(ip net.IP) *Device {
	for i := range c.Devices {
		for _, deviceIP := range c.Devices[i].IPAddresses {
			if deviceIP.Equal(ip) {
				return &c.Devices[i]
			}
		}
	}

	return nil
}

// ParseSimpleConfig parses a flat "name type ip mac [walkfile]" device list — one device per line.
func ParseSimpleConfig(lines []string) (*Config, error) {
	cfg := &Config{
		Devices: make([]Device, 0),
	}

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < minSimpleConfigParts {
			return nil, fmt.Errorf("%w: line %d", ErrInsufficientFields, lineNum+1)
		}

		mac, err := net.ParseMAC(parts[3])
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid MAC address: %w", lineNum+1, err)
		}

		ip := net.ParseIP(parts[2])
		if ip == nil {
			return nil, fmt.Errorf("%w: line %d", ErrInvalidIPAddress, lineNum+1)
		}

		device := Device{
			Name:        parts[0],
			Type:        parts[1],
			MACAddress:  mac,
			IPAddresses: []net.IP{ip},
			Properties:  make(map[string]string),
			SNMPConfig: SNMPConfig{
				Community: "public",
				SysName:   parts[0],
			},
		}

		if len(parts) >= simpleConfigWithWalk {
			device.SNMPConfig.WalkFile = parts[4]
		}

		cfg.Devices = append(cfg.Devices, device)
	}

	return cfg, nil
}

// GenerateMAC generates a random MAC address.
func GenerateMAC() net.HardwareAddr {
	mac := make(net.HardwareAddr, macAddressBytes)
	// Set locally administered bit
	mac[0] = 0x02
	for i := 1; i < 6; i++ {
		mac[i] = byte(i * macPatternMultiplier) // Simple pattern for testing
	}

	return mac
}

// validateWalkFilePath validates and resolves SNMP walk file paths

// Prevents path traversal attacks and ensures file exists.
func validateWalkFilePath(basePath, walkFile, deviceName string) (string, error) {
	cleanPath := filepath.Clean(walkFile)

	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("device %s: %w: %s", deviceName, ErrPathTraversalDetected, walkFile)
	}

	fullPath := resolveWalkFilePath(basePath, cleanPath)

	if err := verifyPathWithinBase(basePath, fullPath, walkFile, deviceName); err != nil {
		return "", err
	}

	return verifyWalkFileAccess(fullPath, deviceName)
}

// resolveWalkFilePath resolves the full path of a walk file from its base and clean path.
func resolveWalkFilePath(basePath, cleanPath string) string {
	switch {
	case filepath.IsAbs(cleanPath):
		return cleanPath
	case basePath != "":
		return filepath.Join(basePath, cleanPath)
	default:
		return cleanPath
	}
}

// verifyPathWithinBase ensures the resolved path stays within the base directory.
func verifyPathWithinBase(basePath, fullPath, walkFile, deviceName string) error {
	if basePath == "" {
		return nil
	}

	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return fmt.Errorf("device %s: invalid base path: %w", deviceName, err)
	}

	absFull, err := filepath.Abs(fullPath)
	if err != nil {
		return fmt.Errorf("device %s: invalid file path: %w", deviceName, err)
	}

	if !strings.HasPrefix(absFull+string(filepath.Separator), absBase+string(filepath.Separator)) {
		return fmt.Errorf("device %s: %w: %s", deviceName, ErrPathOutsideBaseDir, walkFile)
	}

	return nil
}

// verifyWalkFileAccess checks that the walk file exists, is not a symlink, and is a regular file.
func verifyWalkFileAccess(fullPath, deviceName string) (string, error) {
	// Inline barrier: callers (validateWalkFilePath / verifyPathWithinBase)
	// have already cleaned and prefix-checked the path, but make the
	// constraint explicit at the Lstat sink for static analysers.
	cleanedFull := filepath.Clean(fullPath)
	if strings.Contains(cleanedFull, "..") {
		return "", fmt.Errorf("device %s: %w: %s", deviceName, ErrPathTraversalDetected, fullPath)
	}
	info, err := os.Lstat(cleanedFull)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("device %s: %w: %s", deviceName, ErrWalkFileNotFound, fullPath)
		}

		return "", fmt.Errorf("device %s: cannot access walk file %s: %w", deviceName, fullPath, err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("device %s: %w: %s", deviceName, ErrWalkFileIsSymlink, fullPath)
	}

	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("device %s: %w: %s", deviceName, ErrWalkFileNotRegular, fullPath)
	}

	return fullPath, nil
}

// ParseSpeed parses interface speed (e.g., "100M", "1G", "10G").
func ParseSpeed(speedStr string) (int, error) {
	speedStr = strings.ToUpper(strings.TrimSpace(speedStr))

	if val, found := strings.CutSuffix(speedStr, "G"); found {
		num, err := strconv.Atoi(val)
		if err != nil {
			return 0, fmt.Errorf("failed to parse speed value: %w", err)
		}

		return num * gbpsToMbps, nil // Convert to Mbps
	}

	if val, found := strings.CutSuffix(speedStr, "M"); found {
		num, err := strconv.Atoi(val)
		if err != nil {
			return 0, fmt.Errorf("failed to parse speed value: %w", err)
		}
		return num, nil
	}

	num, err := strconv.Atoi(speedStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse speed value: %w", err)
	}
	return num, nil
}
