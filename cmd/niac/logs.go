package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/cliclient"
	"github.com/MustardSeedNetworks/niac-go/internal/ipc"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

type logsOptions struct {
	follow     bool
	level      string
	filter     string
	api        string
	caCert     string
	insecure   bool
	jsonOutput bool
	count      int
}

func addLogsCommand(root *cobra.Command, _ *serviceOptions) {
	options := new(logsOptions)

	logsCmd := &cobra.Command{
		Use:   "logs",
		Short: "View and stream simulation logs",
		Long: `View and stream logs from a running NIAC simulation.

The logs command provides access to simulation logs including device activity,
protocol messages, and error injections. Logs can be filtered by level and
text pattern, and can be streamed in real-time with the tail subcommand.`,
		Example: `  # View recent logs
  niac logs tail

  # Stream logs in real-time
  niac logs tail --follow

  # Filter by log level
  niac logs tail --level warn

  # Filter by text pattern
  niac logs tail --filter "device"

  # Output as JSON
  niac logs tail --json`,
	}

	logsTailCmd := &cobra.Command{
		Use:   "tail",
		Short: "Stream logs from a running simulation",
		Long: `Stream logs from a running NIAC simulation in real-time.

The tail command reads the daemon's live log stream over its HTTPS API and
displays log messages. Use --follow to continuously stream new logs.

Log levels (from most to least verbose):
  - debug: All messages including detailed debugging information
  - info:  Informational messages about normal operation
  - warn:  Warning messages about potential issues
  - error: Error messages about failures

The --filter option performs case-insensitive substring matching on log messages.`,
		Example: `  # View recent logs (one-shot)
  niac logs tail

  # Stream logs continuously
  niac logs tail --follow

  # Show only warnings and errors
  niac logs tail --level warn

  # Filter for specific device
  niac logs tail --filter "router-1"

  # Combine options
  niac logs tail --follow --level info --filter "LLDP"

  # JSON output for scripting
  niac logs tail --json | jq '.message'

  # Read from a daemon on another address
  niac logs tail --api https://10.0.0.5:8445`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runLogsTail(options)
		},
	}

	logsCmd.AddCommand(logsTailCmd)

	// Tail command flags
	logsTailCmd.Flags().BoolVarP(&options.follow, "follow", "f", false,
		"Continuously stream new logs (like tail -f)")
	logsTailCmd.Flags().StringVarP(&options.level, "level", "l", "info",
		"Minimum log level: debug, info, warn, error")
	logsTailCmd.Flags().StringVar(&options.filter, "filter", "",
		"Filter logs by text pattern (case-insensitive)")
	logsTailCmd.Flags().StringVar(&options.api, "api", "",
		"Daemon API address (default: "+cliclient.DefaultBaseURL+", or NIAC_API_URL)")
	logsTailCmd.Flags().StringVar(&options.caCert, "cacert", "",
		"Daemon certificate to trust (default: the local daemon's own, when visible)")
	logsTailCmd.Flags().BoolVar(&options.insecure, "insecure", false,
		"Skip TLS verification, for a daemon whose certificate this host cannot see")
	logsTailCmd.Flags().BoolVar(&options.jsonOutput, "json", false,
		"Output logs as JSON (one object per line)")
	logsTailCmd.Flags().IntVarP(&options.count, "count", "n", maxLogEntries,
		"Number of recent logs to display (default: 100)")

	root.AddCommand(logsCmd)
}

