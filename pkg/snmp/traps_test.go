package snmp

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/config"
)

// TestNewTrapSender_ValidConfig tests creating a trap sender with valid configuration
func TestNewTrapSender_ValidConfig(t *testing.T) {
	trapConfig := &config.TrapConfig{
		Enabled:   true,
		Receivers: []string{"192.168.1.100:162"},
		ColdStart: &config.TrapTriggerConfig{
			Enabled:   true,
			OnStartup: true,
		},
	}

	deviceIP := net.ParseIP("192.168.1.1")
	ts, err := NewTrapSender("test-device", deviceIP, trapConfig, 1)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if ts == nil {
		t.Fatal("Expected trap sender, got nil")
	}
	if ts.deviceName != "test-device" {
		t.Errorf("Expected device name 'test-device', got '%s'", ts.deviceName)
	}
	if !ts.deviceIP.Equal(deviceIP) {
		t.Errorf("Expected device IP %s, got %s", deviceIP, ts.deviceIP)
	}
	if len(ts.receivers) != 1 {
		t.Errorf("Expected 1 receiver, got %d", len(ts.receivers))
	}
}

// TestNewTrapSender_MultipleReceivers tests creating trap sender with multiple receivers
func TestNewTrapSender_MultipleReceivers(t *testing.T) {
	trapConfig := &config.TrapConfig{
		Enabled: true,
		Receivers: []string{
			"192.168.1.100:162",
			"192.168.1.101:162",
			"10.0.0.50:1162",
		},
	}

	deviceIP := net.ParseIP("192.168.1.1")
	ts, err := NewTrapSender("test-device", deviceIP, trapConfig, 1)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(ts.receivers) != 3 {
		t.Errorf("Expected 3 receivers, got %d", len(ts.receivers))
	}

	// Verify receiver configuration
	if ts.receivers[0].Target != "192.168.1.100" {
		t.Errorf("Expected first receiver target '192.168.1.100', got '%s'", ts.receivers[0].Target)
	}
	if ts.receivers[0].Port != 162 {
		t.Errorf("Expected first receiver port 162, got %d", ts.receivers[0].Port)
	}
	if ts.receivers[2].Port != 1162 {
		t.Errorf("Expected third receiver port 1162, got %d", ts.receivers[2].Port)
	}
}

// TestNewTrapSender_ReceiverWithoutPort tests receiver address without explicit port
func TestNewTrapSender_ReceiverWithoutPort(t *testing.T) {
	trapConfig := &config.TrapConfig{
		Enabled:   true,
		Receivers: []string{"192.168.1.100"}, // No port specified
	}

	deviceIP := net.ParseIP("192.168.1.1")
	ts, err := NewTrapSender("test-device", deviceIP, trapConfig, 1)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(ts.receivers) != 1 {
		t.Errorf("Expected 1 receiver, got %d", len(ts.receivers))
	}
	// Should default to port 162
	if ts.receivers[0].Port != 162 {
		t.Errorf("Expected default port 162, got %d", ts.receivers[0].Port)
	}
}

// TestNewTrapSender_DisabledConfig tests creating trap sender with disabled config
func TestNewTrapSender_DisabledConfig(t *testing.T) {
	trapConfig := &config.TrapConfig{
		Enabled:   false,
		Receivers: []string{"192.168.1.100:162"},
	}

	deviceIP := net.ParseIP("192.168.1.1")
	ts, err := NewTrapSender("test-device", deviceIP, trapConfig, 1)

	if err == nil {
		t.Fatal("Expected error for disabled trap config, got nil")
	}
	if ts != nil {
		t.Error("Expected nil trap sender for disabled config")
	}
}

// TestNewTrapSender_NilConfig tests creating trap sender with nil config
func TestNewTrapSender_NilConfig(t *testing.T) {
	deviceIP := net.ParseIP("192.168.1.1")
	ts, err := NewTrapSender("test-device", deviceIP, nil, 1)

	if err == nil {
		t.Fatal("Expected error for nil trap config, got nil")
	}
	if ts != nil {
		t.Error("Expected nil trap sender for nil config")
	}
}

// TestNewTrapSender_NoReceivers tests creating trap sender without receivers
func TestNewTrapSender_NoReceivers(t *testing.T) {
	trapConfig := &config.TrapConfig{
		Enabled:   true,
		Receivers: []string{}, // Empty receivers list
	}

	deviceIP := net.ParseIP("192.168.1.1")
	ts, err := NewTrapSender("test-device", deviceIP, trapConfig, 1)

	if err == nil {
		t.Fatal("Expected error for empty receivers list, got nil")
	}
	if ts != nil {
		t.Error("Expected nil trap sender for empty receivers")
	}
}

