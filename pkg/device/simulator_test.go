package device

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/errors"
	"github.com/krisarmstrong/niac-go/pkg/logging"
	"github.com/krisarmstrong/niac-go/pkg/protocols"
)

// createTestConfig creates a test configuration with one or more devices
func createTestConfig(deviceCount int) *config.Config {
	cfg := &config.Config{
		Devices: make([]config.Device, deviceCount),
	}

	for i := 0; i < deviceCount; i++ {
		cfg.Devices[i] = config.Device{
			Name:        fmt.Sprintf("test-device-%d", i),
			Type:        "router",
			MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, byte(0x55 + i)},
			IPAddresses: []net.IP{net.ParseIP(fmt.Sprintf("192.168.1.%d", i+1))},
			SNMPConfig: config.SNMPConfig{
				Community: "public",
			},
		}
	}

	return cfg
}

// TestNewSimulator tests creating a new simulator
func TestNewSimulator(t *testing.T) {
	cfg := createTestConfig(3)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := errors.NewStateManager()

	sim := NewSimulator(cfg, stack, errorMgr, 0)

	if sim == nil {
		t.Fatal("Expected simulator, got nil")
	}
	if sim.config != cfg {
		t.Error("Config not set correctly")
	}
	if sim.stack != stack {
		t.Error("Stack not set correctly")
	}
	if sim.errorManager != errorMgr {
		t.Error("Error manager not set correctly")
	}
	if len(sim.devices) != 3 {
		t.Errorf("Expected 3 devices, got %d", len(sim.devices))
	}
	if sim.running {
		t.Error("Simulator should not be running initially")
	}
}

// TestNewSimulator_EmptyConfig tests simulator with no devices
func TestNewSimulator_EmptyConfig(t *testing.T) {
	cfg := &config.Config{
		Devices: []config.Device{},
	}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := errors.NewStateManager()

	sim := NewSimulator(cfg, stack, errorMgr, 0)

	if sim == nil {
		t.Fatal("Expected simulator, got nil")
	}
	if len(sim.devices) != 0 {
		t.Errorf("Expected 0 devices, got %d", len(sim.devices))
	}
}

// TestSimulator_GetDevice tests retrieving a device by name
func TestSimulator_GetDevice(t *testing.T) {
	cfg := createTestConfig(2)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := errors.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	// Get existing device
	device := sim.GetDevice("test-device-0")
	if device == nil {
		t.Fatal("Expected device, got nil")
	}
	if device.Config.Name != "test-device-0" {
		t.Errorf("Expected device name 'test-device-0', got '%s'", device.Config.Name)
	}

	// Get non-existent device
	device = sim.GetDevice("non-existent")
	if device != nil {
		t.Error("Expected nil for non-existent device")
	}
}

// TestSimulator_GetAllDevices tests retrieving all devices
func TestSimulator_GetAllDevices(t *testing.T) {
	cfg := createTestConfig(3)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := errors.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	devices := sim.GetAllDevices()

	if len(devices) != 3 {
		t.Errorf("Expected 3 devices, got %d", len(devices))
	}

	// Check that all expected devices are present
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("test-device-%d", i)
		if _, exists := devices[name]; !exists {
			t.Errorf("Expected device '%s' not found", name)
		}
	}
}

// TestSimulator_Lifecycle tests start and stop
func TestSimulator_Lifecycle(t *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := errors.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	// Initial state
	if sim.running {
		t.Error("Simulator should not be running initially")
	}

	// Start simulator
	err := sim.Start()
	if err != nil {
		t.Fatalf("Failed to start simulator: %v", err)
	}
	if !sim.running {
		t.Error("Simulator should be running after Start()")
	}

	// Try to start again (should fail)
	err = sim.Start()
	if err == nil {
		t.Error("Expected error when starting already running simulator")
	}

	// Stop simulator
	sim.Stop()
	time.Sleep(50 * time.Millisecond) // Give it time to stop

	if sim.running {
		t.Error("Simulator should not be running after Stop()")
	}

	// Stop again (should be safe)
	sim.Stop()
}

