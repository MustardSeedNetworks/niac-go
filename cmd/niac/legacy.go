package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/MustardSeedNetworks/niac-go/internal/capture"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// legacyFlags holds all command-line flags for legacy mode.
type legacyFlags struct {
	// Core flags
	debugLevel int
	verbose    bool
	quiet      bool
	dryRun     bool

	// Information flags
	showVersion    bool
	listInterfaces bool
	listDevices    bool

	// Output flags
	noColor bool

	// Advanced flags
	noTraffic bool

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
	storagePath string
}

// defineLegacyFlags defines all command-line flags for legacy mode.
func defineLegacyFlags(flagSet *flag.FlagSet, flags *legacyFlags) {
	defineCoreFlags(flagSet, flags)
	defineInfoFlags(flagSet, flags)
	defineOutputFlags(flagSet, flags)
	defineAdvancedFlags(flagSet, flags)
	defineProfilingFlags(flagSet, flags)
	defineProtocolDebugFlags(flagSet, flags)
	defineServiceFlags(flagSet, flags)
}

func defineCoreFlags(flagSet *flag.FlagSet, flags *legacyFlags) {
	flagSet.IntVar(&flags.debugLevel, "d", 1, "Debug level (0-3)")
	flagSet.IntVar(&flags.debugLevel, "debug", 1, "Debug level (0-3)")
	flagSet.BoolVar(&flags.verbose, "v", false, "Verbose output (equivalent to -d 3)")
	flagSet.BoolVar(&flags.verbose, "verbose", false, "Verbose output (equivalent to -d 3)")
	flagSet.BoolVar(&flags.quiet, "q", false, "Quiet mode (equivalent to -d 0)")
	flagSet.BoolVar(&flags.quiet, "quiet", false, "Quiet mode (equivalent to -d 0)")
	flagSet.BoolVar(&flags.dryRun, "n", false, "Dry run - validate configuration without starting")
	flagSet.BoolVar(
		&flags.dryRun,
		"dry-run",
		false,
		"Dry run - validate configuration without starting",
	)
}

func defineInfoFlags(flagSet *flag.FlagSet, flags *legacyFlags) {
	flagSet.BoolVar(&flags.showVersion, "V", false, "Show version information")
	flagSet.BoolVar(&flags.showVersion, "version", false, "Show version information")
	flagSet.BoolVar(&flags.listInterfaces, "l", false, "List available network interfaces")
	flagSet.BoolVar(
		&flags.listInterfaces,
		"list-interfaces",
		false,
		"List available network interfaces",
	)
	flagSet.BoolVar(&flags.listDevices, "list-devices", false, "List devices in configuration file")
}

func defineOutputFlags(flagSet *flag.FlagSet, flags *legacyFlags) {
	flagSet.BoolVar(&flags.noColor, "no-color", false, "Disable colored output")
}

func defineAdvancedFlags(flagSet *flag.FlagSet, flags *legacyFlags) {
	flagSet.BoolVar(&flags.noTraffic, "no-traffic", false, "Disable background traffic generation")
}

func defineProfilingFlags(flagSet *flag.FlagSet, flags *legacyFlags) {
	flagSet.BoolVar(&flags.enableProfiling, "profile", false, "Enable pprof performance profiling")
	flagSet.BoolVar(&flags.enableProfiling, "p", false, "Enable pprof performance profiling")
	flagSet.IntVar(
		&flags.profilePort,
		"profile-port",
		legacyPprofPort,
		"Port for pprof HTTP server (default: 6060)",
	)
	flagSet.StringVar(
		&flags.exportStatsJSON,
		"export-stats-json",
		"",
		"Export statistics to JSON file on exit",
	)
	flagSet.StringVar(
		&flags.exportStatsCSV,
		"export-stats-csv",
		"",
		"Export statistics to CSV file on exit",
	)
}

func defineProtocolDebugFlags(flagSet *flag.FlagSet, flags *legacyFlags) {
	defineNetworkProtocolDebugFlags(flagSet, flags)
	defineApplicationProtocolDebugFlags(flagSet, flags)
	defineDiscoveryProtocolDebugFlags(flagSet, flags)
}

