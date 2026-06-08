// Package main provides the NIAC command-line interface for network device simulation.
package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/niac-go/internal/capture"
	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/interactive"
	"github.com/krisarmstrong/niac-go/internal/logging"
	"github.com/krisarmstrong/niac-go/internal/protocols"
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
		addContentCommand,
		func(root *cobra.Command, _ *serviceOptions) { addDaemonCommand(root, info) },
		addDumpCommand,
		addInitCommand,
		addInjectCommand,
		func(root *cobra.Command, _ *serviceOptions) { addInstallCACommand(root) },
		addLicenseCommand,
		addListCommand,
		addInteractiveCommand,
		addLogsCommand,
		func(root *cobra.Command, _ *serviceOptions) { addManCommand(root, info) },
		addMibZipCommand,
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
	if shouldUseLegacyCommand(os.Args[1:], rootCmd) {
		runLegacyMode(os.Args, info, services)
		return
	}

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

func exitWithStats(code int, flags *legacyFlags, statsTracker *stats.Statistics) {
	if flags != nil && (flags.exportStatsJSON != "" || flags.exportStatsCSV != "") {
		exportStatistics(flags, statsTracker)
	}
	os.Exit(code)
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

func printBanner(version string) {
	fmt.Fprintf(os.Stdout, "╔══════════════════════════════════════════════════════════════════╗\n")
	fmt.Fprintf(os.Stdout, "║  NIAC - Network In A Can (Go Edition)                           ║\n")
	fmt.Fprintf(
		os.Stdout,
		"║  Version %s                                                 ║\n",
		padRight(version, colWidthHelp),
	)
	fmt.Fprintf(os.Stdout, "╚══════════════════════════════════════════════════════════════════╝\n")
	fmt.Fprintln(os.Stdout)
}

func printVersion(info versionInfo) {
	fmt.Fprintf(os.Stdout, "NIAC-Go version %s\n", info.version)
	fmt.Fprintf(os.Stdout, "Build commit: %s\n", info.commit)
	fmt.Fprintf(os.Stdout, "Build date: %s\n", info.date)
	fmt.Fprintf(os.Stdout, "Go version: %s\n", runtime.Version())
	fmt.Fprintf(os.Stdout, "OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Enhancements over Java version:")
	fmt.Fprintln(os.Stdout, "  • 10x-770x faster performance")
	fmt.Fprintln(os.Stdout, "  • 3.3x less code")
	fmt.Fprintln(os.Stdout, "  • Advanced HTTP server (multi-endpoint)")
	fmt.Fprintln(os.Stdout, "  • Complete FTP server (17 commands)")
	fmt.Fprintln(os.Stdout, "  • Advanced device simulation")
	fmt.Fprintln(os.Stdout, "  • Comprehensive traffic generation")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Original NIAC by Kevin Kayes (2002-2015)")
	fmt.Fprintln(os.Stdout, "Go rewrite by Kris Armstrong (2025)")
}

func printUsage() {
	printUsageHeader()
	printUsageOptions()
	printUsageProtocolDebug()
	printUsageDebugLevels()
	printUsageExamples()
	printUsageProfiling()
	fmt.Fprintln(os.Stdout, "For more information, see: https://github.com/krisarmstrong/niac-go")
}

func printUsageHeader() {
	fmt.Fprintln(os.Stdout, "USAGE:")
	fmt.Fprintln(os.Stdout, "  niac [OPTIONS] <interface> <config_file>")
	fmt.Fprintln(os.Stdout, "  niac --list-interfaces")
	fmt.Fprintln(os.Stdout, "  niac --version")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "REQUIRED ARGUMENTS:")
	fmt.Fprintln(os.Stdout, "  <interface>     Network interface to use (e.g., en0, eth0)")
	fmt.Fprintln(os.Stdout, "  <config_file>   Configuration file path (.cfg, .json, or .yaml)")
	fmt.Fprintln(os.Stdout)
}

