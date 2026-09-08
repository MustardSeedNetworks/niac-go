package snmp

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/gosnmp/gosnmp"
)

// Replay fidelity harness (plan row F1a).
//
// NIAC's charter is to replay a captured network to an NMS. LoadWalkFile
// deliberately does not reproduce a capture byte for byte — the substitutions
// are named and signed off in docs/design/2026-09-replay-fidelity-contract.md
// and encoded in WalkContract. Nothing measured whether everything *else*
// arrives unchanged.
//
// This walks each starter walk twice through the agent's own PDU path —
// a GET-NEXT chain and a GET-BULK sweep — and compares the result to the
// parsed source. Deliberately not to an exported walk file: the two
// formatters disagree (FormatWalkEntries preserves Hex-STRING,
// ExportToWalkFile flattens every OctetString to STRING), so an export-based
// diff would report type changes that are the exporter's, not the agent's.

// fidelityVerdict is what happened to one source OID on the way to the wire.
type fidelityVerdict string

const (
	verdictKept        fidelityVerdict = "kept"
	verdictSubstituted fidelityVerdict = "substituted"
	verdictDropped     fidelityVerdict = "dropped"
	verdictTypeChanged fidelityVerdict = "type_changed"
	verdictValChanged  fidelityVerdict = "value_changed"
)

// fidelityReport is the per-walk artifact. Counts first so a diff of two runs
// reads at a glance; the sample lists are bounded so a red walk does not
// produce a megabyte of JSON.
type fidelityReport struct {
	Walk         string         `json:"walk"`
	SourceOIDs   int            `json:"sourceOids"`
	WireOIDs     int            `json:"wireOids"`
	Verdicts     map[string]int `json:"verdicts"`
	Buckets      map[string]int `json:"buckets"`
	Invented     int            `json:"invented"`
	Unclassified int            `json:"unclassified"`
	// ByColumn counts unclassified rows per table column (the OID with its
	// row index removed). A capture's defects run down a column — every
	// ifPhysAddress row, every ifAlias row — so the column is the unit F1b
	// fixes, and a per-row list of 200 near-identical findings is noise.
	ByColumn map[string]int `json:"unclassifiedByColumn,omitempty"`
	Samples  []string       `json:"unclassifiedSamples,omitempty"`
	// Rejected counts source lines the parser refused because their object name
	// cannot be resolved to a number. Those rows leave the source as well as the
	// wire, so without this count a walk that lost 16,430 of its 16,445 rows
	// would report as byte-perfect.
	Rejected int `json:"rejected,omitempty"`
	// Incomplete means the sweep stopped before end-of-MIB, so the verdict
	// counts below say nothing and are cleared.
	Incomplete    bool   `json:"incomplete,omitempty"`
	OrderingBreak string `json:"orderingBreak,omitempty"`
	SweepMismatch string `json:"sweepMismatch,omitempty"`
}

const maxSamples = 12

// sweepGetNext chains GET-NEXT from the root the way net-snmp's snmpwalk
// does, and is the sweep a discovery tool actually performs.
func sweepGetNext(agent *Agent, budget int) ([]gosnmp.SnmpPDU, string) {
	const root = ".1"
	var out []gosnmp.SnmpPDU
	seen := make(map[string]struct{})
	current := root
	for range budget {
		response := agent.ProcessPDU(gosnmp.GetNextRequest,
			[]gosnmp.SnmpPDU{{Name: current, Type: gosnmp.Null}}, 0, 0)
		if len(response) == 0 || isEndOfMIB(response[0]) {
			return out, ""
		}
		next := response[0]
		if _, repeat := seen[next.Name]; repeat {
			return out, fmt.Sprintf("GET-NEXT returned %s twice; a walk cannot terminate", next.Name)
		}
		if fidelityCompareOIDs(next.Name, current) <= 0 && current != root {
			return out, fmt.Sprintf("GET-NEXT went backwards: %s after %s", next.Name, current)
		}
		seen[next.Name] = struct{}{}
		out = append(out, next)
		current = next.Name
	}
	return out, fmt.Sprintf("GET-NEXT did not reach end-of-MIB within %d steps", budget)
}

