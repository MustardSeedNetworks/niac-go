// Package stats provides runtime statistics collection and export functionality
package stats

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"
)

// Statistics holds all runtime statistics for NIAC
type Statistics struct {
	mu sync.RWMutex

	// General stats
	StartTime   time.Time     `json:"start_time"`
	Uptime      time.Duration `json:"uptime_seconds"`
	Interface   string        `json:"interface"`
	ConfigFile  string        `json:"config_file"`
	DeviceCount int           `json:"device_count"`
	Version     string        `json:"version"`

	// Packet counters (per protocol)
	PacketCounts map[string]int64 `json:"packet_counts"`

	// Error counters (per device)
	ErrorCounts map[string]int64 `json:"error_counts"`

	// SNMP stats
	SNMPQueryCount  int64 `json:"snmp_query_count"`
	SNMPDeviceCount int   `json:"snmp_device_count"`
	SNMPTrapsSent   int64 `json:"snmp_traps_sent"`

	// DHCP stats
	DHCPLeaseCount   int   `json:"dhcp_lease_count"`
	DHCPRequestCount int64 `json:"dhcp_request_count"`

	// System stats
	MemoryUsageMB  uint64 `json:"memory_usage_mb"`
	GoroutineCount int    `json:"goroutine_count"`
	CPUCount       int    `json:"cpu_count"`

	// Protocol-specific stats
	ProtocolStats map[string]ProtocolStat `json:"protocol_stats"`
}

// ProtocolStat holds statistics for a specific protocol
type ProtocolStat struct {
	RequestsReceived  int64 `json:"requests_received"`
	ResponsesSent     int64 `json:"responses_sent"`
	ErrorsEncountered int64 `json:"errors_encountered"`
	BytesProcessed    int64 `json:"bytes_processed"`
}

// StatisticsSnapshot is a mutex-free copy of Statistics for export
type StatisticsSnapshot struct {
	// General stats
	StartTime   time.Time     `json:"start_time"`
	Uptime      time.Duration `json:"uptime_seconds"`
	Interface   string        `json:"interface"`
	ConfigFile  string        `json:"config_file"`
	DeviceCount int           `json:"device_count"`
	Version     string        `json:"version"`

	// Packet counters (per protocol)
	PacketCounts map[string]int64 `json:"packet_counts"`

	// Error counters (per device)
	ErrorCounts map[string]int64 `json:"error_counts"`

	// SNMP stats
	SNMPQueryCount  int64 `json:"snmp_query_count"`
	SNMPDeviceCount int   `json:"snmp_device_count"`
	SNMPTrapsSent   int64 `json:"snmp_traps_sent"`

	// DHCP stats
	DHCPLeaseCount   int   `json:"dhcp_lease_count"`
	DHCPRequestCount int64 `json:"dhcp_request_count"`

	// System stats
	MemoryUsageMB  uint64 `json:"memory_usage_mb"`
	GoroutineCount int    `json:"goroutine_count"`
	CPUCount       int    `json:"cpu_count"`

	// Protocol-specific stats
	ProtocolStats map[string]ProtocolStat `json:"protocol_stats"`
}

// NewStatistics creates a new Statistics instance
func NewStatistics(interfaceName, configFile, version string) *Statistics {
	return &Statistics{
		StartTime:     time.Now(),
		Interface:     interfaceName,
		ConfigFile:    configFile,
		Version:       version,
		PacketCounts:  make(map[string]int64),
		ErrorCounts:   make(map[string]int64),
		ProtocolStats: make(map[string]ProtocolStat),
	}
}

// Update refreshes runtime statistics (should be called periodically)
func (s *Statistics) Update() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Uptime = time.Since(s.StartTime)
	s.GoroutineCount = runtime.NumGoroutine()
	s.CPUCount = runtime.NumCPU()

	// Get memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	s.MemoryUsageMB = m.Alloc / 1024 / 1024
}

// IncrementPacketCount increments the packet count for a protocol
func (s *Statistics) IncrementPacketCount(protocol string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PacketCounts[protocol]++
}

// IncrementErrorCount increments the error count for a device
func (s *Statistics) IncrementErrorCount(deviceName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ErrorCounts[deviceName]++
}

// IncrementSNMPQuery increments SNMP query counter
func (s *Statistics) IncrementSNMPQuery() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SNMPQueryCount++
}

// IncrementSNMPTrap increments SNMP trap counter
func (s *Statistics) IncrementSNMPTrap() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SNMPTrapsSent++
}

// IncrementDHCPRequest increments DHCP request counter
func (s *Statistics) IncrementDHCPRequest() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DHCPRequestCount++
}

// UpdateProtocolStat updates statistics for a specific protocol
func (s *Statistics) UpdateProtocolStat(protocol string, requests, responses, errors, bytes int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stat := s.ProtocolStats[protocol]
	stat.RequestsReceived += requests
	stat.ResponsesSent += responses
	stat.ErrorsEncountered += errors
	stat.BytesProcessed += bytes
	s.ProtocolStats[protocol] = stat
}

// SetDeviceCount sets the total device count
func (s *Statistics) SetDeviceCount(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DeviceCount = count
}

// SetSNMPDeviceCount sets the SNMP-enabled device count
func (s *Statistics) SetSNMPDeviceCount(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SNMPDeviceCount = count
}

