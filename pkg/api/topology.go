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
