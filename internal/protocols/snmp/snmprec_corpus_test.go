package snmp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gosnmp/gosnmp"
)

// CorpusEnv points at a directory of .snmprec recordings. The corpora are
// off-repo and licensed variously (see niac-demo-catalog tools/walk-convert),
// so this runs only when an operator supplies one rather than shipping any
// third-party recording in the tree.
const CorpusEnv = "NIAC_SNMPREC_CORPUS"

// Hand-written cases fix the shape of a recording; only a real corpus shows
// whether the reader survives what devices actually emit.
func TestReadsRealRecordings(t *testing.T) {
	t.Parallel()

	root := os.Getenv(CorpusEnv)
	if root == "" {
		t.Skipf("set %s to a directory of .snmprec files", CorpusEnv)
	}

	matches, err := filepath.Glob(filepath.Join(root, "*.snmprec"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("no .snmprec files under %s", root)
	}

	for _, path := range matches {
		t.Run(filepath.Base(path), func(t *testing.T) {
			t.Parallel()
			assertRecordingReads(t, path)
		})
	}
}

func assertRecordingReads(t *testing.T, path string) {
	t.Helper()

	// LibreNMS carries eighteen zero-byte fixtures. An empty file yielding no
	// entries is correct, and says nothing about the reader.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() == 0 {
		t.Skip("empty recording")
	}

	entries, err := ParseWalkFile(path)
	if err != nil {
		t.Fatalf("ParseWalkFile: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no entries parsed")
	}

	// A recording carries an explicit tag per row, so a zero type means the
	// reader invented one — the failure this whole path exists to avoid.
	for i, entry := range entries {
		if entry.Type == gosnmp.Asn1BER(0) {
			t.Fatalf("entry %d (%s) has no type", i, entry.OID)
		}
		if entry.OID == "" {
			t.Fatalf("entry %d has no OID", i)
		}
	}

	t.Logf("%d varbinds", len(entries))
}