// TestParsePort tests port parsing function
func TestParsePort(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected uint16
	}{
		{"Standard trap port", "162", 162},
		{"Custom port", "1162", 1162},
		{"High port", "10162", 10162},
		{"Port 1", "1", 1},
		{"Max port", "65535", 65535},
		{"Invalid port 0", "0", 162},            // Should default to 162
		{"Invalid port too high", "70000", 162}, // Should default to 162
		{"Invalid string", "abc", 162},          // Should default to 162
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePort(tt.input)
			if result != tt.expected {
				t.Errorf("parsePort(%s) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

// TestTrapSender_Configuration tests trap sender configuration handling
func TestTrapSender_Configuration(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.TrapConfig
		expectError bool
		description string
	}{
		{
			name: "Full configuration",
			config: &config.TrapConfig{
				Enabled:   true,
				Receivers: []string{"192.168.1.100:162"},
				ColdStart: &config.TrapTriggerConfig{
					Enabled:   true,
					OnStartup: true,
				},
				LinkState: &config.LinkStateTrapConfig{
					Enabled:  true,
					LinkDown: true,
					LinkUp:   true,
				},
				AuthenticationFailure: &config.TrapTriggerConfig{
					Enabled: true,
				},
				HighCPU: &config.ThresholdTrapConfig{
					Enabled:   true,
					Threshold: 80,
					Interval:  300,
				},
				HighMemory: &config.ThresholdTrapConfig{
					Enabled:   true,
					Threshold: 85,
					Interval:  300,
				},
				InterfaceErrors: &config.ThresholdTrapConfig{
					Enabled:   true,
					Threshold: 100,
					Interval:  60,
				},
			},
			expectError: false,
			description: "All trap types enabled with thresholds",
		},
		{
			name: "Minimal configuration",
			config: &config.TrapConfig{
				Enabled:   true,
				Receivers: []string{"192.168.1.100:162"},
				ColdStart: &config.TrapTriggerConfig{
					Enabled:   true,
					OnStartup: true,
				},
			},
			expectError: false,
			description: "Only cold start trap enabled",
		},
		{
			name: "Disabled traps",
			config: &config.TrapConfig{
				Enabled:   false,
				Receivers: []string{"192.168.1.100:162"},
			},
			expectError: true,
			description: "Traps disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deviceIP := net.ParseIP("192.168.1.1")
			ts, err := NewTrapSender("test-device", deviceIP, tt.config, 1)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for %s, got %v", tt.description, err)
				}
				if ts == nil {
					t.Errorf("Expected trap sender for %s, got nil", tt.description)
				}
			}
		})
	}
}

// TestTrapSender_Lifecycle tests trap sender start/stop lifecycle
func TestTrapSender_Lifecycle(t *testing.T) {
	trapConfig := &config.TrapConfig{
		Enabled:   true,
		Receivers: []string{"192.168.1.100:162"},
		ColdStart: &config.TrapTriggerConfig{
			Enabled:   true,
			OnStartup: false, // Don't send on startup to avoid connection attempts
		},
	}

	deviceIP := net.ParseIP("192.168.1.1")
	ts, err := NewTrapSender("test-device", deviceIP, trapConfig, 0)
	if err != nil {
		t.Fatalf("Failed to create trap sender: %v", err)
	}

	// Check initial state
	if ts.running {
		t.Error("Expected trap sender to be not running initially")
	}

	// Start the trap sender
	err = ts.Start()
	if err != nil {
		t.Errorf("Failed to start trap sender: %v", err)
	}

	// Verify running state
	if !ts.running {
		t.Error("Expected trap sender to be running after Start()")
	}

	// Try to start again (should fail)
	err = ts.Start()
	if err == nil {
		t.Error("Expected error when starting already running trap sender")
	}

	// Stop the trap sender
	ts.Stop()

	// Give it a moment to stop
	time.Sleep(100 * time.Millisecond)

	// Verify stopped state
	if ts.running {
		t.Error("Expected trap sender to be stopped after Stop()")
	}
}

