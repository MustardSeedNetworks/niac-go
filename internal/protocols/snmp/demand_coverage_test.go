package snmp

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// Plan row F3(c). F1a asks whether a walk reaches the wire unchanged. This asks
// a different question the charter also depends on: of everything a real
// discovery instrument asks a device for, how much can a given walk answer at
// all. The demand side is measured, not assumed -- it comes from
// docs/design/consumer-oid-demand.tsv, cut from a capture of an EtherScope nXG
// discovering the six shipped packs.
//
// This reports; it does not gate. The owner sets the floor (row F3), and a
// driver-chosen threshold would be a number nobody agreed to.

const demandMatrixPath = "../../../docs/design/consumer-oid-demand.tsv"

type demandRow struct {
	pack     string
	role     string
	pdu      string
	oid      string
	requests int
}

func loadDemandMatrix(t *testing.T) []demandRow {
	t.Helper()

	content, err := os.ReadFile(demandMatrixPath)
	if err != nil {
		t.Fatalf("read demand matrix: %v", err)
	}

	var rows []demandRow
	for number, line := range strings.Split(string(content), "\n") {
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "pack\t") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 5 {
			t.Fatalf("%s:%d: got %d fields, want 5", demandMatrixPath, number+1, len(fields))
		}
		requests, convErr := strconv.Atoi(fields[4])
		if convErr != nil {
			t.Fatalf("%s:%d: request count %q: %v", demandMatrixPath, number+1, fields[4], convErr)
		}
		rows = append(rows, demandRow{
			pack: fields[0], role: fields[1], pdu: fields[2], oid: fields[3], requests: requests,
		})
	}
	if len(rows) == 0 {
		t.Fatalf("%s has no rows; the report would measure nothing", demandMatrixPath)
	}
	return rows
}

// servedOIDs sweeps one walk-backed agent the way a manager does and returns
// what it answered.
func servedOIDs(t *testing.T, path string) []string {
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

	served, _ := SweepGetNext(agent, SweepBudget(len(source)))
	oids := make([]string, 0, len(served))
	for _, pdu := range served {
		oids = append(oids, "."+strings.TrimPrefix(pdu.Name, "."))
	}
	sort.Strings(oids)
	return oids
}

// agentOwnedOIDs is what a device answers with no walk loaded at all: the MIB
// the agent synthesises from the authored device plus its live runtime state.
// The fixture is a forwarding device with one neighbour binding because that is
// the only shape that exercises SynthesizeARPTable (arp_topology.go) -- a bare
// device leaves ipNetToMediaTable empty, and measuring against that would
// report the instrument's largest demand block as a capture hole when it is
// runtime state a running pack does serve.
func agentOwnedOIDs(t *testing.T) []string {
	t.Helper()

	mac, err := net.ParseMAC("00:11:22:33:44:66")
	if err != nil {
		t.Fatalf("parse fixture MAC: %v", err)
	}
	device := createTestDevice()
	// SynthesizeARPTable binds a neighbour to the interface whose network
	// contains it, so the fixture needs an addressed interface; createTestDevice
	// has none and every binding would be silently dropped.
	device.Interfaces = []config.Interface{{Name: "GigabitEthernet0/0", Address: "192.168.1.1/24"}}

	agent := NewAgent(device, 0)
	agent.SynthesizeARPTable([]ARPBinding{{Address: net.ParseIP("192.168.1.50"), MAC: mac}})
	agent.Reindex()
	if !answerableInSubtree(sweepOIDs(agent), "."+ipNetToMediaTable) {
		t.Fatalf("fixture serves no .%s; the agent-owned split would report every "+
			"live-state demand as a capture hole", ipNetToMediaTable)
	}

	return sweepOIDs(agent)
}

// sweepOIDs walks an agent and returns what it answered, spelled the way walk
// files and the demand matrix spell an OID. gosnmp renders a PDU name without
// the leading dot; both committed artifacts carry one. Identical OIDs, two
// spellings -- the same asymmetry F2 found between the wire and the parser.
func sweepOIDs(agent *Agent) []string {
	served, _ := SweepGetNext(agent, SweepBudget(4096))
	oids := make([]string, 0, len(served))
	for _, pdu := range served {
		oids = append(oids, "."+strings.TrimPrefix(pdu.Name, "."))
	}
	sort.Strings(oids)
	return oids
}

