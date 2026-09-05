package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/storage"
	"github.com/MustardSeedNetworks/niac-go/internal/version"
)

type versionInfo struct {
	version      string
	commit       string
	date         string
	releaseTrain string
	uiBuildHash  string
}

// Version info constants.
const (
	defaultVersion    = "dev"
	versionFilePrefix = "v"
	versionFileName   = "VERSION"
)

// readVersionInfo returns the running binary's version metadata. Source
// of truth is the internal/version package, which prefers ldflags, then
// falls back to debug.ReadBuildInfo, then to compile-time defaults. A
// final VERSION-file fallback is layered on top for installs that ship
// the file alongside the binary.
func readVersionInfo() versionInfo {
	info := versionInfo{
		version:      version.GetVersion(),
		commit:       version.GetCommit(),
		date:         version.GetBuildTime(),
		releaseTrain: version.GetReleaseTrain(),
		uiBuildHash:  version.GetUIBuildHash(),
	}

	populateVersionFromFile(&info)

	return info
}

// populateVersionFromFile reads version from VERSION file if not already set.
func populateVersionFromFile(info *versionInfo) {
	if info.version != defaultVersion {
		return
	}

	data, err := os.ReadFile(versionFileName)
	if err != nil {
		return
	}

	v := strings.TrimSpace(string(data))
	if v == "" {
		return
	}

	if !strings.HasPrefix(v, versionFilePrefix) {
		v = versionFilePrefix + v
	}
	info.version = v
}

func newRootCommand(
	info versionInfo,
	services *serviceOptions,
	commandBuilders []func(*cobra.Command, *serviceOptions),
) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "niac",
		Short: "Network In A Can - Network device simulator",
		Long: `NIAC (Network In A Can) simulates network devices on a local interface.

It responds to ARP, ICMP, LLDP, CDP, SNMP, HTTP, and other protocols,
making simulated devices appear real on the network.

Perfect for testing network management systems, monitoring tools,
and network discovery without physical hardware.`,
		Example: `  # Quick start with template
  niac template use router router.yaml
  niac validate router.yaml
  sudo niac daemon --once en0 router.yaml --duration 60s

  # Validate configuration
  niac validate config.yaml

  # Serve the web UI and API
  sudo niac daemon

  # List available templates
  niac template list

  # Generate shell completion
  niac completion bash > /etc/bash_completion.d/niac`,
		Version: info.version,
		// Root once accepted arbitrary args and unknown flags so the legacy
		// positional runtime could hide behind them. With that runtime gone,
		// a stray word or flag is a typo, and cobra reports it as one.
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cobra.OnInitialize(func() {
		resolveServiceDefaults(services)
	})
	rootCmd.SetVersionTemplate(
		fmt.Sprintf("niac %s (commit: %s, built: %s)\n", info.version, info.commit, info.date),
	)

	rootCmd.PersistentFlags().
		StringVar(&services.storagePath, "storage-path", "", "Path to NIAC run history database (default: ~/.niac/niac.db)")
	rootCmd.PersistentFlags().
		IntVar(&services.storageKeep, "storage-keep", storage.DefaultRunRetention,
			"Run history records to keep, pruned oldest first on start (0 keeps every run)")

	for _, build := range commandBuilders {
		build(rootCmd, services)
	}

	return rootCmd
}

func executeRootCommand(cmd *cobra.Command) {
	err := cmd.Execute()
	if err == nil {
		return
	}

	fmt.Fprintln(os.Stderr, err)

	if code := exitCodeForError(err); code != 0 {
		os.Exit(code)
	}
}

// exitUsage is the conventional shell exit status for a usage error.
const exitUsage = 2

// exitCodeForError maps a command error to a process exit status. A mistyped
// command or flag is a usage error, not a run that failed: the distinction is
// what lets a script tell "niac does not have that subcommand" from "the
// simulation stopped".
func exitCodeForError(err error) int {
	if err == nil {
		return 0
	}

	if coded, ok := errors.AsType[codedError](err); ok {
		return coded.code
	}

	if isUsageError(err) {
		return exitUsage
	}

	return 1
}

// isUsageError recognises cobra's own "you typed something niac does not have"
// errors. The flag cases matter because the legacy runtime used to swallow
// unknown flags (--list-interfaces and the rest of its vocabulary); now that it
// is gone they reach cobra, and they are usage errors like any other.
func isUsageError(err error) bool {
	message := err.Error()
	for _, prefix := range []string{"unknown command", "unknown flag", "unknown shorthand flag"} {
		if strings.HasPrefix(message, prefix) {
			return true
		}
	}

	return false
}
