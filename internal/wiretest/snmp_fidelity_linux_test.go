//go:build linux && integration

package wiretest_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/daemon"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

// F2: the walk round-trip, over UDP.
//
// F1a already compares a capture against what the agent serves, but it sweeps
// the agent in-process: ProcessPDU is called directly, so nothing about the
// datagram is exercised. This test replays the largest shipped capture
// (brocade, 23,247 lines, 24 subtrees) on the wire and sweeps it with gosnmp
// from the far end of the veth, through the same classifier. What it adds is
// everything between the MIB and the manager: BER encode/decode, the agent's
// own MTU budgeting under GET-BULK, and the manager's max-repetitions choice.
//
// The in-process report is built alongside and the two are required to agree.
// A difference between them IS the finding — it is by construction something
// only the wire can see — and reporting it that way is why the cross-check is
// here rather than a second copy of F1a's assertions.
const (
	brocadeWalk      = "brocade-icx6610-24f-01.walk"
	brocadeDevice    = "wire-brocade-01"
	brocadeTarget    = "10.254.200.2"
	brocadeCommunity = "wire_public"
	// F1a sweeps from ".1" in process. A manager cannot: BER encodes the first
	// two arcs in one byte, so a one-arc OID has no encoding and gosnmp refuses
	// to marshal it. ".1.0" is the lowest OID that can go on the wire, and it
	// still sits below this walk's first row (.1.0.8802.1.1.2, LLDP-MIB under
	// iso.std) — 2,418 of its lines are outside .1.3.6.1, so a sweep rooted at
	// ".1.3" would report every one of them as dropped.
	brocadeSweepRoot  = ".1.0"
	brocadeMaxReps    = 50
	brocadeSweepLimit = 8 * time.Minute
)

// startBrocadeCapture materialises the captured-switch layout — the config in
// networks/, the capture in walks/ beside it — under a temporary
// NIAC_CONFIGS_DIR and starts the daemon from the config's path. Using
// ConfigPath rather than ConfigData is deliberate: the include_path base of an
// inline config is the daemon's own scratch directory, which is not the base a
// walk-backed device is authored against.
func startBrocadeCapture(t *testing.T) (string, string) {
	t.Helper()
	requireWire(t)

	root := t.TempDir()
	networks := filepath.Join(root, "networks")
	walks := filepath.Join(root, "walks")
	for _, dir := range []string{networks, walks} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("create %s: %v", dir, err)
		}
	}
	configPath := filepath.Join(networks, "brocade-wire.yaml")
	walkPath := filepath.Join(walks, brocadeWalk)
	copyFile(t, filepath.Join("testdata", "brocade-wire.yaml"), configPath)
	copyFile(t, filepath.Join("..", "library", "starter", "walks", brocadeWalk), walkPath)

	// Also the root the daemon is allowed to load a managed config from.
	t.Setenv("NIAC_CONFIGS_DIR", root)

	d, err := daemon.NewDaemon(daemon.Config{
		StoragePath: "disabled",
		AttachmentPolicies: []fabric.PhysicalAttachmentPolicy{{
			Interface: simIface, Mode: fabric.ModeAccess, AccessVLAN: accessVLAN,
		}},
	})
	if err != nil {
		t.Fatalf("daemon.NewDaemon: %v", err)
	}
	if startErr := d.StartSimulation(api.SimulationRequest{
		SessionID:      "wiretest-fidelity",
		Interface:      simIface,
		Attachment:     "tester",
		AttachmentMode: fabric.ModeAccess,
		AccessVLAN:     accessVLAN,
		ConfigPath:     configPath,
	}); startErr != nil {
		t.Fatalf("StartSimulation on %s: %v", simIface, startErr)
	}
	t.Cleanup(func() {
		if stopErr := d.StopSimulation(""); stopErr != nil {
			t.Errorf("StopSimulation: %v", stopErr)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = d.Shutdown(ctx)
	})

	return configPath, walkPath
}

func copyFile(t *testing.T, from, to string) {
	t.Helper()
	body, err := os.ReadFile(from)
	if err != nil {
		t.Fatalf("read %s: %v", from, err)
	}
	if writeErr := os.WriteFile(to, body, 0o600); writeErr != nil {
		t.Fatalf("write %s: %v", to, writeErr)
	}
}

