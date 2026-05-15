package config

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

// LoadLegacy loads a legacy key-value configuration file
// Format: device <name> { key = value ... }.
func LoadLegacy(filename string) (*Config, error) {
	cleaned := filepath.Clean(filename)
	if strings.Contains(cleaned, "..") {
		return nil, fmt.Errorf("config path must not contain path traversal: %s", filename)
	}
	file, err := os.Open(cleaned)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}

	defer func() { _ = file.Close() }()

	cfg, scanErr := scanLegacyConfig(file)
	if scanErr != nil {
		return nil, scanErr
	}

	if len(cfg.Devices) == 0 {
		return nil, ErrNoDevicesDefined
	}

	return cfg, nil
}

// scanLegacyConfig reads all lines from a legacy config file and populates the config.
func scanLegacyConfig(file *os.File) (*Config, error) {
	cfg := &Config{
		Devices: make([]Device, 0),
	}

	var currentDevice *Device

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if isCommentOrEmpty(line) {
			continue
		}

		if strings.HasPrefix(line, "device ") {
			device, parseErr := parseDeviceDeclaration(line, lineNum)
			if parseErr != nil {
				return nil, parseErr
			}

			cfg.Devices = append(cfg.Devices, device)
			currentDevice = &cfg.Devices[len(cfg.Devices)-1]

			continue
		}

		if currentDevice == nil {
			continue
		}

		if strings.HasPrefix(line, "}") {
			currentDevice = nil

			continue
		}

		parseLegacyKeyValue(line, currentDevice)
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return nil, fmt.Errorf("error reading config file: %w", scanErr)
	}

	return cfg, nil
}

// isCommentOrEmpty checks if a line is empty or a comment.
func isCommentOrEmpty(line string) bool {
	return line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//")
}

// parseDeviceDeclaration parses a device declaration line.
func parseDeviceDeclaration(line string, lineNum int) (Device, error) {
	parts := strings.Fields(line)
	if len(parts) < minDeviceDeclParts {
		return Device{}, fmt.Errorf("%w: line %d", ErrInvalidDeviceDeclaration, lineNum)
	}

	return Device{
		Name:       parts[1],
		Interfaces: make([]Interface, 0),
		Properties: make(map[string]string),
	}, nil
}

// parseLegacyKeyValue parses a key=value pair and applies it to the device.
func parseLegacyKeyValue(line string, device *Device) {
	parts := strings.SplitN(line, "=", keyValueParts)
	if len(parts) != keyValueParts {
		fmt.Fprintf(os.Stderr, "Warning: ignoring invalid line in legacy config (no '=' found): %s\n", line)
		return
	}

	key := strings.TrimSpace(parts[0])
	value := strings.Trim(strings.TrimSpace(parts[1]), "\"")

	applyLegacyDeviceProperty(device, key, value)
}

// applyLegacyDeviceProperty applies a parsed key-value property to a device.
func applyLegacyDeviceProperty(device *Device, key, value string) {
	switch key {
	case "type":
		device.Type = value
	case ChassisIDTypeMAC:
		if mac, err := net.ParseMAC(value); err == nil {
			device.MACAddress = mac
		}
	case "ip", "ipv6":
		if ip := net.ParseIP(value); ip != nil {
			device.IPAddresses = append(device.IPAddresses, ip)
		}
	case "snmp_community":
		device.SNMPConfig.Community = value
	case "sysName":
		device.SNMPConfig.SysName = value
	case "sysDescr":
		device.SNMPConfig.SysDescr = value
	case "sysContact":
		device.SNMPConfig.SysContact = value
	case "sysLocation":
		device.SNMPConfig.SysLocation = value
	case "walk":
		device.SNMPConfig.WalkFile = value
	default:
		device.Properties[key] = value
	}
}
