package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/ipc"
	"github.com/krisarmstrong/niac-go/pkg/logging"
	"github.com/spf13/cobra"
)

// OutputFormat represents the output format type for monitor command
type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatCSV   OutputFormat = "csv"
)

// monitorStats holds the statistics for a single monitoring sample
type monitorStats struct {
	Time      time.Time `json:"time"`
	PacketsRX uint64    `json:"packets_rx"`
	PacketsTX uint64    `json:"packets_tx"`
	ARP       uint64    `json:"arp"`
	ICMP      uint64    `json:"icmp"`
	DNS       uint64    `json:"dns"`
	DHCP      uint64    `json:"dhcp"`
	SNMP      uint64    `json:"snmp"`
	Errors    uint64    `json:"errors"`
	Uptime    float64   `json:"uptime_seconds"`

	// Rates (packets per second)
	RateRX float64 `json:"rate_rx,omitempty"`
	RateTX float64 `json:"rate_tx,omitempty"`
}

var monitorOpts struct {
	format   string
	interval string
	socket   string
}

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Stream real-time statistics from a running NIAC simulation",
	Long: `Monitor a running NIAC simulation in real-time.

This command connects to a running NIAC instance via IPC socket and
continuously displays statistics, similar to 'top' or 'watch'.

The monitor supports multiple output formats:
  - table: Human-readable table with auto-refresh (default)
  - json:  JSON Lines format (one JSON object per interval)
  - csv:   CSV format with header (suitable for piping)

The table format clears the screen and redraws on each update, while
JSON and CSV formats append new lines for pipe-friendly output.`,
	Example: `  # Monitor with default settings (table format, 1s interval)
  niac monitor

  # Monitor with JSON output for piping to jq
  niac monitor --format json | jq '.packets_rx'

  # Monitor with 2-second interval
  niac monitor --interval 2s

  # Monitor with CSV output, redirect to file
  niac monitor --format csv > stats.csv

  # Use a custom socket path
  niac monitor --socket /tmp/my-niac.sock

  # Monitor with fast 500ms updates
  niac monitor --interval 500ms`,
	RunE: runMonitor,
}

func init() {
	rootCmd.AddCommand(monitorCmd)

	monitorCmd.Flags().StringVar(&monitorOpts.format, "format", "table",
		"Output format: table, json, or csv")
	monitorCmd.Flags().StringVar(&monitorOpts.interval, "interval", "1s",
		"Update interval (e.g., 1s, 500ms, 2s)")
	monitorCmd.Flags().StringVar(&monitorOpts.socket, "socket", "",
		"IPC socket path (default: "+ipc.DefaultSocketPath()+")")
}

func runMonitor(cmd *cobra.Command, args []string) error {
	// Parse interval
	interval, err := time.ParseDuration(monitorOpts.interval)
	if err != nil {
		return fmt.Errorf("invalid interval %q: %w", monitorOpts.interval, err)
	}

	if interval < 100*time.Millisecond {
		return fmt.Errorf("interval must be at least 100ms")
	}

	// Validate format
	format := OutputFormat(strings.ToLower(monitorOpts.format))
	switch format {
	case FormatTable, FormatJSON, FormatCSV:
		// Valid
	default:
		return fmt.Errorf("invalid format %q: must be table, json, or csv", monitorOpts.format)
	}

	// Create IPC client
	socketPath := monitorOpts.socket
	if socketPath == "" {
		socketPath = ipc.GetDefaultSocketPath()
	}

	client := ipc.NewClient(socketPath)

	// Check if server is running
	if !client.IsRunning() {
		return fmt.Errorf("no NIAC simulation running (socket: %s)", socketPath)
	}

	// Set up context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	// Run the appropriate monitor based on format
	switch format {
	case FormatTable:
		return runTableMonitor(ctx, client, interval)
	case FormatJSON:
		return runJSONMonitor(ctx, client, interval)
	case FormatCSV:
		return runCSVMonitor(ctx, client, interval)
	}

	return nil
}

// runTableMonitor displays statistics in a continuously-updated table format
func runTableMonitor(ctx context.Context, client *ipc.Client, interval time.Duration) error {
	var prevStats *monitorStats
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logging.InitColors(true)

	// Initial fetch
	stats, err := fetchStats(client, prevStats, interval)
	if err != nil {
		return fmt.Errorf("failed to fetch initial stats: %w", err)
	}
	printTableHeader()
	printTableRow(stats)
	prevStats = stats

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nMonitoring stopped.")
			return nil
		case <-ticker.C:
			stats, err := fetchStats(client, prevStats, interval)
			if err != nil {
				// Connection lost, but don't exit immediately
				fmt.Printf("\r[%s] Connection lost: %v\n", time.Now().Format("15:04:05"), err)
				continue
			}
			clearScreen()
			printTableHeader()
			printTableRow(stats)
			prevStats = stats
		}
	}
}