// TestSimulator_SetDeviceState tests setting device state
func TestSimulator_SetDeviceState(t *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := errors.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	deviceName := "test-device-0"

	// Initial state should be "up"
	device := sim.GetDevice(deviceName)
	if device.State != StateUp {
		t.Errorf("Expected initial state StateUp, got %s", device.State)
	}

	// Set to down
	err := sim.SetDeviceState(deviceName, StateDown)
	if err != nil {
		t.Errorf("Failed to set device state: %v", err)
	}

	device = sim.GetDevice(deviceName)
	if device.State != StateDown {
		t.Errorf("Expected state StateDown, got %s", device.State)
	}

	// Set to maintenance
	err = sim.SetDeviceState(deviceName, StateMaintenance)
	if err != nil {
		t.Errorf("Failed to set device state: %v", err)
	}

	device = sim.GetDevice(deviceName)
	if device.State != StateMaintenance {
		t.Errorf("Expected state StateMaintenance, got %s", device.State)
	}

	// Try to set state for non-existent device
	err = sim.SetDeviceState("non-existent", StateUp)
	if err == nil {
		t.Error("Expected error for non-existent device")
	}
}

// TestSimulator_DeviceStates tests all device state constants
func TestSimulator_DeviceStates(t *testing.T) {
	states := []DeviceState{
		StateUp,
		StateDown,
		StateStarting,
		StateStopping,
		StateMaintenance,
	}

	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := errors.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	deviceName := "test-device-0"

	for _, state := range states {
		err := sim.SetDeviceState(deviceName, state)
		if err != nil {
			t.Errorf("Failed to set device state to %s: %v", state, err)
		}

		device := sim.GetDevice(deviceName)
		if device.State != state {
			t.Errorf("Expected state %s, got %s", state, device.State)
		}
	}
}

// TestSimulator_IncrementCounter tests counter increments
func TestSimulator_IncrementCounter(t *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := errors.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	deviceName := "test-device-0"
	device := sim.GetDevice(deviceName)

	// Test all counter types
	counters := map[string]*uint64{
		"arp_requests":     &device.Counters.ARPRequestsReceived,
		"arp_replies":      &device.Counters.ARPRepliesSent,
		"icmp_requests":    &device.Counters.ICMPRequestsReceived,
		"icmp_replies":     &device.Counters.ICMPRepliesSent,
		"snmp_queries":     &device.Counters.SNMPQueriesReceived,
		"http_requests":    &device.Counters.HTTPRequestsReceived,
		"ftp_connections":  &device.Counters.FTPConnectionsReceived,
		"packets_sent":     &device.Counters.PacketsSent,
		"packets_received": &device.Counters.PacketsReceived,
		"errors":           &device.Counters.Errors,
	}

	for counterName, counter := range counters {
		t.Run(counterName, func(t *testing.T) {
			initial := *counter
			sim.IncrementCounter(deviceName, counterName)
			if *counter != initial+1 {
				t.Errorf("Counter %s: expected %d, got %d", counterName, initial+1, *counter)
			}
		})
	}
}

// TestSimulator_IncrementCounter_NonExistentDevice tests incrementing counter for non-existent device
func TestSimulator_IncrementCounter_NonExistentDevice(t *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := errors.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	// Should not panic
	sim.IncrementCounter("non-existent", "arp_requests")
}

// TestSimulator_ConcurrentAccess tests thread-safety with concurrent operations
func TestSimulator_ConcurrentAccess(t *testing.T) {
	cfg := createTestConfig(5)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := errors.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	var wg sync.WaitGroup

	// Concurrent device retrieval
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				sim.GetAllDevices()
				sim.GetDevice("test-device-0")
			}
		}()
	}

	// Concurrent counter increments
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(deviceIdx int) {
			defer wg.Done()
			deviceName := fmt.Sprintf("test-device-%d", deviceIdx%5)
			for j := 0; j < 100; j++ {
				sim.IncrementCounter(deviceName, "packets_sent")
				sim.IncrementCounter(deviceName, "packets_received")
			}
		}(i)
	}

	// Concurrent state changes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(deviceIdx int) {
			defer wg.Done()
			deviceName := fmt.Sprintf("test-device-%d", deviceIdx%5)
			for j := 0; j < 50; j++ {
				sim.SetDeviceState(deviceName, StateUp)
				sim.SetDeviceState(deviceName, StateDown)
			}
		}(i)
	}

	wg.Wait()

	// Verify devices still exist and have expected counter values
	devices := sim.GetAllDevices()
	if len(devices) != 5 {
		t.Errorf("Expected 5 devices after concurrent operations, got %d", len(devices))
	}

	// Check that counters increased (should be 200 per device: 2 counters * 100 increments)
	for i := 0; i < 5; i++ {
		deviceName := fmt.Sprintf("test-device-%d", i)
		device := sim.GetDevice(deviceName)
		if device == nil {
			t.Errorf("Device %s not found", deviceName)
			continue
		}

		// Should have received 400 increments (2 goroutines * (100 packets_sent + 100 packets_received))
		expectedCount := uint64(400)
		totalCount := device.Counters.PacketsSent + device.Counters.PacketsReceived
		if totalCount != expectedCount {
			t.Errorf("Device %s: expected %d counter increments, got %d",
				deviceName, expectedCount, totalCount)
		}
	}
}