func printUsageOptions() {
	fmt.Fprintln(os.Stdout, "OPTIONS:")
	fmt.Fprintln(os.Stdout, "  Core:")
	fmt.Fprintln(os.Stdout, "    -d, --debug <level>      Debug level (0-3) [default: 1]")
	fmt.Fprintln(os.Stdout, "                             0=quiet, 1=normal, 2=verbose, 3=debug")
	fmt.Fprintln(os.Stdout, "    -v, --verbose            Verbose output (equivalent to -d 3)")
	fmt.Fprintln(os.Stdout, "    -q, --quiet              Quiet mode (equivalent to -d 0)")
	fmt.Fprintln(os.Stdout, "    -i, --interactive        Enable interactive TUI mode")
	fmt.Fprintln(os.Stdout, "    -n, --dry-run            Validate configuration without starting")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "  Information:")
	fmt.Fprintln(os.Stdout, "    -V, --version            Show version information")
	fmt.Fprintln(os.Stdout, "    -l, --list-interfaces    List available network interfaces")
	fmt.Fprintln(os.Stdout, "        --list-devices       List devices in configuration file")
	fmt.Fprintln(os.Stdout, "    -h, --help               Show this help message")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "  Output:")
	fmt.Fprintln(os.Stdout, "        --no-color           Disable colored output")
	fmt.Fprintln(os.Stdout, "        --log-file <file>    Write log to file")
	fmt.Fprintln(os.Stdout, "        --stats-interval <n> Statistics update interval [default: 1s]")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "  Advanced:")
	fmt.Fprintln(os.Stdout, "        --babble-interval <n>   Traffic generation interval [default: 60s]")
	fmt.Fprintln(os.Stdout, "        --no-traffic            Disable background traffic generation")
	fmt.Fprintln(os.Stdout, "        --snmp-community <str>  Default SNMP community string")
	fmt.Fprintln(os.Stdout, "        --max-packet-size <n>   Maximum packet size [default: 1514]")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "  Performance Profiling:")
	fmt.Fprintln(os.Stdout, "    -p, --profile            Enable pprof performance profiling")
	fmt.Fprintln(os.Stdout, "        --profile-port <port>   Port for pprof HTTP server [default: 6060]")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "  Statistics Export:")
	fmt.Fprintln(os.Stdout, "        --export-stats-json <file>  Export runtime statistics to JSON file on exit")
	fmt.Fprintln(os.Stdout, "        --export-stats-csv <file>   Export runtime statistics to CSV file on exit")
	fmt.Fprintln(os.Stdout)
}

func printUsageProtocolDebug() {
	fmt.Fprintln(os.Stdout, "  Per-Protocol Debug Levels:")
	fmt.Fprintln(os.Stdout, "        --debug-arp <level>     ARP protocol debug level (0-3)")
	fmt.Fprintln(os.Stdout, "        --debug-ip <level>      IP protocol debug level (0-3)")
	fmt.Fprintln(os.Stdout, "        --debug-icmp <level>    ICMP protocol debug level (0-3)")
	fmt.Fprintln(os.Stdout, "        --debug-ipv6 <level>    IPv6 protocol debug level (0-3)")
	fmt.Fprintln(os.Stdout, "        --debug-icmpv6 <level>  ICMPv6 protocol debug level (0-3)")
	fmt.Fprintln(os.Stdout, "        --debug-udp <level>     UDP protocol debug level (0-3)")
	fmt.Fprintln(os.Stdout, "        --debug-tcp <level>     TCP protocol debug level (0-3)")
	fmt.Fprintln(os.Stdout, "        --debug-dns <level>     DNS protocol debug level (0-3)")
	fmt.Fprintln(os.Stdout, "        --debug-dhcp <level>    DHCP protocol debug level (0-3)")
	fmt.Fprintln(os.Stdout, "        --debug-dhcpv6 <level>  DHCPv6 protocol debug level (0-3)")
	fmt.Fprintln(os.Stdout, "        --debug-http <level>    HTTP protocol debug level (0-3)")
	fmt.Fprintln(os.Stdout, "        --debug-ftp <level>     FTP protocol debug level (0-3)")
	fmt.Fprintln(os.Stdout, "        --debug-netbios <level> NetBIOS protocol debug level (0-3)")
	fmt.Fprintln(os.Stdout, "        --debug-stp <level>     STP protocol debug level (0-3)")
	fmt.Fprintln(os.Stdout, "        --debug-lldp <level>    LLDP protocol debug level (0-3)")
	fmt.Fprintln(os.Stdout, "        --debug-cdp <level>     CDP protocol debug level (0-3)")
	fmt.Fprintln(os.Stdout, "        --debug-edp <level>     EDP protocol debug level (0-3)")
	fmt.Fprintln(os.Stdout, "        --debug-fdp <level>     FDP protocol debug level (0-3)")
	fmt.Fprintln(os.Stdout, "        --debug-snmp <level>    SNMP protocol debug level (0-3)")
	fmt.Fprintln(os.Stdout)
}

