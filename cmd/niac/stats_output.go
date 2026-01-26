// Package main provides the NIAC command-line interface for network device simulation.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/krisarmstrong/niac-go/internal/logging"
	"github.com/krisarmstrong/niac-go/internal/protocols"
	"github.com/krisarmstrong/niac-go/internal/stats"
)

// printPeriodicStats prints periodic statistics.
func printPeriodicStats(stack *protocols.Stack, uptime time.Duration) {
	stats := stack.GetStats()
	neighbors := len(stack.GetNeighbors())

	fmt.Fprintf(os.Stdout,
		"[%s] Uptime: %s | Packets: RX=%d TX=%d | ARP: %d/%d | ICMP: %d/%d | "+
			"DNS: %d | DHCP: %d | Neighbors: %d\n",
		time.Now().Format("15:04:05"),
		formatDuration(uptime),
		stats.PacketsReceived,
		stats.PacketsSent,
		stats.ARPRequests,
		stats.ARPReplies,
		stats.ICMPRequests,
		stats.ICMPReplies,
		stats.DNSQueries,
		stats.DHCPRequests,
		neighbors,
	)
}

// printFinalStats prints final statistics on shutdown.
func printFinalStats(stack *protocols.Stack, uptime time.Duration) {
	stats := stack.GetStats()
	neighbors := len(stack.GetNeighbors())

	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "╔══════════════════════════════════════════════════════════════════╗")
	fmt.Fprintln(os.Stdout, "║                       Final Statistics                           ║")
	fmt.Fprintln(os.Stdout, "╠══════════════════════════════════════════════════════════════════╣")
	fmt.Fprintf(os.Stdout, "║ Total Uptime:        %-43s ║\n", formatDuration(uptime))
	fmt.Fprintln(os.Stdout, "║                                                                  ║")
	fmt.Fprintf(os.Stdout, "║ Packets Received:    %-10d                                    ║\n", stats.PacketsReceived)
	fmt.Fprintf(os.Stdout, "║ Packets Sent:        %-10d                                    ║\n", stats.PacketsSent)
	fmt.Fprintln(os.Stdout, "║                                                                  ║")
	fmt.Fprintf(os.Stdout, "║ ARP Requests:        %-10d                                    ║\n", stats.ARPRequests)
	fmt.Fprintf(os.Stdout, "║ ARP Replies:         %-10d                                    ║\n", stats.ARPReplies)
	fmt.Fprintf(os.Stdout, "║ ICMP Requests:       %-10d                                    ║\n", stats.ICMPRequests)
	fmt.Fprintf(os.Stdout, "║ ICMP Replies:        %-10d                                    ║\n", stats.ICMPReplies)
	fmt.Fprintf(os.Stdout, "║ DNS Queries:         %-10d                                    ║\n", stats.DNSQueries)
	fmt.Fprintf(os.Stdout, "║ DHCP Requests:       %-10d                                    ║\n", stats.DHCPRequests)
	fmt.Fprintf(os.Stdout, "║ Neighbors Learned:   %-10d                                    ║\n", neighbors)
	fmt.Fprintln(os.Stdout, "╚══════════════════════════════════════════════════════════════════╝")
	fmt.Fprintln(os.Stdout)
}

// formatDuration formats a [time.Duration] in a readable way.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%secondsPerMinute)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%secondsPerMinute)
}

// exportStatistics exports runtime statistics to JSON and/or CSV files (v1.19.0).
func exportStatistics(flags *legacyFlags, statsTracker *stats.Statistics) {
	if statsTracker == nil {
		return
	}

	// Update final statistics
	statsTracker.Update()

	// Export to JSON if requested
	if flags.exportStatsJSON != "" {
		if err := statsTracker.ExportJSON(flags.exportStatsJSON); err != nil {
			logging.Errorf("Failed to export statistics to JSON: %v", err)
		} else {
			logging.Infof("Statistics exported to JSON: %s", flags.exportStatsJSON)
		}
	}

	// Export to CSV if requested
	if flags.exportStatsCSV != "" {
		if err := statsTracker.ExportCSV(flags.exportStatsCSV); err != nil {
			logging.Errorf("Failed to export statistics to CSV: %v", err)
		} else {
			logging.Infof("Statistics exported to CSV: %s", flags.exportStatsCSV)
		}
	}
}
