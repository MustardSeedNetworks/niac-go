package integration

import (
	"testing"

	"github.com/krisarmstrong/niac-go/pkg/errors"
)

// TestErrorInjection_FCSErrors tests FCS error injection
func TestErrorInjection_FCSErrors(t *testing.T) {
	sm := errors.NewStateManager()

	// Configure interface
	deviceIP := "192.168.1.1"
	iface := "eth0"
	sm.SetInterfaceConfig(deviceIP, iface, 1000, "full")

	// Inject FCS errors at 5% rate
	sm.SetError(deviceIP, iface, errors.ErrorTypeFCS, 5)

	// Verify error state
	state := sm.GetError(deviceIP, iface)
	if state == nil {
		t.Fatal("Error state not set")
	}

	if state.ErrorType != errors.ErrorTypeFCS {
		t.Errorf("Expected FCSErrors, got %s", state.ErrorType)
	}

	if state.Value != 5 {
		t.Errorf("Expected error rate 5, got %d", state.Value)
	}

	// Verify interface config was preserved
	config := sm.GetInterfaceConfig(deviceIP, iface)
	if config.Speed != 1000 {
		t.Errorf("Expected speed 1000, got %d", config.Speed)
	}

	if config.Duplex != "full" {
		t.Errorf("Expected duplex full, got %s", config.Duplex)
	}
}

// TestErrorInjection_PacketDiscards tests packet discard injection
func TestErrorInjection_PacketDiscards(t *testing.T) {
	sm := errors.NewStateManager()

	deviceIP := "192.168.1.2"
	iface := "eth1"

	// Inject packet discards at 10% rate
	sm.SetError(deviceIP, iface, errors.ErrorTypeDiscards, 10)

	state := sm.GetError(deviceIP, iface)
	if state == nil {
		t.Fatal("Error state not set")
	}

	if state.ErrorType != errors.ErrorTypeDiscards {
		t.Errorf("Expected PacketDiscards, got %s", state.ErrorType)
	}

	if state.Value != 10 {
		t.Errorf("Expected error rate 10, got %d", state.Value)
	}

	// Test error calculation
	baseValue := 1000
	errorValue := errors.CalculateErrorValue(errors.ErrorTypeDiscards, baseValue, 10)

	// With 10% error rate, we expect ~100 errors added
	if errorValue < baseValue {
		t.Errorf("Error value %d should be >= base value %d", errorValue, baseValue)
	}
}

// TestErrorInjection_InterfaceErrors tests interface error injection
func TestErrorInjection_InterfaceErrors(t *testing.T) {
	sm := errors.NewStateManager()

	deviceIP := "192.168.1.3"
	iface := "eth2"

	// Inject interface errors at 3% rate
	sm.SetError(deviceIP, iface, errors.ErrorTypeInterface, 3)

	state := sm.GetError(deviceIP, iface)
	if state == nil {
		t.Fatal("Error state not set")
	}

	if state.ErrorType != errors.ErrorTypeInterface {
		t.Errorf("Expected InterfaceErrors, got %s", state.ErrorType)
	}

	// Clear and verify
	sm.ClearError(deviceIP, iface)

	clearedState := sm.GetError(deviceIP, iface)
	if clearedState == nil {
		t.Fatal("Error state should still exist after clear")
	}

	// Verify it's disabled
	if clearedState.Enabled {
		t.Error("Error state should be disabled after clear")
	}

	if clearedState.Value != 0 {
		t.Errorf("Error value should be 0 after clear, got %d", clearedState.Value)
	}
}

// TestErrorInjection_HighUtilization tests high utilization injection
func TestErrorInjection_HighUtilization(t *testing.T) {
	sm := errors.NewStateManager()

	deviceIP := "192.168.1.4"
	iface := "eth3"

	// Inject high utilization at 80% rate
	sm.SetError(deviceIP, iface, errors.ErrorTypeUtilization, 80)

	state := sm.GetError(deviceIP, iface)
	if state == nil {
		t.Fatal("Error state not set")
	}

	if state.ErrorType != errors.ErrorTypeUtilization {
		t.Errorf("Expected HighUtilization, got %s", state.ErrorType)
	}

	if state.Value != 80 {
		t.Errorf("Expected error rate 80, got %d", state.Value)
	}

	// Test utilization calculation
	baseUtilization := 20 // Base 20% utilization
	utilization := errors.CalculateErrorValue(errors.ErrorTypeUtilization, baseUtilization, 80)

	// With 80% error rate, utilization should increase significantly
	if utilization <= baseUtilization {
		t.Errorf("Utilization %d should be > base %d", utilization, baseUtilization)
	}

	// Should not exceed 100%
	if utilization > 100 {
		t.Errorf("Utilization %d should not exceed 100", utilization)
	}
}

