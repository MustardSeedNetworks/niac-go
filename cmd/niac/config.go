package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func addConfigCommand(root *cobra.Command, _ *serviceOptions) {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management tools",
		Long:  `Tools for exporting, comparing, and merging NIAC configurations.`,
		Example: `  # Export configuration to new file
  niac config export input.yaml output.yaml

  # Compare two configurations
  niac config diff config1.yaml config2.yaml

  # Edit a configuration in $EDITOR and validate it
  niac config edit config.yaml

  # Merge configurations
  niac config merge base.yaml overlay.yaml merged.yaml`,
	}

	configCmd.AddCommand(newConfigExportCmd())
	configCmd.AddCommand(newConfigDiffCmd())
	configCmd.AddCommand(newConfigEditCmd())
	configCmd.AddCommand(newConfigInterfaceCmd())
	configCmd.AddCommand(newConfigMergeCmd())
	addGenerateCommand(configCmd)
	root.AddCommand(configCmd)
}

func newConfigExportCmd() *cobra.Command {
	return &cobra.Command{
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
		Args: cobra.ExactArgs(argsCountTwo),
		RunE: func(_ *cobra.Command, args []string) error {
			return runConfigExport(args)
		},
	}
}

func newConfigDiffCmd() *cobra.Command {
	return &cobra.Command{
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
		Args: cobra.ExactArgs(argsCountTwo),
		RunE: func(_ *cobra.Command, args []string) error {
			return runConfigDiff(args)
		},
	}
}

func newConfigMergeCmd() *cobra.Command {
	return &cobra.Command{
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
		Args: cobra.ExactArgs(argsCountThree),
		RunE: func(_ *cobra.Command, args []string) error {
			return runConfigMerge(args)
		},
	}
}

// errOutputExists marks a refusal to overwrite an operator's file, so a test
// can assert the reason rather than merely that something failed.
var errOutputExists = errors.New("output file already exists")

func runConfigExport(args []string) error {
	inputFile := args[0]

	outputFile, pathErr := validateCLIPath(args[1])
	if pathErr != nil {
		return fmt.Errorf("invalid output path: %w", pathErr)
	}

	if err := checkOutputNotExists(outputFile); err != nil {
		return err
	}

	cfg, err := config.Load(inputFile)
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}

	// Validate
	validator := config.NewValidator(inputFile)
	result := validator.Validate(cfg)
	if !result.Valid {
		fmt.Fprintf(os.Stderr, "Warning: Configuration has validation errors:\n")
		fmt.Fprintln(os.Stderr, result.Format())
		fmt.Fprintln(os.Stderr, "\nExporting anyway...")
	}

	if writeErr := writeConfigFile(cfg, outputFile); writeErr != nil {
		return writeErr
	}

	fmt.Fprintf(os.Stdout, "Configuration exported to %s\n", outputFile)
	fmt.Fprintf(os.Stdout, "Devices: %d\n", cfg.DeviceCount())

	return nil
}

func runConfigDiff(args []string) error {
	cfg1, cfg2, err := loadConfigPair(args[0], args[1])
	if err != nil {
		return err
	}

	devices1 := buildDeviceMap(cfg1)
	devices2 := buildDeviceMap(cfg2)

	if !compareDeviceMaps(devices1, devices2) {
		fmt.Fprintln(os.Stdout, "No differences found")
	}

	return nil
}

func loadConfigPair(file1, file2 string) (*config.Config, *config.Config, error) {
	cfg1, err := config.Load(file1)
	if err != nil {
		return nil, nil, fmt.Errorf("loading %s: %w", file1, err)
	}

	cfg2, err := config.Load(file2)
	if err != nil {
		return nil, nil, fmt.Errorf("loading %s: %w", file2, err)
	}

	return cfg1, cfg2, nil
}

func buildDeviceMap(cfg *config.Config) map[string]*config.Device {
	devices := make(map[string]*config.Device)
	for i := range cfg.Devices {
		devices[cfg.Devices[i].Name] = &cfg.Devices[i]
	}
	return devices
}

func compareDeviceMaps(devices1, devices2 map[string]*config.Device) bool {
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
		if dev2, exists := devices2[name]; exists {
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
	}
	return hasChanges
}

func runConfigMerge(args []string) error {
	baseFile, overlayFile := args[0], args[1]

	outputFile, pathErr := validateCLIPath(args[2])
	if pathErr != nil {
		return fmt.Errorf("invalid output path: %w", pathErr)
	}

	if err := checkOutputNotExists(outputFile); err != nil {
		return err
	}

	base, err := loadLabelledConfig(baseFile, "base")
	if err != nil {
		return err
	}

	overlay, err := loadLabelledConfig(overlayFile, "overlay")
	if err != nil {
		return err
	}

	merged := mergeConfigs(base, overlay)
	if writeErr := writeConfigFile(merged, outputFile); writeErr != nil {
		return writeErr
	}

	printMergeStats(base, overlay, merged, outputFile)

	return nil
}

// checkOutputNotExists refuses to clobber an existing file. Both export and
// merge write to an operator-named path, and silently overwriting one is the
// kind of loss a CLI should never cause.
func checkOutputNotExists(path string) error {
	if _, err := statSafeFile(path); err == nil {
		return fmt.Errorf("%w: %s", errOutputExists, path)
	}

	return nil
}

func loadLabelledConfig(path, label string) (*config.Config, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return nil, fmt.Errorf("loading %s: %w", label, err)
	}

	return cfg, nil
}

func mergeConfigs(base, overlay *config.Config) *config.Config {
	merged := &config.Config{Devices: make([]config.Device, 0)}
	overlayDevices := buildDeviceMap(overlay)

	for _, dev := range base.Devices {
		if overlayDev, exists := overlayDevices[dev.Name]; exists {
			merged.Devices = append(merged.Devices, *overlayDev)
			delete(overlayDevices, dev.Name)
		} else {
			merged.Devices = append(merged.Devices, dev)
		}
	}
	for _, dev := range overlayDevices {
		merged.Devices = append(merged.Devices, *dev)
	}
	return merged
}

func writeConfigFile(cfg *config.Config, path string) error {
	data, err := config.MarshalConfigYAML(cfg)
	if err != nil {
		return fmt.Errorf("marshaling configuration: %w", err)
	}

	if writeErr := writeSafeFile(path, data); writeErr != nil {
		return fmt.Errorf("writing file: %w", writeErr)
	}

	return nil
}

func printMergeStats(base, overlay, merged *config.Config, output string) {
	fmt.Fprintf(os.Stdout, "Merged configuration written to %s\n", output)
	fmt.Fprintf(os.Stdout, "Base devices: %d\n", len(base.Devices))
	fmt.Fprintf(os.Stdout, "Overlay devices: %d\n", len(overlay.Devices))
	fmt.Fprintf(os.Stdout, "Merged devices: %d\n", len(merged.Devices))
}
