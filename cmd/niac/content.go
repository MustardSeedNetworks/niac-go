package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/foundation/pkg/instance"

	"github.com/MustardSeedNetworks/niac-go/internal/content"
	"github.com/MustardSeedNetworks/niac-go/internal/daemon"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
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
		root       string
		bundlePath string
		dryRun     bool
		force      bool
	)
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install a content bundle into the library",
		Long: `Install a versioned content bundle (gzip-tar) into the local library
from a local file — no network access is made.

The bundle's top-level directories must be one of: networks, walks,
pcaps. Anything else is rejected. Each entry is re-rooted under
<library>/<kind>/ before any file is touched, so a malicious bundle
cannot escape the library.

A running daemon owns its library, so the bundle is handed to it over
its API and installed there; --root cannot be honoured in that case.
With no daemon running, the install happens here, holding the same
single-instance lock a daemon takes, so one cannot start into a
half-installed library.`,
		Example: `  # Install from a local bundle
  niac content install --bundle ./niac-content-v0.66.41.tar.gz

  # Install into a custom root
  niac content install --bundle ./niac-content.tar.gz --root /var/lib/niac/library

  # Preview what would be installed
  niac content install --bundle ./niac-content.tar.gz --dry-run`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runContentInstall(contentInstallArgs{
				root:       root,
				bundlePath: bundlePath,
				dryRun:     dryRun,
				force:      force,
			})
		},
	}
	cmd.Flags().StringVar(&root, "root", "",
		"Library root when no daemon is running (default: NIAC_LIBRARY_ROOT or ~/.niac/library)")
	cmd.Flags().StringVar(&bundlePath, "bundle", "", "Local bundle file to install (required)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be installed without writing files")
	cmd.Flags().BoolVar(&force, "force", false,
		"Overwrite your own files and any bundle file you have edited (default: preserve them)")
	_ = cmd.MarkFlagRequired("bundle")
	return cmd
}

type contentInstallArgs struct {
	root, bundlePath string
	dryRun           bool
	force            bool
}

// runContentInstall installs a bundle into whichever library owns the data
// directory — the running daemon's, through its API, or this host's, under the
// single-instance lock.
//
// A daemon has the library open, and content.Extract decides per file whether
// to overwrite or preserve. An install that walked in beside it wrote into a
// tree with a live reader and no arbitration; one that fought a second CLI
// left both bundles half-applied. So the writer is singular: the daemon when
// one runs, this process when none does and it holds the lock that says so.
func runContentInstall(args contentInstallArgs) error {
	holder, running, err := instance.Probe(daemon.DefaultDataDir())
	if err != nil {
		return fmt.Errorf("check for a running instance: %w", err)
	}
	if running {
		return installThroughDaemon(args, holder)
	}

	return installLocally(args)
}

// installThroughDaemon hands the bundle to the instance that owns the data
// directory. A holder that has published no port is not serving an API —
// `niac daemon --once` is one — and guessing a port would send the bundle to
// whatever else is listening, so the refusal names what is in the way.
func installThroughDaemon(args contentInstallArgs, holder instance.Info) error {
	held := &instance.HeldError{Dir: daemon.DefaultDataDir(), PID: holder.PID, Port: holder.Port}
	if holder.Port == 0 {
		return fmt.Errorf("%w: it is not serving an API, so the bundle cannot be handed to it", held)
	}
	if args.root != "" {
		return fmt.Errorf(
			"%w: it installs into its own library, so --root cannot be honoured; drop --root or stop it",
			held)
	}

	bundle, err := os.ReadFile(args.bundlePath) // #nosec G304 -- the operator's own --bundle path; see openLocalBundle
	if err != nil {
		return fmt.Errorf("open --bundle %s: %w", args.bundlePath, err)
	}

	client, err := newCLIClient(fmt.Sprintf("https://127.0.0.1:%d", holder.Port), "", false)
	if err != nil {
		return fmt.Errorf("reach the running daemon: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Installing %s through the daemon at %s\n", args.bundlePath, client.BaseURL())

	result, err := client.InstallPack(
		context.Background(), filepath.Base(args.bundlePath), bundle, args.force, args.dryRun)
	if err != nil {
		return fmt.Errorf("install through the daemon: %w", err)
	}

	perKind := make(map[library.Kind]int, len(result.PerKind))
	for kind, n := range result.PerKind {
		perKind[library.Kind(kind)] = n
	}
	printInstallSummary(args.dryRun, content.Manifest{
		Files:       result.Files,
		Directories: result.Directories,
		Bytes:       result.Bytes,
		PerKind:     perKind,
		Preserved:   result.Preserved,
	})

	return nil
}

// installLocally does the work in this process, holding the single-instance
// lock for the whole extraction so a daemon cannot start into the tree
// half-way through it.
func installLocally(args contentInstallArgs) error {
	lock, err := acquireInstanceLock()
	if err != nil {
		return fmt.Errorf("take the instance lock: %w", err)
	}
	defer func() {
		if releaseErr := lock.Release(); releaseErr != nil {
			logging.Warningf("could not release the instance lock: %v", releaseErr)
		}
	}()

	libRoot := args.root
	if libRoot == "" {
		libRoot = library.DefaultRoot()
	}
	if _, openErr := library.Open(libRoot); openErr != nil {
		return fmt.Errorf("prepare library at %s: %w", libRoot, openErr)
	}

	source, sourceLabel, cleanup, err := openLocalBundle(args.bundlePath)
	if err != nil {
		return err
	}
	defer cleanup()
	defer source.Close()

	fmt.Fprintf(os.Stdout, "Installing %s into %s\n", sourceLabel, libRoot)

	manifest, err := daemon.InstallPack(source, libRoot, content.ExtractOptions{
		DryRun: args.dryRun,
		Force:  args.force,
	})
	if err != nil {
		return fmt.Errorf("extract bundle: %w", err)
	}

	printInstallSummary(args.dryRun, manifest)

	return nil
}

func printInstallSummary(dryRun bool, manifest content.Manifest) {
	verb := "Installed"
	if dryRun {
		verb = "Would install"
	}
	fmt.Fprintf(os.Stdout, "%s %d files (%s) across %d directories\n",
		verb, manifest.Files, content.HumanBytes(manifest.Bytes), manifest.Directories)
	for _, kind := range library.AllKinds() {
		if n := manifest.PerKind[kind]; n > 0 {
			fmt.Fprintf(os.Stdout, "  %s: %d\n", kind, n)
		}
	}
	if manifest.Preserved > 0 {
		fmt.Fprintf(os.Stdout,
			"Kept %d existing file(s) the bundle also ships; re-run with --force to replace them\n",
			manifest.Preserved)
	}
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
