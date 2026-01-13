package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/niac-go/pkg/ipc"
)

var statusOptions struct {
	jsonOutput bool
	socketPath string
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Query the status of a running NIAC simulation",
	Long: `Query the status of a running NIAC simulation via IPC socket.

This command connects to a running NIAC simulation and retrieves
current status information including:
  - Simulation state (running/stopped)
  - Network interface in use
  - Configuration file path
  - Device count
  - Uptime
  - Packet statistics (RX/TX)
  - Active error injections

Exit codes:
  0 - Simulation is running
  1 - Simulation is not running (socket not found or connection refused)
  2 - Error occurred (socket error, parse error, etc.)`,
	Example: `  # Check simulation status
  niac status

  # Output status as JSON
  niac status --json

  # Use a custom socket path
  niac status --socket /var/run/niac.sock

  # Use in scripts
  if niac status > /dev/null 2>&1; then
    echo "NIAC is running"
  else
    echo "NIAC is not running"
  fi`,
	Args: cobra.NoArgs,
	Run:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().BoolVar(&statusOptions.jsonOutput, "json", false, "Output status as JSON")
	statusCmd.Flags().StringVar(&statusOptions.socketPath, "socket", "", "Path to IPC socket (default: /tmp/niac.sock)")
}

//nolint:gocognit // Status display aggregates multiple data sources
func runStatus(cmd *cobra.Command, args []string) {
	// Determine socket path
	socketPath := statusOptions.socketPath
	if socketPath == "" {
		socketPath = ipc.DefaultSocketPath()
	}

	// Check if socket exists
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		if statusOptions.jsonOutput {
			outputStatusJSON(map[string]any{
				"running": false,
				"error":   "socket not found",
			})
		} else {
			fmt.Fprintln(os.Stdout, "Status: NOT RUNNING")
			fmt.Fprintf(os.Stdout, "Socket not found: %s\n", socketPath)
		}
		os.Exit(1)
	}

	// Connect to socket
	var dialer net.Dialer
	dialer.Timeout = 5 * time.Second
	conn, err := dialer.DialContext(context.Background(), "unix", socketPath)
	if err != nil {
		if statusOptions.jsonOutput {
			outputStatusJSON(map[string]any{
				"running": false,
				"error":   fmt.Sprintf("connection failed: %v", err),
			})
		} else {
			fmt.Fprintln(os.Stdout, "Status: NOT RUNNING")
			fmt.Fprintf(os.Stdout, "Failed to connect: %v\n", err)
		}
		os.Exit(1)
	}
	closeConn := func() {
		_ = conn.Close()
	}

	// Set timeout for socket operations
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Send status request
	var req ipc.Request
	req.Command = ipc.CommandStatus

	encoder := json.NewEncoder(conn)
	if encodeErr := encoder.Encode(&req); encodeErr != nil {
		if statusOptions.jsonOutput {
			outputStatusJSON(map[string]any{
				"running": false,
				"error":   fmt.Sprintf("failed to send request: %v", encodeErr),
			})
		} else {
			fmt.Fprintf(os.Stderr, "Error: failed to send request: %v\n", encodeErr)
		}
		closeConn()
		os.Exit(2)
	}

	// Read response
	var resp ipc.Response
	decoder := json.NewDecoder(conn)
	if decodeErr := decoder.Decode(&resp); decodeErr != nil {
		if statusOptions.jsonOutput {
			outputStatusJSON(map[string]any{
				"running": false,
				"error":   fmt.Sprintf("failed to read response: %v", decodeErr),
			})
		} else {
			fmt.Fprintf(os.Stderr, "Error: failed to read response: %v\n", decodeErr)
		}
		closeConn()
		os.Exit(2)
	}

	// Check for error in response
	if !resp.Success {
		if statusOptions.jsonOutput {
			outputStatusJSON(map[string]any{
				"running": false,
				"error":   resp.Error,
			})
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Error)
		}
		closeConn()
		os.Exit(2)
	}

	// Parse status data
	statusData, ok := resp.Data["status"]
	if !ok {
		if statusOptions.jsonOutput {
			outputStatusJSON(map[string]any{
				"running": false,
				"error":   "status data not found in response",
			})
		} else {
			fmt.Fprintf(os.Stderr, "Error: status data not found in response\n")
		}
		closeConn()
		os.Exit(2)
	}

	// Convert to StatusData struct
	statusBytes, err := json.Marshal(statusData)
	if err != nil {
		if statusOptions.jsonOutput {
			outputStatusJSON(map[string]any{
				"running": false,
				"error":   fmt.Sprintf("failed to parse status: %v", err),
			})
		} else {
			fmt.Fprintf(os.Stderr, "Error: failed to parse status: %v\n", err)
		}
		closeConn()
		os.Exit(2)
	}

	var status ipc.StatusData
	if unmarshalErr := json.Unmarshal(statusBytes, &status); unmarshalErr != nil {
		if statusOptions.jsonOutput {
			outputStatusJSON(map[string]any{
				"running": false,
				"error":   fmt.Sprintf("failed to unmarshal status: %v", unmarshalErr),
			})
		} else {
			fmt.Fprintf(os.Stderr, "Error: failed to unmarshal status: %v\n", unmarshalErr)
		}
		closeConn()
		os.Exit(2)
	}

	// Output based on format
	if statusOptions.jsonOutput {
		outputStatusJSON(map[string]any{
			"running":          status.Running,
			"interface":        status.Interface,
			"config":           status.ConfigPath,
			"devices":          status.DeviceCount,
			"uptime_seconds":   status.Uptime,
			"uptime_formatted": formatDurationFromSeconds(status.Uptime),
			"packets_rx":       status.PacketsRX,
			"packets_tx":       status.PacketsTX,
			"errors_active":    status.ErrorsActive,
			"started_at":       status.StartedAt,
		})
	} else {
		printHumanStatus(&status)
	}

	// Exit with code 0 for running
	closeConn()
	os.Exit(0)
}

