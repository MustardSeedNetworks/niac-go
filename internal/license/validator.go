// SPDX-License-Identifier: BUSL-1.1

package license

import (
	"slices"
	"strings"
	"time"
)

/*
NIAC licenses are Ed25519-signed tokens (see signing.go). The previous 16-char
rotor-cipher key format was removed: its generator shipped inside the binary, so
any copy of NIAC could mint a valid key. Tokens are now signed by the keygen
tool's private key and verified here with an embedded public key — offline and
un-forgeable.

Product code 5001 = NIAC Pro (tier 1). Free is the unlicensed tier and needs no
token.
*/

const (
	defaultMaxDevices = 3 // default activations per license

	// productName identifies this binary in a signed payload. A token issued
	// for another product (stem/seed) is rejected even if correctly signed.
	productName = "niac"

	// niacProCode is the only product code NIAC accepts.
	niacProCode = "5001"
)

// licensePublicKeyB64 is the standard-base64 Ed25519 public key that verifies
// production license tokens. The matching private key lives only in the keygen
// tool (msn-internal-tools/keygen) and never ships. See ADR-0005.
//
// Pre-launch signing key — rotate via keygen before GA.
const licensePublicKeyB64 = "O+o8n4qHHp/X//JrRXSdgGSWa2Fqz79OtgUkcylNxZg="

// Tier represents the license tier.
type Tier int

// License tier constants. Numeric values match the tier field embedded in the
// signed payload. NIAC has a single paid tier.
const (
	// TierInvalid represents an invalid or unrecognized license tier.
	TierInvalid Tier = -1
	// TierFree is the unlicensed tier. No token needed; only the basic
	// feature set (up to 10 simulated devices, common protocols) is available.
	TierFree Tier = 0
	// TierPro unlocks the full NIAC feature set (unlimited devices, advanced
	// protocols, error injection, etc.). Wire tier value 1.
	TierPro Tier = 1
)

const (
	errProductCodeMismatch = "Product code mismatch for tier"
	// ErrLicenseInvalid is the generic rejection message. Validation failures
	// deliberately do not distinguish "bad signature" from "tampered payload"
	// to a caller — both mean the same thing: not a genuine license.
	ErrLicenseInvalid = "License key is not valid"
)

// String returns the tier name.
func (t Tier) String() string {
	switch t {
	case TierInvalid:
		return "Invalid"
	case TierFree:
		return "Free"
	case TierPro:
		return "Pro"
	}
	return "Invalid"
}

// Info contains parsed license information.
type Info struct {
	Key         string    `json:"key"`
	Valid       bool      `json:"valid"`
	Tier        Tier      `json:"tier"`
	ProductCode string    `json:"productCode"`
	Serial      string    `json:"serial"`
	Activated   bool      `json:"activated"`
	ActivatedAt time.Time `json:"activatedAt,omitzero"`
	ExpiresAt   time.Time `json:"expiresAt,omitzero"`
	DeviceHash  string    `json:"deviceHash,omitempty"`
	MaxDevices  int       `json:"maxDevices"`
	Features    []string  `json:"features"`
	ErrorMsg    string    `json:"error,omitempty"`
}

// ValidateLicenseKey performs offline validation of a license token against the
// embedded production key. The verifier wraps a 32-byte key, so it is rebuilt
// per call rather than held as a package global; validation is not on a hot
// path (HasFeature reads cached Info, it does not re-validate).
func ValidateLicenseKey(key string) *Info {
	return mustVerifierFromB64(licensePublicKeyB64).Validate(key)
}

// Validate verifies a signed token and maps it to product feature data. The
// signature is checked first (in parseAndVerify); only a genuinely signed,
// current-version payload reaches the product-specific interpretation below.
func (v *Verifier) Validate(key string) *Info {
	info := &Info{
		Key:        strings.TrimSpace(key),
		Valid:      false,
		Tier:       TierInvalid,
		MaxDevices: defaultMaxDevices,
	}

	payload, err := v.parseAndVerify(key)
	if err != nil {
		info.ErrorMsg = ErrLicenseInvalid
		return info
	}

	// A correctly signed token for a different product must not validate here.
	if payload.Product != productName {
		info.ErrorMsg = ErrLicenseInvalid
		return info
	}

	info.ProductCode = payload.Code
	info.Serial = payload.Serial

	// Tier and feature set are authoritative in-binary: the payload's tier is
	// mapped to the feature list defined here, so a signed token can only grant
	// what this build knows about.
	switch payload.Tier {
	case int(TierPro):
		info.Tier = TierPro
		info.Features = proFeatures()
	default:
		info.ErrorMsg = "Invalid license tier"
		return info
	}

	if payload.Code != niacProCode || info.Tier != TierPro {
		info.ErrorMsg = errProductCodeMismatch
		return info
	}

	if payload.MaxDevices > 0 {
		info.MaxDevices = payload.MaxDevices
	}
	if payload.ExpiresAt > 0 {
		info.ExpiresAt = time.Unix(payload.ExpiresAt, 0).UTC()
		if time.Now().After(info.ExpiresAt) {
			info.ErrorMsg = "License has expired"
			return info
		}
	}

	info.Valid = true
	return info
}

// FormatKey returns a signed token for display. Tokens are already
// display-ready (single line, copy/paste); only surrounding whitespace is
// trimmed. Unlike the old 16-char format, tokens must NOT have characters
// stripped — base64url uses '-' and '_'.
func FormatKey(key string) string {
	return strings.TrimSpace(key)
}

// HasFeature checks if the license includes a specific feature.
func (li *Info) HasFeature(feature string) bool {
	return slices.Contains(li.Features, feature)
}

// CanRunPro returns true if the license allows Pro features.
func (li *Info) CanRunPro() bool {
	return li.Valid && li.Tier >= TierPro
}

// proFeatures returns the feature list granted to NIAC Pro (product code 5001).
// Mirrors keygen's productCatalog.
//
// snmpv3 was removed from Pro on 2026-05-27: SNMPv3 is the only safe SNMP
// version (v1/v2c send credentials in cleartext), so gating it would push
// customers toward the insecure variants. BGP + OSPF stay on Pro because
// they're large protocol implementations, not a safety floor.
func proFeatures() []string {
	return []string{
		"unlimited_devices",
		"bgp", "ospf", "netbios", "ftp", "stp",
		"ipv6_advanced", "error_injection", "traffic_shaping",
		"config_templates", "multi_ip", "pcap_ingest", "rest_api",
	}
}
