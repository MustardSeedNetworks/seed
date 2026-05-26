/**
 * TierGate — render children regardless, but visually mark them as
 * locked + show an upgrade hint when the active license doesn't
 * include the feature.
 *
 * This is the "disable" variant of feature gating. Use it for
 * settings panels, individual buttons, or any UI where the customer
 * should still see the feature exists (so they know what they're
 * missing) but cannot interact with it without upgrading.
 *
 * For surfaces that should simply not exist on lower tiers, prefer
 * <RequireFeature>.
 *
 * Example:
 * ```tsx
 * <TierGate feature="rest_api" requiredTier="Pro">
 *   <Button onClick={mint}>Create token</Button>
 * </TierGate>
 * ```
 *
 * Interactive children should not need to know they're inside a
 * <TierGate> — the wrapper covers them with a transparent overlay
 * that intercepts clicks and shows the upgrade tooltip on hover.
 */

import type { ReactElement, ReactNode } from 'react';
import { useLicense } from '../../contexts/LicenseContext';

interface TierGateProps {
  feature: string;
  children: ReactNode;
  /** Tier name shown in the hint (e.g. "Pro", "Starter"). */
  requiredTier?: string;
  /** Custom message; default is "Requires the {tier} tier." */
  message?: string;
}

export function TierGate({
  feature,
  children,
  requiredTier = 'Pro',
  message,
}: TierGateProps): ReactElement {
  const { hasFeature, loading } = useLicense();

  // Always render children while the license is loading so layouts
  // don't shift; gate visually only after we know they lack the feature.
  if (loading || hasFeature(feature)) {
    return <>{children}</>;
  }

  const hint = message ?? `Requires the ${requiredTier} tier.`;

  // CSS-only hover (via `group` + `group-hover:`) keeps the tooltip
  // tied to the wrapper without JS event handlers — that lets the
  // outer span stay a presentation-only element and dodges the
  // a11y/noStaticElementInteractions rule.
  return (
    <span
      data-testid="tier-gate-locked"
      data-feature={feature}
      className="relative inline-block group"
    >
      <span aria-disabled="true" className="pointer-events-none opacity-60">
        {children}
      </span>
      {/* Transparent click-blocking layer. */}
      <span aria-hidden="true" className="absolute inset-0 cursor-not-allowed" title={hint} />
      <span
        role="tooltip"
        className="invisible group-hover:visible group-focus-within:visible absolute z-50 bottom-full left-1/2 -translate-x-1/2 mb-1 px-2 py-1 rounded bg-surface-raised border border-surface-border text-xs text-text-primary whitespace-nowrap shadow-lg"
      >
        {hint}
      </span>
    </span>
  );
}
