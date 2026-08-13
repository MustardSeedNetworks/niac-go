package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/cliclient"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// OutputFormat represents the output format type for monitor command.
type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatCSV   OutputFormat = "csv"
)

// monitorStats holds the statistics for a single monitoring sample.
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
	// Session names the scenario these counters belong to. Several run at
	// once, each with its own stack, so a sample without it is ambiguous.
	Session string `json:"session"`

	// Rates (packets per second)
	RateRX float64 `json:"rate_rx,omitempty"`
	RateTX float64 `json:"rate_tx,omitempty"`
}

type monitorOptions struct {
	format   string
	interval string
	api      string
	caCert   string
	session  string
	insecure bool
}

func addMonitorCommand(root *cobra.Command, _ *serviceOptions) {
	options := new(monitorOptions)

	monitorCmd := &cobra.Command{
		Use:   "monitor",
		Short: "Stream real-time statistics from a running NIAC simulation",
		Long: `Monitor a running NIAC simulation in real-time.

This command reads statistics from a running NIAC daemon over its HTTPS API
and displays them continuously, similar to 'top' or 'watch'.

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

  # Read from a daemon on another address
  niac monitor --api https://10.0.0.5:8445

  # Monitor with fast 500ms updates
  niac monitor --interval 500ms`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runMonitor(options)
		},
	}

	monitorCmd.Flags().StringVar(&options.format, "format", "table",
		"Output format: table, json, or csv")
	monitorCmd.Flags().StringVar(&options.interval, "interval", "1s",
		"Update interval (e.g., 1s, 500ms, 2s)")
	monitorCmd.Flags().StringVar(&options.api, "api", "",
		"Daemon API address (default: "+cliclient.DefaultBaseURL+", or NIAC_API_URL)")
	monitorCmd.Flags().StringVar(&options.session, "session", "",
		"Scenario session to watch (default: whichever the daemon has selected)")
	monitorCmd.Flags().StringVar(&options.caCert, "cacert", "",
		"Daemon certificate to trust (default: the local daemon's own, when visible)")
	monitorCmd.Flags().BoolVar(&options.insecure, "insecure", false,
		"Skip TLS verification, for a daemon whose certificate this host cannot see")

	root.AddCommand(monitorCmd)
}

func runMonitor(options *monitorOptions) error {
	// Parse interval
	interval, err := time.ParseDuration(options.interval)
	if err != nil {
		return fmt.Errorf("invalid interval %q: %w", options.interval, err)
	}

	if interval < 100*time.Millisecond {
		return errors.New("interval must be at least 100ms")
	}

	// Validate format
	format := OutputFormat(strings.ToLower(options.format))
	switch format {
	case FormatTable, FormatJSON, FormatCSV:
		// Valid
	default:
		return fmt.Errorf("invalid format %q: must be table, json, or csv", options.format)
	}

	client, err := newCLIClient(options.api, options.caCert, options.insecure)
	if err != nil {
		return err
	}

	// Set up context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	go func() {
		<-sigChan
		cancel()
	}()

	// Run the appropriate monitor based on format
	switch format {
	case FormatTable:
		return runTableMonitor(ctx, client, options.session, interval)
	case FormatJSON:
		return runJSONMonitor(ctx, client, options.session, interval)
	case FormatCSV:
		return runCSVMonitor(ctx, client, options.session, interval)
	}

	return nil
}

// runTableMonitor displays statistics in a continuously-updated table format.
func runTableMonitor(
	ctx context.Context,
	client *cliclient.Client,
	session string,
	interval time.Duration,
) error {
	var prevStats *monitorStats
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logging.InitColors(true)

	// Initial fetch
	stats, err := fetchStats(ctx, client, session, prevStats, interval)
	if err != nil {
		return fmt.Errorf("failed to fetch initial stats: %w", err)
	}
	printTableHeader()
	printTableRow(stats)
	prevStats = stats

	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stdout, "\nMonitoring stopped.")
			return nil
		case <-ticker.C:
			latestStats, fetchErr := fetchStats(ctx, client, session, prevStats, interval)
			if fetchErr != nil {
				// Connection lost, but don't exit immediately
				fmt.Fprintf(os.Stdout, "\r[%s] Connection lost: %v\n", time.Now().Format("15:04:05"), fetchErr)
				continue
			}
			clearScreen()
			printTableHeader()
			printTableRow(latestStats)
			prevStats = latestStats
		}
	}
}

// runJSONMonitor outputs statistics as JSON Lines (one JSON object per line).
func runJSONMonitor(
	ctx context.Context,
	client *cliclient.Client,
	session string,
	interval time.Duration,
) error {
	var prevStats *monitorStats
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial fetch
	stats, err := fetchStats(ctx, client, session, prevStats, interval)
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
			latestStats, fetchErr := fetchStats(ctx, client, session, prevStats, interval)
			if fetchErr != nil {
				// Output error as JSON
				errObj := map[string]any{
					"error": fetchErr.Error(),
					"time":  time.Now().Format(time.RFC3339),
				}
				jsonBytes, _ := json.Marshal(errObj)
				fmt.Fprintln(os.Stdout, string(jsonBytes))
				continue
			}
			outputMonitorJSON(latestStats)
			prevStats = latestStats
		}
	}
}

// runCSVMonitor outputs statistics as CSV.
func runCSVMonitor(
	ctx context.Context,
	client *cliclient.Client,
	session string,
	interval time.Duration,
) error {
	var prevStats *monitorStats
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Write header
	header := []string{
		"time",
		"packets_rx",
		"packets_tx",
		"arp",
		"icmp",
		"dns",
		"dhcp",
		"snmp",
		"errors",
		"rate_rx",
		"rate_tx",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}
	writer.Flush()

	// Initial fetch
	stats, err := fetchStats(ctx, client, session, prevStats, interval)
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
			latestStats, fetchErr := fetchStats(ctx, client, session, prevStats, interval)
			if fetchErr != nil {
				// Skip this interval on error
				continue
			}
			outputCSVRow(writer, latestStats)
			prevStats = latestStats
		}
	}
}

