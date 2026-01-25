package converter

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadYAMLConfig loads a YAML config file into Go config structure.
func LoadYAMLConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		return nil, fmt.Errorf("error reading YAML file: %w", err)
	}

	return LoadYAMLConfigFromBytes(data)
}

// LoadYAMLConfigFromBytes converts in-memory YAML data into a Go config structure.
func LoadYAMLConfigFromBytes(data []byte) (*Config, error) {
	var config Config
	err := yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing YAML: %w", err)
	}

	return &config, nil
}

// ValidateConfig validates that no functionality was lost in conversion.
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

	return nil
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
