package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// addVersionCommand registers `niac version`.
//
// The root command already carries a --version flag, but `niac version` is
// what people type, and without a command by that name it fell through to the
// legacy runner: it printed the usage banner and exited 0, so a script asking
// the binary what it was got a success and no version.
func addVersionCommand(root *cobra.Command, info versionInfo) {
	var asJSON bool

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version, commit and build metadata",
		Long: `Print the build metadata compiled into this binary.

The fields are the ones the daemon serves at /__version — version, commit,
buildTime, uiBuildHash and releaseTrain — so a deployment check can compare
the binary on disk against the daemon it started.`,
		Args: cobra.NoArgs,
		Example: `  # Human-readable
  niac version

  # For scripts
  niac version --json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			if asJSON {
				// Same keys as /__version, so a deployment check reads the
				// binary and the running daemon the same way.
				payload := map[string]string{
					"version":      info.version,
					"commit":       info.commit,
					"buildTime":    info.date,
					"uiBuildHash":  info.uiBuildHash,
					"releaseTrain": info.releaseTrain,
				}
				encoder := json.NewEncoder(out)
				encoder.SetIndent("", "  ")

				return encoder.Encode(payload)
			}

			_, err := fmt.Fprintf(
				out,
				"niac %s (commit: %s, built: %s)\n",
				info.version,
				info.commit,
				info.date,
			)

			return err
		},
	}

	versionCmd.Flags().BoolVar(&asJSON, "json", false, "Print version metadata as JSON")
	root.AddCommand(versionCmd)
}
