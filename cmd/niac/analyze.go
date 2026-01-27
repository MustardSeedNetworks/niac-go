package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// WalkAnalysis represents the analyzed walk file data.
type WalkAnalysis struct {
	Device     DeviceInfo      `json:"device"     yaml:"device"`
	Interfaces []InterfaceInfo `json:"interfaces" yaml:"interfaces"`
	Neighbors  []NeighborInfo  `json:"neighbors"  yaml:"neighbors"`
	Statistics AnalysisStats   `json:"statistics" yaml:"statistics"`
}

// DeviceInfo contains device identification.
type DeviceInfo struct {
	SysName     string `json:"sysname"               yaml:"sysname"`
	SysDescr    string `json:"sysdescr"              yaml:"sysdescr"`
	SysObjectID string `json:"sysobjectid"           yaml:"sysobjectid"`
	SysContact  string `json:"syscontact,omitempty"  yaml:"syscontact,omitempty"`
	SysLocation string `json:"syslocation,omitempty" yaml:"syslocation,omitempty"`
}

// InterfaceInfo contains interface details.
type InterfaceInfo struct {
	Index       int      `json:"index"                  yaml:"index"`
	Name        string   `json:"name"                   yaml:"name"`
	Description string   `json:"description,omitempty"  yaml:"description,omitempty"`
	Type        string   `json:"type"                   yaml:"type"`
	Speed       int64    `json:"speed,omitempty"        yaml:"speed,omitempty"`
	AdminStatus string   `json:"admin_status,omitempty" yaml:"admin_status,omitempty"`
	OperStatus  string   `json:"oper_status,omitempty"  yaml:"oper_status,omitempty"`
	MACAddress  string   `json:"mac_address,omitempty"  yaml:"mac_address,omitempty"`
	MemberOf    string   `json:"member_of,omitempty"    yaml:"member_of,omitempty"`
	Members     []string `json:"members,omitempty"      yaml:"members,omitempty"`
	Trunk       bool     `json:"trunk,omitempty"        yaml:"trunk,omitempty"`
	VLANs       []int    `json:"vlans,omitempty"        yaml:"vlans,omitempty"`
}

// NeighborInfo contains neighbor relationship.
type NeighborInfo struct {
	LocalInterface  string `json:"local_interface"  yaml:"local_interface"`
	RemoteDevice    string `json:"remote_device"    yaml:"remote_device"`
	RemoteInterface string `json:"remote_interface" yaml:"remote_interface"`
	Protocol        string `json:"protocol"         yaml:"protocol"`
}

// AnalysisStats contains statistics.
type AnalysisStats struct {
	TotalInterfaces    int `json:"total_interfaces"    yaml:"total_interfaces"`
	PhysicalInterfaces int `json:"physical_interfaces" yaml:"physical_interfaces"`
	LogicalInterfaces  int `json:"logical_interfaces"  yaml:"logical_interfaces"`
	TotalNeighbors     int `json:"total_neighbors"     yaml:"total_neighbors"`
}

type analyzeOptions struct {
	outputFormat    string
	extractTopology bool
	showNeighbors   bool
	graphvizPath    string
}

const (
	analyzeOutputJSON = "json"
	analyzeOutputYAML = "yaml"
	analyzeOutputText = "text"
)

func addAnalyzeCommand(root *cobra.Command, _ *serviceOptions) {
	options := new(analyzeOptions)

	analyzeCmd := &cobra.Command{
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
		RunE: func(_ *cobra.Command, args []string) error {
			return runAnalyze(args, options)
		},
	}

	analyzeCmd.Flags().StringVar(&options.outputFormat, "output", "yaml", "Output format (yaml, json, text)")
	analyzeCmd.Flags().BoolVar(&options.extractTopology, "extract-topology", false, "Extract full topology")
	analyzeCmd.Flags().BoolVar(&options.showNeighbors, "show-neighbors", false, "Show neighbor relationships only")
	analyzeCmd.Flags().StringVar(
		&options.graphvizPath,
		"graphviz",
		"",
		"Write Graphviz (DOT) output to file (use '-' for stdout)",
	)

	root.AddCommand(analyzeCmd)
}

