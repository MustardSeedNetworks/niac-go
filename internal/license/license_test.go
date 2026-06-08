// SPDX-License-Identifier: BUSL-1.1

package license_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/license"
)

// prodNiacProVector is a production-signed NIAC Pro token (serial NIACPRO),
// produced by the canonical keygen tool against the embedded production public
// key. It MUST validate via the default verifier — this pins the cross-tool
// signing contract the way stem/seed pin theirs. If the production key rotates,
// regenerate this vector and the embedded key together.
const prodNiacProVector = "MSN1.eyJ2IjoxLCJwcm9kdWN0IjoibmlhYyIsImNvZGUiOiI1MDAxIiwic2VyaWFsIjoiTklBQ1BSTyIsInRpZXIiOjEsIm1heERldmljZXMiOjMsImlhdCI6MTc4MDg3NjgwMH0.jfadRV5JJGq951LSF881Ue0IE3nRLDAjiPvSy7V9hR4dR84FWxM7ttP5On1fZWohKEVH8UZz3z64PbfjLL2wDQ"

// signToken builds an MSN1 token signing the given fields with priv. It mirrors
// the keygen/verifier wire format so tests can mint tokens against an ephemeral
// key without the production private key (which never ships).
func signToken(
	t *testing.T,
	priv ed25519.PrivateKey,
	product, code, serial string,
	tier int,
	exp int64,
) string {
	t.Helper()
	payload := map[string]any{
		"v":          1,
		"product":    product,
		"code":       code,
		"serial":     serial,
		"tier":       tier,
		"maxDevices": 3,
		"iat":        time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC).Unix(),
	}
	if exp > 0 {
		payload["exp"] = exp
	}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	sig := ed25519.Sign(priv, b)
	return "MSN1." + base64.RawURLEncoding.EncodeToString(b) +
		"." + base64.RawURLEncoding.EncodeToString(sig)
}

func testKeyPair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate test key: %v", err)
	}
	return pub, priv
}

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

// TestSignedTokenRoundTrip mints a token with an ephemeral key and verifies it
// through a Verifier built on the matching public key.
func TestSignedTokenRoundTrip(t *testing.T) {
	t.Parallel()
	pub, priv := testKeyPair(t)
	v := license.NewVerifier(pub)

	token := signToken(t, priv, "niac", "5001", "ABCDEFG", int(license.TierPro), 0)
	info := v.Validate(token)
	if !info.Valid {
		t.Fatalf("Validate: not valid (err=%q)", info.ErrorMsg)
	}
	if info.ProductCode != "5001" {
		t.Errorf("ProductCode = %q, want 5001", info.ProductCode)
	}
	if info.Tier != license.TierPro {
		t.Errorf("Tier = %v, want Pro", info.Tier)
	}
	if info.Serial != "ABCDEFG" {
		t.Errorf("Serial = %q, want ABCDEFG", info.Serial)
	}
	if !info.HasFeature("bgp") || info.HasFeature("snmpv3") {
		t.Errorf("unexpected feature set: %v", info.Features)
	}
}

// TestForgeryRejected is the core security property: a token signed by an
// attacker's own key is rejected by the production verifier. Under the old
// scheme the generator shipped in-binary, so this attack succeeded.
func TestForgeryRejected(t *testing.T) {
	t.Parallel()
	_, attacker := testKeyPair(t)
	forged := signToken(t, attacker, "niac", "5001", "FORGEDXX", int(license.TierPro), 0)

	info := license.ValidateLicenseKey(forged)
	if info.Valid {
		t.Fatal("forged token (attacker key) validated against production key")
	}
}

