// SPDX-License-Identifier: BUSL-1.1

package license

import (
	"slices"
	"strings"
	"time"
)

/*
Seed licenses are Ed25519-signed tokens (see signing.go). The previous 16-char
rotor-cipher key format was removed: its generator (GenerateLicenseKey) shipped
inside the binary, so any copy of Seed could mint a valid key. Tokens are now
signed by the keygen tool's private key and verified here with an embedded
public key — offline and un-forgeable. See ADR-0019.

Product codes:
  4001: Seed Starter (tier 1)
  4002: Seed Pro     (tier 2). Includes every Starter feature.

Free is the unlicensed tier — no key required.
*/

const (
	defaultMaxDevices = 3 // default activations per license

	// productName identifies this binary in a signed payload. A token issued
	// for another product (stem/niac) is rejected even if correctly signed.
	productName = "seed"
)

// Product codes accepted by Seed.
const (
	codeStarter = "4001"
	codePro     = "4002"
)

// licensePublicKeyB64 is the standard-base64 Ed25519 public key that verifies
// production license tokens. The matching private key lives only in the keygen
// tool (msn-internal-tools/keygen) and never ships. See ADR-0019.
//
// Pre-launch signing key — rotate via keygen before GA.
const licensePublicKeyB64 = "O+o8n4qHHp/X//JrRXSdgGSWa2Fqz79OtgUkcylNxZg="

// Tier represents the license tier.
type Tier int

// License tier constants. Numeric values are the wire tier values embedded in
// the signed token payload. Tier 0 is the implicit Free tier.
const (
	// TierInvalid represents an invalid or unrecognized license tier.
	TierInvalid Tier = -1
	// TierFree is the unlicensed tier. No key needed; only the basic
	// feature set is available.
	TierFree Tier = 0
	// TierStarter unlocks the Starter feature set. Wire tier value 1.
	TierStarter Tier = 1
	// TierPro unlocks the full Professional feature set (includes
	// everything in Starter). Wire tier value 2.
	TierPro Tier = 2
)

// Error messages.
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

// ValidateLicenseKey performs offline validation of a license token against the
// embedded production key. The verifier wraps a 32-byte key, so it is rebuilt
// per call rather than held as a package global; validation is not on a hot
// path (feature checks read cached Info, they do not re-validate).
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
	case int(TierStarter):
		info.Tier = TierStarter
		info.Features = starterFeatures()
	case int(TierPro):
		info.Tier = TierPro
		info.Features = proFeatures()
	default:
		info.ErrorMsg = "Invalid license tier"
		return info
	}

	if !productCodeMatchesTier(payload.Code, info.Tier) {
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

// productCodeMatchesTier enforces that the product code embedded in the payload
// is the one expected for the tier, so a token cannot claim a code/tier pairing
// the catalog never issued.
func productCodeMatchesTier(code string, tier Tier) bool {
	switch code {
	case codeStarter:
		return tier == TierStarter
	case codePro:
		return tier == TierPro
	default:
		return false
	}
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
