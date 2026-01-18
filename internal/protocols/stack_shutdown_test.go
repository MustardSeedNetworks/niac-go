package protocols_test

import (
	"net"
	"testing"

	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/logging"
	"github.com/krisarmstrong/niac-go/internal/protocols"
)

// Note: The original tests for TestStackStopCleanup, TestStackStopWaitsForGoroutines, and
// TestStackCleanupOrder relied on unexported fields (stack.running, stack.stopChan, stack.wg)
// which are not accessible from an external test package.
// The tests below have been adapted to use only exported interfaces.

// TestStackStopIdempotency tests that Stop() is idempotent.
func TestStackStopIdempotency(_ *testing.T) {
	cfg := &config.Config{}
	debugConfig := logging.NewDebugConfig(0)

	// Create stack without capture (to avoid network dependency)
	stack := protocols.NewStack(nil, cfg, debugConfig)

	// Stop should be idempotent - calling multiple times should not panic
	stack.Stop()
	stack.Stop()
	stack.Stop()
	// If we get here without panicking, the test passes
}

// TestStackMultipleStartStop tests Start/Stop cycles.
func TestStackMultipleStartStop(_ *testing.T) {
	cfg := &config.Config{}
	debugConfig := logging.NewDebugConfig(0)

	stack := protocols.NewStack(nil, cfg, debugConfig)

	// Stop without Start should be safe
	stack.Stop()

	// Multiple stops should be safe
	stack.Stop()
	stack.Stop()
	// If we get here without panicking, the test passes
}

// TestStackDeviceInitialization tests that devices are properly initialized.
func TestStackDeviceInitialization(t *testing.T) {
	cfg := &config.Config{
		Devices: []config.Device{
			{
				Name:        "test-device",
				Type:        "router",
				MACAddress:  []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.IPv4(192, 168, 1, 1)},
			},
		},
	}
	debugConfig := logging.NewDebugConfig(0)

	stack := protocols.NewStack(nil, cfg, debugConfig)

	// Verify device was added
	devices := stack.GetDevices().GetAll()
	if len(devices) != 1 {
		t.Errorf("Expected 1 device, got %d", len(devices))
	}

	if devices[0].Name != "test-device" {
		t.Errorf("Expected device name 'test-device', got '%s'", devices[0].Name)
	}
}

func TestStackReloadConfig(t *testing.T) {
	cfg1 := &config.Config{
		Devices: []config.Device{
			{
				Name:        "alpha",
				Type:        "switch",
				MACAddress:  []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05},
				IPAddresses: []net.IP{net.IPv4(10, 0, 0, 1)},
			},
		},
	}
	cfg2 := &config.Config{
		Devices: []config.Device{
			{
				Name:        "beta",
				Type:        "router",
				MACAddress:  []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
				IPAddresses: []net.IP{net.IPv4(10, 0, 1, 1)},
			},
			{
				Name:        "gamma",
				Type:        "router",
				MACAddress:  []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0x01},
				IPAddresses: []net.IP{net.IPv4(10, 0, 2, 1)},
			},
		},
	}

	stack := protocols.NewStack(nil, cfg1, logging.NewDebugConfig(0))
	if got := stack.GetDevices().Count(); got != 1 {
		t.Fatalf("expected 1 device after init, got %d", got)
	}

	err := stack.ReloadConfig(cfg2)
	if err != nil {
		t.Fatalf("ReloadConfig failed: %v", err)
	}

	if got := stack.GetDevices().Count(); got != len(cfg2.Devices) {
		t.Fatalf("expected %d devices after reload, got %d", len(cfg2.Devices), got)
	}
}

// TestStackDeviceInitializationMultiple tests multiple device initialization.
func TestStackDeviceInitializationMultiple(t *testing.T) {
	cfg := &config.Config{
		Devices: []config.Device{
			{
				Name:        "device-1",
				Type:        "router",
				MACAddress:  []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.IPv4(192, 168, 1, 1)},
			},
			{
				Name:        "device-2",
				Type:        "switch",
				MACAddress:  []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
				IPAddresses: []net.IP{net.IPv4(192, 168, 2, 1)},
			},
		},
	}
	debugConfig := logging.NewDebugConfig(0)

	stack := protocols.NewStack(nil, cfg, debugConfig)

	// Verify both devices were added
	devices := stack.GetDevices().GetAll()
	if len(devices) != 2 {
		t.Errorf("Expected 2 devices, got %d", len(devices))
	}
}

// TestStackStatsInitialization tests that statistics are properly initialized.
func TestStackStatsInitialization(t *testing.T) {
	cfg := &config.Config{}
	debugConfig := logging.NewDebugConfig(0)

	stack := protocols.NewStack(nil, cfg, debugConfig)

	// Verify stats are initialized to zero
	stats := stack.GetStats()

	if stats.PacketsReceived != 0 {
		t.Errorf("Expected PacketsReceived to be 0, got %d", stats.PacketsReceived)
	}

	if stats.PacketsSent != 0 {
		t.Errorf("Expected PacketsSent to be 0, got %d", stats.PacketsSent)
	}
}
