package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
)

func TestCountDevicesByType(t *testing.T) {
	tests := []struct {
		name     string
		nodes    []api.TopologyNode
		expected map[string]int
	}{
		{
			name:     "empty nodes",
			nodes:    []api.TopologyNode{},
			expected: map[string]int{},
		},
		{
			name: "single type",
			nodes: []api.TopologyNode{
				{Name: "sw1", Type: "switch"},
				{Name: "sw2", Type: "switch"},
			},
			expected: map[string]int{"switch": 2},
		},
		{
			name: "multiple types",
			nodes: []api.TopologyNode{
				{Name: "rtr1", Type: "router"},
				{Name: "sw1", Type: "switch"},
				{Name: "sw2", Type: "switch"},
				{Name: "ap1", Type: "access-point"},
			},
			expected: map[string]int{"router": 1, "switch": 2, "access-point": 1},
		},
		{
			name: "empty type defaults to unknown",
			nodes: []api.TopologyNode{
				{Name: "dev1", Type: ""},
				{Name: "dev2", Type: ""},
			},
			expected: map[string]int{"unknown": 2},
		},
		{
			name: "mixed with empty type",
			nodes: []api.TopologyNode{
				{Name: "rtr1", Type: "router"},
				{Name: "dev1", Type: ""},
			},
			expected: map[string]int{"router": 1, "unknown": 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countDevicesByType(tt.nodes)

			if len(result) != len(tt.expected) {
				t.Errorf("countDevicesByType() returned %d types, want %d", len(result), len(tt.expected))
			}

			for deviceType, count := range tt.expected {
				if result[deviceType] != count {
					t.Errorf("countDevicesByType()[%q] = %d, want %d", deviceType, result[deviceType], count)
				}
			}
		})
	}
}

func TestExportTopologyJSON(t *testing.T) {
	topology := &api.Topology{
		Nodes: []api.TopologyNode{
			{Name: "router-1", Type: "router"},
			{Name: "switch-1", Type: "switch"},
		},
		Links: []api.TopologyLink{
			{
				Source:          "router-1",
				Target:          "switch-1",
				SourceInterface: "Gi0/1",
				TargetInterface: "Gi0/1",
				LinkType:        "ethernet",
				Speed:           1000,
				Duplex:          "full",
				Status:          "up",
			},
		},
	}

	output, err := exportTopologyJSON(topology)
	if err != nil {
		t.Fatalf("exportTopologyJSON() error = %v", err)
	}

	// Verify it's valid JSON
	var parsed TopologyExport
	if jsonErr := json.Unmarshal([]byte(output), &parsed); jsonErr != nil {
		t.Fatalf("Output is not valid JSON: %v", jsonErr)
	}

	// Verify structure
	if len(parsed.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(parsed.Nodes))
	}
	if len(parsed.Links) != 1 {
		t.Errorf("Expected 1 link, got %d", len(parsed.Links))
	}
	if parsed.Summary.TotalNodes != 2 {
		t.Errorf("Expected TotalNodes=2, got %d", parsed.Summary.TotalNodes)
	}
	if parsed.Summary.TotalLinks != 1 {
		t.Errorf("Expected TotalLinks=1, got %d", parsed.Summary.TotalLinks)
	}
	if parsed.Summary.DevicesByType["router"] != 1 {
		t.Error("Expected 1 router in devices_by_type")
	}
	if parsed.Summary.LinksByType["ethernet"] != 1 {
		t.Error("Expected 1 ethernet link in links_by_type")
	}

	// Verify link details
	if parsed.Links[0].Source != "router-1" {
		t.Errorf("Expected link source=router-1, got %s", parsed.Links[0].Source)
	}
	if parsed.Links[0].SpeedMbps != 1000 {
		t.Errorf("Expected speed=1000, got %d", parsed.Links[0].SpeedMbps)
	}
}