// defineNetworkProtocolDebugFlags defines debug flags for network layer protocols.
func defineNetworkProtocolDebugFlags(flagSet *flag.FlagSet, flags *legacyFlags) {
	flagSet.IntVar(
		&flags.debugARP,
		"debug-arp",
		-1,
		"ARP protocol debug level (0-3, default: global level)",
	)
	flagSet.IntVar(
		&flags.debugIP,
		"debug-ip",
		-1,
		"IP protocol debug level (0-3, default: global level)",
	)
	flagSet.IntVar(
		&flags.debugICMP,
		"debug-icmp",
		-1,
		"ICMP protocol debug level (0-3, default: global level)",
	)
	flagSet.IntVar(
		&flags.debugIPv6,
		"debug-ipv6",
		-1,
		"IPv6 protocol debug level (0-3, default: global level)",
	)
	flagSet.IntVar(
		&flags.debugICMPv6,
		"debug-icmpv6",
		-1,
		"ICMPv6 protocol debug level (0-3, default: global level)",
	)
	flagSet.IntVar(
		&flags.debugUDP,
		"debug-udp",
		-1,
		"UDP protocol debug level (0-3, default: global level)",
	)
	flagSet.IntVar(
		&flags.debugTCP,
		"debug-tcp",
		-1,
		"TCP protocol debug level (0-3, default: global level)",
	)
}

// defineApplicationProtocolDebugFlags defines debug flags for application layer protocols.
func defineApplicationProtocolDebugFlags(flagSet *flag.FlagSet, flags *legacyFlags) {
	flagSet.IntVar(
		&flags.debugDNS,
		"debug-dns",
		-1,
		"DNS protocol debug level (0-3, default: global level)",
	)
	flagSet.IntVar(
		&flags.debugDHCP,
		"debug-dhcp",
		-1,
		"DHCP protocol debug level (0-3, default: global level)",
	)
	flagSet.IntVar(
		&flags.debugDHCPv6,
		"debug-dhcpv6",
		-1,
		"DHCPv6 protocol debug level (0-3, default: global level)",
	)
	flagSet.IntVar(
		&flags.debugHTTP,
		"debug-http",
		-1,
		"HTTP protocol debug level (0-3, default: global level)",
	)
	flagSet.IntVar(
		&flags.debugFTP,
		"debug-ftp",
		-1,
		"FTP protocol debug level (0-3, default: global level)",
	)
	flagSet.IntVar(
		&flags.debugNetBIOS,
		"debug-netbios",
		-1,
		"NetBIOS protocol debug level (0-3, default: global level)",
	)
	flagSet.IntVar(
		&flags.debugSNMP,
		"debug-snmp",
		-1,
		"SNMP protocol debug level (0-3, default: global level)",
	)
}

// defineDiscoveryProtocolDebugFlags defines debug flags for discovery protocols.
func defineDiscoveryProtocolDebugFlags(flagSet *flag.FlagSet, flags *legacyFlags) {
	flagSet.IntVar(
		&flags.debugSTP,
		"debug-stp",
		-1,
		"STP protocol debug level (0-3, default: global level)",
	)
	flagSet.IntVar(
		&flags.debugLLDP,
		"debug-lldp",
		-1,
		"LLDP protocol debug level (0-3, default: global level)",
	)
	flagSet.IntVar(
		&flags.debugCDP,
		"debug-cdp",
		-1,
		"CDP protocol debug level (0-3, default: global level)",
	)
	flagSet.IntVar(
		&flags.debugEDP,
		"debug-edp",
		-1,
		"EDP protocol debug level (0-3, default: global level)",
	)
	flagSet.IntVar(
		&flags.debugFDP,
		"debug-fdp",
		-1,
		"FDP protocol debug level (0-3, default: global level)",
	)
}

func defineServiceFlags(flagSet *flag.FlagSet, flags *legacyFlags) {
	flagSet.StringVar(&flags.storagePath, "storage-path", "", "Path to NIAC run history database")
}

// processFlags applies flag transformations (verbose/quiet override) and validates flag values.
func processFlags(flags *legacyFlags) error {
	if flags.verbose {
		flags.debugLevel = 3
	}

	if flags.quiet {
		flags.debugLevel = 0
	}

	if flags.enableProfiling && (flags.profilePort < 1 || flags.profilePort > 65535) {
		return fmt.Errorf("%w: got %d", errProfilePortRange, flags.profilePort)
	}

	return nil
}

// errProfilePortRange marks an out-of-range --profile-port.
var errProfilePortRange = errors.New("--profile-port must be between 1 and 65535")

func applyLegacyServiceFlags(flags *legacyFlags, services *serviceOptions) {
	if flags.storagePath != "" {
		services.storagePath = flags.storagePath
	}
	if services.storagePath == "" {
		services.storagePath = defaultStoragePath()
	}
}

