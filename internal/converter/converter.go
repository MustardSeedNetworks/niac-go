// Package converter implements Java NIAC DSL to YAML conversion
package converter

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
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
	err := yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing YAML: %w", err)
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
	// Validate devices have required fields
	for i, device := range config.Devices {
		if device.MAC == "" {
			return fmt.Errorf("%w: device %d", ErrDeviceMissingMAC, i)
		}
		// IP address is optional in Java configs (some devices don't have IPs)

		// If SNMP agent specified, validate it (empty SNMP agents are allowed)
		if device.SnmpAgent != nil && len(device.SnmpAgent.AddMibs) > 0 {
			// Validate AddMibs have required fields
			for j, mib := range device.SnmpAgent.AddMibs {
				if mib.OID == "" {
					return fmt.Errorf("%w: device %d AddMib %d", ErrAddMibMissingOID, i, j)
				}
				if mib.Type == "" {
					return fmt.Errorf("%w: device %d AddMib %d", ErrAddMibMissingType, i, j)
				}
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