// answerableExactly reports whether the request would be satisfied verbatim: a
// GET needs that exact instance, a GETNEXT or GETBULK any row inside the
// subtree it opens.
//
// This is the strict measure and it is the wrong one to lead with. Two thirds
// of the instrument's GETs name an instance it learned from the device it was
// pointed at -- ipNetToMediaTable keyed by the manager's own address, ifTable
// columns at that device's ifIndexes. A capture taken from a different network
// cannot contain those instances no matter how complete it is, so judging a
// capture by them measures the two networks' address plans, not coverage.
func answerableExactly(served []string, row demandRow) bool {
	if row.pdu == "get" {
		index := sort.SearchStrings(served, row.oid)
		return index < len(served) && served[index] == row.oid
	}
	return answerableInSubtree(served, row.oid)
}

// answerableInSubtree reports whether anything at all is served at or under an
// OID. The report leads with this applied to the demand's group rather than to
// the demanded OID itself, because a GET names a leaf: asking whether that leaf
// is served is the verbatim question again, and only the group answers "does
// this device carry the object the instrument came looking for", which is the
// question a consumer's collector actually turns on.
func answerableInSubtree(served []string, oid string) bool {
	index := sort.SearchStrings(served, oid)
	if index < len(served) && served[index] == oid {
		return true
	}
	prefix := oid + "."
	index = sort.SearchStrings(served, prefix)
	return index < len(served) && strings.HasPrefix(served[index], prefix)
}

// demandGroup is the unit the report lists a hole in. Grouping by registration
// authority rather than by a fixed arc depth keeps a vendor MIB whole instead
// of splitting it at an arbitrary level: MIB-II groups under .1.3.6.1.2.1
// separate one per group, each enterprise separates by its own number, and
// anything else -- LLDP's .1.0.8802 among them -- keeps five arcs.
func demandGroup(oid string) string {
	switch {
	case strings.HasPrefix(oid, ".1.3.6.1.2.1."):
		return trimArcs(oid, 8)
	case strings.HasPrefix(oid, ".1.3.6.1.4.1."):
		return trimArcs(oid, 8)
	default:
		return trimArcs(oid, 6)
	}
}

func trimArcs(oid string, count int) string {
	arcs := strings.Split(strings.TrimPrefix(oid, "."), ".")
	if len(arcs) <= count {
		return oid
	}
	return "." + strings.Join(arcs[:count], ".")
}

// walkCoverage is one starter walk measured against the whole demand set.
type walkCoverage struct {
	name     string
	served   []string
	answered int
	exact    int
	holes    map[string]int
}

// measureWalks scores every shipped walk against the demand set.
func measureWalks(t *testing.T, distinct map[string]string) []walkCoverage {
	t.Helper()

	var results []walkCoverage
	for _, path := range starterWalkPaths(t) {
		served := servedOIDs(t, path)
		if len(served) == 0 {
			t.Errorf("%s served nothing; coverage against it would read as total demand failure",
				filepath.Base(path))
			continue
		}
		result := walkCoverage{name: filepath.Base(path), served: served, holes: map[string]int{}}
		for key := range distinct {
			oid, pdu, _ := strings.Cut(key, "\t")
			if answerableExactly(served, demandRow{pdu: pdu, oid: oid}) {
				result.exact++
			}
			if answerableInSubtree(served, demandGroup(oid)) {
				result.answered++
				continue
			}
			result.holes[demandGroup(oid)]++
		}
		results = append(results, result)
	}
	return results
}

