package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/krisarmstrong/niac-go/pkg/capture"
	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/interactive"
)

const (
	Version      = "1.0.0-go"
	Enhancements = "Go Rewrite, Interactive Error Injection, Native Performance"
)

func main() {
	// Command line flags
	var (
		debugLevel     int
		interactiveMode bool
	)

	flag.IntVar(&debugLevel, "d", 1, "Debug level (0-3)")
	flag.BoolVar(&interactiveMode, "i", false, "Enable interactive error injection mode")
	flag.BoolVar(&interactiveMode, "interactive", false, "Enable interactive error injection mode")
	flag.Parse()

	fmt.Printf("NIAC Network in a Can (Go Edition) - Version %s\n", Version)
	fmt.Printf("Enhancements: %s\n", Enhancements)
	fmt.Printf("Runtime: Go %s on %s/%s\n\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)

	// Check arguments
	args := flag.Args()
	if len(args) < 2 {
		printUsage()
		os.Exit(1)
	}

	interfaceName := args[0]
	configFile := args[1]

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

	fmt.Printf("Loaded configuration: %s\n", configFile)
	fmt.Printf("  Devices: %d\n", len(cfg.Devices))
	fmt.Printf("  Debug level: %d\n", debugLevel)
	fmt.Printf("  Interactive mode: %v\n\n", interactiveMode)

	// Start simulation
	if interactiveMode {
		// Run with interactive error injection
		if err := interactive.Run(interfaceName, cfg, debugLevel); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Run in normal mode
		fmt.Println("Starting NIAC simulation...")
		fmt.Println("Press Ctrl+C to stop")

		// TODO: Implement normal simulation mode
		fmt.Println("Normal mode not yet implemented - use --interactive for now")
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("USAGE: niac [-d<n>] [-i|--interactive] <interface_name> <network.cfg>")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -d<n>              Debug level (0-3)")
	fmt.Println("  -i, --interactive  Enable interactive error injection mode")
	fmt.Println()
	fmt.Println("Debug levels:")
	fmt.Println("  0 - no debug")
	fmt.Println("  1 - status (default)")
	fmt.Println("  2 - potential problems")
	fmt.Println("  3 - full detail")
	fmt.Println()
	fmt.Println("Available network interfaces:")
	capture.ListInterfaces()
}
