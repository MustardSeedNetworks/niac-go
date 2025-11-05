// Package converter implements Java NIAC DSL to YAML conversion
package converter

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the YAML configuration structure
type Config struct {
	IncludePath     string             `yaml:"include_path,omitempty"`
	CapturePlayback *CapturePlayback   `yaml:"capture_playback,omitempty"`
	Devices         []Device           `yaml:"devices"`
}

// CapturePlayback represents PCAP playback configuration
type CapturePlayback struct {
	FileName  string  `yaml:"file_name"`
	LoopTime  int     `yaml:"loop_time,omitempty"`
	ScaleTime float64 `yaml:"scale_time,omitempty"`
}

// Device represents a network device
type Device struct {
	Name      string     `yaml:"name,omitempty"`
	MAC       string     `yaml:"mac"`
	IP        string     `yaml:"ip"`
	VLAN      int        `yaml:"vlan,omitempty"`
	SnmpAgent *SnmpAgent `yaml:"snmp_agent,omitempty"`
}

// SnmpAgent represents SNMP agent configuration
type SnmpAgent struct {
	WalkFile string   `yaml:"walk_file,omitempty"`
	AddMibs  []AddMib `yaml:"add_mibs,omitempty"`
}

// AddMib represents a MIB override or addition
type AddMib struct {
	OID   string `yaml:"oid"`
	Type  string `yaml:"type"`
	Value string `yaml:"value"`
}

// Parser handles parsing Java DSL format
type Parser struct {
	lines   []string
	pos     int
	verbose bool
}

// ConvertFile converts a Java DSL config file to YAML
func ConvertFile(inputPath, outputPath string, verbose bool) error {
	// Read input file
	data, err := os.ReadFile(inputPath)
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
	if err := os.WriteFile(outputPath, yamlData, 0644); err != nil {
		return fmt.Errorf("error writing output file: %w", err)
	}

	return nil
}

// Parse parses the Java DSL format
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
			config.CapturePlayback = playback
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

// parseCapturePlayback parses a CapturePlayback block
func (p *Parser) parseCapturePlayback() (*CapturePlayback, error) {
	p.pos++ // Skip opening line
	playback := &CapturePlayback{}

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])

		if line == ")" {
			p.pos++
			break
		}

		if strings.HasPrefix(line, "FileName(") {
			playback.FileName = p.extractString(line)
		} else if strings.HasPrefix(line, "LoopTime(") {
			var loopTime int
			fmt.Sscanf(line, "LoopTime(%d)", &loopTime)
			playback.LoopTime = loopTime
		} else if strings.HasPrefix(line, "ScaleTime(") {
			var scaleTime float64
			fmt.Sscanf(line, "ScaleTime(%f)", &scaleTime)
			playback.ScaleTime = scaleTime
		}

		p.pos++
	}

	return playback, nil
}

// parseDevice parses a Device block
func (p *Parser) parseDevice(deviceNum int) (*Device, error) {
	p.pos++ // Skip opening line
	device := &Device{
		Name: fmt.Sprintf("device%d", deviceNum+1),
	}

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])

		if line == ")" {
			p.pos++
			break
		}

		if strings.HasPrefix(line, "MacAddr(") {
			device.MAC = p.formatMAC(p.extractValue(line))
		} else if strings.HasPrefix(line, "IpAddr(") {
			device.IP = p.extractValue(line)
		} else if strings.HasPrefix(line, "Vlan(") {
			var vlan int
			fmt.Sscanf(line, "Vlan(%d)", &vlan)
			device.VLAN = vlan
		} else if strings.HasPrefix(line, "SnmpAgent(") {
			agent, err := p.parseSnmpAgent()
			if err != nil {
				return nil, err
			}
			device.SnmpAgent = agent
			continue
		}

		p.pos++
	}

	return device, nil
}

// parseSnmpAgent parses an SnmpAgent block
func (p *Parser) parseSnmpAgent() (*SnmpAgent, error) {
	p.pos++ // Skip opening line
	agent := &SnmpAgent{}

	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])

		if line == ")" {
			p.pos++
			break
		}

		// Skip comments
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			p.pos++
			continue
		}

		if strings.HasPrefix(line, "Include(") {
			agent.WalkFile = p.extractString(line)
		} else if strings.HasPrefix(line, "AddMib(") {
			mib := p.parseAddMib(line)
			if mib != nil {
				agent.AddMibs = append(agent.AddMibs, *mib)
			}
		}

		p.pos++
	}

	return agent, nil
}