// SetDHCPLeaseCount sets the DHCP lease count
func (s *Statistics) SetDHCPLeaseCount(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DHCPLeaseCount = count
}

// ExportJSON exports statistics to a JSON file
func (s *Statistics) ExportJSON(filename string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a snapshot for export
	snapshot := s.snapshot()

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal statistics to JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	return nil
}

// ExportCSV exports statistics to a CSV file
func (s *Statistics) ExportCSV(filename string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create CSV file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Metric", "Value", "Category"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Helper function to write rows
	writeRow := func(metric, value, category string) error {
		return writer.Write([]string{metric, value, category})
	}

	// General stats
	writeRow("Start Time", s.StartTime.Format(time.RFC3339), "General")
	writeRow("Uptime (seconds)", fmt.Sprintf("%.0f", s.Uptime.Seconds()), "General")
	writeRow("Interface", s.Interface, "General")
	writeRow("Config File", s.ConfigFile, "General")
	writeRow("Device Count", fmt.Sprintf("%d", s.DeviceCount), "General")
	writeRow("Version", s.Version, "General")

	// System stats
	writeRow("Memory Usage (MB)", fmt.Sprintf("%d", s.MemoryUsageMB), "System")
	writeRow("Goroutine Count", fmt.Sprintf("%d", s.GoroutineCount), "System")
	writeRow("CPU Count", fmt.Sprintf("%d", s.CPUCount), "System")

	// SNMP stats
	writeRow("SNMP Query Count", fmt.Sprintf("%d", s.SNMPQueryCount), "SNMP")
	writeRow("SNMP Device Count", fmt.Sprintf("%d", s.SNMPDeviceCount), "SNMP")
	writeRow("SNMP Traps Sent", fmt.Sprintf("%d", s.SNMPTrapsSent), "SNMP")

	// DHCP stats
	writeRow("DHCP Lease Count", fmt.Sprintf("%d", s.DHCPLeaseCount), "DHCP")
	writeRow("DHCP Request Count", fmt.Sprintf("%d", s.DHCPRequestCount), "DHCP")

	// Packet counts
	for protocol, count := range s.PacketCounts {
		writeRow(fmt.Sprintf("Packet Count (%s)", protocol), fmt.Sprintf("%d", count), "Packets")
	}

	// Error counts
	for device, count := range s.ErrorCounts {
		writeRow(fmt.Sprintf("Error Count (%s)", device), fmt.Sprintf("%d", count), "Errors")
	}

	// Protocol stats
	for protocol, stat := range s.ProtocolStats {
		writeRow(fmt.Sprintf("%s - Requests Received", protocol), fmt.Sprintf("%d", stat.RequestsReceived), "Protocol")
		writeRow(fmt.Sprintf("%s - Responses Sent", protocol), fmt.Sprintf("%d", stat.ResponsesSent), "Protocol")
		writeRow(fmt.Sprintf("%s - Errors", protocol), fmt.Sprintf("%d", stat.ErrorsEncountered), "Protocol")
		writeRow(fmt.Sprintf("%s - Bytes Processed", protocol), fmt.Sprintf("%d", stat.BytesProcessed), "Protocol")
	}

	return nil
}

// snapshot creates a read-safe copy of statistics
// Must be called with read lock held
func (s *Statistics) snapshot() StatisticsSnapshot {
	snapshot := StatisticsSnapshot{
		StartTime:        s.StartTime,
		Uptime:           s.Uptime,
		Interface:        s.Interface,
		ConfigFile:       s.ConfigFile,
		DeviceCount:      s.DeviceCount,
		Version:          s.Version,
		SNMPQueryCount:   s.SNMPQueryCount,
		SNMPDeviceCount:  s.SNMPDeviceCount,
		SNMPTrapsSent:    s.SNMPTrapsSent,
		DHCPLeaseCount:   s.DHCPLeaseCount,
		DHCPRequestCount: s.DHCPRequestCount,
		MemoryUsageMB:    s.MemoryUsageMB,
		GoroutineCount:   s.GoroutineCount,
		CPUCount:         s.CPUCount,
		PacketCounts:     make(map[string]int64),
		ErrorCounts:      make(map[string]int64),
		ProtocolStats:    make(map[string]ProtocolStat),
	}

	// Deep copy maps
	for k, v := range s.PacketCounts {
		snapshot.PacketCounts[k] = v
	}
	for k, v := range s.ErrorCounts {
		snapshot.ErrorCounts[k] = v
	}
	for k, v := range s.ProtocolStats {
		snapshot.ProtocolStats[k] = v
	}

	return snapshot
}

// GetSnapshot returns a thread-safe snapshot of current statistics
func (s *Statistics) GetSnapshot() StatisticsSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshot()
}

// String returns a human-readable summary of statistics
func (s *Statistics) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return fmt.Sprintf(
		"Statistics Summary:\n"+
			"  Uptime: %s\n"+
			"  Devices: %d\n"+
			"  Memory: %d MB\n"+
			"  Goroutines: %d\n"+
			"  SNMP Queries: %d\n"+
			"  DHCP Requests: %d\n",
		s.Uptime.Round(time.Second),
		s.DeviceCount,
		s.MemoryUsageMB,
		s.GoroutineCount,
		s.SNMPQueryCount,
		s.DHCPRequestCount,
	)
}