// TestTrapOIDs tests that standard trap OIDs are correctly defined
func TestTrapOIDs(t *testing.T) {
	tests := []struct {
		name     string
		oid      string
		expected string
	}{
		{"ColdStart", OIDColdStart, ".1.3.6.1.6.3.1.1.5.1"},
		{"WarmStart", OIDWarmStart, ".1.3.6.1.6.3.1.1.5.2"},
		{"LinkDown", OIDLinkDown, ".1.3.6.1.6.3.1.1.5.3"},
		{"LinkUp", OIDLinkUp, ".1.3.6.1.6.3.1.1.5.4"},
		{"AuthenticationFailure", OIDAuthenticationFailure, ".1.3.6.1.6.3.1.1.5.5"},
		{"EgpNeighborLoss", OIDEgpNeighborLoss, ".1.3.6.1.6.3.1.1.5.6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.oid != tt.expected {
				t.Errorf("OID %s = %s, expected %s", tt.name, tt.oid, tt.expected)
			}
		})
	}
}

// TestTrapSender_ThresholdDefaults tests default threshold values
func TestTrapSender_ThresholdDefaults(t *testing.T) {
	trapConfig := &config.TrapConfig{
		Enabled:   true,
		Receivers: []string{"192.168.1.100:162"},
		HighCPU: &config.ThresholdTrapConfig{
			Enabled: true,
			// Threshold and Interval not set, should use defaults
		},
	}

	deviceIP := net.ParseIP("192.168.1.1")
	ts, err := NewTrapSender("test-device", deviceIP, trapConfig, 0)

	if err != nil {
		t.Fatalf("Failed to create trap sender: %v", err)
	}

	// Verify trap sender was created successfully
	if ts == nil {
		t.Fatal("Expected trap sender, got nil")
	}

	// Verify configuration is stored
	if ts.trapConfig.HighCPU == nil {
		t.Error("Expected HighCPU config to be set")
	}
}

// TestTrapSender_DebugLevels tests trap sender with different debug levels
func TestTrapSender_DebugLevels(t *testing.T) {
	trapConfig := &config.TrapConfig{
		Enabled:   true,
		Receivers: []string{"192.168.1.100:162"},
	}

	deviceIP := net.ParseIP("192.168.1.1")

	debugLevels := []int{0, 1, 2, 3}
	for _, level := range debugLevels {
		t.Run(fmt.Sprintf("DebugLevel_%d", level), func(t *testing.T) {
			ts, err := NewTrapSender("test-device", deviceIP, trapConfig, level)
			if err != nil {
				t.Fatalf("Failed to create trap sender with debug level %d: %v", level, err)
			}
			if ts.debugLevel != level {
				t.Errorf("Expected debug level %d, got %d", level, ts.debugLevel)
			}
		})
	}
}

// TestTrapSender_IPv4AndIPv6 tests trap sender with different IP versions
func TestTrapSender_IPv4AndIPv6(t *testing.T) {
	trapConfig := &config.TrapConfig{
		Enabled:   true,
		Receivers: []string{"192.168.1.100:162"},
	}

	tests := []struct {
		name     string
		deviceIP string
	}{
		{"IPv4", "192.168.1.1"},
		{"IPv6", "2001:db8::1"},
		{"IPv4 loopback", "127.0.0.1"},
		{"IPv6 loopback", "::1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deviceIP := net.ParseIP(tt.deviceIP)
			ts, err := NewTrapSender("test-device", deviceIP, trapConfig, 0)
			if err != nil {
				t.Fatalf("Failed to create trap sender for %s: %v", tt.deviceIP, err)
			}
			if !ts.deviceIP.Equal(deviceIP) {
				t.Errorf("Expected device IP %s, got %s", deviceIP, ts.deviceIP)
			}
		})
	}
}

// BenchmarkNewTrapSender benchmarks trap sender creation
func BenchmarkNewTrapSender(b *testing.B) {
	trapConfig := &config.TrapConfig{
		Enabled:   true,
		Receivers: []string{"192.168.1.100:162"},
		ColdStart: &config.TrapTriggerConfig{
			Enabled:   true,
			OnStartup: true,
		},
	}
	deviceIP := net.ParseIP("192.168.1.1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewTrapSender("test-device", deviceIP, trapConfig, 0)
	}
}

// BenchmarkParsePort benchmarks port parsing
func BenchmarkParsePort(b *testing.B) {
	ports := []string{"162", "1162", "10162", "invalid", "0", "70000"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parsePort(ports[i%len(ports)])
	}
}
