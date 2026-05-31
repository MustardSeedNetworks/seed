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
Seed License Key Format (16 characters, identical to the format
used by Stem and NIAC — keys are produced by the canonical keygen
tool and the rotor cipher is byte-compatible across products).

+------+--------+-------+------+----------+
| CC   | PPPP   |SSSSSSS| T    | XX       |
|Check |Product |Serial |Tier  | Checksum |
+------+--------+-------+------+----------+

Positions:
  0-1:  Checksum prefix (encoded validation).
  2-5:  Product code (4001=Seed Starter, 4002=Seed Pro).
  6-12: Serial number (unique per license).
  13:   Tier (1=Starter, 2=Pro).
  14-15: Checksum suffix.

Free is the unlicensed tier — no key required. Starter and Pro
require valid keys with matching product codes.
*/

// License key format constants.
const (
	keyLength         = 16
	productCodeLength = 4
	serialLength      = 7
	checksumLength    = 2
	cipherStartPos    = 7 // MSN rotor cipher starting position.
	defaultMaxDevices = 3 // Default activations per license.
)

// Tier represents the license tier.
type Tier int

// License tier constants. Numeric values are wire-compatible with the
// tier digit embedded in the license key (positions 13 of the encoded
// key). Tier 0 is the implicit Free tier.
const (
	// TierInvalid represents an invalid or unrecognized license tier.
	TierInvalid Tier = -1
	// TierFree is the unlicensed tier. No key needed; only the basic
	// feature set is available.
	TierFree Tier = 0
	// TierStarter unlocks the Starter feature set. Wire tier digit 1.
	TierStarter Tier = 1
	// TierPro unlocks the full Professional feature set (includes
	// everything in Starter). Wire tier digit 2.
	TierPro Tier = 2
)

// Error messages.
const (
	errProductCodeMismatch = "Product code mismatch for tier"
	// ErrLicenseKeyLength indicates the key length validation error message.
	ErrLicenseKeyLength = "License key must be 16 characters"
)

// String returns the tier name.
func (t Tier) String() string {
	switch t {
	case TierInvalid:
		return "Invalid"
	case TierFree:
		return "Free"
	case TierStarter:
		return "Starter"
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

	// Normalize the key (strip separators, uppercase).
	normalized := normalizeKey(key)
	if len(normalized) != keyLength {
		info.ErrorMsg = ErrLicenseKeyLength
		return info
	}

	// Reject any character outside the alphanumeric set the cipher
	// emits. This catches transcription errors before we attempt to
	// decode (the cipher would otherwise treat unexpected characters
	// as literal passthrough, producing confusing validation errors).
	if !validKeyChars(normalized) {
		info.ErrorMsg = "License key contains invalid characters"
		return info
	}

	// Decode through the rotor cipher.
	cipher := NewRotorCipher(cipherStartPos)
	decoded := cipher.DecodeString(normalized)

	// Validate the embedded checksum.
	if !validateKeyChecksum(decoded) {
		info.ErrorMsg = "License key failed checksum validation"
		return info
	}

	// Extract components.
	info.ProductCode = decoded[2:6]
	info.Serial = decoded[6:13]
	tierChar := decoded[13]

	// Parse tier.
	switch tierChar {
	case '1':
		info.Tier = TierStarter
		info.Features = starterFeatures()
	case '2':
		info.Tier = TierPro
		info.Features = proFeatures()
	default:
		info.ErrorMsg = "Invalid license tier"
		return info
	}

	// Validate product code matches tier.
	switch info.ProductCode {
	case "4001":
		if info.Tier != TierStarter {
			info.ErrorMsg = errProductCodeMismatch
			return info
		}
	case "4002":
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

// validKeyChars returns true if every character of the normalized key
// is in the cipher's accepted alphabet (digits, uppercase ASCII).
var keyCharRE = regexp.MustCompile(`^[0-9A-Z]+$`)

func validKeyChars(s string) bool {
	return keyCharRE.MatchString(s)
}

// validateKeyChecksum checks the embedded checksum.
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
	if tier < TierStarter || tier > TierPro {
		return "", errors.New("invalid tier")
	}

	payload := productCode + serial + fmt.Sprintf("%d", tier)
	checksum := CalculateChecksum(payload)
	fullKey := checksum[0:checksumLength] + payload + checksum
	cipher := NewRotorCipher(cipherStartPos)
	encoded := cipher.EncodeString(fullKey)
	return strings.ToUpper(encoded), nil
}

// normalizeKey cleans up a license key for validation.
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

// CanRunStarter returns true if the license allows Starter features.
func (li *Info) CanRunStarter() bool {
	return li.Valid && li.Tier >= TierStarter
}

// CanRunPro returns true if the license allows Pro features.
func (li *Info) CanRunPro() bool {
	return li.Valid && li.Tier >= TierPro
}

// starterFeatures returns the feature list granted to Seed Starter
// (product code 4001). Mirrors keygen's productCatalog.
//
// As of keygen v2.1.0 (2026-05-26) multi_interface moved Starter → Pro:
// Free/Starter are capped at 1 ethernet + 1 wifi; Pro is unlimited.
//
// V1.0 NMS expansion (2026-05-30) added three Starter flags:
// topology_local (local-jack LLDP/CDP view), dns_monitoring (5-target
// cap), ssl_cert_monitoring (5-cert cap). See
// msn-docs-internal/01-Strategy/SEED_NMS_EXPANSION.md.
func starterFeatures() []string {
	return []string{
		"monitoring_scheduled",
		"wifi_visibility_basic",
		"compliance_basic",
		"export_csv_json",
		"topology_local",
		"dns_monitoring",
		"ssl_cert_monitoring",
	}
}

// proFeatures returns the feature list granted to Seed Pro (product
// code 4002). Includes every Starter feature plus the Pro additions.
// Mirrors keygen's productCatalog (anchor: v2.3.0, 2026-05-30).
//
// V1.0 NMS expansion (2026-05-30) added 11 Pro flags:
// topology_estate, estate_polling, microburst_detection,
// voip_mos_scoring, server_monitoring, extended_retention,
// bgp_monitoring, wifi_management_capture, wifi_rogue_detection,
// config_backup_diff (V1.1), netflow_collection (V1.1).
func proFeatures() []string {
	pro := []string{
		"wifi_roam_analysis",
		"wifi_association_forensics",
		"airmapper_baseline_diff",
		"anomaly_detection",
		"path_analysis",
		"live_telemetry",
		"compliance_advanced",
		"scheduled_reports",
		"audit_pdf",
		"multi_interface",
		"multi_user",
		"multi_client",
		"sso",
		"white_label",
		"rest_api",
		// V1.0 NMS expansion (Phase 0 anchor, 2026-05-30).
		"topology_estate",
		"estate_polling",
		"microburst_detection",
		"voip_mos_scoring",
		"server_monitoring",
		"extended_retention",
		"bgp_monitoring",
		"wifi_management_capture",
		"wifi_rogue_detection",
		// V1.1 flags — present in catalog now; routes that gate on
		// them return 402 until Phase 4 (NetFlow + config backup/diff)
		// implements them.
		"config_backup_diff",
		"netflow_collection",
	}
	return append(starterFeatures(), pro...)
}