func runLogsTail(options *logsOptions) error {
	// Validate log level
	level := strings.ToLower(options.level)
	switch level {
	case "debug", "info", "warn", "error":
		// Valid
	default:
		return fmt.Errorf("invalid log level %q: must be debug, info, warn, or error", options.level)
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

	if options.follow {
		return runLogsFollow(ctx, client, level, options)
	}

	return runLogsOnce(ctx, client, level, options)
}

// runLogsOnce collects from the live stream until it has the requested count or
// the window closes.
//
// The daemon keeps no backlog: it broadcasts records as they happen and retains
// nothing, so there is no "last N lines" to ask for. This reads what arrives
// and stops, which is honest about what the daemon can answer; --follow is what
// you want if the simulation is quiet.
func runLogsOnce(ctx context.Context, client *cliclient.Client, level string, options *logsOptions) error {
	ctx, cancel := context.WithTimeout(ctx, logsOnceWindow)
	defer cancel()

	if !options.jsonOutput {
		logging.InitColors(true)
	}
	seen := 0
	err := streamLogs(ctx, client, level, options, func() bool {
		seen++

		return seen < options.count
	})
	if err != nil {
		return err
	}
	if seen == 0 && !options.jsonOutput {
		fmt.Fprintf(os.Stdout,
			"No log records in %s. The daemon keeps no backlog; use --follow to watch.\n",
			logsOnceWindow)
	}

	return nil
}

// logsOnceWindow is how long a non-following read waits for records. Long
// enough to catch a busy simulation, short enough not to look hung.
const logsOnceWindow = 2 * time.Second

// runLogsFollow streams logs continuously.
func runLogsFollow(ctx context.Context, client *cliclient.Client, level string, options *logsOptions) error {
	if !options.jsonOutput {
		logging.InitColors(true)
		fmt.Fprintln(os.Stdout, "Streaming logs (press Ctrl+C to stop)...")
		fmt.Fprintln(os.Stdout, strings.Repeat("-", lineWidthStandard))
	}
	err := streamLogs(ctx, client, level, options, func() bool { return true })
	if err != nil && ctx.Err() == nil {
		return err
	}
	if !options.jsonOutput {
		fmt.Fprintln(os.Stdout, "\nLog streaming stopped.")
	}

	return nil
}

// streamLogs applies the level and text filters to the daemon's stream and
// prints what survives, stopping when keepGoing says so.
func streamLogs(
	ctx context.Context,
	client *cliclient.Client,
	level string,
	options *logsOptions,
	keepGoing func() bool,
) error {
	return client.StreamLogs(ctx, func(event cliclient.LogEvent) bool {
		entry := ipc.LogEntry{
			Timestamp: parseLogTime(event.Timestamp),
			Level:     ipc.LogLevel(event.Level),
			Message:   event.Message,
			Device:    event.Device,
			Source:    event.Source,
			Protocol:  event.Protocol,
		}
		if !matchesLevel(string(entry.Level), level) || !ipc.LogMatchesFilter(entry, options.filter) {
			return true
		}
		if options.jsonOutput {
			outputLogJSON(entry)
		} else {
			printLogEntry(entry)
		}

		return keepGoing()
	})
}

// matchesLevel keeps records at or above the requested level.
func matchesLevel(recordLevel, wanted string) bool {
	if wanted == "" {
		return true
	}

	return logLevelRank(recordLevel) >= logLevelRank(wanted)
}

// logLevelRank orders the levels so a request for one keeps everything at or
// above it, which is what "--level warn" means to an operator.
func logLevelRank(level string) int {
	const (
		rankDebug = iota
		rankInfo
		rankWarn
		rankError
	)
	switch strings.ToLower(level) {
	case "debug":
		return rankDebug
	case "warn", "warning":
		return rankWarn
	case "error":
		return rankError
	default:
		return rankInfo
	}
}

// parseLogTime reads the stream's timestamp, falling back to now when the
// daemon sends a shape this build does not know.
func parseLogTime(value string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "15:04:05.000"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}

	return time.Now()
}

// filterLogs filters logs by the text pattern. The match itself lives in the ipc
// package so that this path and the --follow subscription cannot drift apart -
// they are the same flag and must answer the same way.
func filterLogs(logs []ipc.LogEntry, filter string) []ipc.LogEntry {
	if filter == "" {
		return logs
	}
	filtered := make([]ipc.LogEntry, 0, len(logs))
	for _, log := range logs {
		if ipc.LogMatchesFilter(log, filter) {
			filtered = append(filtered, log)
		}
	}

	return filtered
}

// printLogEntry prints a log entry in human-readable format.
func printLogEntry(log ipc.LogEntry) {
	timestamp := log.Timestamp.Format("15:04:05.000")

	// Format level with color
	var levelStr string
	switch log.Level {
	case ipc.LogLevelDebug:
		levelStr = logging.Sprintf("debug", "%-5s", "DEBUG")
	case ipc.LogLevelInfo:
		levelStr = logging.Sprintf("info", "%-5s", "INFO")
	case ipc.LogLevelWarn:
		levelStr = logging.Sprintf("warning", "%-5s", "WARN")
	case ipc.LogLevelError:
		levelStr = logging.Sprintf("error", "%-5s", "ERROR")
	default:
		levelStr = fmt.Sprintf("%-5s", string(log.Level))
	}

	// Build context string from source, device, and protocol
	var context string
	if log.Device != "" {
		context = logging.DeviceString(log.Device)
	} else if log.Source != "" {
		context = logging.ProtocolString(log.Source)
	}
	if log.Protocol != "" && log.Protocol != log.Source {
		if context != "" {
			context += " "
		}
		context += logging.ProtocolString("[" + log.Protocol + "]")
	}

	if context != "" {
		fmt.Fprintf(os.Stdout, "%s %s %s: %s\n", timestamp, levelStr, context, log.Message)
	} else {
		fmt.Fprintf(os.Stdout, "%s %s %s\n", timestamp, levelStr, log.Message)
	}
}

// outputLogJSON outputs a log entry as JSON.
func outputLogJSON(log ipc.LogEntry) {
	output := map[string]any{
		"timestamp": log.Timestamp.Format(time.RFC3339Nano),
		"level":     string(log.Level),
		"message":   log.Message,
	}
	if log.Source != "" {
		output["source"] = log.Source
	}
	if log.Device != "" {
		output["device"] = log.Device
	}
	if log.Protocol != "" {
		output["protocol"] = log.Protocol
	}

	jsonBytes, err := json.Marshal(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "JSON encoding error: %v\n", err)
		return
	}
	fmt.Fprintln(os.Stdout, string(jsonBytes))
}
