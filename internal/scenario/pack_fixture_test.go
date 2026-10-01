package scenario_test

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// Eleven call sites in this package generated a pack, and nine of them then
// decoded the result to get at its devices -- decoding bytes that Generate had
// already decoded and validated on the way to building the manifest. A pack is
// about 5 MB of YAML and yaml.v3 is roughly an order of magnitude slower under
// the race detector, so paying for that second decode once per test per pack
// ran the package past the 10-minute per-package timeout and make test could
// not complete on macOS. CI never saw it: ci.yml's macOS job runs -race over
// internal/truststore alone, and the full race suite runs on ubuntu (#2167).
//
// Generate now carries the config it built, so these two helpers are all a
// test needs.
//
// Generating was still most of the package's cost, because about thirty tests
// walk every pack and each walk generated all seven again: -race took 509 of
// CI's 600 s, and P-PACK-1's larger packs ran it past the limit (#2349). So a
// test binary generates each distinct request once. The result is shared, so
// it is read-only: a test that needs to change a generated config changes the
// request instead, which is a different key.
var generatedPacks sync.Map

type packGeneration struct {
	once   sync.Once
	result scenario.Result
	err    error
}

// generatedPack generates a pack, failing the test if it cannot.
func generatedPack(t *testing.T, pack scenario.Pack) scenario.Result {
	t.Helper()

	return generatedRequest(t, pack.ID, pack.Request)
}

// generatedRequest generates request once per test binary, failing the test if
// it cannot. label names the request in a failure.
func generatedRequest(t *testing.T, label string, request scenario.Request) scenario.Result {
	t.Helper()

	key, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("key %s: %v", label, err)
	}
	entry, _ := generatedPacks.LoadOrStore(string(key), &packGeneration{})
	generation, _ := entry.(*packGeneration)
	generation.once.Do(func() {
		generation.result, generation.err = scenario.Generate(request)
	})
	if generation.err != nil {
		t.Fatalf("generate %s: %v", label, generation.err)
	}

	return generation.result
}

// packConfig is the generated pack's runtime config.
func packConfig(t *testing.T, pack scenario.Pack) *config.Config {
	t.Helper()

	return generatedPack(t, pack).Config
}
