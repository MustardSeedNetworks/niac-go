package main

import (
	"fmt"
	"os"

	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/logging"
	"github.com/spf13/cobra"
)

var interactiveOptions struct {
	debugLevel int
	verbose    bool
	quiet      bool
	noColor    bool
}

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
	Example: `  # Run interactive mode
  sudo niac interactive en0 config.yaml

  # Quick start with template
  niac template use router router.yaml
  sudo niac interactive en0 router.yaml

  # Controls during runtime:
  #   i - Interactive error injection menu
  #   q - Quit
  #   ↑↓ - Navigate devices`,
	Args: cobra.ExactArgs(2),
	Run:  runInteractive,
}

func init() {
	rootCmd.AddCommand(interactiveCmd)
	interactiveCmd.Flags().IntVarP(&interactiveOptions.debugLevel, "debug", "d", 1, "Debug level (0-3)")
	interactiveCmd.Flags().BoolVarP(&interactiveOptions.verbose, "verbose", "v", false, "Verbose output (equivalent to -d 3)")
	interactiveCmd.Flags().BoolVarP(&interactiveOptions.quiet, "quiet", "q", false, "Quiet mode (equivalent to -d 0)")
	interactiveCmd.Flags().BoolVar(&interactiveOptions.noColor, "no-color", false, "Disable colored output")
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

	debugLevel := interactiveOptions.debugLevel
	if interactiveOptions.verbose {
		debugLevel = 3
	}
	if interactiveOptions.quiet {
		debugLevel = 0
	}

	logging.InitColors(!interactiveOptions.noColor)
	debugConfig := logging.NewDebugConfig(debugLevel)

	// Start interactive mode
	if err := runInteractiveMode(interfaceName, cfg, debugConfig, configFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
