package main

import (
	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/cliclient"
)

type simulationCLIOptions struct {
	api, caCert, iface, config, template, session, attachment, mode string
	accessVLAN                                                      uint16
	insecure                                                        bool
}

func addSimulationCommand(root *cobra.Command, _ *serviceOptions) {
	options := new(simulationCLIOptions)
	command := &cobra.Command{
		Use:   "simulation",
		Short: "Control scenarios through the running NIAC daemon",
		Long:  "Preflight, start, select, and stop scenarios through the running NIAC daemon.",
		Example: `  niac simulation start -i eth0 --config clinic.yaml --session clinic
  niac simulation stop clinic`,
		Args: cobra.NoArgs,
	}
	addSimulationConnectionFlags(command, options)
	command.AddCommand(newSimulationPreflightCommand(options))
	command.AddCommand(newSimulationStartCommand(options))
	command.AddCommand(newSimulationSelectCommand(options))
	command.AddCommand(newSimulationStopCommand(options))
	root.AddCommand(command)
}

func addSimulationConnectionFlags(command *cobra.Command, options *simulationCLIOptions) {
	command.PersistentFlags().StringVar(&options.api, "api", "",
		"Daemon API address (default: "+cliclient.DefaultBaseURL+", or NIAC_API_URL)")
	command.PersistentFlags().StringVar(&options.caCert, "cacert", "",
		"Daemon certificate to trust (default: the local daemon's own, when visible)")
	command.PersistentFlags().BoolVar(&options.insecure, "insecure", false,
		"Skip TLS verification, for a daemon whose certificate this host cannot see")
}

func newSimulationPreflightCommand(options *simulationCLIOptions) *cobra.Command {
	command := &cobra.Command{
		Use: "preflight", Short: "Validate a scenario through the daemon",
		Long:    "Compile and validate a managed scenario without changing daemon state.",
		Example: "  niac simulation preflight -i eth0 --config clinic.yaml --session clinic",
		Args:    cobra.NoArgs,
	}
	addSimulationRequestFlags(command, options)
	command.RunE = func(cmd *cobra.Command, _ []string) error {
		return runSimulationPreflight(cmd.Context(), options)
	}
	return command
}

func newSimulationStartCommand(options *simulationCLIOptions) *cobra.Command {
	command := &cobra.Command{
		Use: "start", Short: "Start a scenario through the daemon",
		Long:    "Start a managed scenario through the daemon's simulation registry.",
		Example: "  niac simulation start -i eth0 --config clinic.yaml --session clinic",
		Args:    cobra.NoArgs,
	}
	addSimulationRequestFlags(command, options)
	command.RunE = func(cmd *cobra.Command, _ []string) error {
		return runSimulationStart(cmd.Context(), options)
	}
	return command
}

func addSimulationRequestFlags(command *cobra.Command, options *simulationCLIOptions) {
	command.Flags().StringVarP(&options.iface, "interface", "i", "", "Physical network interface")
	command.Flags().StringVar(&options.config, "config", "", "Managed scenario configuration path")
	command.Flags().StringVar(&options.template, "template", "", "Built-in scenario template name")
	command.Flags().StringVar(&options.session, "session", "", "Scenario session ID")
	command.Flags().StringVar(&options.attachment, "attachment", "", "Attachment name from the scenario")
	command.Flags().StringVar(&options.mode, "mode", "", "Attachment mode: direct, access, or trunk")
	command.Flags().Uint16Var(&options.accessVLAN, "access-vlan", 0, "Physical VLAN for access mode")
	_ = command.MarkFlagRequired("interface")
}

func newSimulationSelectCommand(options *simulationCLIOptions) *cobra.Command {
	return &cobra.Command{
		Use: "select <session>", Short: "Select the scenario used by global status views",
		Long:    "Select which running scenario the daemon exposes through global status views.",
		Example: "  niac simulation select clinic", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSimulationSelect(cmd.Context(), options, args[0])
		},
	}
}

func newSimulationStopCommand(options *simulationCLIOptions) *cobra.Command {
	return &cobra.Command{
		Use: "stop <session>", Short: "Stop one running scenario",
		Long:    "Stop one managed scenario by its session ID without restarting the daemon.",
		Example: "  niac simulation stop clinic", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSimulationStop(cmd.Context(), options, args[0])
		},
	}
}
