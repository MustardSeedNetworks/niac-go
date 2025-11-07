package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "v1.10.0"
	commit  = "dev"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "niac",
	Short: "Network In A Can - Network device simulator",
	Long: `NIAC (Network In A Can) simulates network devices on a local interface.

It responds to ARP, ICMP, LLDP, CDP, SNMP, HTTP, and other protocols,
making simulated devices appear real on the network.

Perfect for testing network management systems, monitoring tools,
and network discovery without physical hardware.`,
	Version: version,
	Run: func(cmd *cobra.Command, args []string) {
		// If no args, show help
		if len(args) == 0 {
			cmd.Help()
			return
		}
		// Legacy mode: if args provided and no subcommand matched, run main simulation
		// This maintains backward compatibility with: niac <interface> <config>
		// Prepend program name to match os.Args format expected by runLegacyMode
		legacyArgs := append([]string{os.Args[0]}, args...)
		runLegacyMode(legacyArgs)
	},
}

func init() {
	rootCmd.SetVersionTemplate(fmt.Sprintf("niac %s (commit: %s, built: %s)\n", version, commit, date))
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
