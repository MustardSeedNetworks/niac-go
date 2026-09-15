package scenario_test

import (
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

// generatedPack generates a pack, failing the test if it cannot.
func generatedPack(t *testing.T, pack scenario.Pack) scenario.Result {
	t.Helper()

	result, err := scenario.Generate(pack.Request)
	if err != nil {
		t.Fatalf("generate %s: %v", pack.ID, err)
	}

	return result
}

// packConfig is the generated pack's runtime config.
func packConfig(t *testing.T, pack scenario.Pack) *config.Config {
	t.Helper()

	return generatedPack(t, pack).Config
}
