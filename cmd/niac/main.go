// Package main provides the NIAC command-line interface for network device simulation.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/niac-go/internal/logging"
	"github.com/krisarmstrong/niac-go/internal/stats"
)

// CLI constants for cobra commands and common values.
const (
	// Argument counts for cobra commands.
	argsCountTwo   = 2
	argsCountThree = 3

	// Exit codes.
	exitCodeError = 2

	// Time constants.
	secondsPerMinute    = 60
	shortTimeout        = 5   // seconds
	tickerInterval      = 2   // seconds
	statsTickerInterval = 10  // seconds for stats reporting
	httpReadTimeout     = 15  // seconds for HTTP read timeout
	logPollMilliseconds = 500 // milliseconds for log polling
	maxLogEntries       = 100
	maxLogWidth         = 500

	// Display widths.
	lineWidthStandard = 80
	lineWidthWide     = 90
	tabPadding        = 2
	colWidthMAC       = 18
	colWidthIP        = 15
	colWidthType      = 17
	colWidthVendor    = 8
	colWidthHelp      = 51

	// Network constants.
	privateIPClassA   = 10
	privateIPClassB   = 172
	privateIPClassC   = 192
	bitShiftOctet     = 8
	defaultMTU        = 1514
	legacyPprofPort   = 6060
	legacyDefaultSecs = 60

	// Other constants.
	protocolCapacity      = 9
	templatePadOffset     = 2
	minPageLen            = 20
	maxDeviceCount        = 20 // maximum devices in generated config
	maxPercentage         = 100
	millisecondsThreshold = 1000
	randomBound           = 10 // for rand.Intn
	baseIPOffset          = 10 // offset for generated device IPs
	cidrParts             = 2  // IP/mask CIDR notation parts
	minArgsForConfig      = 2  // minimum arguments for config operations
	minSaltLen            = 5  // minimum salt length for hashing
	hexCharsPerByte       = 2  // 2 hex characters per byte
	ipRegexParts          = 5  // full match + 4 IP octets in regex

	// Debug level constants.
	debugLevelQuiet   = 0
	debugLevelNormal  = 1
	debugLevelVerbose = 2
	debugLevelDebug   = 3
)

func main() {
	info := readVersionInfo()
	services := new(serviceOptions)

	builders := []func(*cobra.Command, *serviceOptions){
		func(root *cobra.Command, services *serviceOptions) { addRunCommand(root, services, info) },
		func(root *cobra.Command, _ *serviceOptions) { addCompletionCommand(root) },
		addAnalyzeCommand,
		addAnalyzePcapCommand,
		addConfigCommand,
		func(root *cobra.Command, _ *serviceOptions) { addDaemonCommand(root, info) },
		addDumpCommand,
		addInitCommand,
		addInjectCommand,
		addInteractiveCommand,
		addLogsCommand,
		func(root *cobra.Command, _ *serviceOptions) { addManCommand(root, info) },
		addMonitorCommand,
		addNeighborsCommand,
		addSanitizeCommand,
		addServiceCommand,
		addStatusCommand,
		addTemplateCommand,
		addTopologyCommand,
		addValidateCommand,
	}

	rootCmd := newRootCommand(
		info,
		services,
		func(args []string) { runLegacyMode(args, info, services) },
		builders,
	)
	executeRootCommand(rootCmd)
}

