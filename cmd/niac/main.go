// Package main provides the NIAC command-line interface for network device simulation
package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/capture"
	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/interactive"
	"github.com/krisarmstrong/niac-go/pkg/logging"
	"github.com/krisarmstrong/niac-go/pkg/protocols"
	"github.com/krisarmstrong/niac-go/pkg/stats"
)

// Version information is now managed in root.go
// Build-time variables can be set with: go build -ldflags "-X main.version=..."

// Global statistics instance
var globalStats *stats.Statistics

func main() {
	Execute()
}

// runLegacyMode maintains backward compatibility with original command-line interface
// Refactored into smaller, testable functions
func runLegacyMode(osArgs []string) {
	// Reset flag.CommandLine to avoid conflicts with cobra
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Define and parse flags
	var flags legacyFlags
	defineLegacyFlags(&flags)
	flag.Usage = printUsage
	// Parse the provided arguments (skip first element which is program name)
	if len(osArgs) > 1 {
		flag.CommandLine.Parse(osArgs[1:])
	} else {
		flag.Parse()
	}

	// Process flag overrides (verbose/quiet)
	processFlags(&flags)

	// Initialize colors (respects --no-color flag and NO_COLOR env var)
	logging.InitColors(!flags.noColor)

	// Get remaining arguments
	args := flag.Args()

	// Handle informational flags (version, list-interfaces, list-devices)
	if handleInformationalFlags(&flags, args) {
		os.Exit(0)
	}

	// Validate required arguments
	interfaceName, configFile, err := validateLegacyArguments(args)
	if err != nil {
		printUsage()
		os.Exit(1)
	}

	// Start profiling server if enabled
	if flags.enableProfiling {
		startProfilingServer(flags.profilePort, flags.debugLevel)
	}

	// Print banner (unless quiet)
	if flags.debugLevel > 0 {
		printBanner()
	}

	// Validate interface exists
	if err := validateInterface(interfaceName); err != nil {
		os.Exit(2)
	}

	// Load configuration
	cfg, err := loadAndPrintConfig(configFile, interfaceName, &flags)
	if err != nil {
		logging.Error("%v", err)
		os.Exit(1)
	}

	// Handle dry run mode
	if flags.dryRun {
		runDryRunValidation(configFile, interfaceName, cfg)
		// runDryRunValidation calls os.Exit, so this line is unreachable
	}

	// Create debug configuration
	debugConfig := setupDebugConfig(&flags)

	// Initialize global statistics (v1.19.0)
	globalStats = stats.NewStatistics(interfaceName, configFile, version)
	globalStats.SetDeviceCount(len(cfg.Devices))

	// Count SNMP-enabled devices
	snmpCount := 0
	for _, dev := range cfg.Devices {
		if dev.SNMPConfig.Community != "" || dev.SNMPConfig.WalkFile != "" {
			snmpCount++
		}
	}
	globalStats.SetSNMPDeviceCount(snmpCount)

	// Setup deferred stats export on exit
	if flags.exportStatsJSON != "" || flags.exportStatsCSV != "" {
		defer exportStatistics(&flags)
	}

	// Start simulation based on mode
	if flags.interactiveMode {
		if err := runInteractiveMode(interfaceName, cfg, debugConfig); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := runNormalMode(interfaceName, cfg, debugConfig); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	}
}

// startProfilingServer starts the pprof HTTP server for performance profiling
func startProfilingServer(port int, debugLevel int) {
	// Security: bind to localhost only to prevent external access
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	go func() {
		if debugLevel >= 1 {
			logging.Info("Starting pprof server on http://%s/debug/pprof/", addr)
			logging.Info("  CPU profile:    http://%s/debug/pprof/profile?seconds=30", addr)
			logging.Info("  Heap profile:   http://%s/debug/pprof/heap", addr)
			logging.Info("  Goroutines:     http://%s/debug/pprof/goroutine", addr)
			logging.Warning("Profiling server is for local development only - do not expose publicly")
			fmt.Println()
		}

		// Start HTTP server - pprof handlers are automatically registered via import
		if err := http.ListenAndServe(addr, nil); err != nil {
			logging.Error("Failed to start pprof server: %v", err)
		}
	}()
}

