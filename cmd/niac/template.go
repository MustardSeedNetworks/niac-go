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

func addTemplateCommand(root *cobra.Command, _ *serviceOptions) {
	templateCmd := &cobra.Command{
		Use:   "template",
		Short: "Manage configuration templates",
		Long:  `List, show, and use pre-built configuration templates for common scenarios.`,
		Example: `  # List all available templates
  niac template list

  # Show template contents
  niac template show basic-network

  # Create config from template
  niac template use small-office office.yaml

  # Apply template directly (validate and display info)
  niac template apply data-center`,
	}

	templateListCmd := &cobra.Command{
		Use:   "list",
		Short: "List available templates",
		Long: `Print every bundled template name with a one-line description.
Templates cover common scenarios (basic-network, small-office, data-center,
iot-network, etc.) and are the fastest path to a runnable YAML config.`,
		Example: `  # List all templates with descriptions
  niac template list`,
		Run: func(_ *cobra.Command, _ []string) {
			runTemplateList()
		},
	}

	templateShowCmd := &cobra.Command{
		Use:   "show <template-name>",
		Short: "Show template contents",
		Long: `Print the YAML body of a named template to stdout. Useful for
inspecting what a template will produce or piping it into another tool
without writing to disk.`,
		Example: `  # Show basic network template
  niac template show basic-network

  # Show small office template
  niac template show small-office

  # Pipe to file
  niac template show data-center > my-config.yaml`,
		Args: cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			runTemplateShow(args)
		},
	}

	templateUseCmd := &cobra.Command{
		Use:   "use <template-name> <output-file>",
		Short: "Copy template to a new file",
		Long: `Copy a named template's body into a new YAML file at the given
output path. The output file becomes the starting point you edit and run
with 'niac run'; the template itself is unchanged.`,
		Example: `  # Create small office config
  niac template use small-office office.yaml

  # Create IoT network config
  niac template use iot-network sensors.yaml

  # Create data center config
  niac template use data-center dc.yaml

  # Quick workflow
  niac template use basic-network config.yaml && niac validate config.yaml`,
		Args: cobra.ExactArgs(argsCountTwo),
		Run: func(_ *cobra.Command, args []string) {
			runTemplateUse(args)
		},
	}

	templateApplyCmd := &cobra.Command{
		Use:   "apply <template-name>",
		Short: "Validate and display template information",
		Long: `Validate a template and display its configuration details.
This command loads the template, validates it, and shows what devices
and protocols it contains without creating a file.`,
		Example: `  # Validate basic network template
  niac template apply basic-network

  # Check data center template
  niac template apply data-center

  # Verify IoT network configuration
  niac template apply iot-network`,
		Args: cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			runTemplateApply(args)
		},
	}

	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateShowCmd)
	templateCmd.AddCommand(templateUseCmd)
	templateCmd.AddCommand(templateApplyCmd)
	root.AddCommand(templateCmd)
}

func runTemplateList() {
	templateList := templates.List()

	_, _ = color.New(color.Bold).Println("Available Templates:")
	fmt.Fprintln(os.Stdout)

	// Find longest name for alignment
	maxLen := 0
	for _, t := range templateList {
		if len(t.Name) > maxLen {
			maxLen = len(t.Name)
		}
	}

	for _, t := range templateList {
		_, _ = color.New(color.FgCyan).Printf("  %-*s", maxLen+templatePadOffset, t.Name)
		fmt.Fprintf(os.Stdout, " - %s\n", t.Description)
		if t.UseCase != "" {
			fmt.Fprintf(os.Stdout, "  %*s   Use case: %s\n", maxLen, "", t.UseCase)
		}
	}

	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Usage:")
	fmt.Fprintln(os.Stdout, "  niac template show <template-name>         # View template content")
	fmt.Fprintln(os.Stdout, "  niac template use <template-name> <file>   # Create config from template")
	fmt.Fprintln(os.Stdout, "  niac template apply <template-name>        # Validate and show template info")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Quick start:")
	fmt.Fprintln(os.Stdout, "  niac init                                  # Interactive template wizard")
}

