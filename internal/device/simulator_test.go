package device

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/niac-go/internal/apperr"
	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/logging"
	"github.com/krisarmstrong/niac-go/internal/protocols"
)

// createTestConfig creates a test configuration with one or more devices.
func createTestConfig(deviceCount int) *config.Config {
	cfg := &config.Config{
		Devices: make([]config.Device, deviceCount),
	}

	for i := range deviceCount {
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

// helper to get simulated device by name (panic on missing) for tests
// TestNewSimulator tests creating a new simulator.
func TestNewSimulator(t *testing.T) {
	cfg := createTestConfig(3)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()

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

// TestNewSimulator_EmptyConfig tests simulator with no devices.
func TestNewSimulator_EmptyConfig(t *testing.T) {
	cfg := &config.Config{
		Devices: []config.Device{},
	}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()

	sim := NewSimulator(cfg, stack, errorMgr, 0)

	if sim == nil {
		t.Fatal("Expected simulator, got nil")
	}

	if len(sim.devices) != 0 {
		t.Errorf("Expected 0 devices, got %d", len(sim.devices))
	}
}

// TestSimulator_GetDevice tests retrieving a device by name.
func TestSimulator_GetDevice(t *testing.T) {
	cfg := createTestConfig(2)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
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

// TestSimulator_GetAllDevices tests retrieving all devices.
func TestSimulator_GetAllDevices(t *testing.T) {
	cfg := createTestConfig(3)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	devices := sim.GetAllDevices()

	if len(devices) != 3 {
		t.Errorf("Expected 3 devices, got %d", len(devices))
	}

	// Check that all expected devices are present
	for i := range 3 {
		name := fmt.Sprintf("test-device-%d", i)
		if _, exists := devices[name]; !exists {
			t.Errorf("Expected device '%s' not found", name)
		}
	}
}

// TestSimulator_Lifecycle tests start and stop.
func TestSimulator_Lifecycle(t *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
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

// TestSimulator_SetDeviceState tests setting device state.
func TestSimulator_SetDeviceState(t *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
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

// TestSimulator_DeviceStates tests all device state constants.
func TestSimulator_DeviceStates(t *testing.T) {
	states := []State{
		StateUp,
		StateDown,
		StateStarting,
		StateStopping,
		StateMaintenance,
	}

	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
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

// TestSimulator_IncrementCounter tests counter increments.
func TestSimulator_IncrementCounter(t *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
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

// TestSimulator_IncrementCounter_NonExistentDevice tests incrementing counter for non-existent device.
func TestSimulator_IncrementCounter_NonExistentDevice(_ *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	// Should not panic
	sim.IncrementCounter("non-existent", "arp_requests")
}

// TestSimulator_ConcurrentAccess tests thread-safety with concurrent operations.
func TestSimulator_ConcurrentAccess(t *testing.T) {
	cfg := createTestConfig(5)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	var wg sync.WaitGroup

	// Concurrent device retrieval
	for range 10 {
		wg.Go(func() {
			for range 100 {
				sim.GetAllDevices()
				sim.GetDevice("test-device-0")
			}
		})
	}

	// Concurrent counter increments
	for i := range 10 {
		wg.Add(1)

		go func(deviceIdx int) {
			defer wg.Done()

			deviceName := fmt.Sprintf("test-device-%d", deviceIdx%5)
			for range 100 {
				sim.IncrementCounter(deviceName, "packets_sent")
				sim.IncrementCounter(deviceName, "packets_received")
			}
		}(i)
	}

	// Concurrent state changes
	for i := range 10 {
		wg.Add(1)

		go func(deviceIdx int) {
			defer wg.Done()

			deviceName := fmt.Sprintf("test-device-%d", deviceIdx%5)
			for range 50 {
				_ = sim.SetDeviceState(deviceName, StateUp)
				_ = sim.SetDeviceState(deviceName, StateDown)
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
	for i := range 5 {
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

// TestSimulator_DeviceTypes tests different device types.
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
			errorMgr := apperr.NewStateManager()
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

// TestSimulator_WithTrapSender tests simulator with trap sender enabled.
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
	errorMgr := apperr.NewStateManager()
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

// TestSimulator_LastActivity tests that LastActivity is tracked.
func TestSimulator_LastActivity(t *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
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

// TestDeviceCounters_Initial tests that counters are initialized to zero.
func TestDeviceCounters_Initial(t *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
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

// TestSimulator_ReloadUpdatesSNMPAgent verifies reload rebuilds SNMP agent and trap sender state.
func TestSimulator_ReloadUpdatesSNMPAgent(t *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	originalDevice := sim.GetDevice("test-device-0")
	if originalDevice == nil {
		t.Fatal("Device not found")
	}

	if originalDevice.SNMPAgent == nil {
		t.Fatal("SNMP agent should be initialized")
	}

	originalAgent := originalDevice.SNMPAgent

	newCfg := createTestConfig(1)
	newCfg.Devices[0].SNMPConfig.Community = "private"

	// Create a temporary walk file to trigger load logic
	walkFile := filepath.Join(t.TempDir(), "test.walk")
	err := os.WriteFile(walkFile, []byte(".1.3.6.1.2.1.1.1.0 = STRING: \"NIAC\"\n"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create walk file: %v", err)
	}

	newCfg.Devices[0].SNMPConfig.WalkFile = walkFile
	newCfg.Devices[0].SNMPConfig.Traps = &config.TrapConfig{
		Enabled:   true,
		Receivers: []string{"127.0.0.1:162"},
	}

	err = sim.Reload(newCfg)
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	reloadedDevice := sim.GetDevice("test-device-0")
	if reloadedDevice == nil {
		t.Fatal("Reloaded device not found")
	}

	if reloadedDevice.SNMPAgent == nil {
		t.Fatal("SNMP agent should be reinitialized after reload")
	}

	if reloadedDevice.SNMPAgent == originalAgent {
		t.Error("Expected SNMP agent to be recreated with new configuration")
	}

	if reloadedDevice.TrapSender == nil {
		t.Error("Trap sender should be initialized after reload when traps are enabled")
	}
}

// TestSimulator_GetStats tests statistics aggregation.
func TestSimulator_GetStats(t *testing.T) {
	cfg := createTestConfig(2)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	// Increment some counters
	sim.IncrementCounter("test-device-0", "packets_sent")
	sim.IncrementCounter("test-device-0", "packets_sent")
	sim.IncrementCounter("test-device-1", "packets_sent")
	sim.IncrementCounter("test-device-0", "errors")

	stats := sim.GetStats()

	// Check device count
	deviceCount, ok := stats["device_count"].(int)
	if !ok || deviceCount != 2 {
		t.Errorf("Expected device_count=2, got %v", stats["device_count"])
	}

	// Check running state
	running, ok := stats["running"].(bool)
	if !ok || running {
		t.Errorf("Expected running=false, got %v", stats["running"])
	}

	// Check device states map
	deviceStates, ok := stats["device_states"].(map[string]string)
	if !ok {
		t.Fatal("Expected device_states to be map[string]string")
	}
	if deviceStates["test-device-0"] != string(StateUp) {
		t.Errorf("Expected device-0 state 'up', got '%s'", deviceStates["test-device-0"])
	}

	// Check total counters
	total, ok := stats["total_counters"].(*Counters)
	if !ok {
		t.Fatal("Expected total_counters to be *Counters")
	}
	if total.PacketsSent != 3 {
		t.Errorf("Expected total PacketsSent=3, got %d", total.PacketsSent)
	}
	if total.Errors != 1 {
		t.Errorf("Expected total Errors=1, got %d", total.Errors)
	}
}

// TestSimulator_GetStats_Running tests stats while simulator is running.
func TestSimulator_GetStats_Running(t *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	err := sim.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer sim.Stop()

	stats := sim.GetStats()
	running, ok := stats["running"].(bool)
	if !ok || !running {
		t.Errorf("Expected running=true, got %v", stats["running"])
	}
}

// TestSimulator_GetCounters tests retrieving counter copies.
func TestSimulator_GetCounters(t *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	// Increment several counters
	sim.IncrementCounter("test-device-0", "arp_requests")
	sim.IncrementCounter("test-device-0", "arp_requests")
	sim.IncrementCounter("test-device-0", "icmp_replies")
	sim.IncrementCounter("test-device-0", "packets_sent")

	counters := sim.GetCounters("test-device-0")
	if counters.ARPRequestsReceived != 2 {
		t.Errorf("Expected ARPRequestsReceived=2, got %d", counters.ARPRequestsReceived)
	}
	if counters.ICMPRepliesSent != 1 {
		t.Errorf("Expected ICMPRepliesSent=1, got %d", counters.ICMPRepliesSent)
	}
	if counters.PacketsSent != 1 {
		t.Errorf("Expected PacketsSent=1, got %d", counters.PacketsSent)
	}

	// Modifying returned copy should not affect original
	counters.PacketsSent = 999
	fresh := sim.GetCounters("test-device-0")
	if fresh.PacketsSent != 1 {
		t.Errorf("Modifying returned copy should not affect original, got %d", fresh.PacketsSent)
	}
}

// TestSimulator_GetCounters_NonExistent tests GetCounters for a non-existent device.
func TestSimulator_GetCounters_NonExistent(t *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	counters := sim.GetCounters("no-such-device")
	if counters == nil {
		t.Fatal("Expected non-nil counters for non-existent device")
	}
	if counters.PacketsSent != 0 {
		t.Error("Expected zero counters for non-existent device")
	}
}

// TestSimulator_IncrementCounter_UnknownCounter tests incrementing an unknown counter name.
func TestSimulator_IncrementCounter_UnknownCounter(t *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	// Should not panic or change any counter
	sim.IncrementCounter("test-device-0", "unknown_counter")

	counters := sim.GetCounters("test-device-0")
	total := counters.ARPRequestsReceived + counters.ARPRepliesSent +
		counters.ICMPRequestsReceived + counters.ICMPRepliesSent +
		counters.SNMPQueriesReceived + counters.HTTPRequestsReceived +
		counters.FTPConnectionsReceived + counters.PacketsSent +
		counters.PacketsReceived + counters.Errors
	if total != 0 {
		t.Errorf("Expected all counters to be 0 after unknown increment, total=%d", total)
	}
}

// TestSimulator_Reload_RemoveDevice tests reload that removes a device.
func TestSimulator_Reload_RemoveDevice(t *testing.T) {
	cfg := createTestConfig(3) // test-device-0, test-device-1, test-device-2
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	if len(sim.GetAllDevices()) != 3 {
		t.Fatalf("Expected 3 devices initially, got %d", len(sim.GetAllDevices()))
	}

	// New config: keep device-0 and device-2, drop device-1
	newCfg := &config.Config{
		Devices: []config.Device{
			{
				Name:        "test-device-0",
				Type:        "switch",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
				SNMPConfig:  config.SNMPConfig{Community: "public"},
			},
			{
				Name:        "test-device-2",
				Type:        "server",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x57},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.3")},
				SNMPConfig:  config.SNMPConfig{Community: "public"},
			},
		},
	}

	err := sim.Reload(newCfg)
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	devices := sim.GetAllDevices()
	if len(devices) != 2 {
		t.Fatalf("Expected 2 devices after reload, got %d", len(devices))
	}

	if sim.GetDevice("test-device-0") == nil {
		t.Error("test-device-0 should still exist")
	}
	if sim.GetDevice("test-device-1") != nil {
		t.Error("test-device-1 should be removed")
	}
	if sim.GetDevice("test-device-2") == nil {
		t.Error("test-device-2 should still exist")
	}
}

// TestSimulator_Reload_WhileRunning tests reload while the simulator is running.
func TestSimulator_Reload_WhileRunning(t *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	err := sim.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Reload with same device (update, not add) to avoid addDevice deadlock
	newCfg := createTestConfig(1)
	newCfg.Devices[0].Type = "switch" // change type to trigger update path
	err = sim.Reload(newCfg)
	if err != nil {
		t.Fatalf("Reload while running failed: %v", err)
	}

	// Simulator should be running again after reload
	if !sim.running {
		t.Error("Simulator should be running after reload (was running before)")
	}

	sim.Stop()
}

// TestSimulator_Reload_WhileStopped tests reload while the simulator is stopped.
func TestSimulator_Reload_WhileStopped(t *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	// Reload with same device name to exercise update path
	newCfg := createTestConfig(1)
	newCfg.Devices[0].SNMPConfig.Community = "private"
	err := sim.Reload(newCfg)
	if err != nil {
		t.Fatalf("Reload while stopped failed: %v", err)
	}

	if sim.running {
		t.Error("Simulator should not be running after reload (was stopped before)")
	}

	if len(sim.GetAllDevices()) != 1 {
		t.Errorf("Expected 1 device after reload, got %d", len(sim.GetAllDevices()))
	}
}

// TestSimulator_PerformDeviceBehavior tests periodic behavior dispatch for all device types.
func TestSimulator_PerformDeviceBehavior(t *testing.T) {
	deviceTypes := []string{"router", "switch", "ap", "access-point", "server", "unknown-type"}

	for _, dt := range deviceTypes {
		t.Run(dt, func(t *testing.T) {
			cfg := &config.Config{
				Devices: []config.Device{
					{
						Name:        "behavior-device",
						Type:        dt,
						MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
						IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
						SNMPConfig:  config.SNMPConfig{Community: "public"},
					},
				},
			}
			stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
			errorMgr := apperr.NewStateManager()
			sim := NewSimulator(cfg, stack, errorMgr, 0)

			device := sim.GetDevice("behavior-device")
			before := device.LastActivity

			// Should not panic for any device type
			sim.performDeviceBehavior("behavior-device", device)

			if device.LastActivity.Before(before) || device.LastActivity.Equal(before) {
				t.Error("LastActivity should be updated after performDeviceBehavior")
			}
		})
	}
}

// TestSimulator_StateTransitions tests isTransitionToDown and isTransitionToUp logic.
func TestSimulator_StateTransitions(t *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	downTests := []struct {
		name     string
		old      State
		new      State
		expected bool
	}{
		{"up to down", StateUp, StateDown, true},
		{"up to stopping", StateUp, StateStopping, true},
		{"starting to down", StateStarting, StateDown, true},
		{"down to down", StateDown, StateDown, false},
		{"stopping to down", StateStopping, StateDown, false},
		{"up to up", StateUp, StateUp, false},
		{"up to maintenance", StateUp, StateMaintenance, false},
	}

	for _, tc := range downTests {
		t.Run("down/"+tc.name, func(t *testing.T) {
			got := sim.isTransitionToDown(tc.old, tc.new)
			if got != tc.expected {
				t.Errorf("isTransitionToDown(%s, %s) = %v, want %v", tc.old, tc.new, got, tc.expected)
			}
		})
	}

	upTests := []struct {
		name     string
		old      State
		new      State
		expected bool
	}{
		{"down to up", StateDown, StateUp, true},
		{"starting to up", StateStarting, StateUp, true},
		{"stopping to up", StateStopping, StateUp, true},
		{"maintenance to up", StateMaintenance, StateUp, true},
		{"up to up", StateUp, StateUp, false},
		{"down to starting", StateDown, StateStarting, false},
	}

	for _, tc := range upTests {
		t.Run("up/"+tc.name, func(t *testing.T) {
			got := sim.isTransitionToUp(tc.old, tc.new)
			if got != tc.expected {
				t.Errorf("isTransitionToUp(%s, %s) = %v, want %v", tc.old, tc.new, got, tc.expected)
			}
		})
	}
}

// TestSimulator_GetInterfaceDescription tests interface description generation.
func TestSimulator_GetInterfaceDescription(t *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	tests := []struct {
		name        string
		ipAddresses []net.IP
		expected    string
	}{
		{
			name:        "device with IP",
			ipAddresses: []net.IP{net.ParseIP("192.168.1.1")},
			expected:    "Interface 192.168.1.1",
		},
		{
			name:        "device without IP",
			ipAddresses: nil,
			expected:    "Primary Interface",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			device := &SimulatedDevice{
				Config: &config.Device{
					IPAddresses: tc.ipAddresses,
				},
			}
			got := sim.getInterfaceDescription(device)
			if got != tc.expected {
				t.Errorf("getInterfaceDescription() = %q, want %q", got, tc.expected)
			}
		})
	}
}

// TestSimulator_GetAllDevices_ReturnsCopy tests that GetAllDevices returns a separate map.
func TestSimulator_GetAllDevices_ReturnsCopy(t *testing.T) {
	cfg := createTestConfig(2)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	devices := sim.GetAllDevices()
	originalLen := len(devices)

	// Mutating the returned map should not affect the simulator
	delete(devices, "test-device-0")

	fresh := sim.GetAllDevices()
	if len(fresh) != originalLen {
		t.Errorf("Deleting from returned map affected simulator: expected %d devices, got %d",
			originalLen, len(fresh))
	}
}

// TestSimulator_StartStop_Restart tests that a stopped simulator can be restarted.
func TestSimulator_StartStop_Restart(t *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	// First cycle
	err := sim.Start()
	if err != nil {
		t.Fatalf("First start failed: %v", err)
	}
	sim.Stop()

	// Second cycle
	err = sim.Start()
	if err != nil {
		t.Fatalf("Restart failed: %v", err)
	}
	if !sim.running {
		t.Error("Simulator should be running after restart")
	}
	sim.Stop()

	if sim.running {
		t.Error("Simulator should be stopped after second stop")
	}
}

// TestSimulator_SetDeviceState_ErrorWrapping tests error wrapping for missing device.
func TestSimulator_SetDeviceState_ErrorWrapping(t *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	err := sim.SetDeviceState("ghost", StateUp)
	if err == nil {
		t.Fatal("Expected error for non-existent device")
	}
	if !errors.Is(err, ErrDeviceNotFound) {
		t.Errorf("Expected ErrDeviceNotFound, got %v", err)
	}
}

// TestSimulator_SendStateChangeTraps_NilTrapSender tests that trap sending is skipped when nil.
func TestSimulator_SendStateChangeTraps_NilTrapSender(t *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	device := sim.GetDevice("test-device-0")
	// TrapSender should be nil for basic config
	if device.TrapSender != nil {
		t.Skip("TrapSender unexpectedly non-nil")
	}

	// Should not panic
	sim.sendStateChangeTraps(device, "test-device-0", StateUp, StateDown)
}

// TestSimulator_DeviceNoIP tests device creation with no IP addresses.
func TestSimulator_DeviceNoIP(t *testing.T) {
	cfg := &config.Config{
		Devices: []config.Device{
			{
				Name:       "no-ip-device",
				Type:       "switch",
				MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				SNMPConfig: config.SNMPConfig{Community: "public"},
			},
		},
	}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 1) // debugLevel=1 to hit the "no-ip" branch

	device := sim.GetDevice("no-ip-device")
	if device == nil {
		t.Fatal("Device should be created even without IP")
	}
}

// TestSimulator_ShouldCreateTrapSender tests the shouldCreateTrapSender predicate.
func TestSimulator_ShouldCreateTrapSender(t *testing.T) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	tests := []struct {
		name     string
		device   *config.Device
		expected bool
	}{
		{
			name: "traps enabled with IP",
			device: &config.Device{
				IPAddresses: []net.IP{net.ParseIP("10.0.0.1")},
				SNMPConfig: config.SNMPConfig{
					Traps: &config.TrapConfig{Enabled: true, Receivers: []string{"10.0.0.100:162"}},
				},
			},
			expected: true,
		},
		{
			name: "traps disabled",
			device: &config.Device{
				IPAddresses: []net.IP{net.ParseIP("10.0.0.1")},
				SNMPConfig: config.SNMPConfig{
					Traps: &config.TrapConfig{Enabled: false},
				},
			},
			expected: false,
		},
		{
			name: "traps nil",
			device: &config.Device{
				IPAddresses: []net.IP{net.ParseIP("10.0.0.1")},
				SNMPConfig:  config.SNMPConfig{},
			},
			expected: false,
		},
		{
			name: "traps enabled no IP",
			device: &config.Device{
				SNMPConfig: config.SNMPConfig{
					Traps: &config.TrapConfig{Enabled: true},
				},
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sim.shouldCreateTrapSender(tc.device)
			if got != tc.expected {
				t.Errorf("shouldCreateTrapSender() = %v, want %v", got, tc.expected)
			}
		})
	}
}

// BenchmarkNewSimulator benchmarks simulator creation.
func BenchmarkNewSimulator(b *testing.B) {
	cfg := createTestConfig(10)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()

	for b.Loop() {
		NewSimulator(cfg, stack, errorMgr, 0)
	}
}

// BenchmarkGetDevice benchmarks device retrieval.
func BenchmarkGetDevice(b *testing.B) {
	cfg := createTestConfig(10)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	for b.Loop() {
		sim.GetDevice("test-device-5")
	}
}

// BenchmarkIncrementCounter benchmarks counter increments.
func BenchmarkIncrementCounter(b *testing.B) {
	cfg := createTestConfig(1)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	errorMgr := apperr.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	for b.Loop() {
		sim.IncrementCounter("test-device-0", "packets_sent")
	}
}
