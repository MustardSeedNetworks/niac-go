// Command niac-catalog-sync fetches or verifies the demo-catalog walk corpus
// against internal/catalogsync's manifest, so packaging and CI can pin an
// exact catalog commit without checking the (multi-gigabyte) walks into git.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/MustardSeedNetworks/niac-go/internal/catalogsync"
)

func main() {
	if runErr := run(os.Args[1:], os.Stdout); runErr != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", runErr)
		os.Exit(1)
	}
}

func run(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("niac-catalog-sync", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	mode := flags.String("mode", "", "operation: sync or check")
	catalogDir := flags.String("catalog-dir", "", "catalog checkout")
	examplesDir := flags.String("examples-dir", "", "generated examples directory")
	repository := flags.String("repository", "", "catalog repository URL")
	commit := flags.String("commit", "", "immutable catalog commit")
	if parseErr := flags.Parse(args); parseErr != nil {
		return parseErr
	}

	if syncErr := catalogsync.Run(catalogsync.Options{
		Mode:        catalogsync.Mode(*mode),
		CatalogDir:  *catalogDir,
		ExamplesDir: *examplesDir,
		Repository:  *repository,
		Commit:      *commit,
	}); syncErr != nil {
		return syncErr
	}
	fmt.Fprintf(output, "OK: catalog %s completed at %s.\n", *mode, *commit)
	return nil
}
