// Package main provides legacy mode helper functions
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/krisarmstrong/niac-go/pkg/capture"
	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/logging"
)

// legacyFlags holds all command-line flags for legacy mode
type legacyFlags struct {
	// Core flags
	debugLevel      int
	verbose         bool
	quiet           bool
	interactiveMode bool
	dryRun          bool

	// Information flags
	showVersion    bool
	listInterfaces bool
	listDevices    bool

	// Output flags
	noColor       bool
	logFile       string
	statsInterval int

	// Advanced flags
	babbleInterval int
	noTraffic      bool
	snmpCommunity  string
	maxPacketSize  int

	// Profiling flags
	enableProfiling bool
	profilePort     int

	// Statistics export flags
	exportStatsJSON string
	exportStatsCSV  string

	// Per-protocol debug levels
	debugARP     int
	debugIP      int
	debugICMP    int
	debugIPv6    int
	debugICMPv6  int
	debugUDP     int
	debugTCP     int
	debugDNS     int
	debugDHCP    int
	debugDHCPv6  int
	debugHTTP    int
	debugFTP     int
	debugNetBIOS int
	debugSTP     int
	debugLLDP    int
	debugCDP     int
	debugEDP     int
	debugFDP     int
	debugSNMP    int

	// Service / API flags
	apiListen             string
	apiToken              string
	metricsListen         string
	storagePath           string
	alertPacketsThreshold uint64
	alertWebhook          string
}

