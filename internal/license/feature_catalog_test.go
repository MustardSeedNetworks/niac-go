// SPDX-License-Identifier: BUSL-1.1

package license

import "testing"

// TestFeatureCatalogCoversProFeatures ensures every feature ID proFeatures()
// can grant has a non-empty label + description in featureMeta. This is an
// internal-package test (not license_test) specifically so it can reach
// proFeatures() directly and catch a newly added Pro feature that shipped
// without display copy.
func TestFeatureCatalogCoversProFeatures(t *testing.T) {
	ids := proFeatures()
	if len(ids) == 0 {
		t.Fatal("proFeatures() returned no IDs; test can't assert coverage")
	}

	for _, id := range ids {
		meta, ok := featureMeta[id]
		if !ok {
			t.Errorf("feature %q has no entry in featureMeta", id)
			continue
		}
		if meta.Label == "" {
			t.Errorf("feature %q has an empty Label", id)
		}
		if meta.Description == "" {
			t.Errorf("feature %q has an empty Description", id)
		}
	}
}

// TestFeatureCatalogNoOrphans ensures featureMeta doesn't carry stale
// entries for features proFeatures() no longer grants — an orphaned entry
// would silently mislead anyone reading the registry as the source of
// truth for what Pro currently unlocks.
func TestFeatureCatalogNoOrphans(t *testing.T) {
	granted := make(map[string]bool, len(proFeatures()))
	for _, id := range proFeatures() {
		granted[id] = true
	}

	for id := range featureMeta {
		if !granted[id] {
			t.Errorf("featureMeta has an orphaned entry %q not returned by proFeatures()", id)
		}
	}
}

// TestFeatureCatalogOrderMatchesProFeatures ensures FeatureCatalog() returns
// entries in proFeatures() order with IDs populated, so API consumers get a
// stable, predictable ordering.
func TestFeatureCatalogOrderMatchesProFeatures(t *testing.T) {
	ids := proFeatures()
	catalog := FeatureCatalog()

	if len(catalog) != len(ids) {
		t.Fatalf("FeatureCatalog() returned %d entries, want %d", len(catalog), len(ids))
	}
	for i, id := range ids {
		if catalog[i].ID != id {
			t.Errorf("FeatureCatalog()[%d].ID = %q, want %q", i, catalog[i].ID, id)
		}
	}
}
