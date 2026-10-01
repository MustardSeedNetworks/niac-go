package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/templates"
)

func addScenarioCommand(root *cobra.Command, _ *serviceOptions) {
	scenarioCmd := &cobra.Command{
		Use:   "scenario",
		Short: "Manage built-in scenarios",
		Long:  `List, show, and copy the built-in scenarios that ship with NIAC.`,
		Example: `  # List all available scenarios
  niac scenario list

  # Show scenario contents
  niac scenario show basic-network

  # Create config from scenario
  niac scenario use small-office office.yaml

  # Apply scenario directly (validate and display info)
  niac scenario apply data-center`,
	}

	scenarioListCmd := &cobra.Command{
		Use:   "list",
		Short: "List available scenarios",
		Long: `Print every bundled scenario name with a one-line description.
They cover common small networks (basic-network, small-office, data-center,
iot-network, etc.) and are the fastest path to a runnable YAML config.`,
		Example: `  # List all scenarios with descriptions
  niac scenario list`,
		Run: func(_ *cobra.Command, _ []string) {
			runScenarioList()
		},
	}

	scenarioShowCmd := &cobra.Command{
		Use:   "show <scenario-name>",
		Short: "Show scenario contents",
		Long: `Print the YAML body of a named scenario to stdout. Useful for
inspecting what a scenario will produce or piping it into another tool
without writing to disk.`,
		Example: `  # Show basic network scenario
  niac scenario show basic-network

  # Show small office scenario
  niac scenario show small-office

  # Pipe to file
  niac scenario show data-center > my-config.yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runScenarioShow(args)
		},
	}

	scenarioUseCmd := &cobra.Command{
		Use:   "use <scenario-name> <output-file>",
		Short: "Copy scenario to a new file",
		Long: `Copy a named scenario's body into a new YAML file at the given
output path. The output file becomes the starting point you edit and run
with 'niac daemon --once'; the scenario itself is unchanged.`,
		Example: `  # Create small office config
  niac scenario use small-office office.yaml

  # Create IoT network config
  niac scenario use iot-network sensors.yaml

  # Create data center config
  niac scenario use data-center dc.yaml

  # Quick workflow
  niac scenario use basic-network config.yaml && niac validate config.yaml`,
		Args: cobra.ExactArgs(argsCountTwo),
		RunE: func(_ *cobra.Command, args []string) error {
			return runScenarioUse(args)
		},
	}

	scenarioApplyCmd := &cobra.Command{
		Use:   "apply <scenario-name>",
		Short: "Validate and display scenario information",
		Long: `Validate a scenario and display its configuration details.
This command loads the scenario, validates it, and shows what devices
and protocols it contains without creating a file.`,
		Example: `  # Validate basic network scenario
  niac scenario apply basic-network

  # Check data center scenario
  niac scenario apply data-center

  # Verify IoT network configuration
  niac scenario apply iot-network`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runScenarioApply(args)
		},
	}

	scenarioCmd.AddCommand(scenarioListCmd)
	scenarioCmd.AddCommand(scenarioShowCmd)
	scenarioCmd.AddCommand(scenarioUseCmd)
	scenarioCmd.AddCommand(scenarioApplyCmd)
	root.AddCommand(scenarioCmd)
}

func runScenarioList() {
	scenarioList := templates.List()

	_, _ = color.New(color.Bold).Println("Available Scenarios:")
	fmt.Fprintln(os.Stdout)

	// Find longest name for alignment
	maxLen := 0
	for _, t := range scenarioList {
		if len(t.Name) > maxLen {
			maxLen = len(t.Name)
		}
	}

	for _, t := range scenarioList {
		_, _ = color.New(color.FgCyan).Printf("  %-*s", maxLen+scenarioPadOffset, t.Name)
		fmt.Fprintf(os.Stdout, " - %s\n", t.Description)
		if t.UseCase != "" {
			fmt.Fprintf(os.Stdout, "  %*s   Use case: %s\n", maxLen, "", t.UseCase)
		}
	}

	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Usage:")
	fmt.Fprintln(os.Stdout, "  niac scenario show <scenario-name>         # View scenario content")
	fmt.Fprintln(os.Stdout, "  niac scenario use <scenario-name> <file>   # Create config from scenario")
	fmt.Fprintln(os.Stdout, "  niac scenario apply <scenario-name>        # Validate and show scenario info")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Quick start:")
	fmt.Fprintln(os.Stdout, "  niac init                                  # Interactive scenario wizard")
}

func runScenarioShow(args []string) error {
	scenarioName := args[0]

	tmpl, err := templates.Get(scenarioName)
	if err != nil {
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, "Available scenarios:")
		for _, name := range templates.ListNames() {
			fmt.Fprintf(os.Stdout, "  - %s\n", name)
		}
		return fmt.Errorf("loading scenario: %w", err)
	}

	fmt.Fprint(os.Stdout, tmpl.Content)

	return nil
}

func runScenarioUse(args []string) error {
	scenarioName := args[0]
	outputFile, pathErr := validateCLIPath(args[1])
	if pathErr != nil {
		return fmt.Errorf("invalid output path: %w", pathErr)
	}

	// Check if output file exists
	if _, statErr := statSafeFile(outputFile); statErr == nil {
		return fmt.Errorf("%w: %s", errOutputExists, outputFile)
	}

	// Get scenario
	tmpl, err := templates.Get(scenarioName)
	if err != nil {
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, "Available scenarios:")
		for _, name := range templates.ListNames() {
			fmt.Fprintf(os.Stdout, "  - %s\n", name)
		}
		return fmt.Errorf("loading scenario: %w", err)
	}

	// Write to file
	if writeErr := writeSafeFile(outputFile, []byte(tmpl.Content)); writeErr != nil {
		return fmt.Errorf("writing file: %w", writeErr)
	}

	color.Green("✓ Created %s from %s scenario", outputFile, scenarioName)
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "Description: %s\n", tmpl.Description)
	fmt.Fprintf(os.Stdout, "Use case: %s\n", tmpl.UseCase)
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Next steps:")
	fmt.Fprintf(os.Stdout, "  niac validate %s\n", outputFile)
	fmt.Fprintf(os.Stdout, "  sudo niac daemon --once en0 %s\n", outputFile)

	return nil
}

func runScenarioApply(args []string) error {
	scenarioName := args[0]

	tmpl, err := templates.Get(scenarioName)
	if err != nil {
		return fmt.Errorf("loading scenario: %w", err)
	}

	describeScenario(tmpl)

	cfg, cleanup, loadErr := loadAndValidateScenario(tmpl)
	defer cleanup()
	if loadErr != nil {
		return fmt.Errorf("scenario validation failed: %w", loadErr)
	}

	color.Green("✓ Scenario is valid")
	fmt.Fprintln(os.Stdout)

	describeDevices(scenarioName, cfg.Devices)

	return nil
}

func describeScenario(tmpl *templates.Template) {
	_, _ = color.New(color.Bold).Printf("Scenario: %s\n", tmpl.Name)
	fmt.Fprintf(os.Stdout, "Description: %s\n", tmpl.Description)
	fmt.Fprintf(os.Stdout, "Use case: %s\n", tmpl.UseCase)
	fmt.Fprintln(os.Stdout)
	_, _ = color.New(color.Bold).Println("Validating scenario...")
}

func loadAndValidateScenario(tmpl *templates.Template) (*config.Config, func(), error) {
	tmpFile, err := os.CreateTemp("", "niac-scenario-*.yaml")
	if err != nil {
		return nil, func() {}, fmt.Errorf("error creating temporary file: %w", err)
	}

	cleanup := func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	}

	if _, writeErr := tmpFile.WriteString(tmpl.Content); writeErr != nil {
		return nil, cleanup, fmt.Errorf("error writing temporary file: %w", writeErr)
	}
	_ = tmpFile.Close()

	cfg, loadErr := config.Load(tmpFile.Name())
	if loadErr != nil {
		return nil, cleanup, fmt.Errorf("config load error: %w", loadErr)
	}
	return cfg, cleanup, nil
}

func describeDevices(scenarioName string, devices []config.Device) {
	_, _ = color.New(color.Bold).Println("Configuration Summary:")
	fmt.Fprintf(os.Stdout, "  Devices: %d\n", len(devices))
	fmt.Fprintln(os.Stdout)
	_, _ = color.New(color.Bold).Println("Devices:")
	for _, device := range devices {
		describeDeviceInfo(device)
	}
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "To use this scenario:")
	fmt.Fprintf(os.Stdout, "  niac scenario use %s config.yaml\n", scenarioName)
	fmt.Fprintln(os.Stdout, "  sudo niac daemon --once en0 config.yaml")
}

func describeDeviceInfo(device config.Device) {
	fmt.Fprintf(os.Stdout, "  • %s (%s)\n", device.Name, device.Type)
	if len(device.IPAddresses) > 0 {
		fmt.Fprintf(os.Stdout, "    IP: %s", device.IPAddresses[0])
		if len(device.IPAddresses) > 1 {
			fmt.Fprintf(os.Stdout, " (+%d more)", len(device.IPAddresses)-1)
		}
		fmt.Fprintln(os.Stdout)
	}

	protocols := listEnabledProtocols(device)
	if len(protocols) > 0 {
		fmt.Fprintf(os.Stdout, "    Protocols: %s\n", joinStrings(protocols, ", "))
	}
}

func listEnabledProtocols(device config.Device) []string {
	protocols := make([]string, 0, protocolCapacity)
	if device.ICMPConfig != nil && device.ICMPConfig.Enabled {
		protocols = append(protocols, "ICMP")
	}
	if device.LLDPConfig != nil && device.LLDPConfig.Enabled {
		protocols = append(protocols, "LLDP")
	}
	if device.CDPConfig != nil && device.CDPConfig.Enabled {
		protocols = append(protocols, "CDP")
	}
	if device.SNMPConfig.Community != "" || device.SNMPConfig.WalkFile != "" {
		protocols = append(protocols, "SNMP")
	}
	if device.DHCPConfig != nil {
		protocols = append(protocols, "DHCP")
	}
	if device.DNSConfig != nil {
		protocols = append(protocols, "DNS")
	}
	if device.HTTPConfig != nil && device.HTTPConfig.Enabled {
		protocols = append(protocols, "HTTP")
	}
	if device.STPConfig != nil && device.STPConfig.Enabled {
		protocols = append(protocols, "STP")
	}
	return protocols
}

func joinStrings(strs []string, sep string) string {
	return strings.Join(strs, sep)
}
