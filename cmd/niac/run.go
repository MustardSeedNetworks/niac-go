package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

type runOptions struct {
	debugLevel int
	verbose    bool
	quiet      bool
	noColor    bool
	tui        bool
	web        bool
	webPort    string
	dryRun     bool
}

func addRunCommand(root *cobra.Command, services *serviceOptions, info versionInfo) {
	options := new(runOptions)

	runCmd := &cobra.Command{
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
		Args: cobra.ExactArgs(argsCountTwo),
		RunE: func(_ *cobra.Command, args []string) error {
			return runSimulation(args, options, services, info)
		},
	}

	runCmd.Flags().IntVarP(&options.debugLevel, "debug", "d", 1, "Debug level (0-3)")
	runCmd.Flags().BoolVarP(&options.verbose, "verbose", "v", false, "Verbose output (equivalent to -d 3)")
	runCmd.Flags().BoolVarP(&options.quiet, "quiet", "q", false, "Quiet mode (equivalent to -d 0)")
	runCmd.Flags().BoolVar(&options.noColor, "no-color", false, "Disable colored output")
	runCmd.Flags().BoolVar(&options.tui, "tui", false, "Enable interactive Terminal UI")
	runCmd.Flags().BoolVar(&options.web, "web", false, "Enable Web UI")
	runCmd.Flags().StringVar(&options.webPort, "port", "8080", "Web UI port (used with --web)")
	runCmd.Flags().BoolVarP(&options.dryRun, "dry-run", "n", false, "Validate config without starting simulation")

	root.AddCommand(runCmd)
}

func runSimulation(
	args []string,
	options *runOptions,
	services *serviceOptions,
	info versionInfo,
) error {
	interfaceName := args[0]
	configFile := args[1]

	// Determine debug level
	debugLevel := options.debugLevel
	if options.verbose {
		debugLevel = 3
	}
	if options.quiet {
		debugLevel = 0
	}

	logging.InitColors(!options.noColor)

	// Load configuration
	cfg, resolvedConfig, err := loadConfigOrScenario(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Dry run mode - just validate
	if options.dryRun {
		if entitlementErr := validateStoredSimulationConfig(cfg, loadFeatureChecker); entitlementErr != nil {
			return entitlementErr
		}
		logging.Successf("✓ Configuration valid: %s", configFile)
		logging.Infof("  Interface: %s", interfaceName)
		logging.Infof("  Devices: %d", len(cfg.Devices))
		return nil
	}

	// Set up API listen address if web is enabled
	if options.web {
		services.apiListen = ":" + options.webPort
	}

	debugConfig := logging.NewDebugConfig(debugLevel)

	// Print banner unless quiet
	if debugLevel > 0 {
		printBanner(info.version)
		logging.Infof("Interface: %s", interfaceName)
		logging.Infof("Config: %s (%d devices)", resolvedConfig, len(cfg.Devices))
		if options.web {
			logging.Infof("Web UI: http://localhost:%s", options.webPort)
		}
		if options.tui {
			logging.Infof("TUI: Enabled")
		}
		fmt.Fprintln(os.Stdout)
	}

	// Validate interface
	if err = validateInterface(interfaceName); err != nil {
		return err
	}

	// Run in appropriate mode
	if options.tui {
		return runInteractiveMode(interfaceName, cfg, debugConfig, resolvedConfig, services)
	}
	return runNormalMode(interfaceName, cfg, debugConfig, resolvedConfig, services)
}
