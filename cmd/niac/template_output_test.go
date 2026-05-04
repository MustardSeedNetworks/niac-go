package main

import (
	"testing"

	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/templates"
)

func TestDescribeTemplate(t *testing.T) {
	tmpl := &templates.Template{
		Name:        "test-template",
		Description: "A test template",
		UseCase:     "Testing purposes",
		Content:     "devices: []",
	}

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("describeTemplate panicked: %v", r)
		}
	}()
	describeTemplate(tmpl)
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
	describeDevices("test-template", devices)
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

func TestLoadAndValidateTemplate(t *testing.T) {
	t.Run("valid template", func(t *testing.T) {
		tmpl, err := templates.Get("basic-network")
		if err != nil {
			t.Skip("basic-network template not available")
		}

		cfg, cleanup, loadErr := loadAndValidateTemplate(tmpl)
		defer cleanup()

		if loadErr != nil {
			t.Errorf("loadAndValidateTemplate() error = %v", loadErr)
		}
		if cfg == nil {
			t.Error("Expected non-nil config")
		}
	})

	t.Run("invalid template content", func(t *testing.T) {
		tmpl := &templates.Template{
			Name:    "invalid",
			Content: "not: valid: yaml: [[[[",
		}

		_, cleanup, loadErr := loadAndValidateTemplate(tmpl)
		defer cleanup()

		if loadErr == nil {
			t.Error("Expected error for invalid template content")
		}
	})
}

func TestRunTemplateList(t *testing.T) {
	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("runTemplateList panicked: %v", r)
		}
	}()
	runTemplateList()
}
