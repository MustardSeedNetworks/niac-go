package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/niac-go/pkg/ipc"
)

// Default socket path for IPC communication.
const defaultSocketPath = "/var/run/niac/niac.sock"

// Valid error types for injection (must match pkg/errors/errors.go ErrorType constants).
var validErrorTypes = []string{
	"FCS Errors",
	"Packet Discards",
	"Interface Errors",
	"High Utilization",
	"High CPU",
	"High Memory",
	"High Disk",
}

// errorTypeAliases maps user-friendly snake_case names to internal error type names.
var errorTypeAliases = map[string]string{
	"fcs_errors":       "FCS Errors",
	"packet_discards":  "Packet Discards",
	"interface_errors": "Interface Errors",
	"high_utilization": "High Utilization",
	"high_cpu":         "High CPU",
	"high_memory":      "High Memory",
	"high_disk":        "High Disk",
}

// Global socket path flag.
var injectSocketPath string

// Injection represents an active error injection.
type Injection struct {
	Device    string    `json:"device"`
	Interface string    `json:"interface"`
	ErrorType string    `json:"error_type"`
	Value     int       `json:"value"`
	Injected  time.Time `json:"injected"`
}

// InjectionResponse represents the response from the IPC server.
type InjectionResponse struct {
	Success    bool        `json:"success"`
	Message    string      `json:"message"`
	Injections []Injection `json:"injections,omitempty"`
}

var injectCmd = &cobra.Command{
	Use:   "inject <device> <error-type> <value>",
	Short: "Inject network errors on simulated devices",
	Long: `Inject network errors on simulated devices via IPC socket.

This command allows you to simulate various network error conditions
on devices managed by a running NIAC daemon. Error injection is useful
for testing monitoring systems, alerting, and network management tools.

ERROR TYPES (use either format):
  fcs_errors / "FCS Errors"              Frame Check Sequence errors (layer 2)
  packet_discards / "Packet Discards"    Input/output packet discards
  interface_errors / "Interface Errors"  General interface errors
  high_utilization / "High Utilization"  High bandwidth utilization percentage
  high_cpu / "High CPU"                  High CPU usage percentage
  high_memory / "High Memory"            High memory usage percentage
  high_disk / "High Disk"                High disk usage percentage

VALUE:
  0-100   Percentage value for the error injection
          0 = clear the injection
          1-100 = set error rate/percentage`,
	Example: `  # Inject 50% FCS errors on router-1
  niac inject router-1 fcs_errors 50

  # Simulate high CPU on switch-2
  niac inject switch-2 high_cpu 85

  # Clear injection by setting value to 0
  niac inject router-1 fcs_errors 0

  # Use custom socket path
  niac inject --socket /tmp/niac.sock router-1 packet_discards 25

  # List all active injections
  niac inject list

  # List injections in JSON format
  niac inject list --json

  # Clear all injections on a specific device
  niac inject clear --device router-1

  # Clear all injections on all devices
  niac inject clear --all`,
	Args: cobra.ExactArgs(3),
	RunE: runInject,
}

var injectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active error injections",
	Long: `List all active error injections on simulated devices.

By default, output is displayed as a formatted table. Use --json
for machine-readable JSON output suitable for scripting.`,
	Example: `  # List active injections in table format
  niac inject list

  # List active injections in JSON format
  niac inject list --json`,
	Args: cobra.NoArgs,
	RunE: runInjectList,
}

var injectClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear error injections",
	Long: `Clear error injections from simulated devices.

You must specify either --device to clear injections on a specific
device, or --all to clear all injections on all devices.`,
	Example: `  # Clear all injections on router-1
  niac inject clear --device router-1

  # Clear all injections on all devices
  niac inject clear --all`,
	Args: cobra.NoArgs,
	RunE: runInjectClear,
}

var (
	injectListJSON bool
	injectClearDev string
	injectClearAll bool
)

func init() {
	rootCmd.AddCommand(injectCmd)

	// Global socket path flag for all inject subcommands
	injectCmd.PersistentFlags().StringVar(&injectSocketPath, "socket", defaultSocketPath, "Path to NIAC IPC socket")

	// Add subcommands
	injectCmd.AddCommand(injectListCmd)
	injectCmd.AddCommand(injectClearCmd)

	// List subcommand flags
	injectListCmd.Flags().BoolVar(&injectListJSON, "json", false, "Output in JSON format")

	// Clear subcommand flags
	injectClearCmd.Flags().StringVar(&injectClearDev, "device", "", "Device name to clear injections from")
	injectClearCmd.Flags().BoolVar(&injectClearAll, "all", false, "Clear all injections on all devices")
}

func runInject(cmd *cobra.Command, args []string) error {
	device := args[0]
	errorType := args[1]
	valueStr := args[2]

	// Validate error type
	if !isValidErrorType(errorType) {
		// Show both snake_case and canonical formats in error message
		aliasKeys := make([]string, 0, len(errorTypeAliases))
		for k := range errorTypeAliases {
			aliasKeys = append(aliasKeys, k)
		}
		return fmt.Errorf("invalid error type: %s\n\nValid error types are:\n  %s\n\nOr use snake_case aliases:\n  %s",
			errorType, strings.Join(validErrorTypes, "\n  "), strings.Join(aliasKeys, "\n  "))
	}

	// Normalize error type (convert snake_case to canonical name)
	normalizedType := normalizeErrorType(errorType)

	// Parse and validate value
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return fmt.Errorf("invalid value: %s (must be a number 0-100)", valueStr)
	}
	if value < 0 || value > 100 {
		return fmt.Errorf("value must be between 0 and 100, got: %d", value)
	}

	// Connect to IPC socket and send injection command
	resp, err := sendInjectionCommand(device, normalizedType, value)
	if err != nil {
		return fmt.Errorf("failed to inject error: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("injection failed: %s", resp.Message)
	}

	// Print success message with checkmark
	if value == 0 {
		fmt.Printf("\u2713 Cleared %s injection on %s\n", normalizedType, device)
	} else {
		fmt.Printf("\u2713 Injected %s at %d%% on %s\n", normalizedType, value, device)
	}

	return nil
}