// runJSONMonitor outputs statistics as JSON Lines (one JSON object per line)
func runJSONMonitor(ctx context.Context, client *ipc.Client, interval time.Duration) error {
	var prevStats *monitorStats
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial fetch
	stats, err := fetchStats(client, prevStats, interval)
	if err != nil {
		return fmt.Errorf("failed to fetch initial stats: %w", err)
	}
	outputMonitorJSON(stats)
	prevStats = stats

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			stats, err := fetchStats(client, prevStats, interval)
			if err != nil {
				// Output error as JSON
				errObj := map[string]interface{}{
					"error": err.Error(),
					"time":  time.Now().Format(time.RFC3339),
				}
				jsonBytes, _ := json.Marshal(errObj)
				fmt.Println(string(jsonBytes))
				continue
			}
			outputMonitorJSON(stats)
			prevStats = stats
		}
	}
}

// runCSVMonitor outputs statistics as CSV
func runCSVMonitor(ctx context.Context, client *ipc.Client, interval time.Duration) error {
	var prevStats *monitorStats
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Write header
	header := []string{"time", "packets_rx", "packets_tx", "arp", "icmp", "dns", "dhcp", "snmp", "errors", "rate_rx", "rate_tx"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}
	writer.Flush()

	// Initial fetch
	stats, err := fetchStats(client, prevStats, interval)
	if err != nil {
		return fmt.Errorf("failed to fetch initial stats: %w", err)
	}
	outputCSVRow(writer, stats)
	prevStats = stats

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			stats, err := fetchStats(client, prevStats, interval)
			if err != nil {
				// Skip this interval on error
				continue
			}
			outputCSVRow(writer, stats)
			prevStats = stats
		}
	}
}

// fetchStats fetches current statistics from the IPC server and calculates rates
func fetchStats(client *ipc.Client, prev *monitorStats, interval time.Duration) (*monitorStats, error) {
	status, err := client.GetStatus()
	if err != nil {
		return nil, err
	}

	stats := &monitorStats{
		Time:      time.Now(),
		PacketsRX: status.PacketsRX,
		PacketsTX: status.PacketsTX,
		Errors:    uint64(status.ErrorsActive),
		Uptime:    status.Uptime,
	}

	// Note: The IPC StatusData currently only has PacketsRX, PacketsTX, ErrorsActive
	// Protocol-specific stats (ARP, ICMP, DNS, etc.) would need to be added to the IPC protocol
	// For now, we use placeholders that show 0 until the IPC protocol is extended

	// Calculate rates if we have previous stats
	if prev != nil {
		intervalSec := interval.Seconds()
		if intervalSec > 0 {
			stats.RateRX = float64(stats.PacketsRX-prev.PacketsRX) / intervalSec
			stats.RateTX = float64(stats.PacketsTX-prev.PacketsTX) / intervalSec
		}
	}

	return stats, nil
}

// clearScreen clears the terminal screen and moves cursor to top-left
func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

// printTableHeader prints the table header
func printTableHeader() {
	fmt.Println("NIAC Monitor - Press Ctrl+C to stop")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-10s | %10s | %10s | %8s | %8s | %8s | %8s\n",
		"TIME", "RX PKT", "TX PKT", "RX/s", "TX/s", "UPTIME", "ERRORS")
	fmt.Println(strings.Repeat("-", 80))
}

// printTableRow prints a single statistics row
func printTableRow(stats *monitorStats) {
	uptime := formatMonitorDuration(time.Duration(stats.Uptime) * time.Second)

	fmt.Printf("%-10s | %10s | %10s | %8.1f | %8.1f | %8s | %8d\n",
		stats.Time.Format("15:04:05"),
		formatMonitorNumber(stats.PacketsRX),
		formatMonitorNumber(stats.PacketsTX),
		stats.RateRX,
		stats.RateTX,
		uptime,
		stats.Errors,
	)
}

// outputMonitorJSON outputs a single stats object as JSON
func outputMonitorJSON(stats *monitorStats) {
	jsonBytes, err := json.Marshal(stats)
	if err != nil {
		fmt.Fprintf(os.Stderr, "JSON encoding error: %v\n", err)
		return
	}
	fmt.Println(string(jsonBytes))
}

// outputCSVRow writes a single CSV row
func outputCSVRow(writer *csv.Writer, stats *monitorStats) {
	row := []string{
		stats.Time.Format(time.RFC3339),
		fmt.Sprintf("%d", stats.PacketsRX),
		fmt.Sprintf("%d", stats.PacketsTX),
		fmt.Sprintf("%d", stats.ARP),
		fmt.Sprintf("%d", stats.ICMP),
		fmt.Sprintf("%d", stats.DNS),
		fmt.Sprintf("%d", stats.DHCP),
		fmt.Sprintf("%d", stats.SNMP),
		fmt.Sprintf("%d", stats.Errors),
		fmt.Sprintf("%.2f", stats.RateRX),
		fmt.Sprintf("%.2f", stats.RateTX),
	}
	writer.Write(row)
	writer.Flush()
}

// formatMonitorNumber formats a number with thousands separators
func formatMonitorNumber(n uint64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}

	str := fmt.Sprintf("%d", n)
	var result strings.Builder
	length := len(str)

	for i, char := range str {
		if i > 0 && (length-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(char)
	}

	return result.String()
}

// formatMonitorDuration formats a duration as HH:MM:SS for monitor display
func formatMonitorDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh%02dm", h, m)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}
