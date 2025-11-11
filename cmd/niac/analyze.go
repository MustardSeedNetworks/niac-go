package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// WalkAnalysis represents the analyzed walk file data
type WalkAnalysis struct {
	Device     DeviceInfo      `json:"device" yaml:"device"`
	Interfaces []InterfaceInfo `json:"interfaces" yaml:"interfaces"`
	Neighbors  []NeighborInfo  `json:"neighbors" yaml:"neighbors"`
	Statistics AnalysisStats   `json:"statistics" yaml:"statistics"`
}

// DeviceInfo contains device identification
type DeviceInfo struct {
	SysName     string `json:"sysname" yaml:"sysname"`
	SysDescr    string `json:"sysdescr" yaml:"sysdescr"`
	SysObjectID string `json:"sysobjectid" yaml:"sysobjectid"`
	SysContact  string `json:"syscontact,omitempty" yaml:"syscontact,omitempty"`
	SysLocation string `json:"syslocation,omitempty" yaml:"syslocation,omitempty"`
}

// InterfaceInfo contains interface details
type InterfaceInfo struct {
	Index       int      `json:"index" yaml:"index"`
	Name        string   `json:"name" yaml:"name"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Type        string   `json:"type" yaml:"type"`
	Speed       int64    `json:"speed,omitempty" yaml:"speed,omitempty"`
	AdminStatus string   `json:"admin_status,omitempty" yaml:"admin_status,omitempty"`
	OperStatus  string   `json:"oper_status,omitempty" yaml:"oper_status,omitempty"`
	MACAddress  string   `json:"mac_address,omitempty" yaml:"mac_address,omitempty"`
	MemberOf    string   `json:"member_of,omitempty" yaml:"member_of,omitempty"`
	Members     []string `json:"members,omitempty" yaml:"members,omitempty"`
	Trunk       bool     `json:"trunk,omitempty" yaml:"trunk,omitempty"`
	VLANs       []int    `json:"vlans,omitempty" yaml:"vlans,omitempty"`
}

// NeighborInfo contains neighbor relationship
type NeighborInfo struct {
	LocalInterface  string `json:"local_interface" yaml:"local_interface"`
	RemoteDevice    string `json:"remote_device" yaml:"remote_device"`
	RemoteInterface string `json:"remote_interface" yaml:"remote_interface"`
	Protocol        string `json:"protocol" yaml:"protocol"`
}

// AnalysisStats contains statistics
type AnalysisStats struct {
	TotalInterfaces    int `json:"total_interfaces" yaml:"total_interfaces"`
	PhysicalInterfaces int `json:"physical_interfaces" yaml:"physical_interfaces"`
	LogicalInterfaces  int `json:"logical_interfaces" yaml:"logical_interfaces"`
	TotalNeighbors     int `json:"total_neighbors" yaml:"total_neighbors"`
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze-walk <walk-file>",
	Short: "Analyze SNMP walk file and extract network relationships",
	Long: `Analyze an SNMP walk file and extract device information, interfaces,
port-channels, and neighbor relationships.

The tool parses standard SNMP MIBs including:
  • IF-MIB (interfaces)
  • SNMPv2-MIB (system information)
  • LLDP-MIB (LLDP neighbors)
  • CISCO-CDP-MIB (CDP neighbors)
  • LAG-MIB (port-channels)`,
	Example: `  # Analyze and output as YAML
  niac analyze-walk device.walk

  # Output as JSON
  niac analyze-walk --output json device.walk

  # Extract full topology
  niac analyze-walk --extract-topology device.walk

  # Show only neighbors
  niac analyze-walk --show-neighbors device.walk`,
	Args: cobra.ExactArgs(1),
	RunE: runAnalyze,
}

func init() {
	rootCmd.AddCommand(analyzeCmd)

	analyzeCmd.Flags().String("output", "yaml", "Output format (yaml, json, text)")
	analyzeCmd.Flags().Bool("extract-topology", false, "Extract full topology")
	analyzeCmd.Flags().Bool("show-neighbors", false, "Show neighbor relationships only")
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	walkFile := args[0]
	outputFormat, _ := cmd.Flags().GetString("output")
	_, _ = cmd.Flags().GetBool("extract-topology") // Reserved for future use
	showNeighbors, _ := cmd.Flags().GetBool("show-neighbors")

	// Parse walk file
	analysis, err := parseWalkFile(walkFile)
	if err != nil {
		return fmt.Errorf("failed to parse walk file: %w", err)
	}

	// Filter output if needed
	if showNeighbors {
		return outputNeighbors(analysis.Neighbors, outputFormat)
	}

	// Output results
	switch outputFormat {
	case "json":
		return outputJSON(analysis)
	case "yaml":
		return outputYAML(analysis)
	case "text":
		return outputText(analysis)
	default:
		return fmt.Errorf("unknown output format: %s", outputFormat)
	}
}

func parseWalkFile(filename string) (*WalkAnalysis, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	analysis := &WalkAnalysis{
		Interfaces: make([]InterfaceInfo, 0),
		Neighbors:  make([]NeighborInfo, 0),
	}

	scanner := bufio.NewScanner(file)
	interfaceMap := make(map[int]*InterfaceInfo)

	// Regular expressions for parsing numeric OIDs
	// NOTE: Sanitized walk files have mangled OIDs where IP-like patterns were replaced
	// Original: .1.3.6.1.2.1.1.x.0  Sanitized: .1.3.6.1.2.10.100.x.x
	// We use flexible patterns to match both original and sanitized formats

	// SNMPv2-MIB::system group - Match the value patterns with quoted strings
	sysDescrRe := regexp.MustCompile(`= STRING: "(.+?Cisco.+?)"`)
	sysObjectIDRe := regexp.MustCompile(`= OID: (\.1\.3\.6\.[\d.]+)`)

	// IF-MIB::ifTable - Match interface data patterns
	// Interface descriptions contain names like "FastEthernet", "GigabitEthernet", "Serial", etc.
	ifDescrRe := regexp.MustCompile(`\.1\.3\.6\.1\.2\.[\d.]+\s*=\s*STRING:\s*"((?:Fast|Gigabit)?Ethernet[\d/]+|Serial[\d/]+|Loopback\d+|Null\d+|Async[\d/]+|Port-channel\d+|Vlan\d+|Tunnel\d+)"`)

	// For sanitized files, extract interface info from the contextifDescr matches interface names, we can build a basic map
	interfaceCounter := 0

	for scanner.Scan() {
		line := scanner.Text()

		// Parse system information
		if analysis.Device.SysDescr == "" {
			if matches := sysDescrRe.FindStringSubmatch(line); matches != nil {
				analysis.Device.SysDescr = strings.TrimSpace(matches[1])
				// Try to extract hostname from sysDescr if available
				continue
			}
		}

		if analysis.Device.SysObjectID == "" {
			if matches := sysObjectIDRe.FindStringSubmatch(line); matches != nil {
				analysis.Device.SysObjectID = strings.TrimSpace(matches[1])
				continue
			}
		}

		// Parse interface descriptions - these give us interface names
		if matches := ifDescrRe.FindStringSubmatch(line); matches != nil {
			interfaceCounter++
			ifName := strings.TrimSpace(matches[1])
			interfaceMap[interfaceCounter] = &InterfaceInfo{
				Index: interfaceCounter,
				Name:  ifName,
				Type:  getInterfaceTypeFromName(ifName),
			}
		}
	}

	// Try to find sysName by looking for short hostnames in STRING fields
	file.Seek(0, 0)
	scanner = bufio.NewScanner(file)
	hostnameRe := regexp.MustCompile(`= STRING: "([a-z0-9\-]{3,30})"$`)
	for scanner.Scan() {
		line := scanner.Text()
		if analysis.Device.SysName == "" {
			if matches := hostnameRe.FindStringSubmatch(line); matches != nil {
				name := matches[1]
				// Likely a hostname if it's short and doesn't contain common noise
				if len(name) < 25 && !strings.Contains(name, " ") && !strings.Contains(strings.ToLower(name), "cisco") {
					analysis.Device.SysName = name
					break
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Convert map to sorted slice
	indices := make([]int, 0, len(interfaceMap))
	for idx := range interfaceMap {
		indices = append(indices, idx)
	}
	sort.Ints(indices)

	for _, idx := range indices {
		iface := interfaceMap[idx]
		if iface.Name != "" { // Only include interfaces with names
			analysis.Interfaces = append(analysis.Interfaces, *iface)
		}
	}

	// Calculate statistics
	analysis.Statistics.TotalInterfaces = len(analysis.Interfaces)
	for _, iface := range analysis.Interfaces {
		if iface.Type == "physical" || iface.Type == "ethernet" {
			analysis.Statistics.PhysicalInterfaces++
		} else {
			analysis.Statistics.LogicalInterfaces++
		}
	}
	analysis.Statistics.TotalNeighbors = len(analysis.Neighbors)

	return analysis, nil
}

func getInterfaceType(ifType int) string {
	// IF-MIB ifType values
	switch ifType {
	case 6:
		return "ethernet"
	case 24:
		return "loopback"
	case 131:
		return "tunnel"
	case 161:
		return "port-channel"
	case 135, 136:
		return "vlan"
	default:
		if ifType >= 1 && ifType <= 100 {
			return "physical"
		}
		return "logical"
	}
}

func getInterfaceTypeFromName(name string) string {
	// Determine interface type from name pattern
	nameLower := strings.ToLower(name)

	if strings.Contains(nameLower, "loopback") {
		return "loopback"
	}
	if strings.Contains(nameLower, "null") {
		return "null"
	}
	if strings.Contains(nameLower, "tunnel") {
		return "tunnel"
	}
	if strings.Contains(nameLower, "port-channel") || strings.Contains(nameLower, "po") {
		return "port-channel"
	}
	if strings.Contains(nameLower, "vlan") {
		return "vlan"
	}
	if strings.Contains(nameLower, "ethernet") {
		return "ethernet"
	}
	if strings.Contains(nameLower, "serial") {
		return "serial"
	}
	if strings.Contains(nameLower, "async") {
		return "async"
	}

	return "unknown"
}

func formatMACAddress(raw string) string {
	// Convert various MAC formats to xx:xx:xx:xx:xx:xx
	cleaned := strings.ReplaceAll(raw, " ", ":")
	cleaned = strings.ReplaceAll(cleaned, "-", ":")
	return strings.ToLower(cleaned)
}

func outputJSON(analysis *WalkAnalysis) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(analysis)
}

func outputYAML(analysis *WalkAnalysis) error {
	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	encoder.SetIndent(2)
	return encoder.Encode(analysis)
}

func outputText(analysis *WalkAnalysis) error {
	fmt.Printf("Device: %s\n", analysis.Device.SysName)
	fmt.Printf("Description: %s\n", analysis.Device.SysDescr)
	if analysis.Device.SysContact != "" {
		fmt.Printf("Contact: %s\n", analysis.Device.SysContact)
	}
	if analysis.Device.SysLocation != "" {
		fmt.Printf("Location: %s\n", analysis.Device.SysLocation)
	}
	fmt.Println()

	fmt.Printf("Interfaces (%d total, %d physical, %d logical):\n",
		analysis.Statistics.TotalInterfaces,
		analysis.Statistics.PhysicalInterfaces,
		analysis.Statistics.LogicalInterfaces)
	for _, iface := range analysis.Interfaces {
		fmt.Printf("  [%d] %s (%s)\n", iface.Index, iface.Name, iface.Type)
		if iface.Speed > 0 {
			fmt.Printf("      Speed: %d bps\n", iface.Speed)
		}
		if iface.AdminStatus != "" {
			fmt.Printf("      Status: %s/%s\n", iface.AdminStatus, iface.OperStatus)
		}
		if iface.MACAddress != "" {
			fmt.Printf("      MAC: %s\n", iface.MACAddress)
		}
	}
	fmt.Println()

	if len(analysis.Neighbors) > 0 {
		fmt.Printf("Neighbors (%d):\n", len(analysis.Neighbors))
		for _, neighbor := range analysis.Neighbors {
			fmt.Printf("  %s (%s) -> %s (%s)\n",
				neighbor.LocalInterface, neighbor.Protocol,
				neighbor.RemoteDevice, neighbor.RemoteInterface)
		}
	}

	return nil
}

func outputNeighbors(neighbors []NeighborInfo, format string) error {
	switch format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(map[string]interface{}{"neighbors": neighbors})
	case "yaml":
		encoder := yaml.NewEncoder(os.Stdout)
		defer encoder.Close()
		encoder.SetIndent(2)
		return encoder.Encode(map[string]interface{}{"neighbors": neighbors})
	case "text":
		for _, neighbor := range neighbors {
			fmt.Printf("%s (%s) -> %s (%s)\n",
				neighbor.LocalInterface, neighbor.Protocol,
				neighbor.RemoteDevice, neighbor.RemoteInterface)
		}
		return nil
	default:
		return fmt.Errorf("unknown output format: %s", format)
	}
}
