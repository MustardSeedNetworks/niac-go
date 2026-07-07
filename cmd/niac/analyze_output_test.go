package main

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/walkanalysis"
)

// TestOutputJSON tests JSON output of walk analysis (writes to stdout).
func TestOutputJSON(t *testing.T) {
	analysis := &walkanalysis.Analysis{
		Device: walkanalysis.Device{
			SysName:  "test-switch",
			SysDescr: "Cisco IOS",
		},
		Interfaces: []walkanalysis.Interface{
			{Index: 1, Name: "Gi0/1", Type: "ethernet"},
		},
		Neighbors: []walkanalysis.Neighbor{},
		Statistics: walkanalysis.Stats{
			TotalInterfaces:    1,
			PhysicalInterfaces: 1,
		},
	}

	// Should not error or panic
	err := outputJSON(analysis)
	if err != nil {
		t.Errorf("outputJSON() error = %v", err)
	}
}

// TestOutputYAML tests YAML output of walk analysis.
func TestOutputYAML(t *testing.T) {
	analysis := &walkanalysis.Analysis{
		Device: walkanalysis.Device{
			SysName:  "test-router",
			SysDescr: "Cisco IOS XE",
		},
		Interfaces: []walkanalysis.Interface{
			{Index: 1, Name: "Gi0/0/0", Type: "ethernet"},
			{Index: 2, Name: "Loopback0", Type: "loopback"},
		},
		Neighbors: []walkanalysis.Neighbor{},
		Statistics: walkanalysis.Stats{
			TotalInterfaces:    2,
			PhysicalInterfaces: 1,
			LogicalInterfaces:  1,
		},
	}

	err := outputYAML(analysis)
	if err != nil {
		t.Errorf("outputYAML() error = %v", err)
	}
}

// TestOutputText tests text output of walk analysis.
func TestOutputText(t *testing.T) {
	analysis := &walkanalysis.Analysis{
		Device: walkanalysis.Device{
			SysName:     "test-switch",
			SysDescr:    "Cisco IOS Software",
			SysContact:  "admin@test.com",
			SysLocation: "Building A",
		},
		Interfaces: []walkanalysis.Interface{
			{
				Index:       1,
				Name:        "Gi0/1",
				Type:        "ethernet",
				Speed:       1000000000,
				AdminStatus: "up",
				OperStatus:  "up",
				MACAddress:  "00:11:22:33:44:55",
			},
			{Index: 2, Name: "Loopback0", Type: "loopback"},
		},
		Neighbors: []walkanalysis.Neighbor{
			{
				LocalInterface:  "Gi0/1",
				RemoteDevice:    "core-rtr",
				RemoteInterface: "Gi1/0",
				Protocol:        "lldp",
			},
		},
		Statistics: walkanalysis.Stats{
			TotalInterfaces:    2,
			PhysicalInterfaces: 1,
			LogicalInterfaces:  1,
			TotalNeighbors:     1,
		},
	}

	err := outputText(analysis)
	if err != nil {
		t.Errorf("outputText() error = %v", err)
	}
}

// TestOutputTextMinimal tests text output with minimal data.
func TestOutputTextMinimal(t *testing.T) {
	analysis := &walkanalysis.Analysis{
		Device:     walkanalysis.Device{SysName: "dev1"},
		Interfaces: []walkanalysis.Interface{},
		Neighbors:  []walkanalysis.Neighbor{},
	}

	err := outputText(analysis)
	if err != nil {
		t.Errorf("outputText() error = %v", err)
	}
}

// TestOutputNeighborsJSON tests neighbor output in JSON format.
func TestOutputNeighborsJSON(t *testing.T) {
	neighbors := []walkanalysis.Neighbor{
		{LocalInterface: "Gi0/1", RemoteDevice: "sw1", RemoteInterface: "Gi0/2", Protocol: "lldp"},
	}

	err := outputNeighbors(neighbors, "json")
	if err != nil {
		t.Errorf("outputNeighbors(json) error = %v", err)
	}
}

// TestOutputNeighborsYAML tests neighbor output in YAML format.
func TestOutputNeighborsYAML(t *testing.T) {
	neighbors := []walkanalysis.Neighbor{
		{LocalInterface: "Gi0/1", RemoteDevice: "rtr1", RemoteInterface: "Gi1/0", Protocol: "cdp"},
	}

	err := outputNeighbors(neighbors, "yaml")
	if err != nil {
		t.Errorf("outputNeighbors(yaml) error = %v", err)
	}
}

// TestOutputNeighborsText tests neighbor output in text format.
func TestOutputNeighborsText(t *testing.T) {
	neighbors := []walkanalysis.Neighbor{
		{LocalInterface: "Gi0/1", RemoteDevice: "sw1", RemoteInterface: "Gi0/2", Protocol: "lldp"},
		{LocalInterface: "Gi0/2", RemoteDevice: "rtr1", RemoteInterface: "Gi1/0", Protocol: "cdp"},
	}

	err := outputNeighbors(neighbors, "text")
	if err != nil {
		t.Errorf("outputNeighbors(text) error = %v", err)
	}
}

// TestOutputNeighborsEmpty tests neighbor output with empty list.
func TestOutputNeighborsEmpty(t *testing.T) {
	err := outputNeighbors([]walkanalysis.Neighbor{}, "text")
	if err != nil {
		t.Errorf("outputNeighbors(empty) error = %v", err)
	}
}

// TestOutputNeighborsInvalidFormat tests neighbor output with invalid format.
func TestOutputNeighborsInvalidFormat(t *testing.T) {
	err := outputNeighbors([]walkanalysis.Neighbor{}, "invalid")
	if err == nil {
		t.Error("Expected error for invalid format")
	}
}

// TestPcapSummaryStruct tests pcapSummary struct.
func TestPcapSummaryStruct(t *testing.T) {
	summary := pcapSummary{
		File:        "test.pcap",
		Packets:     100,
		ProtocolMap: map[string]int{"ARP": 10, "ICMP": 20},
	}

	if summary.File != "test.pcap" {
		t.Errorf("File = %q, want %q", summary.File, "test.pcap")
	}
	if summary.Packets != 100 {
		t.Errorf("Packets = %d, want %d", summary.Packets, 100)
	}
	if summary.ProtocolMap["ARP"] != 10 {
		t.Errorf("ARP count = %d, want %d", summary.ProtocolMap["ARP"], 10)
	}
}

// TestMonitorStatsStruct tests monitorStats struct.
func TestMonitorStatsStruct(t *testing.T) {
	stats := monitorStats{
		PacketsRX: 1000,
		RateRX:    100.5,
	}

	if stats.PacketsRX != 1000 {
		t.Errorf("PacketsRX = %d, want %d", stats.PacketsRX, 1000)
	}
	if stats.RateRX != 100.5 {
		t.Errorf("RateRX = %f, want %f", stats.RateRX, 100.5)
	}
}

// TestNeighborEntryStruct tests NeighborEntry struct.
func TestNeighborEntryStruct(t *testing.T) {
	entry := NeighborEntry{
		Device:   "switch-1",
		Protocol: "lldp",
	}

	if entry.Device != "switch-1" {
		t.Errorf("Device = %q, want %q", entry.Device, "switch-1")
	}
	if entry.Protocol != "lldp" {
		t.Errorf("Protocol = %q, want %q", entry.Protocol, "lldp")
	}
}
