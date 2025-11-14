package api

import (
	"fmt"
	"strings"

	"github.com/krisarmstrong/niac-go/pkg/config"
)

// Topology describes a simple graph for visualization.
type Topology struct {
	Nodes []TopologyNode `json:"nodes"`
	Links []TopologyLink `json:"links"`
}

// TopologyNode represents a device.
type TopologyNode struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// TopologyLink represents a connection between devices with detailed information.
type TopologyLink struct {
	Source          string  `json:"source"`
	Target          string  `json:"target"`
	Label           string  `json:"label"`
	SourceInterface string  `json:"source_interface"`
	TargetInterface string  `json:"target_interface"`
	LinkType        string  `json:"link_type"` // trunk, access, lag, p2p
	VLANs           []int   `json:"vlans"`
	NativeVLAN      int     `json:"native_vlan,omitempty"`
	Speed           int     `json:"speed_mbps,omitempty"`
	Duplex          string  `json:"duplex,omitempty"`
	Status          string  `json:"status"` // up, down, degraded
	Utilization     float64 `json:"utilization_percent,omitempty"`
}

// BuildTopology derives a topology graph from the configuration.
func BuildTopology(cfg *config.Config) Topology {
	nodes := make(map[string]TopologyNode)
	links := make([]TopologyLink, 0)

	// Create interface lookup map for speed/duplex/status info
	interfaceMap := make(map[string]map[string]config.Interface)
	for _, dev := range cfg.Devices {
		interfaceMap[dev.Name] = make(map[string]config.Interface)
		for _, iface := range dev.Interfaces {
			interfaceMap[dev.Name][iface.Name] = iface
		}
	}

	for _, dev := range cfg.Devices {
		nodes[dev.Name] = TopologyNode{
			Name: dev.Name,
			Type: dev.Type,
		}

		for _, trunk := range dev.TrunkPorts {
			if trunk.RemoteDevice == "" {
				continue
			}
			if _, exists := nodes[trunk.RemoteDevice]; !exists {
				nodes[trunk.RemoteDevice] = TopologyNode{
					Name: trunk.RemoteDevice,
					Type: "external",
				}
			}

			// Build label with VLAN info
			label := trunk.Interface
			if trunk.RemoteInterface != "" {
				label += " ↔ " + trunk.RemoteInterface
			}
			if len(trunk.VLANs) > 0 {
				label += fmt.Sprintf(" (VLANs: %s)", formatVLANList(trunk.VLANs))
			}

			// Determine link type
			linkType := "trunk"
			if len(trunk.VLANs) == 1 {
				linkType = "access"
			} else if strings.Contains(strings.ToLower(trunk.Interface), "port-channel") ||
				strings.Contains(strings.ToLower(trunk.Interface), "po") {
				linkType = "lag"
			}

			// Get interface details if available
			var speed int
			var duplex string
			var status string = "up" // Default to up
			if iface, ok := interfaceMap[dev.Name][trunk.Interface]; ok {
				speed = iface.Speed
				duplex = iface.Duplex
				if iface.AdminStatus != "" {
					status = iface.AdminStatus
				}
				if iface.OperStatus != "" {
					status = iface.OperStatus
				}
			}

			links = append(links, TopologyLink{
				Source:          dev.Name,
				Target:          trunk.RemoteDevice,
				Label:           label,
				SourceInterface: trunk.Interface,
				TargetInterface: trunk.RemoteInterface,
				LinkType:        linkType,
				VLANs:           trunk.VLANs,
				NativeVLAN:      trunk.NativeVLAN,
				Speed:           speed,
				Duplex:          duplex,
				Status:          status,
				Utilization:     0.0, // Could be enhanced with real-time metrics later
			})
		}
	}

	topology := Topology{
		Nodes: make([]TopologyNode, 0, len(nodes)),
		Links: links,
	}
	for _, node := range nodes {
		topology.Nodes = append(topology.Nodes, node)
	}
	return topology
}

// formatVLANList formats a list of VLANs for display (e.g., "1-5,10,20")
func formatVLANList(vlans []int) string {
	if len(vlans) == 0 {
		return ""
	}
	if len(vlans) == 1 {
		return fmt.Sprintf("%d", vlans[0])
	}
	if len(vlans) <= 3 {
		// Show all for small lists
		parts := make([]string, len(vlans))
		for i, v := range vlans {
			parts[i] = fmt.Sprintf("%d", v)
		}
		return strings.Join(parts, ",")
	}
	// For longer lists, show count
	return fmt.Sprintf("%d-%d (+%d more)", vlans[0], vlans[len(vlans)-1], len(vlans)-2)
}

