package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/krisarmstrong/niac-go/pkg/httpapi"
	"github.com/krisarmstrong/niac-go/pkg/ipc"
)

const (
	topologyFormatDOT  = "dot"
	topologyFormatJSON = "json"
	topologyFormatYAML = "yaml"
)

// topologyOptions holds command-line options for the topology command.
type topologyOptions struct {
	format     string
	outputFile string
	socketPath string
}

// topologyCmd is the parent command for topology operations.
func addTopologyCommand(root *cobra.Command, _ *serviceOptions) {
	options := new(topologyOptions)

	topologyCmd := &cobra.Command{
		Use:   "topology",
		Short: "Network topology management commands",
		Long: `Network topology management commands for NIAC simulations.

These commands allow you to export and visualize the current network
topology from a running NIAC simulation.`,
		Example: `  # Export topology in DOT format for Graphviz
  niac topology export --format dot

  # Export topology as JSON
  niac topology export --format json

  # Export topology to a file
  niac topology export --format dot --output network.dot

  # Generate a PNG using Graphviz
  niac topology export --format dot | dot -Tpng -o network.png`,
	}

	// topologyExportCmd exports the current network topology.
	topologyExportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export current network topology",
		Long: `Export the current network topology from a running NIAC simulation.

Supported output formats:
  dot   - Graphviz DOT format for visualization (can be rendered with Graphviz tools)
  json  - JSON format for programmatic use
  yaml  - YAML format for programmatic use

The topology includes:
  - Devices (nodes): name, type (router, switch, ap, etc.)
  - Connections (links): interfaces, VLANs, link type, speed, status
  - Protocols: discovered neighbors via LLDP, CDP, etc.

Exit codes:
  0 - Success
  1 - Connection failed (simulation not running)
  2 - Error occurred (export failed, invalid format, etc.)`,
		Example: `  # Export to stdout in DOT format (default)
  niac topology export

  # Export as JSON
  niac topology export --format json

  # Export as YAML
  niac topology export --format yaml

  # Save to file
  niac topology export --format dot --output topology.dot

  # Use custom socket path
  niac topology export --socket /tmp/niac.sock --format json

  # Generate visualization with Graphviz
  niac topology export --format dot | dot -Tpng -o network.png
  niac topology export --format dot | dot -Tsvg -o network.svg

  # Process with jq
  niac topology export --format json | jq '.nodes[] | .name'`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runTopologyExport(options)
		},
	}

	// Export subcommand flags
	topologyExportCmd.Flags().StringVarP(&options.format, "format", "f", "dot",
		"Output format: dot, json, yaml")
	topologyExportCmd.Flags().StringVarP(&options.outputFile, "output", "o", "",
		"Output file path (default: stdout)")
	topologyExportCmd.Flags().StringVar(&options.socketPath, "socket", "",
		"Path to IPC socket (default: /tmp/niac.sock)")

	topologyCmd.AddCommand(topologyExportCmd)
	root.AddCommand(topologyCmd)
}

// runTopologyExport executes the topology export command.
func runTopologyExport(options *topologyOptions) error {
	// Validate format
	format := strings.ToLower(options.format)
	if format != topologyFormatDOT && format != topologyFormatJSON && format != topologyFormatYAML {
		return fmt.Errorf(
			"invalid format %q: must be one of: %s, %s, %s",
			options.format,
			topologyFormatDOT,
			topologyFormatJSON,
			topologyFormatYAML,
		)
	}

	// Determine socket path
	socketPath := options.socketPath
	if socketPath == "" {
		socketPath = ipc.DefaultSocketPath()
	}

	// Check if socket exists
	if _, statErr := os.Stat(socketPath); os.IsNotExist(statErr) {
		fmt.Fprintf(os.Stderr, "Error: NIAC is not running (socket not found: %s)\n", socketPath)
		os.Exit(1)
	}

	// Create IPC client and fetch topology
	client := ipc.NewClient(socketPath)
	topology, err := client.GetTopology()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get topology: %v\n", err)
		os.Exit(1)
	}

	// Generate output based on format
	var output string
	switch format {
	case topologyFormatDOT:
		output = topology.ExportDOT()
	case topologyFormatJSON:
		output, err = exportTopologyJSON(topology)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to export JSON: %v\n", err)
			os.Exit(exitCodeError)
		}
	case topologyFormatYAML:
		output, err = exportTopologyYAML(topology)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to export YAML: %v\n", err)
			os.Exit(exitCodeError)
		}
	}

	// Write output
	if options.outputFile != "" {
		if writeErr := os.WriteFile(options.outputFile, []byte(output), 0o600); writeErr != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to write to file %s: %v\n", options.outputFile, writeErr)
			os.Exit(exitCodeError)
		}
		fmt.Fprintf(os.Stderr, "Topology exported to %s\n", options.outputFile)
	} else {
		fmt.Fprint(os.Stdout, output)
	}

	return nil
}