// TestSimulator_DeviceTypes tests different device types
func TestSimulator_DeviceTypes(t *testing.T) {
	deviceTypes := []string{"router", "switch", "ap", "access-point", "server", "generic"}

	for _, deviceType := range deviceTypes {
		t.Run(deviceType, func(t *testing.T) {
			cfg := &config.Config{
				Devices: []config.Device{
					{
						Name:        "test-device",
						Type:        deviceType,
						MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
						IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
						SNMPConfig: config.SNMPConfig{
							Community: "public",
						},
					},
				},
			}

			stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
			errorMgr := errors.NewStateManager()
			sim := NewSimulator(cfg, stack, errorMgr, 0)

			device := sim.GetDevice("test-device")
			if device == nil {
				t.Fatal("Device not found")
			}
			if device.Config.Type != deviceType {
				t.Errorf("Expected device type '%s', got '%s'", deviceType, device.Config.Type)
			}
		})
	}
}

// TestSimulator_WithTrapSender tests simulator with trap sender enabled
func TestSimulator_WithTrapSender(t *testing.T) {
	cfg := &config.Config{
		Devices: []config.Device{
			{
				Name:        "router-with-traps",
				Type:        "router",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
				SNMPConfig: config.SNMPConfig{
					Community: "public",
					Traps: &config.TrapConfig{
						Enabled:   true,
						Receivers: []string{"192.168.1.100:162"},
						ColdStart: &config.TrapTriggerConfig{
							Enabled:   true,
							OnStartup: false, // Don't send on startup to avoid connection attempts
						},
					},
				},
			},
		},
	}

	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := errors.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	device := sim.GetDevice("router-with-traps")
	if device == nil {
		t.Fatal("Device not found")
	}

	// Trap sender should be initialized
	if device.TrapSender == nil {
		t.Error("Expected trap sender to be initialized")
	}
}

// TestSimulator_LastActivity tests that LastActivity is tracked
func TestSimulator_LastActivity(t *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := errors.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	device := sim.GetDevice("test-device-0")
	if device == nil {
		t.Fatal("Device not found")
	}

	initialActivity := device.LastActivity
	if initialActivity.IsZero() {
		t.Error("LastActivity should be set during device creation")
	}

	// LastActivity should be recent (within last second)
	if time.Since(initialActivity) > time.Second {
		t.Error("LastActivity should be recent")
	}
}

// TestDeviceCounters_Initial tests that counters are initialized to zero
func TestDeviceCounters_Initial(t *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := errors.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	device := sim.GetDevice("test-device-0")
	if device == nil {
		t.Fatal("Device not found")
	}

	counters := device.Counters
	if counters.ARPRequestsReceived != 0 {
		t.Error("ARPRequestsReceived should be 0 initially")
	}
	if counters.ARPRepliesSent != 0 {
		t.Error("ARPRepliesSent should be 0 initially")
	}
	if counters.PacketsSent != 0 {
		t.Error("PacketsSent should be 0 initially")
	}
	if counters.PacketsReceived != 0 {
		t.Error("PacketsReceived should be 0 initially")
	}
}

// BenchmarkNewSimulator benchmarks simulator creation
func BenchmarkNewSimulator(b *testing.B) {
	cfg := createTestConfig(10)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := errors.NewStateManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewSimulator(cfg, stack, errorMgr, 0)
	}
}

// BenchmarkGetDevice benchmarks device retrieval
func BenchmarkGetDevice(b *testing.B) {
	cfg := createTestConfig(10)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := errors.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sim.GetDevice("test-device-5")
	}
}

// BenchmarkIncrementCounter benchmarks counter increments
func BenchmarkIncrementCounter(b *testing.B) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := errors.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sim.IncrementCounter("test-device-0", "packets_sent")
	}
}
