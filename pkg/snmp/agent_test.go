package snmp

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/krisarmstrong/niac-go/pkg/config"
)

// createTestDevice creates a test device configuration
func createTestDevice() *config.Device {
	mac, _ := net.ParseMAC("00:11:22:33:44:55")
	return &config.Device{
		Name:       "test-device",
		Type:       "router",
		MACAddress: mac,
		IPAddresses: []net.IP{
			net.ParseIP("192.168.1.1"),
		},
		SNMPConfig: config.SNMPConfig{
			Community: "public",
		},
		Properties: make(map[string]string),
	}
}

// TestNewAgent tests agent creation
func TestNewAgent(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	if agent == nil {
		t.Fatal("NewAgent returned nil")
	}

	if agent.device != device {
		t.Error("Agent device not set correctly")
	}

	if agent.mib == nil {
		t.Error("Agent MIB not initialized")
	}

	if agent.community != "public" {
		t.Errorf("Expected community 'public', got '%s'", agent.community)
	}

	if agent.startTime.IsZero() {
		t.Error("Agent start time not set")
	}

	if agent.debugLevel != 0 {
		t.Errorf("Expected debug level 0, got %d", agent.debugLevel)
	}
}

// TestNewAgent_CustomCommunity tests agent with custom community
func TestNewAgent_CustomCommunity(t *testing.T) {
	device := createTestDevice()
	device.SNMPConfig.Community = "private"

	agent := NewAgent(device, 0)

	if agent.community != "private" {
		t.Errorf("Expected community 'private', got '%s'", agent.community)
	}
}

// TestNewAgent_DebugLevel tests agent with different debug levels
func TestNewAgent_DebugLevel(t *testing.T) {
	device := createTestDevice()

	tests := []int{0, 1, 2, 3}
	for _, level := range tests {
		agent := NewAgent(device, level)
		if agent.debugLevel != level {
			t.Errorf("Expected debug level %d, got %d", level, agent.debugLevel)
		}
	}
}

// TestAgent_InitializeSystemMIB tests system MIB initialization
func TestAgent_InitializeSystemMIB(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	// Test that standard system MIB objects are initialized
	tests := []struct {
		oid         string
		name        string
		shouldExist bool
	}{
		{"1.3.6.1.2.1.1.1.0", "sysDescr", true},
		{"1.3.6.1.2.1.1.2.0", "sysObjectID", true},
		{"1.3.6.1.2.1.1.3.0", "sysUpTime", true},
		{"1.3.6.1.2.1.1.4.0", "sysContact", true},
		{"1.3.6.1.2.1.1.5.0", "sysName", true},
		{"1.3.6.1.2.1.1.6.0", "sysLocation", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := agent.mib.Get(tt.oid)
			if tt.shouldExist && value == nil {
				t.Errorf("Expected OID %s (%s) to be initialized", tt.oid, tt.name)
			}
		})
	}
}

// TestAgent_SystemMIB_DefaultValues tests default values for system MIB
func TestAgent_SystemMIB_DefaultValues(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	// sysDescr should contain device type and name
	sysDescr := agent.mib.Get("1.3.6.1.2.1.1.1.0")
	if sysDescr == nil {
		t.Fatal("sysDescr not initialized")
	}
	descr := sysDescr.Value.(string)
	if !strings.Contains(descr, device.Type) || !strings.Contains(descr, device.Name) {
		t.Errorf("sysDescr should contain device type and name, got: %s", descr)
	}

	// sysName should be device name
	sysName := agent.mib.Get("1.3.6.1.2.1.1.5.0")
	if sysName == nil {
		t.Fatal("sysName not initialized")
	}
	if sysName.Value.(string) != device.Name {
		t.Errorf("Expected sysName '%s', got '%s'", device.Name, sysName.Value.(string))
	}
}

// TestAgent_SystemMIB_CustomProperties tests system MIB with custom properties
func TestAgent_SystemMIB_CustomProperties(t *testing.T) {
	device := createTestDevice()
	device.Properties["sysDescr"] = "Custom Description"
	device.Properties["sysContact"] = "admin@test.com"
	device.Properties["sysName"] = "custom-name"
	device.Properties["sysLocation"] = "Data Center 1"
	device.Properties["sysObjectID"] = "1.2.3.4.5"

	agent := NewAgent(device, 0)

	tests := []struct {
		oid      string
		property string
		expected string
	}{
		{"1.3.6.1.2.1.1.1.0", "sysDescr", "Custom Description"},
		{"1.3.6.1.2.1.1.4.0", "sysContact", "admin@test.com"},
		{"1.3.6.1.2.1.1.5.0", "sysName", "custom-name"},
	}

	for _, tt := range tests {
		t.Run(tt.property, func(t *testing.T) {
			value := agent.mib.Get(tt.oid)
			if value == nil {
				t.Fatalf("OID %s not initialized", tt.oid)
			}
			if value.Value.(string) != tt.expected {
				t.Errorf("Expected %s '%s', got '%s'", tt.property, tt.expected, value.Value.(string))
			}
		})
	}
}

