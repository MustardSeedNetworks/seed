/**
 * RequireFeature — render children only if the active license includes the feature.
 *
 * This is the "hide" variant of feature gating. Use it for entire
 * pages, drawer sections, nav items, or any UI surface that should
 * simply not exist for users on a lower tier.
 *
 * For interactive controls that should remain *visible but disabled*
 * with an upgrade hint, prefer <TierGate>.
 *
 * Example:
 * ```tsx
 * <RequireFeature feature="path_analysis">
 *   <PathAnalysisPage />
 * </RequireFeature>
 * ```
 *
 * While the initial license fetch is in flight, the component renders
 * `fallback` (default: null) — this avoids a flash of paid content
 * disappearing once the fetch resolves with no license.
 */

import type { ReactElement, ReactNode } from 'react';
import { useLicense } from '../../contexts/LicenseContext';

interface RequireFeatureProps {
  feature: string;
  children: ReactNode;
  /** Rendered when the feature is absent or license is still loading. */
  fallback?: ReactNode;
}

export function RequireFeature({
  feature,
  children,
  fallback = null,
}: RequireFeatureProps): ReactElement {
  const { hasFeature, loading } = useLicense();
  if (loading || !hasFeature(feature)) {
    return <>{fallback}</>;
  }
  return <>{children}</>;
}