// sweepGetBulk asks the same question the way a modern manager does. A
// conforming agent must answer both identically.
func sweepGetBulk(agent *Agent, budget int) ([]gosnmp.SnmpPDU, string) {
	const root = ".1"
	const repetitions = 50
	var out []gosnmp.SnmpPDU
	current := root
	for range budget {
		response := agent.ProcessPDU(gosnmp.GetBulkRequest,
			[]gosnmp.SnmpPDU{{Name: current, Type: gosnmp.Null}}, 0, repetitions)
		if len(response) == 0 {
			return out, ""
		}
		progressed := false
		for _, binding := range response {
			if isEndOfMIB(binding) {
				return out, ""
			}
			if fidelityCompareOIDs(binding.Name, current) <= 0 && current != root {
				return out, fmt.Sprintf("GET-BULK went backwards: %s after %s", binding.Name, current)
			}
			out = append(out, binding)
			current = binding.Name
			progressed = true
		}
		if !progressed {
			return out, ""
		}
	}
	return out, fmt.Sprintf("GET-BULK did not reach end-of-MIB within %d steps", budget)
}

// sweepBudget bounds a sweep in steps, derived from the walk rather than
// fixed: a sweep needing several times more bindings than the source carries
// is looping, and a fixed cap silently truncates a large capture instead —
// which read as a million dropped OIDs the first time this ran.
func sweepBudget(sourceOIDs int) int {
	const headroom = 4
	const floor = 20000
	return max(floor, sourceOIDs*headroom)
}

func isEndOfMIB(binding gosnmp.SnmpPDU) bool {
	return binding.Type == gosnmp.EndOfMibView ||
		binding.Type == gosnmp.NoSuchObject ||
		binding.Type == gosnmp.NoSuchInstance ||
		binding.Name == ""
}

// fidelityCompareOIDs is deliberately a second implementation rather than a
// call to the package's own compareOIDs: the ordering assertion exists to
// catch an agent that returns OIDs out of order, and the agent's comparator
// is exactly the code that would be wrong.
func fidelityCompareOIDs(left, right string) int {
	leftArcs := strings.Split(strings.TrimPrefix(left, "."), ".")
	rightArcs := strings.Split(strings.TrimPrefix(right, "."), ".")
	for index := 0; index < len(leftArcs) && index < len(rightArcs); index++ {
		leftArc, leftErr := strconv.Atoi(leftArcs[index])
		rightArc, rightErr := strconv.Atoi(rightArcs[index])
		if leftErr != nil || rightErr != nil {
			return strings.Compare(leftArcs[index], rightArcs[index])
		}
		if leftArc != rightArc {
			if leftArc < rightArc {
				return -1
			}
			return 1
		}
	}
	return len(leftArcs) - len(rightArcs)
}

func trimOID(oid string) string { return strings.TrimPrefix(oid, ".") }

// sameValue compares a parsed walk value with what the agent served. Both
// sides come from the same parser, so this is a plain deep-equal with one
// allowance: an OctetString may be held as string or []byte depending on the
// path that stored it, and those are the same octets.
func sameValue(source, wire any) bool {
	if reflect.DeepEqual(source, wire) {
		return true
	}
	sourceBytes, sourceOK := octets(source)
	wireBytes, wireOK := octets(wire)
	return sourceOK && wireOK && string(sourceBytes) == string(wireBytes)
}

func octets(value any) ([]byte, bool) {
	switch typed := value.(type) {
	case []byte:
		return typed, true
	case string:
		return []byte(typed), true
	}
	return nil, false
}

