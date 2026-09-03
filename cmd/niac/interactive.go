package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

type interactiveOptions struct {
	debugLevel int
	verbose    bool
	quiet      bool
	noColor    bool
}

func addInteractiveCommand(root *cobra.Command, services *serviceOptions) {
	options := new(interactiveOptions)

	interactiveCmd := &cobra.Command{
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
		Args: cobra.ExactArgs(argsCountTwo),
		RunE: func(_ *cobra.Command, args []string) error {
			return runInteractive(args, options, services)
		},
	}

	interactiveCmd.Flags().IntVarP(&options.debugLevel, "debug", "d", 1, "Debug level (0-3)")
	interactiveCmd.Flags().BoolVarP(&options.verbose, "verbose", "v", false, "Verbose output (equivalent to -d 3)")
	interactiveCmd.Flags().BoolVarP(&options.quiet, "quiet", "q", false, "Quiet mode (equivalent to -d 0)")
	interactiveCmd.Flags().BoolVar(&options.noColor, "no-color", false, "Disable colored output")

	root.AddCommand(interactiveCmd)
}

func runInteractive(args []string, options *interactiveOptions, services *serviceOptions) error {
	interfaceName := args[0]
	configFile := args[1]

	cfg, resolvedConfig, err := loadConfigOrScenario(configFile)
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}

	debugLevel := options.debugLevel
	if options.verbose {
		debugLevel = 3
	}
	if options.quiet {
		debugLevel = 0
	}

	logging.InitColors(!options.noColor)
	debugConfig := logging.NewDebugConfig(debugLevel)

	return runInteractiveMode(interfaceName, cfg, debugConfig, resolvedConfig, services)
}