func printUsageDebugLevels() {
	fmt.Fprintln(os.Stdout, "DEBUG LEVELS:")
	fmt.Fprintln(os.Stdout, "  0  QUIET   - Only critical errors")
	fmt.Fprintln(os.Stdout, "  1  NORMAL  - Status messages (default)")
	fmt.Fprintln(os.Stdout, "  2  VERBOSE - Protocol details")
	fmt.Fprintln(os.Stdout, "  3  DEBUG   - Full packet details")
	fmt.Fprintln(os.Stdout)
}

func printUsageExamples() {
	fmt.Fprintln(os.Stdout, "EXAMPLES:")
	fmt.Fprintln(os.Stdout, "  # List available interfaces")
	fmt.Fprintln(os.Stdout, "  niac --list-interfaces")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "  # Validate configuration")
	fmt.Fprintln(os.Stdout, "  niac --dry-run en0 network.cfg")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "  # Run in interactive mode with verbose debugging")
	fmt.Fprintln(os.Stdout, "  sudo niac --interactive --verbose en0 network.cfg")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "  # Run in quiet mode with log file")
	fmt.Fprintln(os.Stdout, "  sudo niac --quiet --log-file niac.log en0 network.cfg")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "  # Debug only DHCP protocol at verbose level")
	fmt.Fprintln(os.Stdout, "  sudo niac --debug 1 --debug-dhcp 3 en0 network.cfg")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "  # Show version")
	fmt.Fprintln(os.Stdout, "  niac --version")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "  # Enable profiling for performance analysis")
	fmt.Fprintln(os.Stdout, "  sudo niac --profile en0 network.cfg")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "  # Enable profiling on custom port")
	fmt.Fprintln(os.Stdout, "  sudo niac --profile --profile-port 8080 en0 network.cfg")
	fmt.Fprintln(os.Stdout)
}

func printUsageProfiling() {
	fmt.Fprintln(os.Stdout, "PROFILING:")
	fmt.Fprintln(os.Stdout, "  When --profile is enabled, pprof endpoints are available at:")
	fmt.Fprintln(os.Stdout, "    http://localhost:6060/debug/pprof/          - Index page")
	fmt.Fprintln(os.Stdout, "    http://localhost:6060/debug/pprof/profile   - CPU profile")
	fmt.Fprintln(os.Stdout, "    http://localhost:6060/debug/pprof/heap      - Memory profile")
	fmt.Fprintln(os.Stdout, "    http://localhost:6060/debug/pprof/goroutine - Goroutine profile")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "  Collect CPU profile (30 seconds):")
	fmt.Fprintln(os.Stdout, "    curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof")
	fmt.Fprintln(os.Stdout, "    go tool pprof cpu.prof")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "  Collect memory profile:")
	fmt.Fprintln(os.Stdout, "    curl http://localhost:6060/debug/pprof/heap > mem.prof")
	fmt.Fprintln(os.Stdout, "    go tool pprof mem.prof")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "  Interactive profiling:")
	fmt.Fprintln(os.Stdout, "    go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30")
	fmt.Fprintln(os.Stdout, "    go tool pprof http://localhost:6060/debug/pprof/heap")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "  WARNING: Profiling server binds to localhost only for security.")
	fmt.Fprintln(os.Stdout, "           Do not expose the profiling port on public networks.")
	fmt.Fprintln(os.Stdout)
}