// printHumanStatus prints the status in human-readable format.
func printHumanStatus(status *ipc.StatusData) {
	statusStr := "STOPPED"
	if status.Running {
		statusStr = "RUNNING"
	}

	fmt.Fprintf(os.Stdout, "Status: %s\n", statusStr)
	fmt.Fprintf(os.Stdout, "Interface: %s\n", status.Interface)
	fmt.Fprintf(os.Stdout, "Config: %s\n", status.ConfigPath)
	fmt.Fprintf(os.Stdout, "Devices: %d\n", status.DeviceCount)
	fmt.Fprintf(os.Stdout, "Uptime: %s\n", formatDurationFromSeconds(status.Uptime))
	fmt.Fprintf(os.Stdout, "Packets RX: %s\n", formatStatusNumber(status.PacketsRX))
	fmt.Fprintf(os.Stdout, "Packets TX: %s\n", formatStatusNumber(status.PacketsTX))
	fmt.Fprintf(os.Stdout, "Errors Active: %d\n", status.ErrorsActive)
}

// formatDurationFromSeconds formats a duration in seconds as "2h 15m 43s".
func formatDurationFromSeconds(seconds float64) string {
	d := time.Duration(seconds * float64(time.Second))
	return formatDurationHMS(d)
}

// formatDurationHMS formats a time.Duration as "2h 15m 43s".
func formatDurationHMS(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// formatStatusNumber formats a number with thousand separators (e.g., 125432 -> "125,432").
func formatStatusNumber(n uint64) string {
	if n < 1000 {
		return strconv.FormatUint(n, 10)
	}

	// Convert to string and add separators
	str := strconv.FormatUint(n, 10)
	length := len(str)

	var resultSb262 strings.Builder
	for i, char := range str {
		if i > 0 && (length-i)%3 == 0 {
			resultSb262.WriteString(",")
		}
		resultSb262.WriteRune(char)
	}

	return resultSb262.String()
}

// outputStatusJSON outputs data as formatted JSON.
func outputStatusJSON(data any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(data)
}
