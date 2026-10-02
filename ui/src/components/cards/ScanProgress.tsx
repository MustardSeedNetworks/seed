/**
 * ScanProgress — compact discovery scan progress.
 *
 * The Phase 7 S3 replacement for PipelineProgress: where the legacy pipeline
 * emitted rich per-phase payloads (device counts, current target, per-phase
 * durations), the unified jobs spine surfaces a cumulative progress fraction
 * (useEngineScan) plus the current phase name (useEnginePhase, from the engine
 * event bus). This renders bar + percent + phase + cancel — the agreed S3
 * progress UX.
 */

import { Loader2, X } from 'lucide-react';
import type React from 'react';
import { memo } from 'react';
import { useTranslation } from 'react-i18next';
import { button, cn, icon as iconTokens, radius } from '../../styles/theme';

interface ScanProgressProps {
  /** Cumulative scan progress, 0..100. */
  percent: number;
  /** Engine phase name (e.g. discovery / enrichment); empty if not yet known. */
  phase: string;
  onCancel?: () => void;
}

// Engine scan phase → locale key for its label.
const PHASE_KEYS = {
  discovery: 'pipeline.phases.discovery',
  correlation: 'pipeline.phases.correlation',
  name_resolution: 'pipeline.phases.nameResolution',
  enrichment: 'pipeline.phases.enrichment',
  assessment: 'pipeline.phases.assessment',
} as const;

function isKnownPhase(phase: string): phase is keyof typeof PHASE_KEYS {
  return Object.hasOwn(PHASE_KEYS, phase);
}

export const ScanProgress: React.NamedExoticComponent<ScanProgressProps> = memo(
  function scanProgress({ percent, phase, onCancel }: ScanProgressProps): React.ReactElement {
    const { t } = useTranslation('cards');
    const clamped = Math.min(Math.max(percent, 0), 100);
    const phaseLabel = isKnownPhase(phase) ? t(PHASE_KEYS[phase]) : phase;

    return (
      <div className="stack-xs" data-testid="scan-progress">
        <div className="flex-between">
          <div className="flex items-center gap-compact">
            <Loader2 className={cn(iconTokens.size.sm, 'text-brand-primary animate-spin')} />
            <span className="body-small font-medium text-text-primary">
              {phaseLabel
                ? t('discovery.scanningPhase', { phase: phaseLabel })
                : t('discovery.scanning')}
            </span>
          </div>
          {onCancel ? (
            <button
              type="button"
              onClick={onCancel}
              data-testid="scan-cancel-button"
              className={cn(
                button.base,
                button.size.sm,
                button.variant.secondary,
                'flex items-center gap-tight',
              )}
              aria-label={t('pipeline.cancel')}
            >
              <X className={iconTokens.size.xs} />
              <span className="hidden sm:inline">{t('pipeline.cancel')}</span>
            </button>
          ) : null}
        </div>
        <div className={cn('h-2 bg-surface-sunken overflow-hidden', radius.default)}>
          <div
            className={cn('h-full bg-brand-primary transition-all duration-300', radius.default)}
            style={{ width: `${clamped}%` }}
            data-testid="scan-progress-bar"
          />
        </div>
        <div className="flex justify-end caption text-text-muted">
          <span>{Math.round(clamped)}%</span>
        </div>
      </div>
    );
  },
);
