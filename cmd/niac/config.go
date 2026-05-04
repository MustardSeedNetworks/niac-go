package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// Config command help text as file-level constants to keep addConfigCommand concise.
const (
	configCmdExample = `  # Export configuration to new file
  niac config export input.yaml output.yaml

  # Compare two configurations
  niac config diff config1.yaml config2.yaml

  # Merge configurations
  niac config merge base.yaml overlay.yaml merged.yaml`

	configExportLong = `Export a NIAC configuration file to normalized YAML format.

This command:
- Loads and validates the input configuration
- Normalizes all fields and structures
- Exports to clean YAML format
- Useful for converting legacy .cfg to YAML`

	configExportExample = `  # Export to new file
  niac config export config.yaml normalized.yaml

  # Convert legacy .cfg to YAML
  niac config export legacy.cfg new-config.yaml

  # Validate and normalize
  niac config export messy.yaml clean.yaml`

	configDiffLong = `Compare two NIAC configuration files and show differences.

Compares:
- Device additions/removals
- Device name changes
- MAC/IP address changes
- Protocol configuration changes`

	configDiffExample = `  # Compare two configs
  niac config diff prod.yaml staging.yaml

  # Check for drift
  niac config diff baseline.yaml current.yaml

  # Compare before/after changes
  niac config diff config.yaml config.new.yaml`

	configMergeLong = `Merge two NIAC configuration files.

The overlay file takes precedence:
- Devices with same name are replaced
- New devices are added
- Base devices not in overlay are kept`

	configMergeExample = `  # Merge overlay into base
  niac config merge base.yaml overlay.yaml merged.yaml

  # Apply environment-specific overrides
  niac config merge common.yaml prod-overrides.yaml prod-config.yaml

  # Combine device configs
  niac config merge routers.yaml switches.yaml network.yaml`
)

func addConfigCommand(root *cobra.Command, _ *serviceOptions) {
	configCmd := &cobra.Command{
		Use:     "config",
		Short:   "Configuration management tools",
		Long:    `Tools for exporting, comparing, and merging NIAC configurations.`,
		Example: configCmdExample,
	}

	configCmd.AddCommand(newConfigExportCmd())
	configCmd.AddCommand(newConfigDiffCmd())
	configCmd.AddCommand(newConfigMergeCmd())
	addGenerateCommand(configCmd)
	root.AddCommand(configCmd)
}

func newConfigExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "export <input-file> <output-file>",
		Short:   "Export configuration to YAML",
		Long:    configExportLong,
		Example: configExportExample,
		Args:    cobra.ExactArgs(argsCountTwo),
		Run: func(_ *cobra.Command, args []string) {
			runConfigExport(args)
		},
	}
}

func newConfigDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "diff <file1> <file2>",
		Short:   "Compare two configurations",
		Long:    configDiffLong,
		Example: configDiffExample,
		Args:    cobra.ExactArgs(argsCountTwo),
		Run: func(_ *cobra.Command, args []string) {
			runConfigDiff(args)
		},
	}
}

func newConfigMergeCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "merge <base-file> <overlay-file> <output-file>",
		Short:   "Merge two configurations",
		Long:    configMergeLong,
		Example: configMergeExample,
		Args:    cobra.ExactArgs(argsCountThree),
		Run: func(_ *cobra.Command, args []string) {
			runConfigMerge(args)
		},
	}
}

func runConfigExport(args []string) {
	inputFile := args[0]
	outputFile, pathErr := validateCLIPath(args[1])
	if pathErr != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid output path: %v\n", pathErr)
		os.Exit(1)
	}

	// Check if output exists
	if _, err := statSafeFile(outputFile); err == nil {
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
	data, err := config.MarshalConfigYAML(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling configuration: %v\n", err)
		os.Exit(1)
	}

	// Write to file
	if writeErr := writeSafeFile(outputFile, data); writeErr != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", writeErr)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "Configuration exported to %s\n", outputFile)
	fmt.Fprintf(os.Stdout, "Devices: %d\n", len(cfg.Devices))
}