// readStats reads the named session's counters, or the selected session's when
// no name was given.
func readStats(
	ctx context.Context,
	client *cliclient.Client,
	session string,
) (*cliclient.Stats, error) {
	if session == "" {
		return client.Stats(ctx)
	}

	return client.SessionStats(ctx, session)
}

// fetchStats reads one sample from the daemon and derives the rates.
//
// The per-protocol counters were reported as zero for as long as this command
// read the IPC transport, which carried no protocol breakdown. The daemon has
// always counted them; /api/v1/stats serves them.
func fetchStats(
	ctx context.Context,
	client *cliclient.Client,
	session string,
	prev *monitorStats,
	interval time.Duration,
) (*monitorStats, error) {
	runtime, err := client.Runtime(ctx)
	if err != nil {
		return nil, err
	}
	// Several scenarios run at once, each with its own counters. Watch the one
	// named, or the daemon's selected one - and say which, because a sample
	// that does not name its session reads as though it covered everything.
	watching := session
	if watching == "" {
		watching = runtime.ConfigName
	}
	sample, err := readStats(ctx, client, session)
	if err != nil {
		return nil, err
	}

	stats := &monitorStats{
		Time:      time.Now(),
		PacketsRX: sample.Stack.PacketsReceived,
		PacketsTX: sample.Stack.PacketsSent,
		ARP:       sample.Stack.ARP(),
		ICMP:      sample.Stack.ICMP(),
		DNS:       sample.Stack.DNSQueries,
		DHCP:      sample.Stack.DHCPRequests,
		SNMP:      sample.Stack.SNMPQueries,
		Errors:    sample.Stack.Errors,
		Uptime:    runtime.Uptime,
		Session:   watching,
	}

	// Calculate rates if we have previous stats.
	if prev == nil {
		return stats, nil
	}
	intervalSec := interval.Seconds()
	if intervalSec <= 0 {
		return stats, nil
	}
	// Guard against uint64 underflow if counters reset.
	if stats.PacketsRX >= prev.PacketsRX {
		stats.RateRX = float64(stats.PacketsRX-prev.PacketsRX) / intervalSec
	}
	if stats.PacketsTX >= prev.PacketsTX {
		stats.RateTX = float64(stats.PacketsTX-prev.PacketsTX) / intervalSec
	}

	return stats, nil
}

// clearScreen clears the terminal screen and moves cursor to top-left.
func clearScreen() {
	fmt.Fprint(os.Stdout, "\033[2J\033[H")
}

// printTableHeader prints the table header.
func printTableHeader() {
	fmt.Fprintln(os.Stdout, "NIAC Monitor - Press Ctrl+C to stop")
	fmt.Fprintln(os.Stdout, strings.Repeat("-", lineWidthStandard))
	fmt.Fprintf(os.Stdout, "%-10s | %10s | %10s | %8s | %8s | %8s | %8s\n",
		"TIME", "RX PKT", "TX PKT", "RX/s", "TX/s", "UPTIME", "ERRORS")
	fmt.Fprintln(os.Stdout, strings.Repeat("-", lineWidthStandard))
}

// printTableRow prints a single statistics row.
func printTableRow(stats *monitorStats) {
	uptime := formatMonitorDuration(time.Duration(stats.Uptime) * time.Second)

	fmt.Fprintf(os.Stdout, "%-10s | %10s | %10s | %8.1f | %8.1f | %8s | %8d\n",
		stats.Time.Format("15:04:05"),
		formatMonitorNumber(stats.PacketsRX),
		formatMonitorNumber(stats.PacketsTX),
		stats.RateRX,
		stats.RateTX,
		uptime,
		stats.Errors,
	)
}

// outputMonitorJSON outputs a single stats object as JSON.
func outputMonitorJSON(stats *monitorStats) {
	jsonBytes, err := json.Marshal(stats)
	if err != nil {
		fmt.Fprintf(os.Stderr, "JSON encoding error: %v\n", err)
		return
	}
	fmt.Fprintln(os.Stdout, string(jsonBytes))
}

// outputCSVRow writes a single CSV row.
func outputCSVRow(writer *csv.Writer, stats *monitorStats) {
	row := []string{
		stats.Time.Format(time.RFC3339),
		strconv.FormatUint(stats.PacketsRX, 10),
		strconv.FormatUint(stats.PacketsTX, 10),
		strconv.FormatUint(stats.ARP, 10),
		strconv.FormatUint(stats.ICMP, 10),
		strconv.FormatUint(stats.DNS, 10),
		strconv.FormatUint(stats.DHCP, 10),
		strconv.FormatUint(stats.SNMP, 10),
		strconv.FormatUint(stats.Errors, 10),
		fmt.Sprintf("%.2f", stats.RateRX),
		fmt.Sprintf("%.2f", stats.RateTX),
	}
	_ = writer.Write(row)
	writer.Flush()
}

// formatMonitorNumber formats a number with thousands separators.
func formatMonitorNumber(n uint64) string {
	if n < millisecondsThreshold {
		return strconv.FormatUint(n, 10)
	}

	str := strconv.FormatUint(n, 10)
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

// formatMonitorDuration formats a duration as HH:MM:SS for monitor display.
func formatMonitorDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d %= time.Hour
	m := d / time.Minute
	d %= time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh%02dm", h, m)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}
