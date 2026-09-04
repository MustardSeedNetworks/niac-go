package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/cliclient"
)

type statusOptions struct {
	jsonOutput bool
	api        string
	caCert     string
	insecure   bool
}

func addStatusCommand(root *cobra.Command, _ *serviceOptions) {
	options := new(statusOptions)

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Query the status of a running NIAC simulation",
		Long: `Query the status of a running NIAC simulation over the daemon API.

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
  1 - Simulation is not running (no daemon answered, or it reports idle)
  2 - Error occurred (unreachable daemon, authentication, parse error)`,
		Example: `  # Check simulation status
  niac status

  # Output status as JSON
  niac status --json

  # Reach a daemon on another address
  niac status --api https://127.0.0.1:8445

  # Use in scripts
  if niac status > /dev/null 2>&1; then
    echo "NIAC is running"
  else
    echo "NIAC is not running"
  fi`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runStatus(cmd.Context(), options)
		},
	}

	statusCmd.Flags().BoolVar(&options.jsonOutput, "json", false, "Output status as JSON")
	statusCmd.Flags().StringVar(&options.api, "api", "",
		"Daemon API address (default: "+cliclient.DefaultBaseURL+", or NIAC_API_URL)")
	statusCmd.Flags().StringVar(&options.caCert, "cacert", "",
		"Daemon certificate to trust (default: the local daemon's own, when visible)")
	statusCmd.Flags().BoolVar(&options.insecure, "insecure", false,
		"Skip TLS verification, for a daemon whose certificate this host cannot see")

	root.AddCommand(statusCmd)
}

// Status exit codes.
const (
	exitCodeNotRunning = 1
	exitCodeSuccess    = 0
)

// statusResult holds the result of a status check operation.
type statusResult struct {
	status   *cliclient.Runtime
	exitCode int
	err      error
}

// errStatusReported marks a status result that has already been printed.
var errStatusReported = errors.New("simulation is not running")

func runStatus(ctx context.Context, options *statusOptions) error {
	result := checkStatus(ctx, options)

	outputResult(result, options.jsonOutput)

	if result.exitCode == exitCodeSuccess {
		return nil
	}

	// outputResult has already reported the detail; the error exists to carry
	// the exit code, and cobra's usage text would be noise on top of it.
	return withExitCode(result.exitCode, errStatusReported)
}

// checkStatus asks the daemon what it is running.
//
// It used to open a unix socket whose server was removed in #1012, so this
// command reported NOT RUNNING no matter how many simulations were up.
func checkStatus(ctx context.Context, options *statusOptions) statusResult {
	client, err := newCLIClient(options.api, options.caCert, options.insecure)
	if err != nil {
		return statusResult{exitCode: exitCodeError, err: err}
	}
	runtime, err := client.Runtime(ctx)
	if err != nil {
		if errors.Is(err, cliclient.ErrDaemonUnreachable) {
			return statusResult{exitCode: exitCodeNotRunning, err: err}
		}

		return statusResult{exitCode: exitCodeError, err: err}
	}
	if !runtime.Running {
		return statusResult{
			exitCode: exitCodeNotRunning,
			err:      errors.New("the daemon is up but no simulation is running"),
		}
	}

	return statusResult{status: runtime, exitCode: exitCodeSuccess}
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
func outputSuccess(status *cliclient.Runtime, jsonOutput bool) {
	if jsonOutput {
		outputStatusJSON(buildStatusMap(status))
		return
	}

	printHumanStatus(status)
}

// buildStatusMap creates a map representation of status data for JSON output.
func buildStatusMap(status *cliclient.Runtime) map[string]any {
	return map[string]any{
		"running":          status.Running,
		"interface":        status.Interface,
		"config":           status.ConfigName,
		"devices":          status.DeviceCount,
		"uptime_seconds":   status.Uptime,
		"uptime_formatted": formatDurationFromSeconds(status.Uptime),
		"packets_rx":       status.PacketsRX,
		"packets_tx":       status.PacketsTX,
		"version":          status.Version,
	}
}

// printHumanStatus prints the status in human-readable format.
func printHumanStatus(status *cliclient.Runtime) {
	statusStr := "STOPPED"
	if status.Running {
		statusStr = "RUNNING"
	}

	fmt.Fprintf(os.Stdout, "Status: %s\n", statusStr)
	fmt.Fprintf(os.Stdout, "Interface: %s\n", status.Interface)
	fmt.Fprintf(os.Stdout, "Config: %s\n", status.ConfigName)
	fmt.Fprintf(os.Stdout, "Devices: %d\n", status.DeviceCount)
	fmt.Fprintf(os.Stdout, "Uptime: %s\n", formatDurationFromSeconds(status.Uptime))
	fmt.Fprintf(os.Stdout, "Packets RX: %s\n", formatStatusNumber(status.PacketsRX))
	fmt.Fprintf(os.Stdout, "Packets TX: %s\n", formatStatusNumber(status.PacketsTX))
	fmt.Fprintf(os.Stdout, "Version: %s\n", status.Version)
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