func TestExportTopologyJSONEmpty(t *testing.T) {
	topology := &api.Topology{
		Nodes: []api.TopologyNode{},
		Links: []api.TopologyLink{},
	}

	output, err := exportTopologyJSON(topology)
	if err != nil {
		t.Fatalf("exportTopologyJSON() error = %v", err)
	}

	if !strings.Contains(output, `"nodes"`) {
		t.Error("Expected 'nodes' key in JSON output")
	}
}

func TestExportTopologyYAML(t *testing.T) {
	topology := &api.Topology{
		Nodes: []api.TopologyNode{
			{Name: "router-1", Type: "router"},
			{Name: "switch-1", Type: "switch"},
		},
		Links: []api.TopologyLink{
			{
				Source:          "router-1",
				Target:          "switch-1",
				SourceInterface: "Gi0/1",
				TargetInterface: "Gi0/1",
				LinkType:        "ethernet",
				Speed:           1000,
				Status:          "up",
			},
		},
	}

	output, err := exportTopologyYAML(topology)
	if err != nil {
		t.Fatalf("exportTopologyYAML() error = %v", err)
	}

	// Verify YAML contains expected fields
	expectedStrings := []string{
		"router-1",
		"switch-1",
		"router",
		"ethernet",
		"total_nodes: 2",
		"total_links: 1",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("YAML output missing %q", expected)
		}
	}
}

func TestExportTopologyYAMLEmpty(t *testing.T) {
	topology := &api.Topology{
		Nodes: []api.TopologyNode{},
		Links: []api.TopologyLink{},
	}

	output, err := exportTopologyYAML(topology)
	if err != nil {
		t.Fatalf("exportTopologyYAML() error = %v", err)
	}

	if !strings.Contains(output, "nodes:") {
		t.Error("Expected 'nodes:' key in YAML output")
	}
}

func TestTopologyExportStructs(t *testing.T) {
	// Verify struct field tags are correct by marshaling and unmarshaling
	export := TopologyExport{
		Nodes: []TopologyNodeExport{
			{Name: "test-node", Type: "router"},
		},
		Links: []TopologyLinkExport{
			{
				Source:          "a",
				Target:          "b",
				SourceInterface: "Gi0/1",
				TargetInterface: "Gi0/2",
				LinkType:        "ethernet",
				VLANs:           []int{10, 20},
				NativeVLAN:      1,
				SpeedMbps:       1000,
				Duplex:          "full",
				Status:          "up",
				Utilization:     75.5,
			},
		},
		Summary: TopologySummary{
			TotalNodes:    1,
			TotalLinks:    1,
			DevicesByType: map[string]int{"router": 1},
			LinksByType:   map[string]int{"ethernet": 1},
		},
	}

	data, err := json.Marshal(export)
	if err != nil {
		t.Fatalf("Failed to marshal TopologyExport: %v", err)
	}

	var decoded TopologyExport
	if err = json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal TopologyExport: %v", err)
	}

	if decoded.Nodes[0].Name != "test-node" {
		t.Errorf("Node name mismatch: got %q", decoded.Nodes[0].Name)
	}
	if decoded.Links[0].Utilization != 75.5 {
		t.Errorf("Utilization mismatch: got %v", decoded.Links[0].Utilization)
	}
}

func TestTopologyWithVLANs(t *testing.T) {
	topology := &api.Topology{
		Nodes: []api.TopologyNode{
			{Name: "sw1", Type: "switch"},
			{Name: "sw2", Type: "switch"},
		},
		Links: []api.TopologyLink{
			{
				Source:     "sw1",
				Target:     "sw2",
				LinkType:   "trunk",
				VLANs:      []int{10, 20, 30},
				NativeVLAN: 1,
				Status:     "up",
			},
		},
	}

	output, err := exportTopologyJSON(topology)
	if err != nil {
		t.Fatalf("exportTopologyJSON() error = %v", err)
	}

	if !strings.Contains(output, "10") {
		t.Error("Expected VLAN 10 in output")
	}
	if !strings.Contains(output, "native_vlan") {
		t.Error("Expected native_vlan in output")
	}
}
