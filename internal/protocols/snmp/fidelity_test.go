package snmp

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// F1a: the harness over every shipped starter walk. The harness itself lives
// in fidelity.go, shared with the F8 authoring-parity gate.

// runFidelity loads one walk into a fresh agent and reports what the wire
// makes of it.
func runFidelity(t *testing.T, path string) FidelityReport {
	t.Helper()

	source, err := ParseWalkFile(path)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	agent := NewAgent(createTestDevice(), 0)
	if loadErr := agent.LoadWalkFile(path); loadErr != nil {
		t.Fatalf("load %s: %v", path, loadErr)
	}
	agent.Reindex()

	budget := SweepBudget(len(source))
	next, nextBreak := SweepGetNext(agent, budget)
	bulk, bulkBreak := SweepGetBulk(agent, budget)

	report := BuildFidelityReport(filepath.Base(path), source, next, agent.WalkContract())
	report.Rejected = countRejectedRows(t, path)
	report.OrderingBreak = firstNonEmpty(nextBreak, bulkBreak)
	if mismatch := DiffSweeps(next, bulk); mismatch != "" {
		report.SweepMismatch = mismatch
	}
	// A sweep that stopped early did not visit the whole MIB, so every OID it
	// never reached looks dropped. Say so rather than reporting a number that
	// is the harness's, not the agent's.
	if nextBreak != "" {
		report.Incomplete = true
		report.Verdicts = map[string]int{}
		report.Buckets = map[string]int{}
		report.ByColumn = map[string]int{}
		report.Unclassified = 0
		report.Samples = nil
	}
	return report
}

// countRejectedRows counts the walk's own lines that never became source OIDs
// because NormalizeWalkOID could not resolve their object name. It reads the
// raw file rather than the parser's output for exactly that reason: a rejected
// row is absent from both sides of the comparison, so only the file knows.
func countRejectedRows(t *testing.T, path string) int {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	rejected := 0
	for line := range strings.SplitSeq(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		oid, rest, found := strings.Cut(trimmed, "=")
		if !found {
			continue
		}
		if _, resolved := NormalizeWalkOID(strings.TrimSpace(oid)); !resolved {
			rejected++

			continue
		}
		// An OID-typed *value* is named under the same rules as the key —
		// `sysObjectID.0 = OID: SNMPv2-SMI::enterprises.9.1.1` — and is refused
		// by the same resolver, so the count has to see it too.
		if kind, value, split := strings.Cut(rest, ":"); split &&
			strings.EqualFold(strings.TrimSpace(kind), "OID") {
			if _, resolved := NormalizeWalkOID(strings.TrimSpace(value)); !resolved {
				rejected++
			}
		}
	}

	return rejected
}

// writeFidelityReport drops the artifact where CI can collect it.
func writeFidelityReport(t *testing.T, reports []FidelityReport) {
	t.Helper()
	dir := os.Getenv("NIAC_FIDELITY_REPORT_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("create report dir: %v", err)
	}
	body, err := json.MarshalIndent(reports, "", "  ")
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	if writeErr := os.WriteFile(filepath.Join(dir, "walk-fidelity.json"), body, 0o600); writeErr != nil {
		t.Fatalf("write report: %v", writeErr)
	}
}

// fidelityBaseline is the ratchet. Row F1a merges red on purpose — the
// harness has to land before the defects it finds can be fixed — but a walk
// must never get *worse* than what was measured when it merged. Row F1b
// drives every number here to zero; nothing may be added to this map without
// a finding recorded alongside it in ~/.claude/plans/niac-replay-fidelity.md.
//
// The numbers are unclassified rows: a source OID that the contract says
// should have arrived byte-identical and did not.
func fidelityBaseline() map[string]int {
	return map[string]int{
		"3com-superstack-03.walk":         207,
		"brocade-icx6610-24f-01.walk":     117,
		"huawei-versatile-01.walk":        51,
		"mikrotik-routeros-7161-chr.walk": 1,
		"netgear-gsm7212-managed-01.walk": 13,
		"oracle-linux-01.walk":            4,
		"vmware-esxi-01.walk":             29,
		"voip-device-01.walk":             2,
	}
}

// TestWalkReplayFidelity is the F1a harness over every shipped starter walk.
func TestWalkReplayFidelity(t *testing.T) {
	paths := starterWalkPaths(t)
	// A corpus run is a report, not a gate: the baseline below is the shipped
	// eighteen, and the 745-walk corpus has no per-walk entry to ratchet
	// against. It runs in the ubuntu nightly, where the finding is the report.
	_, corpus := os.LookupEnv("NIAC_WALK_CORPUS")

	reports := make([]FidelityReport, 0, len(paths))
	for _, path := range paths {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			report := runFidelity(t, path)
			reports = append(reports, report)

			if report.OrderingBreak != "" {
				t.Errorf("sweep ordering: %s", report.OrderingBreak)
			}
			if report.SweepMismatch != "" {
				t.Errorf("GET-NEXT and GET-BULK disagree: %s", report.SweepMismatch)
			}
			assertFidelity(t, name, report, corpus)
		})
	}
	writeFidelityReport(t, reports)
}

// assertFidelity judges one walk's report. A corpus run is a report rather
// than a gate: it has no per-walk baseline to ratchet against.
func assertFidelity(t *testing.T, name string, report FidelityReport, corpus bool) {
	t.Helper()

	// A rejected row never reaches the source side either, so the verdict
	// counts cannot see it. None of the shipped eighteen has one; a corpus
	// walk that does is the finding.
	switch {
	case report.Rejected == 0:
	case corpus:
		t.Logf("%d rows dropped, unresolvable object names", report.Rejected)
	default:
		t.Errorf(
			"%d rows dropped: object names that cannot be resolved to numeric OIDs",
			report.Rejected,
		)
	}

	if corpus {
		if report.Unclassified > 0 {
			t.Logf("%d unclassified rows in %d columns", report.Unclassified, len(report.ByColumn))
		}

		return
	}

	allowed := fidelityBaseline()[name]
	switch {
	case report.Unclassified > allowed:
		t.Errorf(
			"%d unclassified rows, baseline %d — every source OID must arrive byte-identical or in one contract bucket\n%s",
			report.Unclassified,
			allowed,
			strings.Join(report.Samples, "\n"),
		)
	case report.Unclassified < allowed:
		t.Errorf(
			"%d unclassified rows but the baseline still allows %d; lower the baseline in fidelityBaseline",
			report.Unclassified, allowed,
		)
	}
}

// starterWalkPaths returns the walks under test: the 18 shipped ones by
// default, or the off-repo corpus when NIAC_WALK_CORPUS points at it. The
// corpus is 745 sanitized walks and runs in the ubuntu nightly, not per-PR.
func starterWalkPaths(t *testing.T) []string {
	t.Helper()

	dir, corpus := os.LookupEnv("NIAC_WALK_CORPUS")
	if !corpus {
		dir = filepath.Join("..", "..", "library", "starter", "walks")
	}
	// The corpus nests walks under a directory per vendor; the shipped set is
	// flat. Walking covers both.
	var paths []string
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".walk") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan %s: %v", dir, err)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		// An empty directory passing silently is how this gate would rot.
		t.Fatalf("no walks under %s; the harness would be measuring nothing", dir)
	}
	// ParseWalkFile refuses a path containing "..", by design. Resolve the
	// relative walks directory rather than weakening that check.
	for index, path := range paths {
		absolute, absErr := filepath.Abs(path)
		if absErr != nil {
			t.Fatalf("resolve %s: %v", path, absErr)
		}
		paths[index] = absolute
	}
	return paths
}
