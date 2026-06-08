package main

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

type configInterfaceOptions struct {
	speed       int
	duplex      string
	adminStatus string
	operStatus  string
	output      string
}

func newConfigInterfaceCmd() *cobra.Command {
	interfaceCmd := &cobra.Command{
		Use:   "interface",
		Short: "Manage device interface metadata",
		Long: `Manage device interface metadata in a NIAC configuration.

Interface metadata controls simulated speed, duplex, administrative status,
operational status, descriptions, and VLAN hints used by runtime and topology
views.`,
		Example: `  # Set speed and duplex in place
  niac config interface set network.yaml switch-a Ethernet1/1 --speed 1000 --duplex full

  # Write the updated config to a new file
  niac config interface set network.yaml switch-a Ethernet1/1 --admin-status down --output updated.yaml`,
	}

	interfaceCmd.AddCommand(newConfigInterfaceSetCmd())
	return interfaceCmd
}

func newConfigInterfaceSetCmd() *cobra.Command {
	options := new(configInterfaceOptions)

	setCmd := &cobra.Command{
		Use:   "set <config-file> <device> <interface>",
		Short: "Set speed, duplex, and status for a device interface",
		Long: `Set speed, duplex, and status for a device interface in a NIAC config.

If the named interface does not exist on the device, it is created. By default
the input file is updated in place; pass --output to write to a separate file.`,
		Example: `  niac config interface set network.yaml switch-a Ethernet1/1 --speed 1000 --duplex full
  niac config interface set network.yaml switch-a Ethernet1/1 --admin-status down --oper-status down`,
		Args: cobra.ExactArgs(argsCountThree),
		RunE: func(_ *cobra.Command, args []string) error {
			return runConfigInterfaceSet(args[0], args[1], args[2], options)
		},
	}

	setCmd.Flags().IntVar(&options.speed, "speed", 0, "Interface speed in Mbps")
	setCmd.Flags().StringVar(&options.duplex, "duplex", "", "Interface duplex: full, half, or auto")
	setCmd.Flags().StringVar(&options.adminStatus, "admin-status", "", "Admin status: up or down")
	setCmd.Flags().StringVar(&options.operStatus, "oper-status", "", "Operational status: up, down, or testing")
	setCmd.Flags().StringVar(&options.output, "output", "", "Output config path; defaults to updating input in place")

	return setCmd
}

func runConfigInterfaceSet(
	configFile, deviceName, interfaceName string,
	options *configInterfaceOptions,
) error {
	if err := validateInterfaceOptions(options); err != nil {
		return err
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	device := findConfigDevice(cfg, deviceName)
	if device == nil {
		return fmt.Errorf("device not found: %s", deviceName)
	}

	iface := findOrCreateConfigInterface(device, interfaceName)
	applyConfigInterfaceOptions(iface, options)

	data, err := config.MarshalConfigYAML(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	outputPath := options.output
	if outputPath == "" {
		outputPath = configFile
	}

	cleanedOutput, pathErr := validateCLIPath(outputPath)
	if pathErr != nil {
		return fmt.Errorf("invalid output path: %w", pathErr)
	}

	if writeErr := writeSafeFile(cleanedOutput, data); writeErr != nil {
		return fmt.Errorf("failed to write config: %w", writeErr)
	}

	fmt.Fprintf(os.Stdout, "Updated %s %s in %s\n", deviceName, interfaceName, cleanedOutput)
	return nil
}

func validateInterfaceOptions(options *configInterfaceOptions) error {
	if options.speed < 0 {
		return errors.New("speed must be zero or greater")
	}
	if options.duplex != "" && !isAllowedConfigInterfaceValue(options.duplex, "full", "half", "auto") {
		return errors.New("duplex must be full, half, or auto")
	}
	if options.adminStatus != "" && !isAllowedConfigInterfaceValue(options.adminStatus, "up", "down") {
		return errors.New("admin status must be up or down")
	}
	if options.operStatus != "" && !isAllowedConfigInterfaceValue(options.operStatus, "up", "down", "testing") {
		return errors.New("oper status must be up, down, or testing")
	}
	return nil
}

func isAllowedConfigInterfaceValue(value string, allowed ...string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return slices.Contains(allowed, value)
}

func findConfigDevice(cfg *config.Config, name string) *config.Device {
	for i := range cfg.Devices {
		if cfg.Devices[i].Name == name {
			return &cfg.Devices[i]
		}
	}
	return nil
}

func findOrCreateConfigInterface(device *config.Device, name string) *config.Interface {
	for i := range device.Interfaces {
		if device.Interfaces[i].Name == name {
			return &device.Interfaces[i]
		}
	}
	device.Interfaces = append(device.Interfaces, config.Interface{Name: name})
	return &device.Interfaces[len(device.Interfaces)-1]
}

func applyConfigInterfaceOptions(iface *config.Interface, options *configInterfaceOptions) {
	if options.speed > 0 {
		iface.Speed = options.speed
	}
	if options.duplex != "" {
		iface.Duplex = strings.ToLower(strings.TrimSpace(options.duplex))
	}
	if options.adminStatus != "" {
		iface.AdminStatus = strings.ToLower(strings.TrimSpace(options.adminStatus))
	}
	if options.operStatus != "" {
		iface.OperStatus = strings.ToLower(strings.TrimSpace(options.operStatus))
	}
}
