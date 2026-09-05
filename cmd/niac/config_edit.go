package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// errNoEditor marks the case where neither $EDITOR nor $VISUAL resolves to an
// executable, so the caller can tell "no editor" from "the editor failed".
var errNoEditor = errors.New("no usable editor")

func newConfigEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <config-file>",
		Short: "Open a configuration in $EDITOR and validate the result",
		Long: `Open a NIAC configuration file in $EDITOR (falling back to $VISUAL, then vi)
and validate it once the editor exits.

An edit that leaves the file unparseable or invalid is reported; the file is
left as the editor wrote it so the mistake can be corrected.`,
		Example: `  # Edit and validate a scenario
  niac config edit clinic.yaml

  # Use a specific editor for one invocation
  EDITOR=nano niac config edit clinic.yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigEdit(cmd, args[0])
		},
	}
}

// resolveEditor returns the executable path of the operator's editor, honouring
// $EDITOR then $VISUAL and finally vi.
func resolveEditor() (string, error) {
	for _, candidate := range []string{os.Getenv("EDITOR"), os.Getenv("VISUAL"), "vi"} {
		if candidate == "" {
			continue
		}

		path, err := exec.LookPath(candidate)
		if err != nil {
			return "", fmt.Errorf("%w: %q not found: %w", errNoEditor, candidate, err)
		}

		return path, nil
	}

	return "", errNoEditor
}

func runConfigEdit(cmd *cobra.Command, file string) error {
	configFile, err := validateCLIPath(file)
	if err != nil {
		return err
	}

	if _, statErr := os.Stat(configFile); statErr != nil {
		return fmt.Errorf("cannot edit %s: %w", configFile, statErr)
	}

	editorPath, err := resolveEditor()
	if err != nil {
		return err
	}

	// editorPath is resolved by exec.LookPath and configFile by validateCLIPath.
	editor := exec.CommandContext(cmd.Context(), editorPath, configFile)
	editor.Stdin = os.Stdin
	editor.Stdout = cmd.OutOrStdout()
	editor.Stderr = cmd.ErrOrStderr()

	if runErr := editor.Run(); runErr != nil {
		return fmt.Errorf("editor %s exited with an error: %w", editorPath, runErr)
	}

	if _, loadErr := config.Load(configFile); loadErr != nil {
		return fmt.Errorf("%s is not a valid configuration after editing: %w", configFile, loadErr)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s is valid\n", configFile)

	return nil
}