func printDeviceList(configFile string) {
	cfg, err := config.Load(configFile)
	if err != nil {
		fmt.Fprintf(os.Stdout, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "Devices in %s:\n\n", configFile)

	if len(cfg.Devices) == 0 {
		fmt.Fprintln(os.Stdout, "No devices found in configuration.")
		return
	}

	// Print table header
	fmt.Fprintln(os.Stdout, "┌────────────────────┬─────────────────┬───────────────────┬──────────┬───────┐")
	fmt.Fprintln(os.Stdout, "│ Name               │ IP Address      │ MAC Address       │ Type     │ SNMP  │")
	fmt.Fprintln(os.Stdout, "├────────────────────┼─────────────────┼───────────────────┼──────────┼───────┤")

	// Print devices
	for _, device := range cfg.Devices {
		ipAddr := "N/A"
		if len(device.IPAddresses) > 0 {
			ipAddr = device.IPAddresses[0].String()
			// Indicate if device has multiple IPs
			if len(device.IPAddresses) > 1 {
				ipAddr = ipAddr + " +" + strconv.Itoa(len(device.IPAddresses)-1)
			}
		}

		macAddr := "N/A"
		if len(device.MACAddress) > 0 {
			macAddr = device.MACAddress.String()
		}

		deviceType := device.Type
		if deviceType == "" {
			deviceType = "generic"
		}

		snmp := "No"
		if device.SNMPConfig.Community != "" || device.SNMPConfig.WalkFile != "" {
			snmp = "Yes"
		}

		fmt.Fprintf(os.Stdout, "│ %-18s │ %-15s │ %-17s │ %-8s │ %-5s │\n",
			padRight(device.Name, colWidthMAC),
			padRight(ipAddr, colWidthIP),
			padRight(macAddr, colWidthType),
			padRight(deviceType, colWidthVendor),
			snmp)
	}

	fmt.Fprintln(os.Stdout, "└────────────────────┴─────────────────┴───────────────────┴──────────┴───────┘")
	fmt.Fprintf(os.Stdout, "\nTotal: %d device(s)\n", len(cfg.Devices))

	// Count SNMP-enabled devices
	snmpCount := 0
	for _, device := range cfg.Devices {
		if device.SNMPConfig.Community != "" || device.SNMPConfig.WalkFile != "" {
			snmpCount++
		}
	}
	if snmpCount > 0 {
		fmt.Fprintf(os.Stdout, "SNMP-enabled: %d device(s)\n", snmpCount)
	}
}

func getDebugLevelName(level int) string {
	switch level {
	case debugLevelQuiet:
		return "QUIET"
	case debugLevelNormal:
		return "NORMAL"
	case debugLevelVerbose:
		return "VERBOSE"
	case debugLevelDebug:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

func padRight(str string, length int) string {
	runes := []rune(str)
	if len(runes) >= length {
		return string(runes[:length])
	}
	return fmt.Sprintf("%-*s", length, str)
}

// startSimulation initializes the capture engine and protocol stack, returning running handles.
func startSimulation(
	interfaceName string,
	cfg *config.Config,
	debugConfig *logging.DebugConfig,
) (*capture.Engine, *protocols.Stack, time.Time, error) {
	debugLevel := debugConfig.GetGlobal()

	if debugLevel >= 1 {
		fmt.Fprintln(os.Stdout, "Starting NIAC simulation...")
		fmt.Fprintf(os.Stdout, "  Interface: %s\n", interfaceName)
		fmt.Fprintf(os.Stdout, "  Devices: %d\n", len(cfg.Devices))
		fmt.Fprintf(os.Stdout, "  Debug level: %d\n", debugLevel)
		fmt.Fprintln(os.Stdout)
	}

	engine, err := initializeCaptureEngine(interfaceName, debugLevel)
	if err != nil {
		return nil, nil, time.Time{}, err
	}

	if debugLevel >= 1 {
		fmt.Fprint(os.Stdout, "⏳ Creating protocol stack... ")
	}
	stack := protocols.NewStack(engine, cfg, debugConfig)
	if debugLevel >= 1 {
		fmt.Fprintln(os.Stdout, "✓")
	}

	dhcpCount, dnsCount := configureServiceHandlers(stack, cfg, debugLevel)
	if debugLevel >= 1 && (dhcpCount > 0 || dnsCount > 0) {
		if dhcpCount > 0 {
			fmt.Fprintf(os.Stdout, "⏳ Configuring DHCP servers (%d)... ✓\n", dhcpCount)
		}
		if dnsCount > 0 {
			fmt.Fprintf(os.Stdout, "⏳ Configuring DNS servers (%d)... ✓\n", dnsCount)
		}
	}

	if debugLevel >= 1 {
		fmt.Fprintf(os.Stdout, "⏳ Starting %d simulated device(s)... ", len(cfg.Devices))
	}
	if startErr := stack.Start(); startErr != nil {
		if debugLevel >= 1 {
			fmt.Fprintln(os.Stdout, "❌")
		}
		engine.Close()
		return nil, nil, time.Time{}, fmt.Errorf("failed to start stack: %w", startErr)
	}
	if debugLevel >= 1 {
		fmt.Fprintln(os.Stdout, "✓")
		printStartupSummary(cfg, debugLevel)
	}

	return engine, stack, time.Now(), nil
}

// runNormalMode runs NIAC in normal (non-interactive) mode.
func runNormalMode(
	interfaceName string,
	cfg *config.Config,
	debugConfig *logging.DebugConfig,
	configFile string,
	services *serviceOptions,
) error {
	engine, stack, startTime, err := startSimulation(interfaceName, cfg, debugConfig)
	if err != nil {
		return err
	}
	defer engine.Close()
	defer stack.Stop()

	servicesRuntime, err := startRuntimeServices(engine, stack, cfg, interfaceName, configFile, services)
	if err != nil {
		return err
	}
	defer func() {
		if servicesRuntime != nil {
			servicesRuntime.Stop()
		}
	}()

	reloadFunc := buildReloadFunc(stack, configFile, servicesRuntime)
	return runSimulationLoop(stack, debugConfig.GetGlobal(), startTime, reloadFunc)
}

// runInteractiveMode runs NIAC with the interactive TUI layered on the live simulator.
func runInteractiveMode(
	interfaceName string,
	cfg *config.Config,
	debugConfig *logging.DebugConfig,
	configFile string,
	services *serviceOptions,
) error {
	engine, stack, startTime, err := startSimulation(interfaceName, cfg, debugConfig)
	if err != nil {
		return err
	}

	servicesRuntime, err := startRuntimeServices(engine, stack, cfg, interfaceName, configFile, services)
	if err != nil {
		engine.Close()
		stack.Stop()
		return err
	}

	defer func() {
		stack.Stop()
		engine.Close()
		if servicesRuntime != nil {
			servicesRuntime.Stop()
		}
	}()

	reloadFunc := buildReloadFunc(stack, configFile, servicesRuntime)
	if runErr := interactive.Run(interfaceName, cfg, debugConfig, stack, startTime, reloadFunc); runErr != nil {
		return fmt.Errorf("failed to run interactive mode: %w", runErr)
	}
	return nil
}

// initializeCaptureEngine initializes the packet capture engine.
func initializeCaptureEngine(interfaceName string, debugLevel int) (*capture.Engine, error) {
	if debugLevel >= 1 {
		fmt.Fprint(os.Stdout, "⏳ Initializing capture engine... ")
	}
	engine, err := capture.New(interfaceName, debugLevel)
	if err != nil {
		if debugLevel >= 1 {
			fmt.Fprintln(os.Stdout, "❌")
		}
		return nil, fmt.Errorf("failed to create capture engine: %w", err)
	}
	if debugLevel >= 1 {
		fmt.Fprintln(os.Stdout, "✓")
	}
	return engine, nil
}

// configureServiceHandlers configures DHCP and DNS service handlers.
func configureServiceHandlers(stack *protocols.Stack, cfg *config.Config, _ int) (int, int) {
	dhcpCount := 0
	dnsCount := 0

	for _, device := range cfg.Devices {
		if configureDHCPForDevice(stack, &device) {
			dhcpCount++
		}
		if configureDNSForDevice(stack, &device) {
			dnsCount++
		}
	}

	return dhcpCount, dnsCount
}

// configureDHCPForDevice configures DHCP handlers for a single device.
// Returns true if DHCP was configured.
func configureDHCPForDevice(stack *protocols.Stack, device *config.Device) bool {
	if device.DHCPConfig == nil || len(device.IPAddresses) == 0 {
		return false
	}

	dhcp := device.DHCPConfig
	configureDHCPv4Basic(stack.GetDHCPHandler(), device.IPAddresses[0], dhcp)
	configureDHCPv4Advanced(stack.GetDHCPHandler(), dhcp)
	configureDHCPv6Advanced(stack.GetDHCPv6Handler(), dhcp)

	return true
}

// configureDHCPv4Basic sets basic DHCPv4 server configuration.
func configureDHCPv4Basic(handler *protocols.DHCPHandler, serverIP net.IP, dhcp *config.DHCPConfig) {
	hasBasicConfig := len(dhcp.DomainNameServer) > 0 || dhcp.Router != nil
	if !hasBasicConfig {
		return
	}

	handler.SetServerConfig(serverIP, dhcp.Router, dhcp.DomainNameServer, dhcp.DomainName)
}

// configureDHCPv4Advanced sets advanced DHCPv4 options.
func configureDHCPv4Advanced(handler *protocols.DHCPHandler, dhcp *config.DHCPConfig) {
	hasAdvancedOptions := len(dhcp.NTPServers) > 0 ||
		len(dhcp.DomainSearch) > 0 ||
		dhcp.TFTPServerName != "" ||
		dhcp.BootfileName != ""

	if !hasAdvancedOptions {
		return
	}

	handler.SetAdvancedOptions(
		dhcp.NTPServers,
		dhcp.DomainSearch,
		dhcp.TFTPServerName,
		dhcp.BootfileName,
		dhcp.VendorSpecific,
	)
}

// configureDHCPv6Advanced sets advanced DHCPv6 options.
func configureDHCPv6Advanced(handler *protocols.DHCPv6Handler, dhcp *config.DHCPConfig) {
	hasV6Options := len(dhcp.SNTPServersV6) > 0 ||
		len(dhcp.NTPServersV6) > 0 ||
		len(dhcp.SIPServersV6) > 0 ||
		len(dhcp.SIPDomainsV6) > 0

	if !hasV6Options {
		return
	}

	handler.SetAdvancedOptions(
		dhcp.SNTPServersV6,
		dhcp.NTPServersV6,
		dhcp.SIPServersV6,
		dhcp.SIPDomainsV6,
	)
}

// configureDNSForDevice configures DNS handler for a single device.
// Returns true if DNS was configured.
func configureDNSForDevice(stack *protocols.Stack, device *config.Device) bool {
	if device.DNSConfig == nil {
		return false
	}

	dnsHandler := stack.GetDNSHandler()
	for _, record := range device.DNSConfig.ForwardRecords {
		dnsHandler.AddRecord(record.Name, record.IP)
	}
	// PTR records are handled automatically by AddRecord

	return true
}

// printStartupSummary displays the enabled features summary.
func printStartupSummary(cfg *config.Config, _ int) {
	fmt.Fprintln(os.Stdout)

	// Display enabled features summary
	fmt.Fprintln(os.Stdout, "Enabled features:")

	// Count and display SNMP-enabled devices
	snmpCount := 0
	trapCount := 0
	for _, dev := range cfg.Devices {
		if dev.SNMPConfig.Community != "" || dev.SNMPConfig.WalkFile != "" {
			snmpCount++
		}
		if dev.SNMPConfig.Traps != nil && dev.SNMPConfig.Traps.Enabled {
			trapCount++
		}
	}
	if snmpCount > 0 {
		fmt.Fprintf(os.Stdout, "  • SNMP agents: %d device(s)\n", snmpCount)
		if trapCount > 0 {
			fmt.Fprintf(os.Stdout, "  • SNMP traps: %d device(s)\n", trapCount)
		}
	}

	// Count devices with traffic patterns
	trafficCount := 0
	for _, dev := range cfg.Devices {
		if dev.TrafficConfig != nil && dev.TrafficConfig.Enabled {
			trafficCount++
		}
	}
	if trafficCount > 0 {
		fmt.Fprintf(os.Stdout, "  • Traffic generation: %d device(s)\n", trafficCount)
	}

	// Show PCAP playback if configured
	if cfg.CapturePlayback != nil {
		fmt.Fprintf(os.Stdout, "  • PCAP playback: %s\n", cfg.CapturePlayback.FileName)
	}

	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "✅ Network simulation is ready")
	fmt.Fprintln(os.Stdout, "   Press Ctrl+C to stop")
	fmt.Fprintln(os.Stdout)
}

// runSimulationLoop runs the main simulation loop with signal handling and stats.
func runSimulationLoop(
	stack *protocols.Stack,
	debugLevel int,
	startTime time.Time,
	reloadConfig func() (*config.Config, error),
) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigChan)

	var statsTicker *time.Ticker
	var statsC <-chan time.Time
	if debugLevel >= 1 {
		statsTicker = time.NewTicker(statsTickerInterval * time.Second)
		statsC = statsTicker.C
		defer statsTicker.Stop()
	}

	for {
		select {
		case sig := <-sigChan:
			if handleSignal(sig, stack, debugLevel, startTime, reloadConfig) {
				return nil
			}
		case <-statsC:
			printPeriodicStats(stack, time.Since(startTime))
		}
	}
}