// runLegacyMode maintains backward compatibility with original command-line interface
// Refactored into smaller, testable functions.
func runLegacyMode(osArgs []string, info versionInfo, services *serviceOptions) {
	flagSet := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	var flags legacyFlags
	defineLegacyFlags(flagSet, &flags)
	flagSet.Usage = printUsage
	// Parse the provided arguments (skip first element which is program name)
	if len(osArgs) > 1 {
		_ = flagSet.Parse(osArgs[1:])
	} else {
		_ = flagSet.Parse(nil)
	}

	// Process flag overrides (verbose/quiet)
	processFlags(&flags)
	applyLegacyServiceFlags(&flags, services)

	// Initialize colors (respects --no-color flag and NO_COLOR env var)
	logging.InitColors(!flags.noColor)

	// Get remaining arguments
	args := flagSet.Args()

	// Handle informational flags (version, list-interfaces, list-devices)
	if handleInformationalFlags(&flags, args, info) {
		exitWithStats(0, &flags, nil)
	}

	// Validate required arguments
	interfaceName, configFile, err := validateLegacyArguments(args)
	if err != nil {
		printUsage()
		exitWithStats(1, &flags, nil)
	}

	// Start profiling server if enabled
	if flags.enableProfiling {
		startProfilingServer(flags.profilePort, flags.debugLevel)
	}

	// Print banner (unless quiet)
	if flags.debugLevel > 0 {
		printBanner(info.version)
	}

	// Validate interface exists
	if err = validateInterface(interfaceName); err != nil {
		exitWithStats(exitCodeError, &flags, nil)
	}

	// Load configuration
	cfg, err := loadAndPrintConfig(configFile, interfaceName, &flags)
	if err != nil {
		logging.Errorf("%v", err)
		exitWithStats(1, &flags, nil)
	}

	// Handle dry run mode
	if flags.dryRun {
		runDryRunValidation(configFile, interfaceName, cfg)
		// runDryRunValidation calls os.Exit, so this line is unreachable
	}

	// Create debug configuration
	debugConfig := setupDebugConfig(&flags)

	// Initialize global statistics (v1.19.0)
	statsTracker := stats.NewStatistics(interfaceName, configFile, info.version)
	statsTracker.SetDeviceCount(len(cfg.Devices))

	// Count SNMP-enabled devices
	snmpCount := 0
	for _, dev := range cfg.Devices {
		if dev.SNMPConfig.Community != "" || dev.SNMPConfig.WalkFile != "" {
			snmpCount++
		}
	}
	statsTracker.SetSNMPDeviceCount(snmpCount)

	// Start simulation based on mode
	if flags.interactiveMode {
		if runErr := runInteractiveMode(interfaceName, cfg, debugConfig, configFile, services); runErr != nil {
			fmt.Fprintf(os.Stdout, "Error: %v\n", runErr)
			exitWithStats(1, &flags, statsTracker)
		}
	} else {
		if runErr := runNormalMode(interfaceName, cfg, debugConfig, configFile, services); runErr != nil {
			fmt.Fprintf(os.Stdout, "Error: %v\n", runErr)
			exitWithStats(1, &flags, statsTracker)
		}
	}
}

// startProfilingServer starts the pprof HTTP server for performance profiling.
// Security: Uses a dedicated mux to avoid polluting the default mux,
// and binds to localhost only to prevent external access.
func startProfilingServer(port int, debugLevel int) {
	// Security: bind to localhost only to prevent external access
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	// Create a dedicated mux for pprof endpoints (not using default mux)
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))

	go func() {
		if debugLevel >= 1 {
			logging.Infof("Starting pprof server on http://%s/debug/pprof/", addr)
			logging.Infof("  CPU profile:    http://%s/debug/pprof/profile?seconds=30", addr)
			logging.Infof("  Heap profile:   http://%s/debug/pprof/heap", addr)
			logging.Infof("  Goroutines:     http://%s/debug/pprof/goroutine", addr)
			logging.Warningf("Profiling server is for local development only - do not expose publicly")
			fmt.Fprintln(os.Stdout)
		}

		// Start HTTP server with timeouts using dedicated mux
		server := &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  httpReadTimeout * time.Second,
			WriteTimeout: httpReadTimeout * time.Second,
			IdleTimeout:  secondsPerMinute * time.Second,
		}
		if err := server.ListenAndServe(); err != nil {
			logging.Errorf("Failed to start pprof server: %v", err)
		}
	}()
}