func runAnalyze(args []string, options *analyzeOptions) error {
	walkFile := args[0]
	_ = options.extractTopology // Reserved for future use
	outputFormat := options.outputFormat
	showNeighbors := options.showNeighbors
	graphvizPath := options.graphvizPath

	// Parse walk file
	analysis, err := parseWalkFile(walkFile)
	if err != nil {
		return fmt.Errorf("failed to parse walk file: %w", err)
	}

	if graphvizPath != "" {
		if graphvizErr := writeGraphviz(analysis, graphvizPath); graphvizErr != nil {
			return graphvizErr
		}
	}

	// Filter output if needed
	if showNeighbors {
		return outputNeighbors(analysis.Neighbors, outputFormat)
	}

	// Output results
	switch outputFormat {
	case analyzeOutputJSON:
		return outputJSON(analysis)
	case analyzeOutputYAML:
		return outputYAML(analysis)
	case analyzeOutputText:
		return outputText(analysis)
	default:
		return fmt.Errorf("unknown output format: %s", outputFormat)
	}
}

func parseWalkFile(filename string) (*WalkAnalysis, error) {
	cleanPath := filepath.Clean(filename)
	if strings.Contains(cleanPath, "..") {
		return nil, errors.New("path traversal detected in filename")
	}

	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open walk file: %w", err)
	}
	defer func() { _ = file.Close() }()

	analysis, interfaceMap, err := parseWalkFileContent(file)
	if err != nil {
		return nil, err
	}

	analysis.Interfaces = collectInterfacesFromMap(interfaceMap)
	calculateStatistics(analysis)

	return analysis, nil
}

func parseWalkFileContent(file *os.File) (*WalkAnalysis, map[int]*InterfaceInfo, error) {
	analysis := new(WalkAnalysis)
	analysis.Interfaces = make([]InterfaceInfo, 0)
	analysis.Neighbors = make([]NeighborInfo, 0)
	interfaceMap := make(map[int]*InterfaceInfo)

	sysDescrRe := regexp.MustCompile(`= STRING: "(.+?Cisco.+?)"`)
	sysObjectIDRe := regexp.MustCompile(`= OID: (\.1\.3\.6\.[\d.]+)`)
	ifDescrRe := regexp.MustCompile(
		`\.1\.3\.6\.1\.2\.[\d.]+\s*=\s*STRING:\s*"((?:Fast|Gigabit)?Ethernet[\d/]+|Serial[\d/]+|Loopback\d+|Null\d+|Async[\d/]+|Port-channel\d+|Vlan\d+|Tunnel\d+)"`,
	)
	hostnameRe := regexp.MustCompile(`= STRING: "([a-z0-9\-]{3,30})"$`)

	if err := parseSystemAndInterfaces(file, analysis, interfaceMap, sysDescrRe, sysObjectIDRe, ifDescrRe); err != nil {
		return nil, nil, err
	}

	if err := extractHostname(file, analysis, hostnameRe); err != nil {
		return nil, nil, err
	}

	return analysis, interfaceMap, nil
}

func collectInterfacesFromMap(interfaceMap map[int]*InterfaceInfo) []InterfaceInfo {
	indices := make([]int, 0, len(interfaceMap))
	for idx := range interfaceMap {
		indices = append(indices, idx)
	}
	sort.Ints(indices)

	interfaces := make([]InterfaceInfo, 0, len(interfaceMap))
	for _, idx := range indices {
		iface := interfaceMap[idx]
		if iface.Name != "" {
			interfaces = append(interfaces, *iface)
		}
	}
	return interfaces
}

