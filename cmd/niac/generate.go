package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate [output-file]",
	Short: "Interactive configuration generator",
	Long: `Interactive configuration generator for NIAC.

Prompts you for all configuration details and generates a complete YAML
configuration file. More detailed than 'niac init' template wizard.

The generator will ask you for:
  - Network name and subnet
  - Number of devices
  - Device details (type, name, IP, MAC)
  - Protocols to enable (LLDP, CDP, SNMP, DHCP, DNS, etc.)
  - Protocol-specific configuration`,
	Example: `  # Generate configuration interactively
  niac config generate

  # Generate with specific output file
  niac config generate my-network.yaml

  # Validate and run
  niac config generate network.yaml && niac validate network.yaml`,
	Run: runGenerate,
}

func init() {
	configCmd.AddCommand(generateCmd)
}

type generatedConfig struct {
	networkName string
	subnet      string
	devices     []generatedDevice
	includePath string
}

type generatedDevice struct {
	name      string
	devType   string
	ip        string
	mac       string
	protocols map[string]protocolConfig
}

type protocolConfig struct {
	enabled bool
	params  map[string]string
}

func runGenerate(cmd *cobra.Command, args []string) {
	reader := bufio.NewReader(os.Stdin)

	// Print header
	color.New(color.Bold, color.FgCyan).Println("\n╔════════════════════════════════════════════════════════════╗")
	color.New(color.Bold, color.FgCyan).Println("║      NIAC Configuration Generator (v1.19.0)               ║")
	color.New(color.Bold, color.FgCyan).Print("╚════════════════════════════════════════════════════════════╝\n\n")

	color.Yellow("This wizard will guide you through creating a complete YAML")
	color.Yellow("configuration file for your network simulation.\n\n")

	// Step 1: Network Information
	color.New(color.Bold, color.FgCyan).Println("Step 1: Network Information")
	color.White("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	cfg := &generatedConfig{
		devices: make([]generatedDevice, 0),
	}

	// Network name
	cfg.networkName = promptString(reader, color.CyanString("Network name: "), "simulation-network")

	// Subnet
	cfg.subnet = promptString(reader, color.CyanString("Network subnet (CIDR, e.g., 192.168.1.0/24): "), "192.168.1.0/24")

	// Include path for walk files
	cfg.includePath = promptString(reader, color.CyanString("Path for SNMP walk files (leave empty for none): "), "")

	fmt.Println()

	// Step 2: Device Configuration
	color.New(color.Bold, color.FgCyan).Println("Step 2: Device Configuration")
	color.White("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	deviceCount := mustPromptInt(reader, "How many devices to create (1-20): ", 1, 20)
	fmt.Println()

	for i := 0; i < deviceCount; i++ {
		color.New(color.Bold, color.FgYellow).Printf("Device %d/%d:\n", i+1, deviceCount)
		color.White("──────────────────────────────────────────────────────────────\n")

		device := generatedDevice{
			protocols: make(map[string]protocolConfig),
		}

		// Device type
		fmt.Println(color.CyanString("Device type:"))
		fmt.Println("  1) router       2) switch       3) access-point")
		fmt.Println("  4) server       5) workstation  6) firewall")
		typeChoice := mustPromptChoice(reader, "Select type (1-6): ", []string{"1", "2", "3", "4", "5", "6"})
		device.devType = mapDeviceType(typeChoice)

		// Device name
		defaultName := fmt.Sprintf("%s-%d", device.devType, i+1)
		device.name = promptString(reader, fmt.Sprintf("Device name [%s]: ", defaultName), defaultName)

		// IP address
		defaultIP := generateDefaultIP(cfg.subnet, i+1)
		device.ip = promptString(reader, fmt.Sprintf("IP address [%s]: ", defaultIP), defaultIP)

		// MAC address
		defaultMAC := generateDefaultMAC(i + 1)
		device.mac = promptString(reader, fmt.Sprintf("MAC address [%s]: ", defaultMAC), defaultMAC)

		fmt.Println()

		// Protocol selection
		color.Cyan("Select protocols to enable for %s:\n", device.name)
		device.protocols = selectProtocols(reader, device.devType)

		cfg.devices = append(cfg.devices, device)
		fmt.Println()
	}

	// Step 3: Output File
	color.New(color.Bold, color.FgCyan).Println("Step 3: Save Configuration")
	color.White("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	var outputFile string
	if len(args) > 0 {
		outputFile = args[0]
	} else {
		outputFile = promptString(reader, "Output filename [config.yaml]: ", "config.yaml")
	}

	// Check if file exists
	if _, err := os.Stat(outputFile); err == nil {
		color.Yellow("Warning: File %s already exists!\n", outputFile)
		if !mustPromptYesNo(reader, "Overwrite? (y/n): ") {
			color.Red("Aborted.\n")
			os.Exit(0)
		}
	}

	// Generate YAML
	yamlContent := generateYAML(cfg)

	// Write to file
	if err := os.WriteFile(outputFile, []byte(yamlContent), 0644); err != nil {
		color.Red("Error writing file: %v\n", err)
		os.Exit(1)
	}

	// Success message
	fmt.Println()
	color.Green("✓ Successfully created %s\n", outputFile)
	fmt.Println()

	// Summary
	color.New(color.Bold).Println("Configuration Summary:")
	fmt.Printf("  Network:  %s (%s)\n", cfg.networkName, cfg.subnet)
	fmt.Printf("  Devices:  %d\n", len(cfg.devices))
	for i, dev := range cfg.devices {
		enabledProtos := countEnabledProtocols(dev.protocols)
		fmt.Printf("    %d. %s (%s) - %s - %d protocol(s)\n", i+1, dev.name, dev.devType, dev.ip, enabledProtos)
	}
	fmt.Println()

	// Next steps
	color.New(color.Bold).Println("Next Steps:")
	fmt.Println()
	fmt.Println("1. Validate your configuration:")
	fmt.Printf("   %s\n", color.CyanString("niac validate %s", outputFile))
	fmt.Println()
	fmt.Println("2. Run the simulation:")
	fmt.Printf("   %s\n", color.CyanString("sudo niac interactive en0 %s", outputFile))
	fmt.Println()
}

func mapDeviceType(choice string) string {
	types := map[string]string{
		"1": "router",
		"2": "switch",
		"3": "access-point",
		"4": "server",
		"5": "workstation",
		"6": "firewall",
	}
	return types[choice]
}

func generateDefaultIP(subnet string, deviceNum int) string {
	// Parse subnet to extract base IP
	parts := strings.Split(subnet, "/")
	if len(parts) != 2 {
		return fmt.Sprintf("192.168.1.%d", deviceNum+10)
	}

	ip := net.ParseIP(parts[0])
	if ip == nil {
		return fmt.Sprintf("192.168.1.%d", deviceNum+10)
	}

	ip4 := ip.To4()
	if ip4 == nil {
		return fmt.Sprintf("192.168.1.%d", deviceNum+10)
	}

	// Increment last octet
	ip4[3] = byte(int(ip4[3]) + deviceNum + 10)
	return ip4.String()
}

func generateDefaultMAC(deviceNum int) string {
	return fmt.Sprintf("02:00:00:00:00:%02x", deviceNum)
}

func selectProtocols(reader *bufio.Reader, devType string) map[string]protocolConfig {
	protocols := make(map[string]protocolConfig)

	// Discovery protocols
	fmt.Println()
	color.Yellow("Discovery Protocols:")
	if mustPromptYesNo(reader, "  Enable LLDP? (y/n): ") {
		protocols["lldp"] = protocolConfig{
			enabled: true,
			params: map[string]string{
				"advertise_interval": "30",
				"ttl":                "120",
			},
		}
	}
	if mustPromptYesNo(reader, "  Enable CDP? (y/n): ") {
		protocols["cdp"] = protocolConfig{
			enabled: true,
			params: map[string]string{
				"advertise_interval": "60",
				"holdtime":           "180",
			},
		}
	}

	// Management protocols
	fmt.Println()
	color.Yellow("Management Protocols:")
	if mustPromptYesNo(reader, "  Enable SNMP? (y/n): ") {
		community := promptString(reader, "    SNMP community [public]: ", "public")
		walkFile := promptString(reader, "    Walk file (leave empty for none): ", "")
		protocols["snmp"] = protocolConfig{
			enabled: true,
			params: map[string]string{
				"community": community,
				"walk_file": walkFile,
			},
		}
	}

	// Network services
	fmt.Println()
	color.Yellow("Network Services:")
	if devType == "router" || devType == "server" {
		if mustPromptYesNo(reader, "  Enable DHCP server? (y/n): ") {
			protocols["dhcp"] = protocolConfig{
				enabled: true,
				params: map[string]string{
					"subnet_mask": "255.255.255.0",
					"router":      "",
				},
			}
		}
		if mustPromptYesNo(reader, "  Enable DNS server? (y/n): ") {
			protocols["dns"] = protocolConfig{
				enabled: true,
				params:  make(map[string]string),
			}
		}
	}

	// Application protocols
	if devType == "server" || devType == "workstation" {
		fmt.Println()
		color.Yellow("Application Protocols:")
		if mustPromptYesNo(reader, "  Enable HTTP server? (y/n): ") {
			protocols["http"] = protocolConfig{
				enabled: true,
				params: map[string]string{
					"server_name": "NIAC-Go/1.0.0",
				},
			}
		}
		if mustPromptYesNo(reader, "  Enable FTP server? (y/n): ") {
			protocols["ftp"] = protocolConfig{
				enabled: true,
				params: map[string]string{
					"allow_anonymous": "true",
				},
			}
		}
	}

	return protocols
}

func generateYAML(cfg *generatedConfig) string {
	var sb strings.Builder

	// Header comment
	sb.WriteString("# NIAC Configuration File\n")
	sb.WriteString(fmt.Sprintf("# Network: %s\n", cfg.networkName))
	sb.WriteString(fmt.Sprintf("# Generated: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("# NIAC version: v1.19.0\n\n"))

	// Include path
	if cfg.includePath != "" {
		sb.WriteString(fmt.Sprintf("includePath: \"%s\"\n\n", cfg.includePath))
	}

	// Devices
	sb.WriteString("devices:\n")
	for _, dev := range cfg.devices {
		sb.WriteString(fmt.Sprintf("  - name: \"%s\"\n", dev.name))
		sb.WriteString(fmt.Sprintf("    mac: \"%s\"\n", dev.mac))
		sb.WriteString(fmt.Sprintf("    ip: \"%s\"\n", dev.ip))

		// SNMP configuration
		if proto, ok := dev.protocols["snmp"]; ok && proto.enabled {
			sb.WriteString("    snmpAgent:\n")
			sb.WriteString(fmt.Sprintf("      community: \"%s\"\n", proto.params["community"]))
			if proto.params["walk_file"] != "" {
				sb.WriteString(fmt.Sprintf("      walkFile: \"%s\"\n", proto.params["walk_file"]))
			}
		}

		// LLDP configuration
		if proto, ok := dev.protocols["lldp"]; ok && proto.enabled {
			sb.WriteString("    lldp:\n")
			sb.WriteString("      enabled: true\n")
			sb.WriteString(fmt.Sprintf("      advertiseInterval: %s\n", proto.params["advertise_interval"]))
			sb.WriteString(fmt.Sprintf("      ttl: %s\n", proto.params["ttl"]))
			sb.WriteString(fmt.Sprintf("      systemDescription: \"%s on %s\"\n", dev.devType, dev.name))
		}

		// CDP configuration
		if proto, ok := dev.protocols["cdp"]; ok && proto.enabled {
			sb.WriteString("    cdp:\n")
			sb.WriteString("      enabled: true\n")
			sb.WriteString(fmt.Sprintf("      advertiseInterval: %s\n", proto.params["advertise_interval"]))
			sb.WriteString(fmt.Sprintf("      holdtime: %s\n", proto.params["holdtime"]))
			sb.WriteString(fmt.Sprintf("      platform: \"NIAC %s\"\n", dev.devType))
		}

		// DHCP configuration
		if proto, ok := dev.protocols["dhcp"]; ok && proto.enabled {
			sb.WriteString("    dhcp:\n")
			sb.WriteString(fmt.Sprintf("      subnetMask: \"%s\"\n", proto.params["subnet_mask"]))
			if proto.params["router"] != "" {
				sb.WriteString(fmt.Sprintf("      router: \"%s\"\n", proto.params["router"]))
			}
		}

		// DNS configuration
		if proto, ok := dev.protocols["dns"]; ok && proto.enabled {
			sb.WriteString("    dns:\n")
			sb.WriteString("      forwardRecords:\n")
			sb.WriteString(fmt.Sprintf("        - name: \"%s.local\"\n", dev.name))
			sb.WriteString(fmt.Sprintf("          ip: \"%s\"\n", dev.ip))
			sb.WriteString("          ttl: 3600\n")
		}

		// HTTP configuration
		if proto, ok := dev.protocols["http"]; ok && proto.enabled {
			sb.WriteString("    http:\n")
			sb.WriteString("      enabled: true\n")
			sb.WriteString(fmt.Sprintf("      serverName: \"%s\"\n", proto.params["server_name"]))
		}

		// FTP configuration
		if proto, ok := dev.protocols["ftp"]; ok && proto.enabled {
			sb.WriteString("    ftp:\n")
			sb.WriteString("      enabled: true\n")
			sb.WriteString(fmt.Sprintf("      allowAnonymous: %s\n", proto.params["allow_anonymous"]))
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

func countEnabledProtocols(protocols map[string]protocolConfig) int {
	count := 0
	for _, proto := range protocols {
		if proto.enabled {
			count++
		}
	}
	return count
}

func promptString(reader *bufio.Reader, prompt string, defaultValue string) string {
	for {
		fmt.Print(prompt)
		input, err := readLine(reader)
		if err != nil {
			if errors.Is(err, io.EOF) && defaultValue != "" {
				return defaultValue
			}
			handleInputError(err)
			continue
		}
		if input == "" {
			return defaultValue
		}
		return input
	}
}

// promptInt and promptChoice are already defined in init.go
// promptYesNo is already defined in init.go
