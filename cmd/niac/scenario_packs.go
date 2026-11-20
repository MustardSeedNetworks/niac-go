package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

const (
	packYAMLExt          = ".yaml"
	packListNameWidthPad = 2
)

var (
	errUnknownPack      = errors.New("unknown scenario pack")
	errPackOutputChoice = errors.New("--output and --save are mutually exclusive")
	errNetworkExists    = errors.New("the daemon library already holds a network with that name")
	errInvalidSavedName = errors.New("--save needs a network name")
)

type scenarioGenerateOptions struct {
	simulationCLIOptions

	output string
	save   string
	force  bool
}

// printScenarioPacks lists the generated packs under their own heading. A pack
// is not a template: it is built on demand, so `show` and `use` do not apply
// and the usage line says which command does.
func printScenarioPacks(w io.Writer) {
	packs := scenario.Packs()
	width := 0
	for _, pack := range packs {
		width = max(width, len(pack.ID))
	}
	fmt.Fprintln(w, "Scenario packs (generated on demand):")
	for _, pack := range packs {
		fmt.Fprintf(w, "  %-*s  %3d devices  %s\n",
			width+packListNameWidthPad, pack.ID, pack.Manifest.DeviceCount, pack.Description)
	}
}

func findScenarioPack(id string) (scenario.Pack, error) {
	ids := make([]string, 0, len(scenario.Packs()))
	for _, pack := range scenario.Packs() {
		if pack.ID == id {
			return pack, nil
		}
		ids = append(ids, pack.ID)
	}

	return scenario.Pack{}, fmt.Errorf("%w %q (available: %s)", errUnknownPack, id, strings.Join(ids, ", "))
}

func newScenarioGenerateCommand() *cobra.Command {
	options := new(scenarioGenerateOptions)
	command := &cobra.Command{
		Use:   "generate <pack-id>",
		Short: "Generate a scenario pack",
		Long: `Generate the YAML for one scenario pack (see 'niac scenario list').

With no flags the YAML goes to stdout. -o writes it to a file. --save stores it
in the running daemon's library instead and prints the command that starts it;
the daemon must approve the host interface with --attachment-policy before that
start will pass preflight.`,
		Example: `  # Write the hospital pack to a file
  niac scenario generate hospital -o hospital.yaml

  # Store it in the running daemon's library, then start it
  niac scenario generate hospital --save hospital
  niac simulation start -i en0 --config hospital.yaml --attachment cyberscope --attachment-mode direct`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScenarioGenerate(cmd, options, args[0])
		},
	}
	command.Flags().StringVarP(&options.output, "output", "o", "", "Write the YAML to this file instead of stdout")
	command.Flags().StringVar(&options.save, "save", "", "Save into the running daemon's library under this name")
	command.Flags().BoolVar(&options.force, "force", false, "With --save, replace a library network of the same name")
	addSimulationConnectionFlags(command, &options.simulationCLIOptions)

	return command
}

func runScenarioGenerate(cmd *cobra.Command, options *scenarioGenerateOptions, id string) error {
	if options.output != "" && cmd.Flags().Changed("save") {
		return errPackOutputChoice
	}
	if cmd.Flags().Changed("save") && strings.TrimSuffix(options.save, packYAMLExt) == "" {
		return errInvalidSavedName
	}
	pack, err := findScenarioPack(id)
	if err != nil {
		return err
	}
	result, err := scenario.Generate(pack.Request)
	if err != nil {
		return fmt.Errorf("generating pack %s: %w", id, err)
	}

	switch {
	case cmd.Flags().Changed("save"):
		return savePackToDaemon(cmd, options, pack, string(result.YAML))
	case options.output != "":
		return writePackFile(options.output, result.YAML)
	default:
		_, err = os.Stdout.Write(result.YAML)

		return err
	}
}

func writePackFile(path string, data []byte) error {
	outputFile, err := validateCLIPath(path)
	if err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}
	if _, statErr := statSafeFile(outputFile); statErr == nil {
		return fmt.Errorf("%w: %s", errOutputExists, outputFile)
	}
	if err = writeSafeFile(outputFile, data); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}
	fmt.Fprintf(os.Stdout, "Wrote %s\n", outputFile)

	return nil
}

func savePackToDaemon(cmd *cobra.Command, options *scenarioGenerateOptions, pack scenario.Pack, content string) error {
	name := strings.TrimSuffix(options.save, packYAMLExt)
	client, err := simulationClient(&options.simulationCLIOptions)
	if err != nil {
		return err
	}
	if !options.force {
		exists, existsErr := client.NetworkExists(cmd.Context(), name)
		if existsErr != nil {
			return existsErr
		}
		if exists {
			return fmt.Errorf("%w: %q (use --force to replace it)", errNetworkExists, name)
		}
	}
	if err = client.SaveNetwork(cmd.Context(), name, content); err != nil {
		return err
	}
	fmt.Fprint(os.Stdout, packStartHint(name, pack.Request.AttachmentName))

	return nil
}

// packStartHint is the follow-up after a save: the exact start command, and
// the policy line that command depends on, which is the step operators miss.
func packStartHint(name, attachment string) string {
	return fmt.Sprintf(`Saved %q to the daemon library.

Start it (replace <interface> with the host NIC):
  niac simulation start -i <interface> --config %s.yaml --attachment %s --attachment-mode direct

The daemon must approve that NIC first: start it with --attachment-policy <interface>=direct
(see docs/ATTACHMENTS.md).
`, name, name, attachment)
}
