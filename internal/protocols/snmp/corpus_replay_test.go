package snmp

import (
	"bufio"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gosnmp/gosnmp"
)

const replayTestWalk = `.1.3.6.1.2.1.1.1.0 = STRING: "Test Router"
.1.3.6.1.2.1.1.2.0 = OID: .1.3.6.1.4.1.9.1.1
.1.3.6.1.2.1.1.5.0 = STRING: "edge-1"
.1.3.6.1.2.1.2.2.1.10.1 = Counter32: 1234
.1.3.6.1.4.1.99999.1.0 = INTEGER: 7
`

func writeReplayFixture(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	return path
}

func TestReplayRecordingWalk(t *testing.T) {
	t.Parallel()
	receipt := ReplayRecording(writeReplayFixture(t, "router.walk", replayTestWalk), ReplayOptions{})

	if receipt.Status != ReplayCompleted {
		t.Fatalf("status = %q, want %q", receipt.Status, ReplayCompleted)
	}
	if receipt.SourceEntries != 5 || receipt.ProbeBindings != 5 || receipt.RejectedLines != 0 {
		t.Errorf("entries/probes/rejected = %d/%d/%d, want 5/5/0",
			receipt.SourceEntries, receipt.ProbeBindings, receipt.RejectedLines)
	}
	if !receipt.Identity {
		t.Error("sysDescr and sysObjectID are present but source_identity is false")
	}
	if receipt.BERFailures != 0 || receipt.KeptValueChanges != 0 || receipt.KeptTypeChanges != 0 {
		t.Errorf("unexpected fidelity failures: %+v", receipt)
	}
	if receipt.OrderingFailures != 0 || receipt.NextChecks == 0 || receipt.BulkChecks == 0 {
		t.Errorf("walk checks = next %d bulk %d ordering failures %d",
			receipt.NextChecks, receipt.BulkChecks, receipt.OrderingFailures)
	}
}

func TestReplayRecordingSnmprec(t *testing.T) {
	t.Parallel()
	const recording = "1.3.6.1.2.1.1.1.0|4|Test Switch\n" +
		"1.3.6.1.2.1.1.2.0|6|1.3.6.1.4.1.9.1.2\n" +
		"1.3.6.1.2.1.2.2.1.5.1|66|1000000000\n"
	receipt := ReplayRecording(writeReplayFixture(t, "switch.snmprec", recording), ReplayOptions{})

	if receipt.Status != ReplayCompleted || receipt.SourceEntries != 3 || !receipt.Identity {
		t.Fatalf("snmprec replay = %+v", receipt)
	}
}

// A multi-line quoted STRING must reach the wire byte for byte, and must not
// swallow the binding after it.
func TestReplayRecordingMultilineLiteral(t *testing.T) {
	t.Parallel()
	const payload = "first  \n\n#literal\n  last\\\"quote"
	raw := ".1.3.6.1.4.1.99999.1.0 = STRING: \"" + payload + "\"\n.1.3.6.1.4.1.99999.2.0 = INTEGER: 7\n"

	scanner := &corpusRecordScanner{scanner: bufio.NewScanner(strings.NewReader(raw))}
	if !scanner.Scan() {
		t.Fatal("no record")
	}
	if got, ok := corpusLiteralString(scanner.Text()); !ok || got != payload {
		t.Fatalf("literal = %q, %v; want %q", got, ok, payload)
	}
	if !scanner.Scan() {
		t.Fatal("the INTEGER record was swallowed by the string")
	}
	if scanner.Scan() {
		t.Fatal("extra record")
	}

	receipt := ReplayRecording(writeReplayFixture(t, "literal.walk", raw), ReplayOptions{})
	if receipt.Status != ReplayCompleted || receipt.SourceEntries != 2 || receipt.RejectedLines != 0 ||
		receipt.KeptValueChanges != 0 || receipt.BERFailures != 0 {
		t.Fatalf("literal replay = %+v", receipt)
	}
}

