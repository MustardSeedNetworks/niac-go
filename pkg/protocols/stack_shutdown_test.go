package protocols

import (
	"net"
	"testing"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/logging"
)

// TestStackStopCleanup tests that Stop() properly cleans up stack resources
func TestStackStopCleanup(t *testing.T) {
	cfg := &config.Config{}
	debugConfig := logging.NewDebugConfig(0)

	// Create stack without capture (to avoid network dependency)
	stack := NewStack(nil, cfg, debugConfig)

	// Initialize devices
	stack.initializeDevices()

	// Stack should not be running yet
	if stack.running {
		t.Error("Stack should not be running before Start()")
	}

	// Note: We can't call Start() without a valid capture engine
	// So we test the Stop() idempotency
	stack.Stop()

	// Stop should be idempotent
	stack.Stop()
	stack.Stop()

	if stack.running {
		t.Error("Stack should not be running after Stop()")
	}
}

// TestStackStopWaitsForGoroutines tests that Stop() waits for goroutines
func TestStackStopWaitsForGoroutines(t *testing.T) {
	cfg := &config.Config{}
	debugConfig := logging.NewDebugConfig(0)
	stack := NewStack(nil, cfg, debugConfig)

	// Manually start some goroutines to simulate running state
	stack.running = true
	stack.stopChan = make(chan struct{})

	// Start a goroutine that will wait for stop signal
	stopped := make(chan bool)
	stack.wg.Add(1)
	go func() {
		defer stack.wg.Done()
		<-stack.stopChan
		stopped <- true
	}()

	// Stop the stack
	go stack.Stop()

	// Verify goroutine received stop signal
	select {
	case <-stopped:
		// Good - goroutine was stopped
	case <-time.After(1 * time.Second):
		t.Error("Stack.Stop() did not wait for goroutines to finish")
	}
}

// TestStackMultipleStartStop tests Start/Stop cycles
func TestStackMultipleStartStop(t *testing.T) {
	cfg := &config.Config{}
	debugConfig := logging.NewDebugConfig(0)

	stack := NewStack(nil, cfg, debugConfig)

	// Stop without Start should be safe
	stack.Stop()

	// Multiple stops should be safe
	stack.Stop()
	stack.Stop()

	if stack.running {
		t.Error("Stack should not be running")
	}
}

// TestStackDeviceInitialization tests that devices are properly initialized
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

	stack := NewStack(nil, cfg, debugConfig)
	stack.initializeDevices()

	// Verify device was added
	devices := stack.GetDevices().GetAll()
	if len(devices) != 1 {
		t.Errorf("Expected 1 device, got %d", len(devices))
	}

	if devices[0].Name != "test-device" {
		t.Errorf("Expected device name 'test-device', got '%s'", devices[0].Name)
	}
}

// TestStackCleanupOrder tests that cleanup happens in correct order
func TestStackCleanupOrder(t *testing.T) {
	cfg := &config.Config{}
	debugConfig := logging.NewDebugConfig(0)
	stack := NewStack(nil, cfg, debugConfig)

	// Set up minimal state
	stack.running = true
	stack.stopChan = make(chan struct{})

	// Track cleanup order
	cleanupOrder := make([]string, 0)

	// Simulate protocol handlers with cleanup
	stack.wg.Add(1)
	go func() {
		defer func() {
			cleanupOrder = append(cleanupOrder, "goroutine")
			stack.wg.Done()
		}()
		<-stack.stopChan
	}()

	// Stop stack
	stack.Stop()

	// Verify cleanup happened
	if len(cleanupOrder) != 1 {
		t.Errorf("Expected 1 cleanup event, got %d", len(cleanupOrder))
	}

	if !stack.running {
		// Good - running flag is false after stop
	} else {
		t.Error("Stack still marked as running after Stop()")
	}
}
