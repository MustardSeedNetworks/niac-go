package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunRejectsIncompleteArguments(t *testing.T) {
	var output bytes.Buffer
	runErr := run([]string{"-mode", "sync"}, &output)
	if runErr == nil || !strings.Contains(runErr.Error(), "catalog and examples directories are required") {
		t.Fatalf("run() error = %v", runErr)
	}
	if output.Len() != 0 {
		t.Fatalf("run() output = %q", output.String())
	}
}

func TestRunRejectsUnknownFlag(t *testing.T) {
	runErr := run([]string{"-unknown"}, new(bytes.Buffer))
	if runErr == nil {
		t.Fatal("run() accepted an unknown flag")
	}
}