// splitUnreached separates the demand no walk covers into the part the agent
// answers from its own runtime and the part nothing answers. A demand no walk
// serves and the agent does is not a capture hole, and reporting the two
// together would put the instrument's largest demand block in the wrong column.
func splitUnreached(
	t *testing.T, distinct map[string]string, results []walkCoverage,
) (map[string]int, map[string]int) {
	t.Helper()

	agentOwned := agentOwnedOIDs(t)
	agent, nothing := map[string]int{}, map[string]int{}
	for key := range distinct {
		oid, _, _ := strings.Cut(key, "\t")
		group := demandGroup(oid)
		reachable := false
		for _, result := range results {
			if answerableInSubtree(result.served, group) {
				reachable = true
				break
			}
		}
		switch {
		case reachable:
		case answerableInSubtree(agentOwned, group):
			agent[group]++
		default:
			nothing[group]++
		}
	}
	return agent, nothing
}

// bestWalkForRole names the shipped walk that carries most of what the
// instrument asks a device in this role for.
func bestWalkForRole(demands map[string]bool, results []walkCoverage) (string, int) {
	best, count := "", -1
	for _, result := range results {
		answered := 0
		for key := range demands {
			oid, _, _ := strings.Cut(key, "\t")
			if answerableInSubtree(result.served, demandGroup(oid)) {
				answered++
			}
		}
		if answered > count {
			best, count = result.name, answered
		}
	}
	return best, count
}