// brocadeContract rebuilds the device's substitution contract from the same
// config file the daemon loaded. The contract is a property of the device and
// its walk, not of the transport, so a second agent over the same two inputs
// answers what the running one would — and F1b's lesson applies here too: it
// must be read after LoadWalkFile, because the bridge-port mapping it needs is
// the walk's.
func brocadeContract(t *testing.T, configPath, walkPath string) (*snmp.Agent, snmp.WalkContract) {
	t.Helper()

	cfg, err := config.LoadYAML(configPath)
	if err != nil {
		t.Fatalf("load %s: %v", configPath, err)
	}
	index := slices.IndexFunc(cfg.Devices, func(d config.Device) bool { return d.Name == brocadeDevice })
	if index < 0 {
		t.Fatalf("no device named %q in %s", brocadeDevice, configPath)
	}
	agent := snmp.NewAgent(&cfg.Devices[index], 0)
	if loadErr := agent.LoadWalkFile(walkPath); loadErr != nil {
		t.Fatalf("load %s: %v", walkPath, loadErr)
	}
	agent.Reindex()

	return agent, agent.WalkContract()
}

// TestSNMPWalkFidelityOverTheWire is the F2 acceptance.
func TestSNMPWalkFidelityOverTheWire(t *testing.T) {
	configPath, walkPath := startBrocadeCapture(t)

	source, err := snmp.ParseWalkFile(walkPath)
	if err != nil {
		t.Fatalf("parse %s: %v", walkPath, err)
	}
	agent, contract := brocadeContract(t, configPath, walkPath)

	client := dialBrocade(t)
	wireNext, nextTimings := walkSweep(t, client)
	reportTimings(t, "get-next", nextTimings)
	wireBulk, bulkTimings := bulkSweep(t, client, brocadeMaxReps)
	reportTimings(t, "get-bulk", bulkTimings)
	normalizeWireOIDValues(wireNext)
	normalizeWireOIDValues(wireBulk)

	if len(wireNext) == 0 {
		t.Fatal("the GET-NEXT sweep returned nothing; the agent is not answering on the wire")
	}
	if mismatch := snmp.DiffSweeps(wireNext, wireBulk); mismatch != "" {
		t.Errorf("GET-NEXT and GET-BULK disagree on the wire: %s", mismatch)
	}

	wire := snmp.BuildFidelityReport(brocadeWalk+" (wire)", source, wireNext, contract)
	logReport(t, wire)
	if wire.Unclassified > 0 {
		t.Errorf(
			"%d unclassified rows on the wire — every source OID must arrive byte-identical or in one contract bucket\n%s",
			wire.Unclassified,
			strings.Join(wire.Samples, "\n"),
		)
	}

	// The cross-check: the same capture, the same classifier, swept in process.
	// F1a asserts this report is clean; what is asserted here is that the wire
	// did not change it.
	budget := snmp.SweepBudget(len(source))
	inProcess, breakReason := snmp.SweepGetNext(agent, budget)
	if breakReason != "" {
		t.Fatalf("in-process sweep stopped early (%s); the cross-check would compare a partial run", breakReason)
	}
	local := snmp.BuildFidelityReport(brocadeWalk+" (in-process)", source, inProcess, contract)
	compareReports(t, local, wire)
	compareServed(t, inProcess, wireNext, contract)
}

// TestSNMPBulkRepetitionsDoNotChangeWhatIsServed drives the same sweep at three
// max-repetitions settings. A manager picks this number itself — net-snmp
// defaults to 10, a CyberScope asks for more — and the agent trims its response
// to stay inside one frame, so the number changes how many datagrams carry the
// walk but must not change the walk.
func TestSNMPBulkRepetitionsDoNotChangeWhatIsServed(t *testing.T) {
	startBrocadeCapture(t)
	client := dialBrocade(t)

	var baseline []string
	for _, reps := range []uint32{1, 10, brocadeMaxReps} {
		pdus, timings := bulkSweep(t, client, reps)
		reportTimings(t, fmt.Sprintf("get-bulk max-repetitions=%d", reps), timings)

		oids := make([]string, 0, len(pdus))
		for _, pdu := range pdus {
			oids = append(oids, pdu.Name)
		}
		if baseline == nil {
			baseline = oids
			t.Logf("max-repetitions=%d: %d OIDs in %d datagrams", reps, len(oids), len(timings))

			continue
		}
		if len(oids) != len(baseline) {
			t.Errorf(
				"max-repetitions=%d served %d OIDs, max-repetitions=1 served %d — the setting changed the walk",
				reps, len(oids), len(baseline),
			)

			continue
		}
		for i := range oids {
			if oids[i] != baseline[i] {
				t.Errorf(
					"max-repetitions=%d diverges from max-repetitions=1 at index %d: %s vs %s",
					reps, i, oids[i], baseline[i],
				)

				break
			}
		}
	}
}