func printBanner() {
	fmt.Printf("╔══════════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  NIAC - Network In A Can (Go Edition)                           ║\n")
	fmt.Printf("║  Version %s                                                 ║\n", padRight(version, 51))
	fmt.Printf("╚══════════════════════════════════════════════════════════════════╝\n")
	fmt.Println()
}

func printVersion() {
	fmt.Printf("NIAC-Go version %s\n", version)
	fmt.Printf("Build commit: %s\n", commit)
	fmt.Printf("Build date: %s\n", date)
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println()
	fmt.Println("Enhancements over Java version:")
	fmt.Println("  • 10x-770x faster performance")
	fmt.Println("  • 3.3x less code")
	fmt.Println("  • Advanced HTTP server (multi-endpoint)")
	fmt.Println("  • Complete FTP server (17 commands)")
	fmt.Println("  • Advanced device simulation")
	fmt.Println("  • Comprehensive traffic generation")
	fmt.Println()
	fmt.Println("Original NIAC by Kevin Kayes (2002-2015)")
	fmt.Println("Go rewrite by Kris Armstrong (2025)")
}

func printUsage() {
	fmt.Println("USAGE:")
	fmt.Println("  niac [OPTIONS] <interface> <config_file>")
	fmt.Println("  niac --list-interfaces")
	fmt.Println("  niac --version")
	fmt.Println()
	fmt.Println("REQUIRED ARGUMENTS:")
	fmt.Println("  <interface>     Network interface to use (e.g., en0, eth0)")
	fmt.Println("  <config_file>   Configuration file path (.cfg, .json, or .yaml)")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  Core:")
	fmt.Println("    -d, --debug <level>      Debug level (0-3) [default: 1]")
	fmt.Println("                             0=quiet, 1=normal, 2=verbose, 3=debug")
	fmt.Println("    -v, --verbose            Verbose output (equivalent to -d 3)")
	fmt.Println("    -q, --quiet              Quiet mode (equivalent to -d 0)")
	fmt.Println("    -i, --interactive        Enable interactive TUI mode")
	fmt.Println("    -n, --dry-run            Validate configuration without starting")
	fmt.Println()
	fmt.Println("  Information:")
	fmt.Println("    -V, --version            Show version information")
	fmt.Println("    -l, --list-interfaces    List available network interfaces")
	fmt.Println("        --list-devices       List devices in configuration file")
	fmt.Println("    -h, --help               Show this help message")
	fmt.Println()
	fmt.Println("  Output:")
	fmt.Println("        --no-color           Disable colored output")
	fmt.Println("        --log-file <file>    Write log to file")
	fmt.Println("        --stats-interval <n> Statistics update interval [default: 1s]")
	fmt.Println()
	fmt.Println("  Advanced:")
	fmt.Println("        --babble-interval <n>   Traffic generation interval [default: 60s]")
	fmt.Println("        --no-traffic            Disable background traffic generation")
	fmt.Println("        --snmp-community <str>  Default SNMP community string")
	fmt.Println("        --max-packet-size <n>   Maximum packet size [default: 1514]")
	fmt.Println()
	fmt.Println("  Performance Profiling:")
	fmt.Println("    -p, --profile            Enable pprof performance profiling")
	fmt.Println("        --profile-port <port>   Port for pprof HTTP server [default: 6060]")
	fmt.Println()
	fmt.Println("  Statistics Export:")
	fmt.Println("        --export-stats-json <file>  Export runtime statistics to JSON file on exit")
	fmt.Println("        --export-stats-csv <file>   Export runtime statistics to CSV file on exit")
	fmt.Println()
	fmt.Println("  Per-Protocol Debug Levels:")
	fmt.Println("        --debug-arp <level>     ARP protocol debug level (0-3)")
	fmt.Println("        --debug-ip <level>      IP protocol debug level (0-3)")
	fmt.Println("        --debug-icmp <level>    ICMP protocol debug level (0-3)")
	fmt.Println("        --debug-ipv6 <level>    IPv6 protocol debug level (0-3)")
	fmt.Println("        --debug-icmpv6 <level>  ICMPv6 protocol debug level (0-3)")
	fmt.Println("        --debug-udp <level>     UDP protocol debug level (0-3)")
	fmt.Println("        --debug-tcp <level>     TCP protocol debug level (0-3)")
	fmt.Println("        --debug-dns <level>     DNS protocol debug level (0-3)")
	fmt.Println("        --debug-dhcp <level>    DHCP protocol debug level (0-3)")
	fmt.Println("        --debug-dhcpv6 <level>  DHCPv6 protocol debug level (0-3)")
	fmt.Println("        --debug-http <level>    HTTP protocol debug level (0-3)")
	fmt.Println("        --debug-ftp <level>     FTP protocol debug level (0-3)")
	fmt.Println("        --debug-netbios <level> NetBIOS protocol debug level (0-3)")
	fmt.Println("        --debug-stp <level>     STP protocol debug level (0-3)")
	fmt.Println("        --debug-lldp <level>    LLDP protocol debug level (0-3)")
	fmt.Println("        --debug-cdp <level>     CDP protocol debug level (0-3)")
	fmt.Println("        --debug-edp <level>     EDP protocol debug level (0-3)")
	fmt.Println("        --debug-fdp <level>     FDP protocol debug level (0-3)")
	fmt.Println("        --debug-snmp <level>    SNMP protocol debug level (0-3)")
	fmt.Println()
	fmt.Println("DEBUG LEVELS:")
	fmt.Println("  0  QUIET   - Only critical errors")
	fmt.Println("  1  NORMAL  - Status messages (default)")
	fmt.Println("  2  VERBOSE - Protocol details")
	fmt.Println("  3  DEBUG   - Full packet details")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  # List available interfaces")
	fmt.Println("  niac --list-interfaces")
	fmt.Println()
	fmt.Println("  # Validate configuration")
	fmt.Println("  niac --dry-run en0 network.cfg")
	fmt.Println()
	fmt.Println("  # Run in interactive mode with verbose debugging")
	fmt.Println("  sudo niac --interactive --verbose en0 network.cfg")
	fmt.Println()
	fmt.Println("  # Run in quiet mode with log file")
	fmt.Println("  sudo niac --quiet --log-file niac.log en0 network.cfg")
	fmt.Println()
	fmt.Println("  # Debug only DHCP protocol at verbose level")
	fmt.Println("  sudo niac --debug 1 --debug-dhcp 3 en0 network.cfg")
	fmt.Println()
	fmt.Println("  # Show version")
	fmt.Println("  niac --version")
	fmt.Println()
	fmt.Println("  # Enable profiling for performance analysis")
	fmt.Println("  sudo niac --profile en0 network.cfg")
	fmt.Println()
	fmt.Println("  # Enable profiling on custom port")
	fmt.Println("  sudo niac --profile --profile-port 8080 en0 network.cfg")
	fmt.Println()
	fmt.Println("PROFILING:")
	fmt.Println("  When --profile is enabled, pprof endpoints are available at:")
	fmt.Println("    http://localhost:6060/debug/pprof/          - Index page")
	fmt.Println("    http://localhost:6060/debug/pprof/profile   - CPU profile")
	fmt.Println("    http://localhost:6060/debug/pprof/heap      - Memory profile")
	fmt.Println("    http://localhost:6060/debug/pprof/goroutine - Goroutine profile")
	fmt.Println()
	fmt.Println("  Collect CPU profile (30 seconds):")
	fmt.Println("    curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof")
	fmt.Println("    go tool pprof cpu.prof")
	fmt.Println()
	fmt.Println("  Collect memory profile:")
	fmt.Println("    curl http://localhost:6060/debug/pprof/heap > mem.prof")
	fmt.Println("    go tool pprof mem.prof")
	fmt.Println()
	fmt.Println("  Interactive profiling:")
	fmt.Println("    go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30")
	fmt.Println("    go tool pprof http://localhost:6060/debug/pprof/heap")
	fmt.Println()
	fmt.Println("  WARNING: Profiling server binds to localhost only for security.")
	fmt.Println("           Do not expose the profiling port on public networks.")
	fmt.Println()
	fmt.Println("For more information, see: https://github.com/krisarmstrong/niac-go")
}

