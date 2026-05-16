package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/niac-go/internal/content"
	"github.com/krisarmstrong/niac-go/internal/library"
)

// addContentCommand wires `niac content {install,list,update}` onto
// root. The bundle layout, library root resolution, and security
// rules all live in internal/content + internal/library — this file is
// the thin cobra glue.
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

Bundled content is published per release at:
  https://github.com/krisarmstrong/niac-go/releases

Use 'niac content install' to fetch and unpack the bundle matching the
running daemon, or 'niac content install --bundle path.tar.gz' to install
from a local mirror.`,
	}

	contentCmd.AddCommand(newContentInstallCmd())
	contentCmd.AddCommand(newContentListCmd())
	contentCmd.AddCommand(newContentUpdateCmd())
	root.AddCommand(contentCmd)
}

func newContentInstallCmd() *cobra.Command {
	var (
		root         string
		bundlePath   string
		bundleURL    string
		version      string
		dryRun       bool
		skipExisting bool
	)
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install a content bundle into the library",
		Long: `Install a versioned content bundle (gzip-tar) into the local library.

Source resolution (first match wins):
  --bundle <path>    Local tarball — useful for offline installs
  --url <url>        Explicit HTTP(S) URL
  --version <vX.Y.Z> Pull from the matching GitHub release
  (default)          Pull the bundle for the running daemon's version

The bundle's top-level directories must be one of: networks, walks,
pcaps. Anything else is rejected. Each entry is re-rooted under
<library>/<kind>/ before any file is touched, so a malicious bundle
cannot escape the library.`,
		Example: `  # Install the bundle matching this binary's version
  niac content install

  # Install from a local file (no network needed)
  niac content install --bundle ./niac-content-v0.66.41.tar.gz

  # Install a specific version into a custom root
  niac content install --version v0.66.40 --root /var/lib/niac/library

  # Preview what would be installed
  niac content install --dry-run`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runContentInstall(cmd.Context(), contentInstallArgs{
				root:         root,
				bundlePath:   bundlePath,
				bundleURL:    bundleURL,
				version:      version,
				dryRun:       dryRun,
				skipExisting: skipExisting,
			})
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "Library root (default: NIAC_LIBRARY_ROOT or ~/.niac/library)")
	cmd.Flags().StringVar(&bundlePath, "bundle", "", "Local bundle file to install")
	cmd.Flags().StringVar(&bundleURL, "url", "", "Explicit HTTP(S) URL to fetch the bundle from")
	cmd.Flags().StringVar(&version, "version", "", "Release tag to pull (default: this binary's version)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be installed without writing files")
	cmd.Flags().BoolVar(&skipExisting, "skip-existing", false, "Skip files that already exist (default: overwrite)")
	return cmd
}

type contentInstallArgs struct {
	root, bundlePath, bundleURL, version string
	dryRun, skipExisting                 bool
}

func runContentInstall(ctx context.Context, args contentInstallArgs) error {
	libRoot := args.root
	if libRoot == "" {
		libRoot = library.DefaultRoot()
	}
	if _, err := library.Open(libRoot); err != nil {
		return fmt.Errorf("prepare library at %s: %w", libRoot, err)
	}

	source, sourceLabel, cleanup, err := resolveBundleSource(ctx, args)
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

// resolveBundleSource picks a source based on the install args and
// returns its bytes alongside a human-friendly label and a cleanup
// func that removes any temp file the remote-fetch path created.
func resolveBundleSource(ctx context.Context, args contentInstallArgs) (io.ReadCloser, string, func(), error) {
	if args.bundlePath != "" {
		return openLocalBundle(args.bundlePath)
	}
	version := strings.TrimSpace(args.version)
	if version == "" {
		version = readVersionInfo().version
	}
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	return downloadBundle(ctx, version, args.bundleURL)
}

func openLocalBundle(path string) (io.ReadCloser, string, func(), error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", noopCleanup, fmt.Errorf("open --bundle %s: %w", path, err)
	}
	return f, path, noopCleanup, nil
}

func downloadBundle(ctx context.Context, version, explicitURL string) (io.ReadCloser, string, func(), error) {
	res, err := content.Download(ctx, version, content.DownloadOptions{URL: explicitURL})
	if err != nil {
		return nil, "", noopCleanup, fmt.Errorf("download bundle: %w", err)
	}
	f, openErr := os.Open(res.Path)
	if openErr != nil {
		_ = os.Remove(res.Path)
		return nil, "", noopCleanup, fmt.Errorf("reopen downloaded bundle: %w", openErr)
	}
	if !res.ChecksumOK {
		fmt.Fprintf(os.Stderr, "warning: %s\n", res.ChecksumReason)
	}
	label := fmt.Sprintf("bundle %s (%s)", version, content.HumanBytes(res.Bytes))
	cleanup := func() { _ = os.Remove(res.Path) }
	return f, label, cleanup, nil
}

func noopCleanup() {}

func newContentListCmd() *cobra.Command {
	var root string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List what's installed in the library",
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

func newContentUpdateCmd() *cobra.Command {
	var (
		root         string
		dryRun       bool
		skipExisting bool
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Re-install the bundle matching this binary's version",
		Long: `Convenience wrapper for 'niac content install' that always pulls the
bundle for the version the running niac binary was built from.

This is the safest way to keep library content paired with the binary
after upgrading via the system package manager:

  sudo dnf upgrade niac
  niac content update`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runContentInstall(cmd.Context(), contentInstallArgs{
				root:         root,
				dryRun:       dryRun,
				skipExisting: skipExisting,
			})
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "Library root (default: NIAC_LIBRARY_ROOT or ~/.niac/library)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would change without writing files")
	cmd.Flags().BoolVar(&skipExisting, "skip-existing", false, "Skip files that already exist (default: overwrite)")
	return cmd
}
