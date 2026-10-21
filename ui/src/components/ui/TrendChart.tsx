/**
 * TrendChart — a recent window of bucketed samples as SVG lines, with each
 * series' peak, and the window control that picks the window.
 *
 * The server bins a window into buckets and sends only the buckets that hold
 * a sample. A stretch with no sample, such as a target that stopped
 * answering, breaks the line rather than being drawn across.
 */

import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';
import { useLocale } from '../../hooks/useLocale';
import { cn, input } from '../../styles/theme';

export const HISTORY_RANGES = ['1h', '24h', '7d'] as const;
export type HistoryRange = (typeof HISTORY_RANGES)[number];

const WIDTH = 600;
const HEIGHT = 120;
/** A gap this many poll intervals wide is a gap in the data, not a slow poll. */
const GAP_INTERVALS = 3;

const selectClass = cn(input.base, input.state.default, input.size.sm, 'body-small');

export interface TrendPoint {
  sampledAt: string;
}

export interface TrendSeries<P extends TrendPoint> {
  label: string;
  stroke: string;
  swatch: string;
  value: (p: P) => number | undefined;
}

/** The narrowest spacing between two samples: the poll interval, near enough. */
function pollStep(points: TrendPoint[], bucketSeconds: number): number {
  let step = Number.POSITIVE_INFINITY;
  for (let i = 1; i < points.length; i++) {
    const prev = points[i - 1];
    const cur = points[i];
    if (prev && cur) {
      step = Math.min(step, Date.parse(cur.sampledAt) - Date.parse(prev.sampledAt));
    }
  }
  return Math.max(bucketSeconds * 1000, Number.isFinite(step) ? step : 0);
}

/** SVG path for one series: a run of points, broken at gaps and missing values. */
export function seriesPath<P extends TrendPoint>(
  points: P[],
  value: TrendSeries<P>['value'],
  from: number,
  to: number,
  max: number,
  gapMs: number,
): string {
  const span = to - from || 1;
  let d = '';
  let prevAt: number | undefined;
  for (const p of points) {
    const at = Date.parse(p.sampledAt);
    const v = value(p);
    if (v === undefined) {
      prevAt = undefined;
      continue;
    }
    const x = ((at - from) / span) * WIDTH;
    const y = HEIGHT - (max > 0 ? (v / max) * HEIGHT : 0);
    const joined = prevAt !== undefined && at - prevAt <= gapMs;
    d += `${joined ? 'L' : 'M'}${x.toFixed(1)} ${y.toFixed(1)} `;
    prevAt = at;
  }
  return d.trim();
}

interface TrendChartProps<P extends TrendPoint> {
  testId: string;
  title: string;
  series: TrendSeries<P>[];
  points: P[];
  from: string;
  to: string;
  bucketSeconds: number;
  format: (v: number) => string;
}

export function TrendChart<P extends TrendPoint>({
  testId,
  title,
  series,
  points,
  from,
  to,
  bucketSeconds,
  format,
}: TrendChartProps<P>): JSX.Element {
  const { t } = useTranslation('common');
  const locale = useLocale();
  const peaks = series.map((s) => points.reduce((peak, p) => Math.max(peak, s.value(p) ?? 0), 0));
  const max = Math.max(...peaks);
  const fromMs = Date.parse(from);
  const toMs = Date.parse(to);
  const gapMs = pollStep(points, bucketSeconds) * GAP_INTERVALS;
  const time = new Intl.DateTimeFormat(locale, { dateStyle: 'short', timeStyle: 'short' });
  const summary = series
    .map((s, i) => t('trend.peak', { series: s.label, value: format(peaks[i] ?? 0) }))
    .join(' · ');

  return (
    <figure className="flex flex-col gap-1" data-testid={testId}>
      <figcaption className="flex flex-wrap items-baseline justify-between gap-x-3 text-xs">
        <span className="font-medium text-text-primary">{title}</span>
        <span className="text-text-secondary" data-testid={`${testId}-peaks`}>
          {summary}
        </span>
      </figcaption>
      <div className="flex gap-2">
        <div className="flex flex-col justify-between text-right text-[10px] text-text-muted tabular-nums">
          <span>{format(max)}</span>
          <span>{format(0)}</span>
        </div>
        <svg
          viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
          preserveAspectRatio="none"
          role="img"
          aria-label={`${title}: ${summary}`}
          className="h-28 min-w-0 flex-1 rounded-sm border border-surface-border bg-surface-sunken"
        >
          {series.map((s) => (
            <path
              key={s.label}
              d={seriesPath(points, s.value, fromMs, toMs, max, gapMs)}
              fill="none"
              strokeWidth={1.5}
              vectorEffect="non-scaling-stroke"
              className={s.stroke}
            />
          ))}
        </svg>
      </div>
      <div className="flex justify-between text-[10px] text-text-muted">
        <span>{time.format(fromMs)}</span>
        <ul className="flex gap-3">
          {series.map((s) => (
            <li key={s.label} className="flex items-center gap-1">
              <span className={`h-0.5 w-3 ${s.swatch}`} aria-hidden="true" />
              {s.label}
            </li>
          ))}
        </ul>
        <span>{time.format(toMs)}</span>
      </div>
    </figure>
  );
}

interface TrendWindowSelectProps {
  value: HistoryRange;
  onChange: (range: HistoryRange) => void;
  testId: string;
}

export function TrendWindowSelect({
  value,
  onChange,
  testId,
}: TrendWindowSelectProps): JSX.Element {
  const { t } = useTranslation('common');
  return (
    <label className="flex flex-col gap-1 text-xs text-text-muted">
      {t('trend.windowLabel')}
      <select
        value={value}
        onChange={(e): void => onChange(HISTORY_RANGES.find((r) => r === e.target.value) ?? value)}
        data-testid={testId}
        className={selectClass}
      >
        {HISTORY_RANGES.map((r) => (
          <option key={r} value={r}>
            {t(`trend.window.${r}`)}
          </option>
        ))}
      </select>
    </label>
  );
}
