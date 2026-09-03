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
// dns_monitoring and ssl_cert_monitoring are count caps, not switches: Free
// keeps recurring probes (the probe engine is Free in server_engine_tiers.go)
// and Starter raises the ceiling. monitoring_scheduled, compliance_basic and
// wifi_visibility_basic left on 2026-09-03 (#2327) because each named
// something Free already does and nothing distinguished the paid version --
// selling a string with no boundary behind it is what this catalogue keeps
// getting wrong.
func starterFeatures() []string {
	return []string{
		"export_csv_json",
		"dns_monitoring",
		"ssl_cert_monitoring",
	}
}

// proFeatures returns the feature list granted to Seed Pro (product
// code 4002). Includes every Starter feature plus the Pro additions.
// Mirrors keygen's productCatalog (anchor: v2.3.0, 2026-05-30).
//
// Removed 2026-09-03 (#2327) because nothing a customer can reach delivers
// them: airmapper_baseline_diff has no implementation at all and the survey
// import it named belongs to another product; white_label has a clients table
// with a branding_json column and a repository, and db.Clients() has no caller,
// no route and nothing that reads the branding; scheduled_reports has a working
// tick engine that Start() runs at boot, and no route through which anyone can
// ever create a schedule for it to find. The code stays where it is useful --
// what is deleted is the claim that it is for sale. Re-add each string when the
// capability is reachable.
func proFeatures() []string {
	pro := []string{
		"wifi_association_forensics",
		"anomaly_detection",
		"path_analysis",
		"live_telemetry",
		"compliance_advanced",
		"audit_pdf",
		"multi_interface",
		"multi_user",
		"multi_client",
		"sso",
		"rest_api",
		// Each of these has a real implementation, verified rather than
		// asserted: bgp4 and hostresources are registered collectors in
		// internal/polling/snmp/orchestrator, and estate_polling is the
		// number of devices being polled. Their gates land next (#2327).
		//
		// topology_estate and extended_retention left on 2026-09-03: the
		// topology reconcilers are already Starter-gated in
		// server_engine_tiers.go and retention.tierHorizons already varies
		// by tier, so both boundaries existed and neither needed a second
		// mechanism. wifi_roam_analysis and wifi_rogue_detection left too --
		// both are surfaced by GET /wifi/anomalies, which is already Pro via
		// wifi_association_forensics. Three strings for one boundary.
		"estate_polling",
		"server_monitoring",
		"bgp_monitoring",
		"wifi_management_capture",
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
