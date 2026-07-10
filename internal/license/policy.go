// SPDX-License-Identifier: BUSL-1.1

// Package license is NIAC's thin product layer over the fleet-shared
// github.com/MustardSeedNetworks/foundation/pkg/license core. The generic
// signing format, device fingerprint, activation state machine and encrypted
// on-disk state all live in foundation; this package supplies only NIAC's
// product policy (tier vocabulary, product code, feature catalog, trial terms)
// and re-exports the manager surface its callers use.
package license

import fnd "github.com/MustardSeedNetworks/foundation/pkg/license"

// NIAC licenses are Ed25519-signed tokens verified offline against an embedded
// public key (the matching private key lives only in the keygen tool). Product
// code 5001 = NIAC Pro (tier 1); Free is the unlicensed tier and needs no token.
const (
	// productName identifies this binary in a signed payload. A token issued
	// for another product (stem/seed) is rejected even if correctly signed.
	productName = "niac"

	// niacProCode is the only product code NIAC accepts.
	niacProCode = "5001"

	// defaultMaxDevices is the activation cap assumed until a validated token
	// specifies its own MaxDevices.
	defaultMaxDevices = 3

	// TrialDays is the length of the no-token trial. Exported for the CLI's
	// "days left of N" display.
	TrialDays = 14

	// encryptionSalt is mixed into the on-disk state key so a license file
	// from another product can't be reused by renaming it into NIAC's config
	// directory.
	encryptionSalt = "MSN-NIAC-SIM-2026-LICENSE"

	// configSubdir is the directory under ~/.config where activation state is
	// persisted; licenseFileName is the file within it.
	configSubdir    = "niac"
	licenseFileName = ".license"
)

// Tier represents the license tier. Numeric values match the wire-tier field
// in the signed payload. NIAC has a single paid tier.
type Tier int

// License tier constants.
const (
	// TierInvalid represents an invalid or unrecognized license tier.
	TierInvalid Tier = -1
	// TierFree is the unlicensed tier. No token needed; only the basic feature
	// set (up to 10 simulated devices, common protocols) is available.
	TierFree Tier = 0
	// TierPro unlocks the full NIAC feature set (unlimited devices, advanced
	// protocols, error injection, etc.). Wire tier value 1.
	TierPro Tier = 1
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

// featuresForTier maps a signed wire-tier to the features NIAC grants and the
// product code expected for that tier. Only Pro carries a token; Free (and any
// unrecognized tier) is rejected so a signed token can only grant what this
// build knows about. Passed to foundation as ProductPolicy.FeaturesForTier.
func featuresForTier(wireTier int) ([]string, string, bool) {
	if Tier(wireTier) == TierPro {
		return proFeatures(), niacProCode, true
	}
	return nil, "", false
}

// Policy is NIAC's product configuration for the foundation license core.
func Policy() fnd.ProductPolicy {
	return fnd.ProductPolicy{
		ProductName:       productName,
		FeaturesForTier:   featuresForTier,
		EncryptionSalt:    encryptionSalt,
		ConfigSubdir:      configSubdir,
		LicenseFileName:   licenseFileName,
		DefaultMaxDevices: defaultMaxDevices,
		TrialDays:         TrialDays,
		TrialTier:         int(TierPro),
	}
}
