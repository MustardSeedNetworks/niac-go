package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/capture"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
	"github.com/MustardSeedNetworks/niac-go/internal/support"
)

type supportOptions struct {
	libraryRoot string
	logPath     string
	configs     []string
	force       bool
}

func addBackupCommand(root *cobra.Command, _ *serviceOptions) {
	options := new(supportOptions)

	cmd := &cobra.Command{
		Use:   "backup <archive.tar.gz>",
		Short: "Archive the content library",
		Long: `Archive the content library -- networks, walks, captures and drafts --
into a single compressed file.

The archive holds authored content and nothing else. Certificates, the run
history database and the daemon token stay behind: they are the identity of
this host, not authored truth, and a backup carrying them would move a private
key onto whatever machine restores it.

Two backups of an unchanged library are byte-identical, so a diff of the
archives is a diff of the content.`,
		Example: `  # Back the library up
  niac backup niac-library.tar.gz

  # Back up a library somewhere other than the default
  niac backup niac-library.tar.gz --library /var/lib/niac/library`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runBackup(args[0], options)
		},
	}
	addLibraryFlag(cmd, options)
	root.AddCommand(cmd)
}

func addRestoreCommand(root *cobra.Command, _ *serviceOptions) {
	options := new(supportOptions)

	cmd := &cobra.Command{
		Use:   "restore <archive.tar.gz>",
		Short: "Restore a content library from a backup",
		Long: `Restore a content library from an archive written by 'niac backup'.

The archive is expanded beside the library and swapped in only once every
entry has landed, so a truncated or refused archive leaves the existing
library exactly as it was.

Restoring replaces the whole library. An existing library that is not empty
is refused unless --force is given. Stop the daemon first: a restore under a
running daemon replaces content it holds open.`,
		Example: `  # Restore into an empty or missing library
  niac restore niac-library.tar.gz

  # Replace an existing library
  niac restore niac-library.tar.gz --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runRestore(args[0], options)
		},
	}
	addLibraryFlag(cmd, options)
	cmd.Flags().BoolVar(&options.force, "force", false,
		"Replace an existing non-empty library")
	root.AddCommand(cmd)
}

func addSupportBundleCommand(root *cobra.Command, _ *serviceOptions) {
	options := new(supportOptions)

	cmd := &cobra.Command{
		Use:   "support-bundle <bundle.tar.gz>",
		Short: "Collect redacted diagnostics for support",
		Long: `Collect a diagnostics bundle: build metadata, the host's interface
inventory, the library's scenarios and, when named, a tail of the daemon log.

Every credential is removed before anything is written. Scenario files are
stripped over their typed fields -- SNMP communities, SNMPv3 auth and privacy
passwords, FTP passwords -- and the log tail is scrubbed of those same values
plus anything else shaped like a token or a password. Certificates are listed
by name only; no key material is read.

Read the bundle before sending it. It is your content, redacted, not
anonymised: device names, addresses and topology are all still in it.`,
		Example: `  # Bundle the library's scenarios and the host inventory
  niac support-bundle niac-support.tar.gz

  # Include a log captured from the service
  journalctl -u niac --no-pager > /tmp/niac.log
  niac support-bundle niac-support.tar.gz --log /tmp/niac.log

  # Bundle one scenario instead of the whole library
  niac support-bundle niac-support.tar.gz --config office.yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runSupportBundle(args[0], options)
		},
	}
	addLibraryFlag(cmd, options)
	cmd.Flags().StringVar(&options.logPath, "log", "",
		"Daemon log file to include, scrubbed (default: none)")
	cmd.Flags().StringArrayVar(&options.configs, "config", nil,
		"Scenario file to include, repeatable (default: every scenario in the library)")
	root.AddCommand(cmd)
}

func addLibraryFlag(cmd *cobra.Command, options *supportOptions) {
	cmd.Flags().StringVar(&options.libraryRoot, "library", "",
		"Content library directory (default: the library NIAC would use)")
}

func (o *supportOptions) root() string {
	if o.libraryRoot != "" {
		return o.libraryRoot
	}
	return library.DefaultRoot()
}

