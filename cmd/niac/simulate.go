package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/krisarmstrong/niac-go/internal/capture"
	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/interactive"
	"github.com/krisarmstrong/niac-go/internal/logging"
	"github.com/krisarmstrong/niac-go/internal/protocols"
)

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
