package interactive

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/krisarmstrong/niac-go/internal/api"
)

// generateTopologyView generates an ASCII network topology diagram.
func (m *model) generateTopologyView() {
	var sb strings.Builder

	// Build topology using the api package
	topology := api.BuildTopology(m.cfg)

	sb.WriteString(diffHeaderStyle.Render("=== Network Topology ==="))
	sb.WriteString("\n\n")

	// Display nodes
	sb.WriteString(topologyNodeStyle.Render("Nodes:"))
	sb.WriteString("\n")

	// Group nodes by type
	nodesByType := make(map[string][]api.TopologyNode)
	for _, node := range topology.Nodes {
		nodesByType[node.Type] = append(nodesByType[node.Type], node)
	}

	for nodeType, nodes := range nodesByType {
		fmt.Fprintf(&sb, "  [%s]\n", nodeType)

		for _, node := range nodes {
			fmt.Fprintf(&sb, "    +-- %s\n", node.Name)
		}
	}

	sb.WriteString("\n")
	sb.WriteString(topologyLinkStyle.Render("Links:"))
	sb.WriteString("\n")

	if len(topology.Links) == 0 {
		sb.WriteString("  (no trunk connections defined)\n")
	} else {
		for _, link := range topology.Links {
			linkInfo := fmt.Sprintf("  %s [%s] <---> [%s] %s",
				link.Source,
				link.SourceInterface,
				link.TargetInterface,
				link.Target)

			if link.LinkType != "" {
				linkInfo += fmt.Sprintf(" (%s)", link.LinkType)
			}

			if len(link.VLANs) > 0 {
				linkInfo += fmt.Sprintf(" VLANs: %v", link.VLANs)
			}

			sb.WriteString(linkInfo + "\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(diffHeaderStyle.Render("=== ASCII Diagram ==="))
	sb.WriteString("\n\n")

	// Generate simple ASCII diagram
	sb.WriteString(m.generateASCIIDiagram(topology))

	m.topologyContent = sb.String()
}

// categorizeTopologyNodes splits nodes into routers, switches, and others.
func categorizeTopologyNodes(
	nodes []api.TopologyNode,
) ([]api.TopologyNode, []api.TopologyNode, []api.TopologyNode) {
	var routers, switches, others []api.TopologyNode
	for _, node := range nodes {
		switch node.Type {
		case "router":
			routers = append(routers, node)
		case "switch":
			switches = append(switches, node)
		default:
			others = append(others, node)
		}
	}
	return routers, switches, others
}

// writeNodeRow writes a row of device boxes to the builder.
func writeNodeRow(sb *strings.Builder, nodes []api.TopologyNode) {
	sb.WriteString("  ")
	for i, n := range nodes {
		if i > 0 {
			sb.WriteString("    ")
		}
		fmt.Fprintf(sb, "[%s]", truncateName(n.Name, maxTruncateName))
	}
	sb.WriteString("\n")
}

// writeConnectors writes vertical connector lines for the diagram.
func writeConnectors(sb *strings.Builder, count int) {
	sb.WriteString("      ")
	for range count {
		sb.WriteString("    |       ")
	}
	sb.WriteString("\n")
}

// writeDeviceLayer writes a complete device layer (header, nodes, optional connectors).
func writeDeviceLayer(
	sb *strings.Builder,
	label string,
	nodes []api.TopologyNode,
	hasMoreBelow bool,
) {
	if len(nodes) == 0 {
		return
	}

	fmt.Fprintf(sb, "                        %s\n", label)
	writeNodeRow(sb, nodes)

	if hasMoreBelow {
		writeConnectors(sb, len(nodes))
	}
}

func (m *model) generateASCIIDiagram(topology api.Topology) string {
	if len(topology.Nodes) == 0 {
		return "  (no devices configured)\n"
	}

	var sb strings.Builder
	routers, switches, others := categorizeTopologyNodes(topology.Nodes)

	hasDevicesBelow := len(switches) > 0 || len(others) > 0
	writeDeviceLayer(&sb, "ROUTERS", routers, hasDevicesBelow)
	writeDeviceLayer(&sb, "SWITCHES", switches, len(others) > 0)
	writeDeviceLayer(&sb, "DEVICES", others, false)

	return sb.String()
}

// renderTopology renders the topology view.
func (m *model) renderTopology() string {
	var panel strings.Builder

	panel.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	panel.WriteString("║                      Network Topology                            ║\n")
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	if m.topologyContent == "" {
		panel.WriteString(padPanelLine("No topology data available"))
		panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

		return panel.String()
	}

	lines := strings.Split(m.topologyContent, "\n")

	// Calculate visible lines
	maxLines := 18
	totalLines := len(lines)

	startLine := m.topologyScrollY
	if startLine >= totalLines {
		startLine = max(totalLines-1, 0)
	}

	endLine := min(startLine+maxLines, totalLines)

	for i := startLine; i < endLine; i++ {
		panel.WriteString(padPanelLine(lines[i]))
	}

	if totalLines > maxLines {
		panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
		panel.WriteString(
			padPanelLine(
				fmt.Sprintf(
					"Lines %d-%d of %d (use arrows/PgUp/PgDn)",
					startLine+1,
					endLine,
					totalLines,
				),
			),
		)
	}

	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine("[T] or [ESC] Close"))
	panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return panel.String()
}

// handleTopologyToggle toggles the topology view.
func (m *model) handleTopologyToggle() (tea.Model, tea.Cmd) {
	if m.showTopology {
		m.showTopology = false

		return m, nil
	}

	m.generateTopologyView()
	m.showTopology = true
	m.topologyScrollY = 0
	m.closeAllOverlays()
	m.statusMessage = "Topology View - ASCII network diagram"
	m.statusIsError = false

	return m, nil
}
