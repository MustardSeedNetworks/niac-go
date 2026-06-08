package api

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
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
			want:  "eth0 (VLANs 100)",
		},
		{
			name:  "with multiple VLANs",
			trunk: config.TrunkPort{Interface: "eth0", VLANs: []int{100, 200, 300}},
			want:  "eth0 (VLANs 100,200,300)",
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
	if link.Target != "router1" {
		t.Errorf("Target = %q, want %q", link.Target, "router1")
	}
	if link.Label != "trunk" {
		t.Errorf("Label = %q, want %q", link.Label, "trunk")
	}
	if link.SourceInterface != "ge-0/0/1" {
		t.Errorf("SourceInterface = %q, want %q", link.SourceInterface, "ge-0/0/1")
	}
	if link.TargetInterface != "eth0" {
		t.Errorf("TargetInterface = %q, want %q", link.TargetInterface, "eth0")
	}
	if link.LinkType != "trunk" {
		t.Errorf("LinkType = %q, want %q", link.LinkType, "trunk")
	}
	if link.NativeVLAN != 1 {
		t.Errorf("NativeVLAN = %d, want %d", link.NativeVLAN, 1)
	}
	if link.Speed != 10000 {
		t.Errorf("Speed = %d, want %d", link.Speed, 10000)
	}
	if link.Duplex != "full" {
		t.Errorf("Duplex = %q, want %q", link.Duplex, "full")
	}
	if link.Status != "up" {
		t.Errorf("Status = %q, want %q", link.Status, "up")
	}
	if link.Utilization != 45.5 {
		t.Errorf("Utilization = %f, want %f", link.Utilization, 45.5)
	}
	if len(link.VLANs) != 2 {
		t.Errorf("VLANs count = %d, want 2", len(link.VLANs))
	}
}

func TestEscapeXML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "no escaping needed", input: "hello", want: "hello"},
		{name: "ampersand", input: "foo & bar", want: "foo &amp; bar"},
		{name: "less than", input: "a < b", want: "a &lt; b"},
		{name: "greater than", input: "a > b", want: "a &gt; b"},
		{name: "double quote", input: `say "hello"`, want: "say &quot;hello&quot;"},
		{name: "single quote", input: "it's", want: "it&apos;s"},
		{name: "multiple special chars", input: `<tag attr="value">`, want: "&lt;tag attr=&quot;value&quot;&gt;"},
		{name: "empty string", input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeXML(tt.input)
			if got != tt.want {
				t.Errorf("escapeXML(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEscapeDOT(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "no escaping needed", input: "hello", want: "hello"},
		{name: "double quote", input: `say "hello"`, want: `say \"hello\"`},
		{name: "newline", input: "line1\nline2", want: "line1\\nline2"},
		{name: "both", input: "\"quote\"\nnewline", want: "\\\"quote\\\"\\nnewline"},
		{name: "empty string", input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeDOT(tt.input)
			if got != tt.want {
				t.Errorf("escapeDOT(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExportGraphML(t *testing.T) {
	topology := &Topology{
		Nodes: []TopologyNode{
			{Name: "router1", Type: "router"},
			{Name: "switch1", Type: "switch"},
		},
		Links: []TopologyLink{
			{
				Source:          "switch1",
				Target:          "router1",
				Label:           "trunk",
				SourceInterface: "ge-0/0/1",
				TargetInterface: "eth0",
				LinkType:        "trunk",
				VLANs:           []int{100, 200},
				Speed:           10000,
				Status:          "up",
			},
		},
	}

	result := topology.ExportGraphML()

	// Check XML header
	if !contains(result, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("Missing XML header")
	}

	// Check graphml namespace
	if !contains(result, `<graphml xmlns="http://graphml.graphdrawing.org/xmlns"`) {
		t.Error("Missing graphml namespace")
	}

	// Check nodes
	if !contains(result, `<node id="router1">`) {
		t.Error("Missing router1 node")
	}
	if !contains(result, `<node id="switch1">`) {
		t.Error("Missing switch1 node")
	}

	// Check edge
	if !contains(result, `<edge id="e0" source="switch1" target="router1">`) {
		t.Error("Missing edge from switch1 to router1")
	}

	// Check data keys
	if !contains(result, `<data key="d6">10000</data>`) {
		t.Error("Missing speed data")
	}
}

func TestExportDOT(t *testing.T) {
	topology := &Topology{
		Nodes: []TopologyNode{
			{Name: "router1", Type: "router"},
			{Name: "switch1", Type: "switch"},
			{Name: "ap1", Type: "ap"},
			{Name: "device1", Type: "unknown"},
		},
		Links: []TopologyLink{
			{
				Source:          "switch1",
				Target:          "router1",
				SourceInterface: "ge-0/0/1",
				TargetInterface: "eth0",
				LinkType:        "trunk",
				VLANs:           []int{100, 200},
				Speed:           10000,
				Status:          "up",
			},
			{
				Source:          "switch1",
				Target:          "ap1",
				SourceInterface: "ge-0/0/2",
				TargetInterface: "eth0",
				LinkType:        "access",
				Status:          "down",
			},
			{
				Source:          "switch1",
				Target:          "device1",
				SourceInterface: "ge-0/0/3",
				TargetInterface: "eth0",
				LinkType:        "lag",
				Status:          "up",
			},
		},
	}

	result := topology.ExportDOT()

	// Check DOT header
	if !contains(result, "graph niac_topology {") {
		t.Error("Missing DOT header")
	}

	// Check node shapes
	if !contains(result, `"router1" [shape=ellipse`) {
		t.Error("Missing router1 with ellipse shape")
	}
	if !contains(result, `"switch1" [shape=box`) {
		t.Error("Missing switch1 with box shape")
	}
	if !contains(result, `"ap1" [shape=diamond`) {
		t.Error("Missing ap1 with diamond shape")
	}

	// Check link styles
	if !contains(result, "style=bold, color=blue") {
		t.Error("Missing trunk style (bold/blue)")
	}
	if !contains(result, "style=dashed, color=red") {
		t.Error("Missing down link style (dashed/red)")
	}
	if !contains(result, "color=orange") {
		t.Error("Missing lag style (orange)")
	}

	// Check closing brace
	if !contains(result, "}") {
		t.Error("Missing closing brace")
	}
}

func TestExportGraphMLEmpty(t *testing.T) {
	topology := &Topology{
		Nodes: []TopologyNode{},
		Links: []TopologyLink{},
	}

	result := topology.ExportGraphML()

	// Should still produce valid GraphML
	if !contains(result, `<graphml`) {
		t.Error("Missing graphml tag in empty topology")
	}
	if !contains(result, `</graphml>`) {
		t.Error("Missing closing graphml tag")
	}
}

func TestExportDOTEmpty(t *testing.T) {
	topology := &Topology{
		Nodes: []TopologyNode{},
		Links: []TopologyLink{},
	}

	result := topology.ExportDOT()

	// Should still produce valid DOT
	if !contains(result, "graph niac_topology {") {
		t.Error("Missing DOT header in empty topology")
	}
	if !contains(result, "}") {
		t.Error("Missing closing brace")
	}
}

// helper function for string contains.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
