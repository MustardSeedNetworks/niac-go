package main

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/templates"
)

func TestDescribeScenario(t *testing.T) {
	tmpl := &templates.Template{
		Name:        "test-scenario",
		Description: "A test scenario",
		UseCase:     "Testing purposes",
		Content:     "devices: []",
	}

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("describeScenario panicked: %v", r)
		}
	}()
	describeScenario(tmpl)
}

func TestDescribeDevices(t *testing.T) {
	devices := []config.Device{
		{
			Name:       "router-1",
			Type:       "router",
			ICMPConfig: &config.ICMPConfig{Enabled: true},
			LLDPConfig: &config.LLDPConfig{Enabled: true},
		},
		{
			Name:       "switch-1",
			Type:       "switch",
			SNMPConfig: config.SNMPConfig{Community: "public"},
		},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("describeDevices panicked: %v", r)
		}
	}()
	describeDevices("test-scenario", devices)
}

func TestDescribeDeviceInfo(t *testing.T) {
	tests := []struct {
		name   string
		device config.Device
	}{
		{
			name: "device with single IP",
			device: config.Device{
				Name: "router-1",
				Type: "router",
			},
		},
		{
			name: "device with protocols",
			device: config.Device{
				Name:       "switch-1",
				Type:       "switch",
				ICMPConfig: &config.ICMPConfig{Enabled: true},
				LLDPConfig: &config.LLDPConfig{Enabled: true},
				CDPConfig:  &config.CDPConfig{Enabled: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("describeDeviceInfo panicked: %v", r)
				}
			}()
			describeDeviceInfo(tt.device)
		})
	}
}

func TestLoadAndValidateScenario(t *testing.T) {
	t.Run("valid scenario", func(t *testing.T) {
		tmpl, err := templates.Get("basic-network")
		if err != nil {
			t.Skip("basic-network scenario not available")
		}

		cfg, cleanup, loadErr := loadAndValidateScenario(tmpl)
		defer cleanup()

		if loadErr != nil {
			t.Errorf("loadAndValidateScenario() error = %v", loadErr)
		}
		if cfg == nil {
			t.Error("Expected non-nil config")
		}
	})

	t.Run("invalid scenario content", func(t *testing.T) {
		tmpl := &templates.Template{
			Name:    "invalid",
			Content: "not: valid: yaml: [[[[",
		}

		_, cleanup, loadErr := loadAndValidateScenario(tmpl)
		defer cleanup()

		if loadErr == nil {
			t.Error("Expected error for invalid scenario content")
		}
	})
}

func TestRunScenarioList(t *testing.T) {
	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("runScenarioList panicked: %v", r)
		}
	}()
	runScenarioList()
}
