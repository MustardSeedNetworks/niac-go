package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/cliclient"
)

// dumpOptions holds command-line options for the dump command.
type dumpOptions struct {
	device     string
	iface      string
	count      int
	api        string
	caCert     string
	insecure   bool
	jsonOutput bool
	// pcapFile writes the daemon's retained frames to a pcapng file instead
	// of hex-dumping the live stream. The two are different reads: the stream
	// only carries what arrives while this command waits, the ring already
	// holds what happened before it started.
	pcapFile string
	session  string
	filter   string
}

const dumpLongHelp = `Dump packets from a running NIAC simulation, read from the daemon's live stream.

This command connects to a running NIAC simulation and retrieves
hex dumps of recently captured packets. The output format is similar
to the standard hexdump or xxd utilities.

Packets can be filtered by device name or interface name. Use the
--count flag to limit the number of packets returned.

Exit codes:
  0 - Success
  1 - Connection failed (no daemon answered)
  2 - Error occurred (request failed, parse error, etc.)`

const dumpExample = `  # Dump all captured packets
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

  # Read from a daemon on another address
  niac dump --api https://10.0.0.5:8445

  # Save the session's retained frames as pcapng for Wireshark
  niac dump --session hospital --pcap /tmp/hospital.pcapng

  # Only the ARP frames, newest 200
  niac dump --session hospital --pcap arp.pcapng --filter arp --count 200`

func addDumpCommand(root *cobra.Command, _ *serviceOptions) {
	options := new(dumpOptions)
	dumpCmd := &cobra.Command{
		Use:     "dump",
		Short:   "Dump captured packets from a running NIAC simulation",
		Long:    dumpLongHelp,
		Example: dumpExample,
		Args:    cobra.NoArgs,
		RunE:    func(cmd *cobra.Command, _ []string) error { return runDump(cmd.Context(), options) },
	}
	addDumpFlags(dumpCmd, options)
	root.AddCommand(dumpCmd)
}

