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

	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

func addGenerateCommand(configCmd *cobra.Command) {
	generateCmd := &cobra.Command{
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
		RunE: func(_ *cobra.Command, args []string) error {
			return runGenerate(args)
		},
	}

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

const defaultGenerateOutputFile = "config.yaml"

// maxIPOctet is the maximum value for the last octet of an IPv4 address (avoiding broadcast).
const maxIPOctet = 254

type deviceTypeOption struct {
	key   string
	label string
}

func deviceTypeOptions() []deviceTypeOption {
	return []deviceTypeOption{
		{"1", "router"},
		{"2", "switch"},
		{"3", "access-point"},
		{"4", "server"},
		{"5", "workstation"},
		{"6", "firewall"},
	}
}

func runGenerate(args []string) error {
	reader := bufio.NewReader(os.Stdin)
	printGeneratorHeader()

	cfg, err := collectNetworkInfo(reader)
	if err != nil {
		return stopIfCancelled(err)
	}

	devices, err := collectDevices(reader, cfg)
	if err != nil {
		return stopIfCancelled(err)
	}
	cfg.devices = devices

	outputFile, err := chooseOutputFile(args, reader)
	if err != nil {
		if errors.Is(err, errGenerateAborted) {
			return nil
		}
		return stopIfCancelled(err)
	}

	if writeErr := writeConfiguration(outputFile, cfg); writeErr != nil {
		return writeErr
	}

	printSummary(outputFile, cfg)

	return nil
}

func printGeneratorHeader() {
	_, _ = color.New(color.Bold, color.FgCyan).
		Println("\n╔════════════════════════════════════════════════════════════╗")
	_, _ = color.New(color.Bold, color.FgCyan).
		Println("║      NIAC Configuration Generator (v1.19.0)               ║")
	_, _ = color.New(color.Bold, color.FgCyan).
		Print("╚════════════════════════════════════════════════════════════╝\n\n")

	color.Yellow("This wizard will guide you through creating a complete YAML")
	color.Yellow("configuration file for your network simulation.\n\n")
}

func collectNetworkInfo(reader *bufio.Reader) (*generatedConfig, error) {
	_, _ = color.New(color.Bold, color.FgCyan).Println("Step 1: Network Information")
	color.White("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	cfg := new(generatedConfig)
	cfg.devices = make([]generatedDevice, 0)

	var err error
	cfg.networkName, err = promptString(reader, color.CyanString("Network name: "), "simulation-network")
	if err != nil {
		return nil, err
	}
	cfg.subnet, err = promptString(
		reader,
		color.CyanString("Network subnet (CIDR, e.g., 192.168.1.0/24): "),
		"192.168.1.0/24",
	)
	if err != nil {
		return nil, err
	}
	cfg.includePath, err = promptString(
		reader, color.CyanString("Path for SNMP walk files (leave empty for none): "), "",
	)
	if err != nil {
		return nil, err
	}
	fmt.Fprintln(os.Stdout)

	return cfg, nil
}

func collectDevices(reader *bufio.Reader, cfg *generatedConfig) ([]generatedDevice, error) {
	_, _ = color.New(color.Bold, color.FgCyan).Println("Step 2: Device Configuration")
	color.White("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	deviceCount, err := mustPromptInt(reader, "How many devices to create (1-20): ", 1, maxDeviceCount)
	if err != nil {
		return nil, err
	}
	fmt.Fprintln(os.Stdout)

	devices := make([]generatedDevice, 0, deviceCount)
	for i := range deviceCount {
		_, _ = color.New(color.Bold, color.FgYellow).Printf("Device %d/%d:\n", i+1, deviceCount)
		color.White("──────────────────────────────────────────────────────────────\n")

		device := generatedDevice{
			protocols: make(map[string]protocolConfig),
		}

		device.devType, err = promptDeviceType(reader)
		if err != nil {
			return nil, err
		}
		device.name, err = promptDeviceName(reader, device.devType, i+1)
		if err != nil {
			return nil, err
		}
		device.ip, err = promptDeviceIP(reader, cfg.subnet, i+1)
		if err != nil {
			return nil, err
		}
		device.mac, err = promptDeviceMAC(reader, i+1)
		if err != nil {
			return nil, err
		}

		color.Cyan("Select protocols to enable for %s:\n", device.name)
		device.protocols, err = selectProtocols(reader, device.devType)
		if err != nil {
			return nil, err
		}

		devices = append(devices, device)
		fmt.Fprintln(os.Stdout)
	}

	return devices, nil
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
	if len(parts) != cidrParts {
		return fmt.Sprintf("192.168.1.%d", deviceNum+baseIPOffset)
	}

	ip := net.ParseIP(parts[0])
	if ip == nil {
		return fmt.Sprintf("192.168.1.%d", deviceNum+baseIPOffset)
	}

	ip4 := ip.To4()
	if ip4 == nil {
		return fmt.Sprintf("192.168.1.%d", deviceNum+baseIPOffset)
	}

	// Increment last octet with overflow protection
	newOctet := int(ip4[3]) + deviceNum + baseIPOffset
	if newOctet > maxIPOctet {
		// Advance to next subnet for overflow
		ip4[2] = safeconv.Byte(int(ip4[2]) + newOctet/(maxIPOctet+1))
		ip4[3] = safeconv.Byte(newOctet % (maxIPOctet + 1))
	} else {
		ip4[3] = safeconv.Byte(newOctet)
	}

	return ip4.String()
}

func generateDefaultMAC(deviceNum int) string {
	return fmt.Sprintf("02:00:00:00:00:%02x", deviceNum)
}

func selectProtocols(reader *bufio.Reader, devType string) (map[string]protocolConfig, error) {
	protocols := make(map[string]protocolConfig)

	sections := []func(*bufio.Reader, string, map[string]protocolConfig) error{
		selectDiscoveryProtocols,
		selectManagementProtocols,
		selectNetworkServices,
		selectApplicationProtocols,
	}

	for _, section := range sections {
		if err := section(reader, devType, protocols); err != nil {
			return nil, err
		}
	}

	return protocols, nil
}

// selectDiscoveryProtocols prompts for LLDP/CDP.
func selectDiscoveryProtocols(reader *bufio.Reader, _ string, protocols map[string]protocolConfig) error {
	fmt.Fprintln(os.Stdout)
	color.Yellow("Discovery Protocols:")
	lldp, err := mustPromptYesNo(reader, "  Enable LLDP? (y/n): ")
	if err != nil {
		return err
	}
	if lldp {
		protocols["lldp"] = protocolConfig{
			enabled: true,
			params: map[string]string{
				"advertise_interval": "30",
				"ttl":                "120",
			},
		}
	}
	cdp, err := mustPromptYesNo(reader, "  Enable CDP? (y/n): ")
	if err != nil {
		return err
	}
	if cdp {
		protocols["cdp"] = protocolConfig{
			enabled: true,
			params: map[string]string{
				"advertise_interval": "60",
				"holdtime":           "180",
			},
		}
	}

	return nil
}

// selectManagementProtocols prompts for SNMP.
func selectManagementProtocols(reader *bufio.Reader, _ string, protocols map[string]protocolConfig) error {
	fmt.Fprintln(os.Stdout)
	color.Yellow("Management Protocols:")
	snmp, err := mustPromptYesNo(reader, "  Enable SNMP? (y/n): ")
	if err != nil {
		return err
	}
	if snmp {
		community, promptErr := promptString(reader, "    SNMP community [public]: ", "public")
		if promptErr != nil {
			return promptErr
		}
		walkFile, walkErr := promptString(reader, "    Walk file (leave empty for none): ", "")
		if walkErr != nil {
			return walkErr
		}
		protocols["snmp"] = protocolConfig{
			enabled: true,
			params: map[string]string{
				"community": community,
				"walk_file": walkFile,
			},
		}
	}

	return nil
}

// selectNetworkServices prompts for DHCP/DNS on routers and servers.
func selectNetworkServices(reader *bufio.Reader, devType string, protocols map[string]protocolConfig) error {
	fmt.Fprintln(os.Stdout)
	color.Yellow("Network Services:")
	if devType == "router" || devType == "server" {
		dhcp, promptErr := mustPromptYesNo(reader, "  Enable DHCP server? (y/n): ")
		if promptErr != nil {
			return promptErr
		}
		if dhcp {
			protocols["dhcp"] = protocolConfig{
				enabled: true,
				params: map[string]string{
					"subnet_mask": "255.255.255.0",
					"router":      "",
				},
			}
		}
		dns, err := mustPromptYesNo(reader, "  Enable DNS server? (y/n): ")
		if err != nil {
			return err
		}
		if dns {
			protocols["dns"] = protocolConfig{
				enabled: true,
				params:  make(map[string]string),
			}
		}
	}

	return nil
}

// selectApplicationProtocols prompts for HTTP/FTP on servers and workstations.
func selectApplicationProtocols(reader *bufio.Reader, devType string, protocols map[string]protocolConfig) error {
	if devType == "server" || devType == "workstation" {
		fmt.Fprintln(os.Stdout)
		color.Yellow("Application Protocols:")
		httpEnabled, promptErr := mustPromptYesNo(reader, "  Enable HTTP server? (y/n): ")
		if promptErr != nil {
			return promptErr
		}
		if httpEnabled {
			protocols["http"] = protocolConfig{
				enabled: true,
				params: map[string]string{
					"server_name": "NIAC-Go/1.0.0",
				},
			}
		}
		ftpEnabled, err := mustPromptYesNo(reader, "  Enable FTP server? (y/n): ")
		if err != nil {
			return err
		}
		if ftpEnabled {
			protocols["ftp"] = protocolConfig{
				enabled: true,
				params: map[string]string{
					"allow_anonymous": "true",
				},
			}
		}
	}

	return nil
}

func generateYAML(cfg *generatedConfig) string {
	var sb strings.Builder

	writeYAMLHeader(&sb, cfg)
	writeYAMLDevices(&sb, cfg.devices)

	return sb.String()
}

func writeYAMLHeader(sb *strings.Builder, cfg *generatedConfig) {
	sb.WriteString("# NIAC Configuration File\n")
	_, _ = fmt.Fprintf(sb, "# Network: %s\n", cfg.networkName)
	_, _ = fmt.Fprintf(sb, "# Generated: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	sb.WriteString("# NIAC version: v1.19.0\n\n")

	if cfg.includePath != "" {
		_, _ = fmt.Fprintf(sb, "includePath: \"%s\"\n\n", cfg.includePath)
	}
}

func writeYAMLDevices(sb *strings.Builder, devices []generatedDevice) {
	sb.WriteString("devices:\n")
	for _, dev := range devices {
		_, _ = fmt.Fprintf(sb, "  - name: \"%s\"\n", dev.name)
		_, _ = fmt.Fprintf(sb, "    mac: \"%s\"\n", dev.mac)
		_, _ = fmt.Fprintf(sb, "    ips:\n      - \"%s\"\n", dev.ip)

		writeYAMLSNMP(sb, dev.protocols)
		writeYAMLLLDP(sb, dev)
		writeYAMLCDP(sb, dev)
		writeYAMLDHCP(sb, dev.protocols)
		writeYAMLDNS(sb, dev)
		writeYAMLHTTP(sb, dev.protocols)
		writeYAMLFTP(sb, dev.protocols)

		sb.WriteString("\n")
	}
}

func writeYAMLSNMP(sb *strings.Builder, protocols map[string]protocolConfig) {
	proto, ok := protocols["snmp"]
	if !ok || !proto.enabled {
		return
	}
	sb.WriteString("    snmpAgent:\n")
	_, _ = fmt.Fprintf(sb, "      community: \"%s\"\n", proto.params["community"])
	if proto.params["walk_file"] != "" {
		_, _ = fmt.Fprintf(sb, "      walkFile: \"%s\"\n", proto.params["walk_file"])
	}
}

func writeYAMLLLDP(sb *strings.Builder, dev generatedDevice) {
	proto, ok := dev.protocols["lldp"]
	if !ok || !proto.enabled {
		return
	}
	sb.WriteString("    lldp:\n")
	sb.WriteString("      enabled: true\n")
	_, _ = fmt.Fprintf(sb, "      advertiseInterval: %s\n", proto.params["advertise_interval"])
	_, _ = fmt.Fprintf(sb, "      ttl: %s\n", proto.params["ttl"])
	_, _ = fmt.Fprintf(sb, "      systemDescription: \"%s on %s\"\n", dev.devType, dev.name)
}

func writeYAMLCDP(sb *strings.Builder, dev generatedDevice) {
	proto, ok := dev.protocols["cdp"]
	if !ok || !proto.enabled {
		return
	}
	sb.WriteString("    cdp:\n")
	sb.WriteString("      enabled: true\n")
	_, _ = fmt.Fprintf(sb, "      advertiseInterval: %s\n", proto.params["advertise_interval"])
	_, _ = fmt.Fprintf(sb, "      holdtime: %s\n", proto.params["holdtime"])
	_, _ = fmt.Fprintf(sb, "      platform: \"NIAC %s\"\n", dev.devType)
}

func writeYAMLDHCP(sb *strings.Builder, protocols map[string]protocolConfig) {
	proto, ok := protocols["dhcp"]
	if !ok || !proto.enabled {
		return
	}
	sb.WriteString("    dhcp:\n")
	_, _ = fmt.Fprintf(sb, "      subnetMask: \"%s\"\n", proto.params["subnet_mask"])
	if proto.params["router"] != "" {
		_, _ = fmt.Fprintf(sb, "      router: \"%s\"\n", proto.params["router"])
	}
}

func writeYAMLDNS(sb *strings.Builder, dev generatedDevice) {
	proto, ok := dev.protocols["dns"]
	if !ok || !proto.enabled {
		return
	}
	sb.WriteString("    dns:\n")
	sb.WriteString("      forwardRecords:\n")
	_, _ = fmt.Fprintf(sb, "        - name: \"%s.local\"\n", dev.name)
	_, _ = fmt.Fprintf(sb, "          ip: \"%s\"\n", dev.ip)
	sb.WriteString("          ttl: 3600\n")
}

func writeYAMLHTTP(sb *strings.Builder, protocols map[string]protocolConfig) {
	proto, ok := protocols["http"]
	if !ok || !proto.enabled {
		return
	}
	sb.WriteString("    http:\n")
	sb.WriteString("      enabled: true\n")
	_, _ = fmt.Fprintf(sb, "      serverName: \"%s\"\n", proto.params["server_name"])
}

func writeYAMLFTP(sb *strings.Builder, protocols map[string]protocolConfig) {
	proto, ok := protocols["ftp"]
	if !ok || !proto.enabled {
		return
	}
	sb.WriteString("    ftp:\n")
	sb.WriteString("      enabled: true\n")
	_, _ = fmt.Fprintf(sb, "      allowAnonymous: %s\n", proto.params["allow_anonymous"])
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

// errGenerateAborted marks the operator declining to overwrite an existing
// file, so runGenerate can stop cleanly instead of treating it as a failure.
var errGenerateAborted = errors.New("generation aborted")

func chooseOutputFile(args []string, reader *bufio.Reader) (string, error) {
	var output string

	if len(args) > 0 {
		output = args[0]
	} else {
		fmt.Fprintln(os.Stdout)
		color.Yellow("Step 3: Save Configuration")

		var err error
		output, err = promptString(
			reader,
			fmt.Sprintf("  Output filename [%s]: ", defaultGenerateOutputFile),
			defaultGenerateOutputFile,
		)
		if err != nil {
			return "", err
		}
	}

	cleaned, pathErr := validateCLIPath(output)
	if pathErr != nil {
		return "", fmt.Errorf("invalid output path: %w", pathErr)
	}
	output = cleaned

	if _, statErr := statSafeFile(output); statErr == nil {
		fmt.Fprintln(os.Stdout)
		color.Yellow("Warning: %s already exists!", output)
		overwrite, err := mustPromptYesNo(reader, "Overwrite? (y/n): ")
		if err != nil {
			return "", err
		}
		if !overwrite {
			color.Red("Aborted.")
			return "", errGenerateAborted
		}
	}

	return output, nil
}

func writeConfiguration(outputFile string, cfg *generatedConfig) error {
	if err := validateFilePath(outputFile, true); err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}

	yaml := generateYAML(cfg)
	if err := writeSafeFile(outputFile, []byte(yaml)); err != nil {
		return fmt.Errorf("writing configuration: %w", err)
	}

	return nil
}

func printSummary(outputFile string, cfg *generatedConfig) {
	fmt.Fprintln(os.Stdout)
	color.Green("✓ Configuration generated: %s", outputFile)
	fmt.Fprintf(os.Stdout, "  Network: %s\n", cfg.networkName)
	fmt.Fprintf(os.Stdout, "  Subnet: %s\n", cfg.subnet)
	fmt.Fprintf(os.Stdout, "  Devices: %d\n", len(cfg.devices))
	if cfg.includePath != "" {
		fmt.Fprintf(os.Stdout, "  SNMP walk path: %s\n", cfg.includePath)
	}

	totalProtocols := 0
	for _, device := range cfg.devices {
		totalProtocols += countEnabledProtocols(device.protocols)
	}
	fmt.Fprintf(os.Stdout, "  Enabled protocols: %d\n", totalProtocols)
	fmt.Fprintln(os.Stdout)

	_, _ = color.New(color.Bold).Println("Next steps:")
	_, _ = fmt.Fprintf(
		os.Stdout,
		"  %s\n",
		color.CyanString("niac validate %s", outputFile),
	)
	_, _ = fmt.Fprintf(
		os.Stdout,
		"  %s\n",
		color.CyanString("sudo niac daemon --once en0 %s", outputFile),
	)
}

func promptDeviceType(reader *bufio.Reader) (string, error) {
	fmt.Fprintln(os.Stdout, "    Select a device type:")
	options := deviceTypeOptions()
	for _, option := range options {
		fmt.Fprintf(os.Stdout, "      %s) %s\n", option.key, option.label)
	}

	keys := make([]string, len(options))
	for i, option := range options {
		keys[i] = option.key
	}

	choice, err := mustPromptChoice(reader, "    Choice (1-6): ", keys)
	if err != nil {
		return "", err
	}
	if devType := mapDeviceType(choice); devType != "" {
		return devType, nil
	}

	color.Red("Invalid device type selected")
	return promptDeviceType(reader)
}

func promptDeviceName(reader *bufio.Reader, devType string, deviceNum int) (string, error) {
	defaultName := fmt.Sprintf("%s-%d", devType, deviceNum)
	return promptString(reader, fmt.Sprintf("    Name [%s]: ", defaultName), defaultName)
}

func promptDeviceIP(reader *bufio.Reader, subnet string, deviceNum int) (string, error) {
	defaultIP := generateDefaultIP(subnet, deviceNum)
	for {
		ip, err := promptString(reader, fmt.Sprintf("    IP [%s]: ", defaultIP), defaultIP)
		if err != nil {
			return "", err
		}
		if net.ParseIP(ip) != nil {
			return ip, nil
		}
		color.Red("Please enter a valid IP address")
	}
}

func promptDeviceMAC(reader *bufio.Reader, deviceNum int) (string, error) {
	defaultMAC := generateDefaultMAC(deviceNum)
	for {
		mac, err := promptString(reader, fmt.Sprintf("    MAC [%s]: ", defaultMAC), defaultMAC)
		if err != nil {
			return "", err
		}
		if _, macErr := net.ParseMAC(mac); macErr == nil {
			return mac, nil
		}
		color.Red("Please enter a valid MAC address")
	}
}

func promptString(reader *bufio.Reader, prompt string, defaultValue string) (string, error) {
	fmt.Fprint(os.Stdout, prompt)
	input, err := readLine(reader)
	if err != nil {
		if errors.Is(err, io.EOF) && defaultValue != "" {
			return defaultValue, nil
		}
		return "", inputError(err)
	}
	if input == "" {
		return defaultValue, nil
	}
	return input, nil
}

// promptInt and promptChoice are already defined in init.go
// promptYesNo is already defined in init.go
