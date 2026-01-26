package api

import (
	"testing"

	"github.com/krisarmstrong/niac-go/internal/config"
)

func TestBuildInterfaceMap(t *testing.T) {
	devices := []config.Device{
		{
			Name: "router1",
			Interfaces: []config.Interface{
				{Name: "eth0"},
				{Name: "eth1"},
			},
		},
		{
			Name: "switch1",
			Interfaces: []config.Interface{
				{Name: "ge-0/0/0"},
			},
		},
	}

	result := buildInterfaceMap(devices)

	// Check router1 interfaces
	if _, ok := result["router1"]; !ok {
		t.Error("missing router1 in interface map")
	}
	if _, ok := result["router1"]["eth0"]; !ok {
		t.Error("missing eth0 interface for router1")
	}
	if _, ok := result["router1"]["eth1"]; !ok {
		t.Error("missing eth1 interface for router1")
	}

	// Check switch1 interfaces
	if _, ok := result["switch1"]; !ok {
		t.Error("missing switch1 in interface map")
	}
	if _, ok := result["switch1"]["ge-0/0/0"]; !ok {
		t.Error("missing ge-0/0/0 interface for switch1")
	}
}

func TestBuildInterfaceMapEmpty(t *testing.T) {
	result := buildInterfaceMap([]config.Device{})

	if len(result) != 0 {
		t.Errorf("expected empty map, got %d entries", len(result))
	}
}

func TestDetermineLinkType(t *testing.T) {
	tests := []struct {
		name  string
		trunk config.TrunkPort
		want  string
	}{
		{
			name:  "single VLAN is access",
			trunk: config.TrunkPort{Interface: "eth0", VLANs: []int{100}},
			want:  "access",
		},
		{
			name:  "multiple VLANs is trunk",
			trunk: config.TrunkPort{Interface: "eth0", VLANs: []int{100, 200, 300}},
			want:  "trunk",
		},
		{
			name:  "port-channel is lag",
			trunk: config.TrunkPort{Interface: "Port-Channel1", VLANs: []int{100, 200}},
			want:  "lag",
		},
		{
			name:  "po is lag",
			trunk: config.TrunkPort{Interface: "Po1", VLANs: []int{100, 200}},
			want:  "lag",
		},
		{
			name:  "no VLANs is trunk",
			trunk: config.TrunkPort{Interface: "eth0", VLANs: []int{}},
			want:  "trunk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineLinkType(tt.trunk)
			if got != tt.want {
				t.Errorf("determineLinkType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildTrunkLabel(t *testing.T) {
	tests := []struct {
		name  string
		trunk config.TrunkPort
		want  string
	}{
		{
			name:  "interface only",
			trunk: config.TrunkPort{Interface: "eth0"},
			want:  "eth0",
		},
		{
			name:  "with remote interface",
			trunk: config.TrunkPort{Interface: "eth0", RemoteInterface: "eth1"},
			want:  "eth0 ↔ eth1",
		},
		{
			name:  "with single VLAN",
			trunk: config.TrunkPort{Interface: "eth0", VLANs: []int{100}},
			want:  "eth0 (VLANs: 100)",
		},
		{
			name:  "with multiple VLANs",
			trunk: config.TrunkPort{Interface: "eth0", VLANs: []int{100, 200, 300}},
			want:  "eth0 (VLANs: 100,200,300)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildTrunkLabel(tt.trunk)
			if got != tt.want {
				t.Errorf("buildTrunkLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatVLANList(t *testing.T) {
	tests := []struct {
		name  string
		vlans []int
		want  string
	}{
		{
			name:  "empty list",
			vlans: []int{},
			want:  "",
		},
		{
			name:  "single VLAN",
			vlans: []int{100},
			want:  "100",
		},
		{
			name:  "two VLANs",
			vlans: []int{100, 200},
			want:  "100,200",
		},
		{
			name:  "three VLANs at threshold",
			vlans: []int{100, 200, 300},
			want:  "100,200,300",
		},
		{
			name:  "more than threshold",
			vlans: []int{100, 200, 300, 400},
			want:  "100-400 (+2 more)",
		},
		{
			name:  "many VLANs",
			vlans: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
			want:  "10-100 (+8 more)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatVLANList(tt.vlans)
			if got != tt.want {
				t.Errorf("formatVLANList() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildTopology(t *testing.T) {
	cfg := &config.Config{
		Devices: []config.Device{
			{
				Name: "router1",
				Type: "router",
			},
			{
				Name: "switch1",
				Type: "switch",
				TrunkPorts: []config.TrunkPort{
					{
						Interface:       "ge-0/0/1",
						RemoteDevice:    "router1",
						RemoteInterface: "eth0",
						VLANs:           []int{100, 200},
					},
				},
			},
		},
	}

	topology := BuildTopology(cfg)

	// Check nodes
	if len(topology.Nodes) != 2 {
		t.Errorf("nodes count = %d, want 2", len(topology.Nodes))
	}

	// Check that nodes have correct types
	nodeMap := make(map[string]string)
	for _, node := range topology.Nodes {
		nodeMap[node.Name] = node.Type
	}

	if nodeMap["router1"] != "router" {
		t.Errorf("router1 type = %q, want %q", nodeMap["router1"], "router")
	}
	if nodeMap["switch1"] != "switch" {
		t.Errorf("switch1 type = %q, want %q", nodeMap["switch1"], "switch")
	}

	// Check links
	if len(topology.Links) != 1 {
		t.Errorf("links count = %d, want 1", len(topology.Links))
	}

	if len(topology.Links) > 0 {
		link := topology.Links[0]
		if link.Source != "switch1" || link.Target != "router1" {
			t.Errorf("link = %s -> %s, want switch1 -> router1", link.Source, link.Target)
		}
	}
}

func TestBuildTopologyEmptyConfig(t *testing.T) {
	cfg := &config.Config{
		Devices: []config.Device{},
	}

	topology := BuildTopology(cfg)

	if len(topology.Nodes) != 0 {
		t.Errorf("nodes count = %d, want 0", len(topology.Nodes))
	}
	if len(topology.Links) != 0 {
		t.Errorf("links count = %d, want 0", len(topology.Links))
	}
}

func TestTopologyNodeJSON(t *testing.T) {
	node := TopologyNode{
		Name: "router1",
		Type: "router",
	}

	if node.Name != "router1" {
		t.Errorf("Name = %q, want %q", node.Name, "router1")
	}
	if node.Type != "router" {
		t.Errorf("Type = %q, want %q", node.Type, "router")
	}
}

func TestTopologyLinkFields(t *testing.T) {
	link := TopologyLink{
		Source:          "switch1",
		Target:          "router1",
		Label:           "trunk",
		SourceInterface: "ge-0/0/1",
		TargetInterface: "eth0",
		LinkType:        "trunk",
		VLANs:           []int{100, 200},
		NativeVLAN:      1,
		Speed:           10000,
		Duplex:          "full",
		Status:          "up",
		Utilization:     45.5,
	}

	if link.Source != "switch1" {
		t.Errorf("Source = %q, want %q", link.Source, "switch1")
	}
	if link.LinkType != "trunk" {
		t.Errorf("LinkType = %q, want %q", link.LinkType, "trunk")
	}
	if link.Status != "up" {
		t.Errorf("Status = %q, want %q", link.Status, "up")
	}
	if len(link.VLANs) != 2 {
		t.Errorf("VLANs count = %d, want 2", len(link.VLANs))
	}
}
