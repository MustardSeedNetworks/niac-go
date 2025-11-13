package snmp

import (
	"fmt"
	"net"
	"os"
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

// TestAgent_HandleGet tests SNMP GET request handling
func TestAgent_HandleGet(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	// Test successful GET
	result, err := agent.HandleGet("1.3.6.1.2.1.1.1.0")
	if err != nil {
		t.Fatalf("HandleGet failed: %v", err)
	}
	if result == nil {
		t.Fatal("HandleGet returned nil result")
	}

	// Test GET for non-existent OID
	_, err = agent.HandleGet("1.2.3.4.5.6.7.8.9")
	if err == nil {
		t.Error("Expected error for non-existent OID")
	}
}

// TestAgent_HandleGet_AllSystemOIDs tests GET for all system MIB OIDs
func TestAgent_HandleGet_AllSystemOIDs(t *testing.T) {
	device := createTestDevice()
	device.Properties["sysDescr"] = "Test System"
	device.Properties["sysContact"] = "test@example.com"
	device.Properties["sysName"] = "test-name"
	device.Properties["sysLocation"] = "Test Location"
	device.Properties["sysObjectID"] = "1.3.6.1.4.1.9.1.1"

	agent := NewAgent(device, 0)

	systemOIDs := []string{
		"1.3.6.1.2.1.1.1.0", // sysDescr
		"1.3.6.1.2.1.1.2.0", // sysObjectID
		"1.3.6.1.2.1.1.3.0", // sysUpTime
		"1.3.6.1.2.1.1.4.0", // sysContact
		"1.3.6.1.2.1.1.5.0", // sysName
		"1.3.6.1.2.1.1.6.0", // sysLocation
		"1.3.6.1.2.1.1.7.0", // sysServices
	}

	for _, oid := range systemOIDs {
		t.Run(oid, func(t *testing.T) {
			result, err := agent.HandleGet(oid)
			if err != nil {
				t.Errorf("HandleGet(%s) failed: %v", oid, err)
			}
			if result == nil {
				t.Errorf("HandleGet(%s) returned nil", oid)
			}
		})
	}
}

// TestAgent_HandleGetNext tests SNMP GET-NEXT request handling
func TestAgent_HandleGetNext(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	// Test GET-NEXT from beginning
	nextOID, value, err := agent.HandleGetNext("1.3.6.1.2.1.1")
	if err != nil {
		t.Fatalf("HandleGetNext failed: %v", err)
	}
	if nextOID == "" {
		t.Error("HandleGetNext returned empty OID")
	}
	if value == nil {
		t.Error("HandleGetNext returned nil value")
	}
}

// TestAgent_HandleGetNext_Traversal tests GET-NEXT traversal through MIB
func TestAgent_HandleGetNext_Traversal(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	// Walk through system MIB
	currentOID := "1.3.6.1.2.1.1"
	visited := make(map[string]bool)
	maxIterations := 20

	for i := 0; i < maxIterations; i++ {
		nextOID, value, err := agent.HandleGetNext(currentOID)
		if err != nil {
			break // End of MIB
		}
		if nextOID == "" || value == nil {
			break
		}

		// Check for loops
		if visited[nextOID] {
			t.Errorf("Loop detected at OID %s", nextOID)
			break
		}
		visited[nextOID] = true

		// Verify OID progression
		if compareOIDs(nextOID, currentOID) <= 0 {
			t.Errorf("OID did not advance: %s -> %s", currentOID, nextOID)
		}

		currentOID = nextOID
	}

	if len(visited) == 0 {
		t.Error("No OIDs traversed")
	}
}

// TestAgent_HandleGetNext_EndOfMIB tests GET-NEXT at end of MIB
func TestAgent_HandleGetNext_EndOfMIB(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	// Request next after a very high OID
	_, _, err := agent.HandleGetNext("9.9.9.9.9.9.9")
	if err == nil {
		t.Error("Expected error at end of MIB")
	}
}

// TestAgent_HandleGetBulk tests SNMP GET-BULK request handling
func TestAgent_HandleGetBulk(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	// Test GET-BULK with maxRepetitions = 5
	results, err := agent.HandleGetBulk("1.3.6.1.2.1.1", 5)
	if err != nil {
		t.Fatalf("HandleGetBulk failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("HandleGetBulk returned no results")
	}

	if len(results) > 5 {
		t.Errorf("HandleGetBulk returned too many results: %d > 5", len(results))
	}

	// Verify results are in order
	for i := 1; i < len(results); i++ {
		if compareOIDs(results[i].OID, results[i-1].OID) <= 0 {
			t.Errorf("Results not in order: %s -> %s", results[i-1].OID, results[i].OID)
		}
	}
}

// TestAgent_HandleGetBulk_LargeRequest tests GET-BULK with large maxRepetitions
func TestAgent_HandleGetBulk_LargeRequest(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	results, err := agent.HandleGetBulk("1.3.6.1.2.1.1", 100)
	if err != nil {
		t.Fatalf("HandleGetBulk failed: %v", err)
	}

	// Should return all available OIDs under the subtree
	if len(results) == 0 {
		t.Error("HandleGetBulk returned no results")
	}
}

// TestAgent_HandleGetBulk_ZeroRepetitions tests GET-BULK with zero maxRepetitions
func TestAgent_HandleGetBulk_ZeroRepetitions(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	results, err := agent.HandleGetBulk("1.3.6.1.2.1.1", 0)
	if err != nil {
		t.Fatalf("HandleGetBulk failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

// TestAgent_SetOID tests SNMP SET operation
func TestAgent_SetOID(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	testOID := "1.3.6.1.4.1.9999.1.0"
	testValue := &OIDValue{
		Type:  gosnmp.OctetString,
		Value: "test value",
	}

	err := agent.SetOID(testOID, testValue)
	if err != nil {
		t.Fatalf("SetOID failed: %v", err)
	}

	// Verify value was set
	result, err := agent.HandleGet(testOID)
	if err != nil {
		t.Fatalf("HandleGet after SetOID failed: %v", err)
	}

	if result.Value.(string) != testValue.Value.(string) {
		t.Errorf("Expected value '%s', got '%s'", testValue.Value, result.Value)
	}
}

// TestAgent_SetOID_UpdateExisting tests updating an existing OID
func TestAgent_SetOID_UpdateExisting(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	// Update sysContact
	newContact := "newadmin@example.com"
	err := agent.SetOID("1.3.6.1.2.1.1.4.0", &OIDValue{
		Type:  gosnmp.OctetString,
		Value: newContact,
	})
	if err != nil {
		t.Fatalf("SetOID failed: %v", err)
	}

	// Verify update
	result, err := agent.HandleGet("1.3.6.1.2.1.1.4.0")
	if err != nil {
		t.Fatalf("HandleGet failed: %v", err)
	}

	if result.Value.(string) != newContact {
		t.Errorf("Expected contact '%s', got '%s'", newContact, result.Value)
	}
}

// TestAgent_GetCommunity tests community string retrieval
func TestAgent_GetCommunity(t *testing.T) {
	device := createTestDevice()
	device.SNMPConfig.Community = "test-community"

	agent := NewAgent(device, 0)

	community := agent.GetCommunity()
	if community != "test-community" {
		t.Errorf("Expected community 'test-community', got '%s'", community)
	}
}

// TestAgent_GetCommunity_Default tests default community string
func TestAgent_GetCommunity_Default(t *testing.T) {
	device := createTestDevice()
	device.SNMPConfig.Community = ""

	agent := NewAgent(device, 0)

	community := agent.GetCommunity()
	if community != "public" {
		t.Errorf("Expected default community 'public', got '%s'", community)
	}
}

// TestAgent_LoadWalkFile tests loading SNMP walk file
func TestAgent_LoadWalkFile(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	// Create temporary walk file
	tmpDir := t.TempDir()
	walkFile := tmpDir + "/test.walk"

	walkContent := `.1.3.6.1.4.1.9999.1.1.0 = STRING: "Test Value"
.1.3.6.1.4.1.9999.1.2.0 = INTEGER: 42
.1.3.6.1.4.1.9999.1.3.0 = Counter32: 12345
`

	if err := os.WriteFile(walkFile, []byte(walkContent), 0644); err != nil {
		t.Fatalf("Failed to create walk file: %v", err)
	}

	// Load walk file
	err := agent.LoadWalkFile(walkFile)
	if err != nil {
		t.Fatalf("LoadWalkFile failed: %v", err)
	}

	// Verify entries were loaded
	result, err := agent.HandleGet("1.3.6.1.4.1.9999.1.1.0")
	if err != nil {
		t.Fatalf("HandleGet failed: %v", err)
	}

	if result.Value.(string) != "Test Value" {
		t.Errorf("Expected 'Test Value', got '%s'", result.Value)
	}
}

// TestAgent_LoadWalkFile_EmptyPath tests loading with empty path
func TestAgent_LoadWalkFile_EmptyPath(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	err := agent.LoadWalkFile("")
	if err == nil {
		t.Error("Expected error for empty path")
	}
}

// TestAgent_LoadWalkFile_NonExistent tests loading non-existent file
func TestAgent_LoadWalkFile_NonExistent(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	err := agent.LoadWalkFile("/nonexistent/file.walk")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

// TestAgent_ProcessPDU_GetRequest tests ProcessPDU with GET request
func TestAgent_ProcessPDU_GetRequest(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	vars := []gosnmp.SnmpPDU{
		{Name: "1.3.6.1.2.1.1.1.0", Type: gosnmp.Null},
		{Name: "1.3.6.1.2.1.1.5.0", Type: gosnmp.Null},
	}

	response := agent.ProcessPDU(gosnmp.GetRequest, vars, 0)

	if len(response) != len(vars) {
		t.Errorf("Expected %d responses, got %d", len(vars), len(response))
	}

	for i, pdu := range response {
		if pdu.Type == gosnmp.NoSuchObject {
			t.Errorf("Response %d has NoSuchObject", i)
		}
	}
}

// TestAgent_ProcessPDU_GetNextRequest tests ProcessPDU with GET-NEXT request
func TestAgent_ProcessPDU_GetNextRequest(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	vars := []gosnmp.SnmpPDU{
		{Name: "1.3.6.1.2.1.1", Type: gosnmp.Null},
	}

	response := agent.ProcessPDU(gosnmp.GetNextRequest, vars, 0)

	if len(response) == 0 {
		t.Error("Expected non-empty response")
	}

	// Response OID should be different from request
	if response[0].Name == vars[0].Name {
		t.Error("GET-NEXT should return different OID")
	}
}

// TestAgent_ProcessPDU_GetBulkRequest tests ProcessPDU with GET-BULK request
func TestAgent_ProcessPDU_GetBulkRequest(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	vars := []gosnmp.SnmpPDU{
		{Name: "1.3.6.1.2.1.1", Type: gosnmp.Null},
	}

	response := agent.ProcessPDU(gosnmp.GetBulkRequest, vars, 5)

	if len(response) == 0 {
		t.Error("Expected non-empty response")
	}
}

// TestAgent_ProcessPDU_InvalidRequest tests ProcessPDU with invalid request type
func TestAgent_ProcessPDU_InvalidRequest(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	vars := []gosnmp.SnmpPDU{
		{Name: "1.3.6.1.2.1.1.1.0", Type: gosnmp.Null},
	}

	response := agent.ProcessPDU(gosnmp.PDUType(99), vars, 0)

	if len(response) == 0 {
		t.Error("Expected error response")
	}

	if response[0].Type != gosnmp.NoSuchObject {
		t.Errorf("Expected NoSuchObject, got %v", response[0].Type)
	}
}

// TestAgent_ProcessPDU_NoSuchObject tests ProcessPDU with non-existent OID
func TestAgent_ProcessPDU_NoSuchObject(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	vars := []gosnmp.SnmpPDU{
		{Name: "1.2.3.4.5.6.7.8.9", Type: gosnmp.Null},
	}

	response := agent.ProcessPDU(gosnmp.GetRequest, vars, 0)

	if len(response) != 1 {
		t.Errorf("Expected 1 response, got %d", len(response))
	}

	if response[0].Type != gosnmp.NoSuchObject {
		t.Errorf("Expected NoSuchObject, got %v", response[0].Type)
	}
}

// TestParseOID tests OID parsing
func TestParseOID(t *testing.T) {
	tests := []struct {
		input       string
		shouldError bool
	}{
		{"1.3.6.1.2.1.1.1.0", false},
		{".1.3.6.1.2.1.1.1.0", false},
		{"", true},
		{"1.2.a.4", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseOID(tt.input)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(result) == 0 {
					t.Error("Expected non-empty result")
				}
			}
		})
	}
}

// TestFormatIP tests IP address formatting
func TestFormatIP(t *testing.T) {
	tests := []struct {
		ip       net.IP
		expected string
	}{
		{net.ParseIP("192.168.1.1"), "192.168.1.1"},
		{net.ParseIP("10.0.0.1"), "10.0.0.1"},
		{net.ParseIP("::1"), "::1"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatIP(tt.ip)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestAgent_DebugLevels tests agent with different debug levels
func TestAgent_DebugLevels(t *testing.T) {
	device := createTestDevice()

	for debugLevel := 0; debugLevel <= 3; debugLevel++ {
		t.Run(fmt.Sprintf("level_%d", debugLevel), func(t *testing.T) {
			agent := NewAgent(device, debugLevel)

			// Operations should work at all debug levels
			_, err := agent.HandleGet("1.3.6.1.2.1.1.1.0")
			if err != nil {
				t.Errorf("HandleGet failed at debug level %d: %v", debugLevel, err)
			}
		})
	}
}

// TestAgent_ConcurrentReadWrite tests concurrent read/write operations
func TestAgent_ConcurrentReadWrite(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	done := make(chan bool, 200)

	// Writers
	for i := 0; i < 100; i++ {
		go func(id int) {
			oid := fmt.Sprintf("1.3.6.1.4.1.9999.%d.0", id)
			_ = agent.SetOID(oid, &OIDValue{
				Type:  gosnmp.Integer,
				Value: id,
			})
			done <- true
		}(i)
	}

	// Readers
	for i := 0; i < 100; i++ {
		go func() {
			_, _ = agent.HandleGet("1.3.6.1.2.1.1.3.0")
			done <- true
		}()
	}

	// Wait for completion
	for i := 0; i < 200; i++ {
		<-done
	}
}

// TestAgent_LargeWalkFile tests loading large walk file
func TestAgent_LargeWalkFile(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	tmpDir := t.TempDir()
	walkFile := tmpDir + "/large.walk"

	// Generate large walk file
	var content strings.Builder
	for i := 1; i <= 1000; i++ {
		content.WriteString(fmt.Sprintf(".1.3.6.1.4.1.9999.%d.0 = INTEGER: %d\n", i, i))
	}

	if err := os.WriteFile(walkFile, []byte(content.String()), 0644); err != nil {
		t.Fatalf("Failed to create walk file: %v", err)
	}

	err := agent.LoadWalkFile(walkFile)
	if err != nil {
		t.Fatalf("LoadWalkFile failed: %v", err)
	}

	// Verify some entries
	result, err := agent.HandleGet("1.3.6.1.4.1.9999.500.0")
	if err != nil {
		t.Fatalf("HandleGet failed: %v", err)
	}

	if result.Value.(int) != 500 {
		t.Errorf("Expected value 500, got %v", result.Value)
	}
}

// TestAgent_OIDTreeNavigation tests navigating OID tree structure
func TestAgent_OIDTreeNavigation(t *testing.T) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	// Add custom OIDs to create a tree structure
	oids := []string{
		"1.3.6.1.4.1.9999.1.1.0",
		"1.3.6.1.4.1.9999.1.2.0",
		"1.3.6.1.4.1.9999.2.1.0",
		"1.3.6.1.4.1.9999.2.2.0",
	}

	for i, oid := range oids {
		agent.SetOID(oid, &OIDValue{
			Type:  gosnmp.Integer,
			Value: i,
		})
	}

	// Navigate tree using GET-NEXT
	currentOID := "1.3.6.1.4.1.9999"
	visited := []string{}

	for i := 0; i < 10; i++ {
		nextOID, value, err := agent.HandleGetNext(currentOID)
		if err != nil || nextOID == "" || value == nil {
			break
		}
		visited = append(visited, nextOID)
		currentOID = nextOID
	}

	if len(visited) < len(oids) {
		t.Errorf("Expected to visit %d OIDs, visited %d", len(oids), len(visited))
	}
}

// TestAgent_EmptyCommunity tests agent with no community string set
func TestAgent_EmptyCommunity(t *testing.T) {
	device := createTestDevice()
	device.SNMPConfig.Community = ""

	agent := NewAgent(device, 0)

	// Should default to "public"
	community := agent.GetCommunity()
	if community != "public" {
		t.Errorf("Expected default community 'public', got '%s'", community)
	}
}

// BenchmarkAgent_HandleGet benchmarks HandleGet operation
func BenchmarkAgent_HandleGet(b *testing.B) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = agent.HandleGet("1.3.6.1.2.1.1.1.0")
	}
}

// BenchmarkAgent_HandleGetNext benchmarks HandleGetNext operation
func BenchmarkAgent_HandleGetNext(b *testing.B) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = agent.HandleGetNext("1.3.6.1.2.1.1")
	}
}

// BenchmarkAgent_HandleGetBulk benchmarks HandleGetBulk operation
func BenchmarkAgent_HandleGetBulk(b *testing.B) {
	device := createTestDevice()
	agent := NewAgent(device, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = agent.HandleGetBulk("1.3.6.1.2.1.1", 10)
	}
}

// BenchmarkAgent_SetOID benchmarks SetOID operation
func BenchmarkAgent_SetOID(b *testing.B) {
	device := createTestDevice()
	agent := NewAgent(device, 0)
	value := &OIDValue{Type: gosnmp.Integer, Value: 42}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = agent.SetOID("1.3.6.1.4.1.9999.1.0", value)
	}
}

// BenchmarkAgent_LoadWalkFile benchmarks walk file loading
func BenchmarkAgent_LoadWalkFile(b *testing.B) {
	device := createTestDevice()

	tmpDir := b.TempDir()
	walkFile := tmpDir + "/bench.walk"

	content := `.1.3.6.1.4.1.9999.1.0 = INTEGER: 42
.1.3.6.1.4.1.9999.2.0 = STRING: "test"
.1.3.6.1.4.1.9999.3.0 = Counter32: 12345
`

	if err := os.WriteFile(walkFile, []byte(content), 0644); err != nil {
		b.Fatalf("Failed to create walk file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agent := NewAgent(device, 0)
		_ = agent.LoadWalkFile(walkFile)
	}
}