func runConfigDiff(args []string) {
	file1 := args[0]
	file2 := args[1]

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

	devices1 := buildDeviceMap(cfg1.Devices)
	devices2 := buildDeviceMap(cfg2.Devices)

	hasChanges := reportDeviceDiffs(devices1, devices2)
	if !hasChanges {
		fmt.Fprintln(os.Stdout, "No differences found")
	}
}

// buildDeviceMap indexes devices by name into a lookup map.
func buildDeviceMap(devices []config.Device) map[string]*config.Device {
	out := make(map[string]*config.Device, len(devices))
	for i := range devices {
		out[devices[i].Name] = &devices[i]
	}
	return out
}

// reportDeviceDiffs prints additions/removals/changes and returns whether any were found.
func reportDeviceDiffs(devices1, devices2 map[string]*config.Device) bool {
	hasChanges := false

	for name := range devices1 {
		if _, exists := devices2[name]; !exists {
			fmt.Fprintf(os.Stdout, "- Device removed: %s\n", name)
			hasChanges = true
		}
	}

	for name := range devices2 {
		if _, exists := devices1[name]; !exists {
			fmt.Fprintf(os.Stdout, "+ Device added: %s\n", name)
			hasChanges = true
		}
	}

	for name, dev1 := range devices1 {
		dev2, exists := devices2[name]
		if !exists {
			continue
		}
		if dev1.MACAddress.String() != dev2.MACAddress.String() {
			fmt.Fprintf(os.Stdout, "~ Device %s: MAC changed from %s to %s\n",
				name, dev1.MACAddress, dev2.MACAddress)
			hasChanges = true
		}
		if dev1.Type != dev2.Type {
			fmt.Fprintf(os.Stdout, "~ Device %s: Type changed from %s to %s\n",
				name, dev1.Type, dev2.Type)
			hasChanges = true
		}
	}

	return hasChanges
}

func runConfigMerge(args []string) {
	baseFile := args[0]
	overlayFile := args[1]
	outputFile, pathErr := validateCLIPath(args[2])
	if pathErr != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid output path: %v\n", pathErr)
		os.Exit(1)
	}

	// Check if output exists
	if _, err := statSafeFile(outputFile); err == nil {
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

	merged := mergeConfigs(base, overlay)

	// Marshal to YAML
	data, err := config.MarshalConfigYAML(merged)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling configuration: %v\n", err)
		os.Exit(1)
	}

	// Write to file
	if writeErr := writeSafeFile(outputFile, data); writeErr != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", writeErr)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "Merged configuration written to %s\n", outputFile)
	fmt.Fprintf(os.Stdout, "Base devices: %d\n", len(base.Devices))
	fmt.Fprintf(os.Stdout, "Overlay devices: %d\n", len(overlay.Devices))
	fmt.Fprintf(os.Stdout, "Merged devices: %d\n", len(merged.Devices))
}

// mergeConfigs combines base and overlay device lists, with overlay taking precedence.
func mergeConfigs(base, overlay *config.Config) *config.Config {
	merged := new(config.Config)
	merged.Devices = make([]config.Device, 0, len(base.Devices)+len(overlay.Devices))

	overlayDevices := make(map[string]*config.Device, len(overlay.Devices))
	for i := range overlay.Devices {
		overlayDevices[overlay.Devices[i].Name] = &overlay.Devices[i]
	}

	// Add/replace devices from base
	for _, dev := range base.Devices {
		if overlayDev, exists := overlayDevices[dev.Name]; exists {
			merged.Devices = append(merged.Devices, *overlayDev)
			delete(overlayDevices, dev.Name)
		} else {
			merged.Devices = append(merged.Devices, dev)
		}
	}

	// Add remaining overlay devices
	for _, dev := range overlayDevices {
		merged.Devices = append(merged.Devices, *dev)
	}

	return merged
}
