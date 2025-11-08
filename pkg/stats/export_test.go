package stats

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewStatistics(t *testing.T) {
	stats := NewStatistics("en0", "/path/to/config.yaml", "v1.19.0")

	if stats.Interface != "en0" {
		t.Errorf("Expected interface 'en0', got '%s'", stats.Interface)
	}
	if stats.ConfigFile != "/path/to/config.yaml" {
		t.Errorf("Expected config file '/path/to/config.yaml', got '%s'", stats.ConfigFile)
	}
	if stats.Version != "v1.19.0" {
		t.Errorf("Expected version 'v1.19.0', got '%s'", stats.Version)
	}
	if stats.PacketCounts == nil {
		t.Error("PacketCounts map should be initialized")
	}
	if stats.ErrorCounts == nil {
		t.Error("ErrorCounts map should be initialized")
	}
}

func TestIncrementPacketCount(t *testing.T) {
	stats := NewStatistics("en0", "config.yaml", "v1.19.0")

	stats.IncrementPacketCount("ARP")
	stats.IncrementPacketCount("ARP")
	stats.IncrementPacketCount("ICMP")

	if stats.PacketCounts["ARP"] != 2 {
		t.Errorf("Expected ARP count 2, got %d", stats.PacketCounts["ARP"])
	}
	if stats.PacketCounts["ICMP"] != 1 {
		t.Errorf("Expected ICMP count 1, got %d", stats.PacketCounts["ICMP"])
	}
}

func TestIncrementErrorCount(t *testing.T) {
	stats := NewStatistics("en0", "config.yaml", "v1.19.0")

	stats.IncrementErrorCount("router-1")
	stats.IncrementErrorCount("router-1")
	stats.IncrementErrorCount("switch-1")

	if stats.ErrorCounts["router-1"] != 2 {
		t.Errorf("Expected router-1 error count 2, got %d", stats.ErrorCounts["router-1"])
	}
	if stats.ErrorCounts["switch-1"] != 1 {
		t.Errorf("Expected switch-1 error count 1, got %d", stats.ErrorCounts["switch-1"])
	}
}

func TestUpdate(t *testing.T) {
	stats := NewStatistics("en0", "config.yaml", "v1.19.0")

	// Sleep briefly to ensure uptime changes
	time.Sleep(10 * time.Millisecond)

	stats.Update()

	if stats.Uptime == 0 {
		t.Error("Uptime should be greater than 0 after Update()")
	}
	if stats.GoroutineCount == 0 {
		t.Error("GoroutineCount should be greater than 0")
	}
	if stats.CPUCount == 0 {
		t.Error("CPUCount should be greater than 0")
	}
}

func TestIncrementSNMPQuery(t *testing.T) {
	stats := NewStatistics("en0", "config.yaml", "v1.19.0")

	stats.IncrementSNMPQuery()
	stats.IncrementSNMPQuery()
	stats.IncrementSNMPQuery()

	if stats.SNMPQueryCount != 3 {
		t.Errorf("Expected SNMP query count 3, got %d", stats.SNMPQueryCount)
	}
}

func TestIncrementSNMPTrap(t *testing.T) {
	stats := NewStatistics("en0", "config.yaml", "v1.19.0")

	stats.IncrementSNMPTrap()
	stats.IncrementSNMPTrap()

	if stats.SNMPTrapsSent != 2 {
		t.Errorf("Expected SNMP traps sent 2, got %d", stats.SNMPTrapsSent)
	}
}

func TestIncrementDHCPRequest(t *testing.T) {
	stats := NewStatistics("en0", "config.yaml", "v1.19.0")

	stats.IncrementDHCPRequest()
	stats.IncrementDHCPRequest()
	stats.IncrementDHCPRequest()

	if stats.DHCPRequestCount != 3 {
		t.Errorf("Expected DHCP request count 3, got %d", stats.DHCPRequestCount)
	}
}

func TestUpdateProtocolStat(t *testing.T) {
	stats := NewStatistics("en0", "config.yaml", "v1.19.0")

	stats.UpdateProtocolStat("DNS", 5, 4, 1, 1024)
	stats.UpdateProtocolStat("DNS", 3, 3, 0, 512)

	dnsStat := stats.ProtocolStats["DNS"]
	if dnsStat.RequestsReceived != 8 {
		t.Errorf("Expected DNS requests 8, got %d", dnsStat.RequestsReceived)
	}
	if dnsStat.ResponsesSent != 7 {
		t.Errorf("Expected DNS responses 7, got %d", dnsStat.ResponsesSent)
	}
	if dnsStat.ErrorsEncountered != 1 {
		t.Errorf("Expected DNS errors 1, got %d", dnsStat.ErrorsEncountered)
	}
	if dnsStat.BytesProcessed != 1536 {
		t.Errorf("Expected DNS bytes 1536, got %d", dnsStat.BytesProcessed)
	}
}