// ExportGraphML exports the topology in GraphML format (for yEd, Gephi)
func (t *Topology) ExportGraphML() string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<graphml xmlns="http://graphml.graphdrawing.org/xmlns"` + "\n")
	sb.WriteString(`  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"` + "\n")
	sb.WriteString(`  xsi:schemaLocation="http://graphml.graphdrawing.org/xmlns` + "\n")
	sb.WriteString(`  http://graphml.graphdrawing.org/xmlns/1.0/graphml.xsd">` + "\n")

	// Define keys (attributes)
	sb.WriteString(`  <key id="d0" for="node" attr.name="type" attr.type="string"/>` + "\n")
	sb.WriteString(`  <key id="d1" for="edge" attr.name="label" attr.type="string"/>` + "\n")
	sb.WriteString(`  <key id="d2" for="edge" attr.name="source_interface" attr.type="string"/>` + "\n")
	sb.WriteString(`  <key id="d3" for="edge" attr.name="target_interface" attr.type="string"/>` + "\n")
	sb.WriteString(`  <key id="d4" for="edge" attr.name="link_type" attr.type="string"/>` + "\n")
	sb.WriteString(`  <key id="d5" for="edge" attr.name="vlans" attr.type="string"/>` + "\n")
	sb.WriteString(`  <key id="d6" for="edge" attr.name="speed_mbps" attr.type="int"/>` + "\n")
	sb.WriteString(`  <key id="d7" for="edge" attr.name="status" attr.type="string"/>` + "\n")

	sb.WriteString(`  <graph id="G" edgedefault="undirected">` + "\n")

	// Nodes
	for _, node := range t.Nodes {
		sb.WriteString(fmt.Sprintf(`    <node id="%s">`, escapeXML(node.Name)) + "\n")
		sb.WriteString(fmt.Sprintf(`      <data key="d0">%s</data>`, escapeXML(node.Type)) + "\n")
		sb.WriteString(`    </node>` + "\n")
	}

	// Edges
	for i, link := range t.Links {
		sb.WriteString(fmt.Sprintf(`    <edge id="e%d" source="%s" target="%s">`, i, escapeXML(link.Source), escapeXML(link.Target)) + "\n")
		sb.WriteString(fmt.Sprintf(`      <data key="d1">%s</data>`, escapeXML(link.Label)) + "\n")
		sb.WriteString(fmt.Sprintf(`      <data key="d2">%s</data>`, escapeXML(link.SourceInterface)) + "\n")
		sb.WriteString(fmt.Sprintf(`      <data key="d3">%s</data>`, escapeXML(link.TargetInterface)) + "\n")
		sb.WriteString(fmt.Sprintf(`      <data key="d4">%s</data>`, escapeXML(link.LinkType)) + "\n")
		sb.WriteString(fmt.Sprintf(`      <data key="d5">%s</data>`, escapeXML(formatVLANList(link.VLANs))) + "\n")
		if link.Speed > 0 {
			sb.WriteString(fmt.Sprintf(`      <data key="d6">%d</data>`, link.Speed) + "\n")
		}
		sb.WriteString(fmt.Sprintf(`      <data key="d7">%s</data>`, escapeXML(link.Status)) + "\n")
		sb.WriteString(`    </edge>` + "\n")
	}

	sb.WriteString(`  </graph>` + "\n")
	sb.WriteString(`</graphml>` + "\n")
	return sb.String()
}

// ExportDOT exports the topology in DOT format (for Graphviz)
func (t *Topology) ExportDOT() string {
	var sb strings.Builder
	sb.WriteString("graph niac_topology {\n")
	sb.WriteString("  // Nodes\n")

	// Nodes
	for _, node := range t.Nodes {
		shape := "box"
		if node.Type == "router" {
			shape = "ellipse"
		} else if node.Type == "switch" {
			shape = "box"
		} else if node.Type == "ap" {
			shape = "diamond"
		}
		sb.WriteString(fmt.Sprintf("  \"%s\" [shape=%s, label=\"%s\\n(%s)\"];\n",
			escapeDOT(node.Name), shape, escapeDOT(node.Name), escapeDOT(node.Type)))
	}

	sb.WriteString("\n  // Links\n")

	// Links
	for _, link := range t.Links {
		style := "solid"
		color := "black"
		if link.LinkType == "trunk" {
			style = "bold"
			color = "blue"
		} else if link.LinkType == "lag" {
			style = "bold"
			color = "orange"
		} else if link.LinkType == "access" {
			color = "green"
		}

		if link.Status == "down" {
			style = "dashed"
			color = "red"
		}

		label := fmt.Sprintf("%s-%s", escapeDOT(link.SourceInterface), escapeDOT(link.TargetInterface))
		if len(link.VLANs) > 0 {
			label += fmt.Sprintf("\\nVLANs: %s", escapeDOT(formatVLANList(link.VLANs)))
		}
		if link.Speed > 0 {
			label += fmt.Sprintf("\\n%dMbps", link.Speed)
		}

		sb.WriteString(fmt.Sprintf("  \"%s\" -- \"%s\" [label=\"%s\", style=%s, color=%s];\n",
			escapeDOT(link.Source), escapeDOT(link.Target), label, style, color))
	}

	sb.WriteString("}\n")
	return sb.String()
}

// escapeXML escapes special XML characters
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// escapeDOT escapes special DOT characters
func escapeDOT(s string) string {
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