func runInjectList(cmd *cobra.Command, args []string) error {
	// Fetch active injections from IPC
	resp, err := fetchInjections()
	if err != nil {
		return fmt.Errorf("failed to fetch injections: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("failed to list injections: %s", resp.Message)
	}

	injections := resp.Injections

	// JSON output
	if injectListJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if encodeErr := encoder.Encode(injections); encodeErr != nil {
			return fmt.Errorf("failed to encode injections: %w", encodeErr)
		}
		return nil
	}

	// Table output
	if len(injections) == 0 {
		fmt.Println("No active injections")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DEVICE\tINTERFACE\tERROR TYPE\tVALUE\tINJECTED")
	fmt.Fprintln(w, "------\t---------\t----------\t-----\t--------")

	for _, inj := range injections {
		iface := inj.Interface
		if iface == "" {
			iface = "*"
		}
		injectedAt := inj.Injected.Format("2006-01-02 15:04:05")
		fmt.Fprintf(w, "%s\t%s\t%s\t%d%%\t%s\n",
			inj.Device, iface, inj.ErrorType, inj.Value, injectedAt)
	}

	if flushErr := w.Flush(); flushErr != nil {
		return fmt.Errorf("failed to flush output: %w", flushErr)
	}
	return nil
}

func runInjectClear(cmd *cobra.Command, args []string) error {
	// Validate flags: require either --device or --all
	if injectClearDev == "" && !injectClearAll {
		return errors.New("must specify either --device <name> or --all")
	}
	if injectClearDev != "" && injectClearAll {
		return errors.New("cannot specify both --device and --all")
	}

	var resp *InjectionResponse
	var err error

	if injectClearAll {
		resp, err = clearAllInjections()
		if err != nil {
			return fmt.Errorf("failed to clear all injections: %w", err)
		}
		if !resp.Success {
			return fmt.Errorf("failed to clear injections: %s", resp.Message)
		}
		fmt.Println("\u2713 Cleared all injections on all devices")
	} else {
		resp, err = clearDeviceInjections(injectClearDev)
		if err != nil {
			return fmt.Errorf("failed to clear injections for %s: %w", injectClearDev, err)
		}
		if !resp.Success {
			return fmt.Errorf("failed to clear injections: %s", resp.Message)
		}
		fmt.Printf("\u2713 Cleared all injections on %s\n", injectClearDev)
	}

	return nil
}

// isValidErrorType checks if the given error type is valid.
func isValidErrorType(errorType string) bool {
	// Check if it's a valid canonical name
	if slices.Contains(validErrorTypes, errorType) {
		return true
	}
	// Check if it's a valid alias
	_, ok := errorTypeAliases[errorType]
	return ok
}

// normalizeErrorType converts snake_case aliases to canonical error type names.
func normalizeErrorType(errorType string) string {
	if canonical, ok := errorTypeAliases[errorType]; ok {
		return canonical
	}
	return errorType
}

// getIPCClient creates an IPC client with the configured socket path.
func getIPCClient() *ipc.Client {
	return ipc.NewClient(injectSocketPath)
}

// sendInjectionCommand sends an injection command via IPC.
func sendInjectionCommand(device, errorType string, value int) (*InjectionResponse, error) {
	client := getIPCClient()

	err := client.InjectError(device, errorType, value)
	if err != nil {
		return nil, fmt.Errorf("failed to send injection command: %w", err)
	}

	return &InjectionResponse{
		Success: true,
		Message: fmt.Sprintf("Injected %s at %d%% on %s", errorType, value, device),
	}, nil
}

// fetchInjections fetches the list of active injections via IPC.
func fetchInjections() (*InjectionResponse, error) {
	client := getIPCClient()

	injections, err := client.ListInjections()
	if err != nil {
		return nil, fmt.Errorf("failed to list injections: %w", err)
	}

	// Convert ipc.ErrorInjectionData to our Injection type
	result := make([]Injection, len(injections))
	for i, inj := range injections {
		result[i] = Injection{
			Device:    inj.Device,
			Interface: inj.Interface,
			ErrorType: string(inj.ErrorType),
			Value:     inj.Value,
			Injected:  inj.Injected,
		}
	}

	return &InjectionResponse{
		Success:    true,
		Injections: result,
	}, nil
}

// clearAllInjections clears all injections on all devices via IPC.
func clearAllInjections() (*InjectionResponse, error) {
	client := getIPCClient()

	// Pass empty string to clear all injections
	err := client.ClearInjections("")
	if err != nil {
		return nil, fmt.Errorf("failed to clear all injections: %w", err)
	}

	return &InjectionResponse{
		Success: true,
		Message: "All injections cleared",
	}, nil
}

// clearDeviceInjections clears all injections on a specific device via IPC.
func clearDeviceInjections(device string) (*InjectionResponse, error) {
	client := getIPCClient()

	err := client.ClearInjections(device)
	if err != nil {
		return nil, fmt.Errorf("failed to clear device injections: %w", err)
	}

	return &InjectionResponse{
		Success: true,
		Message: fmt.Sprintf("Cleared all injections on %s", device),
	}, nil
}
