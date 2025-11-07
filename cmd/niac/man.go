package main

import (
	"fmt"
	"os"

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

func runMan(cmd *cobra.Command, args []string) {
	header := &doc.GenManHeader{
		Title:   "NIAC",
		Section: "1",
		Source:  fmt.Sprintf("NIAC %s", version),
		Manual:  "NIAC Manual",
	}

	manDir := "docs/man"
	if err := os.MkdirAll(manDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating man directory: %v\n", err)
		os.Exit(1)
	}

	if err := doc.GenManTree(rootCmd, header, manDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating man pages: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Man pages generated in %s/\n", manDir)
	fmt.Println("\nTo install:")
	fmt.Println("  sudo cp docs/man/* /usr/local/share/man/man1/")
	fmt.Println("  sudo mandb")
}
