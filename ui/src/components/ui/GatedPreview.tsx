/**
 * GatedPreview — a Pro-gated page on a lower tier shows what it does.
 *
 * The third variant of licence gating, for whole pages. <RequireFeature>
 * hides a surface and <TierGate> disables a control; a page rendered through
 * either is a banner over an empty body, which reads as "failed to load"
 * (#2669). This renders the pitch first, then a non-interactive sample of the
 * feature, so the customer can see what the tier buys before buying it.
 *
 * The sample is a fixture, never the live page body: the real body's fetches
 * are the ones `requireFeature` answers with 402, so rendering it here would
 * fill the preview with errors.
 *
 * While the licence fetch is in flight this renders nothing — children would
 * fire the gated fetches, and the pitch would flash at an entitled user.
 *
 * Example:
 * ```tsx
 * <GatedPreview feature="path_analysis" preview={<PathAnalysisPreview />}>
 *   <PathAnalysisBody />
 * </GatedPreview>
 * ```
 */

import type { ReactElement, ReactNode } from 'react';
import { Trans, useTranslation } from 'react-i18next';
import { FEATURE_CATALOG, type GatedFeature } from '../../constants/featureCatalog';
import { useLicense } from '../../contexts/LicenseContext';
import { cn, radius, spacing } from '../../styles/theme';

interface GatedPreviewProps {
  feature: GatedFeature;
  /** Non-interactive sample of the feature, shown when it is not licensed. */
  preview: ReactNode;
  /** The real page body, rendered only when the feature is licensed. */
  children: ReactNode;
}

export function GatedPreview({
  feature,
  preview,
  children,
}: GatedPreviewProps): ReactElement | null {
  const { hasFeature, loading } = useLicense();
  const { t } = useTranslation('errors');

  if (loading) {
    return null;
  }
  if (hasFeature(feature)) {
    return <>{children}</>;
  }

  const { tier } = FEATURE_CATALOG[feature];
  const name = t(`license.gated.features.${feature}.name`);

  return (
    <div data-testid="gated-preview" data-feature={feature} className="stack-sm">
      {/* Pitch before sample in DOM order: the sample is inert and out of the
          accessibility tree, so the explanation has to be reached first. */}
      <section
        data-testid="gated-pitch"
        className={cn(
          'border border-brand-primary/30 bg-brand-primary/5',
          radius.default,
          spacing.pad.sm,
          'stack-xs',
        )}
      >
        <h2 className="body-large font-semibold text-text-primary">
          {t('license.gated.title', { feature: name, tier })}
        </h2>
        <p className="body-small text-text-secondary">
          {t(`license.gated.features.${feature}.pitch`)}
        </p>
        <p className="caption text-text-secondary">
          <Trans
            i18nKey="license.gated.actions"
            ns="errors"
            values={{ tier }}
            components={{
              code: <code className="mx-1 px-1 rounded bg-surface-raised" />,
              code2: <code className="ml-tight px-1 rounded bg-surface-raised" />,
            }}
          />
        </p>
      </section>

      <p className="caption text-text-secondary">{t('license.gated.sampleLabel')}</p>
      {/* inert, not pointer-events alone: the sample must leave the tab order
          and the accessibility tree, or it reads as a page that does nothing. */}
      <div inert={true} aria-hidden="true" className="pointer-events-none opacity-60">
        {preview}
      </div>
    </div>
  );
}
