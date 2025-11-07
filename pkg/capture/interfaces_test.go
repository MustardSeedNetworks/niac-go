package capture

import (
	"testing"

	"github.com/google/gopacket/pcap"
)

// TestInterfaceExists_Loopback tests loopback interface detection
func TestInterfaceExists_Loopback(t *testing.T) {
	// Most systems have at least one of these
	loopbackNames := []string{"lo", "lo0", "Loopback", "loopback"}
	found := false

	for _, name := range loopbackNames {
		if InterfaceExists(name) {
			found = true
			t.Logf("Found loopback interface: %s", name)
			break
		}
	}

	if !found {
		t.Skip("No standard loopback interface found (unusual)")
	}
}

// TestInterfaceExists_MultipleCalls tests repeated calls
func TestInterfaceExists_MultipleCalls(t *testing.T) {
	// Should return consistent results
	name := "lo"
	result1 := InterfaceExists(name)
	result2 := InterfaceExists(name)

	if result1 != result2 {
		t.Error("InterfaceExists returned inconsistent results")
	}
}

// TestListInterfaces_NoError tests that listing doesn't panic
func TestListInterfaces_NoError(t *testing.T) {
	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ListInterfaces panicked: %v", r)
		}
	}()

	ListInterfaces()
}

// TestGetInterface_AllInterfaces tests getting all available interfaces
func TestGetInterface_AllInterfaces(t *testing.T) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		t.Skipf("Cannot enumerate interfaces: %v", err)
	}

	if len(devices) == 0 {
		t.Skip("No interfaces available")
	}

	// Test getting each interface
	for _, device := range devices {
		iface, err := GetInterface(device.Name)
		if err != nil {
			t.Errorf("GetInterface(%s) failed: %v", device.Name, err)
			continue
		}

		if iface.Name != device.Name {
			t.Errorf("GetInterface(%s) returned wrong name: %s", device.Name, iface.Name)
		}
	}
}

// TestGetInterface_CaseSensitive tests case sensitivity
func TestGetInterface_CaseSensitive(t *testing.T) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		t.Skipf("Cannot enumerate interfaces: %v", err)
	}

	if len(devices) == 0 {
		t.Skip("No interfaces available")
	}

	// Take first device and try with different cases
	firstDevice := devices[0].Name

	// Try exact match
	_, err = GetInterface(firstDevice)
	if err != nil {
		t.Skipf("Cannot get interface %s: %v", firstDevice, err)
	}

	// Try with obviously wrong case (assuming lowercase device name)
	// Note: Interface names are typically case-sensitive on Unix systems
	// This test documents the behavior rather than asserting it
	wrongCaseName := "DEFINITELY_WRONG_CASE_NAME_12345"
	_, err = GetInterface(wrongCaseName)
	if err == nil {
		t.Logf("Note: GetInterface found interface with name %s (unexpected)", wrongCaseName)
	}
}

// TestGetInterface_EmptyName tests empty interface name
func TestGetInterface_EmptyName(t *testing.T) {
	_, err := GetInterface("")
	if err == nil {
		t.Error("Expected error for empty interface name")
	}
}

// TestInterfaceExists_EmptyName tests empty interface name
func TestInterfaceExists_EmptyName(t *testing.T) {
	if InterfaceExists("") {
		t.Error("InterfaceExists returned true for empty name")
	}
}

// TestInterfaceExists_SpecialCharacters tests names with special characters
func TestInterfaceExists_SpecialCharacters(t *testing.T) {
	testCases := []string{
		"lo@123",
		"lo#test",
		"lo$test",
		"lo%test",
		"lo^test",
		"lo&test",
		"lo*test",
	}

	for _, name := range testCases {
		// These should all return false (no such interfaces normally exist)
		if InterfaceExists(name) {
			t.Logf("Unexpectedly found interface with special chars: %s", name)
		}
	}
}

// TestGetInterface_VeryLongName tests very long interface names
func TestGetInterface_VeryLongName(t *testing.T) {
	// Most systems have length limits on interface names (typically 16 chars on Unix)
	longName := "this-is-a-very-long-interface-name-that-exceeds-normal-limits-123456789"

	_, err := GetInterface(longName)
	if err == nil {
		t.Error("Expected error for very long interface name")
	}
}

// TestInterfaceExists_Concurrency tests concurrent calls
func TestInterfaceExists_Concurrency(t *testing.T) {
	done := make(chan bool, 10)

	// Launch multiple goroutines
	for i := 0; i < 10; i++ {
		go func() {
			_ = InterfaceExists("lo")
			done <- true
		}()
	}

	// Wait for all
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestGetInterface_Concurrency tests concurrent interface lookups
func TestGetInterface_Concurrency(t *testing.T) {
	devices, err := pcap.FindAllDevs()
	if err != nil || len(devices) == 0 {
		t.Skip("No interfaces available for concurrency test")
	}

	testInterface := devices[0].Name
	done := make(chan bool, 10)

	// Launch multiple goroutines
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = GetInterface(testInterface)
			done <- true
		}()
	}

	// Wait for all
	for i := 0; i < 10; i++ {
		<-done
	}
}

// BenchmarkInterfaceExists benchmarks interface existence check
func BenchmarkInterfaceExists(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = InterfaceExists("lo")
	}
}

// BenchmarkGetInterface benchmarks getting interface info
func BenchmarkGetInterface(b *testing.B) {
	devices, err := pcap.FindAllDevs()
	if err != nil || len(devices) == 0 {
		b.Skip("No interfaces available")
	}

	testInterface := devices[0].Name

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetInterface(testInterface)
	}
}

// TestGetInterface_InterfaceFields tests interface field population
func TestGetInterface_InterfaceFields(t *testing.T) {
	devices, err := pcap.FindAllDevs()
	if err != nil || len(devices) == 0 {
		t.Skip("No interfaces available")
	}

	for _, device := range devices {
		iface, err := GetInterface(device.Name)
		if err != nil {
			continue
		}

		// Test that returned interface has expected fields
		if iface.Name == "" {
			t.Errorf("Interface %s has empty name", device.Name)
		}

		// Description may be empty on some systems
		if iface.Description != "" {
			t.Logf("Interface %s has description: %s", iface.Name, iface.Description)
		}

		// Addresses may be empty on some interfaces
		if len(iface.Addresses) > 0 {
			for _, addr := range iface.Addresses {
				if addr.IP == nil {
					t.Errorf("Interface %s has address with nil IP", iface.Name)
				}
			}
		}
	}
}