func dialBrocade(t *testing.T) *gosnmp.GoSNMP {
	t.Helper()
	client := &gosnmp.GoSNMP{
		Target:    brocadeTarget,
		Port:      161,
		Community: brocadeCommunity,
		Version:   gosnmp.Version2c,
		Timeout:   5 * time.Second,
		Retries:   4,
		MaxOids:   gosnmp.MaxOids,
	}
	if err := client.Connect(); err != nil {
		t.Fatalf("connecting to %s: %v", brocadeTarget, err)
	}
	t.Cleanup(func() { _ = client.Conn.Close() })

	return client
}

// walkSweep is the GET-NEXT chain a discovery tool performs, one round trip per
// OID. It is chained by hand rather than through gosnmp's WalkAll for the same
// reason the sweep root is ".1.0": WalkAll is subtree-scoped and stops at the
// first OID outside its root, which on this capture ends the walk after 2,295
// of 23,122 rows — the LLDP subtree under iso.std and nothing else.
func walkSweep(t *testing.T, client *gosnmp.GoSNMP) ([]gosnmp.SnmpPDU, []time.Duration) {
	t.Helper()

	var (
		results  []gosnmp.SnmpPDU
		timings  []time.Duration
		cursor   = brocadeSweepRoot
		deadline = time.Now().Add(brocadeSweepLimit)
	)
	for time.Now().Before(deadline) {
		started := time.Now()
		packet, err := client.GetNext([]string{cursor})
		timings = append(timings, time.Since(started))
		if err != nil {
			t.Fatalf("GET-NEXT from %s: %v", cursor, err)
		}
		if len(packet.Variables) == 0 {
			return results, timings
		}
		pdu := packet.Variables[0]
		if endOfWalk(pdu) {
			return results, timings
		}
		results = append(results, pdu)
		cursor = pdu.Name
	}
	t.Fatalf(
		"GET-NEXT sweep did not reach end-of-MIB within %s (%d OIDs)",
		brocadeSweepLimit, len(results),
	)

	return nil, nil
}

// endOfWalk reports the three exceptions that end a sweep.
func endOfWalk(pdu gosnmp.SnmpPDU) bool {
	return pdu.Type == gosnmp.EndOfMibView ||
		pdu.Type == gosnmp.NoSuchObject ||
		pdu.Type == gosnmp.NoSuchInstance
}

// normalizeWireOIDValues makes one representation difference between the two
// sides comparable. An ObjectIdentifier's *value* travels as BER-encoded
// sub-identifiers, so the bytes are identical either way, but gosnmp's decoder
// renders it with a leading dot ("​.0.0") and the walk parser stores it without
// ("0.0"). Comparing those two strings as a value change would report every
// sysObjectID and every LLDP management-address row as a defect. Nothing else
// is touched: a type coercion that is NOT a pure rendering difference must
// still fail.
func normalizeWireOIDValues(pdus []gosnmp.SnmpPDU) {
	for index := range pdus {
		if pdus[index].Type != gosnmp.ObjectIdentifier {
			continue
		}
		if value, ok := pdus[index].Value.(string); ok {
			pdus[index].Value = strings.TrimPrefix(value, ".")
		}
	}
}

// bulkSweep chains GET-BULK by hand rather than calling BulkWalkAll, because the
// per-datagram round-trip time is one of the three things this row exists to
// measure and the library reports none of it.
func bulkSweep(t *testing.T, client *gosnmp.GoSNMP, maxRepetitions uint32) ([]gosnmp.SnmpPDU, []time.Duration) {
	t.Helper()

	var (
		results  []gosnmp.SnmpPDU
		timings  []time.Duration
		cursor   = brocadeSweepRoot
		deadline = time.Now().Add(brocadeSweepLimit)
	)
	for time.Now().Before(deadline) {
		started := time.Now()
		packet, err := client.GetBulk([]string{cursor}, 0, maxRepetitions)
		timings = append(timings, time.Since(started))
		if err != nil {
			t.Fatalf("GET-BULK from %s (max-repetitions %d): %v", cursor, maxRepetitions, err)
		}

		progressed := false
		for _, pdu := range packet.Variables {
			if pdu.Type == gosnmp.EndOfMibView || pdu.Type == gosnmp.NoSuchObject ||
				pdu.Type == gosnmp.NoSuchInstance {
				return results, timings
			}
			results = append(results, pdu)
			cursor = pdu.Name
			progressed = true
		}
		if !progressed {
			return results, timings
		}
	}
	t.Fatalf(
		"GET-BULK sweep did not reach end-of-MIB within %s (%d OIDs, %d datagrams)",
		brocadeSweepLimit, len(results), len(timings),
	)

	return nil, nil
}

