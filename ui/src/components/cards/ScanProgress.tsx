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

// Engine scan phase → human label. Engine phases differ from the legacy
// pipeline's (discovery/correlation/name_resolution/enrichment/assessment).
const PHASE_LABELS: Record<string, string> = {
  discovery: 'Discovery',
  correlation: 'Correlating',
  name_resolution: 'Name resolution',
  enrichment: 'Enrichment',
  assessment: 'Assessment',
};

export const ScanProgress: React.NamedExoticComponent<ScanProgressProps> = memo(
  function scanProgress({ percent, phase, onCancel }: ScanProgressProps): React.ReactElement {
    const { t } = useTranslation('cards');
    const clamped = Math.min(Math.max(percent, 0), 100);
    const phaseLabel = phase ? PHASE_LABELS[phase] || phase : '';

    return (
      <div className="stack-xs" data-testid="scan-progress">
        <div className="flex-between">
          <div className="flex items-center gap-compact">
            <Loader2 className={cn(iconTokens.size.sm, 'text-brand-primary animate-spin')} />
            <span className="body-small font-medium text-text-primary">
              {phaseLabel
                ? t('discovery.scanningPhase', {
                    phase: phaseLabel,
                    defaultValue: `Scanning — ${phaseLabel}`,
                  })
                : t('discovery.scanning', { defaultValue: 'Scanning…' })}
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
              aria-label={t('pipeline.cancel', { defaultValue: 'Cancel' })}
            >
              <X className={iconTokens.size.xs} />
              <span className="hidden sm:inline">
                {t('pipeline.cancel', { defaultValue: 'Cancel' })}
              </span>
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
