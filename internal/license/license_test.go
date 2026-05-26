// SPDX-License-Identifier: BUSL-1.1

package license_test

import (
	"testing"

	"github.com/krisarmstrong/niac-go/internal/license"
)

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

func TestGenerateAndValidateRoundTrip(t *testing.T) {
	t.Parallel()
	key, err := license.GenerateLicenseKey("5001", "ABCDEFG", license.TierPro)
	if err != nil {
		t.Fatalf("GenerateLicenseKey: %v", err)
	}
	info := license.ValidateLicenseKey(key)
	if !info.Valid {
		t.Fatalf("ValidateLicenseKey(%q): not valid (err=%q)", key, info.ErrorMsg)
	}
	if info.ProductCode != "5001" {
		t.Errorf("ProductCode = %q, want 5001", info.ProductCode)
	}
	if info.Tier != license.TierPro {
		t.Errorf("Tier = %v, want Pro", info.Tier)
	}
}

func TestValidateRejectsBadInputs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		key  string
	}{
		{"empty", ""},
		{"too short", "AAAA-BBBB"},
		{"too long", "AAAA-BBBB-CCCC-DDDD-EEEE"},
		{"non-alphanumeric", "AAAA-BBBB-CCCC-D!DD"},
		{"bad checksum", "AAAA-BBBB-CCCC-DDDD"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			info := license.ValidateLicenseKey(c.key)
			if info.Valid {
				t.Errorf("expected invalid, got valid (key=%q)", c.key)
			}
			if info.ErrorMsg == "" {
				t.Errorf("expected non-empty ErrorMsg")
			}
		})
	}
}

// TestKeygenContract pins the cross-tool cipher contract. The key
// below was produced by the canonical keygen tool
// (msn-internal-tools/keygen) and MUST validate identically in every
// product's license package. Stem locked vectors in stem PR #266 and
// seed locked theirs in seed PR #1095 — this is NIAC's contribution
// to the same shared spec.
//
// Anchored to keygen v2.0.0 (2026-05-21).
func TestKeygenContract(t *testing.T) {
	t.Parallel()
	vector := keygenVector{
		name:    "niac-pro / serial NIACPRO",
		key:     "WQ57-20TQ-NGVZ-P2YC",
		tier:    license.TierPro,
		product: "5001",
		serial:  "NIACPRO",
		features: []string{
			"unlimited_devices",
			"bgp", "ospf", "snmpv3", "netbios", "ftp", "stp",
			"ipv6_advanced", "error_injection", "traffic_shaping",
			"config_templates", "multi_ip", "pcap_ingest", "rest_api",
		},
	}
	assertKeygenVector(t, vector)
}

type keygenVector struct {
	name     string
	key      string
	tier     license.Tier
	product  string
	serial   string
	features []string
}

func assertKeygenVector(t *testing.T, v keygenVector) {
	t.Helper()
	info := license.ValidateLicenseKey(v.key)
	if !info.Valid {
		t.Fatalf("Valid = false, want true (err=%q)", info.ErrorMsg)
	}
	if info.Tier != v.tier {
		t.Errorf("Tier = %v, want %v", info.Tier, v.tier)
	}
	if info.ProductCode != v.product {
		t.Errorf("ProductCode = %q, want %q", info.ProductCode, v.product)
	}
	if info.Serial != v.serial {
		t.Errorf("Serial = %q, want %q", info.Serial, v.serial)
	}
	if len(info.Features) != len(v.features) {
		t.Errorf("Features count = %d, want %d (got %v)",
			len(info.Features), len(v.features), info.Features)
	}
	for _, f := range v.features {
		if !info.HasFeature(f) {
			t.Errorf("missing feature %q (got %v)", f, info.Features)
		}
	}
}

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
	if !trial.Success || !trial.IsTrialMode || trial.Tier != license.TierPro {
		t.Errorf("StartTrial unexpected: %+v", trial)
	}
	if !mgr.IsActivated() || !mgr.IsTrialValid() {
		t.Error("expected trial to be active")
	}

	key, _ := license.GenerateLicenseKey("5001", "ABCDEFG", license.TierPro)
	res := mgr.Activate(key)
	if !res.Success || res.Tier != license.TierPro {
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
	if mgr2.GetState().Tier != license.TierPro {
		t.Errorf("reloaded tier = %v, want Pro", mgr2.GetState().Tier)
	}

	if deactErr := mgr2.Deactivate(); deactErr != nil {
		t.Fatalf("Deactivate: %v", deactErr)
	}
	if mgr2.IsActivated() {
		t.Error("expected !IsActivated after Deactivate")
	}
}

// TestManagerConcurrentReadsAndWrites exercises the RWMutex so `go test
// -race` fails loudly if the locking ever regresses.
func TestManagerConcurrentReadsAndWrites(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	mgr, err := license.NewManagerWithDir(tmp)
	if err != nil {
		t.Fatalf("NewManagerWithDir: %v", err)
	}

	key, _ := license.GenerateLicenseKey("5001", "ABCDEFG", license.TierPro)

	done := make(chan struct{})
	go func() {
		for range 50 {
			mgr.Activate(key)
			_ = mgr.Deactivate()
		}
		close(done)
	}()

	for range 8 {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					_ = mgr.IsActivated()
					_ = mgr.GetState()
					_ = mgr.IsTrialValid()
					_ = mgr.TrialDaysRemaining()
					_ = mgr.NeedsCheckIn()
				}
			}
		}()
	}

	<-done
}