// setupDebugConfig creates debug configuration from flags.
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

// errListDevicesNeedsConfig marks --list-devices used without a configuration
// file, so a caller can tell "nothing to do" apart from "handled".
var errListDevicesNeedsConfig = errors.New("--list-devices requires a configuration file")

// handleInformationalFlags processes version/list flags.
//
// It reports whether a flag was handled, in which case the caller should stop
// rather than start a simulation.
func handleInformationalFlags(flags *legacyFlags, args []string, info versionInfo) (bool, error) {
	if flags.showVersion {
		printVersion(info)

		return true, nil
	}

	if flags.listInterfaces {
		fmt.Fprintln(os.Stdout, "Available network interfaces:")
		capture.ListInterfaces(os.Stdout)

		return true, nil
	}

	if flags.listDevices {
		if len(args) < 1 {
			printUsage()

			return true, errListDevicesNeedsConfig
		}

		printDeviceList(args[0])

		return true, nil
	}

	return false, nil
}

// validateLegacyArguments validates required command-line arguments.
func validateLegacyArguments(args []string) (string, string, error) {
	if len(args) < minArgsForConfig {
		return "", "", errors.New("missing required arguments: interface and config file")
	}
	return args[0], args[1], nil
}

// validateInterface checks if interface exists.
func validateInterface(interfaceName string) error {
	if !capture.InterfaceExists(interfaceName) {
		logging.Errorf("Interface '%s' not found", interfaceName)
		fmt.Fprintln(os.Stdout, "\nAvailable interfaces:")
		capture.ListInterfaces(os.Stdout)
		return fmt.Errorf("interface not found: %s", interfaceName)
	}
	return nil
}

// loadAndPrintConfig loads config and prints info.
func loadAndPrintConfig(
	configFile, interfaceName string,
	flags *legacyFlags,
) (*config.Config, error) {
	cfg, err := config.Load(configFile)
	if err != nil {
		return nil, fmt.Errorf("loading configuration: %w", err)
	}

	if flags.debugLevel >= 1 {
		logConfigurationDebug(configFile, interfaceName, flags, cfg)
	}

	return cfg, nil
}

// logConfigurationDebug logs configuration details when debug level >= 1.
func logConfigurationDebug(configFile, interfaceName string, flags *legacyFlags, cfg *config.Config) {
	logging.Successf("Loaded configuration: %s", configFile)
	logging.Infof("  Devices: %d", len(cfg.Devices))
	logging.Infof("  Interface: %s", interfaceName)
	logging.Infof(
		"  Debug level: %d (%s)",
		flags.debugLevel,
		getDebugLevelName(flags.debugLevel),
	)

	logCapturePlaybackDebug(cfg)
	fmt.Fprintln(os.Stdout)
}

// logCapturePlaybackDebug logs PCAP playback configuration if present.
func logCapturePlaybackDebug(cfg *config.Config) {
	if cfg.CapturePlayback == nil {
		return
	}

	logging.Infof("  PCAP Playback: %s", cfg.CapturePlayback.FileName)

	if cfg.CapturePlayback.LoopTime > 0 {
		logging.Infof("    Loop interval: %dms", cfg.CapturePlayback.LoopTime)
	}

	if cfg.CapturePlayback.ScaleTime > 0 && cfg.CapturePlayback.ScaleTime != 1.0 {
		logging.Infof("    Time scaling: %.2fx", cfg.CapturePlayback.ScaleTime)
	}
}

// runDryRunValidation reports whether the configuration would start.
func runDryRunValidation(configFile, interfaceName string, cfg *config.Config) error {
	// Run comprehensive configuration validation
	validator := config.NewValidator(configFile)
	result := validator.Validate(cfg)

	if result.HasErrors() || result.HasWarnings() {
		fmt.Fprintln(os.Stdout, result.Format())
	}

	if !result.Valid {
		return errConfigInvalid
	}

	// Additional runtime checks
	logging.Successf("Interface exists and is accessible")
	logging.Successf("Ready to simulate %d devices on %s", len(cfg.Devices), interfaceName)
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Configuration is valid. Use without --dry-run to start simulation.")

	return nil
}

// errConfigInvalid marks a dry run whose validation output has already been
// printed, so the caller adds an exit code rather than another message.
var errConfigInvalid = errors.New("configuration validation failed")
