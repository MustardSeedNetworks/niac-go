package scenario_test

import (
	"encoding/json"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

func TestScenarioPackManifestsMatchComposerOutput(t *testing.T) {
	definitions := scenario.Packs()
	if len(definitions) != 6 {
		t.Fatalf("scenario pack count = %d, want 6", len(definitions))
	}
	seen := make(map[string]bool, len(definitions))
	for _, pack := range definitions {
		if pack.ID == "" || pack.Name == "" || pack.Description == "" || seen[pack.ID] {
			t.Errorf("invalid scenario pack metadata: %+v", pack)
		}
		seen[pack.ID] = true
		result, err := scenario.Generate(pack.Request)
		if err != nil {
			t.Errorf("Generate(%s): %v", pack.ID, err)
			continue
		}
		if result.Manifest != pack.Manifest {
			t.Errorf("%s manifest = %#v", pack.ID, result.Manifest)
		}
		if pack.ManifestVersion != 2 || pack.Version != "1.1.0" {
			t.Errorf("%s versions = %q/%d", pack.ID, pack.Version, pack.ManifestVersion)
		}
	}
}

func TestScenarioPacksCannotGrantPhysicalAttachmentPermission(t *testing.T) {
	for _, pack := range scenario.Packs() {
		data, err := json.Marshal(pack.Request)
		if err != nil {
			t.Fatalf("Marshal(%s): %v", pack.ID, err)
		}
		var fields map[string]json.RawMessage
		if err = json.Unmarshal(data, &fields); err != nil {
			t.Fatalf("Unmarshal(%s): %v", pack.ID, err)
		}
		for _, forbidden := range []string{"attachmentMode", "accessVlan", "interface"} {
			if _, found := fields[forbidden]; found {
				t.Errorf("%s includes forbidden attachment permission %q", pack.ID, forbidden)
			}
		}
	}
}
