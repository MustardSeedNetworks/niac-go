package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krisarmstrong/niac-go/internal/protocols/snmp"
)

func TestRunMibZipCompressInspectExpand(t *testing.T) {
	tmpDir := t.TempDir()
	walkFile := filepath.Join(tmpDir, "switch.walk")
	mibZipFile := filepath.Join(tmpDir, "switch.mz")
	expandedFile := filepath.Join(tmpDir, "switch-expanded.walk")

	writeTestFile(t, walkFile, []byte(strings.Join([]string{
		".1.3.6.1.2.1.1.1.0 = STRING: \"NIAC switch\"",
		".1.3.6.1.2.1.1.3.0 = Timeticks: (12345)",
		".1.3.6.1.2.1.2.2.1.5.1 = Gauge32: 1000000000",
	}, "\n")))

	output := captureStdout(t, func() {
		if err := runMibZipCompress(walkFile, mibZipFile); err != nil {
			t.Fatalf("runMibZipCompress: %v", err)
		}
	})
	if !strings.Contains(output, "Compressed") {
		t.Fatalf("compress output = %q, want status", output)
	}

	isMibZip, err := snmp.IsMibZipFile(mibZipFile)
	if err != nil {
		t.Fatalf("IsMibZipFile: %v", err)
	}
	if !isMibZip {
		t.Fatal("compressed output was not recognized as MibZip")
	}

	inspectOutput := captureStdout(t, func() {
		if inspectErr := runMibZipInspect(mibZipFile); inspectErr != nil {
			t.Fatalf("runMibZipInspect: %v", inspectErr)
		}
	})
	if !strings.Contains(inspectOutput, "mibzip (3 entries)") {
		t.Fatalf("inspect output = %q, want entry count", inspectOutput)
	}

	expandOutput := captureStdout(t, func() {
		if expandErr := runMibZipExpand(mibZipFile, expandedFile); expandErr != nil {
			t.Fatalf("runMibZipExpand: %v", expandErr)
		}
	})
	if !strings.Contains(expandOutput, "Expanded") {
		t.Fatalf("expand output = %q, want status", expandOutput)
	}

	expanded, err := os.ReadFile(expandedFile)
	if err != nil {
		t.Fatalf("ReadFile expanded: %v", err)
	}
	if !strings.Contains(string(expanded), `.1.3.6.1.2.1.1.1.0 = STRING: "NIAC switch"`) {
		t.Fatalf("expanded walk missing sysDescr: %s", expanded)
	}
}

func TestRunMibZipInspectTextFile(t *testing.T) {
	tmpDir := t.TempDir()
	walkFile := filepath.Join(tmpDir, "plain.walk")
	writeTestFile(t, walkFile, []byte(".1.3.6.1.2.1.1.1.0 = STRING: \"plain\"\n"))

	output := captureStdout(t, func() {
		if err := runMibZipInspect(walkFile); err != nil {
			t.Fatalf("runMibZipInspect: %v", err)
		}
	})

	if !strings.Contains(output, "text or unknown format") {
		t.Fatalf("inspect output = %q, want text status", output)
	}
}
