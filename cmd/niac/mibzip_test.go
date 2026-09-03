package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
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

// formatWalkEntry had no coverage, which is part of why the EndOfContents case
// was spelled as a bitwise OR of two constants for so long: the expression
// matched only because gosnmp gives both the value 0x00, and nothing asserted
// what a 0x00-typed binding actually formatted as.
func TestFormatWalkEntryRendersEachType(t *testing.T) {
	tests := []struct {
		name  string
		entry snmp.WalkEntry
		want  string
	}{
		{
			name:  "octet string",
			entry: snmp.WalkEntry{OID: ".1.3.6.1.2.1.1.1.0", Type: gosnmp.OctetString, Value: "NIAC switch"},
			want:  `.1.3.6.1.2.1.1.1.0 = STRING: "NIAC switch"`,
		},
		{
			name:  "integer",
			entry: snmp.WalkEntry{OID: ".1.3.6.1.2.1.2.1.0", Type: gosnmp.Integer, Value: 7},
			want:  ".1.3.6.1.2.1.2.1.0 = INTEGER: 7",
		},
		{
			name:  "counter64",
			entry: snmp.WalkEntry{OID: ".1.3.6.1.2.1.31.1.1.1.6.1", Type: gosnmp.Counter64, Value: 42},
			want:  ".1.3.6.1.2.1.31.1.1.1.6.1 = Counter64: 42",
		},
		{
			name:  "uinteger32 shares the gauge rendering",
			entry: snmp.WalkEntry{OID: ".1.3.6.1.2.1.2.2.1.5.1", Type: gosnmp.Uinteger32, Value: 1000},
			want:  ".1.3.6.1.2.1.2.2.1.5.1 = Gauge32: 1000",
		},
		{
			name:  "timeticks",
			entry: snmp.WalkEntry{OID: ".1.3.6.1.2.1.1.3.0", Type: gosnmp.TimeTicks, Value: 12345},
			want:  ".1.3.6.1.2.1.1.3.0 = Timeticks: (12345)",
		},
		{
			name:  "null",
			entry: snmp.WalkEntry{OID: ".1.3.6.1.2.1.1.9.0", Type: gosnmp.Null, Value: nil},
			want:  ".1.3.6.1.2.1.1.9.0 = NULL: null",
		},
		{
			// EndOfContents and UnknownType are the same value, so this one case
			// covers both spellings a caller might reach for.
			name:  "end of contents falls back to the string rendering",
			entry: snmp.WalkEntry{OID: ".1.3.6.1.2.1.1.2.0", Type: gosnmp.EndOfContents, Value: "unset"},
			want:  `.1.3.6.1.2.1.1.2.0 = STRING: "unset"`,
		},
		{
			name:  "unknown type is indistinguishable from end of contents",
			entry: snmp.WalkEntry{OID: ".1.3.6.1.2.1.1.2.0", Type: gosnmp.UnknownType, Value: "unset"},
			want:  `.1.3.6.1.2.1.1.2.0 = STRING: "unset"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := formatWalkEntry(test.entry); got != test.want {
				t.Errorf("formatWalkEntry() = %q, want %q", got, test.want)
			}
		})
	}
}
