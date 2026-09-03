/**
 * LimitedFeature — it works here, but not fully, so say what is missing.
 *
 * The platform-axis sibling of <TierGate>: renders children regardless and adds
 * a note about what is reduced. Where <FeatureUnavailable> replaces a surface
 * the OS cannot support, this annotates one it supports partially — "reports
 * negotiated speed; duplex is not exposed" is more useful than a blank field
 * the operator has to guess about.
 *
 * ⚠️ Distinct from the licence axis by design. A limited capability is not an
 * upsell and must never read like one (#750).
 *
 * `limited` and `partial` are both annotated: the difference between them
 * matters to the matrix (vendor tooling versus reduced API) but not to the
 * operator standing in front of the screen, who needs the note either way.
 */

import type { ReactElement, ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { usePlatformCapabilities } from '../../hooks/usePlatformCapabilities';
import { cn, spacing } from '../../styles/theme';

interface LimitedFeatureProps {
  capability: string;
  children: ReactNode;
}

export function LimitedFeature({ capability, children }: LimitedFeatureProps): ReactElement {
  const { levelOf, all, loading } = usePlatformCapabilities();
  const { t } = useTranslation('errors');

  const level = levelOf(capability);
  if (loading || (level !== 'partial' && level !== 'limited')) {
    return <>{children}</>;
  }

  const entry = all.find((candidate) => candidate.capability === capability);

  return (
    <div className="stack-xs">
      {children}
      <p
        role="status"
        data-testid="limited-feature"
        data-capability={capability}
        className={cn('caption text-status-warning', spacing.margin.top.tight)}
      >
        {entry?.note ?? t('platform.limitedGeneric')}
      </p>
    </div>
  );
}
