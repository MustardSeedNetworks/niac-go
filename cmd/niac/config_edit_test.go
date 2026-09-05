package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeEditor writes a script that records its argv and rewrites the file it is
// handed, so the test can prove niac invoked $EDITOR on the config and read the
// result back rather than merely exiting zero.
func fakeEditor(t *testing.T, dir, body string) string {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("fake editor script needs a POSIX shell")
	}

	path := filepath.Join(dir, "fake-editor")
	script := "#!/bin/sh\nprintf '%s' \"$1\" > " + filepath.Join(dir, "argv") + "\n" + body + "\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}

	return path
}

const editTestConfig = `devices:
  - name: "router-1"
    type: "router"
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
`

func TestConfigEditInvokesEditorAndValidates(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(editTestConfig), 0o600); err != nil {
		t.Fatal(err)
	}

	editor := fakeEditor(
		t,
		tmpDir,
		`printf 'devices:\n  - name: "router-9"\n    type: "router"\n    mac: "00:11:22:33:44:99"\n    ips:\n      - "192.168.1.9"\n' > "$1"`,
	)
	t.Setenv("EDITOR", editor)

	root := newTestRootCommand()
	root.SetArgs([]string{"config", "edit", configFile})
	if err := root.Execute(); err != nil {
		t.Fatalf("config edit failed: %v", err)
	}

	argv, err := os.ReadFile(filepath.Join(tmpDir, "argv"))
	if err != nil {
		t.Fatalf("editor was not invoked: %v", err)
	}

	if string(argv) != configFile {
		t.Errorf("editor argv = %q, want %q", argv, configFile)
	}

	edited, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(edited), "router-9") {
		t.Errorf("edit did not persist; file still reads:\n%s", edited)
	}
}

func TestConfigEditRejectsInvalidResult(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(editTestConfig), 0o600); err != nil {
		t.Fatal(err)
	}

	editor := fakeEditor(t, tmpDir, `printf 'devices: [oops\n' > "$1"`)
	t.Setenv("EDITOR", editor)

	root := newTestRootCommand()
	root.SetArgs([]string{"config", "edit", configFile})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected an error for a config the editor left invalid")
	}

	if !strings.Contains(err.Error(), configFile) {
		t.Errorf("error should name the file being edited, got: %v", err)
	}
}

func TestConfigEditReportsMissingEditor(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(editTestConfig), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("EDITOR", "niac-no-such-editor")
	t.Setenv("VISUAL", "")

	root := newTestRootCommand()
	root.SetArgs([]string{"config", "edit", configFile})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected an error when $EDITOR cannot be resolved")
	}

	if !strings.Contains(err.Error(), "niac-no-such-editor") {
		t.Errorf("error should name the editor it could not find, got: %v", err)
	}
}
