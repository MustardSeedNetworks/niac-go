package topology

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// VLAN display formatting constants.
const (
	vlanDisplayThreshold = 3 // max VLANs to show individually before summarizing
	vlanRangeEndpoints   = 2 // number of endpoints shown in range (first and last)
)

// Link type constants.
const (
	linkTypeTrunk = "trunk"
)

// Device type constants for topology.
const (
	deviceTypeRouter       = "router"
	deviceTypeSwitch       = "switch"
	deviceTypeLayer3Switch = "layer3-switch"
	deviceTypeAP           = "ap"
)

// Graph describes a simple graph for visualization.
type Graph struct {
	Nodes []Node `json:"nodes"`
	Links []Link `json:"links"`
}

// Node represents a device.
type Node struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Link represents a connection between devices with detailed information.
type Link struct {
	Source          string  `json:"source"`
	Target          string  `json:"target"`
	Label           string  `json:"label"`
	SourceInterface string  `json:"sourceInterface"`
	TargetInterface string  `json:"targetInterface"`
	LinkType        string  `json:"linkType"` // trunk, access, lag, p2p
	VLANs           []int   `json:"vlans"`
	NativeVLAN      int     `json:"nativeVlan,omitempty"`
	Speed           int     `json:"speedMbps,omitempty"`
	Duplex          string  `json:"duplex,omitempty"`
	Status          string  `json:"status"` // up, down, degraded
	Utilization     float64 `json:"utilizationPercent,omitempty"`
}

// buildInterfaceMap creates a lookup map of device interfaces.
func buildInterfaceMap(devices []config.Device) map[string]map[string]config.Interface {
	interfaceMap := make(map[string]map[string]config.Interface)
	for _, dev := range devices {
		interfaceMap[dev.Name] = make(map[string]config.Interface)
		for _, iface := range dev.Interfaces {
			interfaceMap[dev.Name][iface.Name] = iface
		}
	}

	return interfaceMap
}

// determineLinkType determines the type of a trunk link.
func determineLinkType(trunk config.TrunkPort) string {
	if len(trunk.VLANs) == 1 {
		return "access"
	}

	lowerIface := strings.ToLower(trunk.Interface)
	if strings.Contains(lowerIface, "port-channel") || strings.Contains(lowerIface, "po") {
		return "lag"
	}

	return linkTypeTrunk
}

// buildTrunkLabel creates the label for a trunk link.
func buildTrunkLabel(trunk config.TrunkPort) string {
	label := abbreviateInterface(trunk.Interface)
	if trunk.RemoteInterface != "" {
		label += " ↔ " + abbreviateInterface(trunk.RemoteInterface)
	}

	if len(trunk.VLANs) > 0 {
		label += fmt.Sprintf(" (VLANs %s)", formatVLANList(trunk.VLANs))
	}

	return label
}

// abbreviateInterface shortens the long Cisco-style interface names
// (GigabitEthernet0/1 → Gi0/1) that hog edge-label space in the
// topology view. Covers the IOS-style prefixes most templates use;
// anything unrecognised passes through unchanged.
func abbreviateInterface(name string) string {
	abbrevs := []struct{ long, short string }{
		{"TenGigabitEthernet", "Te"},
		{"TwentyFiveGigE", "Twe"},
		{"FortyGigabitEthernet", "Fo"},
		{"HundredGigE", "Hu"},
		{"GigabitEthernet", "Gi"},
		{"FastEthernet", "Fa"},
		{"Ethernet", "Eth"},
		{"Loopback", "Lo"},
		{"Port-channel", "Po"},
		{"Management", "Mgmt"},
		{"Vlan", "Vl"},
	}
	for _, a := range abbrevs {
		if strings.HasPrefix(name, a.long) {
			return a.short + name[len(a.long):]
		}
	}
	return name
}

// getInterfaceDetails retrieves presentation data from the authored interface.
func getInterfaceDetails(
	interfaceMap map[string]map[string]config.Interface,
	deviceName, ifaceName string,
) (int, string, string, float64) {
	const defaultStatus = "up"

	iface, ok := interfaceMap[deviceName][ifaceName]
	if !ok {
		return 0, "", defaultStatus, 0
	}

	status := defaultStatus
	if iface.AdminStatus != "" {
		status = iface.AdminStatus
	}

	if iface.OperStatus != "" {
		status = iface.OperStatus
	}

	return iface.Speed, iface.Duplex, status, max(iface.InUtilization, iface.OutUtilization)
}

