package main

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// skipOnWindows guards the drift checks. Windows registers `niac service`,
// which the generated reference cannot carry: it is produced on Linux and
// macOS, and docs/CLI_REFERENCE.md documents that command by hand under
// "Platform-specific commands".
func skipOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("generated docs describe the cross-platform command tree")
	}
}

// repoFile reads a file relative to the repository root. The tests run with a
// working directory of cmd/niac.
func repoFile(t *testing.T, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

// TestCLIReferenceCoversEveryCommand is the acceptance check for the generated
// reference: every command a user can reach has its own section. It reads the
// committed file rather than the generator's output, so a stale commit fails
// here and not only in CI.
func TestCLIReferenceCoversEveryCommand(t *testing.T) {
	skipOnWindows(t)

	reference := repoFile(t, "docs/CLI_REFERENCE.md")

	var missing []string
	walkCommands(buildDocsRoot(), func(c *cobra.Command) {
		if c.Root() == c || !isDocumented(c) {
			return
		}
		if !strings.Contains(reference, "\n### `"+c.CommandPath()+"`\n") {
			missing = append(missing, c.CommandPath())
		}
	})

	if len(missing) == 0 {
		return
	}
	sort.Strings(missing)
	t.Fatalf("docs/CLI_REFERENCE.md is missing %d command(s); run `make cli-docs`:\n  %s",
		len(missing), strings.Join(missing, "\n  "))
}

// TestReadmeCommandTableCoversTopLevelCommands guards the README summary table
// the same way, at its one-line-per-top-level-command granularity.
func TestReadmeCommandTableCoversTopLevelCommands(t *testing.T) {
	skipOnWindows(t)

	readme := repoFile(t, "README.md")

	var missing []string
	for _, c := range buildDocsRoot().Commands() {
		if !isDocumented(c) {
			continue
		}
		if !strings.Contains(readme, "| `"+escapePipes(usageLine(c))+"` |") {
			missing = append(missing, c.CommandPath())
		}
	}

	if len(missing) == 0 {
		return
	}
	sort.Strings(missing)
	t.Fatalf("README.md command table is missing %d command(s); run `make cli-docs`:\n  %s",
		len(missing), strings.Join(missing, "\n  "))
}

// TestGeneratedDocsAreCommitted fails when the committed files differ from what
// the generator produces, the same drift gate CI runs.
func TestGeneratedDocsAreCommitted(t *testing.T) {
	skipOnWindows(t)

	for _, tc := range []struct {
		path     string
		generate func(string) (string, error)
	}{
		{"docs/CLI_REFERENCE.md", renderCLIReference},
		{"README.md", renderReadmeCommandTable},
	} {
		got, err := tc.generate(repoFile(t, tc.path))
		if err != nil {
			t.Fatalf("render %s: %v", tc.path, err)
		}
		if got != repoFile(t, tc.path) {
			t.Errorf("%s is stale; run `make cli-docs` and commit the result", tc.path)
		}
	}
}