// TestErrorInjection_HighCPU tests CPU utilization injection
func TestErrorInjection_HighCPU(t *testing.T) {
	sm := errors.NewStateManager()

	deviceIP := "192.168.1.5"
	iface := "cpu0"

	// Inject high CPU at 90% rate
	sm.SetError(deviceIP, iface, errors.ErrorTypeCPU, 90)

	state := sm.GetError(deviceIP, iface)
	if state == nil {
		t.Fatal("Error state not set")
	}

	if state.ErrorType != errors.ErrorTypeCPU {
		t.Errorf("Expected HighCPU, got %s", state.ErrorType)
	}

	if state.Value != 90 {
		t.Errorf("Expected error rate 90, got %d", state.Value)
	}

	// Test CPU calculation
	baseCPU := 5 // Base 5% CPU
	cpuValue := errors.CalculateErrorValue(errors.ErrorTypeCPU, baseCPU, 90)

	if cpuValue <= baseCPU {
		t.Errorf("CPU value %d should be > base %d", cpuValue, baseCPU)
	}
}

// TestErrorInjection_HighMemory tests memory utilization injection
func TestErrorInjection_HighMemory(t *testing.T) {
	sm := errors.NewStateManager()

	deviceIP := "192.168.1.6"
	iface := "mem0"

	// Inject high memory at 85% rate
	sm.SetError(deviceIP, iface, errors.ErrorTypeMemory, 85)

	state := sm.GetError(deviceIP, iface)
	if state == nil {
		t.Fatal("Error state not set")
	}

	if state.ErrorType != errors.ErrorTypeMemory {
		t.Errorf("Expected HighMemory, got %s", state.ErrorType)
	}

	if state.Value != 85 {
		t.Errorf("Expected error rate 85, got %d", state.Value)
	}
}

// TestErrorInjection_HighDisk tests disk utilization injection
func TestErrorInjection_HighDisk(t *testing.T) {
	sm := errors.NewStateManager()

	deviceIP := "192.168.1.7"
	iface := "disk0"

	// Inject high disk at 95% rate
	sm.SetError(deviceIP, iface, errors.ErrorTypeDisk, 95)

	state := sm.GetError(deviceIP, iface)
	if state == nil {
		t.Fatal("Error state not set")
	}

	if state.ErrorType != errors.ErrorTypeDisk {
		t.Errorf("Expected HighDisk, got %s", state.ErrorType)
	}

	if state.Value != 95 {
		t.Errorf("Expected error rate 95, got %d", state.Value)
	}
}

// TestErrorInjection_MultipleDevicesAndInterfaces tests error injection across multiple devices
func TestErrorInjection_MultipleDevicesAndInterfaces(t *testing.T) {
	sm := errors.NewStateManager()

	// Device 1 with 2 interfaces
	sm.SetError("192.168.1.1", "eth0", errors.ErrorTypeFCS, 5)
	sm.SetError("192.168.1.1", "eth1", errors.ErrorTypeDiscards, 10)

	// Device 2 with 2 interfaces
	sm.SetError("192.168.1.2", "eth0", errors.ErrorTypeInterface, 3)
	sm.SetError("192.168.1.2", "eth1", errors.ErrorTypeUtilization, 80)

	// Get all states
	states := sm.GetAllStates()
	if len(states) != 4 {
		t.Errorf("Expected 4 error states, got %d", len(states))
	}

	// Verify each state
	stateMap := make(map[string]*errors.ErrorState)
	for _, state := range states {
		key := state.DeviceIP + "-" + state.Interface
		stateMap[key] = state
	}

	// Check device 1, eth0
	if state, ok := stateMap["192.168.1.1-eth0"]; !ok {
		t.Error("Missing state for 192.168.1.1-eth0")
	} else if state.ErrorType != errors.ErrorTypeFCS {
		t.Errorf("Wrong error type for 192.168.1.1-eth0: %s", state.ErrorType)
	}

	// Check device 2, eth1
	if state, ok := stateMap["192.168.1.2-eth1"]; !ok {
		t.Error("Missing state for 192.168.1.2-eth1")
	} else if state.Value != 80 {
		t.Errorf("Wrong error rate for 192.168.1.2-eth1: %d", state.Value)
	}
}

