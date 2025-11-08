package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/templates"
	"github.com/spf13/cobra"
)

var templateCmd = &cobra.Command{
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

var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available templates",
	Example: `  # List all templates with descriptions
  niac template list`,
	Run: runTemplateList,
}

var templateShowCmd = &cobra.Command{
	Use:   "show <template-name>",
	Short: "Show template contents",
	Example: `  # Show basic network template
  niac template show basic-network

  # Show small office template
  niac template show small-office

  # Pipe to file
  niac template show data-center > my-config.yaml`,
	Args: cobra.ExactArgs(1),
	Run:  runTemplateShow,
}

var templateUseCmd = &cobra.Command{
	Use:   "use <template-name> <output-file>",
	Short: "Copy template to a new file",
	Example: `  # Create small office config
  niac template use small-office office.yaml

  # Create IoT network config
  niac template use iot-network sensors.yaml

  # Create data center config
  niac template use data-center dc.yaml

  # Quick workflow
  niac template use basic-network config.yaml && niac validate config.yaml`,
	Args: cobra.ExactArgs(2),
	Run:  runTemplateUse,
}

var templateApplyCmd = &cobra.Command{
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
	Run:  runTemplateApply,
}

func init() {
	rootCmd.AddCommand(templateCmd)
	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateShowCmd)
	templateCmd.AddCommand(templateUseCmd)
	templateCmd.AddCommand(templateApplyCmd)
}

func runTemplateList(cmd *cobra.Command, args []string) {
	templateList := templates.List()

	color.New(color.Bold).Println("Available Templates:")
	fmt.Println()

	// Find longest name for alignment
	maxLen := 0
	for _, t := range templateList {
		if len(t.Name) > maxLen {
			maxLen = len(t.Name)
		}
	}

	for _, t := range templateList {
		color.New(color.FgCyan).Printf("  %-*s", maxLen+2, t.Name)
		fmt.Printf(" - %s\n", t.Description)
		if t.UseCase != "" {
			fmt.Printf("  %*s   Use case: %s\n", maxLen, "", t.UseCase)
		}
	}

	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  niac template show <template-name>         # View template content")
	fmt.Println("  niac template use <template-name> <file>   # Create config from template")
	fmt.Println("  niac template apply <template-name>        # Validate and show template info")
	fmt.Println()
	fmt.Println("Quick start:")
	fmt.Println("  niac init                                  # Interactive template wizard")
}

func runTemplateShow(cmd *cobra.Command, args []string) {
	templateName := args[0]

	tmpl, err := templates.Get(templateName)
	if err != nil {
		color.Red("Error: %v", err)
		fmt.Println()
		fmt.Println("Available templates:")
		for _, name := range templates.ListNames() {
			fmt.Printf("  - %s\n", name)
		}
		os.Exit(1)
	}

	fmt.Print(tmpl.Content)
}

func runTemplateUse(cmd *cobra.Command, args []string) {
	templateName := args[0]
	outputFile := args[1]

	// Check if output file exists
	if _, err := os.Stat(outputFile); err == nil {
		color.Red("Error: file already exists: %s", outputFile)
		os.Exit(1)
	}

	// Get template
	tmpl, err := templates.Get(templateName)
	if err != nil {
		color.Red("Error: %v", err)
		fmt.Println()
		fmt.Println("Available templates:")
		for _, name := range templates.ListNames() {
			fmt.Printf("  - %s\n", name)
		}
		os.Exit(1)
	}

	// Write to file
	if err := os.WriteFile(outputFile, []byte(tmpl.Content), 0644); err != nil {
		color.Red("Error writing file: %v", err)
		os.Exit(1)
	}

	color.Green("✓ Created %s from %s template", outputFile, templateName)
	fmt.Println()
	fmt.Printf("Description: %s\n", tmpl.Description)
	fmt.Printf("Use case: %s\n", tmpl.UseCase)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  niac validate %s\n", outputFile)
	fmt.Printf("  sudo niac interactive en0 %s\n", outputFile)
}

func runTemplateApply(cmd *cobra.Command, args []string) {
	templateName := args[0]

	// Get template
	tmpl, err := templates.Get(templateName)
	if err != nil {
		color.Red("Error: %v", err)
		os.Exit(1)
	}

	// Display template info
	color.New(color.Bold).Printf("Template: %s\n", tmpl.Name)
	fmt.Printf("Description: %s\n", tmpl.Description)
	fmt.Printf("Use case: %s\n", tmpl.UseCase)
	fmt.Println()

	// Validate template by loading it as config
	color.New(color.Bold).Println("Validating template...")

	// Create temporary file for validation
	tmpFile, err := os.CreateTemp("", "niac-template-*.yaml")
	if err != nil {
		color.Red("Error creating temporary file: %v", err)
		os.Exit(1)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(tmpl.Content); err != nil {
		color.Red("Error writing temporary file: %v", err)
		os.Exit(1)
	}
	tmpFile.Close()

	// Load and validate
	cfg, err := config.Load(tmpFile.Name())
	if err != nil {
		color.Red("✗ Template validation failed: %v", err)
		os.Exit(1)
	}

	color.Green("✓ Template is valid")
	fmt.Println()

	// Display configuration summary
	color.New(color.Bold).Println("Configuration Summary:")
	fmt.Printf("  Devices: %d\n", len(cfg.Devices))
	fmt.Println()

	// List devices with details
	color.New(color.Bold).Println("Devices:")
	for _, device := range cfg.Devices {
		fmt.Printf("  • %s (%s)\n", device.Name, device.Type)
		if len(device.IPAddresses) > 0 {
			fmt.Printf("    IP: %s", device.IPAddresses[0])
			if len(device.IPAddresses) > 1 {
				fmt.Printf(" (+%d more)", len(device.IPAddresses)-1)
			}
			fmt.Println()
		}

		// Show enabled protocols
		protocols := []string{}
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

		if len(protocols) > 0 {
			fmt.Printf("    Protocols: %s\n", joinStrings(protocols, ", "))
		}
	}

	fmt.Println()
	fmt.Println("To use this template:")
	fmt.Printf("  niac template use %s config.yaml\n", templateName)
	fmt.Println("  sudo niac interactive en0 config.yaml")
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
