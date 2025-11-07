// Package errors provides error injection state management for network simulation testing
package errors

import (
	"fmt"
	"sync"
)

// ErrorType represents types of errors that can be injected
type ErrorType string

const (
	ErrorTypeFCS         ErrorType = "FCS Errors"
	ErrorTypeDiscards    ErrorType = "Packet Discards"
	ErrorTypeInterface   ErrorType = "Interface Errors"
	ErrorTypeUtilization ErrorType = "High Utilization"
	ErrorTypeCPU         ErrorType = "High CPU"
	ErrorTypeMemory      ErrorType = "High Memory"
	ErrorTypeDisk        ErrorType = "High Disk"
)

// AllErrorTypes returns all available error types
func AllErrorTypes() []ErrorType {
	return []ErrorType{
		ErrorTypeFCS,
		ErrorTypeDiscards,
		ErrorTypeInterface,
		ErrorTypeUtilization,
		ErrorTypeCPU,
		ErrorTypeMemory,
		ErrorTypeDisk,
	}
}

// InterfaceConfig represents interface configuration
type InterfaceConfig struct {
	Speed  int    // Mbps
	Duplex string // "full" or "half"
}

// ErrorState represents the current error injection state for a device
type ErrorState struct {
	DeviceIP  string
	Interface string
	ErrorType ErrorType
	Value     int // Error rate or percentage
	IfConfig  InterfaceConfig
	Enabled   bool
}

// StateManager manages error injection state (thread-safe)
type StateManager struct {
	mu     sync.RWMutex
	states map[string]*ErrorState // key: deviceIP:interface
}

// NewStateManager creates a new state manager
func NewStateManager() *StateManager {
	return &StateManager{
		states: make(map[string]*ErrorState),
	}
}

// SetError sets error injection for a device interface
func (sm *StateManager) SetError(deviceIP, iface string, errorType ErrorType, value int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	key := sm.makeKey(deviceIP, iface)
	state, exists := sm.states[key]

	if !exists {
		state = &ErrorState{
			DeviceIP:  deviceIP,
			Interface: iface,
			IfConfig: InterfaceConfig{
				Speed:  1000, // Default 1Gbps
				Duplex: "full",
			},
		}
		sm.states[key] = state
	}

	state.ErrorType = errorType
	state.Value = value
	state.Enabled = value > 0
}

// GetError retrieves error state for a device interface
func (sm *StateManager) GetError(deviceIP, iface string) *ErrorState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	key := sm.makeKey(deviceIP, iface)
	if state, exists := sm.states[key]; exists {
		// Return a copy to avoid race conditions
		stateCopy := *state
		return &stateCopy
	}

	return nil
}

// ClearError clears error injection for a device interface
func (sm *StateManager) ClearError(deviceIP, iface string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	key := sm.makeKey(deviceIP, iface)
	if state, exists := sm.states[key]; exists {
		state.Enabled = false
		state.Value = 0
	}
}

// ClearAll clears all error injections
func (sm *StateManager) ClearAll() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, state := range sm.states {
		state.Enabled = false
		state.Value = 0
	}
}

// GetAllStates returns all current error states
func (sm *StateManager) GetAllStates() []*ErrorState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	states := make([]*ErrorState, 0, len(sm.states))
	for _, state := range sm.states {
		if state.Enabled {
			stateCopy := *state
			states = append(states, &stateCopy)
		}
	}

	return states
}

// SetInterfaceConfig sets interface configuration
func (sm *StateManager) SetInterfaceConfig(deviceIP, iface string, speed int, duplex string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	key := sm.makeKey(deviceIP, iface)
	state, exists := sm.states[key]

	if !exists {
		state = &ErrorState{
			DeviceIP:  deviceIP,
			Interface: iface,
		}
		sm.states[key] = state
	}

	state.IfConfig.Speed = speed
	state.IfConfig.Duplex = duplex
}

// GetInterfaceConfig retrieves interface configuration
func (sm *StateManager) GetInterfaceConfig(deviceIP, iface string) InterfaceConfig {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	key := sm.makeKey(deviceIP, iface)
	if state, exists := sm.states[key]; exists {
		return state.IfConfig
	}

	return InterfaceConfig{
		Speed:  1000,
		Duplex: "full",
	}
}

func (sm *StateManager) makeKey(deviceIP, iface string) string {
	return fmt.Sprintf("%s:%s", deviceIP, iface)
}

// ShouldInjectError determines if an error should be injected based on probability
func ShouldInjectError(errorRate int) bool {
	// Simple implementation - can be enhanced with real randomization
	// For now, inject errors based on a percentage
	return errorRate > 0 && (errorRate >= 100)
}

// CalculateErrorValue calculates the error value based on type and rate
func CalculateErrorValue(errorType ErrorType, baseValue, errorRate int) int {
	if errorRate == 0 {
		return baseValue
	}

	switch errorType {
	case ErrorTypeUtilization, ErrorTypeCPU, ErrorTypeMemory, ErrorTypeDisk:
		// For percentage-based errors, just return the error rate
		return errorRate
	case ErrorTypeFCS, ErrorTypeDiscards, ErrorTypeInterface:
		// For counter-based errors, scale based on rate
		return baseValue + (baseValue * errorRate / 100)
	default:
		return baseValue
	}
}
