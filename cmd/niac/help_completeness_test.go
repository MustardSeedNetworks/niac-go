package main

// help_completeness_test.go — locks CLI help completeness in CI.
//
// Walks the full cobra command tree — the same tree main() builds, via
// buildDocsRoot — and asserts every command has a non-empty
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

func TestCLIHelpCompleteness(t *testing.T) {
	exempt := exemptExample()
	root := buildDocsRoot()

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
