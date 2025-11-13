package api

import "github.com/krisarmstrong/niac-go/pkg/config"

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

// TopologyLink represents a connection between devices.
type TopologyLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label"`
}

// BuildTopology derives a topology graph from the configuration.
func BuildTopology(cfg *config.Config) Topology {
	nodes := make(map[string]TopologyNode)
	links := make([]TopologyLink, 0)

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

			label := trunk.Interface
			if trunk.RemoteInterface != "" {
				label += " ↔ " + trunk.RemoteInterface
			}

			links = append(links, TopologyLink{
				Source: dev.Name,
				Target: trunk.RemoteDevice,
				Label:  label,
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
