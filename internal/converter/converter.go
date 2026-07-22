// Package converter implements Java NIAC DSL to YAML conversion
package converter

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// validateNoTraversal cleans path and rejects any residual ".." element,
// which would indicate an attempt to escape the caller's intended
// directory via a relative traversal sequence (e.g. "../../etc/passwd").
// It is the single sanitizer shared by every file-path entry point in
// this package — ConvertFile's input/output paths and LoadYAMLConfig's
// filename all route through it before touching the filesystem.
func validateNoTraversal(label, path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("%s path cannot be empty", label)
	}
	clean := filepath.Clean(path)
	if strings.Contains(clean, "..") {
		return "", fmt.Errorf("%s path must not contain path traversal", label)
	}
	return clean, nil
}

// ConvertFile converts a Java DSL config file to YAML.
func ConvertFile(inputPath, outputPath string, verbose bool) error {
	cleanInput, err := validateNoTraversal("input", inputPath)
	if err != nil {
		return err
	}

	// Check file size before reading to prevent memory exhaustion
	info, err := os.Stat(cleanInput)
	if err != nil {
		return fmt.Errorf("error accessing input file: %w", err)
	}
	if info.Size() > maxInputFileSize {
		return fmt.Errorf("input file too large: %d bytes (max %d)", info.Size(), maxInputFileSize)
	}

	cleanOutput, err := validateNoTraversal("output", outputPath)
	if err != nil {
		return err
	}

	// Read input file
	data, err := os.ReadFile(cleanInput)
	if err != nil {
		return fmt.Errorf("error reading input file: %w", err)
	}

	// Parse Java DSL
	parser := &Parser{
		lines:   strings.Split(string(data), "\n"),
		pos:     0,
		verbose: verbose,
	}

	config, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("error parsing config: %w", err)
	}

	// Convert to YAML
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshaling YAML: %w", err)
	}

	// Write output file
	if writeErr := os.WriteFile(cleanOutput, yamlData, 0o600); writeErr != nil {
		return fmt.Errorf("error writing output file: %w", writeErr)
	}

	return nil
}

// Parse parses the Java DSL format.
func (p *Parser) Parse() (*Config, error) {
	config := &Config{}
	deviceCount := 0

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			p.pos++

			continue
		}

		// Parse directives
		if strings.HasPrefix(line, "IncludePath(") {
			if path := p.extractString(line); path != "" {
				config.IncludePath = path
			}
			p.pos++

			continue
		}

		if strings.HasPrefix(line, "CapturePlayback(") {
			playback, err := p.parseCapturePlayback()
			if err != nil {
				return nil, err
			}
			config.CapturePlaybacks = append(config.CapturePlaybacks, *playback)

			continue
		}

		if strings.HasPrefix(line, "Device(") {
			device, err := p.parseDevice(deviceCount)
			if err != nil {
				return nil, err
			}
			config.Devices = append(config.Devices, *device)
			deviceCount++

			continue
		}

		p.pos++
	}

	return config, nil
}

const MaxYAMLConfigSize = 16 * 1024 * 1024

// LoadYAMLConfig loads a YAML config file into Go config structure.
func LoadYAMLConfig(filename string) (*Config, error) {
	clean, err := validateNoTraversal("YAML config", filename)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(clean)
	if err != nil {
		return nil, fmt.Errorf("error accessing YAML file: %w", err)
	}
	if info.Size() > MaxYAMLConfigSize {
		return nil, fmt.Errorf(
			"YAML config too large: %d bytes (max %d)", info.Size(), MaxYAMLConfigSize)
	}

	data, err := os.ReadFile(clean)
	if err != nil {
		return nil, fmt.Errorf("error reading YAML file: %w", err)
	}

	return LoadYAMLConfigFromBytes(data)
}

// LoadYAMLConfigFromBytes converts in-memory YAML data into a Go config
// structure. Returns the parsed Config and any validation error so callers
// that need the partially-loaded Config (e.g., for diagnostics) can still
// access it.
func LoadYAMLConfigFromBytes(data []byte) (*Config, error) {
	if len(data) > MaxYAMLConfigSize {
		return nil, fmt.Errorf(
			"YAML config too large: %d bytes (max %d)", len(data), MaxYAMLConfigSize)
	}

	var config Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	err := decoder.Decode(&config)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("error parsing YAML: %w", err)
	}
	var extra yaml.Node
	if extraErr := decoder.Decode(&extra); !errors.Is(extraErr, io.EOF) {
		if extraErr != nil {
			return nil, fmt.Errorf("error parsing YAML: %w", extraErr)
		}
		return nil, errors.New("error parsing YAML: multiple documents are not allowed")
	}

	if validateErr := ValidateConfig(&config); validateErr != nil {
		return &config, validateErr
	}

	return &config, nil
}