// buildFidelityReport is the comparison itself: every source OID is kept,
// covered by exactly one contract bucket, or a finding.
func buildFidelityReport(name string, source []WalkEntry, wire []gosnmp.SnmpPDU, contract WalkContract) fidelityReport {
	served := make(map[string]gosnmp.SnmpPDU, len(wire))
	for _, binding := range wire {
		served[trimOID(binding.Name)] = binding
	}

	report := fidelityReport{
		Walk:       name,
		SourceOIDs: len(source),
		WireOIDs:   len(wire),
		Verdicts:   map[string]int{},
		Buckets:    map[string]int{},
		ByColumn:   map[string]int{},
	}

	fromSource := make(map[string]struct{}, len(source))
	for _, entry := range source {
		oid := trimOID(entry.OID)
		fromSource[oid] = struct{}{}
		bucket := contract.Classify(oid)
		binding, onWire := served[oid]

		switch {
		case bucket != BucketKept:
			report.Verdicts[string(verdictSubstituted)]++
			report.Buckets[string(bucket)]++
		case !onWire:
			report.Verdicts[string(verdictDropped)]++
			report.Unclassified++
			report.ByColumn[columnOf(oid)]++
			report.addSamplef("dropped %s", oid)
		case binding.Type != entry.Type:
			report.Verdicts[string(verdictTypeChanged)]++
			report.Unclassified++
			report.ByColumn[columnOf(oid)]++
			report.addSamplef("type %s: %v -> %v", oid, entry.Type, binding.Type)
		case !sameValue(entry.Value, binding.Value):
			report.Verdicts[string(verdictValChanged)]++
			report.Unclassified++
			report.ByColumn[columnOf(oid)]++
			report.addSamplef("value %s: %v -> %v", oid, entry.Value, binding.Value)
		default:
			report.Verdicts[string(verdictKept)]++
		}
	}

	// Everything on the wire the source never carried is MIB-II the agent
	// synthesizes for every device — the contract's agent_added bucket.
	for _, binding := range wire {
		if _, known := fromSource[trimOID(binding.Name)]; !known {
			report.Invented++
		}
	}
	report.Buckets[string(BucketAgentAdded)] = report.Invented
	return report
}

// columnOf strips a table row's index so findings group by the column they
// are in. A scalar (ending .0) keeps its own name.
func columnOf(oid string) string {
	trimmed := strings.TrimSuffix(oid, ".0")
	if trimmed != oid {
		return oid
	}
	cut := strings.LastIndex(oid, ".")
	if cut <= 0 {
		return oid
	}
	return oid[:cut]
}

func (r *fidelityReport) addSamplef(format string, args ...any) {
	if len(r.Samples) < maxSamples {
		r.Samples = append(r.Samples, fmt.Sprintf(format, args...))
	}
}

// runFidelity loads one walk into a fresh agent and reports what the wire
// makes of it.
func runFidelity(t *testing.T, path string) fidelityReport {
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

	budget := sweepBudget(len(source))
	next, nextBreak := sweepGetNext(agent, budget)
	bulk, bulkBreak := sweepGetBulk(agent, budget)

	report := buildFidelityReport(filepath.Base(path), source, next, agent.WalkContract())
	report.Rejected = countRejectedRows(t, path)
	report.OrderingBreak = firstNonEmpty(nextBreak, bulkBreak)
	if mismatch := diffSweeps(next, bulk); mismatch != "" {
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
		oid, _, found := strings.Cut(trimmed, "=")
		if !found {
			continue
		}
		if _, resolved := NormalizeWalkOID(strings.TrimSpace(oid)); !resolved {
			rejected++
		}
	}

	return rejected
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// diffSweeps: GET-NEXT and GET-BULK ask the same question. A manager that
// switches between them must see the same MIB.
func diffSweeps(next, bulk []gosnmp.SnmpPDU) string {
	if len(next) != len(bulk) {
		return fmt.Sprintf("GET-NEXT returned %d bindings, GET-BULK %d", len(next), len(bulk))
	}
	for index := range next {
		if trimOID(next[index].Name) != trimOID(bulk[index].Name) {
			return fmt.Sprintf("binding %d: GET-NEXT %s, GET-BULK %s",
				index, next[index].Name, bulk[index].Name)
		}
	}
	return ""
}

// writeFidelityReport drops the artifact where CI can collect it.
func writeFidelityReport(t *testing.T, reports []fidelityReport) {
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

	reports := make([]fidelityReport, 0, len(paths))
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
func assertFidelity(t *testing.T, name string, report fidelityReport, corpus bool) {
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
