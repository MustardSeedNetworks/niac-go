package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/niac-go/internal/ipc"
)

// NeighborEntry represents a neighbor record for display.
type NeighborEntry struct {
	Device    string    `json:"device"`
	Interface string    `json:"interface"`
	Neighbor  string    `json:"neighbor"`
	Port      string    `json:"port"`
	Protocol  string    `json:"protocol"`
	LastSeen  time.Time `json:"last_seen"`
}

type neighborsOptions struct {
	device     string
	protocol   string
	socketPath string
	jsonOutput bool
}

const (
	neighborsProtocolAll  = "all"
	neighborsProtocolLLDP = "lldp"
)

func addNeighborsCommand(root *cobra.Command, _ *serviceOptions) {
	options := new(neighborsOptions)

	neighborsCmd := &cobra.Command{
		Use:   "neighbors [watch]",
		Short: "Display neighbor discovery table from LLDP/CDP protocols",
		Long: `Display the neighbor discovery table from a running NIAC simulation.

This command shows network neighbors discovered via LLDP (Link Layer Discovery
Protocol) and CDP (Cisco Discovery Protocol) from simulated devices.

The table displays:
  - Device:    Local device name
  - Interface: Local interface where neighbor was discovered
  - Neighbor:  Remote device name/chassis ID
  - Protocol:  Discovery protocol (LLDP, CDP, EDP, FDP)
  - LastSeen:  When the neighbor was last seen

Without arguments, shows the current neighbor table snapshot.
Use the 'watch' subcommand for continuous live updates.`,
		Example: `  # Show current neighbors table
  niac neighbors

  # Show neighbors in JSON format
  niac neighbors --json

  # Filter by device
  niac neighbors --device router-1

  # Filter by protocol
  niac neighbors --protocol lldp

  # Watch for live updates
  niac neighbors watch

  # Watch with filters
  niac neighbors watch --device switch-1 --protocol cdp

  # Use custom socket path
  niac neighbors --socket /tmp/my-niac.sock`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNeighbors(cmd, args, options)
		},
	}

	neighborsWatchCmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch neighbor table for live updates",
		Long: `Watch the neighbor discovery table in real-time.

This subcommand continuously monitors the neighbor table and displays
updates as neighbors are discovered or expire. Similar to 'watch' command,
the display refreshes periodically to show current state.

Press Ctrl+C to stop watching.`,
		Example: `  # Watch all neighbors
  niac neighbors watch

  # Watch with JSON output
  niac neighbors watch --json

  # Watch specific device
  niac neighbors watch --device router-1

  # Watch only LLDP neighbors
  niac neighbors watch --protocol lldp`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNeighborsWatch(cmd, args, options)
		},
	}

	// Global flags for neighbors command
	neighborsCmd.PersistentFlags().StringVar(&options.device, "device", "",
		"Filter by device name")
	neighborsCmd.PersistentFlags().StringVar(&options.protocol, "protocol", "all",
		"Filter by protocol: lldp, cdp, or all")
	neighborsCmd.PersistentFlags().StringVar(&options.socketPath, "socket", "",
		"Path to IPC socket (default: "+ipc.DefaultSocketPath()+")")
	neighborsCmd.PersistentFlags().BoolVar(&options.jsonOutput, "json", false,
		"Output in JSON format")

	// Add watch subcommand
	neighborsCmd.AddCommand(neighborsWatchCmd)

	root.AddCommand(neighborsCmd)
}

func runNeighbors(cmd *cobra.Command, args []string, options *neighborsOptions) error {
	// Check if 'watch' was provided as an argument (backward compatibility)
	if len(args) == 1 && strings.ToLower(args[0]) == "watch" {
		return runNeighborsWatch(cmd, []string{}, options)
	}

	if len(args) > 0 {
		return fmt.Errorf("unknown argument: %s (use 'niac neighbors watch' for live updates)", args[0])
	}

	// Validate protocol flag
	if err := validateProtocolFlag(options.protocol); err != nil {
		return err
	}

	// Create IPC client
	client := getNeighborsClient(options)

	// Fetch neighbors
	neighbors, err := fetchNeighbors(client)
	if err != nil {
		return fmt.Errorf("failed to fetch neighbors: %w", err)
	}

	// Filter neighbors
	filtered := filterNeighbors(neighbors, options.device, options.protocol)

	// Output based on format
	if options.jsonOutput {
		return outputNeighborsJSON(filtered)
	}
	return outputNeighborsTable(filtered)
}

func runNeighborsWatch(_ *cobra.Command, _ []string, options *neighborsOptions) error {
	// Validate protocol flag
	err := validateProtocolFlag(options.protocol)
	if err != nil {
		return err
	}

	// Create IPC client
	client := getNeighborsClient(options)

	// Check if server is running
	if !client.IsRunning() {
		return fmt.Errorf("no NIAC simulation running (socket: %s)", client.SocketPath())
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

	// Run watch loop
	return runNeighborsWatchLoop(ctx, client, options)
}

func runNeighborsWatchLoop(ctx context.Context, client *ipc.Client, options *neighborsOptions) error {
	ticker := time.NewTicker(tickerInterval * time.Second)
	defer ticker.Stop()

	// Initial fetch
	err := displayNeighborsUpdate(client, options)
	if err != nil {
		return fmt.Errorf("failed to fetch initial neighbors: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			if !options.jsonOutput {
				fmt.Fprintln(os.Stdout, "\nNeighbors watch stopped.")
			}
			return nil
		case <-ticker.C:
			if updateErr := displayNeighborsUpdate(client, options); updateErr != nil {
				// Connection lost, but don't exit immediately
				if !options.jsonOutput {
					fmt.Fprintf(os.Stdout, "\r[%s] Connection lost: %v\n", time.Now().Format("15:04:05"), updateErr)
				} else {
					errObj := map[string]any{
						"error": updateErr.Error(),
						"time":  time.Now().Format(time.RFC3339),
					}
					jsonBytes, _ := json.Marshal(errObj)
					fmt.Fprintln(os.Stdout, string(jsonBytes))
				}
				continue
			}
		}
	}
}

