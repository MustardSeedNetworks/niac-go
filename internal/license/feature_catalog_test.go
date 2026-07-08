// SPDX-License-Identifier: BUSL-1.1

package license_test

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/license"
)

// TestFeatureCatalogCoversProFeatures ensures every feature the Pro catalog
// exposes ships with non-empty display copy, so a newly added Pro feature
// can't reach the /license UI as a bare ID. FeatureCatalog() enumerates
// exactly the granted Pro features; a feature with no registry entry surfaces
// here as an empty Label/Description and fails this test.
func TestFeatureCatalogCoversProFeatures(t *testing.T) {
	catalog := license.FeatureCatalog()
	if len(catalog) == 0 {
		t.Fatal("FeatureCatalog() returned no entries; test can't assert coverage")
	}

	for _, f := range catalog {
		if f.ID == "" {
			t.Error("FeatureCatalog() entry has an empty ID")
		}
		if f.Label == "" {
			t.Errorf("feature %q has an empty Label", f.ID)
		}
		if f.Description == "" {
			t.Errorf("feature %q has an empty Description", f.ID)
		}
	}
}

// TestFeatureCatalogIsStable ensures FeatureCatalog() returns a deterministic,
// duplicate-free ordering so API consumers get a predictable list every call.
func TestFeatureCatalogIsStable(t *testing.T) {
	first := license.FeatureCatalog()
	second := license.FeatureCatalog()

	if len(first) != len(second) {
		t.Fatalf("FeatureCatalog() length not stable: %d vs %d", len(first), len(second))
	}

	seen := make(map[string]bool, len(first))
	for i := range first {
		if first[i] != second[i] {
			t.Errorf("FeatureCatalog()[%d] not stable: %+v vs %+v", i, first[i], second[i])
		}
		if seen[first[i].ID] {
			t.Errorf("FeatureCatalog() has a duplicate ID %q", first[i].ID)
		}
		seen[first[i].ID] = true
	}
}
