package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// Markers delimiting the generated regions. Everything outside them is
// hand-written prose the generator preserves verbatim.
const (
	beginMarker = "<!-- BEGIN GENERATED COMMANDS -->"
	endMarker   = "<!-- END GENERATED COMMANDS -->"

	cliReferencePath = "docs/CLI_REFERENCE.md"
	readmePath       = "README.md"
)

// errStaleDocs reports that the committed docs differ from the generated ones.
var errStaleDocs = errors.New("generated CLI documentation is stale")

func addDocsCommand(root *cobra.Command) {
	var check bool

	docsCmd := &cobra.Command{
		Use:   "docs",
		Short: "Regenerate the CLI sections of README.md and docs/CLI_REFERENCE.md",
		Long: `Regenerate the command reference from the cobra command tree.

The command tree is the single source of truth: this rewrites the generated
region of docs/CLI_REFERENCE.md and the command table in README.md from it,
leaving the surrounding hand-written prose untouched.`,
		Example: `  # Rewrite the generated regions in place
  niac docs

  # Fail if the committed files are out of date (what CI runs)
  niac docs --check`,
		Hidden: true, // maintainer tooling, invoked through `make cli-docs`
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDocs(cmd.OutOrStdout(), check)
		},
	}
	docsCmd.Flags().BoolVar(&check, "check", false,
		"report drift and exit non-zero instead of writing the files")

	root.AddCommand(docsCmd)
}

// runDocs regenerates, or with check set only verifies, both documents.
func runDocs(out interface{ Write([]byte) (int, error) }, check bool) error {
	targets := []struct {
		path   string
		render func(string) (string, error)
	}{
		{cliReferencePath, renderCLIReference},
		{readmePath, renderReadmeCommandTable},
	}

	var stale []string
	for _, target := range targets {
		current, err := os.ReadFile(filepath.Clean(target.path))
		if err != nil {
			return fmt.Errorf("read %s: %w", target.path, err)
		}
		updated, err := target.render(string(current))
		if err != nil {
			return fmt.Errorf("render %s: %w", target.path, err)
		}
		if updated == string(current) {
			continue
		}
		if check {
			stale = append(stale, target.path)
			continue
		}
		if writeErr := os.WriteFile(target.path, []byte(updated), 0o600); writeErr != nil {
			return fmt.Errorf("write %s: %w", target.path, writeErr)
		}
		fmt.Fprintf(out, "updated %s\n", target.path)
	}

	if len(stale) > 0 {
		return fmt.Errorf("%w: %s (run `make cli-docs`)", errStaleDocs, strings.Join(stale, ", "))
	}
	return nil
}

// buildDocsRoot builds the command tree the generator documents. The version
// info is a fixed stub: nothing in the generated output may depend on the build
// metadata, or the drift gate would fail on every commit.
func buildDocsRoot() *cobra.Command {
	info := versionInfo{version: "dev", commit: "none", date: "unknown"}
	return newRootCommand(info, new(serviceOptions), func([]string) {}, commandBuilders(info))
}

// walkCommands applies fn to c and every command beneath it.
func walkCommands(c *cobra.Command, fn func(*cobra.Command)) {
	fn(c)
	for _, sub := range c.Commands() {
		walkCommands(sub, fn)
	}
}

// isDocumented reports whether a command belongs in the generated reference.
// Hidden commands are maintainer tooling, and `help` is cobra's own.
func isDocumented(c *cobra.Command) bool {
	return !c.Hidden && c.Name() != "help"
}

// usageLine renders "niac run <interface> <config>" — the command path plus the
// positional arguments declared in Use, without cobra's "[flags]" suffix.
func usageLine(c *cobra.Command) string {
	_, args, _ := strings.Cut(strings.TrimSpace(c.Use), " ")
	return strings.TrimSpace(c.CommandPath() + " " + args)
}

// documentedCommands returns every documented command in the tree, parents
// before children, sorted so the output does not depend on registration order.
func documentedCommands(root *cobra.Command) []*cobra.Command {
	var found []*cobra.Command
	walkCommands(root, func(c *cobra.Command) {
		if c != root && isDocumented(c) {
			found = append(found, c)
		}
	})
	sort.Slice(found, func(i, j int) bool {
		return found[i].CommandPath() < found[j].CommandPath()
	})
	return found
}