func addDumpFlags(cmd *cobra.Command, options *dumpOptions) {
	cmd.Flags().StringVar(&options.device, "device", "", "Filter by device name")
	cmd.Flags().StringVar(&options.iface, "interface", "", "Filter by interface name")
	cmd.Flags().IntVar(&options.count, "count", 0, "Maximum number of packets to display (0 = all)")
	cmd.Flags().StringVar(&options.api, "api", "",
		"Daemon API address (default: "+cliclient.DefaultBaseURL+", or NIAC_API_URL)")
	cmd.Flags().StringVar(&options.caCert, "cacert", "",
		"Daemon certificate to trust (default: the local daemon's own, when visible)")
	cmd.Flags().BoolVar(&options.insecure, "insecure", false,
		"Skip TLS verification, for a daemon whose certificate this host cannot see")
	cmd.Flags().BoolVar(&options.jsonOutput, "json", false, "Output packets as JSON")
	cmd.Flags().StringVar(&options.pcapFile, "pcap", "",
		"Write the session's retained frames to this pcapng file instead of hex-dumping the live stream")
	cmd.Flags().StringVar(&options.session, "session", "",
		"Session to export with --pcap (default: the only running session)")
	cmd.Flags().StringVar(&options.filter, "filter", "",
		"BPF expression applied to the exported frames (--pcap only)")
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

// runDump collects packets from the daemon's live stream.
//
// The daemon broadcasts frames as it handles them and keeps no capture buffer,
// so this waits for the requested count rather than asking for history that
// does not exist. A quiet simulation yields nothing, and says so.
func runDump(ctx context.Context, options *dumpOptions) error {
	client, err := newCLIClient(options.api, options.caCert, options.insecure)
	if err != nil {
		return handleDumpError(err, options.jsonOutput)
	}
	if options.pcapFile != "" {
		return runDumpPcap(ctx, client, options)
	}
	// The window bounds the stream read only. Anything after it - including
	// asking why nothing arrived - needs a context that has not just expired.
	streamCtx, cancel := context.WithTimeout(ctx, dumpWindow)
	defer cancel()

	packets := make([]cliclient.PacketData, 0, options.count)
	err = client.StreamPackets(streamCtx, func(event cliclient.PacketEvent) bool {
		if !matchesDumpFilter(event, options) {
			return true
		}
		raw, decodeErr := event.Bytes()
		if decodeErr != nil {
			return true
		}
		packets = append(packets, cliclient.PacketData{
			Timestamp: parsePacketTime(event.Timestamp),
			Length:    event.Size,
			Device:    event.Device,
			Interface: event.Direction,
			Data:      raw,
		})

		return len(packets) < options.count
	})
	if err != nil && streamCtx.Err() == nil {
		return handleDumpError(err, options.jsonOutput)
	}

	if len(packets) == 0 {
		explainEmptyStream(ctx, client, options.jsonOutput)
	}
	outputPackets(packets, options.jsonOutput)

	return nil
}

// runDumpPcap writes the session's retained frames to a pcapng file.
//
// This reads the daemon's ring rather than the live stream, so it returns
// what already happened instead of waiting for the next frame — the export a
// dump of a finished exchange actually needs.
func runDumpPcap(ctx context.Context, client *cliclient.Client, options *dumpOptions) error {
	sessionID, err := resolveDumpSession(ctx, client, options.session)
	if err != nil {
		return handleDumpError(err, options.jsonOutput)
	}

	file, err := os.OpenFile(options.pcapFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return handleDumpError(fmt.Errorf("create %s: %w", options.pcapFile, err), options.jsonOutput)
	}
	defer file.Close()

	written, err := client.ExportCapture(ctx, sessionID, cliclient.CaptureExportOptions{
		Filter: options.filter,
		Last:   options.count,
	}, file)
	if err != nil {
		return handleDumpError(err, options.jsonOutput)
	}
	if err = file.Close(); err != nil {
		return handleDumpError(fmt.Errorf("close %s: %w", options.pcapFile, err), options.jsonOutput)
	}

	if options.jsonOutput {
		outputDumpJSON(map[string]any{
			"success": true, "session": sessionID, "file": options.pcapFile, "bytes": written,
		})
	} else {
		fmt.Fprintf(os.Stdout, "Wrote %d bytes from session %s to %s\n",
			written, sessionID, options.pcapFile)
	}

	return nil
}

// resolveDumpSession picks the session to export. An explicit --session wins;
// otherwise a single running session is unambiguous and anything else is a
// question the operator has to answer, not one to guess at.
func resolveDumpSession(ctx context.Context, client *cliclient.Client, requested string) (string, error) {
	if requested != "" {
		return requested, nil
	}
	sessions, err := client.Sessions(ctx)
	if err != nil {
		return "", err
	}
	switch len(sessions) {
	case 0:
		return "", errors.New("no simulation is running, so there is nothing to export")
	case 1:
		return sessions[0].SessionID, nil
	default:
		names := make([]string, len(sessions))
		for i, session := range sessions {
			names[i] = session.SessionID
		}

		return "", fmt.Errorf("%d scenarios are running (%s); name one with --session",
			len(sessions), strings.Join(names, ", "))
	}
}

// explainEmptyStream says why nothing arrived when the reason is structural
// rather than a quiet network: the daemon broadcasts a packet on its unscoped
// stream only when one scenario owns the capture. With several running, their
// frames are scoped to each session and this stream stays silent no matter how
// busy they are - and "No packets captured" would read as a quiet lab.
func explainEmptyStream(ctx context.Context, client *cliclient.Client, jsonOutput bool) {
	sessions, err := client.Sessions(ctx)
	if err != nil || len(sessions) < 2 || jsonOutput {
		return
	}
	fmt.Fprintf(os.Stderr,
		"Note: %d scenarios are running, and their packets are scoped to each session.\n"+
			"The unscoped stream this command reads only carries frames when one scenario is up.\n",
		len(sessions))
}

// dumpWindow bounds how long a dump waits for its packets, so a quiet
// simulation returns instead of hanging.
const dumpWindow = 10 * time.Second

// matchesDumpFilter keeps the frames the operator asked for.
func matchesDumpFilter(event cliclient.PacketEvent, options *dumpOptions) bool {
	if options.device != "" && !strings.EqualFold(event.Device, options.device) {
		return false
	}

	return options.iface == "" || strings.EqualFold(event.Direction, options.iface)
}

// parsePacketTime reads the stream's timestamp, falling back to now when the
// daemon sends a shape this build does not know.
func parsePacketTime(value string) time.Time {
	for _, layout := range []string{"2006-01-02T15:04:05.000Z", time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}

	return time.Now()
}

func handleDumpError(err error, jsonOutput bool) error {
	if jsonOutput {
		outputDumpJSON(map[string]any{"success": false, "error": err.Error()})
	} else {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
	return withExitCode(exitCodeError, err)
}

func outputPackets(packets []cliclient.PacketData, jsonOutput bool) {
	if len(packets) == 0 {
		if jsonOutput {
			outputDumpJSON(map[string]any{"success": true, "packets": []any{}, "count": 0})
		} else {
			fmt.Fprintln(os.Stdout, "No packets captured")
		}
		return
	}
	if jsonOutput {
		outputPacketsJSON(packets)
	} else {
		printPacketsHexDump(packets)
	}
}

// printPacketsHexDump prints packets in hex dump format (similar to xxd).
func printPacketsHexDump(packets []cliclient.PacketData) {
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
func outputPacketsJSON(packets []cliclient.PacketData) {
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