// defineLegacyFlags defines all command-line flags for legacy mode
func defineLegacyFlags(flags *legacyFlags) {
	// Core flags
	flag.IntVar(&flags.debugLevel, "d", 1, "Debug level (0-3)")
	flag.IntVar(&flags.debugLevel, "debug", 1, "Debug level (0-3)")
	flag.BoolVar(&flags.verbose, "v", false, "Verbose output (equivalent to -d 3)")
	flag.BoolVar(&flags.verbose, "verbose", false, "Verbose output (equivalent to -d 3)")
	flag.BoolVar(&flags.quiet, "q", false, "Quiet mode (equivalent to -d 0)")
	flag.BoolVar(&flags.quiet, "quiet", false, "Quiet mode (equivalent to -d 0)")

	flag.BoolVar(&flags.interactiveMode, "i", false, "Enable interactive TUI mode")
	flag.BoolVar(&flags.interactiveMode, "interactive", false, "Enable interactive TUI mode")
	flag.BoolVar(&flags.dryRun, "n", false, "Dry run - validate configuration without starting")
	flag.BoolVar(&flags.dryRun, "dry-run", false, "Dry run - validate configuration without starting")

	// Information flags
	flag.BoolVar(&flags.showVersion, "V", false, "Show version information")
	flag.BoolVar(&flags.showVersion, "version", false, "Show version information")
	flag.BoolVar(&flags.listInterfaces, "l", false, "List available network interfaces")
	flag.BoolVar(&flags.listInterfaces, "list-interfaces", false, "List available network interfaces")
	flag.BoolVar(&flags.listDevices, "list-devices", false, "List devices in configuration file")

	// Output flags
	flag.BoolVar(&flags.noColor, "no-color", false, "Disable colored output")
	flag.StringVar(&flags.logFile, "log-file", "", "Write log to file")
	flag.IntVar(&flags.statsInterval, "stats-interval", 1, "Statistics update interval in seconds")

	// Advanced flags
	flag.IntVar(&flags.babbleInterval, "babble-interval", 60, "Traffic generation interval in seconds")
	flag.BoolVar(&flags.noTraffic, "no-traffic", false, "Disable background traffic generation")
	flag.StringVar(&flags.snmpCommunity, "snmp-community", "", "Default SNMP community string")
	flag.IntVar(&flags.maxPacketSize, "max-packet-size", 1514, "Maximum packet size in bytes")

	// Profiling flags
	flag.BoolVar(&flags.enableProfiling, "profile", false, "Enable pprof performance profiling")
	flag.BoolVar(&flags.enableProfiling, "p", false, "Enable pprof performance profiling")
	flag.IntVar(&flags.profilePort, "profile-port", 6060, "Port for pprof HTTP server (default: 6060)")

	// Statistics export flags
	flag.StringVar(&flags.exportStatsJSON, "export-stats-json", "", "Export statistics to JSON file on exit")
	flag.StringVar(&flags.exportStatsCSV, "export-stats-csv", "", "Export statistics to CSV file on exit")

	// Per-protocol debug flags (-1 means use global level)
	flag.IntVar(&flags.debugARP, "debug-arp", -1, "ARP protocol debug level (0-3, default: global level)")
	flag.IntVar(&flags.debugIP, "debug-ip", -1, "IP protocol debug level (0-3, default: global level)")
	flag.IntVar(&flags.debugICMP, "debug-icmp", -1, "ICMP protocol debug level (0-3, default: global level)")
	flag.IntVar(&flags.debugIPv6, "debug-ipv6", -1, "IPv6 protocol debug level (0-3, default: global level)")
	flag.IntVar(&flags.debugICMPv6, "debug-icmpv6", -1, "ICMPv6 protocol debug level (0-3, default: global level)")
	flag.IntVar(&flags.debugUDP, "debug-udp", -1, "UDP protocol debug level (0-3, default: global level)")
	flag.IntVar(&flags.debugTCP, "debug-tcp", -1, "TCP protocol debug level (0-3, default: global level)")
	flag.IntVar(&flags.debugDNS, "debug-dns", -1, "DNS protocol debug level (0-3, default: global level)")
	flag.IntVar(&flags.debugDHCP, "debug-dhcp", -1, "DHCP protocol debug level (0-3, default: global level)")
	flag.IntVar(&flags.debugDHCPv6, "debug-dhcpv6", -1, "DHCPv6 protocol debug level (0-3, default: global level)")
	flag.IntVar(&flags.debugHTTP, "debug-http", -1, "HTTP protocol debug level (0-3, default: global level)")
	flag.IntVar(&flags.debugFTP, "debug-ftp", -1, "FTP protocol debug level (0-3, default: global level)")
	flag.IntVar(&flags.debugNetBIOS, "debug-netbios", -1, "NetBIOS protocol debug level (0-3, default: global level)")
	flag.IntVar(&flags.debugSTP, "debug-stp", -1, "STP protocol debug level (0-3, default: global level)")
	flag.IntVar(&flags.debugLLDP, "debug-lldp", -1, "LLDP protocol debug level (0-3, default: global level)")
	flag.IntVar(&flags.debugCDP, "debug-cdp", -1, "CDP protocol debug level (0-3, default: global level)")
	flag.IntVar(&flags.debugEDP, "debug-edp", -1, "EDP protocol debug level (0-3, default: global level)")
	flag.IntVar(&flags.debugFDP, "debug-fdp", -1, "FDP protocol debug level (0-3, default: global level)")
	flag.IntVar(&flags.debugSNMP, "debug-snmp", -1, "SNMP protocol debug level (0-3, default: global level)")

	// Service / API flags
	flag.StringVar(&flags.apiListen, "api-listen", "", "Expose REST API and Web UI on this address (e.g., :8080)")
	flag.StringVar(&flags.apiToken, "api-token", "", "Bearer token required for API/Web UI access")
	flag.StringVar(&flags.metricsListen, "metrics-listen", "", "Expose Prometheus metrics on this address (defaults to --api-listen)")
	flag.StringVar(&flags.storagePath, "storage-path", "", "Path to NIAC run history database (default: ~/.niac/niac.db)")
	flag.Uint64Var(&flags.alertPacketsThreshold, "alert-packets-threshold", 0, "Trigger alerts when total packet count exceeds this value")
	flag.StringVar(&flags.alertWebhook, "alert-webhook", "", "Optional webhook URL to notify when alerts fire")
}

