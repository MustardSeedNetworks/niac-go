package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func addManCommand(root *cobra.Command, info versionInfo) {
	var outputDir string

	manCmd := &cobra.Command{
		Use:   "man",
		Short: "Generate man pages",
		Long:  `Generate Unix man pages for NIAC commands.`,
		// Not hidden: generating man pages is a normal operator task, and both
		// the README and the CLI reference have always documented it.
		Example: `  # Generate man pages to ./docs/man/ (default)
  niac man

  # Generate to a specific directory
  niac man --output /tmp/niac-man

  # Install man pages (requires sudo)
  sudo cp docs/man/* /usr/local/share/man/man1/
  sudo mandb`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runMan(root, info, outputDir)
		},
	}

	manCmd.Flags().StringVarP(&outputDir, "output", "o", "docs/man",
		"output directory (relative paths resolve against the current working directory)")

	root.AddCommand(manCmd)
}

func runMan(root *cobra.Command, _ versionInfo, outputDir string) error {
	header := new(doc.GenManHeader)
	header.Title = "NIAC"
	header.Section = "1"
	// No version in the source line and no explicit date: the committed pages
	// under docs/man would otherwise claim whichever version and day they were
	// last regenerated on. A nil Date lets cobra honour SOURCE_DATE_EPOCH.
	header.Source = "NIAC"
	header.Manual = "NIAC Manual"

	if err := os.MkdirAll(outputDir, 0o750); err != nil {
		return fmt.Errorf("create man directory: %w", err)
	}

	if err := doc.GenManTree(root, header, outputDir); err != nil {
		return fmt.Errorf("generate man pages: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Man pages generated in %s/\n", outputDir)
	fmt.Fprintln(os.Stdout, "\nTo install:")
	fmt.Fprintf(os.Stdout, "  sudo cp %s/* /usr/local/share/man/man1/\n", outputDir)
	fmt.Fprintln(os.Stdout, "  sudo mandb")
	return nil
}
