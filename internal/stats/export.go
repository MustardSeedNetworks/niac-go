// Package stats provides runtime statistics collection and export functionality
package stats

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// Byte conversion constants.
const bytesPerKB = 1024

// Statistics holds all runtime statistics for NIAC
// PERFORMANCE FIX MEDIUM-1: Use atomic operations and [sync.Map] for high-throughput counters.
type Statistics struct {
	mu sync.RWMutex

	// General stats (read/written infrequently, kept under mutex)
	StartTime   time.Time     `json:"start_time"`
	Uptime      time.Duration `json:"uptime_seconds"`
	Interface   string        `json:"interface"`
	ConfigFile  string        `json:"config_file"`
	DeviceCount int           `json:"device_count"`
	Version     string        `json:"version"`

	// Packet counters (per protocol) - lock-free for high performance
	// Uses sync.Map with *atomic.Int64 values
	packetCounts sync.Map // map[string]*atomic.Int64

	// Error counters (per device) - lock-free for high performance
	errorCounts sync.Map // map[string]*atomic.Int64

	// SNMP stats - atomic for lock-free updates
	SNMPQueryCount  atomic.Int64 `json:"snmp_query_count"`
	SNMPDeviceCount int          `json:"snmp_device_count"`
	SNMPTrapsSent   atomic.Int64 `json:"snmp_traps_sent"`

	// DHCP stats - atomic for lock-free updates
	DHCPLeaseCount   int          `json:"dhcp_lease_count"`
	DHCPRequestCount atomic.Int64 `json:"dhcp_request_count"`

	// System stats
	MemoryUsageMB  uint64 `json:"memory_usage_mb"`
	GoroutineCount int    `json:"goroutine_count"`
	CPUCount       int    `json:"cpu_count"`

	// Protocol-specific stats
	ProtocolStats map[string]ProtocolStat `json:"protocol_stats"`
}

// ProtocolStat holds statistics for a specific protocol.
type ProtocolStat struct {
	RequestsReceived  int64 `json:"requests_received"`
	ResponsesSent     int64 `json:"responses_sent"`
	ErrorsEncountered int64 `json:"errors_encountered"`
	BytesProcessed    int64 `json:"bytes_processed"`
}

// StatisticsSnapshot is a mutex-free copy of Statistics for export.
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
// PERFORMANCE FIX MEDIUM-1: [sync.Map] doesn't need initialization.
func NewStatistics(interfaceName, configFile, version string) *Statistics {
	return &Statistics{
		StartTime:     time.Now(),
		Interface:     interfaceName,
		ConfigFile:    configFile,
		Version:       version,
		ProtocolStats: make(map[string]ProtocolStat),
		// packetCounts and errorCounts are sync.Map - no initialization needed
		// atomic counters are zero-initialized automatically
	}
}

// Update refreshes runtime statistics (should be called periodically).
func (s *Statistics) Update() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Uptime = time.Since(s.StartTime)
	s.GoroutineCount = runtime.NumGoroutine()
	s.CPUCount = runtime.NumCPU()

	// Get memory stats
	var m runtime.MemStats

	runtime.ReadMemStats(&m)
	s.MemoryUsageMB = m.Alloc / bytesPerKB / bytesPerKB
}

// IncrementPacketCount increments the packet count for a protocol
// PERFORMANCE FIX MEDIUM-1: Lock-free atomic increment (10-20% CPU reduction at high load).
func (s *Statistics) IncrementPacketCount(protocol string) {
	// Load or create atomic counter for this protocol
	val, _ := s.packetCounts.LoadOrStore(protocol, &atomic.Int64{})

	counter, ok := val.(*atomic.Int64)
	if !ok {
		return
	}

	counter.Add(1)
}