// handleSignal processes incoming signals. Returns true if the loop should exit.
func handleSignal(
	sig os.Signal,
	stack *protocols.Stack,
	debugLevel int,
	startTime time.Time,
	reloadConfig func() (*config.Config, error),
) bool {
	if sig == syscall.SIGHUP {
		handleReload(reloadConfig, debugLevel)
		return false
	}

	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Shutting down...")
	stack.Stop()

	if debugLevel >= 1 {
		printFinalStats(stack, time.Since(startTime))
	}
	return true
}

// handleReload attempts to reload the configuration.
func handleReload(reloadConfig func() (*config.Config, error), debugLevel int) {
	if reloadConfig == nil {
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, "Reload requested but no reload handler is available")
		return
	}

	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Reloading configuration...")
	cfg, err := reloadConfig()
	if err != nil {
		fmt.Fprintf(os.Stdout, "Reload failed: %v\n", err)
		return
	}
	if cfg != nil && debugLevel >= 1 {
		fmt.Fprintf(os.Stdout, "Reloaded configuration (%d devices)\n", len(cfg.Devices))
	}
}

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

func buildReloadFunc(
	stack *protocols.Stack,
	configFile string,
	services *runtimeServices,
) func() (*config.Config, error) {
	if configFile == "" || stack == nil {
		return nil
	}
	abs, err := filepath.Abs(configFile)
	if err != nil {
		abs = configFile
	}
	return func() (*config.Config, error) {
		newCfg, loadErr := config.Load(abs)
		if loadErr != nil {
			return nil, fmt.Errorf("failed to load config: %w", loadErr)
		}
		if services != nil {
			if applyErr := services.applyConfig(newCfg); applyErr != nil {
				return nil, fmt.Errorf("failed to apply config: %w", applyErr)
			}
		} else {
			if reloadErr := stack.ReloadConfig(newCfg); reloadErr != nil {
				return nil, fmt.Errorf("failed to reload config: %w", reloadErr)
			}
			configureServiceHandlers(stack, newCfg, stack.GetDebugLevel())
		}
		return newCfg, nil
	}
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
