package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/krisarmstrong/niac-go/internal/config"
)

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
	if len(str) >= length {
		return str[:length]
	}
	return str + strings.Repeat(" ", length-len(str))
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