// exportTopologyJSON exports topology as formatted JSON.
func exportTopologyJSON(topology *httpapi.Topology) (string, error) {
	// Create an extended structure with more details
	var export TopologyExport
	export.Nodes = make([]TopologyNodeExport, len(topology.Nodes))
	export.Links = make([]TopologyLinkExport, len(topology.Links))
	export.GeneratedAt = ""

	// Convert nodes
	for i, node := range topology.Nodes {
		export.Nodes[i] = TopologyNodeExport{
			Name: node.Name,
			Type: node.Type,
		}
	}

	// Convert links and count by type
	linkTypeCount := make(map[string]int)
	for i, link := range topology.Links {
		export.Links[i] = TopologyLinkExport{
			Source:          link.Source,
			Target:          link.Target,
			SourceInterface: link.SourceInterface,
			TargetInterface: link.TargetInterface,
			LinkType:        link.LinkType,
			VLANs:           link.VLANs,
			NativeVLAN:      link.NativeVLAN,
			SpeedMbps:       link.Speed,
			Duplex:          link.Duplex,
			Status:          link.Status,
			Utilization:     link.Utilization,
		}
		linkTypeCount[link.LinkType]++
	}

	// Build summary
	var summary TopologySummary
	summary.TotalNodes = len(topology.Nodes)
	summary.TotalLinks = len(topology.Links)
	summary.LinksByType = linkTypeCount
	summary.DevicesByType = countDevicesByType(topology.Nodes)
	export.Summary = summary

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal topology JSON: %w", err)
	}
	return string(data) + "\n", nil
}

// exportTopologyYAML exports topology as YAML.
func exportTopologyYAML(topology *httpapi.Topology) (string, error) {
	// Create an extended structure with more details
	var export TopologyExport
	export.Nodes = make([]TopologyNodeExport, len(topology.Nodes))
	export.Links = make([]TopologyLinkExport, len(topology.Links))
	export.GeneratedAt = ""

	// Convert nodes
	for i, node := range topology.Nodes {
		export.Nodes[i] = TopologyNodeExport{
			Name: node.Name,
			Type: node.Type,
		}
	}

	// Convert links
	linkTypeCount := make(map[string]int)
	for i, link := range topology.Links {
		export.Links[i] = TopologyLinkExport{
			Source:          link.Source,
			Target:          link.Target,
			SourceInterface: link.SourceInterface,
			TargetInterface: link.TargetInterface,
			LinkType:        link.LinkType,
			VLANs:           link.VLANs,
			NativeVLAN:      link.NativeVLAN,
			SpeedMbps:       link.Speed,
			Duplex:          link.Duplex,
			Status:          link.Status,
			Utilization:     link.Utilization,
		}
		linkTypeCount[link.LinkType]++
	}

	// Build summary
	var summary TopologySummary
	summary.TotalNodes = len(topology.Nodes)
	summary.TotalLinks = len(topology.Links)
	summary.LinksByType = linkTypeCount
	summary.DevicesByType = countDevicesByType(topology.Nodes)
	export.Summary = summary

	data, err := yaml.Marshal(export)
	if err != nil {
		return "", fmt.Errorf("failed to marshal topology YAML: %w", err)
	}
	return string(data), nil
}

// countDevicesByType counts devices by their type.
func countDevicesByType(nodes []httpapi.TopologyNode) map[string]int {
	counts := make(map[string]int)
	for _, node := range nodes {
		deviceType := node.Type
		if deviceType == "" {
			deviceType = "unknown"
		}
		counts[deviceType]++
	}
	return counts
}

// TopologyExport is the export structure for JSON/YAML output.
type TopologyExport struct {
	Nodes       []TopologyNodeExport `json:"nodes"                  yaml:"nodes"`
	Links       []TopologyLinkExport `json:"links"                  yaml:"links"`
	Summary     TopologySummary      `json:"summary"                yaml:"summary"`
	GeneratedAt string               `json:"generated_at,omitempty" yaml:"generated_at,omitempty"`
}

// TopologyNodeExport represents a device node in the export.
type TopologyNodeExport struct {
	Name string `json:"name" yaml:"name"`
	Type string `json:"type" yaml:"type"`
}

// TopologyLinkExport represents a connection link in the export.
type TopologyLinkExport struct {
	Source          string  `json:"source"                        yaml:"source"`
	Target          string  `json:"target"                        yaml:"target"`
	SourceInterface string  `json:"source_interface"              yaml:"source_interface"`
	TargetInterface string  `json:"target_interface"              yaml:"target_interface"`
	LinkType        string  `json:"link_type"                     yaml:"link_type"`
	VLANs           []int   `json:"vlans,omitempty"               yaml:"vlans,omitempty"`
	NativeVLAN      int     `json:"native_vlan,omitempty"         yaml:"native_vlan,omitempty"`
	SpeedMbps       int     `json:"speed_mbps,omitempty"          yaml:"speed_mbps,omitempty"`
	Duplex          string  `json:"duplex,omitempty"              yaml:"duplex,omitempty"`
	Status          string  `json:"status"                        yaml:"status"`
	Utilization     float64 `json:"utilization_percent,omitempty" yaml:"utilization_percent,omitempty"`
}

// TopologySummary provides summary statistics about the topology.
type TopologySummary struct {
	TotalNodes    int            `json:"total_nodes"     yaml:"total_nodes"`
	TotalLinks    int            `json:"total_links"     yaml:"total_links"`
	DevicesByType map[string]int `json:"devices_by_type" yaml:"devices_by_type"`
	LinksByType   map[string]int `json:"links_by_type"   yaml:"links_by_type"`
}