func printDeviceList(configFile string) {
	cfg, err := config.Load(configFile)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Devices in %s:\n\n", configFile)

	if len(cfg.Devices) == 0 {
		fmt.Println("No devices found in configuration.")
		return
	}

	// Print table header
	fmt.Println("┌────────────────────┬─────────────────┬───────────────────┬──────────┬───────┐")
	fmt.Println("│ Name               │ IP Address      │ MAC Address       │ Type     │ SNMP  │")
	fmt.Println("├────────────────────┼─────────────────┼───────────────────┼──────────┼───────┤")

	// Print devices
	for _, device := range cfg.Devices {
		ipAddr := "N/A"
		if len(device.IPAddresses) > 0 {
			ipAddr = device.IPAddresses[0].String()
			// Indicate if device has multiple IPs
			if len(device.IPAddresses) > 1 {
				ipAddr = ipAddr + " +" + fmt.Sprintf("%d", len(device.IPAddresses)-1)
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

		fmt.Printf("│ %-18s │ %-15s │ %-17s │ %-8s │ %-5s │\n",
			padRight(device.Name, 18),
			padRight(ipAddr, 15),
			padRight(macAddr, 17),
			padRight(deviceType, 8),
			snmp)
	}

	fmt.Println("└────────────────────┴─────────────────┴───────────────────┴──────────┴───────┘")
	fmt.Printf("\nTotal: %d device(s)\n", len(cfg.Devices))

	// Count SNMP-enabled devices
	snmpCount := 0
	for _, device := range cfg.Devices {
		if device.SNMPConfig.Community != "" || device.SNMPConfig.WalkFile != "" {
			snmpCount++
		}
	}
	if snmpCount > 0 {
		fmt.Printf("SNMP-enabled: %d device(s)\n", snmpCount)
	}
}

func getDebugLevelName(level int) string {
	switch level {
	case 0:
		return "QUIET"
	case 1:
		return "NORMAL"
	case 2:
		return "VERBOSE"
	case 3:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

func padRight(str string, length int) string {
	if len(str) >= length {
		return str[:length]
	}
	return str + strings.Repeat(" ", length-len(str))
}

// startSimulation initializes the capture engine and protocol stack, returning running handles
func startSimulation(interfaceName string, cfg *config.Config, debugConfig *logging.DebugConfig) (*capture.Engine, *protocols.Stack, time.Time, error) {
	debugLevel := debugConfig.GetGlobal()

	if debugLevel >= 1 {
		fmt.Println("Starting NIAC simulation...")
		fmt.Printf("  Interface: %s\n", interfaceName)
		fmt.Printf("  Devices: %d\n", len(cfg.Devices))
		fmt.Printf("  Debug level: %d\n", debugLevel)
		fmt.Println()
	}

	engine, err := initializeCaptureEngine(interfaceName, debugLevel)
	if err != nil {
		return nil, nil, time.Time{}, err
	}

	if debugLevel >= 1 {
		fmt.Print("⏳ Creating protocol stack... ")
	}
	stack := protocols.NewStack(engine, cfg, debugConfig)
	if debugLevel >= 1 {
		fmt.Println("✓")
	}

	dhcpCount, dnsCount := configureServiceHandlers(stack, cfg, debugLevel)
	if debugLevel >= 1 && (dhcpCount > 0 || dnsCount > 0) {
		if dhcpCount > 0 {
			fmt.Printf("⏳ Configuring DHCP servers (%d)... ✓\n", dhcpCount)
		}
		if dnsCount > 0 {
			fmt.Printf("⏳ Configuring DNS servers (%d)... ✓\n", dnsCount)
		}
	}

	if debugLevel >= 1 {
		fmt.Printf("⏳ Starting %d simulated device(s)... ", len(cfg.Devices))
	}
	if err := stack.Start(); err != nil {
		if debugLevel >= 1 {
			fmt.Println("❌")
		}
		engine.Close()
		return nil, nil, time.Time{}, fmt.Errorf("failed to start stack: %w", err)
	}
	if debugLevel >= 1 {
		fmt.Println("✓")
		printStartupSummary(cfg, debugLevel)
	}

	return engine, stack, time.Now(), nil
}

// runNormalMode runs NIAC in normal (non-interactive) mode
func runNormalMode(interfaceName string, cfg *config.Config, debugConfig *logging.DebugConfig) error {
	engine, stack, startTime, err := startSimulation(interfaceName, cfg, debugConfig)
	if err != nil {
		return err
	}
	defer engine.Close()
	defer stack.Stop()

	return runSimulationLoop(stack, debugConfig.GetGlobal(), startTime)
}

// runInteractiveMode runs NIAC with the interactive TUI layered on the live simulator
func runInteractiveMode(interfaceName string, cfg *config.Config, debugConfig *logging.DebugConfig) error {
	engine, stack, startTime, err := startSimulation(interfaceName, cfg, debugConfig)
	if err != nil {
		return err
	}

	defer func() {
		stack.Stop()
		engine.Close()
	}()

	return interactive.Run(interfaceName, cfg, debugConfig, stack, startTime)
}

// initializeCaptureEngine initializes the packet capture engine
func initializeCaptureEngine(interfaceName string, debugLevel int) (*capture.Engine, error) {
	if debugLevel >= 1 {
		fmt.Print("⏳ Initializing capture engine... ")
	}
	engine, err := capture.New(interfaceName, debugLevel)
	if err != nil {
		if debugLevel >= 1 {
			fmt.Println("❌")
		}
		return nil, fmt.Errorf("failed to create capture engine: %w", err)
	}
	if debugLevel >= 1 {
		fmt.Println("✓")
	}
	return engine, nil
}

// configureServiceHandlers configures DHCP and DNS service handlers
func configureServiceHandlers(stack *protocols.Stack, cfg *config.Config, debugLevel int) (dhcpCount, dnsCount int) {
	for _, device := range cfg.Devices {
		// Configure DHCP if present
		if device.DHCPConfig != nil && len(device.IPAddresses) > 0 {
			dhcpCount++
			dhcp := device.DHCPConfig
			dhcpHandler := stack.GetDHCPHandler()
			dhcpv6Handler := stack.GetDHCPv6Handler()

			// Basic DHCPv4 configuration
			if len(dhcp.DomainNameServer) > 0 || dhcp.Router != nil {
				dhcpHandler.SetServerConfig(
					device.IPAddresses[0], // Server IP
					dhcp.Router,           // Gateway
					dhcp.DomainNameServer, // DNS servers
					dhcp.DomainName,       // Domain name
				)
			}

			// Advanced DHCPv4 options
			if len(dhcp.NTPServers) > 0 || len(dhcp.DomainSearch) > 0 || dhcp.TFTPServerName != "" || dhcp.BootfileName != "" {
				dhcpHandler.SetAdvancedOptions(
					dhcp.NTPServers,
					dhcp.DomainSearch,
					dhcp.TFTPServerName,
					dhcp.BootfileName,
					dhcp.VendorSpecific,
				)
			}

			// DHCPv6 configuration
			if len(dhcp.SNTPServersV6) > 0 || len(dhcp.NTPServersV6) > 0 || len(dhcp.SIPServersV6) > 0 || len(dhcp.SIPDomainsV6) > 0 {
				dhcpv6Handler.SetAdvancedOptions(
					dhcp.SNTPServersV6,
					dhcp.NTPServersV6,
					dhcp.SIPServersV6,
					dhcp.SIPDomainsV6,
				)
			}
		}

		// Configure DNS if present
		if device.DNSConfig != nil {
			dnsCount++
			dnsHandler := stack.GetDNSHandler()

			// Load DNS records
			for _, record := range device.DNSConfig.ForwardRecords {
				dnsHandler.AddRecord(record.Name, record.IP)
			}
			// PTR records are handled automatically by AddRecord
		}
	}
	return dhcpCount, dnsCount
}

// printStartupSummary displays the enabled features summary
func printStartupSummary(cfg *config.Config, debugLevel int) {
	fmt.Println()

	// Display enabled features summary
	fmt.Println("Enabled features:")

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
		fmt.Printf("  • SNMP agents: %d device(s)\n", snmpCount)
		if trapCount > 0 {
			fmt.Printf("  • SNMP traps: %d device(s)\n", trapCount)
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
		fmt.Printf("  • Traffic generation: %d device(s)\n", trafficCount)
	}

	// Show PCAP playback if configured
	if cfg.CapturePlayback != nil {
		fmt.Printf("  • PCAP playback: %s\n", cfg.CapturePlayback.FileName)
	}

	fmt.Println()
	fmt.Println("✅ Network simulation is ready")
	fmt.Println("   Press Ctrl+C to stop")
	fmt.Println()
}

// runSimulationLoop runs the main simulation loop with signal handling and stats
func runSimulationLoop(stack *protocols.Stack, debugLevel int, startTime time.Time) error {
	// Setup signal handler for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Stats ticker (print stats every 10 seconds if debug >= 1)
	var statsTicker *time.Ticker
	var statsC <-chan time.Time
	if debugLevel >= 1 {
		statsTicker = time.NewTicker(10 * time.Second)
		statsC = statsTicker.C
		defer statsTicker.Stop()
	}

	// Main loop
	for {
		select {
		case <-sigChan:
			// Graceful shutdown
			fmt.Println()
			fmt.Println("Shutting down...")
			stack.Stop()

			// Print final stats
			if debugLevel >= 1 {
				printFinalStats(stack, time.Since(startTime))
			}

			return nil

		case <-statsC:
			// Print periodic stats
			printPeriodicStats(stack, time.Since(startTime))
		}
	}
}

// printPeriodicStats prints periodic statistics
func printPeriodicStats(stack *protocols.Stack, uptime time.Duration) {
	stats := stack.GetStats()

	fmt.Printf("[%s] Uptime: %s | Packets: RX=%d TX=%d | ARP: %d/%d | ICMP: %d/%d | DNS: %d | DHCP: %d\n",
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
	)
}

// printFinalStats prints final statistics on shutdown
func printFinalStats(stack *protocols.Stack, uptime time.Duration) {
	stats := stack.GetStats()

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                       Final Statistics                           ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ Total Uptime:        %-43s ║\n", formatDuration(uptime))
	fmt.Println("║                                                                  ║")
	fmt.Printf("║ Packets Received:    %-10d                                    ║\n", stats.PacketsReceived)
	fmt.Printf("║ Packets Sent:        %-10d                                    ║\n", stats.PacketsSent)
	fmt.Println("║                                                                  ║")
	fmt.Printf("║ ARP Requests:        %-10d                                    ║\n", stats.ARPRequests)
	fmt.Printf("║ ARP Replies:         %-10d                                    ║\n", stats.ARPReplies)
	fmt.Printf("║ ICMP Requests:       %-10d                                    ║\n", stats.ICMPRequests)
	fmt.Printf("║ ICMP Replies:        %-10d                                    ║\n", stats.ICMPReplies)
	fmt.Printf("║ DNS Queries:         %-10d                                    ║\n", stats.DNSQueries)
	fmt.Printf("║ DHCP Requests:       %-10d                                    ║\n", stats.DHCPRequests)
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// formatDuration formats a duration in a readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

// exportStatistics exports runtime statistics to JSON and/or CSV files (v1.19.0)
func exportStatistics(flags *legacyFlags) {
	if globalStats == nil {
		return
	}

	// Update final statistics
	globalStats.Update()

	// Export to JSON if requested
	if flags.exportStatsJSON != "" {
		if err := globalStats.ExportJSON(flags.exportStatsJSON); err != nil {
			logging.Error("Failed to export statistics to JSON: %v", err)
		} else {
			logging.Info("Statistics exported to JSON: %s", flags.exportStatsJSON)
		}
	}

	// Export to CSV if requested
	if flags.exportStatsCSV != "" {
		if err := globalStats.ExportCSV(flags.exportStatsCSV); err != nil {
			logging.Error("Failed to export statistics to CSV: %v", err)
		} else {
			logging.Info("Statistics exported to CSV: %s", flags.exportStatsCSV)
		}
	}
}