func runTemplateShow(args []string) {
	templateName := args[0]

	tmpl, err := templates.Get(templateName)
	if err != nil {
		color.Red("Error: %v", err)
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, "Available templates:")
		for _, name := range templates.ListNames() {
			fmt.Fprintf(os.Stdout, "  - %s\n", name)
		}
		exitProcess(1)
	}

	fmt.Fprint(os.Stdout, tmpl.Content)
}

func runTemplateUse(args []string) {
	templateName := args[0]
	outputFile, pathErr := validateCLIPath(args[1])
	if pathErr != nil {
		color.Red("Error: invalid output path: %v", pathErr)
		exitProcess(1)
	}

	// Check if output file exists
	if _, statErr := statSafeFile(outputFile); statErr == nil {
		color.Red("Error: file already exists: %s", outputFile)
		exitProcess(1)
	}

	// Get template
	tmpl, err := templates.Get(templateName)
	if err != nil {
		color.Red("Error: %v", err)
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, "Available templates:")
		for _, name := range templates.ListNames() {
			fmt.Fprintf(os.Stdout, "  - %s\n", name)
		}
		exitProcess(1)
	}

	// Write to file
	if writeErr := writeSafeFile(outputFile, []byte(tmpl.Content)); writeErr != nil {
		color.Red("Error writing file: %v", writeErr)
		exitProcess(1)
	}

	color.Green("✓ Created %s from %s template", outputFile, templateName)
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "Description: %s\n", tmpl.Description)
	fmt.Fprintf(os.Stdout, "Use case: %s\n", tmpl.UseCase)
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Next steps:")
	fmt.Fprintf(os.Stdout, "  niac validate %s\n", outputFile)
	fmt.Fprintf(os.Stdout, "  sudo niac interactive en0 %s\n", outputFile)
}

func runTemplateApply(args []string) {
	templateName := args[0]

	tmpl, err := templates.Get(templateName)
	if err != nil {
		color.Red("Error: %v", err)
		exitProcess(1)
	}

	describeTemplate(tmpl)

	cfg, cleanup, loadErr := loadAndValidateTemplate(tmpl)
	if loadErr != nil {
		color.Red("✗ Template validation failed: %v", loadErr)
		cleanup()
		exitProcess(1)
	}
	cleanup()

	color.Green("✓ Template is valid")
	fmt.Fprintln(os.Stdout)

	describeDevices(templateName, cfg.Devices)
}

func describeTemplate(tmpl *templates.Template) {
	_, _ = color.New(color.Bold).Printf("Template: %s\n", tmpl.Name)
	fmt.Fprintf(os.Stdout, "Description: %s\n", tmpl.Description)
	fmt.Fprintf(os.Stdout, "Use case: %s\n", tmpl.UseCase)
	fmt.Fprintln(os.Stdout)
	_, _ = color.New(color.Bold).Println("Validating template...")
}

func loadAndValidateTemplate(tmpl *templates.Template) (*config.Config, func(), error) {
	tmpFile, err := os.CreateTemp("", "niac-template-*.yaml")
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

func describeDevices(templateName string, devices []config.Device) {
	_, _ = color.New(color.Bold).Println("Configuration Summary:")
	fmt.Fprintf(os.Stdout, "  Devices: %d\n", len(devices))
	fmt.Fprintln(os.Stdout)
	_, _ = color.New(color.Bold).Println("Devices:")
	for _, device := range devices {
		describeDeviceInfo(device)
	}
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "To use this template:")
	fmt.Fprintf(os.Stdout, "  niac template use %s config.yaml\n", templateName)
	fmt.Fprintln(os.Stdout, "  sudo niac interactive en0 config.yaml")
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