// renderCLIReference returns the reference with its generated region replaced.
func renderCLIReference(current string) (string, error) {
	var b strings.Builder
	b.WriteString("## Commands\n")

	commands := documentedCommands(buildDocsRoot())

	for _, c := range commands {
		fmt.Fprintf(&b, "\n- [`%s`](#%s) — %s", c.CommandPath(), anchor(c.CommandPath()), lowerFirst(c.Short))
	}
	b.WriteString("\n")
	for _, c := range commands {
		b.WriteString(commandSection(c))
	}

	return replaceGenerated(current, strings.TrimRight(b.String(), "\n"))
}

// commandSection renders one command: usage, description, flags and examples.
func commandSection(c *cobra.Command) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n### `%s`\n\n%s.\n\n", c.CommandPath(), strings.TrimRight(c.Short, "."))

	fmt.Fprintf(&b, "```text\n%s\n```\n", strings.TrimSpace(c.UseLine()))

	// Long is terminal help text, not markdown: it carries `#` comment lines,
	// `<placeholder>` angle brackets and its own indentation. Fencing it keeps
	// the markdown lint clean and renders it as the user sees it in --help.
	if long := strings.TrimSpace(c.Long); long != "" && long != strings.TrimSpace(c.Short) {
		fmt.Fprintf(&b, "\n```text\n%s\n```\n", scrubHome(dedent(long)))
	}

	if flags := strings.TrimRight(c.NonInheritedFlags().FlagUsages(), "\n"); flags != "" {
		fmt.Fprintf(&b, "\nFlags:\n\n```text\n%s\n```\n", scrubHome(flags))
	}

	if example := strings.TrimSpace(dedent(c.Example)); example != "" {
		fmt.Fprintf(&b, "\nExamples:\n\n```bash\n%s\n```\n", example)
	}
	return b.String()
}

// renderReadmeCommandTable returns the README with its command table replaced.
func renderReadmeCommandTable(current string) (string, error) {
	var b strings.Builder
	b.WriteString("| Command | Description |\n| --- | --- |\n")
	for _, c := range buildDocsRoot().Commands() {
		if !isDocumented(c) {
			continue
		}
		fmt.Fprintf(&b, "| `%s` | %s |\n", escapePipes(usageLine(c)), escapePipes(strings.TrimRight(c.Short, ".")))
	}
	return replaceGenerated(current, strings.TrimRight(b.String(), "\n"))
}

// replaceGenerated swaps the text between the markers for body.
func replaceGenerated(current, body string) (string, error) {
	start := strings.Index(current, beginMarker)
	end := strings.Index(current, endMarker)
	if start < 0 || end < start {
		return "", fmt.Errorf("missing %s / %s markers", beginMarker, endMarker)
	}
	return current[:start] + beginMarker + "\n\n" + body + "\n\n" + current[end:], nil
}

// escapePipes escapes the pipe characters a usage string such as
// "[bash|zsh|fish]" carries, which would otherwise split a markdown table cell.
func escapePipes(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}

// anchor renders a GitHub heading anchor for a command path.
func anchor(path string) string {
	return strings.ReplaceAll(path, " ", "-")
}

// lowerFirst lowercases the first rune so a Short reads as a list item.
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + strings.TrimRight(s[1:], ".")
}

// dedent removes the common leading indentation cobra's Example fields carry.
func dedent(s string) string {
	lines := strings.Split(s, "\n")
	indent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if n := len(line) - len(strings.TrimLeft(line, " ")); indent < 0 || n < indent {
			indent = n
		}
	}
	for i, line := range lines {
		if len(line) >= indent && indent > 0 {
			lines[i] = line[indent:]
		}
		lines[i] = strings.TrimRight(lines[i], " ")
	}
	return strings.Join(lines, "\n")
}

// scrubHome replaces the invoking user's home directory with ~ so a flag
// default computed from $HOME does not make the output machine-specific.
func scrubHome(s string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" || home == "/" {
		return s
	}
	return strings.ReplaceAll(s, home, "~")
}