// ValidateConfig validates a Config. The legacy sentinel-error checks
// (ErrDeviceMissingMAC etc.) run first so existing callers that use
// errors.Is(err, ErrDeviceMissingMAC) keep working; the struct-tag validator
// then catches everything else (IP formats, VLAN range, enum membership,
// mutual exclusion of ip and ips).
func ValidateConfig(config *Config) error {
	for i, device := range config.Devices {
		if err := validateDeviceConfig(device, i); err != nil {
			return err
		}
	}
	for segmentIndex, segment := range config.Segments {
		for deviceIndex, device := range segment.Devices {
			if err := validateDeviceConfig(device, deviceIndex); err != nil {
				return fmt.Errorf("segment %d: %w", segmentIndex, err)
			}
		}
	}

	// If capture playbacks specified, validate them
	for i, playback := range config.CapturePlaybacks {
		if playback.FileName == "" {
			return fmt.Errorf("%w: index %d", ErrCapturePlaybackMissingFile, i)
		}
	}

	return validateConfigStruct(config)
}

func validateDeviceConfig(device Device, index int) error {
	vendor := strings.TrimSpace(device.Vendor)
	if device.MAC == "" && vendor == "" {
		return fmt.Errorf("%w: device %d", ErrDeviceMissingMAC, index)
	}
	if device.MAC != "" && vendor != "" {
		return fmt.Errorf("%w: device %d", ErrDeviceMACSourceConflict, index)
	}
	if err := validateSSHConfig(device.SSH); err != nil {
		return fmt.Errorf("device %d: %w", index, err)
	}
	if err := validateSyslogConfig(device.Syslog); err != nil {
		return fmt.Errorf("device %d: %w", index, err)
	}
	if device.SnmpAgent == nil {
		return nil
	}
	for mibIndex, mib := range device.SnmpAgent.AddMibs {
		if mib.OID == "" {
			return fmt.Errorf("%w: device %d AddMib %d", ErrAddMibMissingOID, index, mibIndex)
		}
		if mib.Type == "" {
			return fmt.Errorf("%w: device %d AddMib %d", ErrAddMibMissingType, index, mibIndex)
		}
	}

	return nil
}

func validateSSHConfig(config *SSHConfig) error {
	if config == nil || !config.Enabled {
		return nil
	}
	if strings.TrimSpace(config.Username) == "" {
		return errors.New("ssh.username is required when SSH is enabled")
	}
	valid, _ := regexp.MatchString(`^[A-Za-z_][A-Za-z0-9_]*$`, config.PasswordEnv)
	if !valid {
		return errors.New("ssh.password_env must name an environment variable when SSH is enabled")
	}
	return nil
}

func validateSyslogConfig(config *SyslogConfig) error {
	if config == nil || !config.Enabled {
		return nil
	}
	if len(config.Receivers) == 0 {
		return errors.New("syslog.receivers is required when SYSLOG is enabled")
	}
	for _, receiver := range config.Receivers {
		host, port, err := net.SplitHostPort(receiver)
		address, addressErr := netip.ParseAddr(host)
		if err != nil || addressErr != nil || !address.Is4() {
			return fmt.Errorf("invalid SYSLOG receiver %q", receiver)
		}
		value, err := strconv.ParseUint(port, 10, 16)
		if !validPortSyntax(port) || err != nil || value == 0 {
			return fmt.Errorf("invalid SYSLOG receiver port in %q", receiver)
		}
	}
	return nil
}

func validPortSyntax(port string) bool {
	if len(port) == 0 || len(port) > 5 || port[0] == '0' {
		return false
	}
	for _, character := range port {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

// PrintSummary prints a summary of the config.
func PrintSummary(config *Config, w *bufio.Writer) {
	_, _ = fmt.Fprintf(w, "Configuration Summary:\n")
	_, _ = fmt.Fprintf(w, "  Devices: %d\n", len(config.Devices))

	if config.IncludePath != "" {
		_, _ = fmt.Fprintf(w, "  Include Path: %s\n", config.IncludePath)
	}

	if len(config.CapturePlaybacks) > 0 {
		_, _ = fmt.Fprintf(w, "  PCAP Playbacks: %d\n", len(config.CapturePlaybacks))
		for i, playback := range config.CapturePlaybacks {
			_, _ = fmt.Fprintf(w, "    [%d] %s\n", i+1, playback.FileName)
			if playback.LoopTime > 0 {
				_, _ = fmt.Fprintf(w, "        Loop Time: %d ms\n", playback.LoopTime)
			}
			if playback.ScaleTime > 0 {
				_, _ = fmt.Fprintf(w, "        Scale Time: %.2f\n", playback.ScaleTime)
			}
		}
	}

	snmpCount := 0
	mibCount := 0
	for _, device := range config.Devices {
		if device.SnmpAgent != nil {
			snmpCount++
			mibCount += len(device.SnmpAgent.AddMibs)
		}
	}

	if snmpCount > 0 {
		_, _ = fmt.Fprintf(w, "  SNMP Agents: %d\n", snmpCount)
		if mibCount > 0 {
			_, _ = fmt.Fprintf(w, "  Custom MIBs: %d\n", mibCount)
		}
	}

	_ = w.Flush()
}
