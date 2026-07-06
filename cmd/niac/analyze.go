package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/MustardSeedNetworks/niac-go/internal/walkanalysis"
)

type analyzeOptions struct {
	outputFormat  string
	showNeighbors bool
	graphvizPath  string
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
		Short: "Analyze an SNMP walk file: device, interfaces, and LLDP/CDP neighbors",
		Long: `Analyze an SNMP walk file and extract device identity, the interface
inventory, and LLDP/CDP neighbor adjacencies.

The tool parses these standard SNMP MIBs:
  • SNMPv2-MIB        (system identity)
  • IF-MIB + ifXTable (interfaces: index, name, type, speed, status, MAC)
  • LLDP-MIB          (LLDP neighbors)
  • CISCO-CDP-MIB     (CDP neighbors)`,
		Example: `  # Analyze and output as YAML
  niac analyze-walk device.walk

  # Output as JSON
  niac analyze-walk --output json device.walk

  # Show only neighbor relationships
  niac analyze-walk --show-neighbors device.walk

  # Write a Graphviz (DOT) neighbor graph
  niac analyze-walk --graphviz topology.dot device.walk`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runAnalyze(args, options)
		},
	}

	analyzeCmd.Flags().StringVar(&options.outputFormat, "output", "yaml", "Output format (yaml, json, text)")
	analyzeCmd.Flags().BoolVar(&options.showNeighbors, "show-neighbors", false, "Show neighbor relationships only")
	analyzeCmd.Flags().StringVar(
		&options.graphvizPath,
		"graphviz",
		"",
		"Write Graphviz (DOT) neighbor graph to file (use '-' for stdout)",
	)

	root.AddCommand(analyzeCmd)
}

func runAnalyze(args []string, options *analyzeOptions) error {
	walkFile := args[0]

	analysis, err := walkanalysis.AnalyzeFile(walkFile)
	if err != nil {
		return fmt.Errorf("failed to parse walk file: %w", err)
	}

	if options.graphvizPath != "" {
		if graphvizErr := writeGraphviz(analysis, options.graphvizPath); graphvizErr != nil {
			return graphvizErr
		}
	}

	if options.showNeighbors {
		return outputNeighbors(analysis.Neighbors, options.outputFormat)
	}

	switch options.outputFormat {
	case analyzeOutputJSON:
		return outputJSON(analysis)
	case analyzeOutputYAML:
		return outputYAML(analysis)
	case analyzeOutputText:
		return outputText(analysis)
	default:
		return fmt.Errorf("unknown output format: %s", options.outputFormat)
	}
}

func outputJSON(analysis *walkanalysis.Analysis) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(analysis)
	if err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}

func outputYAML(analysis *walkanalysis.Analysis) error {
	encoder := yaml.NewEncoder(os.Stdout)
	defer func() { _ = encoder.Close() }()
	encoder.SetIndent(tabPadding)
	err := encoder.Encode(analysis)
	if err != nil {
		return fmt.Errorf("failed to encode YAML: %w", err)
	}
	return nil
}

func outputText(analysis *walkanalysis.Analysis) error {
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

func writeGraphviz(analysis *walkanalysis.Analysis, target string) error {
	if len(analysis.Neighbors) == 0 {
		return errors.New("no neighbor information available for graph export")
	}

	content := buildGraphvizContent(analysis)
	return writeGraphvizOutput(content, target)
}

func buildGraphvizContent(analysis *walkanalysis.Analysis) string {
	local := analysis.Device.SysName
	if local == "" {
		local = "local-device"
	}

	var builder strings.Builder
	builder.WriteString("digraph niac_topology {\n")
	builder.WriteString("  rankdir=LR;\n")
	fmt.Fprintf(
		&builder,
		"  \"%s\" [shape=box, style=filled, fillcolor=\"#2563EB\", fontcolor=\"white\"];\n",
		dotEscape(local),
	)

	seen := make(map[string]struct{})
	for _, neighbor := range analysis.Neighbors {
		if neighbor.RemoteDevice == "" {
			continue
		}
		key := strings.ToLower(neighbor.RemoteDevice)
		if _, ok := seen[key]; !ok {
			fmt.Fprintf(&builder,
				"  \"%s\" [shape=ellipse, style=filled, fillcolor=\"#0f172a\", fontcolor=\"white\"];\n",
				dotEscape(neighbor.RemoteDevice),
			)
			seen[key] = struct{}{}
		}

		label := fmt.Sprintf("%s → %s (%s)",
			neighbor.LocalInterface, neighbor.RemoteInterface, strings.ToUpper(neighbor.Protocol))
		fmt.Fprintf(&builder, "  \"%s\" -> \"%s\" [label=\"%s\"];\n",
			dotEscape(local), dotEscape(neighbor.RemoteDevice), dotEscape(label))
	}
	builder.WriteString("}\n")

	return builder.String()
}

func dotEscape(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return r.Replace(s)
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

func outputNeighbors(neighbors []walkanalysis.Neighbor, format string) error {
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
