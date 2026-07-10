// SPDX-License-Identifier: BUSL-1.1

// Package license is Seed's thin product layer over the fleet-shared
// github.com/MustardSeedNetworks/foundation/pkg/license core. The generic
// signing format, device fingerprint, activation state machine and encrypted
// on-disk state all live in foundation; this package supplies only Seed's
// product policy (tier vocabulary, product codes, feature catalog, trial terms)
// and re-exports the manager surface its callers use. See ADR-0019.
package license

import fnd "github.com/MustardSeedNetworks/foundation/pkg/license"

// Seed licenses are Ed25519-signed tokens verified offline against an embedded
// public key (the matching private key lives only in the keygen tool). Product
// codes: 4001 = Starter (tier 1), 4002 = Pro (tier 2, includes every Starter
// feature). Free is the unlicensed tier and needs no token.
const (
	// productName identifies this binary in a signed payload. A token issued
	// for another product (stem/niac) is rejected even if correctly signed.
	productName = "seed"

	// Product codes accepted by Seed.
	codeStarter = "4001"
	codePro     = "4002"

	// defaultMaxDevices is the activation cap assumed until a validated token
	// specifies its own MaxDevices.
	defaultMaxDevices = 3

	// TrialDays is the trial-period length before a paid key is required.
	// Exported for the CLI's "days left of N" display.
	TrialDays = 14

	// encryptionSalt is product-distinct so a license file from another
	// product can't be reused by renaming it into Seed's config directory.
	encryptionSalt = "MSN-SEED-DIAG-2026-LICENSE"

	// configSubdir is the directory under ~/.config where activation state is
	// persisted (~/.config/seed); licenseFileName is the file within it.
	configSubdir    = "seed"
	licenseFileName = ".license"
)

// Tier represents the license tier. Numeric values are the wire-tier values
// embedded in the signed token payload; tier 0 is the implicit Free tier.
type Tier int

// License tier constants.
const (
	// TierInvalid represents an invalid or unrecognized license tier.
	TierInvalid Tier = -1
	// TierFree is the unlicensed tier. No key needed; only the basic feature
	// set is available.
	TierFree Tier = 0
	// TierStarter unlocks the Starter feature set. Wire tier value 1.
	TierStarter Tier = 1
	// TierPro unlocks the full Professional feature set (includes everything
	// in Starter). Wire tier value 2.
	TierPro Tier = 2
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

// featuresForTier maps a signed wire-tier to the features Seed grants and the
// product code expected for that tier. Only Starter/Pro carry a token; Free
// (and any unrecognized tier) is rejected so a signed token can only grant what
// this build knows about. Passed to foundation as ProductPolicy.FeaturesForTier.
func featuresForTier(wireTier int) ([]string, string, bool) {
	switch Tier(wireTier) {
	case TierStarter:
		return starterFeatures(), codeStarter, true
	case TierPro:
		return proFeatures(), codePro, true
	case TierFree, TierInvalid:
		return nil, "", false
	default:
		return nil, "", false
	}
}

// Policy is Seed's product configuration for the foundation license core.
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
