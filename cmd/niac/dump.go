package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/ipc"
	"github.com/spf13/cobra"
)

// dumpOptions holds command-line options for the dump command.
var dumpOptions struct {
	device     string
	iface      string
	count      int
	socketPath string
	jsonOutput bool
}

var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Dump captured packets from a running NIAC simulation",
	Long: `Dump captured packets from a running NIAC simulation via IPC socket.

This command connects to a running NIAC simulation and retrieves
hex dumps of recently captured packets. The output format is similar
to the standard hexdump or xxd utilities.

Packets can be filtered by device name or interface name. Use the
--count flag to limit the number of packets returned.

Exit codes:
  0 - Success
  1 - Connection failed (socket not found or connection refused)
  2 - Error occurred (request failed, parse error, etc.)`,
	Example: `  # Dump all captured packets
  niac dump

  # Dump packets for a specific device
  niac dump --device router-1

  # Dump packets for a specific interface
  niac dump --interface eth0

  # Limit output to 10 packets
  niac dump --count 10

  # Output as JSON
  niac dump --json

  # Combine filters
  niac dump --device router-1 --interface eth0 --count 5

  # Use a custom socket path
  niac dump --socket /var/run/niac/niac.sock`,
	Args: cobra.NoArgs,
	RunE: runDump,
}

func init() {
	rootCmd.AddCommand(dumpCmd)
	dumpCmd.Flags().StringVar(&dumpOptions.device, "device", "", "Filter by device name")
	dumpCmd.Flags().StringVar(&dumpOptions.iface, "interface", "", "Filter by interface name")
	dumpCmd.Flags().IntVar(&dumpOptions.count, "count", 0, "Maximum number of packets to display (0 = all)")
	dumpCmd.Flags().StringVar(&dumpOptions.socketPath, "socket", "", "Path to IPC socket (default: /tmp/niac.sock)")
	dumpCmd.Flags().BoolVar(&dumpOptions.jsonOutput, "json", false, "Output packets as JSON")
}

// PacketDump represents a captured packet for display.
type PacketDump struct {
	Index     int       `json:"index"`
	Timestamp time.Time `json:"timestamp"`
	Length    int       `json:"length"`
	Device    string    `json:"device,omitempty"`
	Interface string    `json:"interface,omitempty"`
	Data      []byte    `json:"data"`
	HexDump   string    `json:"hex_dump,omitempty"`
}

// runDump executes the dump command.
func runDump(cmd *cobra.Command, args []string) error {
	// Determine socket path
	socketPath := dumpOptions.socketPath
	if socketPath == "" {
		socketPath = ipc.DefaultSocketPath()
	}

	// Check if socket exists
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		if dumpOptions.jsonOutput {
			outputDumpJSON(map[string]any{
				"success": false,
				"error":   "socket not found",
			})
		} else {
			fmt.Fprintf(os.Stderr, "Error: Socket not found: %s\n", socketPath)
			fmt.Fprintf(os.Stderr, "Is the NIAC simulation running?\n")
		}
		os.Exit(1)
	}

	// Create IPC client
	client := ipc.NewClient(socketPath)

	// Request packet dump
	packets, err := client.DumpPackets(dumpOptions.device, dumpOptions.iface, dumpOptions.count)
	if err != nil {
		if dumpOptions.jsonOutput {
			outputDumpJSON(map[string]any{
				"success": false,
				"error":   err.Error(),
			})
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(2)
	}

	if len(packets) == 0 {
		if dumpOptions.jsonOutput {
			outputDumpJSON(map[string]any{
				"success": true,
				"packets": []any{},
				"count":   0,
			})
		} else {
			fmt.Println("No packets captured")
		}
		return nil
	}

	// Output based on format
	if dumpOptions.jsonOutput {
		outputPacketsJSON(packets)
	} else {
		printPacketsHexDump(packets)
	}

	return nil
}

// printPacketsHexDump prints packets in hex dump format (similar to xxd).
func printPacketsHexDump(packets []ipc.PacketData) {
	for i, pkt := range packets {
		// Print packet header
		fmt.Printf("Packet #%d: %d bytes @ %s\n",
			i+1,
			pkt.Length,
			pkt.Timestamp.Format("15:04:05.000"))

		if pkt.Device != "" {
			fmt.Printf("  Device: %s", pkt.Device)
			if pkt.Interface != "" {
				fmt.Printf("  Interface: %s", pkt.Interface)
			}
			fmt.Println()
		}

		// Print hex dump
		fmt.Println(formatHexDump(pkt.Data))
		fmt.Println()
	}

	fmt.Printf("Total: %d packet(s)\n", len(packets))
}

// formatHexDump formats binary data as a hex dump (xxd-style).
//nolint:gocognit // Hex dump formatting requires byte-by-byte processing
func formatHexDump(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	var sb strings.Builder
	bytesPerLine := 16

	for offset := 0; offset < len(data); offset += bytesPerLine {
		// Address column
		sb.WriteString(fmt.Sprintf("%08x: ", offset))

		// Hex bytes
		end := min(offset+bytesPerLine, len(data))

		hexPart := hex.EncodeToString(data[offset:end])

		// Add spaces between byte pairs
		var hexFormatted strings.Builder
		for j := 0; j < len(hexPart); j += 2 {
			if j > 0 && j%4 == 0 {
				hexFormatted.WriteString(" ")
			}
			if j+2 <= len(hexPart) {
				hexFormatted.WriteString(hexPart[j : j+2])
			}
		}

		// Pad hex part if needed
		hexStr := hexFormatted.String()
		paddedLen := (bytesPerLine/2)*2 + (bytesPerLine/4 - 1) // 8 pairs + 3 spaces
		for len(hexStr) < paddedLen {
			hexStr += " "
		}
		sb.WriteString(hexStr)

		sb.WriteString("  ")

		// ASCII representation
		for j := offset; j < end; j++ {
			b := data[j]
			if b >= 32 && b < 127 {
				sb.WriteByte(b)
			} else {
				sb.WriteByte('.')
			}
		}

		if offset+bytesPerLine < len(data) {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// outputPacketsJSON outputs packets as formatted JSON.
func outputPacketsJSON(packets []ipc.PacketData) {
	result := make([]PacketDump, len(packets))
	for i, pkt := range packets {
		result[i] = PacketDump{
			Index:     i + 1,
			Timestamp: pkt.Timestamp,
			Length:    pkt.Length,
			Device:    pkt.Device,
			Interface: pkt.Interface,
			Data:      pkt.Data,
			HexDump:   hex.EncodeToString(pkt.Data),
		}
	}

	output := map[string]any{
		"success": true,
		"packets": result,
		"count":   len(packets),
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(output)
}

// outputDumpJSON outputs data as formatted JSON.
func outputDumpJSON(data any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(data)
}
