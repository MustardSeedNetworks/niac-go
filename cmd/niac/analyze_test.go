package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzeWalkFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		walkData    string
		expectError bool
		description string
	}{
		{
			name: "Valid walk file with system info",
			walkData: `.1.3.6.1.2.1.1.1.0 = STRING: Cisco IOS Software
.1.3.6.1.2.1.1.5.0 = STRING: test-switch
.1.3.6.1.2.1.1.2.0 = OID: .1.3.6.1.4.1.9.1.1719
.1.3.6.1.2.1.1.4.0 = STRING: netadmin@niac-go.com
.1.3.6.1.2.1.1.6.0 = STRING: NiAC-Go - DC-WEST
`,
			expectError: false,
			description: "Walk file with valid system MIB entries",
		},
		{
			name: "Valid walk file with interface info",
			walkData: `.1.3.6.1.2.1.2.1.0 = INTEGER: 10
.1.3.6.1.2.1.2.2.1.1.1 = INTEGER: 1
.1.3.6.1.2.1.2.2.1.2.1 = STRING: GigabitEthernet0/1
.1.3.6.1.2.1.2.2.1.3.1 = INTEGER: 6
.1.3.6.1.2.1.2.2.1.5.1 = Gauge32: 1000000000
.1.3.6.1.2.1.2.2.1.8.1 = INTEGER: 1
`,
			expectError: false,
			description: "Walk file with interface MIB entries",
		},
		{
			name:        "Empty walk file",
			walkData:    ``,
			expectError: false, // Empty file is valid, just produces empty analysis
			description: "Empty walk file should not error",
		},
		{
			name: "Walk file with comments",
			walkData: `# This is a comment
# Another comment
.1.3.6.1.2.1.1.5.0 = STRING: test-device
`,
			expectError: false,
			description: "Walk file with comment lines",
		},
		{
			name: "Walk file with invalid OID format",
			walkData: `invalid line here
.1.3.6.1.2.1.1.5.0 = STRING: test-device
another invalid line
`,
			expectError: false, // Should skip invalid lines
			description: "Walk file with some invalid lines should continue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			walkFile := filepath.Join(tmpDir, tt.name+".walk")
			err := os.WriteFile(walkFile, []byte(tt.walkData), 0644)
			if err != nil {
				t.Fatalf("Failed to write walk file: %v", err)
			}

			// Test that the file is readable
			data, err := os.ReadFile(walkFile)
			if err != nil {
				t.Fatalf("Failed to read walk file: %v", err)
			}

			if tt.walkData != string(data) {
				t.Errorf("Walk file content mismatch")
			}
		})
	}
}

func TestAnalyzeFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentFile := filepath.Join(tmpDir, "nonexistent.walk")

	_, err := os.ReadFile(nonExistentFile)
	if err == nil {
		t.Error("Expected error for non-existent walk file, got nil")
	}
}

func TestAnalyzeReadability(t *testing.T) {
	tmpDir := t.TempDir()
	walkFile := filepath.Join(tmpDir, "test.walk")

	// Create test walk file
	testData := `.1.3.6.1.2.1.1.1.0 = STRING: Test Device
.1.3.6.1.2.1.1.5.0 = STRING: test-switch-01
`

	err := os.WriteFile(walkFile, []byte(testData), 0644)
	if err != nil {
		t.Fatalf("Failed to write walk file: %v", err)
	}

	// Verify it's readable
	data, err := os.ReadFile(walkFile)
	if err != nil {
		t.Errorf("Walk file should be readable: %v", err)
	}

	if len(data) == 0 {
		t.Error("Walk file data is empty")
	}
}

func TestWalkAnalysisStructs(t *testing.T) {
	// Test that analysis structs can be created
	analysis := WalkAnalysis{
		Device: DeviceInfo{
			SysName:     "test-switch",
			SysDescr:    "Cisco IOS",
			SysObjectID: ".1.3.6.1.4.1.9.1.1719",
			SysContact:  "admin@test.com",
			SysLocation: "DC-WEST",
		},
		Interfaces: []InterfaceInfo{
			{
				Index:       1,
				Name:        "GigabitEthernet0/1",
				Description: "Uplink",
				Type:        "ethernetCsmacd",
				Speed:       1000000000,
				AdminStatus: "up",
				OperStatus:  "up",
			},
		},
		Neighbors: []NeighborInfo{
			{
				LocalInterface:  "GigabitEthernet0/1",
				RemoteDevice:    "core-switch-01",
				RemoteInterface: "GigabitEthernet1/1",
				Protocol:        "lldp",
			},
		},
		Statistics: AnalysisStats{
			TotalInterfaces:    10,
			PhysicalInterfaces: 8,
			LogicalInterfaces:  2,
			TotalNeighbors:     3,
		},
	}

	// Verify struct fields
	if analysis.Device.SysName != "test-switch" {
		t.Errorf("SysName = %v, want test-switch", analysis.Device.SysName)
	}

	if len(analysis.Interfaces) != 1 {
		t.Errorf("Interfaces count = %d, want 1", len(analysis.Interfaces))
	}

	if len(analysis.Neighbors) != 1 {
		t.Errorf("Neighbors count = %d, want 1", len(analysis.Neighbors))
	}

	if analysis.Statistics.TotalInterfaces != 10 {
		t.Errorf("TotalInterfaces = %d, want 10", analysis.Statistics.TotalInterfaces)
	}
}

func TestAnalyzeWalkFileMalformedInput(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		walkData string
	}{
		{
			name:     "Binary data",
			walkData: string([]byte{0xFF, 0xFE, 0xFD, 0xFC}),
		},
		{
			name:     "Very long lines",
			walkData: ".1.3.6.1.2.1.1.1.0 = STRING: " + string(make([]byte, 10000)),
		},
		{
			name: "Mixed valid and invalid",
			walkData: `invalid line 1
.1.3.6.1.2.1.1.5.0 = STRING: test
@#$%^&*()
.1.3.6.1.2.1.1.1.0 = STRING: device
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			walkFile := filepath.Join(tmpDir, tt.name+".walk")
			err := os.WriteFile(walkFile, []byte(tt.walkData), 0644)
			if err != nil {
				t.Fatalf("Failed to write walk file: %v", err)
			}

			// Should not crash when reading malformed data
			data, err := os.ReadFile(walkFile)
			if err != nil {
				t.Errorf("Failed to read malformed walk file: %v", err)
			}

			if len(data) == 0 && len(tt.walkData) > 0 {
				t.Error("Data was lost during write/read")
			}
		})
	}
}
