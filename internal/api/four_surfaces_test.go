package api

import (
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// TestFourSurfacesAgreeOnConfigDefects is P1b-4's acceptance in one place: the
// authoring surfaces (`niac validate`, library upload) and the runtime
// surfaces (preflight, start) reach the same compiler over the same file and
// therefore report the same codes.
//
// The surfaces are exercised through the one function each of them calls, not
// through four transports, because the transports are covered separately --
// what this pins is that they cannot drift apart again by calling different
// validators.
func TestFourSurfacesAgreeOnConfigDefects(t *testing.T) {
	cfg, err := config.LoadYAMLBytes([]byte(routedScenarioWithFabricDefects))
	if err != nil {
		t.Fatalf("fixture no longer parses: %v", err)
	}

	// Authoring: no binding to speak of.
	authoring := detailCodes(validateScenarioConfig(cfg))

	// Runtime: preflight and start both compile with the deployment binding.
	approved := fabric.Binding{
		Attachment: "tester", Interface: "eth0",
		Mode: fabric.ModeAccess, AccessVLAN: 100, PolicyApproved: true,
	}
	runtime := detailCodes(diagnosticDetails(fabric.Compile(cfg, approved).Diagnostics))

	if len(authoring) == 0 {
		t.Fatal("fixture produced no findings; it no longer proves anything")
	}
	for _, code := range authoring {
		if !slices.Contains(runtime, code) {
			t.Fatalf("authoring reported %q, runtime did not (authoring %v, runtime %v)",
				code, authoring, runtime)
		}
	}
	for _, code := range runtime {
		if !slices.Contains(authoring, code) && !fabric.DiagnosticCode(code).IsBinding() {
			t.Fatalf("runtime reported config code %q, authoring did not (authoring %v, runtime %v)",
				code, authoring, runtime)
		}
	}
}