// IncrementErrorCount increments the error count for a device
// PERFORMANCE FIX MEDIUM-1: Lock-free atomic increment.
func (s *Statistics) IncrementErrorCount(deviceName string) {
	val, _ := s.errorCounts.LoadOrStore(deviceName, &atomic.Int64{})

	counter, ok := val.(*atomic.Int64)
	if !ok {
		return
	}

	counter.Add(1)
}

// IncrementSNMPQuery increments SNMP query counter
// PERFORMANCE FIX MEDIUM-1: Lock-free atomic increment.
func (s *Statistics) IncrementSNMPQuery() {
	s.SNMPQueryCount.Add(1)
}

// IncrementSNMPTrap increments SNMP trap counter
// PERFORMANCE FIX MEDIUM-1: Lock-free atomic increment.
func (s *Statistics) IncrementSNMPTrap() {
	s.SNMPTrapsSent.Add(1)
}

// IncrementDHCPRequest increments DHCP request counter
// PERFORMANCE FIX MEDIUM-1: Lock-free atomic increment.
func (s *Statistics) IncrementDHCPRequest() {
	s.DHCPRequestCount.Add(1)
}

// UpdateProtocolStat updates statistics for a specific protocol.
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

// SetDeviceCount sets the total device count.
func (s *Statistics) SetDeviceCount(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.DeviceCount = count
}

// SetSNMPDeviceCount sets the SNMP-enabled device count.
func (s *Statistics) SetSNMPDeviceCount(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.SNMPDeviceCount = count
}

// SetDHCPLeaseCount sets the DHCP lease count.
func (s *Statistics) SetDHCPLeaseCount(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.DHCPLeaseCount = count
}

// ExportJSON exports statistics to a JSON file.
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
	if writeErr := os.WriteFile(filename, data, 0o600); writeErr != nil {
		return fmt.Errorf("failed to write JSON file: %w", writeErr)
	}

	return nil
}

// ExportCSV exports statistics to a CSV file.
func (s *Statistics) ExportCSV(filename string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// SECURITY FIX #163: Create CSV file with restricted permissions (owner-only)
	file, err := os.OpenFile(
		filepath.Clean(filename),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0o600,
	)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer func() { _ = file.Close() }()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if writeErr := writer.Write([]string{"Metric", "Value", "Category"}); writeErr != nil {
		return fmt.Errorf("failed to write CSV header: %w", writeErr)
	}

	writeRow := func(metric, value, category string) {
		_ = writer.Write([]string{metric, value, category})
	}

	s.writeGeneralCSV(writeRow)
	s.writeSystemCSV(writeRow)
	s.writeCounterCSV(writeRow)
	s.writeProtocolCSV(writeRow)

	return nil
}

func (s *Statistics) writeGeneralCSV(writeRow func(metric, value, category string)) {
	writeRow("Start Time", s.StartTime.Format(time.RFC3339), "General")
	writeRow("Uptime (seconds)", fmt.Sprintf("%.0f", s.Uptime.Seconds()), "General")
	writeRow("Interface", s.Interface, "General")
	writeRow("Config File", s.ConfigFile, "General")
	writeRow("Device Count", strconv.Itoa(s.DeviceCount), "General")
	writeRow("Version", s.Version, "General")
}

func (s *Statistics) writeSystemCSV(writeRow func(metric, value, category string)) {
	writeRow("Memory Usage (MB)", strconv.FormatUint(s.MemoryUsageMB, 10), "System")
	writeRow("Goroutine Count", strconv.Itoa(s.GoroutineCount), "System")
	writeRow("CPU Count", strconv.Itoa(s.CPUCount), "System")
}