// processFlags applies flag transformations (verbose/quiet override)
func processFlags(flags *legacyFlags) {
	if flags.verbose {
		flags.debugLevel = 3
	}
	if flags.quiet {
		flags.debugLevel = 0
	}
}

func applyLegacyServiceFlags(flags *legacyFlags) {
	if flags.apiListen != "" {
		servicesOpts.apiListen = flags.apiListen
	}
	if flags.apiToken != "" {
		servicesOpts.apiToken = flags.apiToken
	}
	if flags.metricsListen != "" {
		servicesOpts.metricsListen = flags.metricsListen
	}
	if flags.storagePath != "" {
		servicesOpts.storagePath = flags.storagePath
	}
	if flags.alertPacketsThreshold > 0 {
		servicesOpts.alertPacketsThreshold = flags.alertPacketsThreshold
	}
	if flags.alertWebhook != "" {
		servicesOpts.alertWebhook = flags.alertWebhook
	}
	if servicesOpts.storagePath == "" {
		servicesOpts.storagePath = defaultStoragePath()
	}
}

// setupDebugConfig creates debug configuration from flags
func setupDebugConfig(flags *legacyFlags) *logging.DebugConfig {
	debugConfig := logging.NewDebugConfig(flags.debugLevel)

	// Set per-protocol debug levels if specified (value >= 0)
	if flags.debugARP >= 0 {
		debugConfig.SetProtocolLevel(logging.ProtocolARP, flags.debugARP)
	}
	if flags.debugIP >= 0 {
		debugConfig.SetProtocolLevel(logging.ProtocolIP, flags.debugIP)
	}
	if flags.debugICMP >= 0 {
		debugConfig.SetProtocolLevel(logging.ProtocolICMP, flags.debugICMP)
	}
	if flags.debugIPv6 >= 0 {
		debugConfig.SetProtocolLevel(logging.ProtocolIPv6, flags.debugIPv6)
	}
	if flags.debugICMPv6 >= 0 {
		debugConfig.SetProtocolLevel(logging.ProtocolICMPv6, flags.debugICMPv6)
	}
	if flags.debugUDP >= 0 {
		debugConfig.SetProtocolLevel(logging.ProtocolUDP, flags.debugUDP)
	}
	if flags.debugTCP >= 0 {
		debugConfig.SetProtocolLevel(logging.ProtocolTCP, flags.debugTCP)
	}
	if flags.debugDNS >= 0 {
		debugConfig.SetProtocolLevel(logging.ProtocolDNS, flags.debugDNS)
	}
	if flags.debugDHCP >= 0 {
		debugConfig.SetProtocolLevel(logging.ProtocolDHCP, flags.debugDHCP)
	}
	if flags.debugDHCPv6 >= 0 {
		debugConfig.SetProtocolLevel(logging.ProtocolDHCPv6, flags.debugDHCPv6)
	}
	if flags.debugHTTP >= 0 {
		debugConfig.SetProtocolLevel(logging.ProtocolHTTP, flags.debugHTTP)
	}
	if flags.debugFTP >= 0 {
		debugConfig.SetProtocolLevel(logging.ProtocolFTP, flags.debugFTP)
	}
	if flags.debugNetBIOS >= 0 {
		debugConfig.SetProtocolLevel(logging.ProtocolNetBIOS, flags.debugNetBIOS)
	}
	if flags.debugSTP >= 0 {
		debugConfig.SetProtocolLevel(logging.ProtocolSTP, flags.debugSTP)
	}
	if flags.debugLLDP >= 0 {
		debugConfig.SetProtocolLevel(logging.ProtocolLLDP, flags.debugLLDP)
	}
	if flags.debugCDP >= 0 {
		debugConfig.SetProtocolLevel(logging.ProtocolCDP, flags.debugCDP)
	}
	if flags.debugEDP >= 0 {
		debugConfig.SetProtocolLevel(logging.ProtocolEDP, flags.debugEDP)
	}
	if flags.debugFDP >= 0 {
		debugConfig.SetProtocolLevel(logging.ProtocolFDP, flags.debugFDP)
	}
	if flags.debugSNMP >= 0 {
		debugConfig.SetProtocolLevel(logging.ProtocolSNMP, flags.debugSNMP)
	}

	return debugConfig
}

