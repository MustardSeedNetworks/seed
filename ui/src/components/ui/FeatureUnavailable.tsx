/**
 * FeatureUnavailable — this operating system cannot do it, so say so.
 *
 * The platform-axis sibling of <RequireFeature>: same shape, different reason.
 * RequireFeature hides a surface the licence does not include; this replaces a
 * surface the OS cannot support with an explanation of why, and the alternative
 * if there is one.
 *
 * ⚠️ The two axes must stay visibly distinct in the copy. "Your licence does
 * not include this" and "your OS cannot do this" have different remedies — buy
 * something, or run it somewhere else — and conflating them would be worse than
 * the silent failure this replaces (#750).
 *
 * While the capability report is in flight this renders children rather than
 * the notice, so a supported feature does not flash an "unavailable" message
 * before the fetch resolves. The report describes the OS, which cannot change
 * under us; guessing "unavailable" first would be wrong more often than right.
 *
 * Example:
 * ```tsx
 * <FeatureUnavailable capability="cable_diagnostics">
 *   <CableTestCard />
 * </FeatureUnavailable>
 * ```
 */

import type { ReactElement, ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { usePlatformCapabilities } from '../../hooks/usePlatformCapabilities';
import { cn, radius, spacing } from '../../styles/theme';

interface FeatureUnavailableProps {
  capability: string;
  children: ReactNode;
  /** Rendered instead of the default notice, for surfaces with their own empty state. */
  fallback?: ReactNode;
}

export function FeatureUnavailable({
  capability,
  children,
  fallback,
}: FeatureUnavailableProps): ReactElement {
  const { levelOf, all, loading } = usePlatformCapabilities();
  const { t } = useTranslation('errors');

  if (loading || levelOf(capability) !== 'none') {
    return <>{children}</>;
  }
  if (fallback !== undefined) {
    return <>{fallback}</>;
  }

  const entry = all.find((candidate) => candidate.capability === capability);

  return (
    <div
      role="status"
      data-testid="feature-unavailable"
      data-capability={capability}
      className={cn(
        'border border-surface-border bg-surface-base',
        radius.default,
        spacing.pad.sm,
        'stack-xs',
      )}
    >
      <p className="body-small font-medium text-text-primary">
        {t('platform.unavailableTitle', { feature: entry?.title ?? capability })}
      </p>
      {/* The note is the backend's, so the reason a platform cannot do
          something is written once, next to the level that says so. */}
      <p className="caption text-text-secondary">
        {entry?.note ?? t('platform.unavailableGeneric')}
      </p>
    </div>
  );
}
