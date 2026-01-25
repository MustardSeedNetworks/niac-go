// Package converter implements Java NIAC DSL to YAML conversion
package converter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConvertFile converts a Java DSL config file to YAML.
func ConvertFile(inputPath, outputPath string, verbose bool) error {
	// Read input file
	data, err := os.ReadFile(filepath.Clean(inputPath))
	if err != nil {
		return fmt.Errorf("error reading input file: %w", err)
	}

	// Parse Java DSL
	parser := NewParser(strings.Split(string(data), "\n"), verbose)

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
	if writeErr := os.WriteFile(outputPath, yamlData, 0o600); writeErr != nil {
		return fmt.Errorf("error writing output file: %w", writeErr)
	}

	return nil
}

// NewParser creates a new Parser from a string slice of lines.
func NewParser(lines []string, verbose bool) *Parser {
	return &Parser{
		lines:   lines,
		pos:     0,
		verbose: verbose,
	}
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