func sortedEndpointKey(a, aInterface, b, bInterface string) string {
	left, right := a+"|"+aInterface, b+"|"+bInterface
	if left <= right {
		return left + "|" + right
	}
	return right + "|" + left
}

// processTrunkPort creates a Link from a trunk port configuration.
func processTrunkPort(
	trunk config.TrunkPort,
	deviceName string,
	interfaceMap map[string]map[string]config.Interface,
) Link {
	speed, duplex, status, utilization := getInterfaceDetails(
		interfaceMap,
		deviceName,
		trunk.Interface,
	)

	return Link{
		Source: deviceName,
		Target: trunk.RemoteDevice,
		Label:  buildTrunkLabel(trunk),
		// Abbreviate interface names everywhere they're surfaced —
		// the per-side labels on the topology edge use these fields
		// directly, not the combined label, so they need the same
		// shortening (GigabitEthernet0/1 → Gi0/1).
		SourceInterface: abbreviateInterface(trunk.Interface),
		TargetInterface: abbreviateInterface(trunk.RemoteInterface),
		LinkType:        determineLinkType(trunk),
		VLANs:           trunk.VLANs,
		NativeVLAN:      trunk.NativeVLAN,
		Speed:           speed,
		Duplex:          duplex,
		Status:          status,
		Utilization:     utilization,
	}
}

// Build derives a topology graph from the configuration.
//
// Trunk links are inherently bidirectional — both switches declare a
// trunk_port pointing at each other, so we'd otherwise emit two
// Link entries for one physical wire and ReactFlow would draw
// a second edge looping back around the cards. We dedupe via a canonical
// endpoint key so reciprocal declarations collapse without hiding parallel
// links between the same pair of devices.
func Build(cfg *config.Config) Graph {
	nodes := make(map[string]Node)
	links := make([]Link, 0)
	interfaceMap := buildInterfaceMap(cfg.Devices)
	seenLinks := make(map[string]bool)

	for _, dev := range cfg.Devices {
		nodes[dev.Name] = Node{
			Name: dev.Name,
			Type: dev.Type,
		}

		for _, trunk := range dev.TrunkPorts {
			if trunk.RemoteDevice == "" {
				continue
			}

			// Sorted endpoint pair is the dedupe key so A↔B and B↔A
			// collapse to one entry. First-seen direction wins so the
			// Source/Target on the emitted link matches the device that
			// declared the trunk_port we found first in the YAML.
			pair := sortedEndpointKey(
				dev.Name,
				trunk.Interface,
				trunk.RemoteDevice,
				trunk.RemoteInterface,
			)
			if seenLinks[pair] {
				continue
			}
			seenLinks[pair] = true

			if _, exists := nodes[trunk.RemoteDevice]; !exists {
				nodes[trunk.RemoteDevice] = Node{
					Name: trunk.RemoteDevice,
					Type: "external",
				}
			}

			links = append(links, processTrunkPort(trunk, dev.Name, interfaceMap))
		}
	}

	topology := Graph{
		Nodes: make([]Node, 0, len(nodes)),
		Links: links,
	}
	for _, node := range nodes {
		topology.Nodes = append(topology.Nodes, node)
	}

	return topology
}

// formatVLANList formats a list of VLANs for display (e.g., "1-5,10,20").
func formatVLANList(vlans []int) string {
	if len(vlans) == 0 {
		return ""
	}

	if len(vlans) == 1 {
		return strconv.Itoa(vlans[0])
	}

	if len(vlans) <= vlanDisplayThreshold {
		// Show all for small lists
		parts := make([]string, len(vlans))
		for i, v := range vlans {
			parts[i] = strconv.Itoa(v)
		}

		return strings.Join(parts, ",")
	}
	// For longer lists, show count
	return fmt.Sprintf(
		"%d-%d (+%d more)",
		vlans[0],
		vlans[len(vlans)-1],
		len(vlans)-vlanRangeEndpoints,
	)
}

