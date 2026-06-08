package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/ipc"
)

type statusOptions struct {
	jsonOutput bool
	socketPath string
}

func addStatusCommand(root *cobra.Command, _ *serviceOptions) {
	options := new(statusOptions)

	statusCmd := &cobra.Command{
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
		Run: func(_ *cobra.Command, _ []string) {
			runStatus(options)
		},
	}

	statusCmd.Flags().BoolVar(&options.jsonOutput, "json", false, "Output status as JSON")
	statusCmd.Flags().StringVar(&options.socketPath, "socket", "", "Path to IPC socket (default: /tmp/niac.sock)")

	root.AddCommand(statusCmd)
}

// Status exit codes.
const (
	exitCodeNotRunning = 1
	exitCodeSuccess    = 0
)

// statusResult holds the result of a status check operation.
type statusResult struct {
	status   *ipc.StatusData
	exitCode int
	err      error
}

func runStatus(options *statusOptions) {
	socketPath := resolveSocketPath(options.socketPath)

	result := checkStatus(socketPath)

	outputResult(result, options.jsonOutput)
	os.Exit(result.exitCode)
}

// resolveSocketPath returns the socket path, using default if not specified.
func resolveSocketPath(path string) string {
	if path == "" {
		return ipc.DefaultSocketPath()
	}
	return path
}

// checkStatus performs the full status check workflow.
func checkStatus(socketPath string) statusResult {
	if err := verifySocketExists(socketPath); err != nil {
		return statusResult{exitCode: exitCodeNotRunning, err: err}
	}

	conn, err := connectToSocket(socketPath)
	if err != nil {
		return statusResult{exitCode: exitCodeNotRunning, err: err}
	}
	defer conn.Close()

	status, err := requestStatus(conn)
	if err != nil {
		return statusResult{exitCode: exitCodeError, err: err}
	}

	return statusResult{status: status, exitCode: exitCodeSuccess}
}

// verifySocketExists checks if the IPC socket file exists.
func verifySocketExists(socketPath string) error {
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return fmt.Errorf("socket not found: %s", socketPath)
	}
	return nil
}

// connectToSocket establishes a connection to the IPC socket.
func connectToSocket(socketPath string) (net.Conn, error) {
	dialer := net.Dialer{Timeout: shortTimeout * time.Second}

	conn, err := dialer.DialContext(context.Background(), "unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}

	_ = conn.SetDeadline(time.Now().Add(shortTimeout * time.Second))

	return conn, nil
}

// requestStatus sends a status request and parses the response.
func requestStatus(conn net.Conn) (*ipc.StatusData, error) {
	if err := sendStatusRequest(conn); err != nil {
		return nil, err
	}

	resp, err := receiveResponse(conn)
	if err != nil {
		return nil, err
	}

	return parseStatusResponse(resp)
}

// sendStatusRequest sends the status command to the IPC server.
func sendStatusRequest(conn net.Conn) error {
	req := ipc.Request{Command: ipc.CommandStatus}
	encoder := json.NewEncoder(conn)

	if err := encoder.Encode(&req); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	return nil
}

// receiveResponse reads and decodes the IPC response.
func receiveResponse(conn net.Conn) (*ipc.Response, error) {
	var resp ipc.Response
	decoder := json.NewDecoder(conn)

	if err := decoder.Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("%s", resp.Error)
	}

	return &resp, nil
}

// parseStatusResponse extracts and parses StatusData from the response.
func parseStatusResponse(resp *ipc.Response) (*ipc.StatusData, error) {
	statusData, ok := resp.Data["status"]
	if !ok {
		return nil, errors.New("status data not found in response")
	}

	statusBytes, err := json.Marshal(statusData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse status: %w", err)
	}

	var status ipc.StatusData
	if err = json.Unmarshal(statusBytes, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal status: %w", err)
	}

	return &status, nil
}

// outputResult formats and outputs the status result.
func outputResult(result statusResult, jsonOutput bool) {
	if result.err != nil {
		outputError(result.err, result.exitCode, jsonOutput)
		return
	}

	outputSuccess(result.status, jsonOutput)
}

// outputError outputs an error in the appropriate format.
func outputError(err error, exitCode int, jsonOutput bool) {
	if jsonOutput {
		outputStatusJSON(map[string]any{
			"running": false,
			"error":   err.Error(),
		})
		return
	}

	if exitCode == exitCodeNotRunning {
		fmt.Fprintln(os.Stdout, "Status: NOT RUNNING")
		fmt.Fprintf(os.Stdout, "%v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}

// outputSuccess outputs successful status in the appropriate format.
func outputSuccess(status *ipc.StatusData, jsonOutput bool) {
	if jsonOutput {
		outputStatusJSON(buildStatusMap(status))
		return
	}

	printHumanStatus(status)
}

// buildStatusMap creates a map representation of status data for JSON output.
func buildStatusMap(status *ipc.StatusData) map[string]any {
	return map[string]any{
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
	}
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

// formatDurationHMS formats a [time.Duration] as "2h 15m 43s".
func formatDurationHMS(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % secondsPerMinute
	s := int(d.Seconds()) % secondsPerMinute

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
	if n < millisecondsThreshold {
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
