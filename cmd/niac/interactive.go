package main

import (
	"fmt"
	"os"

	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/interactive"
	"github.com/krisarmstrong/niac-go/pkg/logging"
	"github.com/spf13/cobra"
)

var interactiveCmd = &cobra.Command{
	Use:   "interactive <interface> <config-file>",
	Short: "Run NIAC in interactive TUI mode",
	Long: `Run NIAC with an interactive Terminal User Interface (TUI).

The TUI provides:
- Real-time device monitoring
- Live statistics and packet counts
- Interactive error injection (press 'i')
- Device status visualization
- Keyboard controls (q to quit)`,
	Args: cobra.ExactArgs(2),
	Run:  runInteractive,
}

func init() {
	rootCmd.AddCommand(interactiveCmd)
}

func runInteractive(cmd *cobra.Command, args []string) {
	interfaceName := args[0]
	configFile := args[1]

	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		logging.Error("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	// Start interactive mode
	if err := interactive.Run(interfaceName, cfg, nil); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
