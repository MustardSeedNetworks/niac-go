package api

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"runtime"
	"time"

	"github.com/krisarmstrong/niac-go/internal/capture"
	"github.com/krisarmstrong/niac-go/internal/protocols"
)

func (s *Server) handleStats(w http.ResponseWriter, _ *http.Request) {
	// SECURITY FIX #161: Thread-safe access to all config fields
	s.configMu.RLock()
	stack := s.cfg.Stack
	cfg := s.cfg.Config
	iface := s.cfg.Interface
	s.configMu.RUnlock()

	if stack == nil {
		http.Error(w, "no simulation running", http.StatusServiceUnavailable)

		return
	}

	stats := stack.GetStats()

	deviceCount := 0
	if cfg != nil {
		deviceCount = len(cfg.Devices)
	}

	// FEATURE #119: Include goroutine count for debugging and monitoring
	goroutineCount := runtime.NumGoroutine()

	payload := map[string]any{
		"timestamp":    time.Now().UTC(),
		"interface":    iface,
		"version":      s.cfg.Version,
		"device_count": deviceCount,
		"goroutines":   goroutineCount, // FEATURE #119: Monitor goroutine count
		"stack": map[string]uint64{
			"packets_sent":     stats.PacketsSent,
			"packets_received": stats.PacketsReceived,
			"arp_requests":     stats.ARPRequests,
			"arp_replies":      stats.ARPReplies,
			"icmp_requests":    stats.ICMPRequests,
			"icmp_replies":     stats.ICMPReplies,
			"dns_queries":      stats.DNSQueries,
			"dhcp_requests":    stats.DHCPRequests,
			"snmp_queries":     stats.SNMPQueries,
			"errors":           stats.Errors,
		},
	}
	s.writeJSON(w, payload)
}

func (s *Server) handleRuntime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	// SECURITY FIX #161: Thread-safe access to all config fields
	s.configMu.RLock()
	stack := s.cfg.Stack
	cfg := s.cfg.Config
	iface := s.cfg.Interface
	cfgPath := s.cfg.ConfigPath
	s.configMu.RUnlock()

	if stack == nil {
		http.Error(w, "no simulation running", http.StatusServiceUnavailable)

		return
	}

	stats := stack.GetStats()

	runtimeInfo := map[string]any{
		"running":          true, // API server is running
		"interface":        iface,
		"config_path":      filepath.Base(cfgPath),
		"version":          s.cfg.Version,
		"device_count":     0,
		"packets_sent":     stats.PacketsSent,
		"packets_received": stats.PacketsReceived,
		"uptime_seconds":   time.Since(s.startTime).Seconds(),
	}

	if cfg != nil {
		runtimeInfo["device_count"] = len(cfg.Devices)
		runtimeInfo["config_name"] = filepath.Base(cfgPath)
	}

	s.writeJSON(w, runtimeInfo)
}

func (s *Server) handleVersion(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, map[string]string{"version": s.cfg.Version})
}

func (s *Server) handleInterfaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	// Get available network interfaces from pcap
	ifaces, err := capture.GetAllInterfaces()
	if err != nil {
		// SECURITY FIX MEDIUM-6: Don't expose internal error details
		s.logger.Error("[API] Failed to list interfaces", "error", err)
		writeError(w, r, http.StatusInternalServerError, "interface_list_failed",
			"Failed to retrieve network interfaces", nil)

		return
	}

	type interfaceInfo struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Addresses   []string `json:"addresses"`
		Current     bool     `json:"current"`
	}

	// SECURITY FIX #161: Thread-safe access to Interface
	currentIface := s.currentInterface()

	result := make([]interfaceInfo, 0, len(ifaces))
	for _, iface := range ifaces {
		addrs := make([]string, 0, len(iface.Addresses))
		for _, addr := range iface.Addresses {
			addrs = append(addrs, addr.IP.String())
		}

		result = append(result, interfaceInfo{
			Name:        iface.Name,
			Description: iface.Description,
			Addresses:   addrs,
			Current:     iface.Name == currentIface,
		})
	}

	s.writeJSON(w, map[string]any{
		"interfaces":        result,
		"current_interface": currentIface,
	})
}

// SECURITY FIX #161: Thread-safe access to Interface to prevent race conditions.
func (s *Server) currentInterface() string {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	return s.cfg.Interface
}

// SECURITY FIX #161: Thread-safe access to Stack to prevent race conditions.
func (s *Server) currentStack() *protocols.Stack {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	return s.cfg.Stack
}

// writePrometheusMetric writes a single Prometheus metric with help text, type, and value.
func writePrometheusMetric(w io.Writer, name, help, metricType string, value any) {
	_, _ = fmt.Fprintf(w, "# HELP %s %s\n", name, help)
	_, _ = fmt.Fprintf(w, "# TYPE %s %s\n", name, metricType)
	_, _ = fmt.Fprintf(w, "%s %v\n", name, value)
}