// TestErrorInjection_ClearAll tests clearing all error states
func TestErrorInjection_ClearAll(t *testing.T) {
	sm := errors.NewStateManager()

	// Inject errors on multiple devices
	sm.SetError("192.168.1.1", "eth0", errors.ErrorTypeFCS, 5)
	sm.SetError("192.168.1.2", "eth0", errors.ErrorTypeDiscards, 10)
	sm.SetError("192.168.1.3", "eth0", errors.ErrorTypeInterface, 3)

	// Verify states exist
	states := sm.GetAllStates()
	if len(states) != 3 {
		t.Errorf("Expected 3 error states before clear, got %d", len(states))
	}

	// Clear all
	sm.ClearAll()

	// Verify all cleared
	statesAfter := sm.GetAllStates()
	if len(statesAfter) != 0 {
		t.Errorf("Expected 0 error states after clear, got %d", len(statesAfter))
	}
}

// TestErrorInjection_UpdateValue tests updating error rates
func TestErrorInjection_UpdateValue(t *testing.T) {
	sm := errors.NewStateManager()

	deviceIP := "192.168.1.1"
	iface := "eth0"

	// Set initial error rate
	sm.SetError(deviceIP, iface, errors.ErrorTypeFCS, 5)

	state1 := sm.GetError(deviceIP, iface)
	if state1.Value != 5 {
		t.Errorf("Expected initial rate 5, got %d", state1.Value)
	}

	// Update to higher rate
	sm.SetError(deviceIP, iface, errors.ErrorTypeFCS, 20)

	state2 := sm.GetError(deviceIP, iface)
	if state2.Value != 20 {
		t.Errorf("Expected updated rate 20, got %d", state2.Value)
	}

	// Verify it's still the same error type
	if state2.ErrorType != errors.ErrorTypeFCS {
		t.Errorf("Error type changed unexpectedly: %s", state2.ErrorType)
	}
}

// TestErrorInjection_InterfaceConfigPersistence tests that interface config persists through error changes
func TestErrorInjection_InterfaceConfigPersistence(t *testing.T) {
	sm := errors.NewStateManager()

	deviceIP := "192.168.1.1"
	iface := "eth0"

	// Set interface config first
	sm.SetInterfaceConfig(deviceIP, iface, 10000, "full")

	// Verify config
	config1 := sm.GetInterfaceConfig(deviceIP, iface)
	if config1.Speed != 10000 || config1.Duplex != "full" {
		t.Error("Initial interface config not set correctly")
	}

	// Set error
	sm.SetError(deviceIP, iface, errors.ErrorTypeFCS, 5)

	// Verify config still exists
	config2 := sm.GetInterfaceConfig(deviceIP, iface)
	if config2.Speed != 10000 {
		t.Errorf("Interface speed changed: expected 10000, got %d", config2.Speed)
	}
	if config2.Duplex != "full" {
		t.Errorf("Interface duplex changed: expected full, got %s", config2.Duplex)
	}

	// Update interface config
	sm.SetInterfaceConfig(deviceIP, iface, 1000, "half")

	// Verify error still exists
	state := sm.GetError(deviceIP, iface)
	if state == nil {
		t.Fatal("Error state was cleared unexpectedly")
	}
	if state.Value != 5 {
		t.Errorf("Error rate changed: expected 5, got %d", state.Value)
	}

	// Verify new config
	config3 := sm.GetInterfaceConfig(deviceIP, iface)
	if config3.Speed != 1000 || config3.Duplex != "half" {
		t.Error("Interface config not updated correctly")
	}
}

// TestErrorInjection_AllErrorTypes tests all error types are supported
func TestErrorInjection_AllErrorTypes(t *testing.T) {
	sm := errors.NewStateManager()
	deviceIP := "192.168.1.1"

	allTypes := errors.AllErrorTypes()
	if len(allTypes) < 7 {
		t.Errorf("Expected at least 7 error types, got %d", len(allTypes))
	}

	// Inject each error type
	for i, errorType := range allTypes {
		iface := "eth" + string(rune('0'+i))
		sm.SetError(deviceIP, iface, errorType, 10)

		state := sm.GetError(deviceIP, iface)
		if state == nil {
			t.Errorf("Error state not set for %s", errorType)
			continue
		}

		if state.ErrorType != errorType {
			t.Errorf("Wrong error type: expected %s, got %s", errorType, state.ErrorType)
		}
	}

	// Verify all were set
	states := sm.GetAllStates()
	if len(states) != len(allTypes) {
		t.Errorf("Expected %d states, got %d", len(allTypes), len(states))
	}
}
