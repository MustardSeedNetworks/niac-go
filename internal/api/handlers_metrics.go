package api

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"github.com/krisarmstrong/niac-go/internal/protocols"
)

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
