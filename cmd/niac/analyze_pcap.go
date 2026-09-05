package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/pcap"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/MustardSeedNetworks/niac-go/internal/packetdecode"
)

type pcapSummary struct {
	File        string            `json:"file"            yaml:"file"`
	Packets     int               `json:"packets"         yaml:"packets"`
	CapturedAt  time.Time         `json:"captured_at"     yaml:"captured_at"`
	ProtocolMap map[string]int    `json:"protocols"       yaml:"protocols"`
	Notes       map[string]string `json:"notes,omitempty" yaml:"notes,omitempty"`
	// PacketList is populated only with --packets. Omitted otherwise so the
	// summary output a script already parses is unchanged.
	PacketList []pcapPacketLine `json:"packetList,omitempty" yaml:"packetList,omitempty"`
}

type analyzePcapOptions struct {
	outputFormat string
	packets      bool
}

// pcapPacketLine is one row of the per-packet listing, the same fields the web
// packet list and the live inspector render.
type pcapPacketLine struct {
	Number   int    `json:"number"   yaml:"number"`
	Protocol string `json:"protocol" yaml:"protocol"`
	Source   string `json:"source"   yaml:"source"`
	Dest     string `json:"dest"     yaml:"dest"`
	Summary  string `json:"summary"  yaml:"summary"`
}

func addAnalyzePcapCommand(root *cobra.Command, _ *serviceOptions) {
	options := new(analyzePcapOptions)

	analyzePcapCmd := &cobra.Command{
		Use:   "analyze-pcap <pcap-file>",
		Short: "Summarise a packet capture by protocol",
		Long: `Parse a PCAP file and emit protocol counters for rapid troubleshooting.
The tool classifies packets into ARP, LLDP, CDP, STP, IPv4, IPv6, TCP, UDP,
and generic application protocols.`,
		Example: `  # Summarise a capture (text output)
  niac analyze-pcap capture.pcap

  # Machine-readable JSON output
  niac analyze-pcap --output json capture.pcap

  # YAML output (handy for diffing two captures)
  niac analyze-pcap --output yaml capture.pcap

  # Every packet, the same rows the web packet list shows
  niac analyze-pcap --packets capture.pcap`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runAnalyzePcap(args, options)
		},
	}

	analyzePcapCmd.Flags().StringVar(&options.outputFormat, "output", "text", "Output format (text, json, yaml)")
	analyzePcapCmd.Flags().BoolVar(&options.packets, "packets", false,
		"List every packet, the same rows the web packet list shows")

	root.AddCommand(analyzePcapCmd)
}

func runAnalyzePcap(args []string, options *analyzePcapOptions) error {
	summary, err := summarizePCAP(args[0], options.packets)
	if err != nil {
		return err
	}

	switch options.outputFormat {
	case "json":
		data, marshalErr := json.MarshalIndent(summary, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal JSON: %w", marshalErr)
		}

		_, _ = os.Stdout.WriteString(string(data) + "\n")
	case "yaml":
		data, yamlErr := yaml.Marshal(summary)
		if yamlErr != nil {
			return fmt.Errorf("failed to marshal YAML: %w", yamlErr)
		}

		_, _ = os.Stdout.WriteString(string(data) + "\n")
	default:
		fmt.Fprintf(os.Stdout, "File: %s\nPackets: %d\n", summary.File, summary.Packets)
		for proto, count := range summary.ProtocolMap {
			fmt.Fprintf(os.Stdout, "  %s: %d\n", proto, count)
		}
		printPacketLines(summary.PacketList)
	}
	return nil
}

// summarizePCAP counts protocols in a capture and, with list set, records one
// row per packet.
//
// Classification is internal/packetdecode, the same decoder the live inspector
// and the web analyzer use. It used to be a third, private one here that tested
// each layer independently, so a single HTTP segment incremented IPv4 *and* TCP
// and the totals never summed to the packet count -- and the protocol names
// disagreed with both UIs for anything it did not know.
func summarizePCAP(filename string, list bool) (*pcapSummary, error) {
	handle, err := pcap.OpenOffline(filename)
	if err != nil {
		return nil, fmt.Errorf("open pcap: %w", err)
	}
	defer handle.Close()

	source := gopacket.NewPacketSource(handle, handle.LinkType())

	summary := new(pcapSummary)
	summary.File = filename
	summary.CapturedAt = time.Now().UTC()
	summary.ProtocolMap = make(map[string]int)

	for packet := range source.Packets() {
		summary.Packets++

		decoded := map[string]any{"protocol": "Unknown", "summary": ""}
		packetdecode.Enrich(decoded, packet.Data())

		protocol, _ := decoded["protocol"].(string)
		summary.ProtocolMap[protocol]++

		if !list {
			continue
		}
		source, _ := decoded["source_ip"].(string)
		dest, _ := decoded["dest_ip"].(string)
		text, _ := decoded["summary"].(string)
		summary.PacketList = append(summary.PacketList, pcapPacketLine{
			Number:   summary.Packets,
			Protocol: protocol,
			Source:   source,
			Dest:     dest,
			Summary:  text,
		})
	}

	if info, statErr := os.Stat(filename); statErr == nil {
		summary.CapturedAt = info.ModTime()
	}

	return summary, nil
}

// printPacketLines renders the per-packet rows under the protocol counters.
func printPacketLines(lines []pcapPacketLine) {
	if len(lines) == 0 {
		return
	}

	fmt.Fprintf(os.Stdout, "\nPackets:\n")
	for _, line := range lines {
		fmt.Fprintf(os.Stdout, "  %5d  %-10s %-15s -> %-15s %s\n",
			line.Number, line.Protocol, orDash(line.Source), orDash(line.Dest), line.Summary)
	}
}

// orDash keeps the columns aligned for a frame with no network layer.
func orDash(value string) string {
	if value == "" {
		return "-"
	}

	return value
}