func writeCoverageReport(
	t *testing.T,
	destination string,
	distinct map[string]string,
	roleDemand map[string]map[string]bool,
	results []walkCoverage,
	agentAnswered, unanswered map[string]int,
) {
	t.Helper()

	agentTotal := 0
	for _, count := range agentAnswered {
		agentTotal += count
	}

	var report strings.Builder
	report.WriteString("# Consumer OID demand coverage\n\n")
	report.WriteString("Plan row F3(c). What a discovery instrument asks for, joined against what\n")
	report.WriteString("each shipped starter walk can answer. Report only -- there is no floor and\n")
	report.WriteString("no gate until the owner sets one.\n\n")
	report.WriteString("Generated by:\n\n")
	report.WriteString("```bash\nNIAC_DEMAND_COVERAGE_OUT=$PWD/docs/design/consumer-oid-coverage.md \\\n")
	report.WriteString("  go test ./internal/protocols/snmp -run TestConsumerDemandCoverage\n```\n\n")
	fmt.Fprintf(&report, "Demand: %d distinct (OID, PDU) pairs from `consumer-oid-demand.tsv`.\n\n",
		len(distinct))
	report.WriteString("Demand is grouped by registration authority: one group per MIB-II group,\n")
	report.WriteString("one per enterprise, five arcs for everything else.\n\n")
	fmt.Fprintf(&report, "%d of them no walk serves and the agent does: live state such as\n"+
		"`ipNetToMediaTable`, which `arp_topology.go` fills from authoritative fleet\n"+
		"bindings on a forwarding device. Those are not capture holes and are listed\n"+
		"apart from the ones that are.\n\n", agentTotal)

	report.WriteString("## Demand the agent answers without a walk\n\n")
	report.WriteString("Runtime state, not capture content. Listed so the walk tables below are\n")
	report.WriteString("read as capture coverage and nothing else.\n\n")
	writeGroupTable(&report, agentAnswered)

	report.WriteString("\n## Demand neither the agent nor any shipped walk answers\n\n")
	report.WriteString("These are holes in the shipped set, not in any one walk.\n\n")
	writeGroupTable(&report, unanswered)

	report.WriteString("\n## Per starter walk\n\n")
	report.WriteString("`Group present` is the measure that matters: the walk serves something in\n")
	report.WriteString("the MIB group the instrument asked in, so the object it came looking for\n")
	report.WriteString("is on the device. `Verbatim` additionally requires the exact instance it\n")
	report.WriteString("asked this network's devices for -- an ifIndex, an ARP entry keyed by the\n")
	report.WriteString("manager's own address -- which a capture of a different network cannot\n")
	report.WriteString("hold. Verbatim is shown to size that gap, never as a target.\n\n")
	report.WriteString("Group presence rewards breadth over depth on purpose, which is why a\n")
	report.WriteString("588-row walk can outrank a 23,067-row one here and lose badly on verbatim:\n")
	report.WriteString("the two columns answer different questions and a pack needs both.\n\n")
	report.WriteString("| Walk | Served OIDs | Group present | Verbatim | Largest holes |\n")
	report.WriteString("| --- | --- | --- | --- | --- |\n")
	sort.Slice(results, func(i, j int) bool { return results[i].answered > results[j].answered })
	for _, result := range results {
		fmt.Fprintf(&report, "| %s | %d | %d / %d | %d | %s |\n",
			result.name, len(result.served), result.answered, len(distinct), result.exact,
			topHoles(result.holes, 3))
	}

	report.WriteString("\n## Per catalog role\n\n")
	report.WriteString("Demand each role attracts from the instrument, and the shipped walk that\n")
	report.WriteString("answers most of it. No catalog profile names a walk yet (row F5), so this\n")
	report.WriteString("says which capture would suit the role, not which one it uses.\n\n")
	report.WriteString("| Role | Distinct demands | Best walk | Answered |\n")
	report.WriteString("| --- | --- | --- | --- |\n")
	roles := make([]string, 0, len(roleDemand))
	for role := range roleDemand {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	for _, role := range roles {
		best, count := bestWalkForRole(roleDemand[role], results)
		fmt.Fprintf(&report, "| %s | %d | %s | %d |\n", role, len(roleDemand[role]), best, count)
	}

	if err := os.WriteFile(destination, []byte(report.String()), 0o600); err != nil {
		t.Fatalf("write %s: %v", destination, err)
	}
}

func writeGroupTable(report *strings.Builder, groups map[string]int) {
	report.WriteString("| Subtree | Distinct demands |\n| --- | --- |\n")
	for _, item := range rankGroups(groups) {
		fmt.Fprintf(report, "| `%s` | %d |\n", item.group, item.count)
	}
}

// TestConsumerDemandCoverage joins the measured demand against every shipped
// walk and writes the report when asked for it. The assertions are about the
// inputs staying real -- a demand matrix that stopped parsing, or a walk that
// answers nothing at all, would make the report silently meaningless.
func TestConsumerDemandCoverage(t *testing.T) {
	distinct := map[string]string{} // (oid, pdu) -> pdu class it is asked with
	roleDemand := map[string]map[string]bool{}
	for _, row := range loadDemandMatrix(t) {
		distinct[row.oid+"\t"+row.pdu] = row.pdu
		if roleDemand[row.role] == nil {
			roleDemand[row.role] = map[string]bool{}
		}
		roleDemand[row.role][row.oid+"\t"+row.pdu] = true
	}

	results := measureWalks(t, distinct)
	if len(results) == 0 {
		t.Fatal("no walks measured")
	}
	agentAnswered, unanswered := splitUnreached(t, distinct, results)

	destination := os.Getenv("NIAC_DEMAND_COVERAGE_OUT")
	if destination == "" {
		t.Logf("%d distinct (oid, pdu) demands across %d roles; %d walks measured; "+
			"%d demand groups neither the agent nor a shipped walk answers. "+
			"Set NIAC_DEMAND_COVERAGE_OUT to write the report.",
			len(distinct), len(roleDemand), len(results), len(unanswered))
		return
	}
	writeCoverageReport(t, destination, distinct, roleDemand, results, agentAnswered, unanswered)
}

type groupCount struct {
	group string
	count int
}

func rankGroups(groups map[string]int) []groupCount {
	ranked := make([]groupCount, 0, len(groups))
	for group, count := range groups {
		ranked = append(ranked, groupCount{group, count})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].count != ranked[j].count {
			return ranked[i].count > ranked[j].count
		}
		return ranked[i].group < ranked[j].group
	})
	return ranked
}

func topHoles(holes map[string]int, limit int) string {
	ranked := rankGroups(holes)
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	parts := make([]string, 0, len(ranked))
	for _, item := range ranked {
		parts = append(parts, fmt.Sprintf("`%s` (%d)", item.group, item.count))
	}
	return strings.Join(parts, ", ")
}
