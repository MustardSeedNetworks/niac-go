package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var manCmd = &cobra.Command{
	Use:    "man",
	Short:  "Generate man pages",
	Long:   `Generate Unix man pages for NIAC commands.`,
	Hidden: true, // Hidden from help, mainly for maintainers
	Example: `  # Generate man pages to docs/man/
  niac man

  # Install man pages (requires sudo)
  sudo cp docs/man/* /usr/local/share/man/man1/
  sudo mandb`,
	Run: runMan,
}

func init() {
	rootCmd.AddCommand(manCmd)
}

func runMan(_ *cobra.Command, _ []string) {
	header := new(doc.GenManHeader)
	header.Title = "NIAC"
	header.Section = "1"
	header.Source = fmt.Sprintf("NIAC %s", version)
	header.Manual = "NIAC Manual"
	now := time.Now()
	header.Date = &now

	manDir := "docs/man"
	err := os.MkdirAll(manDir, 0o750)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating man directory: %v\n", err)
		os.Exit(1)
	}

	err = doc.GenManTree(rootCmd, header, manDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating man pages: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "Man pages generated in %s/\n", manDir)
	fmt.Fprintln(os.Stdout, "\nTo install:")
	fmt.Fprintln(os.Stdout, "  sudo cp docs/man/* /usr/local/share/man/man1/")
	fmt.Fprintln(os.Stdout, "  sudo mandb")
}