func TestSetters(t *testing.T) {
	stats := NewStatistics("en0", "config.yaml", "v1.19.0")

	stats.SetDeviceCount(10)
	stats.SetSNMPDeviceCount(5)
	stats.SetDHCPLeaseCount(20)

	if stats.DeviceCount != 10 {
		t.Errorf("Expected device count 10, got %d", stats.DeviceCount)
	}
	if stats.SNMPDeviceCount != 5 {
		t.Errorf("Expected SNMP device count 5, got %d", stats.SNMPDeviceCount)
	}
	if stats.DHCPLeaseCount != 20 {
		t.Errorf("Expected DHCP lease count 20, got %d", stats.DHCPLeaseCount)
	}
}

func TestExportJSON(t *testing.T) {
	stats := NewStatistics("en0", "config.yaml", "v1.19.0")
	stats.SetDeviceCount(5)
	stats.IncrementPacketCount("ARP")
	stats.IncrementPacketCount("ARP")
	stats.IncrementSNMPQuery()
	stats.Update()

	// Create temp directory for test
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "stats.json")

	// Export to JSON
	if err := stats.ExportJSON(jsonFile); err != nil {
		t.Fatalf("Failed to export JSON: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(jsonFile); os.IsNotExist(err) {
		t.Fatal("JSON file was not created")
	}

	// Read and parse JSON
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON file: %v", err)
	}

	var loaded Statistics
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify contents
	if loaded.Interface != "en0" {
		t.Errorf("Expected interface 'en0', got '%s'", loaded.Interface)
	}
	if loaded.DeviceCount != 5 {
		t.Errorf("Expected device count 5, got %d", loaded.DeviceCount)
	}
	if loaded.PacketCounts["ARP"] != 2 {
		t.Errorf("Expected ARP count 2, got %d", loaded.PacketCounts["ARP"])
	}
	if loaded.SNMPQueryCount != 1 {
		t.Errorf("Expected SNMP query count 1, got %d", loaded.SNMPQueryCount)
	}
}

func TestExportCSV(t *testing.T) {
	stats := NewStatistics("en0", "config.yaml", "v1.19.0")
	stats.SetDeviceCount(3)
	stats.IncrementPacketCount("LLDP")
	stats.IncrementPacketCount("CDP")
	stats.IncrementErrorCount("router-1")
	stats.IncrementDHCPRequest()
	stats.UpdateProtocolStat("DNS", 10, 9, 1, 2048)
	stats.Update()

	// Create temp directory for test
	tmpDir := t.TempDir()
	csvFile := filepath.Join(tmpDir, "stats.csv")

	// Export to CSV
	if err := stats.ExportCSV(csvFile); err != nil {
		t.Fatalf("Failed to export CSV: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(csvFile); os.IsNotExist(err) {
		t.Fatal("CSV file was not created")
	}

	// Read and parse CSV
	file, err := os.Open(csvFile)
	if err != nil {
		t.Fatalf("Failed to open CSV file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	// Verify header
	if len(records) < 2 {
		t.Fatal("CSV should have at least header and one row")
	}
	header := records[0]
	if len(header) != 3 || header[0] != "Metric" || header[1] != "Value" || header[2] != "Category" {
		t.Errorf("Invalid CSV header: %v", header)
	}

	// Verify some content exists
	foundDeviceCount := false
	foundInterface := false
	for _, record := range records[1:] {
		if len(record) != 3 {
			continue
		}
		if record[0] == "Device Count" && record[1] == "3" {
			foundDeviceCount = true
		}
		if record[0] == "Interface" && record[1] == "en0" {
			foundInterface = true
		}
	}

	if !foundDeviceCount {
		t.Error("CSV should contain Device Count = 3")
	}
	if !foundInterface {
		t.Error("CSV should contain Interface = en0")
	}
}

func TestGetSnapshot(t *testing.T) {
	stats := NewStatistics("en0", "config.yaml", "v1.19.0")
	stats.SetDeviceCount(5)
	stats.IncrementPacketCount("ARP")

	snapshot := stats.GetSnapshot()

	// Verify snapshot is independent
	stats.SetDeviceCount(10)
	stats.IncrementPacketCount("ARP")

	if snapshot.DeviceCount != 5 {
		t.Errorf("Snapshot device count should be 5, got %d", snapshot.DeviceCount)
	}
	if snapshot.PacketCounts["ARP"] != 1 {
		t.Errorf("Snapshot ARP count should be 1, got %d", snapshot.PacketCounts["ARP"])
	}
}

func TestString(t *testing.T) {
	stats := NewStatistics("en0", "config.yaml", "v1.19.0")
	stats.SetDeviceCount(5)
	stats.Update()

	str := stats.String()
	if str == "" {
		t.Error("String() should return non-empty string")
	}
	// Just verify it doesn't panic and returns something
}

func TestConcurrentAccess(t *testing.T) {
	stats := NewStatistics("en0", "config.yaml", "v1.19.0")

	// Test concurrent reads and writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				stats.IncrementPacketCount("ARP")
				stats.IncrementSNMPQuery()
				stats.Update()
				_ = stats.GetSnapshot()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify counts (should be 1000 total)
	if stats.PacketCounts["ARP"] != 1000 {
		t.Errorf("Expected ARP count 1000, got %d", stats.PacketCounts["ARP"])
	}
	if stats.SNMPQueryCount != 1000 {
		t.Errorf("Expected SNMP query count 1000, got %d", stats.SNMPQueryCount)
	}
}
