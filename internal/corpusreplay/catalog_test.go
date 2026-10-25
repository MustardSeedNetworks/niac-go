package corpusreplay_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/corpusreplay"
)

// CatalogTestEnv opts the qualified-corpus test in. It is separate from
// NIAC_DEMO_CATALOG_DIR so a developer who keeps that set for example syncing
// does not run a multi-minute replay on every `go test ./...`.
const CatalogTestEnv = "NIAC_CORPUS_REPLAY_CATALOG"

// TestQualifiedCatalog replays every recording in a real niac-demo-catalog
// checkout and requires each to reproduce the receipt it qualified with. CI
// has no catalog checkout, so it skips there.
func TestQualifiedCatalog(t *testing.T) {
	catalog := os.Getenv(CatalogTestEnv)
	if catalog == "" || testing.Short() {
		t.Skipf("set %s to a niac-demo-catalog checkout to replay its qualified recordings", CatalogTestEnv)
	}
	rows, err := corpusreplay.LoadCatalog(catalog)
	if err != nil {
		t.Skipf("catalog not usable: %v", err)
	}

	results := corpusreplay.Run(context.Background(), rows, corpusreplay.Options{UDPSmoke: true})
	summary := corpusreplay.Summarize(results)
	var report strings.Builder
	_ = summary.WriteText(&report)
	t.Log(report.String())
	for _, result := range results {
		if len(result.Drift) > 0 || result.Receipt.Status != "completed" {
			t.Errorf("%s: status %s, drift %q %s",
				result.Path, result.Receipt.Status, result.Drift, result.Receipt.Error)
		}
	}
}
