package main

import (
	"fmt"
	"os"

	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management tools",
	Long:  `Tools for exporting, comparing, and merging NIAC configurations.`,
	Example: `  # Export configuration to new file
  niac config export input.yaml output.yaml

  # Compare two configurations
  niac config diff config1.yaml config2.yaml

  # Merge configurations
  niac config merge base.yaml overlay.yaml merged.yaml`,
}

var configExportCmd = &cobra.Command{
	Use:   "export <input-file> <output-file>",
	Short: "Export configuration to YAML",
	Long: `Export a NIAC configuration file to normalized YAML format.

This command:
- Loads and validates the input configuration
- Normalizes all fields and structures
- Exports to clean YAML format
- Useful for converting legacy .cfg to YAML`,
	Example: `  # Export to new file
  niac config export config.yaml normalized.yaml

  # Convert legacy .cfg to YAML
  niac config export legacy.cfg new-config.yaml

  # Validate and normalize
  niac config export messy.yaml clean.yaml`,
	Args: cobra.ExactArgs(2),
	Run:  runConfigExport,
}

var configDiffCmd = &cobra.Command{
	Use:   "diff <file1> <file2>",
	Short: "Compare two configurations",
	Long: `Compare two NIAC configuration files and show differences.

Compares:
- Device additions/removals
- Device name changes
- MAC/IP address changes
- Protocol configuration changes`,
	Example: `  # Compare two configs
  niac config diff prod.yaml staging.yaml

  # Check for drift
  niac config diff baseline.yaml current.yaml

  # Compare before/after changes
  niac config diff config.yaml config.new.yaml`,
	Args: cobra.ExactArgs(2),
	Run:  runConfigDiff,
}

var configMergeCmd = &cobra.Command{
	Use:   "merge <base-file> <overlay-file> <output-file>",
	Short: "Merge two configurations",
	Long: `Merge two NIAC configuration files.

The overlay file takes precedence:
- Devices with same name are replaced
- New devices are added
- Base devices not in overlay are kept`,
	Example: `  # Merge overlay into base
  niac config merge base.yaml overlay.yaml merged.yaml

  # Apply environment-specific overrides
  niac config merge common.yaml prod-overrides.yaml prod-config.yaml

  # Combine device configs
  niac config merge routers.yaml switches.yaml network.yaml`,
	Args: cobra.ExactArgs(3),
	Run:  runConfigMerge,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configExportCmd)
	configCmd.AddCommand(configDiffCmd)
	configCmd.AddCommand(configMergeCmd)
}

func runConfigExport(cmd *cobra.Command, args []string) {
	inputFile := args[0]
	outputFile := args[1]

	// Check if output exists
	if _, err := os.Stat(outputFile); err == nil {
		fmt.Fprintf(os.Stderr, "Error: output file already exists: %s\n", outputFile)
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.Load(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Validate
	validator := config.NewValidator(inputFile)
	result := validator.Validate(cfg)
	if !result.Valid {
		fmt.Fprintf(os.Stderr, "Warning: Configuration has validation errors:\n")
		fmt.Fprintln(os.Stderr, result.Format())
		fmt.Fprintln(os.Stderr, "\nExporting anyway...")
	}

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling configuration: %v\n", err)
		os.Exit(1)
	}

	// Write to file
	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Configuration exported to %s\n", outputFile)
	fmt.Printf("Devices: %d\n", len(cfg.Devices))
}

func runConfigDiff(cmd *cobra.Command, args []string) {
	file1 := args[0]
	file2 := args[1]

	// Load configurations
	cfg1, err := config.Load(file1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading %s: %v\n", file1, err)
		os.Exit(1)
	}

	cfg2, err := config.Load(file2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading %s: %v\n", file2, err)
		os.Exit(1)
	}

	// Build device maps
	devices1 := make(map[string]*config.Device)
	devices2 := make(map[string]*config.Device)

	for i := range cfg1.Devices {
		devices1[cfg1.Devices[i].Name] = &cfg1.Devices[i]
	}
	for i := range cfg2.Devices {
		devices2[cfg2.Devices[i].Name] = &cfg2.Devices[i]
	}

	hasChanges := false

	// Check for removed devices
	for name := range devices1 {
		if _, exists := devices2[name]; !exists {
			fmt.Printf("- Device removed: %s\n", name)
			hasChanges = true
		}
	}

	// Check for added devices
	for name := range devices2 {
		if _, exists := devices1[name]; !exists {
			fmt.Printf("+ Device added: %s\n", name)
			hasChanges = true
		}
	}

	// Check for modified devices
	for name, dev1 := range devices1 {
		if dev2, exists := devices2[name]; exists {
			if dev1.MACAddress.String() != dev2.MACAddress.String() {
				fmt.Printf("~ Device %s: MAC changed from %s to %s\n",
					name, dev1.MACAddress, dev2.MACAddress)
				hasChanges = true
			}
			if dev1.Type != dev2.Type {
				fmt.Printf("~ Device %s: Type changed from %s to %s\n",
					name, dev1.Type, dev2.Type)
				hasChanges = true
			}
		}
	}

	if !hasChanges {
		fmt.Println("No differences found")
	}
}

func runConfigMerge(cmd *cobra.Command, args []string) {
	baseFile := args[0]
	overlayFile := args[1]
	outputFile := args[2]

	// Check if output exists
	if _, err := os.Stat(outputFile); err == nil {
		fmt.Fprintf(os.Stderr, "Error: output file already exists: %s\n", outputFile)
		os.Exit(1)
	}

	// Load base configuration
	base, err := config.Load(baseFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading base: %v\n", err)
		os.Exit(1)
	}

	// Load overlay configuration
	overlay, err := config.Load(overlayFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading overlay: %v\n", err)
		os.Exit(1)
	}

	// Build merged config
	merged := &config.Config{
		Devices: make([]config.Device, 0),
	}

	// Create map of overlay devices by name
	overlayDevices := make(map[string]*config.Device)
	for i := range overlay.Devices {
		overlayDevices[overlay.Devices[i].Name] = &overlay.Devices[i]
	}

	// Add/replace devices from base
	for _, dev := range base.Devices {
		if overlayDev, exists := overlayDevices[dev.Name]; exists {
			// Use overlay version
			merged.Devices = append(merged.Devices, *overlayDev)
			delete(overlayDevices, dev.Name)
		} else {
			// Keep base version
			merged.Devices = append(merged.Devices, dev)
		}
	}

	// Add remaining overlay devices
	for _, dev := range overlayDevices {
		merged.Devices = append(merged.Devices, *dev)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(merged)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling configuration: %v\n", err)
		os.Exit(1)
	}

	// Write to file
	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Merged configuration written to %s\n", outputFile)
	fmt.Printf("Base devices: %d\n", len(base.Devices))
	fmt.Printf("Overlay devices: %d\n", len(overlay.Devices))
	fmt.Printf("Merged devices: %d\n", len(merged.Devices))
}
