/**
 * Feature catalog — the licence features the UI gates a whole page on.
 *
 * One entry per feature the UI is allowed to gate on. The tier is the tier
 * `internal/license/policy.go` grants the feature in, and the feature must be
 * enforced by `requireFeature` on at least one route; both are held by
 * `scripts/check-feature-gate-parity.py`, so a UI gate can no longer name a
 * feature the server does not enforce or advertise the wrong tier (#2669).
 *
 * The pitch and the feature's display name are locale copy, keyed by the
 * feature id under `errors:license.gated.features.*`.
 */

export type FeatureTier = 'Starter' | 'Pro';

export interface CatalogEntry {
  /** Tier that grants the feature — must match policy.go. */
  tier: FeatureTier;
}

export const FEATURE_CATALOG = {
  path_analysis: { tier: 'Pro' },
  export_csv_json: { tier: 'Starter' },
} as const satisfies Record<string, CatalogEntry>;

/** Feature ids the UI may gate a page on. */
export type GatedFeature = keyof typeof FEATURE_CATALOG;
