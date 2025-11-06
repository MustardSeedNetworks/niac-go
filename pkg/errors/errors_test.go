package errors

import (
	"testing"
)

func TestStateManager(t *testing.T) {
	sm := NewStateManager()

	// Test SetError
	sm.SetError("192.168.1.1", "eth0", ErrorTypeFCS, 50)

	// Test GetError
	state := sm.GetError("192.168.1.1", "eth0")
	if state == nil {
		t.Fatal("GetError returned nil")
	}
	if state.ErrorType != ErrorTypeFCS {
		t.Errorf("Expected ErrorTypeFCS, got %v", state.ErrorType)
	}
	if state.Value != 50 {
		t.Errorf("Expected value 50, got %d", state.Value)
	}
	if !state.Enabled {
		t.Error("Expected state to be enabled")
	}

	// Test ClearError
	sm.ClearError("192.168.1.1", "eth0")
	state = sm.GetError("192.168.1.1", "eth0")
	if state.Enabled {
		t.Error("Expected state to be disabled")
	}
}

func TestStateManagerMultipleDevices(t *testing.T) {
	sm := NewStateManager()

	// Set errors on multiple devices
	sm.SetError("192.168.1.1", "eth0", ErrorTypeFCS, 50)
	sm.SetError("192.168.1.2", "eth0", ErrorTypeDiscards, 25)
	sm.SetError("192.168.1.3", "eth1", ErrorTypeCPU, 90)

	// Get all states
	states := sm.GetAllStates()
	if len(states) != 3 {
		t.Errorf("Expected 3 active states, got %d", len(states))
	}

	// Clear all
	sm.ClearAll()
	states = sm.GetAllStates()
	if len(states) != 0 {
		t.Errorf("Expected 0 active states after ClearAll, got %d", len(states))
	}
}

func TestInterfaceConfig(t *testing.T) {
	sm := NewStateManager()

	// Set interface config
	sm.SetInterfaceConfig("192.168.1.1", "eth0", 10000, "full")

	// Get interface config
	cfg := sm.GetInterfaceConfig("192.168.1.1", "eth0")
	if cfg.Speed != 10000 {
		t.Errorf("Expected speed 10000, got %d", cfg.Speed)
	}
	if cfg.Duplex != "full" {
		t.Errorf("Expected duplex 'full', got '%s'", cfg.Duplex)
	}

	// Get non-existent interface (should return defaults)
	cfg = sm.GetInterfaceConfig("192.168.1.99", "eth99")
	if cfg.Speed != 1000 {
		t.Errorf("Expected default speed 1000, got %d", cfg.Speed)
	}
	if cfg.Duplex != "full" {
		t.Errorf("Expected default duplex 'full', got '%s'", cfg.Duplex)
	}
}

func TestAllErrorTypes(t *testing.T) {
	types := AllErrorTypes()
	if len(types) != 7 {
		t.Errorf("Expected 7 error types, got %d", len(types))
	}

	// Verify all expected types are present
	expectedTypes := map[ErrorType]bool{
		ErrorTypeFCS:         false,
		ErrorTypeDiscards:    false,
		ErrorTypeInterface:   false,
		ErrorTypeUtilization: false,
		ErrorTypeCPU:         false,
		ErrorTypeMemory:      false,
		ErrorTypeDisk:        false,
	}

	for _, et := range types {
		if _, exists := expectedTypes[et]; !exists {
			t.Errorf("Unexpected error type: %v", et)
		}
		expectedTypes[et] = true
	}

	for et, found := range expectedTypes {
		if !found {
			t.Errorf("Missing error type: %v", et)
		}
	}
}

func TestCalculateErrorValue(t *testing.T) {
	tests := []struct {
		errorType ErrorType
		baseValue int
		errorRate int
		expected  int
	}{
		{ErrorTypeCPU, 50, 90, 90},        // Percentage-based
		{ErrorTypeMemory, 60, 85, 85},     // Percentage-based
		{ErrorTypeFCS, 100, 50, 150},      // Counter-based: 100 + (100 * 50 / 100)
		{ErrorTypeDiscards, 100, 25, 125}, // Counter-based
		{ErrorTypeFCS, 100, 0, 100},       // Zero rate
	}

	for _, tt := range tests {
		result := CalculateErrorValue(tt.errorType, tt.baseValue, tt.errorRate)
		if result != tt.expected {
			t.Errorf("%v: expected %d, got %d", tt.errorType, tt.expected, result)
		}
	}
}

func TestConcurrentAccess(t *testing.T) {
	sm := NewStateManager()

	// Test concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				sm.SetError("192.168.1.1", "eth0", ErrorTypeFCS, j)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic, state should be valid
	state := sm.GetError("192.168.1.1", "eth0")
	if state == nil {
		t.Fatal("State should exist after concurrent writes")
	}
}

func BenchmarkSetError(b *testing.B) {
	sm := NewStateManager()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sm.SetError("192.168.1.1", "eth0", ErrorTypeFCS, 50)
	}
}

func BenchmarkGetError(b *testing.B) {
	sm := NewStateManager()
	sm.SetError("192.168.1.1", "eth0", ErrorTypeFCS, 50)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = sm.GetError("192.168.1.1", "eth0")
	}
}

func BenchmarkGetAllStates(b *testing.B) {
	sm := NewStateManager()
	for i := 0; i < 100; i++ {
		sm.SetError("192.168.1.1", "eth0", ErrorTypeFCS, 50)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = sm.GetAllStates()
	}
}
