package interactive

import (
	"fmt"
	"strings"

	"github.com/krisarmstrong/niac-go/internal/api"
	"github.com/krisarmstrong/niac-go/internal/config"
)

// renderDeviceConfigTabBar renders the tab bar for device config view.
func (m *model) renderDeviceConfigTabBar() string {
	tabs := []string{"General", "Interfaces", "Protocols", "SNMP"}
	var tabBar strings.Builder

	for i, tab := range tabs {
		if i == m.deviceConfigTab {
			tabBar.WriteString(selectedStyle.Render("[" + tab + "]"))
		} else {
			tabBar.WriteString(" " + tab + " ")
		}
		if i < len(tabs)-1 {
			tabBar.WriteString(" | ")
		}
	}
	return tabBar.String()
}

// renderDeviceGeneralTab renders the general tab content.
func renderDeviceGeneralTab(panel *strings.Builder, device *config.Device) {
	panel.WriteString(padPanelLine("Name:        " + device.Name))
	panel.WriteString(padPanelLine("Type:        " + device.Type))
	panel.WriteString(padPanelLine("MAC Address: " + device.MACAddress.String()))

	if len(device.IPAddresses) > 0 {
		panel.WriteString(padPanelLine("IP Address:  " + device.IPAddresses[0].String()))
	}
	if device.VLAN > 0 {
		panel.WriteString(padPanelLine(fmt.Sprintf("VLAN:        %d", device.VLAN)))
	}
	panel.WriteString(padPanelLine("Babble:      " + boolToEnabled(device.Babble)))
}

// renderDeviceInterfacesTab renders the interfaces tab content.
func renderDeviceInterfacesTab(panel *strings.Builder, device *config.Device) {
	if len(device.Interfaces) == 0 {
		panel.WriteString(padPanelLine("No interfaces configured"))
		return
	}

	panel.WriteString(
		padPanelLine(
			fmt.Sprintf("%-15s %-10s %-10s %-8s", "Interface", "Speed", "Duplex", "Status"),
		),
	)
	panel.WriteString(padPanelLine(strings.Repeat("-", deviceConfigTableWidth)))

	for _, iface := range device.Interfaces {
		status := iface.AdminStatus
		if status == "" {
			status = "up"
		}
		panel.WriteString(padPanelLine(fmt.Sprintf("%-15s %-10d %-10s %-8s",
			iface.Name, iface.Speed, iface.Duplex, status)))
	}
}

// renderDeviceProtocolsTab renders the protocols tab content.
func renderDeviceProtocolsTab(panel *strings.Builder, device *config.Device) {
	panel.WriteString(padPanelLine("LLDP:    " + boolToEnabled(device.LLDPConfig != nil)))
	panel.WriteString(padPanelLine("CDP:     " + boolToEnabled(device.CDPConfig != nil)))
	panel.WriteString(padPanelLine("STP:     " + boolToEnabled(device.STPConfig != nil)))
	panel.WriteString(padPanelLine("EDP:     " + boolToEnabled(device.EDPConfig != nil)))
	panel.WriteString(padPanelLine("FDP:     " + boolToEnabled(device.FDPConfig != nil)))
}

// renderDeviceSNMPTab renders the SNMP tab content.
func renderDeviceSNMPTab(panel *strings.Builder, device *config.Device) {
	if device.SNMPConfig.Community == "" {
		panel.WriteString(padPanelLine("SNMP not configured for this device"))
		return
	}
	panel.WriteString(padPanelLine("Community:  " + device.SNMPConfig.Community))
	panel.WriteString(padPanelLine("SysName:    " + device.SNMPConfig.SysName))
	panel.WriteString(padPanelLine("SysDescr:   " + device.SNMPConfig.SysDescr))
	panel.WriteString(padPanelLine("Contact:    " + device.SNMPConfig.SysContact))
	panel.WriteString(padPanelLine("Location:   " + device.SNMPConfig.SysLocation))
}

// renderDeviceConfigContent renders the content for the current tab.
func (m *model) renderDeviceConfigContent(panel *strings.Builder, device *config.Device) {
	if device == nil {
		panel.WriteString(padPanelLine("Select a device with [D] to view configuration"))
		return
	}

	switch m.deviceConfigTab {
	case deviceConfigTabGeneral:
		renderDeviceGeneralTab(panel, device)
	case deviceConfigTabInterface:
		renderDeviceInterfacesTab(panel, device)
	case deviceConfigTabProtocols:
		renderDeviceProtocolsTab(panel, device)
	case deviceConfigTabSNMP:
		renderDeviceSNMPTab(panel, device)
	}
}

// renderDeviceConfig renders the device configuration panel.
func (m *model) renderDeviceConfig() string {
	var panel strings.Builder

	panel.WriteString("+=================================================================+\n")

	var device *config.Device
	deviceName := "No Device Selected"

	if m.selectedDeviceIdx >= 0 && m.selectedDeviceIdx < len(m.cfg.Devices) {
		device = &m.cfg.Devices[m.selectedDeviceIdx]
		deviceName = device.Name
	}

	panel.WriteString(padPanelLine(diffHeaderStyle.Render("Device Configuration: " + deviceName)))
	panel.WriteString("+=================================================================+\n")
	panel.WriteString(padPanelLine(m.renderDeviceConfigTabBar()))
	panel.WriteString("+=================================================================+\n")

	m.renderDeviceConfigContent(&panel, device)

	panel.WriteString("+=================================================================+\n")
	panel.WriteString(padPanelLine("[Tab] Switch Tab  [up/dn] Scroll  [ESC] Close"))
	panel.WriteString("+=================================================================+")

	return panel.String()
}

// renderTopology renders the topology view.
func (m *model) renderTopology() string {
	var panel strings.Builder

	panel.WriteString("+=================================================================+\n")
	panel.WriteString("|                      Network Topology                            |\n")
	panel.WriteString("+=================================================================+\n")

	if m.topologyContent == "" {
		panel.WriteString(padPanelLine("No topology data available"))
		panel.WriteString("+=================================================================+")

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
		panel.WriteString("+=================================================================+\n")
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

	panel.WriteString("+=================================================================+\n")
	panel.WriteString(padPanelLine("[T] or [ESC] Close"))
	panel.WriteString("+=================================================================+")

	return panel.String()
}

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
		sb.WriteString(fmt.Sprintf("  [%s]\n", nodeType))

		for _, node := range nodes {
			sb.WriteString(fmt.Sprintf("    +-- %s\n", node.Name))
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

// generateASCIIDiagram generates the ASCII diagram portion of the topology.
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