// TestAgent_SysUpTime tests dynamic sysUpTime OID
func TestAgent_SysUpTime(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	// Get sysUpTime immediately
	uptime1 := agent.mib.Get("1.3.6.1.2.1.1.3.0")
	if uptime1 == nil {
		t.Fatal("sysUpTime not initialized")
	}

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Get sysUpTime again
	uptime2 := agent.mib.Get("1.3.6.1.2.1.1.3.0")
	if uptime2 == nil {
		t.Fatal("sysUpTime not available on second read")
	}

	// Second value should be greater than first
	val1 := uptime1.Value.(uint32)
	val2 := uptime2.Value.(uint32)

	if val2 <= val1 {
		t.Errorf("sysUpTime should increase over time: first=%d, second=%d", val1, val2)
	}
}

// TestAgent_MIB_GetSet tests MIB get/set operations through agent
func TestAgent_MIB_GetSet(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	// Set a custom OID
	testOID := "1.3.6.1.4.1.9999.1.1.0"
	testValue := &OIDValue{
		Type:  gosnmp.OctetString,
		Value: "test value",
	}

	agent.mib.Set(testOID, testValue)

	// Get the value back
	retrieved := agent.mib.Get(testOID)
	if retrieved == nil {
		t.Fatal("Failed to retrieve set OID")
	}

	if retrieved.Type != testValue.Type {
		t.Errorf("Expected type %v, got %v", testValue.Type, retrieved.Type)
	}

	if retrieved.Value.(string) != testValue.Value.(string) {
		t.Errorf("Expected value '%s', got '%s'", testValue.Value, retrieved.Value)
	}
}

// TestAgent_ConcurrentAccess tests concurrent access to agent
func TestAgent_ConcurrentAccess(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	done := make(chan bool, 100)

	// Launch multiple goroutines accessing MIB
	for i := 0; i < 100; i++ {
		go func(id int) {
			// Read sysUpTime
			_ = agent.mib.Get("1.3.6.1.2.1.1.3.0")

			// Read sysName
			_ = agent.mib.Get("1.3.6.1.2.1.1.5.0")

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}
}

// TestAgent_StartTime tests that start time is set correctly
func TestAgent_StartTime(t *testing.T) {
	before := time.Now()
	device := createTestDevice()
	agent := NewAgent(device, 0)
	after := time.Now()

	if agent.startTime.Before(before) || agent.startTime.After(after) {
		t.Error("Agent start time not within expected range")
	}
}

// TestAgent_DeviceTypes tests agents with different device types
func TestAgent_DeviceTypes(t *testing.T) {
	types := []string{"router", "switch", "ap", "server", "firewall"}

	for _, deviceType := range types {
		t.Run(deviceType, func(t *testing.T) {
			device := createTestDevice()
			device.Type = deviceType

			agent := NewAgent(device, 0)

			sysDescr := agent.mib.Get("1.3.6.1.2.1.1.1.0")
			if sysDescr == nil {
				t.Fatal("sysDescr not initialized")
			}

			descr := sysDescr.Value.(string)
			if !strings.Contains(descr, deviceType) {
				t.Errorf("sysDescr should contain device type '%s', got: %s", deviceType, descr)
			}
		})
	}
}

// TestAgent_MultipleSNMPAgents tests creating multiple agents
func TestAgent_MultipleSNMPAgents(t *testing.T) {
	devices := make([]*config.Device, 10)
	agents := make([]*Agent, 10)

	for i := 0; i < 10; i++ {
		devices[i] = createTestDevice()
		devices[i].Name = "device-" + string(rune('0'+i))
		agents[i] = NewAgent(devices[i], 0)
	}

	// Verify each agent has correct device
	for i := 0; i < 10; i++ {
		if agents[i].device.Name != devices[i].Name {
			t.Errorf("Agent %d has wrong device name", i)
		}

		sysName := agents[i].mib.Get("1.3.6.1.2.1.1.5.0")
		if sysName.Value.(string) != devices[i].Name {
			t.Errorf("Agent %d has wrong sysName", i)
		}
	}
}

// BenchmarkNewAgent benchmarks agent creation
func BenchmarkNewAgent(b *testing.B) {
	device := createTestDevice()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewAgent(device, 0)
	}
}

// BenchmarkAgent_MIBGet benchmarks MIB Get operations
func BenchmarkAgent_MIBGet(b *testing.B) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = agent.mib.Get("1.3.6.1.2.1.1.1.0")
	}
}

// BenchmarkAgent_SysUpTime benchmarks dynamic sysUpTime reads
func BenchmarkAgent_SysUpTime(b *testing.B) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = agent.mib.Get("1.3.6.1.2.1.1.3.0")
	}
}

// BenchmarkAgent_ConcurrentGet benchmarks concurrent MIB access
func BenchmarkAgent_ConcurrentGet(b *testing.B) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = agent.mib.Get("1.3.6.1.2.1.1.1.0")
		}
	})
}
