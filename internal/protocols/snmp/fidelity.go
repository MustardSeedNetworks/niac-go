package snmp

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"
)

// Replay fidelity harness (plan rows F1a and F8).
//
// NIAC's charter is to replay a captured network to an NMS. LoadWalkFile
// deliberately does not reproduce a capture byte for byte -- the substitutions
// are named and signed off in docs/design/2026-09-replay-fidelity-contract.md
// and encoded in WalkContract. This measures whether everything *else* arrives
// unchanged: it sweeps an agent the way a manager does, and judges every source
// OID against the contract.
//
// It lives outside _test.go because two gates share it. F1a runs it over the
// shipped starter walks (fidelity_test.go); F8 runs it over the same device
// authored through each of the three surfaces and requires the reports to
// match (internal/api/authoring_fidelity_test.go). A second copy would be free
// to drift from the first, which is the whole failure this measures.

// FidelityVerdict is what happened to one source OID on the way to the wire.
type FidelityVerdict string

// The verdicts a source OID can receive on its way to the wire.
const (
	VerdictKept         FidelityVerdict = "kept"
	VerdictSubstituted  FidelityVerdict = "substituted"
	VerdictDropped      FidelityVerdict = "dropped"
	VerdictTypeChanged  FidelityVerdict = "type_changed"
	VerdictValueChanged FidelityVerdict = "value_changed"
)

// FidelityReport is the per-walk artifact. Counts first so a diff of two runs
// reads at a glance; the sample lists are bounded so a red walk does not
// produce a megabyte of JSON.
type FidelityReport struct {
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

const maxFidelitySamples = 12

// SweepGetNext chains GET-NEXT from the root the way net-snmp's snmpwalk
// does, and is the sweep a discovery tool actually performs.
func SweepGetNext(agent *Agent, budget int) ([]gosnmp.SnmpPDU, string) {
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

// SweepGetBulk asks the same question the way a modern manager does. A
// conforming agent must answer both identically.
func SweepGetBulk(agent *Agent, budget int) ([]gosnmp.SnmpPDU, string) {
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

// SweepBudget bounds a sweep in steps, derived from the walk rather than
// fixed: a sweep needing several times more bindings than the source carries
// is looping, and a fixed cap silently truncates a large capture instead —
// which read as a million dropped OIDs the first time this ran.
func SweepBudget(sourceOIDs int) int {
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

// BuildFidelityReport is the comparison itself: every source OID is kept,
// covered by exactly one contract bucket, or a finding.
func BuildFidelityReport(name string, source []WalkEntry, wire []gosnmp.SnmpPDU, contract WalkContract) FidelityReport {
	served := make(map[string]gosnmp.SnmpPDU, len(wire))
	for _, binding := range wire {
		served[trimOID(binding.Name)] = binding
	}

	report := FidelityReport{
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
			report.Verdicts[string(VerdictSubstituted)]++
			report.Buckets[string(bucket)]++
		case !onWire:
			report.Verdicts[string(VerdictDropped)]++
			report.Unclassified++
			report.ByColumn[columnOf(oid)]++
			report.addSamplef("dropped %s", oid)
		case binding.Type != entry.Type:
			report.Verdicts[string(VerdictTypeChanged)]++
			report.Unclassified++
			report.ByColumn[columnOf(oid)]++
			report.addSamplef("type %s: %v -> %v", oid, entry.Type, binding.Type)
		case !sameValue(entry.Value, binding.Value):
			report.Verdicts[string(VerdictValueChanged)]++
			report.Unclassified++
			report.ByColumn[columnOf(oid)]++
			report.addSamplef("value %s: %v -> %v", oid, entry.Value, binding.Value)
		default:
			report.Verdicts[string(VerdictKept)]++
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

func (r *FidelityReport) addSamplef(format string, args ...any) {
	if len(r.Samples) < maxFidelitySamples {
		r.Samples = append(r.Samples, fmt.Sprintf(format, args...))
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// DiffSweeps reports where two sweeps of the same agent disagree.
//
// GET-NEXT and GET-BULK ask the same question. A manager that
// switches between them must see the same MIB.
func DiffSweeps(next, bulk []gosnmp.SnmpPDU) string {
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