// An unterminated quote ends at the next line that starts a binding.
func TestCorpusRecordScannerUnterminatedQuote(t *testing.T) {
	t.Parallel()
	raw := ".1.3.6.1.2.1.1.1.0 = STRING: \"open\ncontinued\n.1.3.6.1.2.1.1.5.0 = STRING: \"name\"\n"
	scanner := &corpusRecordScanner{scanner: bufio.NewScanner(strings.NewReader(raw))}

	var records []string
	for scanner.Scan() {
		records = append(records, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	want := []string{".1.3.6.1.2.1.1.1.0 = STRING: \"open\ncontinued", ".1.3.6.1.2.1.1.5.0 = STRING: \"name\""}
	if !reflect.DeepEqual(records, want) {
		t.Fatalf("records = %q, want %q", records, want)
	}
}

func TestCorpusQuoteClosed(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		`"done"`:        true,
		`"done"  `:      true,
		`"open`:         false,
		`"esc\"`:        false,
		`"backslash\\"`: true,
		`unquoted`:      false,
	}
	for line, want := range cases {
		if got := corpusQuoteClosed(line); got != want {
			t.Errorf("corpusQuoteClosed(%q) = %v, want %v", line, got, want)
		}
	}
}

func TestReplayRecordingCountsRejectedLines(t *testing.T) {
	t.Parallel()
	walk := replayTestWalk + ".1.3.6.1.4.1.99999.3.0 = INTEGER: not-a-number\n"
	receipt := ReplayRecording(writeReplayFixture(t, "rejects.walk", walk), ReplayOptions{})

	if receipt.Status != ReplayCompleted || receipt.RejectedLines != 1 || receipt.SourceEntries != 5 {
		t.Fatalf("receipt = %+v, want one rejected line and five entries", receipt)
	}
}

func TestReplayRecordingMissingFile(t *testing.T) {
	t.Parallel()
	receipt := ReplayRecording(filepath.Join(t.TempDir(), "absent.walk"), ReplayOptions{})
	if receipt.Status != ReplayLoadError {
		t.Fatalf("status = %q, want %q", receipt.Status, ReplayLoadError)
	}
}

func TestReplayRecordingUDPSmoke(t *testing.T) {
	t.Parallel()
	receipt := ReplayRecording(writeReplayFixture(t, "router.walk", replayTestWalk), ReplayOptions{UDPSmoke: true})
	if receipt.Status != ReplayCompleted || !receipt.Smoke {
		t.Fatalf("smoke receipt = %+v", receipt)
	}
}

// Two replays of one recording differ only in wall time; the corpus command
// relies on that to produce output that diffs cleanly between runs.
func TestReplayRecordingDeterministic(t *testing.T) {
	t.Parallel()
	path := writeReplayFixture(t, "router.walk", replayTestWalk)
	first := ReplayRecording(path, ReplayOptions{})
	second := ReplayRecording(path, ReplayOptions{})
	first.Milliseconds, second.Milliseconds = 0, 0
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("replays differ:\n%+v\n%+v", first, second)
	}
}

func TestReplaySameValue(t *testing.T) {
	t.Parallel()
	cases := []struct {
		kind        gosnmp.Asn1BER
		left, right any
		want        bool
	}{
		{gosnmp.Null, nil, nil, true},
		{gosnmp.OctetString, "abc", []byte("abc"), true},
		{gosnmp.OctetString, "abc", []byte("abd"), false},
		{gosnmp.ObjectIdentifier, ".1.3.6", "1.3.6", true},
		{gosnmp.IPAddress, "10.0.0.1", "10.0.0.1", true},
		{gosnmp.IPAddress, "10.0.0.1", "10.0.0.2", false},
		{gosnmp.Integer, 7, 7, true},
	}
	for _, tc := range cases {
		if got := replaySameValue(tc.kind, tc.left, tc.right); got != tc.want {
			t.Errorf("replaySameValue(%v, %v, %v) = %v, want %v", tc.kind, tc.left, tc.right, got, tc.want)
		}
	}
}

func TestReplayOIDPrefix(t *testing.T) {
	t.Parallel()
	if got := replayOIDPrefix(".1.3.6.1.2.1.2.2.1.10.1", replayMissingPrefixArcs); got != "1.3.6.1.2.1.2" {
		t.Fatalf("prefix = %q", got)
	}
	if got := replayOIDPrefix("1.3.6", replayFailurePrefixArcs); got != "1.3.6" {
		t.Fatalf("short prefix = %q", got)
	}
}
