// SPDX-License-Identifier: BUSL-1.1

package license_test

import (
	"slices"
	"testing"

	foundation "github.com/MustardSeedNetworks/foundation/pkg/license"

	"github.com/MustardSeedNetworks/niac-go/internal/license"
)

// prodNiacProVector is a production-signed NIAC Pro token (serial NIACPRO),
// produced by the canonical keygen tool against the embedded production public
// key. It MUST activate through NIAC's product policy — this pins the
// cross-tool signing contract (product name "niac", code 5001, Pro feature set)
// the way stem/seed pin theirs. If the production key rotates, regenerate this
// vector and the embedded key together. Generic crypto properties (forgery,
// tampering, wrong-product, expiry, bad input) are covered in foundation's
// pkg/license tests; this file only exercises NIAC's product-specific wiring.
const prodNiacProVector = "MSN1.eyJ2IjoxLCJwcm9kdWN0IjoibmlhYyIsImNvZGUiOiI1MDAxIiwic2VyaWFsIjoiTklBQ1BSTyIsInRpZXIiOjEsIm1heERldmljZXMiOjMsImlhdCI6MTc4MDg3NjgwMH0.jfadRV5JJGq951LSF881Ue0IE3nRLDAjiPvSy7V9hR4dR84FWxM7ttP5On1fZWohKEVH8UZz3z64PbfjLL2wDQ"

func TestTierString(t *testing.T) {
	t.Parallel()
	cases := []struct {
		tier license.Tier
		want string
	}{
		{license.TierFree, "Free"},
		{license.TierPro, "Pro"},
		{license.TierInvalid, "Invalid"},
	}
	for _, c := range cases {
		if got := c.tier.String(); got != c.want {
			t.Errorf("Tier(%d).String() = %q, want %q", c.tier, got, c.want)
		}
	}
}

// wantProFeatures is the feature set NIAC Pro grants, in catalog order.
func wantProFeatures() []string {
	return []string{
		"unlimited_devices",
		"routed_labs", "netbios", "ftp", "stp",
		"ipv6_advanced", "error_injection",
		"config_templates", "multi_ip", "pcap_ingest", "rest_api",
	}
}

// TestKeygenContract pins the cross-tool signing contract end-to-end: the
// production-signed vector activates through NIAC's policy and yields the
// expected Pro tier and feature set. This catches a wrong product code, salt,
// or embedded key in NIAC's policy wiring.
func TestKeygenContract(t *testing.T) {
	t.Parallel()
	mgr, err := license.NewManagerWithDir(t.TempDir())
	if err != nil {
		t.Fatalf("NewManagerWithDir: %v", err)
	}
	res := mgr.Activate(prodNiacProVector)
	if !res.Success {
		t.Fatalf("production vector did not activate: %s", res.Message)
	}
	if res.Tier != int(license.TierPro) {
		t.Errorf("Tier = %d, want Pro (%d)", res.Tier, license.TierPro)
	}
	st := mgr.GetState()
	if st.Tier != int(license.TierPro) {
		t.Errorf("state Tier = %d, want Pro (%d)", st.Tier, license.TierPro)
	}
	for _, f := range wantProFeatures() {
		if !slices.Contains(st.Features, f) {
			t.Errorf("missing feature %q (got %v)", f, st.Features)
		}
	}
	if slices.Contains(st.Features, "snmpv3") {
		t.Errorf("snmpv3 unexpectedly granted: %v", st.Features)
	}
}

// TestActivationLifecycle exercises NIAC's trial → activate → reload →
// deactivate path through the foundation manager, asserting the NIAC-specific
// tier at each step.
func TestActivationLifecycle(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	mgr, err := license.NewManagerWithDir(tmp)
	if err != nil {
		t.Fatalf("NewManagerWithDir: %v", err)
	}

	if mgr.IsActivated() {
		t.Error("expected !IsActivated on fresh manager")
	}

	trial := mgr.StartTrial()
	if !trial.Success || !trial.IsTrialMode || trial.Tier != int(license.TierPro) {
		t.Errorf("StartTrial unexpected: %+v", trial)
	}
	if !mgr.IsActivated() || !mgr.IsTrialValid() {
		t.Error("expected trial to be active")
	}

	res := mgr.Activate(prodNiacProVector)
	if !res.Success || res.Tier != int(license.TierPro) {
		t.Errorf("Activate unexpected: %+v", res)
	}
	if mgr.GetState().IsTrialMode {
		t.Error("expected non-trial state after Activate")
	}

	mgr2, err := license.NewManagerWithDir(tmp)
	if err != nil {
		t.Fatalf("reload NewManagerWithDir: %v", err)
	}
	if !mgr2.IsActivated() {
		t.Error("expected reloaded state to be activated")
	}
	if mgr2.GetState().Tier != int(license.TierPro) {
		t.Errorf("reloaded tier = %d, want Pro (%d)", mgr2.GetState().Tier, license.TierPro)
	}

	if deactErr := mgr2.Deactivate(); deactErr != nil {
		t.Fatalf("Deactivate: %v", deactErr)
	}
	if mgr2.IsActivated() {
		t.Error("expected !IsActivated after Deactivate")
	}
}

func TestPersistedProStateUsesCurrentFeaturePolicy(t *testing.T) {
	tmp := t.TempDir()
	oldPolicy := license.Policy()
	_, proCode, _ := oldPolicy.FeaturesForTier(int(license.TierPro))
	oldPolicy.FeaturesForTier = func(wireTier int) ([]string, string, bool) {
		if wireTier != int(license.TierPro) {
			return nil, "", false
		}
		return []string{"unlimited_devices"}, proCode, true
	}
	oldManager, err := foundation.NewManagerWithDir(
		foundation.NewProductionVerifier(oldPolicy), oldPolicy, tmp,
	)
	if err != nil {
		t.Fatal(err)
	}
	if result := oldManager.StartTrial(); !result.Success {
		t.Fatalf("StartTrial() = %#v", result)
	}

	manager, err := license.NewManagerWithDir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if !manager.HasFeature("routed_labs") {
		t.Fatal("persisted Pro state did not receive routed_labs from current policy")
	}
	if !slices.Contains(manager.GetState().Features, "routed_labs") {
		t.Fatalf("state features = %v", manager.GetState().Features)
	}
}

// TestFeaturesForTier verifies NIAC's product policy directly: only Pro carries
// a token and it maps to the Pro feature set and code 5001; Free and any
// unrecognized tier are rejected so a signed token can't grant more than this
// build knows about.
func TestFeaturesForTier(t *testing.T) {
	t.Parallel()
	p := license.Policy()

	features, code, ok := p.FeaturesForTier(int(license.TierPro))
	if !ok {
		t.Fatal("Pro tier not recognized")
	}
	if code != "5001" {
		t.Errorf("Pro expected code = %q, want 5001", code)
	}
	if !slices.Equal(features, wantProFeatures()) {
		t.Errorf("Pro features = %v, want %v", features, wantProFeatures())
	}

	for _, tier := range []int{int(license.TierFree), int(license.TierInvalid), 99} {
		if _, _, recognized := p.FeaturesForTier(tier); recognized {
			t.Errorf("tier %d unexpectedly recognized", tier)
		}
	}
}
