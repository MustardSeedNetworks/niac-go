package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func addManCommand(root *cobra.Command, info versionInfo) {
	var outputDir string

	manCmd := &cobra.Command{
		Use:    "man",
		Short:  "Generate man pages",
		Long:   `Generate Unix man pages for NIAC commands.`,
		Hidden: true, // Hidden from help, mainly for maintainers
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

func runMan(root *cobra.Command, info versionInfo, outputDir string) error {
	header := new(doc.GenManHeader)
	header.Title = "NIAC"
	header.Section = "1"
	header.Source = fmt.Sprintf("NIAC %s", info.version)
	header.Manual = "NIAC Manual"
	now := time.Now()
	header.Date = &now

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
