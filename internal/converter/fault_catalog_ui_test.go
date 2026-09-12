package converter

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

var uiFaultLiteral = regexp.MustCompile(`'([a-z_]+)',`)

// TestWizardFaultTypesCoverTheRuntimeCatalog holds the browser's copy of the
// fault catalog to the runtime's. The wizard restates the set a third time, in
// TypeScript, so that the fault composer can offer the right value control per
// type -- and it had drifted the same way the YAML enum had: no `poe_loss`, so
// the fault a PoE demo needs could not be authored from the UI either.
func TestWizardFaultTypesCoverTheRuntimeCatalog(t *testing.T) {
	path := filepath.Join("..", "..", "ui", "src", "api", "behavior-fault-types.ts")
	source, err := os.ReadFile(path) // #nosec G304 -- fixed in-repo UI path
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	var offered []string
	for _, match := range uiFaultLiteral.FindAllStringSubmatch(string(source), -1) {
		offered = append(offered, match[1])
	}
	for _, faultType := range devicestate.AuthorableFaultTypes() {
		if !slices.Contains(offered, faultType) {
			t.Errorf("%q is a runtime fault the wizard cannot author", faultType)
		}
	}
}
