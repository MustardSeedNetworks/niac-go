package main

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile creates a file with the given contents and returns its path.
func writeFile(t *testing.T, dir, name, contents string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}

	return path
}

// minimalConfig is the smallest document config.Load accepts, used where a test
// needs a valid input and cares only about what happens to the output file.
const minimalConfig = `devices:
  - name: SW1
    type: switch
`
