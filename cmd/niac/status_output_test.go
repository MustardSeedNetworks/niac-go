package main

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/ipc"
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

	status := &ipc.StatusData{
		Running:      true,
		Interface:    "en0",
		ConfigPath:   "/path/config.yaml",
		DeviceCount:  5,
		Uptime:       3661,
		PacketsRX:    10000,
		PacketsTX:    20000,
		ErrorsActive: 0,
	}
	outputSuccess(status, false)
}

func TestOutputSuccessJSON(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("outputSuccess panicked: %v", r)
		}
	}()

	status := &ipc.StatusData{
		Running:      true,
		Interface:    "en0",
		ConfigPath:   "/path/config.yaml",
		DeviceCount:  5,
		Uptime:       3661,
		PacketsRX:    10000,
		PacketsTX:    20000,
		ErrorsActive: 0,
	}
	outputSuccess(status, true)
}

func TestPrintHumanStatus(t *testing.T) {
	tests := []struct {
		name   string
		status *ipc.StatusData
	}{
		{
			name: "running",
			status: &ipc.StatusData{
				Running:      true,
				Interface:    "en0",
				ConfigPath:   "config.yaml",
				DeviceCount:  3,
				Uptime:       100,
				PacketsRX:    5000,
				PacketsTX:    3000,
				ErrorsActive: 0,
			},
		},
		{
			name: "stopped",
			status: &ipc.StatusData{
				Running: false,
			},
		},
		{
			name: "with errors",
			status: &ipc.StatusData{
				Running:      true,
				Interface:    "eth0",
				DeviceCount:  10,
				Uptime:       86400,
				PacketsRX:    1000000,
				PacketsTX:    500000,
				ErrorsActive: 3,
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
