/**
 * SurveyView heatmap metric selector.
 *
 * Categorised button grid for picking which metric to render as a
 * heatmap overlay (signal / interference / performance). Hidden once a
 * metric is selected and only shown when at least one sample exists.
 */

import { Activity, Clock, Gauge, Hash, Radio, Waves, Wifi } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import type { HeatmapMetric, SurveyType } from '../../hooks/useSurvey';
import { button, cn, icon as iconTokens, layout, radius, spacing } from '../../styles/theme';

interface SurveyViewHeatmapSelectorProps {
  heatmapMetric: HeatmapMetric;
  setHeatmapMetric: (metric: HeatmapMetric) => void;
  sampleCount: number;
  surveyType: SurveyType;
}

export function SurveyViewHeatmapSelector({
  heatmapMetric,
  setHeatmapMetric,
  sampleCount,
  surveyType,
}: SurveyViewHeatmapSelectorProps): JSX.Element | null {
  const { t } = useTranslation('survey');

  if (heatmapMetric !== null || sampleCount === 0) {
    return null;
  }

  return (
    <div class={cn(spacing.margin.bottom.content, spacing.stack.sm)}>
      {/* Signal Category */}
      <div>
        <div class={cn('body-small text-text-muted', spacing.margin.bottom.tight)}>
          {t('heatmaps.categories.signal')}
        </div>
        <div class={layout.inline.default}>
          <button
            type="button"
            onClick={() => setHeatmapMetric('rssi')}
            class={cn(
              button.size.sm,
              'body-small border border-surface-border',
              radius.md,
              'hover:bg-surface-hover',
              layout.inline.tight,
            )}
          >
            <Wifi class={iconTokens.size.sm} />
            {t('heatmaps.rssi')}
          </button>
          <button
            type="button"
            onClick={() => setHeatmapMetric('snr')}
            class={cn(
              button.size.sm,
              'body-small border border-surface-border',
              radius.md,
              'hover:bg-surface-hover',
              layout.inline.tight,
            )}
          >
            <Activity class={iconTokens.size.sm} />
            {t('heatmaps.snr')}
          </button>
          <button
            type="button"
            onClick={() => setHeatmapMetric('noise')}
            class={cn(
              button.size.sm,
              'body-small border border-surface-border',
              radius.md,
              'hover:bg-surface-hover',
              layout.inline.tight,
            )}
          >
            <Radio class={iconTokens.size.sm} />
            {t('heatmaps.noise')}
          </button>
        </div>
      </div>

      {/* Interference Category */}
      <div>
        <div class={cn('body-small text-text-muted', spacing.margin.bottom.tight)}>
          {t('heatmaps.categories.interference')}
        </div>
        <div class={layout.inline.default}>
          <button
            type="button"
            onClick={() => setHeatmapMetric('cochannel')}
            class={cn(
              button.size.sm,
              'body-small border border-surface-border',
              radius.md,
              'hover:bg-surface-hover',
              layout.inline.tight,
            )}
          >
            <Waves class={iconTokens.size.sm} />
            {t('heatmaps.cochannel')}
          </button>
          <button
            type="button"
            onClick={() => setHeatmapMetric('adjacent')}
            class={cn(
              button.size.sm,
              'body-small border border-surface-border',
              radius.md,
              'hover:bg-surface-hover',
              layout.inline.tight,
            )}
          >
            <Waves class={iconTokens.size.sm} />
            {t('heatmaps.adjacent')}
          </button>
          <button
            type="button"
            onClick={() => setHeatmapMetric('apDensity')}
            class={cn(
              button.size.sm,
              'body-small border border-surface-border',
              radius.md,
              'hover:bg-surface-hover',
              layout.inline.tight,
            )}
          >
            <Hash class={iconTokens.size.sm} />
            {t('heatmaps.apDensity')}
          </button>
          <button
            type="button"
            onClick={() => setHeatmapMetric('ssidCount')}
            class={cn(
              button.size.sm,
              'body-small border border-surface-border',
              radius.md,
              'hover:bg-surface-hover',
              layout.inline.tight,
            )}
          >
            <Hash class={iconTokens.size.sm} />
            {t('heatmaps.ssidCount')}
          </button>
        </div>
      </div>

      {/* Performance Category - only for throughput surveys */}
      {surveyType === 'throughput' && (
        <div>
          <div class={cn('body-small text-text-muted', spacing.margin.bottom.tight)}>
            {t('heatmaps.categories.performance')}
          </div>
          <div class={layout.inline.default}>
            <button
              type="button"
              onClick={() => setHeatmapMetric('throughput')}
              class={cn(
                button.size.sm,
                'body-small border border-surface-border',
                radius.md,
                'hover:bg-surface-hover',
                layout.inline.tight,
              )}
            >
              <Gauge class={iconTokens.size.sm} />
              {t('heatmaps.throughput')}
            </button>
            <button
              type="button"
              onClick={() => setHeatmapMetric('latency')}
              class={cn(
                button.size.sm,
                'body-small border border-surface-border',
                radius.md,
                'hover:bg-surface-hover',
                layout.inline.tight,
              )}
            >
              <Clock class={iconTokens.size.sm} />
              {t('heatmaps.latency')}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