func runBackup(target string, options *supportOptions) error {
	output, err := validateCLIPath(target)
	if err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}
	if existsErr := checkOutputNotExists(output); existsErr != nil {
		return existsErr
	}
	source := options.root()
	if _, statErr := os.Stat(source); statErr != nil {
		return fmt.Errorf("library %s: %w", source, statErr)
	}

	return writeArchiveFile(output, func(file *os.File) error {
		return support.Backup(source, file)
	}, fmt.Sprintf("Library %s backed up to %s\n", source, output))
}

func runRestore(source string, options *supportOptions) error {
	input, err := validateCLIPath(source)
	if err != nil {
		return fmt.Errorf("invalid archive path: %w", err)
	}
	target := options.root()
	if targetErr := checkRestoreTarget(target, options.force); targetErr != nil {
		return targetErr
	}

	file, openErr := os.Open(input)
	if openErr != nil {
		return fmt.Errorf("open %s: %w", input, openErr)
	}
	defer func() { _ = file.Close() }()

	if restoreErr := support.Restore(file, target); restoreErr != nil {
		return restoreErr
	}
	fmt.Fprintf(os.Stdout, "Library restored from %s into %s\n", input, target)
	return nil
}

// checkRestoreTarget refuses to overwrite content the operator did not ask to
// lose. A missing or empty library needs no confirmation.
func checkRestoreTarget(target string, force bool) error {
	entries, err := os.ReadDir(target)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect library %s: %w", target, err)
	}
	if len(entries) == 0 || force {
		return nil
	}
	return fmt.Errorf("library %s is not empty; restore replaces it entirely (pass --force)", target)
}

func runSupportBundle(target string, options *supportOptions) error {
	output, err := validateCLIPath(target)
	if err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}
	if existsErr := checkOutputNotExists(output); existsErr != nil {
		return existsErr
	}

	configs := options.configs
	if len(configs) == 0 {
		if configs, err = libraryScenarios(options.root()); err != nil {
			return err
		}
	}

	bundle := support.BundleOptions{
		Configs:    configs,
		LogPath:    options.logPath,
		CertDir:    defaultCertDir(),
		Token:      os.Getenv("NIAC_TOKEN"),
		Interfaces: interfaceInventory(),
	}
	return writeArchiveFile(output, func(file *os.File) error {
		return support.WriteBundle(bundle, file)
	}, fmt.Sprintf("Support bundle written to %s (%d scenarios)\n", output, len(configs)))
}

// libraryScenarios lists the scenarios a default bundle carries. A library
// with none is not an error: the manifest and the interface inventory are
// still worth sending.
func libraryScenarios(root string) ([]string, error) {
	dir := filepath.Join(root, string(library.KindNetworks))
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", dir, err)
	}

	var scenarios []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		switch filepath.Ext(entry.Name()) {
		case ".yaml", ".yml":
			scenarios = append(scenarios, filepath.Join(dir, entry.Name()))
		}
	}
	sort.Strings(scenarios)
	return scenarios, nil
}

// interfaceInventory renders what NIAC itself can see, which is the question
// support asks when a simulation binds nothing. A host where libpcap refuses
// to enumerate still produces a bundle, carrying the refusal.
func interfaceInventory() []string {
	devices, err := capture.GetAllInterfaces()
	if err != nil {
		return []string{"interface enumeration failed: " + err.Error()}
	}

	inventory := make([]string, 0, len(devices))
	for _, device := range devices {
		fields := []string{device.Name}
		if device.Description != "" {
			fields = append(fields, device.Description)
		}
		for _, addr := range device.Addresses {
			fields = append(fields, addr.IP.String())
		}
		inventory = append(inventory, strings.Join(fields, " "))
	}
	return inventory
}

// writeArchiveFile creates the output file, hands it to write, and removes a
// half-written archive if write fails -- an operator must never mail a
// truncated bundle believing it complete.
func writeArchiveFile(output string, write func(*os.File) error, success string) error {
	file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create %s: %w", output, err)
	}

	if writeErr := write(file); writeErr != nil {
		_ = file.Close()
		_ = os.Remove(output)
		return writeErr
	}
	if closeErr := file.Close(); closeErr != nil {
		_ = os.Remove(output)
		return fmt.Errorf("close %s: %w", output, closeErr)
	}

	fmt.Fprint(os.Stdout, success)
	return nil
}
