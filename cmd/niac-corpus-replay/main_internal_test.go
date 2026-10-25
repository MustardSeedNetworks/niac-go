package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSkipsMissingCatalog(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	missing := filepath.Join(t.TempDir(), "no-catalog")

	if code := run([]string{"-catalog", missing, "-skip-missing"}, &stdout, &stderr); code != exitOK {
		t.Fatalf("exit = %d, want %d; stderr %s", code, exitOK, stderr.String())
	}
	if !strings.Contains(stderr.String(), "skipped") || stdout.Len() != 0 {
		t.Fatalf("stdout %q stderr %q", stdout.String(), stderr.String())
	}
	if code := run([]string{"-catalog", missing}, &stdout, &stderr); code != exitUsage {
		t.Fatalf("missing catalog without -skip-missing exit = %d, want %d", code, exitUsage)
	}
}

func TestRunRecordingArguments(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	walk := filepath.Join(dir, "router.walk")
	body := ".1.3.6.1.2.1.1.1.0 = STRING: \"Router\"\n.1.3.6.1.2.1.1.2.0 = OID: .1.3.6.1.4.1.9.1.1\n"
	if err := os.WriteFile(walk, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	summaryPath := filepath.Join(dir, "summary.json")
	var stdout, stderr bytes.Buffer

	if code := run([]string{"-summary", summaryPath, walk}, &stdout, &stderr); code != exitOK {
		t.Fatalf("exit = %d; stderr %s", code, stderr.String())
	}
	var result struct {
		Path    string `json:"path"`
		Receipt struct {
			Status        string `json:"status"`
			SourceEntries int    `json:"source_entries"`
		} `json:"receipt"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("output is not one JSON object: %v\n%s", err, stdout.String())
	}
	if result.Receipt.Status != "completed" || result.Receipt.SourceEntries != 2 {
		t.Fatalf("result = %+v", result)
	}
	if _, err := os.Stat(summaryPath); err != nil {
		t.Fatalf("summary not written: %v", err)
	}
}

func TestRunFailsOnUnreadableRecording(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	missing := filepath.Join(t.TempDir(), "absent.walk")
	if code := run([]string{missing}, &stdout, &stderr); code != exitFailed {
		t.Fatalf("exit = %d, want %d", code, exitFailed)
	}
}

func TestRunRejectsUnknownFlag(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-no-such-flag"}, &stdout, &stderr); code != exitUsage {
		t.Fatalf("exit = %d, want %d", code, exitUsage)
	}
}