// TestTamperRejected mutates a single payload byte of an otherwise valid token
// and confirms the signature check rejects it.
func TestTamperRejected(t *testing.T) {
	t.Parallel()
	pub, priv := testKeyPair(t)
	v := license.NewVerifier(pub)
	token := signToken(t, priv, "niac", "5001", "ABCDEFG", int(license.TierPro), 0)

	parts := strings.Split(token, ".")
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	raw[len(raw)-2] ^= 0x01 // flip a bit in the payload, keep the old signature
	tampered := parts[0] + "." + base64.RawURLEncoding.EncodeToString(raw) + "." + parts[2]

	if v.Validate(tampered).Valid {
		t.Fatal("tampered payload validated")
	}
}

// TestWrongProductRejected confirms a correctly signed token issued for another
// product does not validate as NIAC.
func TestWrongProductRejected(t *testing.T) {
	t.Parallel()
	pub, priv := testKeyPair(t)
	v := license.NewVerifier(pub)
	token := signToken(t, priv, "stem", "2001", "ABCDEFG", int(license.TierPro), 0)
	if v.Validate(token).Valid {
		t.Fatal("stem-product token validated as NIAC")
	}
}

// TestExpiredRejected confirms expiry is enforced even on a validly signed token.
func TestExpiredRejected(t *testing.T) {
	t.Parallel()
	pub, priv := testKeyPair(t)
	v := license.NewVerifier(pub)
	past := time.Now().Add(-time.Hour).Unix()
	token := signToken(t, priv, "niac", "5001", "ABCDEFG", int(license.TierPro), past)
	info := v.Validate(token)
	if info.Valid {
		t.Fatal("expired token validated")
	}
}

// TestInvalidTierRejected confirms a validly signed token carrying an
// unrecognized tier (here Free/0, which needs no token) is rejected.
func TestInvalidTierRejected(t *testing.T) {
	t.Parallel()
	pub, priv := testKeyPair(t)
	v := license.NewVerifier(pub)
	token := signToken(t, priv, "niac", "5001", "ABCDEFG", int(license.TierFree), 0)
	if v.Validate(token).Valid {
		t.Fatal("token with Free tier validated")
	}
}

func TestValidateRejectsBadInputs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		key  string
	}{
		{"empty", ""},
		{"garbage", "not-a-token"},
		{"wrong scheme", "MSN9.AAAA.BBBB"},
		{"two parts", "MSN1.AAAA"},
		{"bad base64", "MSN1.!!!.???"},
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

// TestKeygenContract pins the cross-tool signing contract: the production-signed
// vector validates against the embedded production key and decodes to the
// expected NIAC Pro metadata and feature set.
func TestKeygenContract(t *testing.T) {
	t.Parallel()
	info := license.ValidateLicenseKey(prodNiacProVector)
	if !info.Valid {
		t.Fatalf("production vector did not validate (err=%q)", info.ErrorMsg)
	}
	if info.Tier != license.TierPro {
		t.Errorf("Tier = %v, want Pro", info.Tier)
	}
	if info.ProductCode != "5001" {
		t.Errorf("ProductCode = %q, want 5001", info.ProductCode)
	}
	if info.Serial != "NIACPRO" {
		t.Errorf("Serial = %q, want NIACPRO", info.Serial)
	}
	want := []string{
		"unlimited_devices",
		"bgp", "ospf", "netbios", "ftp", "stp",
		"ipv6_advanced", "error_injection", "traffic_shaping",
		"config_templates", "multi_ip", "pcap_ingest", "rest_api",
	}
	if len(info.Features) != len(want) {
		t.Errorf(
			"Features count = %d, want %d (got %v)",
			len(info.Features),
			len(want),
			info.Features,
		)
	}
	for _, f := range want {
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

	res := mgr.Activate(prodNiacProVector)
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

// TestManagerConcurrentReadsAndWrites exercises the RWMutex so `go test -race`
// fails loudly if the locking ever regresses.
func TestManagerConcurrentReadsAndWrites(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	mgr, err := license.NewManagerWithDir(tmp)
	if err != nil {
		t.Fatalf("NewManagerWithDir: %v", err)
	}

	done := make(chan struct{})
	go func() {
		for range 50 {
			mgr.Activate(prodNiacProVector)
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
