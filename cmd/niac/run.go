package main

import (
	"fmt"

	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/logging"
	"github.com/spf13/cobra"
)

var runOptions struct {
	debugLevel int
	verbose    bool
	quiet      bool
	noColor    bool
	tui        bool
	web        bool
	webPort    string
	dryRun     bool
}

var runCmd = &cobra.Command{
	Use:   "run <interface> <config-file>",
	Short: "Run network simulation",
	Long: `Run NIAC network simulation with optional TUI and/or WebUI.

By default, runs in headless mode. Add --tui for interactive terminal UI,
--web for browser-based UI, or both for full control.`,
	Example: `  # Headless simulation
  sudo niac run en0 config.yaml

  # With Terminal UI (TUI)
  sudo niac run en0 config.yaml --tui

  # With Web UI on port 8080
  sudo niac run en0 config.yaml --web

  # With both TUI and Web UI
  sudo niac run en0 config.yaml --tui --web

  # Web UI on custom port
  sudo niac run en0 config.yaml --web --port 9000

  # Validate config without running
  niac run en0 config.yaml --dry-run`,
	Args: cobra.ExactArgs(2),
	RunE: runSimulation,
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().IntVarP(&runOptions.debugLevel, "debug", "d", 1, "Debug level (0-3)")
	runCmd.Flags().BoolVarP(&runOptions.verbose, "verbose", "v", false, "Verbose output (equivalent to -d 3)")
	runCmd.Flags().BoolVarP(&runOptions.quiet, "quiet", "q", false, "Quiet mode (equivalent to -d 0)")
	runCmd.Flags().BoolVar(&runOptions.noColor, "no-color", false, "Disable colored output")
	runCmd.Flags().BoolVar(&runOptions.tui, "tui", false, "Enable interactive Terminal UI")
	runCmd.Flags().BoolVar(&runOptions.web, "web", false, "Enable Web UI")
	runCmd.Flags().StringVar(&runOptions.webPort, "port", "8080", "Web UI port (used with --web)")
	runCmd.Flags().BoolVarP(&runOptions.dryRun, "dry-run", "n", false, "Validate config without starting simulation")
}

func runSimulation(cmd *cobra.Command, args []string) error {
	interfaceName := args[0]
	configFile := args[1]

	// Determine debug level
	debugLevel := runOptions.debugLevel
	if runOptions.verbose {
		debugLevel = 3
	}
	if runOptions.quiet {
		debugLevel = 0
	}

	logging.InitColors(!runOptions.noColor)

	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Dry run mode - just validate
	if runOptions.dryRun {
		logging.Success("✓ Configuration valid: %s", configFile)
		logging.Info("  Interface: %s", interfaceName)
		logging.Info("  Devices: %d", len(cfg.Devices))
		return nil
	}

	// Set up API listen address if web is enabled
	if runOptions.web {
		servicesOpts.apiListen = ":" + runOptions.webPort
	}

	debugConfig := logging.NewDebugConfig(debugLevel)

	// Print banner unless quiet
	if debugLevel > 0 {
		printBanner()
		logging.Info("Interface: %s", interfaceName)
		logging.Info("Config: %s (%d devices)", configFile, len(cfg.Devices))
		if runOptions.web {
			logging.Info("Web UI: http://localhost:%s", runOptions.webPort)
		}
		if runOptions.tui {
			logging.Info("TUI: Enabled")
		}
		fmt.Println()
	}

	// Validate interface
	if err := validateInterface(interfaceName); err != nil {
		return err
	}

	// Run in appropriate mode
	if runOptions.tui {
		return runInteractiveMode(interfaceName, cfg, debugConfig, configFile)
	}
	return runNormalMode(interfaceName, cfg, debugConfig, configFile)
}