// writeBasicMetrics writes packet and error metrics.
func writeBasicMetrics(w io.Writer, stats *protocols.Statistics, deviceCount int) {
	writePrometheusMetric(
		w,
		"niac_packets_sent_total",
		"Total packets sent",
		"counter",
		stats.PacketsSent,
	)
	writePrometheusMetric(
		w,
		"niac_packets_received_total",
		"Total packets received",
		"counter",
		stats.PacketsReceived,
	)
	writePrometheusMetric(
		w,
		"niac_snmp_queries_total",
		"Total SNMP queries processed",
		"counter",
		stats.SNMPQueries,
	)
	writePrometheusMetric(w, "niac_errors_total", "Total errors", "counter", stats.Errors)
	writePrometheusMetric(
		w,
		"niac_devices_total",
		"Number of simulated devices",
		"gauge",
		deviceCount,
	)
}

// writeProtocolMetrics writes protocol-specific metrics (ARP, ICMP, DNS, DHCP).
func writeProtocolMetrics(w io.Writer, stats *protocols.Statistics) {
	writePrometheusMetric(
		w,
		"niac_arp_requests_total",
		"Total ARP requests sent",
		"counter",
		stats.ARPRequests,
	)
	writePrometheusMetric(
		w,
		"niac_arp_replies_total",
		"Total ARP replies sent",
		"counter",
		stats.ARPReplies,
	)
	writePrometheusMetric(
		w,
		"niac_icmp_requests_total",
		"Total ICMP requests sent",
		"counter",
		stats.ICMPRequests,
	)
	writePrometheusMetric(
		w,
		"niac_icmp_replies_total",
		"Total ICMP replies sent",
		"counter",
		stats.ICMPReplies,
	)
	writePrometheusMetric(
		w,
		"niac_dns_queries_total",
		"Total DNS queries processed",
		"counter",
		stats.DNSQueries,
	)
	writePrometheusMetric(
		w,
		"niac_dhcp_requests_total",
		"Total DHCP requests processed",
		"counter",
		stats.DHCPRequests,
	)
}

// writeSystemMetrics writes runtime and memory metrics.
func writeSystemMetrics(w io.Writer, startTime time.Time, memStats *runtime.MemStats) {
	writePrometheusMetric(
		w,
		"niac_uptime_seconds",
		"Server uptime in seconds",
		"gauge",
		int64(time.Since(startTime).Seconds()),
	)
	writePrometheusMetric(
		w,
		"niac_goroutines_total",
		"Number of goroutines",
		"gauge",
		runtime.NumGoroutine(),
	)
	writePrometheusMetric(
		w,
		"niac_memory_usage_bytes",
		"Memory usage in bytes",
		"gauge",
		memStats.Alloc,
	)
	writePrometheusMetric(
		w,
		"niac_memory_sys_bytes",
		"Total memory obtained from OS in bytes",
		"gauge",
		memStats.Sys,
	)
	writePrometheusMetric(
		w,
		"niac_gc_runs_total",
		"Total number of GC runs",
		"counter",
		memStats.NumGC,
	)
}

func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	s.configMu.RLock()
	stk := s.cfg.Stack
	cfg := s.cfg.Config
	s.configMu.RUnlock()

	if stk == nil {
		http.Error(w, "no simulation running", http.StatusServiceUnavailable)
		return
	}

	stats := stk.GetStats()

	deviceCount := 0
	if cfg != nil {
		deviceCount = len(cfg.Devices)
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	writeBasicMetrics(w, &stats, deviceCount)
	writeProtocolMetrics(w, &stats)
	writeSystemMetrics(w, s.startTime, &memStats)
}

func (s *Server) handleNeighbors(w http.ResponseWriter, _ *http.Request) {
	// SECURITY FIX #161: Thread-safe access to Stack
	stack := s.currentStack()
	if stack == nil {
		http.Error(w, "no simulation running", http.StatusServiceUnavailable)

		return
	}

	neighbors := stack.GetNeighbors()
	if neighbors == nil {
		neighbors = []protocols.NeighborRecord{}
	}

	s.writeJSON(w, neighbors)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Storage == nil {
		s.writeJSON(w, []any{})

		return
	}

	history, err := s.cfg.Storage.ListRuns(historyListLimit)
	if err != nil {
		// SECURITY FIX MEDIUM-6: Don't expose internal error details
		s.logger.Error("[API] Failed to list run history", "error", err)
		writeError(w, r, http.StatusInternalServerError, "storage_error",
			"Failed to retrieve run history", nil)

		return
	}

	s.writeJSON(w, history)
}

func (s *Server) handleDevices(w http.ResponseWriter, _ *http.Request) {
	cfg := s.currentConfig()
	if cfg == nil {
		s.writeJSON(w, []map[string]any{})

		return
	}

	devices := make([]map[string]any, 0, len(cfg.Devices))
	for i := range cfg.Devices {
		dev := &cfg.Devices[i]
		devices = append(devices, map[string]any{
			"name":      dev.Name,
			"type":      dev.Type,
			"ips":       ipAddressesToStrings(dev.IPAddresses),
			"protocols": getDeviceProtocols(dev),
		})
	}

	s.writeJSON(w, devices)
}