// reportTimings records the response time per datagram. It is a report, not a
// gate: a threshold here would fail on a loaded build host and say nothing
// about the agent.
func reportTimings(t *testing.T, label string, timings []time.Duration) {
	t.Helper()
	if len(timings) == 0 {
		return
	}
	sorted := slices.Clone(timings)
	slices.Sort(sorted)
	t.Logf(
		"%s: %d datagrams, min %s p50 %s max %s",
		label,
		len(sorted),
		sorted[0].Round(time.Microsecond),
		sorted[len(sorted)/2].Round(time.Microsecond),
		sorted[len(sorted)-1].Round(time.Microsecond),
	)
}

func logReport(t *testing.T, report snmp.FidelityReport) {
	t.Helper()
	t.Logf(
		"%s: source %d, wire %d, verdicts %v, buckets %v, invented %d, unclassified %d",
		report.Walk, report.SourceOIDs, report.WireOIDs,
		report.Verdicts, report.Buckets, report.Invented, report.Unclassified,
	)
}

// compareReports is the wire-versus-in-process assertion on the source side:
// every row the capture carries must receive the same verdict whichever way the
// agent was asked. agent_added is excluded here and checked by compareServed,
// which can say something sharper about it than a count.
func compareReports(t *testing.T, local, wire snmp.FidelityReport) {
	t.Helper()

	if local.SourceOIDs != wire.SourceOIDs {
		t.Errorf("source OIDs: in-process %d, wire %d", local.SourceOIDs, wire.SourceOIDs)
	}
	for key, want := range local.Verdicts {
		if got := wire.Verdicts[key]; got != want {
			t.Errorf("verdict %q: in-process %d, wire %d", key, want, got)
		}
	}
	for key, got := range wire.Verdicts {
		if _, known := local.Verdicts[key]; !known {
			t.Errorf("verdict %q: absent in process, %d on the wire", key, got)
		}
	}
	for key, want := range local.Buckets {
		if key == string(snmp.BucketAgentAdded) {
			continue
		}
		if got := wire.Buckets[key]; got != want {
			t.Errorf("bucket %q: in-process %d, wire %d", key, want, got)
		}
	}
}

// compareServed diffs the two served OID sets.
//
// A row the in-process sweep served and the wire did not is a loss in the
// datagram path and always a failure — that is the whole reason for this test.
//
// The other direction is not symmetric, and the asymmetry is a fact about what
// is being compared rather than a licence to ignore it. The wire is answered by
// a device the daemon is running: it holds an address, it has open listeners,
// it has seen traffic. The in-process agent is a second agent over the same
// config and the same capture with no runtime at all. So the running device
// legitimately serves live rows the fresh one cannot — on this network the
// udpTable entries for its own port 53 and port 161 listeners. What is asserted
// is that every such row is one the contract already calls live or
// agent-owned; a wire-only row the contract says should have come from the
// capture is invented, and fails.
func compareServed(t *testing.T, local, wire []gosnmp.SnmpPDU, contract snmp.WalkContract) {
	t.Helper()

	onWire := servedOIDs(wire)
	inProcess := servedOIDs(local)

	var lost []string
	for oid := range inProcess {
		if !onWire[oid] {
			lost = append(lost, oid)
		}
	}
	if len(lost) > 0 {
		sort.Strings(lost)
		t.Errorf(
			"%d OIDs served in process never arrived on the wire: %s",
			len(lost), strings.Join(clip(lost), ", "),
		)
	}

	var invented []string
	live := 0
	for oid := range onWire {
		if inProcess[oid] {
			continue
		}
		if contract.Classify(oid) == snmp.BucketKept {
			invented = append(invented, oid)

			continue
		}
		live++
	}
	if len(invented) > 0 {
		sort.Strings(invented)
		t.Errorf(
			"%d OIDs appear only on the wire and the contract does not account for them: %s",
			len(invented), strings.Join(clip(invented), ", "),
		)
	}
	if live > 0 {
		t.Logf("%d live rows served by the running device that a fresh agent has no state for", live)
	}
}

func servedOIDs(pdus []gosnmp.SnmpPDU) map[string]bool {
	set := make(map[string]bool, len(pdus))
	for _, pdu := range pdus {
		set[strings.TrimPrefix(pdu.Name, ".")] = true
	}

	return set
}

// clip bounds a failure message: a systematic loss is thousands of near
// identical OIDs and the first few name the column just as well.
func clip(oids []string) []string {
	const most = 10
	if len(oids) <= most {
		return oids
	}

	return append(slices.Clone(oids[:most]), fmt.Sprintf("and %d more", len(oids)-most))
}