func calculateStatistics(analysis *WalkAnalysis) {
	analysis.Statistics.TotalInterfaces = len(analysis.Interfaces)
	for _, iface := range analysis.Interfaces {
		if iface.Type == "physical" || iface.Type == "ethernet" {
			analysis.Statistics.PhysicalInterfaces++
		} else {
			analysis.Statistics.LogicalInterfaces++
		}
	}
	analysis.Statistics.TotalNeighbors = len(analysis.Neighbors)
}

func parseSystemAndInterfaces(
	file *os.File,
	analysis *WalkAnalysis,
	interfaceMap map[int]*InterfaceInfo,
	sysDescrRe, sysObjectIDRe, ifDescrRe *regexp.Regexp,
) error {
	scanner := bufio.NewScanner(file)
	interfaceCounter := 0

	for scanner.Scan() {
		line := scanner.Text()

		if analysis.Device.SysDescr == "" {
			if matches := sysDescrRe.FindStringSubmatch(line); matches != nil {
				analysis.Device.SysDescr = strings.TrimSpace(matches[1])
				continue
			}
		}

		if analysis.Device.SysObjectID == "" {
			if matches := sysObjectIDRe.FindStringSubmatch(line); matches != nil {
				analysis.Device.SysObjectID = strings.TrimSpace(matches[1])
				continue
			}
		}

		if matches := ifDescrRe.FindStringSubmatch(line); matches != nil {
			interfaceCounter++
			ifName := strings.TrimSpace(matches[1])
			ifaceInfo := new(InterfaceInfo)
			ifaceInfo.Index = interfaceCounter
			ifaceInfo.Name = ifName
			ifaceInfo.Type = getInterfaceTypeFromName(ifName)
			interfaceMap[interfaceCounter] = ifaceInfo
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to scan walk file: %w", err)
	}
	return nil
}

func extractHostname(file *os.File, analysis *WalkAnalysis, hostnameRe *regexp.Regexp) error {
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to reset walk file: %w", err)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if analysis.Device.SysName == "" {
			if matches := hostnameRe.FindStringSubmatch(line); matches != nil {
				name := matches[1]
				if len(name) < 25 && !strings.Contains(name, " ") && !strings.Contains(strings.ToLower(name), "cisco") {
					analysis.Device.SysName = name
					break
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to scan walk file: %w", err)
	}
	return nil
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

func outputJSON(analysis *WalkAnalysis) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(analysis)
	if err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}

func outputYAML(analysis *WalkAnalysis) error {
	encoder := yaml.NewEncoder(os.Stdout)
	defer func() { _ = encoder.Close() }()
	encoder.SetIndent(tabPadding)
	err := encoder.Encode(analysis)
	if err != nil {
		return fmt.Errorf("failed to encode YAML: %w", err)
	}
	return nil
}

func outputText(analysis *WalkAnalysis) error {
	fmt.Fprintf(os.Stdout, "Device: %s\n", analysis.Device.SysName)
	fmt.Fprintf(os.Stdout, "Description: %s\n", analysis.Device.SysDescr)
	if analysis.Device.SysContact != "" {
		fmt.Fprintf(os.Stdout, "Contact: %s\n", analysis.Device.SysContact)
	}
	if analysis.Device.SysLocation != "" {
		fmt.Fprintf(os.Stdout, "Location: %s\n", analysis.Device.SysLocation)
	}
	fmt.Fprintln(os.Stdout)

	fmt.Fprintf(os.Stdout, "Interfaces (%d total, %d physical, %d logical):\n",
		analysis.Statistics.TotalInterfaces,
		analysis.Statistics.PhysicalInterfaces,
		analysis.Statistics.LogicalInterfaces)
	for _, iface := range analysis.Interfaces {
		fmt.Fprintf(os.Stdout, "  [%d] %s (%s)\n", iface.Index, iface.Name, iface.Type)
		if iface.Speed > 0 {
			fmt.Fprintf(os.Stdout, "      Speed: %d bps\n", iface.Speed)
		}
		if iface.AdminStatus != "" {
			fmt.Fprintf(os.Stdout, "      Status: %s/%s\n", iface.AdminStatus, iface.OperStatus)
		}
		if iface.MACAddress != "" {
			fmt.Fprintf(os.Stdout, "      MAC: %s\n", iface.MACAddress)
		}
	}
	fmt.Fprintln(os.Stdout)

	if len(analysis.Neighbors) > 0 {
		fmt.Fprintf(os.Stdout, "Neighbors (%d):\n", len(analysis.Neighbors))
		for _, neighbor := range analysis.Neighbors {
			fmt.Fprintf(os.Stdout, "  %s (%s) -> %s (%s)\n",
				neighbor.LocalInterface, neighbor.Protocol,
				neighbor.RemoteDevice, neighbor.RemoteInterface)
		}
	}

	return nil
}

func writeGraphviz(analysis *WalkAnalysis, target string) error {
	if len(analysis.Neighbors) == 0 {
		return errors.New("no neighbor information available for graph export")
	}

	content := buildGraphvizContent(analysis)
	return writeGraphvizOutput(content, target)
}

func buildGraphvizContent(analysis *WalkAnalysis) string {
	local := analysis.Device.SysName
	if local == "" {
		local = "local-device"
	}

	var builder strings.Builder
	builder.WriteString("digraph niac_topology {\n")
	builder.WriteString("  rankdir=LR;\n")
	fmt.Fprintf(&builder, "  \"%s\" [shape=box, style=filled, fillcolor=\"#2563EB\", fontcolor=\"white\"];\n", local)

	seen := make(map[string]struct{})
	for _, neighbor := range analysis.Neighbors {
		if neighbor.RemoteDevice == "" {
			continue
		}
		key := strings.ToLower(neighbor.RemoteDevice)
		if _, ok := seen[key]; !ok {
			fmt.Fprintf(&builder,
				"  \"%s\" [shape=ellipse, style=filled, fillcolor=\"#0f172a\", fontcolor=\"white\"];\n",
				neighbor.RemoteDevice,
			)
			seen[key] = struct{}{}
		}

		label := fmt.Sprintf("%s → %s (%s)",
			neighbor.LocalInterface, neighbor.RemoteInterface, strings.ToUpper(neighbor.Protocol))
		fmt.Fprintf(&builder, "  \"%s\" -> \"%s\" [label=\"%s\"];\n", local, neighbor.RemoteDevice, label)
	}
	builder.WriteString("}\n")

	return builder.String()
}

func writeGraphvizOutput(content, target string) error {
	if target == "-" {
		_, _ = os.Stdout.WriteString(content)
		return nil
	}
	if err := os.WriteFile(target, []byte(content), 0o600); err != nil {
		return fmt.Errorf("failed to write DOT file: %w", err)
	}
	return nil
}

func outputNeighbors(neighbors []NeighborInfo, format string) error {
	switch format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		err := encoder.Encode(map[string]any{"neighbors": neighbors})
		if err != nil {
			return fmt.Errorf("failed to encode neighbors JSON: %w", err)
		}
		return nil
	case "yaml":
		encoder := yaml.NewEncoder(os.Stdout)
		defer func() { _ = encoder.Close() }()
		encoder.SetIndent(tabPadding)
		err := encoder.Encode(map[string]any{"neighbors": neighbors})
		if err != nil {
			return fmt.Errorf("failed to encode neighbors YAML: %w", err)
		}
		return nil
	case "text":
		for _, neighbor := range neighbors {
			fmt.Fprintf(os.Stdout, "%s (%s) -> %s (%s)\n",
				neighbor.LocalInterface, neighbor.Protocol,
				neighbor.RemoteDevice, neighbor.RemoteInterface)
		}
		return nil
	default:
		return fmt.Errorf("unknown output format: %s", format)
	}
}