// ExportGraphML exports the topology in GraphML format (for yEd, Gephi).
func (t *Graph) ExportGraphML() string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<graphml xmlns="http://graphml.graphdrawing.org/xmlns"` + "\n")
	sb.WriteString(`  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"` + "\n")
	sb.WriteString(`  xsi:schemaLocation="http://graphml.graphdrawing.org/xmlns` + "\n")
	sb.WriteString(`  http://graphml.graphdrawing.org/xmlns/1.0/graphml.xsd">` + "\n")

	// Define keys (attributes)
	sb.WriteString(`  <key id="d0" for="node" attr.name="type" attr.type="string"/>` + "\n")
	sb.WriteString(`  <key id="d1" for="edge" attr.name="label" attr.type="string"/>` + "\n")
	sb.WriteString(
		`  <key id="d2" for="edge" attr.name="source_interface" attr.type="string"/>` + "\n",
	)
	sb.WriteString(
		`  <key id="d3" for="edge" attr.name="target_interface" attr.type="string"/>` + "\n",
	)
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
		sb.WriteString(
			fmt.Sprintf(
				`    <edge id="e%d" source="%s" target="%s">`,
				i,
				escapeXML(link.Source),
				escapeXML(link.Target),
			) + "\n",
		)
		sb.WriteString(fmt.Sprintf(`      <data key="d1">%s</data>`, escapeXML(link.Label)) + "\n")
		sb.WriteString(
			fmt.Sprintf(`      <data key="d2">%s</data>`, escapeXML(link.SourceInterface)) + "\n",
		)
		sb.WriteString(
			fmt.Sprintf(`      <data key="d3">%s</data>`, escapeXML(link.TargetInterface)) + "\n",
		)
		sb.WriteString(
			fmt.Sprintf(`      <data key="d4">%s</data>`, escapeXML(link.LinkType)) + "\n",
		)
		sb.WriteString(
			fmt.Sprintf(
				`      <data key="d5">%s</data>`,
				escapeXML(formatVLANList(link.VLANs)),
			) + "\n",
		)

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

// ExportDOT exports the topology in DOT format (for Graphviz).
func (t *Graph) ExportDOT() string {
	var sb strings.Builder
	sb.WriteString("graph niac_topology {\n")
	sb.WriteString("  // Nodes\n")

	// Nodes
	for _, node := range t.Nodes {
		var shape string

		switch node.Type {
		case deviceTypeRouter:
			shape = "ellipse"
		case deviceTypeSwitch, deviceTypeLayer3Switch:
			shape = "box"
		case deviceTypeAP:
			shape = "diamond"
		default:
			shape = "box"
		}

		fmt.Fprintf(&sb, "  \"%s\" [shape=%s, label=\"%s\\n(%s)\"];\n",
			escapeDOT(node.Name), shape, escapeDOT(node.Name), escapeDOT(node.Type))
	}

	sb.WriteString("\n  // Links\n")

	// Links
	for _, link := range t.Links {
		style := "solid"
		color := "black"

		switch link.LinkType {
		case linkTypeTrunk:
			style = "bold"
			color = "blue"
		case "lag":
			style = "bold"
			color = "orange"
		case "access":
			color = "green"
		}

		if link.Status == "down" {
			style = "dashed"
			color = "red"
		}

		label := fmt.Sprintf(
			"%s-%s",
			escapeDOT(link.SourceInterface),
			escapeDOT(link.TargetInterface),
		)
		if len(link.VLANs) > 0 {
			label += "\\nVLANs: " + escapeDOT(formatVLANList(link.VLANs))
		}

		if link.Speed > 0 {
			label += fmt.Sprintf("\\n%dMbps", link.Speed)
		}

		fmt.Fprintf(&sb, "  \"%s\" -- \"%s\" [label=\"%s\", style=%s, color=%s];\n",
			escapeDOT(link.Source), escapeDOT(link.Target), label, style, color)
	}

	sb.WriteString("}\n")

	return sb.String()
}

// escapeXML escapes special XML characters.
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")

	return s
}

// escapeDOT escapes special DOT characters.
func escapeDOT(s string) string {
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")

	return s
}
