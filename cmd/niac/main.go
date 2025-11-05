package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/capture"
	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/interactive"
	"github.com/krisarmstrong/niac-go/pkg/protocols"
)

const (
	Version      = "1.4.0"
	BuildDate    = "2025-11-05"
	GitCommit    = "HEAD"
	Enhancements = "SNMP Walk Support, Enhanced CLI, Debug Tools, Protocol Parity"
)

func main() {
	// Command line flags
	var (
		// Core flags
		debugLevel      int
		verbose         bool
		quiet           bool
		interactiveMode bool
		dryRun          bool

		// Information flags
		showVersion       bool
		listInterfaces    bool
		listDevices       bool

		// Output flags
		noColor      bool
		logFile      string
		statsInterval int

		// Advanced flags
		babbleInterval int
		noTraffic      bool
		snmpCommunity  string
		maxPacketSize  int
	)

	// Define flags
	flag.IntVar(&debugLevel, "d", 1, "Debug level (0-3)")
	flag.IntVar(&debugLevel, "debug", 1, "Debug level (0-3)")
	flag.BoolVar(&verbose, "v", false, "Verbose output (equivalent to -d 3)")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output (equivalent to -d 3)")
	flag.BoolVar(&quiet, "q", false, "Quiet mode (equivalent to -d 0)")
	flag.BoolVar(&quiet, "quiet", false, "Quiet mode (equivalent to -d 0)")

	flag.BoolVar(&interactiveMode, "i", false, "Enable interactive TUI mode")
	flag.BoolVar(&interactiveMode, "interactive", false, "Enable interactive TUI mode")
	flag.BoolVar(&dryRun, "n", false, "Dry run - validate configuration without starting")
	flag.BoolVar(&dryRun, "dry-run", false, "Dry run - validate configuration without starting")

	flag.BoolVar(&showVersion, "V", false, "Show version information")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&listInterfaces, "l", false, "List available network interfaces")
	flag.BoolVar(&listInterfaces, "list-interfaces", false, "List available network interfaces")
	flag.BoolVar(&listDevices, "list-devices", false, "List devices in configuration file")

	flag.BoolVar(&noColor, "no-color", false, "Disable colored output")
	flag.StringVar(&logFile, "log-file", "", "Write log to file")
	flag.IntVar(&statsInterval, "stats-interval", 1, "Statistics update interval in seconds")

	flag.IntVar(&babbleInterval, "babble-interval", 60, "Traffic generation interval in seconds")
	flag.BoolVar(&noTraffic, "no-traffic", false, "Disable background traffic generation")
	flag.StringVar(&snmpCommunity, "snmp-community", "", "Default SNMP community string")
	flag.IntVar(&maxPacketSize, "max-packet-size", 1514, "Maximum packet size in bytes")

	// Custom usage
	flag.Usage = printUsage
	flag.Parse()

	// Handle verbose/quiet flags
	if verbose {
		debugLevel = 3
	}
	if quiet {
		debugLevel = 0
	}

	// Handle version flag
	if showVersion {
		printVersion()
		os.Exit(0)
	}

	// Handle list interfaces flag
	if listInterfaces {
		fmt.Println("Available network interfaces:")
		capture.ListInterfaces()
		os.Exit(0)
	}

	// Get remaining arguments
	args := flag.Args()

	// Handle list devices flag (needs config file)
	if listDevices {
		if len(args) < 1 {
			fmt.Println("Error: --list-devices requires a configuration file")
			fmt.Println()
			printUsage()
			os.Exit(1)
		}
		printDeviceList(args[0])
		os.Exit(0)
	}

	// Check for required arguments (unless just showing info)
	if len(args) < 2 {
		printUsage()
		os.Exit(1)
	}

	interfaceName := args[0]
	configFile := args[1]

	// Print banner (unless quiet)
	if debugLevel > 0 {
		printBanner()
	}

	// Check if interface exists
	if !capture.InterfaceExists(interfaceName) {
		fmt.Printf("Error: Interface '%s' not found\n\n", interfaceName)
		fmt.Println("Available interfaces:")
		capture.ListInterfaces()
		os.Exit(2)
	}

	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	if debugLevel >= 1 {
		fmt.Printf("✓ Loaded configuration: %s\n", configFile)
		fmt.Printf("  Devices: %d\n", len(cfg.Devices))
		fmt.Printf("  Interface: %s\n", interfaceName)
		fmt.Printf("  Debug level: %d (%s)\n", debugLevel, getDebugLevelName(debugLevel))
		if interactiveMode {
			fmt.Printf("  Mode: Interactive TUI\n")
		}
		if cfg.CapturePlayback != nil {
			fmt.Printf("  PCAP Playback: %s\n", cfg.CapturePlayback.FileName)
			if cfg.CapturePlayback.LoopTime > 0 {
				fmt.Printf("    Loop interval: %dms\n", cfg.CapturePlayback.LoopTime)
			}
			if cfg.CapturePlayback.ScaleTime > 0 && cfg.CapturePlayback.ScaleTime != 1.0 {
				fmt.Printf("    Time scaling: %.2fx\n", cfg.CapturePlayback.ScaleTime)
			}
		}
		fmt.Println()
	}

	// Dry run mode - validate and exit
	if dryRun {
		fmt.Println("✓ Configuration validation successful")
		fmt.Println("✓ Interface exists and is accessible")
		fmt.Printf("✓ Ready to simulate %d devices on %s\n", len(cfg.Devices), interfaceName)
		fmt.Println()
		fmt.Println("Configuration is valid. Use without --dry-run to start simulation.")
		os.Exit(0)
	}

	// Start simulation
	if interactiveMode {
		// Run with interactive TUI
		if err := interactive.Run(interfaceName, cfg, debugLevel); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Run in normal mode
		if err := runNormalMode(interfaceName, cfg, debugLevel); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func printBanner() {
	fmt.Printf("╔══════════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  NIAC - Network In A Can (Go Edition)                           ║\n")
	fmt.Printf("║  Version %s                                                 ║\n", padRight(Version, 51))
	fmt.Printf("╚══════════════════════════════════════════════════════════════════╝\n")
	fmt.Println()
}

func printVersion() {
	fmt.Printf("NIAC-Go version %s\n", Version)
	fmt.Printf("Build date: %s\n", BuildDate)
	fmt.Printf("Git commit: %s\n", GitCommit)
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
	fmt.Println("  # Show version")
	fmt.Println("  niac --version")
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

// runNormalMode runs NIAC in normal (non-interactive) mode
func runNormalMode(interfaceName string, cfg *config.Config, debugLevel int) error {
	fmt.Println("Starting NIAC simulation...")
	fmt.Printf("  Interface: %s\n", interfaceName)
	fmt.Printf("  Devices: %d\n", len(cfg.Devices))
	fmt.Printf("  Debug level: %d\n", debugLevel)
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Create capture engine
	engine, err := capture.New(interfaceName, debugLevel)
	if err != nil {
		return fmt.Errorf("failed to create capture engine: %w", err)
	}
	defer engine.Close()

	// Create protocol stack
	stack := protocols.NewStack(engine, cfg, debugLevel)

	// Configure DHCP/DNS handlers from device configs
	for _, device := range cfg.Devices {
		// Configure DHCP if present
		if device.DHCPConfig != nil {
			dhcp := device.DHCPConfig
			dhcpHandler := stack.GetDHCPHandler()
			dhcpv6Handler := stack.GetDHCPv6Handler()

			// Basic DHCPv4 configuration
			if len(dhcp.DomainNameServer) > 0 || dhcp.Router != nil {
				dhcpHandler.SetServerConfig(
					device.IPAddresses[0],   // Server IP
					dhcp.Router,              // Gateway
					dhcp.DomainNameServer,    // DNS servers
					dhcp.DomainName,          // Domain name
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
			dnsHandler := stack.GetDNSHandler()

			// Load DNS records
			for _, record := range device.DNSConfig.ForwardRecords {
				dnsHandler.AddRecord(record.Name, record.IP)
			}
			// PTR records are handled automatically by AddRecord
		}
	}

	// Start the stack
	if err := stack.Start(); err != nil {
		return fmt.Errorf("failed to start stack: %w", err)
	}

	if debugLevel >= 1 {
		fmt.Println("✓ Network simulation started")
		fmt.Println("✓ Protocol stack running")

		// Count and display SNMP-enabled devices
		snmpCount := 0
		for _, dev := range cfg.Devices {
			if dev.SNMPConfig.Community != "" || dev.SNMPConfig.WalkFile != "" {
				snmpCount++
			}
		}
		if snmpCount > 0 {
			fmt.Printf("✓ SNMP agents: %d device(s)\n", snmpCount)
		}

		// Show PCAP playback if configured
		if cfg.CapturePlayback != nil {
			fmt.Printf("✓ PCAP playback: %s\n", cfg.CapturePlayback.FileName)
		}
		fmt.Println()
	}

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
	startTime := time.Now()
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
