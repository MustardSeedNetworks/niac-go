package main

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/cliclient"
)

func TestOutputErrorNotRunning(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("outputError panicked: %v", r)
		}
	}()

	err := errors.New("socket not found: /tmp/niac.sock")
	outputError(err, exitCodeNotRunning, false)
}

func TestOutputErrorNotRunningJSON(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("outputError panicked: %v", r)
		}
	}()

	err := errors.New("socket not found: /tmp/niac.sock")
	outputError(err, exitCodeNotRunning, true)
}

func TestOutputErrorGeneric(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("outputError panicked: %v", r)
		}
	}()

	err := errors.New("unknown error")
	outputError(err, exitCodeError, false)
}

func TestOutputErrorGenericJSON(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("outputError panicked: %v", r)
		}
	}()

	err := errors.New("unknown error")
	outputError(err, exitCodeError, true)
}

func TestOutputSuccessText(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("outputSuccess panicked: %v", r)
		}
	}()

	status := &cliclient.Runtime{
		Running:     true,
		Interface:   "en0",
		ConfigName:  "/path/config.yaml",
		DeviceCount: 5,
		Uptime:      3661,
		PacketsRX:   10000,
		PacketsTX:   20000,
	}
	outputSuccess(status, false)
}

func TestOutputSuccessJSON(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("outputSuccess panicked: %v", r)
		}
	}()

	status := &cliclient.Runtime{
		Running:     true,
		Interface:   "en0",
		ConfigName:  "/path/config.yaml",
		DeviceCount: 5,
		Uptime:      3661,
		PacketsRX:   10000,
		PacketsTX:   20000,
	}
	outputSuccess(status, true)
}

func TestPrintHumanStatus(t *testing.T) {
	tests := []struct {
		name   string
		status *cliclient.Runtime
	}{
		{
			name: "running",
			status: &cliclient.Runtime{
				Running:     true,
				Interface:   "en0",
				ConfigName:  "config.yaml",
				DeviceCount: 3,
				Uptime:      100,
				PacketsRX:   5000,
				PacketsTX:   3000,
			},
		},
		{
			name: "stopped",
			status: &cliclient.Runtime{
				Running: false,
			},
		},
		{
			name: "with errors",
			status: &cliclient.Runtime{
				Running:     true,
				Interface:   "eth0",
				DeviceCount: 10,
				Uptime:      86400,
				PacketsRX:   1000000,
				PacketsTX:   500000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("printHumanStatus panicked: %v", r)
				}
			}()
			printHumanStatus(tt.status)
		})
	}
}

func TestOutputStatusJSON(t *testing.T) {
	data := map[string]any{
		"running":   true,
		"interface": "en0",
		"devices":   5,
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("outputStatusJSON panicked: %v", r)
		}
	}()
	outputStatusJSON(data)
}
