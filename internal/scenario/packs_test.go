package scenario_test

import (
	"encoding/json"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

func TestScenarioPackManifestsMatchComposerOutput(t *testing.T) {
	definitions := scenario.Packs()
	if len(definitions) != 7 {
		t.Fatalf("scenario pack count = %d, want 7", len(definitions))
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
		if pack.ManifestVersion != 3 || pack.Version != "1.3.0" {
			t.Errorf("%s versions = %q/%d", pack.ID, pack.Version, pack.ManifestVersion)
		}
	}
}

func TestPresentationScenarioPacksFitLinkLiveMapBudget(t *testing.T) {
	const maximumPresentationDevices = 160

	for _, pack := range scenario.Packs() {
		if pack.MapPurpose == scenario.MapPurposeStress {
			continue
		}
		if pack.MapPurpose != scenario.MapPurposePresentation {
			t.Errorf("%s map purpose = %q", pack.ID, pack.MapPurpose)
		}
		if pack.Manifest.DeviceCount > maximumPresentationDevices {
			t.Errorf("%s devices = %d, presentation budget = %d", pack.ID,
				pack.Manifest.DeviceCount, maximumPresentationDevices)
		}
	}
}

func TestEnterpriseScalePackIsNotPresentedAsAMapDemo(t *testing.T) {
	for _, pack := range scenario.Packs() {
		if pack.ID == "enterprise-scale" && pack.MapPurpose != scenario.MapPurposeStress {
			t.Errorf("enterprise-scale map purpose = %q, want stress", pack.MapPurpose)
		}
	}
}

func TestVerticalDemoPacksAreSingleSite(t *testing.T) {
	verticals := map[string]bool{"hospital": true, "warehouse": true, "manufacturing": true}
	for _, pack := range scenario.Packs() {
		if verticals[pack.ID] && len(pack.Request.Sites) != 1 {
			t.Errorf("%s sites = %d, want one", pack.ID, len(pack.Request.Sites))
		}
	}
}

// Every presentation pack keeps the full layered spine, radios and controllers
// included. Service provider used to be the one pack generating neither, which
// left its map missing a tier every real POP has.
func TestEveryPresentationPackKeepsTheWirelessTier(t *testing.T) {
	for _, pack := range scenario.Packs() {
		if pack.MapPurpose != scenario.MapPurposePresentation {
			continue
		}
		if pack.Request.Counts.AccessPointsPerAccess < 1 ||
			pack.Request.Counts.WirelessControllers < 1 {
			t.Errorf("%s generates %d radios per access switch and %d controllers",
				pack.ID, pack.Request.Counts.AccessPointsPerAccess,
				pack.Request.Counts.WirelessControllers)
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
