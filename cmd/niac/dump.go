package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/niac-go/internal/ipc"
)

// dumpOptions holds command-line options for the dump command.
type dumpOptions struct {
	device     string
	iface      string
	count      int
	socketPath string
	jsonOutput bool
}

func addDumpCommand(root *cobra.Command, _ *serviceOptions) {
	options := new(dumpOptions)

	dumpCmd := &cobra.Command{
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
		RunE: func(_ *cobra.Command, _ []string) error {
			return runDump(options)
		},
	}

	dumpCmd.Flags().StringVar(&options.device, "device", "", "Filter by device name")
	dumpCmd.Flags().StringVar(&options.iface, "interface", "", "Filter by interface name")
	dumpCmd.Flags().IntVar(&options.count, "count", 0, "Maximum number of packets to display (0 = all)")
	dumpCmd.Flags().StringVar(&options.socketPath, "socket", "", "Path to IPC socket (default: /tmp/niac.sock)")
	dumpCmd.Flags().BoolVar(&options.jsonOutput, "json", false, "Output packets as JSON")

	root.AddCommand(dumpCmd)
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
func runDump(options *dumpOptions) error {
	// Determine socket path
	socketPath := options.socketPath
	if socketPath == "" {
		socketPath = ipc.DefaultSocketPath()
	}

	// Check if socket exists
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		if options.jsonOutput {
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
	packets, err := client.DumpPackets(options.device, options.iface, options.count)
	if err != nil {
		if options.jsonOutput {
			outputDumpJSON(map[string]any{
				"success": false,
				"error":   err.Error(),
			})
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(exitCodeError)
	}

	if len(packets) == 0 {
		if options.jsonOutput {
			outputDumpJSON(map[string]any{
				"success": true,
				"packets": []any{},
				"count":   0,
			})
		} else {
			fmt.Fprintln(os.Stdout, "No packets captured")
		}
		return nil
	}

	// Output based on format
	if options.jsonOutput {
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
		fmt.Fprintf(os.Stdout, "Packet #%d: %d bytes @ %s\n",
			i+1,
			pkt.Length,
			pkt.Timestamp.Format("15:04:05.000"))

		if pkt.Device != "" {
			fmt.Fprintf(os.Stdout, "  Device: %s", pkt.Device)
			if pkt.Interface != "" {
				fmt.Fprintf(os.Stdout, "  Interface: %s", pkt.Interface)
			}
			fmt.Fprintln(os.Stdout)
		}

		// Print hex dump
		fmt.Fprintln(os.Stdout, formatHexDump(pkt.Data))
		fmt.Fprintln(os.Stdout)
	}

	fmt.Fprintf(os.Stdout, "Total: %d packet(s)\n", len(packets))
}

// formatHexDump formats binary data as a hex dump (xxd-style).
func formatHexDump(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	var sb strings.Builder
	bytesPerLine := 16

	for offset := 0; offset < len(data); offset += bytesPerLine {
		end := min(offset+bytesPerLine, len(data))
		formatHexLine(&sb, data[offset:end], offset, bytesPerLine, offset+bytesPerLine < len(data))
	}

	return sb.String()
}

// formatHexLine formats a single line of hex dump output.
func formatHexLine(sb *strings.Builder, lineData []byte, offset, bytesPerLine int, hasMore bool) {
	_, _ = fmt.Fprintf(sb, "%08x: ", offset)

	hexStr := formatHexBytes(lineData, bytesPerLine)
	sb.WriteString(hexStr)
	sb.WriteString("  ")

	formatASCII(sb, lineData)

	if hasMore {
		sb.WriteString("\n")
	}
}

// formatHexBytes formats bytes as hex with spacing.
func formatHexBytes(data []byte, bytesPerLine int) string {
	hexPart := hex.EncodeToString(data)

	var hexFormatted strings.Builder
	for j := 0; j < len(hexPart); j += 2 {
		if j > 0 && j%4 == 0 {
			hexFormatted.WriteString(" ")
		}
		if j+2 <= len(hexPart) {
			hexFormatted.WriteString(hexPart[j : j+2])
		}
	}

	hexStr := hexFormatted.String()
	paddedLen := (bytesPerLine/hexCharsPerByte)*hexCharsPerByte + (bytesPerLine/4 - 1)
	for len(hexStr) < paddedLen {
		hexStr += " "
	}
	return hexStr
}

// formatASCII writes the ASCII representation of bytes.
func formatASCII(sb *strings.Builder, data []byte) {
	for _, b := range data {
		if b >= 32 && b < 127 {
			sb.WriteByte(b)
		} else {
			sb.WriteByte('.')
		}
	}
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
