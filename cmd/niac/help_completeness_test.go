package main

// help_completeness_test.go — locks CLI help completeness in CI.
//
// Walks the full cobra command tree and asserts every command has a non-empty
// Short, Long, and Example, and every flag has a non-empty Usage. The audit
// flagged template / license / content subcommands as missing some of these;
// this test catches the next regression automatically.

import (
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// exemptExample lists commands cobra adds for free (`help`) or that we ship
// without examples by convention (`completion`).
func exemptExample() map[string]bool {
	return map[string]bool{
		"niac help":       true,
		"niac completion": true,
	}
}

// buildFullRoot constructs the same command tree main() builds, calling every
// real builder. info+services are stub values — the builders only use them
// for default population, not for any I/O.
func buildFullRoot() *cobra.Command {
	info := versionInfo{version: "test", commit: "abc1234", date: "2026-05-29"}
	services := new(serviceOptions)
	root := &cobra.Command{
		Use:     "niac",
		Short:   "Network In A Can - Network device simulator",
		Long:    "Stub root for testing — real root carries the same fields.",
		Example: "  niac help",
	}
	root.SetVersionTemplate("test\n")

	builders := []func(*cobra.Command, *serviceOptions){
		func(r *cobra.Command, s *serviceOptions) { addRunCommand(r, s, info) },
		func(r *cobra.Command, _ *serviceOptions) { addCompletionCommand(r) },
		addAnalyzeCommand,
		addAnalyzePcapCommand,
		addConfigCommand,
		addContentCommand,
		func(r *cobra.Command, _ *serviceOptions) { addDaemonCommand(r, info) },
		addDumpCommand,
		addInitCommand,
		func(r *cobra.Command, _ *serviceOptions) { addInstallCACommand(r) },
		addListCommand,
		addInteractiveCommand,
		addLogsCommand,
		func(r *cobra.Command, _ *serviceOptions) { addManCommand(r, info) },
		addMibZipCommand,
		addMonitorCommand,
		addNeighborsCommand,
		addSanitizeCommand,
		addServiceCommand,
		addStatusCommand,
		addTemplateCommand,
		addTopologyCommand,
		addValidateCommand,
	}
	for _, b := range builders {
		b(root, services)
	}
	return root
}

func TestCLIHelpCompleteness(t *testing.T) {
	exempt := exemptExample()
	root := buildFullRoot()

	var gaps []string
	walkCommands(root, func(c *cobra.Command) {
		path := c.CommandPath()
		if c.Short == "" {
			gaps = append(gaps, path+": missing Short")
		}
		if c.Long == "" {
			gaps = append(gaps, path+": missing Long")
		}
		if c.Example == "" && !exempt[path] {
			gaps = append(gaps, path+": missing Example")
		}
		c.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Usage == "" {
				gaps = append(gaps, path+" --"+f.Name+": missing flag Usage")
			}
		})
		c.PersistentFlags().VisitAll(func(f *pflag.Flag) {
			if f.Usage == "" {
				gaps = append(gaps, path+" --"+f.Name+": missing persistent-flag Usage")
			}
		})
	})

	if len(gaps) == 0 {
		return
	}
	sort.Strings(gaps)
	t.Fatalf("CLI help completeness gaps (%d):\n  %s", len(gaps), strings.Join(gaps, "\n  "))
}

func walkCommands(c *cobra.Command, fn func(*cobra.Command)) {
	fn(c)
	for _, sub := range c.Commands() {
		walkCommands(sub, fn)
	}
}
