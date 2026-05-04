package main

import (
	"testing"
)

func TestPrintGeneratorHeader(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printGeneratorHeader panicked: %v", r)
		}
	}()
	printGeneratorHeader()
}

func TestWriteConfiguration(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := tmpDir + "/test-config.yaml"

	cfg := &generatedConfig{
		networkName: "test-network",
		subnet:      "192.168.1.0/24",
		devices: []generatedDevice{
			{
				name:      "router-1",
				devType:   "router",
				ip:        "192.168.1.1",
				mac:       "02:00:00:00:00:01",
				protocols: map[string]protocolConfig{},
			},
		},
	}

	// Should not panic
	writeConfiguration(outputFile, cfg)
}

func TestPrintSummary(t *testing.T) {
	cfg := &generatedConfig{
		networkName: "test-network",
		subnet:      "192.168.1.0/24",
		includePath: "/path/to/walks",
		devices: []generatedDevice{
			{
				name:    "router-1",
				devType: "router",
				ip:      "192.168.1.1",
				mac:     "02:00:00:00:00:01",
				protocols: map[string]protocolConfig{
					"lldp": {enabled: true, params: map[string]string{}},
					"cdp":  {enabled: true, params: map[string]string{}},
				},
			},
			{
				name:      "switch-1",
				devType:   "switch",
				ip:        "192.168.1.2",
				mac:       "02:00:00:00:00:02",
				protocols: map[string]protocolConfig{},
			},
		},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printSummary panicked: %v", r)
		}
	}()
	printSummary("test-config.yaml", cfg)
}

func TestPrintSummaryNoIncludePath(t *testing.T) {
	cfg := &generatedConfig{
		networkName: "minimal",
		subnet:      "10.0.0.0/24",
		devices:     []generatedDevice{},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printSummary panicked: %v", r)
		}
	}()
	printSummary("minimal.yaml", cfg)
}
