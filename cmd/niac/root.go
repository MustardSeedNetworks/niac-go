package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/version"
)

type versionInfo struct {
	version     string
	commit      string
	date        string
	uiBuildHash string
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
		version:     version.GetVersion(),
		commit:      version.GetCommit(),
		date:        version.GetBuildTime(),
		uiBuildHash: version.GetUIBuildHash(),
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
	legacyRunner func([]string),
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
  sudo niac interactive en0 router.yaml

  # Validate configuration
  niac validate config.yaml

  # Run simulation (legacy mode)
  sudo niac en0 config.yaml

  # Run with profiling enabled (legacy mode)
  sudo niac -- --profile en0 config.yaml

  # List available templates
  niac template list

  # Generate shell completion
  niac completion bash > /etc/bash_completion.d/niac`,
		Version: info.version,
		Args:    cobra.ArbitraryArgs,
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
		Run: func(cmd *cobra.Command, args []string) {
			// If no args, show help
			if len(args) == 0 {
				_ = cmd.Help()
				return
			}
			legacyArgs := append([]string{os.Args[0]}, args...)
			legacyRunner(legacyArgs)
		},
	}

	cobra.OnInitialize(func() {
		resolveServiceDefaults(services)
	})
	rootCmd.SetVersionTemplate(
		fmt.Sprintf("niac %s (commit: %s, built: %s)\n", info.version, info.commit, info.date),
	)

	rootCmd.PersistentFlags().
		StringVar(&services.apiListen, "api-listen", "", "Expose the REST API and Web UI on this address (e.g., :8080)")
	// SECURITY FIX #101: Deprecate --api-token flag in favor of environment variable
	rootCmd.PersistentFlags().
		StringVar(&services.apiToken, "api-token", "", "Bearer token required for API/Web UI access (DEPRECATED: use NIAC_API_TOKEN env var)")
	if err := rootCmd.PersistentFlags().MarkDeprecated("api-token",
		"the value lands in /proc/<pid>/cmdline and `ps`. Set NIAC_API_TOKEN in the environment instead."); err != nil {
		// MarkDeprecated only fails for an unknown flag name, which is a programming error.
		panic(fmt.Errorf("mark --api-token deprecated: %w", err))
	}
	rootCmd.PersistentFlags().
		StringVar(&services.metricsListen, "metrics-listen", "", "Expose Prometheus metrics on this address (defaults to --api-listen)")
	rootCmd.PersistentFlags().
		StringVar(&services.storagePath, "storage-path", "", "Path to NIAC run history database (default: ~/.niac/niac.db)")
	rootCmd.PersistentFlags().
		Uint64Var(&services.alertPacketsThreshold, "alert-packets-threshold", 0, "Trigger alerts when total packets exceed this value")
	rootCmd.PersistentFlags().
		StringVar(&services.alertWebhook, "alert-webhook", "", "Optional webhook URL to notify when alerts fire")

	for _, build := range commandBuilders {
		build(rootCmd, services)
	}

	return rootCmd
}

func executeRootCommand(cmd *cobra.Command) {
	err := cmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func shouldUseLegacyCommand(args []string, root *cobra.Command) bool {
	firstArg := firstCommandArg(args, root)
	if firstArg == "" || firstArg == "--" {
		return false
	}

	if firstArg == "help" || firstArg == cobra.ShellCompRequestCmd || firstArg == cobra.ShellCompNoDescRequestCmd {
		return false
	}

	for _, cmd := range root.Commands() {
		if cmd.Name() == firstArg || cmd.HasAlias(firstArg) {
			return false
		}
	}

	return true
}

func firstCommandArg(args []string, root *cobra.Command) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return arg
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			return arg
		}
		if !isKnownRootFlag(arg, root) {
			return arg
		}
		if shouldSkipRootFlagValue(arg, root) && i+1 < len(args) {
			i++
		}
	}

	return ""
}

func isKnownRootFlag(arg string, root *cobra.Command) bool {
	name := strings.TrimLeft(arg, "-")
	if name == "" {
		return false
	}
	if flagName, _, found := strings.Cut(name, "="); found {
		name = flagName
	}

	return root.PersistentFlags().Lookup(name) != nil || root.Flags().Lookup(name) != nil
}

func shouldSkipRootFlagValue(arg string, root *cobra.Command) bool {
	name := strings.TrimLeft(arg, "-")
	if name == "" || strings.Contains(name, "=") {
		return false
	}

	flag := root.PersistentFlags().Lookup(name)
	if flag == nil {
		flag = root.Flags().Lookup(name)
	}
	if flag == nil {
		return false
	}

	return flag.Value.Type() != "bool"
}
