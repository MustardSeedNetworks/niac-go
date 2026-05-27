// SPDX-License-Identifier: BUSL-1.1

package license

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

/*
NIAC License Key Format (16 characters, identical to the format used
by Seed and Stem — keys are produced by the canonical keygen tool
and the rotor cipher is byte-compatible across products).

+------+--------+-------+------+----------+
| CC   | PPPP   |SSSSSSS| T    | XX       |
|Check |Product |Serial |Tier  | Checksum |
+------+--------+-------+------+----------+

Positions:
  0-1:  Checksum prefix (encoded validation).
  2-5:  Product code (5001 = NIAC Pro).
  6-12: Serial number (unique per license).
  13:   Tier (1 = Pro; NIAC has one paid tier).
  14-15: Checksum suffix.

Free is the unlicensed tier — no key required. Pro requires a valid
5001 key.
*/

const (
	keyLength         = 16
	productCodeLength = 4
	serialLength      = 7
	checksumLength    = 2
	cipherStartPos    = 7
	defaultMaxDevices = 3
)

// Tier represents the license tier.
type Tier int

// License tier constants. Numeric values match the tier digit embedded
// in the license key (position 13 of the encoded key). NIAC has a
// single paid tier; the wire digit '1' is consistent with keygen's
// productCatalog entry for 5001.
const (
	// TierInvalid represents an invalid or unrecognized license tier.
	TierInvalid Tier = -1
	// TierFree is the unlicensed tier. No key needed; only the basic
	// feature set (up to 10 simulated devices, common protocols) is
	// available.
	TierFree Tier = 0
	// TierPro unlocks the full NIAC feature set (unlimited devices,
	// advanced protocols, error injection, etc.). Wire tier digit 1.
	TierPro Tier = 1
)

const (
	errProductCodeMismatch = "Product code mismatch for tier"
	// ErrLicenseKeyLength is the key-length validation error string.
	ErrLicenseKeyLength = "License key must be 16 characters"
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

// ValidateLicenseKey performs offline validation of a license key.
func ValidateLicenseKey(key string) *Info {
	info := &Info{
		Key:         key,
		Valid:       false,
		Tier:        TierInvalid,
		ProductCode: "",
		Serial:      "",
		MaxDevices:  defaultMaxDevices,
	}

	normalized := normalizeKey(key)
	if len(normalized) != keyLength {
		info.ErrorMsg = ErrLicenseKeyLength
		return info
	}

	if !validKeyChars(normalized) {
		info.ErrorMsg = "License key contains invalid characters"
		return info
	}

	cipher := NewRotorCipher(cipherStartPos)
	decoded := cipher.DecodeString(normalized)

	if !validateKeyChecksum(decoded) {
		info.ErrorMsg = "License key failed checksum validation"
		return info
	}

	info.ProductCode = decoded[2:6]
	info.Serial = decoded[6:13]
	tierChar := decoded[13]

	switch tierChar {
	case '1':
		info.Tier = TierPro
		info.Features = proFeatures()
	default:
		info.ErrorMsg = "Invalid license tier"
		return info
	}

	switch info.ProductCode {
	case "5001":
		if info.Tier != TierPro {
			info.ErrorMsg = errProductCodeMismatch
			return info
		}
	default:
		info.ErrorMsg = "Unknown product code"
		return info
	}

	info.Valid = true
	return info
}

var keyCharRE = regexp.MustCompile(`^[0-9A-Z]+$`)

func validKeyChars(s string) bool {
	return keyCharRE.MatchString(s)
}

func validateKeyChecksum(key string) bool {
	payload := key[2:14]
	expected := CalculateChecksum(payload)
	prefixMatch := key[0:2] == expected
	suffixMatch := key[14:16] == expected
	return prefixMatch && suffixMatch
}

// GenerateLicenseKey creates a new license key (for testing — production
// keys are issued by the keygen tool).
func GenerateLicenseKey(productCode string, serial string, tier Tier) (string, error) {
	if len(productCode) != productCodeLength {
		return "", errors.New("product code must be 4 characters")
	}
	if len(serial) != serialLength {
		return "", errors.New("serial must be 7 characters")
	}
	if tier != TierPro {
		return "", errors.New("invalid tier (NIAC has only TierPro)")
	}

	payload := productCode + serial + fmt.Sprintf("%d", tier)
	checksum := CalculateChecksum(payload)
	fullKey := checksum[0:checksumLength] + payload + checksum
	cipher := NewRotorCipher(cipherStartPos)
	encoded := cipher.EncodeString(fullKey)
	return strings.ToUpper(encoded), nil
}

func normalizeKey(key string) string {
	key = strings.ReplaceAll(key, "-", "")
	key = strings.ReplaceAll(key, " ", "")
	key = strings.ReplaceAll(key, ".", "")
	return strings.ToUpper(key)
}

// FormatKey formats a license key for display (adds dashes).
func FormatKey(key string) string {
	key = normalizeKey(key)
	if len(key) != keyLength {
		return key
	}
	return key[0:4] + "-" + key[4:8] + "-" + key[8:12] + "-" + key[12:16]
}

// HasFeature checks if the license includes a specific feature.
func (li *Info) HasFeature(feature string) bool {
	return slices.Contains(li.Features, feature)
}

// CanRunPro returns true if the license allows Pro features.
func (li *Info) CanRunPro() bool {
	return li.Valid && li.Tier >= TierPro
}

// proFeatures returns the feature list granted to NIAC Pro (product
// code 5001). Mirrors keygen's productCatalog.
//
// snmpv3 was removed from Pro on 2026-05-27: SNMPv3 is the only safe
// SNMP version (v1/v2c send credentials in cleartext), so gating it
// would push customers toward the insecure variants. BGP + OSPF stay
// on Pro because they're large protocol implementations, not a
// safety floor.
func proFeatures() []string {
	return []string{
		"unlimited_devices",
		"bgp", "ospf", "netbios", "ftp", "stp",
		"ipv6_advanced", "error_injection", "traffic_shaping",
		"config_templates", "multi_ip", "pcap_ingest", "rest_api",
	}
}
