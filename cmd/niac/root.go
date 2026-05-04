package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

// Populated by -ldflags at build time.
var (
	version = ""
	commit  = "" //nolint:gochecknoglobals // Set by -ldflags at build time
	date    = "" //nolint:gochecknoglobals // Set by -ldflags at build time
)

type versionInfo struct {
	version string
	commit  string
	date    string
}

// Version info constants.
const (
	defaultVersion    = "dev"
	defaultCommit     = "dev"
	defaultDate       = "unknown"
	develVersion      = "(devel)"
	vcsRevisionKey    = "vcs.revision"
	vcsTimeKey        = "vcs.time"
	versionFilePrefix = "v"
	versionFileName   = "VERSION"
)

func readVersionInfo() versionInfo {
	info := versionInfo{
		version: defaultVersion,
		commit:  defaultCommit,
		date:    defaultDate,
	}

	// Priority 1: ldflags-injected values (set during go build)
	if version != "" {
		info.version = version
	}
	if commit != "" {
		info.commit = commit
	}
	if date != "" {
		info.date = date
	}

	// Priority 2: Go build metadata (if ldflags didn't set values)
	if info.version == defaultVersion || info.commit == defaultCommit {
		populateFromBuildInfo(&info)
	}

	// Priority 3: VERSION file fallback
	populateVersionFromFile(&info)

	return info
}

// populateFromBuildInfo extracts version info from Go build metadata.
func populateFromBuildInfo(info *versionInfo) {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	if isValidVersion(buildInfo.Main.Version) {
		info.version = buildInfo.Main.Version
	}

	extractVCSSettings(info, buildInfo.Settings)
}

// isValidVersion checks if a version string is usable.
func isValidVersion(version string) bool {
	return version != "" && version != develVersion
}

// extractVCSSettings extracts VCS revision and time from build settings.
func extractVCSSettings(info *versionInfo, settings []debug.BuildSetting) {
	for _, setting := range settings {
		switch setting.Key {
		case vcsRevisionKey:
			if setting.Value != "" {
				info.commit = setting.Value
			}
		case vcsTimeKey:
			if setting.Value != "" {
				info.date = setting.Value
			}
		}
	}
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
