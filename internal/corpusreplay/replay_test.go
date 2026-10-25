package corpusreplay_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/corpusreplay"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

const fixtureWalk = `.1.3.6.1.2.1.1.1.0 = STRING: "Fixture Router"
.1.3.6.1.2.1.1.2.0 = OID: .1.3.6.1.4.1.9.1.1
.1.3.6.1.2.1.1.5.0 = STRING: "fixture-1"
.1.3.6.1.2.1.2.2.1.10.1 = Counter32: 1234
`

const fixtureRecording = "1.3.6.1.2.1.1.1.0|4|Fixture Switch\n" +
	"1.3.6.1.2.1.1.2.0|6|1.3.6.1.4.1.9.1.2\n"

// fixtureCatalog lays out a miniature niac-demo-catalog: three recordings
// under walks/, one of them bzip2-compressed, and a qualified.csv whose
// baseline columns are filled from a replay of the same files.
func fixtureCatalog(t *testing.T) string {
	t.Helper()
	catalog := t.TempDir()
	walks := filepath.Join(catalog, corpusreplay.RecordingsDir)
	compressed, err := os.ReadFile(filepath.Join("testdata", "compressed.snmprec.bz2"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	files := map[string][]byte{
		"sanitized/router.walk":           []byte(fixtureWalk),
		"external/switch.snmprec":         []byte(fixtureRecording),
		"external/compressed.snmprec.bz2": compressed,
	}
	var manifest strings.Builder
	manifest.WriteString("original_path,effective_path,source_sha256,status,source_entries,source_identity," +
		"kept_value_changes,udp_smoke,net_snmp_client,failure_types\n")
	for _, rel := range []string{"external/compressed.snmprec.bz2", "external/switch.snmprec", "sanitized/router.walk"} {
		path := filepath.Join(walks, filepath.FromSlash(rel))
		if err = os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, files[rel], 0o600); err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(files[rel])
		receipt := snmp.ReplayRecording(path, snmp.ReplayOptions{})
		if strings.HasSuffix(rel, ".bz2") {
			// The reader cannot load .bz2; the baseline is what the expanded
			// recording replays to, which the test asserts below.
			receipt = snmp.ReplayReceipt{Status: snmp.ReplayCompleted, SourceEntries: 3, Identity: true}
		}
		manifest.WriteString(strings.Join([]string{
			rel, rel, hex.EncodeToString(digest[:]), receipt.Status,
			strconv.Itoa(receipt.SourceEntries), boolCell(receipt.Identity), "0", "True", "False", "{}",
		}, ",") + "\n")
	}
	manifestPath := filepath.Join(catalog, filepath.FromSlash(corpusreplay.ManifestPath))
	if err = os.MkdirAll(filepath.Dir(manifestPath), 0o750); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(manifestPath, []byte(manifest.String()), 0o600); err != nil {
		t.Fatal(err)
	}

	return catalog
}

// boolCell writes booleans the way the qualification worker (Python) did.
func boolCell(b bool) string {
	if b {
		return "True"
	}

	return "False"
}

func TestRunMatchesBaseline(t *testing.T) {
	t.Parallel()
	rows, err := corpusreplay.LoadCatalog(fixtureCatalog(t))
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	results := corpusreplay.Run(context.Background(), rows, corpusreplay.Options{Jobs: 2, UDPSmoke: true})

	if len(results) != 3 {
		t.Fatalf("results = %d, want 3", len(results))
	}
	for _, result := range results {
		if result.Receipt.Status != snmp.ReplayCompleted || len(result.Drift) > 0 || result.SourceChanged {
			t.Errorf("%s: status %q drift %v changed %v error %q", result.Path,
				result.Receipt.Status, result.Drift, result.SourceChanged, result.Receipt.Error)
		}
	}
	if results[0].Path != "external/compressed.snmprec.bz2" || results[0].Receipt.SourceEntries != 3 {
		t.Errorf("compressed recording did not replay expanded: %+v", results[0])
	}
	if summary := corpusreplay.Summarize(results); !summary.OK() || summary.SourceEntries != 9 {
		t.Errorf("summary = %+v", summary)
	}
}

// Output must not depend on concurrency or scheduling, so two runs diff
// cleanly.
func TestRunDeterministic(t *testing.T) {
	t.Parallel()
	rows, err := corpusreplay.LoadCatalog(fixtureCatalog(t))
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	var serial, parallel bytes.Buffer
	if err = corpusreplay.WriteJSONL(&serial, corpusreplay.Run(context.Background(), rows,
		corpusreplay.Options{Jobs: 1})); err != nil {
		t.Fatal(err)
	}
	if err = corpusreplay.WriteJSONL(&parallel, corpusreplay.Run(context.Background(), rows,
		corpusreplay.Options{Jobs: 8})); err != nil {
		t.Fatal(err)
	}
	if serial.String() != parallel.String() {
		t.Fatalf("output depends on -jobs:\n%s\n---\n%s", serial.String(), parallel.String())
	}
	if strings.Contains(serial.String(), "milliseconds") {
		t.Fatal("wall time leaked into default output")
	}
}

func TestRunReportsChangedSourceAndDrift(t *testing.T) {
	t.Parallel()
	catalog := fixtureCatalog(t)
	router := filepath.Join(catalog, corpusreplay.RecordingsDir, "sanitized", "router.walk")
	extra := fixtureWalk + ".1.3.6.1.2.1.1.6.0 = STRING: \"lab\"\n"
	if err := os.WriteFile(router, []byte(extra), 0o600); err != nil {
		t.Fatal(err)
	}
	rows, err := corpusreplay.LoadCatalog(catalog)
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	results := corpusreplay.Run(context.Background(), rows, corpusreplay.Options{})

	got := results[2]
	if got.Path != "sanitized/router.walk" || !got.SourceChanged {
		t.Fatalf("changed recording not flagged: %+v", got)
	}
	if len(got.Drift) != 1 || !strings.HasPrefix(got.Drift[0], "source_entries: baseline 4, replay 5") {
		t.Fatalf("drift = %q", got.Drift)
	}
	if corpusreplay.Summarize(results).OK() {
		t.Fatal("a drifted run reported OK")
	}
}

func TestRunMissingRecording(t *testing.T) {
	t.Parallel()
	rows, err := corpusreplay.FileRows([]string{filepath.Join(t.TempDir(), "absent.walk")})
	if err != nil {
		t.Fatal(err)
	}
	results := corpusreplay.Run(context.Background(), rows, corpusreplay.Options{})
	if results[0].Receipt.Status != corpusreplay.StatusSourceError || results[0].Error == "" {
		t.Fatalf("missing recording = %+v", results[0])
	}
}

func TestRunCanceled(t *testing.T) {
	t.Parallel()
	rows, err := corpusreplay.LoadCatalog(fixtureCatalog(t))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, result := range corpusreplay.Run(ctx, rows, corpusreplay.Options{}) {
		if result.Receipt.Status != corpusreplay.StatusCanceled {
			t.Errorf("%s: status %q after cancel", result.Path, result.Receipt.Status)
		}
	}
}

func TestLocateManifestMissing(t *testing.T) {
	t.Parallel()
	for _, dir := range []string{"", t.TempDir()} {
		if _, err := corpusreplay.LocateManifest(dir); !errors.Is(err, corpusreplay.ErrCatalogMissing) {
			t.Errorf("LocateManifest(%q) = %v, want ErrCatalogMissing", dir, err)
		}
	}
}

func TestReadManifestRejects(t *testing.T) {
	t.Parallel()
	cases := map[string]error{
		"effective_path\nx.walk\n":                 corpusreplay.ErrManifestColumn,
		"original_path,effective_path\na,../x\n":   corpusreplay.ErrUnsafePath,
		"original_path,effective_path\na,/etc/x\n": corpusreplay.ErrUnsafePath,
	}
	for body, want := range cases {
		if _, err := corpusreplay.ReadManifest(strings.NewReader(body), t.TempDir()); !errors.Is(err, want) {
			t.Errorf("ReadManifest(%q) = %v, want %v", body, err, want)
		}
	}
}

func TestCompare(t *testing.T) {
	t.Parallel()
	receipt := snmp.ReplayReceipt{
		Status: snmp.ReplayCompleted, SourceEntries: 4, Identity: true, Smoke: false,
		FailureTypes: map[string]int{"kept_value:1.3.6": 1},
	}
	baseline := map[string]string{
		"status": "completed", "source_entries": "4", "source_identity": "True",
		"udp_smoke": "True", "failure_types": `{'kept_value:1.3.6': 1}`, "net_snmp_client": "True",
	}
	if drift := corpusreplay.Compare(baseline, receipt, false); len(drift) != 0 {
		t.Errorf("drift without smoke = %q", drift)
	}
	if drift := corpusreplay.Compare(baseline, receipt, true); len(drift) != 1 ||
		drift[0] != "udp_smoke: baseline True, replay false" {
		t.Errorf("drift with smoke = %q", drift)
	}
	if drift := corpusreplay.Compare(nil, receipt, true); drift != nil {
		t.Errorf("drift without baseline = %q", drift)
	}
}