// handleInformationalFlags processes version/list flags and exits if handled
// Returns true if a flag was handled and program should exit
func handleInformationalFlags(flags *legacyFlags, args []string) bool {
	// Handle version flag
	if flags.showVersion {
		printVersion()
		return true
	}

	// Handle list interfaces flag
	if flags.listInterfaces {
		fmt.Println("Available network interfaces:")
		capture.ListInterfaces()
		return true
	}

	// Handle list devices flag (needs config file)
	if flags.listDevices {
		if len(args) < 1 {
			fmt.Println("Error: --list-devices requires a configuration file")
			fmt.Println()
			printUsage()
			os.Exit(1)
		}
		printDeviceList(args[0])
		return true
	}

	return false
}

// validateLegacyArguments validates required command-line arguments
func validateLegacyArguments(args []string) (interfaceName, configFile string, err error) {
	if len(args) < 2 {
		return "", "", fmt.Errorf("missing required arguments: interface and config file")
	}
	return args[0], args[1], nil
}

// validateInterface checks if interface exists
func validateInterface(interfaceName string) error {
	if !capture.InterfaceExists(interfaceName) {
		logging.Error("Interface '%s' not found", interfaceName)
		fmt.Println("\nAvailable interfaces:")
		capture.ListInterfaces()
		return fmt.Errorf("interface not found: %s", interfaceName)
	}
	return nil
}

// loadAndPrintConfig loads config and prints info
func loadAndPrintConfig(configFile, interfaceName string, flags *legacyFlags) (*config.Config, error) {
	cfg, err := config.Load(configFile)
	if err != nil {
		return nil, fmt.Errorf("loading configuration: %w", err)
	}

	if flags.debugLevel >= 1 {
		logging.Success("Loaded configuration: %s", configFile)
		logging.Info("  Devices: %d", len(cfg.Devices))
		logging.Info("  Interface: %s", interfaceName)
		logging.Info("  Debug level: %d (%s)", flags.debugLevel, getDebugLevelName(flags.debugLevel))
		if flags.interactiveMode {
			logging.Info("  Mode: Interactive TUI")
		}
		if cfg.CapturePlayback != nil {
			logging.Info("  PCAP Playback: %s", cfg.CapturePlayback.FileName)
			if cfg.CapturePlayback.LoopTime > 0 {
				logging.Info("    Loop interval: %dms", cfg.CapturePlayback.LoopTime)
			}
			if cfg.CapturePlayback.ScaleTime > 0 && cfg.CapturePlayback.ScaleTime != 1.0 {
				logging.Info("    Time scaling: %.2fx", cfg.CapturePlayback.ScaleTime)
			}
		}
		fmt.Println()
	}

	return cfg, nil
}

// runDryRunValidation runs dry-run validation and exits
func runDryRunValidation(configFile, interfaceName string, cfg *config.Config) {
	// Run comprehensive configuration validation
	validator := config.NewValidator(configFile)
	result := validator.Validate(cfg)

	if result.HasErrors() || result.HasWarnings() {
		fmt.Println(result.Format())
	}

	if !result.Valid {
		logging.Error("Configuration validation failed")
		os.Exit(1)
	}

	// Additional runtime checks
	logging.Success("Interface exists and is accessible")
	logging.Success("Ready to simulate %d devices on %s", len(cfg.Devices), interfaceName)
	fmt.Println()
	fmt.Println("Configuration is valid. Use without --dry-run to start simulation.")
	os.Exit(0)
}