func displayNeighborsUpdate(client *ipc.Client, options *neighborsOptions) error {
	neighbors, err := fetchNeighbors(client)
	if err != nil {
		return err
	}

	// Filter neighbors
	filtered := filterNeighbors(neighbors, options.device, options.protocol)

	if options.jsonOutput {
		// JSON Lines format for watch mode
		output := map[string]any{
			"timestamp": time.Now().Format(time.RFC3339),
			"count":     len(filtered),
			"neighbors": filtered,
		}
		jsonBytes, _ := json.Marshal(output)
		fmt.Fprintln(os.Stdout, string(jsonBytes))
	} else {
		// Clear screen and redraw table
		clearScreen()
		fmt.Fprintln(os.Stdout, "NIAC Neighbors Watch - Press Ctrl+C to stop")
		fmt.Fprintf(os.Stdout, "Last updated: %s\n", time.Now().Format("15:04:05"))
		if options.device != "" {
			fmt.Fprintf(os.Stdout, "Device filter: %s\n", options.device)
		}
		if options.protocol != neighborsProtocolAll && options.protocol != "" {
			fmt.Fprintf(os.Stdout, "Protocol filter: %s\n", strings.ToUpper(options.protocol))
		}
		fmt.Fprintln(os.Stdout, strings.Repeat("-", lineWidthWide))

		if tableErr := outputNeighborsTable(filtered); tableErr != nil {
			return tableErr
		}
	}
	return nil
}

func validateProtocolFlag(protocol string) error {
	protocol = strings.ToLower(protocol)
	switch protocol {
	case neighborsProtocolLLDP, "cdp", neighborsProtocolAll, "":
		return nil
	default:
		return fmt.Errorf("invalid protocol %q: must be lldp, cdp, or %s", protocol, neighborsProtocolAll)
	}
}

func getNeighborsClient(options *neighborsOptions) *ipc.Client {
	socketPath := options.socketPath
	if socketPath == "" {
		socketPath = ipc.GetDefaultSocketPath()
	}
	return ipc.NewClient(socketPath)
}

func fetchNeighbors(client *ipc.Client) ([]NeighborEntry, error) {
	neighbors, err := client.GetNeighbors()
	if err != nil {
		return nil, fmt.Errorf("failed to get neighbors: %w", err)
	}

	entries := make([]NeighborEntry, 0, len(neighbors))
	for _, n := range neighbors {
		entries = append(entries, NeighborEntry{
			Device:    n.LocalDevice,
			Interface: n.RemotePort,
			Neighbor:  n.RemoteDevice,
			Port:      n.RemotePort,
			Protocol:  n.Protocol,
			LastSeen:  n.LastSeen,
		})
	}
	return entries, nil
}

func filterNeighbors(neighbors []NeighborEntry, device, protocol string) []NeighborEntry {
	if device == "" && (protocol == "" || protocol == neighborsProtocolAll) {
		return neighbors
	}

	filtered := make([]NeighborEntry, 0, len(neighbors))
	for _, n := range neighbors {
		// Filter by device
		if device != "" && !strings.EqualFold(n.Device, device) {
			continue
		}

		// Filter by protocol
		if protocol != "" && protocol != neighborsProtocolAll {
			if !strings.EqualFold(n.Protocol, protocol) {
				continue
			}
		}

		filtered = append(filtered, n)
	}
	return filtered
}

func outputNeighborsTable(neighbors []NeighborEntry) error {
	if len(neighbors) == 0 {
		fmt.Fprintln(os.Stdout, "No neighbors discovered")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, tabPadding, ' ', 0)
	fmt.Fprintln(w, "DEVICE\tINTERFACE\tNEIGHBOR\tPROTOCOL\tLAST SEEN")
	fmt.Fprintln(w, "------\t---------\t--------\t--------\t---------")

	for _, n := range neighbors {
		lastSeen := formatLastSeen(n.LastSeen)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			n.Device, n.Interface, n.Neighbor, n.Protocol, lastSeen)
	}

	err := w.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush output: %w", err)
	}
	return nil
}

func outputNeighborsJSON(neighbors []NeighborEntry) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(neighbors)
	if err != nil {
		return fmt.Errorf("failed to encode neighbors: %w", err)
	}
	return nil
}

func formatLastSeen(t time.Time) string {
	if t.IsZero() {
		return "-"
	}

	since := time.Since(t)
	if since < time.Minute {
		return fmt.Sprintf("%ds ago", int(since.Seconds()))
	}
	if since < time.Hour {
		return fmt.Sprintf("%dm ago", int(since.Minutes()))
	}
	if since < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(since.Hours()))
	}
	return t.Format("2006-01-02 15:04")
}
