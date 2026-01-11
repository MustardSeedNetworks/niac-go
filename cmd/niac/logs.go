// Package main provides the logs command for streaming and viewing simulation logs
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

	"github.com/krisarmstrong/niac-go/pkg/ipc"
	"github.com/krisarmstrong/niac-go/pkg/logging"
	"github.com/spf13/cobra"
)

var logsOpts struct {
	follow     bool
	level      string
	filter     string
	socketPath string
	jsonOutput bool
	count      int
}

var logsCmd = &cobra.Command{
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

var logsTailCmd = &cobra.Command{
	Use:   "tail",
	Short: "Stream logs from a running simulation",
	Long: `Stream logs from a running NIAC simulation in real-time.

The tail command connects to the running simulation via IPC socket and
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

  # Custom socket path
  niac logs tail --socket /var/run/niac.sock`,
	Args: cobra.NoArgs,
	RunE: runLogsTail,
}

func init() {
	rootCmd.AddCommand(logsCmd)
	logsCmd.AddCommand(logsTailCmd)

	// Tail command flags
	logsTailCmd.Flags().BoolVarP(&logsOpts.follow, "follow", "f", false,
		"Continuously stream new logs (like tail -f)")
	logsTailCmd.Flags().StringVarP(&logsOpts.level, "level", "l", "info",
		"Minimum log level: debug, info, warn, error")
	logsTailCmd.Flags().StringVar(&logsOpts.filter, "filter", "",
		"Filter logs by text pattern (case-insensitive)")
	logsTailCmd.Flags().StringVar(&logsOpts.socketPath, "socket", "",
		"Path to IPC socket (default: "+ipc.DefaultSocketPath()+")")
	logsTailCmd.Flags().BoolVar(&logsOpts.jsonOutput, "json", false,
		"Output logs as JSON (one object per line)")
	logsTailCmd.Flags().IntVarP(&logsOpts.count, "count", "n", 100,
		"Number of recent logs to display (default: 100)")
}

func runLogsTail(cmd *cobra.Command, args []string) error {
	// Validate log level
	level := strings.ToLower(logsOpts.level)
	switch level {
	case "debug", "info", "warn", "error":
		// Valid
	default:
		return fmt.Errorf("invalid log level %q: must be debug, info, warn, or error", logsOpts.level)
	}

	// Create IPC client
	socketPath := logsOpts.socketPath
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

	if logsOpts.follow {
		return runLogsFollow(ctx, client, level)
	}

	return runLogsOnce(client, level)
}

// runLogsOnce fetches and displays logs once (no follow mode)
func runLogsOnce(client *ipc.Client, level string) error {
	logs, err := client.GetLogs(level, logsOpts.count)
	if err != nil {
		return fmt.Errorf("failed to get logs: %w", err)
	}

	// Apply text filter
	filtered := filterLogs(logs, logsOpts.filter)

	if logsOpts.jsonOutput {
		for _, log := range filtered {
			outputLogJSON(log)
		}
	} else {
		logging.InitColors(true)
		for _, log := range filtered {
			printLogEntry(log)
		}
	}

	return nil
}

// runLogsFollow streams logs continuously
func runLogsFollow(ctx context.Context, client *ipc.Client, level string) error {
	sub := client.SubscribeLogs(level, logsOpts.filter, 500*time.Millisecond)
	defer sub.Stop()

	if !logsOpts.jsonOutput {
		logging.InitColors(true)
		fmt.Println("Streaming logs (press Ctrl+C to stop)...")
		fmt.Println(strings.Repeat("-", 80))
	}

	for {
		select {
		case <-ctx.Done():
			if !logsOpts.jsonOutput {
				fmt.Println("\nLog streaming stopped.")
			}
			return nil
		case log, ok := <-sub.Logs():
			if !ok {
				return nil
			}
			if logsOpts.jsonOutput {
				outputLogJSON(log)
			} else {
				printLogEntry(log)
			}
		case err := <-sub.Errors():
			if !logsOpts.jsonOutput {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
		}
	}
}

// filterLogs filters logs by the text pattern
func filterLogs(logs []ipc.LogEntry, filter string) []ipc.LogEntry {
	if filter == "" {
		return logs
	}

	filter = strings.ToLower(filter)
	filtered := make([]ipc.LogEntry, 0)
	for _, log := range logs {
		if strings.Contains(strings.ToLower(log.Message), filter) ||
			strings.Contains(strings.ToLower(log.Device), filter) ||
			strings.Contains(strings.ToLower(log.Source), filter) ||
			strings.Contains(strings.ToLower(log.Protocol), filter) {
			filtered = append(filtered, log)
		}
	}
	return filtered
}

// printLogEntry prints a log entry in human-readable format
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
		fmt.Printf("%s %s %s: %s\n", timestamp, levelStr, context, log.Message)
	} else {
		fmt.Printf("%s %s %s\n", timestamp, levelStr, log.Message)
	}
}

// outputLogJSON outputs a log entry as JSON
func outputLogJSON(log ipc.LogEntry) {
	output := map[string]interface{}{
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
	fmt.Println(string(jsonBytes))
}
