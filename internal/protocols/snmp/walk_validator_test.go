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

// TestWriteValidatedFile verifies the normal case (write lands at the given
// path) and that a symlink planted at the destination pointing outside its
// directory is rejected rather than followed — the TOCTOU escape
// pathconfine.WriteFile exists to close.
func TestWriteValidatedFile(t *testing.T) {
	t.Run("writes to the given path", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "out.walk")

		if err := writeValidatedFile(target, []byte("data")); err != nil {
			t.Fatalf("writeValidatedFile() error = %v", err)
		}

		got, err := os.ReadFile(filepath.Clean(target))
		if err != nil {
			t.Fatalf("read back: %v", err)
		}
		if string(got) != "data" {
			t.Errorf("content = %q, want %q", got, "data")
		}
	})

	t.Run("rejects a symlink escaping its directory", func(t *testing.T) {
		dir := t.TempDir()
		outside := t.TempDir()
		secret := filepath.Join(outside, "secret.walk")
		if err := os.WriteFile(secret, []byte("original"), 0o600); err != nil {
			t.Fatalf("seed secret file: %v", err)
		}

		link := filepath.Join(dir, "link.walk")
		if err := os.Symlink(secret, link); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}

		if err := writeValidatedFile(link, []byte("clobbered")); err == nil {
			t.Fatal("writeValidatedFile() followed a symlink escaping its directory, want error")
		}

		got, err := os.ReadFile(filepath.Clean(secret))
		if err != nil {
			t.Fatalf("read back: %v", err)
		}
		if string(got) != "original" {
			t.Errorf("symlink target was modified: content = %q, want %q", got, "original")
		}
	})
}