// parseAddMib parses an AddMib directive
func (p *Parser) parseAddMib(line string) *AddMib {
	// AddMib("OID", "type", "value")
	// Extract using regex
	re := regexp.MustCompile(`AddMib\("([^"]+)",\s*"([^"]+)",\s*"([^"]+)"\)`)
	matches := re.FindStringSubmatch(line)

	if len(matches) != 4 {
		return nil
	}

	return &AddMib{
		OID:   matches[1],
		Type:  matches[2],
		Value: matches[3],
	}
}

// extractString extracts a quoted string from a directive
func (p *Parser) extractString(line string) string {
	start := strings.Index(line, "\"")
	if start == -1 {
		return ""
	}
	end := strings.Index(line[start+1:], "\"")
	if end == -1 {
		return ""
	}
	return line[start+1 : start+1+end]
}

// extractValue extracts a value from parentheses (no quotes)
func (p *Parser) extractValue(line string) string {
	start := strings.Index(line, "(")
	if start == -1 {
		return ""
	}
	end := strings.Index(line[start+1:], ")")
	if end == -1 {
		return ""
	}
	return line[start+1 : start+1+end]
}

// formatMAC converts XXXXXXXXXXXX to XX:XX:XX:XX:XX:XX
func (p *Parser) formatMAC(mac string) string {
	if len(mac) != 12 {
		return mac
	}
	return fmt.Sprintf("%s:%s:%s:%s:%s:%s",
		mac[0:2], mac[2:4], mac[4:6], mac[6:8], mac[8:10], mac[10:12])
}

// LoadYAMLConfig loads a YAML config file into Go config structure
func LoadYAMLConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading YAML file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing YAML: %w", err)
	}

	return &config, nil
}

// ValidateConfig validates that no functionality was lost in conversion
func ValidateConfig(config *Config) error {
	// Validate devices have required fields
	for i, device := range config.Devices {
		if device.MAC == "" {
			return fmt.Errorf("device %d missing MAC address", i)
		}
		if device.IP == "" {
			return fmt.Errorf("device %d missing IP address", i)
		}

		// If SNMP agent specified, validate it
		if device.SnmpAgent != nil {
			// Must have either walk file or add mibs
			if device.SnmpAgent.WalkFile == "" && len(device.SnmpAgent.AddMibs) == 0 {
				return fmt.Errorf("device %d SNMP agent has no walk file or MIBs", i)
			}

			// Validate AddMibs have required fields
			for j, mib := range device.SnmpAgent.AddMibs {
				if mib.OID == "" {
					return fmt.Errorf("device %d AddMib %d missing OID", i, j)
				}
				if mib.Type == "" {
					return fmt.Errorf("device %d AddMib %d missing type", i, j)
				}
			}
		}
	}

	// If capture playback specified, validate it
	if config.CapturePlayback != nil {
		if config.CapturePlayback.FileName == "" {
			return fmt.Errorf("CapturePlayback missing file name")
		}
	}

	return nil
}

// PrintSummary prints a summary of the config
func PrintSummary(config *Config, w *bufio.Writer) {
	fmt.Fprintf(w, "Configuration Summary:\n")
	fmt.Fprintf(w, "  Devices: %d\n", len(config.Devices))

	if config.IncludePath != "" {
		fmt.Fprintf(w, "  Include Path: %s\n", config.IncludePath)
	}

	if config.CapturePlayback != nil {
		fmt.Fprintf(w, "  PCAP Playback: %s\n", config.CapturePlayback.FileName)
		if config.CapturePlayback.LoopTime > 0 {
			fmt.Fprintf(w, "    Loop Time: %d ms\n", config.CapturePlayback.LoopTime)
		}
		if config.CapturePlayback.ScaleTime > 0 {
			fmt.Fprintf(w, "    Scale Time: %.2f\n", config.CapturePlayback.ScaleTime)
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
		fmt.Fprintf(w, "  SNMP Agents: %d\n", snmpCount)
		if mibCount > 0 {
			fmt.Fprintf(w, "  Custom MIBs: %d\n", mibCount)
		}
	}

	w.Flush()
}