func (s *Statistics) writeCounterCSV(writeRow func(metric, value, category string)) {
	writeRow("SNMP Query Count", strconv.FormatInt(s.SNMPQueryCount.Load(), 10), "SNMP")
	writeRow("SNMP Device Count", strconv.Itoa(s.SNMPDeviceCount), "SNMP")
	writeRow("SNMP Traps Sent", strconv.FormatInt(s.SNMPTrapsSent.Load(), 10), "SNMP")
	writeRow("DHCP Lease Count", strconv.Itoa(s.DHCPLeaseCount), "DHCP")
	writeRow("DHCP Request Count", strconv.FormatInt(s.DHCPRequestCount.Load(), 10), "DHCP")

	s.packetCounts.Range(func(key, value any) bool {
		protocol, ok := key.(string)
		if !ok {
			return true
		}
		counter, ok := value.(*atomic.Int64)
		if !ok {
			return true
		}
		writeRow(fmt.Sprintf("Packet Count (%s)", protocol), strconv.FormatInt(counter.Load(), 10), "Packets")
		return true
	})

	s.errorCounts.Range(func(key, value any) bool {
		device, ok := key.(string)
		if !ok {
			return true
		}
		counter, ok := value.(*atomic.Int64)
		if !ok {
			return true
		}
		writeRow(fmt.Sprintf("Error Count (%s)", device), strconv.FormatInt(counter.Load(), 10), "Errors")
		return true
	})
}

func (s *Statistics) writeProtocolCSV(writeRow func(metric, value, category string)) {
	for protocol, stat := range s.ProtocolStats {
		writeRow(
			protocol+" - Requests Received",
			strconv.FormatInt(stat.RequestsReceived, 10),
			"Protocol",
		)
		writeRow(protocol+" - Responses Sent", strconv.FormatInt(stat.ResponsesSent, 10), "Protocol")
		writeRow(protocol+" - Errors", strconv.FormatInt(stat.ErrorsEncountered, 10), "Protocol")
		writeRow(protocol+" - Bytes Processed", strconv.FormatInt(stat.BytesProcessed, 10), "Protocol")
	}
}

// snapshot creates a read-safe copy of statistics
// PERFORMANCE FIX MEDIUM-1: Read from atomic counters and [sync.Map]
// Must be called with read lock held.
func (s *Statistics) snapshot() StatisticsSnapshot {
	snapshot := StatisticsSnapshot{
		StartTime:        s.StartTime,
		Uptime:           s.Uptime,
		Interface:        s.Interface,
		ConfigFile:       s.ConfigFile,
		DeviceCount:      s.DeviceCount,
		Version:          s.Version,
		SNMPQueryCount:   s.SNMPQueryCount.Load(),
		SNMPDeviceCount:  s.SNMPDeviceCount,
		SNMPTrapsSent:    s.SNMPTrapsSent.Load(),
		DHCPLeaseCount:   s.DHCPLeaseCount,
		DHCPRequestCount: s.DHCPRequestCount.Load(),
		MemoryUsageMB:    s.MemoryUsageMB,
		GoroutineCount:   s.GoroutineCount,
		CPUCount:         s.CPUCount,
		PacketCounts:     copyAtomicInt64Map(&s.packetCounts),
		ErrorCounts:      copyAtomicInt64Map(&s.errorCounts),
		ProtocolStats:    make(map[string]ProtocolStat),
	}

	// Deep copy protocol stats (still under mutex protection)
	maps.Copy(snapshot.ProtocolStats, s.ProtocolStats)

	return snapshot
}

// copyAtomicInt64Map flattens a [sync.Map] keyed by string with [atomic.Int64] pointer values into a plain int64 map.
func copyAtomicInt64Map(src *sync.Map) map[string]int64 {
	out := make(map[string]int64)
	src.Range(func(key, value any) bool {
		k, ok := key.(string)
		if !ok {
			return true
		}
		counter, ok := value.(*atomic.Int64)
		if !ok {
			return true
		}
		out[k] = counter.Load()
		return true
	})
	return out
}

// GetSnapshot returns a thread-safe snapshot of current statistics.
func (s *Statistics) GetSnapshot() StatisticsSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.snapshot()
}

// String returns a human-readable summary of statistics.
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
		s.SNMPQueryCount.Load(),
		s.DHCPRequestCount.Load(),
	)
}
