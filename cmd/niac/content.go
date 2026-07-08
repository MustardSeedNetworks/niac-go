package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/content"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

// addContentCommand wires `niac content {install,list}` onto root. The
// bundle layout, library root resolution, and security rules all live
// in internal/content + internal/library — this file is the thin cobra
// glue. Content is installed exclusively from local bundles (embedded
// essentials, the niac-content package, or a UI upload) — niac never
// fetches content over the network at runtime.
func addContentCommand(root *cobra.Command, _ *serviceOptions) {
	contentCmd := &cobra.Command{
		Use:   "content",
		Short: "Install and inspect the on-disk content library",
		Long: `Manage the content library that the daemon serves to the UI.

The library lives at ~/.niac/library by default (or /var/lib/niac/library
on packaged installs) and contains three sibling directories:

  networks/   YAML network configs
  walks/      SNMP walk files
  pcaps/      packet captures

Content ships as local bundles (embedded essentials, the niac-content
deb/rpm package, or a bundle uploaded through the UI) — there is no
network fetch. Use 'niac content install --bundle path.tar.gz' to
install one.`,
		Example: `  # Show what's in the library right now
  niac content list

  # Install a local bundle
  niac content install --bundle /tmp/niac-content.tar.gz`,
	}

	contentCmd.AddCommand(newContentInstallCmd())
	contentCmd.AddCommand(newContentListCmd())
	root.AddCommand(contentCmd)
}

func newContentInstallCmd() *cobra.Command {
	var (
		root         string
		bundlePath   string
		dryRun       bool
		skipExisting bool
	)
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install a content bundle into the library",
		Long: `Install a versioned content bundle (gzip-tar) into the local library
from a local file — no network access is made.

The bundle's top-level directories must be one of: networks, walks,
pcaps. Anything else is rejected. Each entry is re-rooted under
<library>/<kind>/ before any file is touched, so a malicious bundle
cannot escape the library.`,
		Example: `  # Install from a local bundle
  niac content install --bundle ./niac-content-v0.66.41.tar.gz

  # Install into a custom root
  niac content install --bundle ./niac-content.tar.gz --root /var/lib/niac/library

  # Preview what would be installed
  niac content install --bundle ./niac-content.tar.gz --dry-run`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runContentInstall(contentInstallArgs{
				root:         root,
				bundlePath:   bundlePath,
				dryRun:       dryRun,
				skipExisting: skipExisting,
			})
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "Library root (default: NIAC_LIBRARY_ROOT or ~/.niac/library)")
	cmd.Flags().StringVar(&bundlePath, "bundle", "", "Local bundle file to install (required)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be installed without writing files")
	cmd.Flags().BoolVar(&skipExisting, "skip-existing", false, "Skip files that already exist (default: overwrite)")
	_ = cmd.MarkFlagRequired("bundle")
	return cmd
}

type contentInstallArgs struct {
	root, bundlePath string
	dryRun           bool
	skipExisting     bool
}

func runContentInstall(args contentInstallArgs) error {
	libRoot := args.root
	if libRoot == "" {
		libRoot = library.DefaultRoot()
	}
	if _, err := library.Open(libRoot); err != nil {
		return fmt.Errorf("prepare library at %s: %w", libRoot, err)
	}

	source, sourceLabel, cleanup, err := openLocalBundle(args.bundlePath)
	if err != nil {
		return err
	}
	defer cleanup()
	defer source.Close()

	fmt.Fprintf(os.Stdout, "Installing %s into %s\n", sourceLabel, libRoot)

	manifest, err := content.Extract(source, libRoot, content.ExtractOptions{
		DryRun:       args.dryRun,
		SkipExisting: args.skipExisting,
	})
	if err != nil {
		return fmt.Errorf("extract bundle: %w", err)
	}

	verb := "Installed"
	if args.dryRun {
		verb = "Would install"
	}
	fmt.Fprintf(os.Stdout, "%s %d files (%s) across %d directories\n",
		verb, manifest.Files, content.HumanBytes(manifest.Bytes), manifest.Directories)
	for _, kind := range library.AllKinds() {
		if n := manifest.PerKind[kind]; n > 0 {
			fmt.Fprintf(os.Stdout, "  %s: %d\n", kind, n)
		}
	}
	return nil
}

func openLocalBundle(path string) (io.ReadCloser, string, func(), error) {
	// G304 waiver: path is the value of the operator-supplied --bundle
	// flag. The whole point of the flag is to let the user point at a
	// local tarball; the only file we're authorised to open is the one
	// they pass. The user can already read whatever the process uid
	// can read, so opening their chosen file gains them nothing they
	// couldn't already do with `cat`.
	f, err := os.Open(path) // #nosec G304
	if err != nil {
		return nil, "", noopCleanup, fmt.Errorf("open --bundle %s: %w", path, err)
	}
	return f, path, noopCleanup, nil
}

func noopCleanup() {}

func newContentListCmd() *cobra.Command {
	var root string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List what's installed in the library",
		Long: `Print every kind (networks / walks / pcaps) currently in the library
along with the file count and on-disk size for each, plus a TOTAL row.`,
		Example: `  # List the default library
  niac content list

  # Inspect a non-default library
  niac content list --root /var/lib/niac/library`,
		RunE: func(_ *cobra.Command, _ []string) error {
			libRoot := root
			if libRoot == "" {
				libRoot = library.DefaultRoot()
			}
			inv, err := content.Scan(libRoot)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "Library root: %s\n", inv.Root)
			fmt.Fprintf(os.Stdout, "%-12s %8s %12s\n", "Kind", "Files", "Size")
			for _, k := range inv.Kinds {
				fmt.Fprintf(os.Stdout, "%-12s %8d %12s\n", k.Kind, k.Files, content.HumanBytes(k.Bytes))
			}
			fmt.Fprintf(os.Stdout, "%-12s %8d %12s\n",
				"TOTAL", inv.Total.Files, content.HumanBytes(inv.Total.Bytes))
			return nil
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "Library root (default: NIAC_LIBRARY_ROOT or ~/.niac/library)")
	return cmd
}
