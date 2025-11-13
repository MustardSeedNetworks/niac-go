package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type pcapSummary struct {
	File        string            `json:"file" yaml:"file"`
	Packets     int               `json:"packets" yaml:"packets"`
	CapturedAt  time.Time         `json:"captured_at" yaml:"captured_at"`
	ProtocolMap map[string]int    `json:"protocols" yaml:"protocols"`
	Notes       map[string]string `json:"notes,omitempty" yaml:"notes,omitempty"`
}

var analyzePcapCmd = &cobra.Command{
	Use:   "analyze-pcap <pcap-file>",
	Short: "Summarise a packet capture by protocol",
	Long: `Parse a PCAP file and emit protocol counters for rapid troubleshooting.
The tool classifies packets into ARP, LLDP, CDP, STP, IPv4, IPv6, TCP, UDP,
and generic application protocols.`,
	Args: cobra.ExactArgs(1),
	RunE: runAnalyzePcap,
}

func init() {
	rootCmd.AddCommand(analyzePcapCmd)
	analyzePcapCmd.Flags().String("output", "text", "Output format (text, json, yaml)")
}

func runAnalyzePcap(cmd *cobra.Command, args []string) error {
	outputFormat, _ := cmd.Flags().GetString("output")
	summary, err := summarizePCAP(args[0])
	if err != nil {
		return err
	}

	switch outputFormat {
	case "json":
		data, _ := json.MarshalIndent(summary, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		data, _ := yaml.Marshal(summary)
		fmt.Println(string(data))
	default:
		fmt.Printf("File: %s\nPackets: %d\n", summary.File, summary.Packets)
		for proto, count := range summary.ProtocolMap {
			fmt.Printf("  %s: %d\n", proto, count)
		}
	}
	return nil
}

func summarizePCAP(filename string) (*pcapSummary, error) {
	handle, err := pcap.OpenOffline(filename)
	if err != nil {
		return nil, fmt.Errorf("open pcap: %w", err)
	}
	defer handle.Close()

	source := gopacket.NewPacketSource(handle, handle.LinkType())

	summary := &pcapSummary{
		File:        filename,
		CapturedAt:  time.Now().UTC(),
		ProtocolMap: make(map[string]int),
	}

	for packet := range source.Packets() {
		summary.Packets++
		if layer := packet.Layer(layers.LayerTypeARP); layer != nil {
			summary.ProtocolMap["ARP"]++
			continue
		}
		if layer := packet.Layer(layers.LayerTypeLinkLayerDiscovery); layer != nil {
			summary.ProtocolMap["LLDP"]++
		}
		if layer := packet.Layer(layers.LayerTypeCiscoDiscovery); layer != nil {
			summary.ProtocolMap["CDP"]++
		}
		if layer := packet.Layer(layers.LayerTypeSTP); layer != nil {
			summary.ProtocolMap["STP"]++
		}
		if ip4 := packet.Layer(layers.LayerTypeIPv4); ip4 != nil {
			summary.ProtocolMap["IPv4"]++
		}
		if ip6 := packet.Layer(layers.LayerTypeIPv6); ip6 != nil {
			summary.ProtocolMap["IPv6"]++
		}
		if tcp := packet.Layer(layers.LayerTypeTCP); tcp != nil {
			summary.ProtocolMap["TCP"]++
		}
		if udp := packet.Layer(layers.LayerTypeUDP); udp != nil {
			summary.ProtocolMap["UDP"]++
		}
	}

	if _, err := os.Stat(filename); err == nil {
		if info, err := os.Stat(filename); err == nil {
			summary.CapturedAt = info.ModTime()
		}
	}
	return summary, nil
}
