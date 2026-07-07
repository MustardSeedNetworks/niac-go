package snmp

import (
	"os"
	"path/filepath"
	"testing"
)

// TestValidateWalkFile_IssueOID verifies that ValidationIssue.OID is populated
// with the OID parsed from the offending line, using the same line parser
// ParseWalkFile relies on (extractLineOID), for issues tied to a specific line.
func TestValidateWalkFile_IssueOID(t *testing.T) {
	tmpDir := t.TempDir()
	walkPath := filepath.Join(tmpDir, "test.walk")

	// Line 1 is valid. Line 2 has a misspelled type ("STRNG") on a known OID,
	// which should surface a warning issue carrying that OID. Line 3 has no
	// "=" separator at all, so no OID can be parsed for its issue.
	content := ".1.3.6.1.2.1.1.1.0 = STRING: \"Cisco IOS Software\"\n" +
		".1.3.6.1.2.1.1.5.0 = STRNG: \"router1\"\n" +
		"not-a-valid-line-at-all\n"

	if err := os.WriteFile(walkPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test walk file: %v", err)
	}

	result, err := ValidateWalkFile(walkPath)
	if err != nil {
		t.Fatalf("ValidateWalkFile returned error: %v", err)
	}

	var misspellingIssue, missingEqualsIssue *ValidationIssue
	for i := range result.Issues {
		issue := &result.Issues[i]
		switch issue.Line {
		case 2:
			misspellingIssue = issue
		case 3:
			missingEqualsIssue = issue
		}
	}

	if misspellingIssue == nil {
		t.Fatalf("expected an issue on line 2, got issues: %+v", result.Issues)
	}

	wantOID := ".1.3.6.1.2.1.1.5.0"
	if misspellingIssue.OID != wantOID {
		t.Errorf("line 2 issue OID = %q, want %q", misspellingIssue.OID, wantOID)
	}

	if missingEqualsIssue == nil {
		t.Fatalf("expected an issue on line 3, got issues: %+v", result.Issues)
	}

	if missingEqualsIssue.OID != "" {
		t.Errorf("line 3 (no '=' separator) issue OID = %q, want empty", missingEqualsIssue.OID)
	}
}

// TestValidateWalkLine_OIDMatchesExtractLineOID verifies every issue returned
// for a line carries the same OID extractLineOID would parse from that line,
// confirming the validator reuses the shared walk-line OID parser rather than
// a bespoke extraction.
func TestValidateWalkLine_OIDMatchesExtractLineOID(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{name: "double dot OID", line: ".1.3.6.1..2.1.1.1.0 = STRING: \"x\""},
		{name: "trailing dot OID", line: ".1.3.6.1.2.1.1.1.0. = STRING: \"x\""},
		{name: "unquoted string value", line: ".1.3.6.1.2.1.1.5.0 = STRING: router1"},
		{name: "misspelled type", line: ".1.3.6.1.2.1.1.5.0 = STRNG: \"router1\""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := validateWalkLine(1, tt.line)
			if len(issues) == 0 {
				t.Fatalf("expected at least one issue for line %q", tt.line)
			}

			want := extractLineOID(tt.line)
			for _, issue := range issues {
				if issue.OID != want {
					t.Errorf("issue OID = %q, want %q (line=%q)", issue.OID, want, tt.line)
				}
			}
		})
	}
}
