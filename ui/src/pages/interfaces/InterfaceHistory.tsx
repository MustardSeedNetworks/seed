/**
 * InterfaceHistory — one interface's traffic and errors over a recent window
 * (UI-SEED-21), opened from a row of the Interfaces page.
 *
 * The server bins the window into at most 240 buckets: traffic is a bucket's
 * mean and errors its peak, so one errored poll still shows. A stretch with no
 * poll, such as a target that stopped answering, breaks the line rather than
 * being drawn across.
 */

import { type JSX, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { X } from '../../components/ui/Icons';
import {
  HISTORY_RANGES,
  type HistoryRange,
  useInterfaceHistory,
} from '../../hooks/useInterfaceHistory';
import { useLocale } from '../../hooks/useLocale';
import { formatBitRate, formatPerSecond } from '../../lib/format';
import { cn, input } from '../../styles/theme';
import type { InterfaceRatesResponse } from '../../types/generated/interface-history-response';
import type { InterfaceStatsResponse } from '../../types/generated/interface-stats-list-response';

const BITS_PER_OCTET = 8;
const WIDTH = 600;
const HEIGHT = 120;
/** A gap this many poll intervals wide is a gap in the data, not a slow poll. */
const GAP_INTERVALS = 3;

const selectClass = cn(input.base, input.state.default, input.size.sm, 'body-small');

interface Series {
  label: string;
  stroke: string;
  swatch: string;
  value: (p: InterfaceRatesResponse) => number | undefined;
}

/** The narrowest spacing between two polls: the poll interval, near enough. */
function pollStep(points: InterfaceRatesResponse[], bucketSeconds: number): number {
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
export function seriesPath(
  points: InterfaceRatesResponse[],
  value: Series['value'],
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

interface RateChartProps {
  testId: string;
  title: string;
  series: Series[];
  points: InterfaceRatesResponse[];
  from: string;
  to: string;
  bucketSeconds: number;
  format: (v: number) => string;
}

function RateChart({
  testId,
  title,
  series,
  points,
  from,
  to,
  bucketSeconds,
  format,
}: RateChartProps): JSX.Element {
  const { t } = useTranslation('pages');
  const locale = useLocale();
  const peaks = series.map((s) => points.reduce((peak, p) => Math.max(peak, s.value(p) ?? 0), 0));
  const max = Math.max(...peaks);
  const fromMs = Date.parse(from);
  const toMs = Date.parse(to);
  const gapMs = pollStep(points, bucketSeconds) * GAP_INTERVALS;
  const time = new Intl.DateTimeFormat(locale, { dateStyle: 'short', timeStyle: 'short' });
  const summary = series
    .map((s, i) => t('interfaces.history.peak', { series: s.label, value: format(peaks[i] ?? 0) }))
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

function bits(octets: number | undefined): number | undefined {
  return octets === undefined ? undefined : octets * BITS_PER_OCTET;
}

interface InterfaceHistoryProps {
  iface: InterfaceStatsResponse;
  onClose: () => void;
}

export function InterfaceHistory({ iface, onClose }: InterfaceHistoryProps): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const [range, setRange] = useState<HistoryRange>('24h');
  const { history, loading, error } = useInterfaceHistory(iface.targetId, iface.ifIndex, range);
  const heading = useRef<HTMLHeadingElement>(null);

  // Opening moves focus here, so a keyboard or screen reader user lands on
  // what they opened rather than back at the top of the table.
  useEffect(() => {
    heading.current?.focus();
  }, []);

  let body: JSX.Element;
  if (error) {
    body = (
      <p
        role="alert"
        data-testid="interface-history-error"
        className="text-sm text-status-error-strong"
      >
        {error}
      </p>
    );
  } else if (loading || !history) {
    body = (
      <p className="body-small" data-testid="interface-history-loading">
        {t('common:status.loading')}
      </p>
    );
  } else if (history.points.length === 0) {
    body = (
      <p className="body-small" data-testid="interface-history-empty">
        {t('interfaces.history.empty')}
      </p>
    );
  } else {
    const shown = {
      points: history.points,
      from: history.from,
      to: history.to,
      bucketSeconds: history.bucketSeconds,
    };
    body = (
      <div className="flex flex-col gap-4">
        <RateChart
          testId="interface-history-traffic"
          title={t('interfaces.history.traffic')}
          series={[
            {
              label: t('interfaces.history.in'),
              stroke: 'stroke-cat-1',
              swatch: 'bg-cat-1',
              value: (p) => bits(p.inOctetsPerSec),
            },
            {
              label: t('interfaces.history.out'),
              stroke: 'stroke-cat-2',
              swatch: 'bg-cat-2',
              value: (p) => bits(p.outOctetsPerSec),
            },
          ]}
          format={(v): string => formatBitRate(v)}
          {...shown}
        />
        <RateChart
          testId="interface-history-errors"
          title={t('interfaces.history.errors')}
          series={[
            {
              label: t('interfaces.colErrors'),
              stroke: 'stroke-cat-3',
              swatch: 'bg-cat-3',
              value: (p) => p.inErrorsPerSec + p.outErrorsPerSec,
            },
            {
              label: t('interfaces.colDiscards'),
              stroke: 'stroke-cat-4',
              swatch: 'bg-cat-4',
              value: (p) => p.inDiscardsPerSec + p.outDiscardsPerSec,
            },
          ]}
          format={(v): string => formatPerSecond(v)}
          {...shown}
        />
      </div>
    );
  }

  return (
    <section
      aria-labelledby="interface-history-heading"
      data-testid="interface-history"
      className="flex flex-col gap-3 rounded-lg border border-surface-border bg-surface-raised pad"
    >
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <h2
            id="interface-history-heading"
            ref={heading}
            tabIndex={-1}
            className="break-all text-sm font-medium text-text-primary outline-none"
          >
            {iface.name || `#${iface.ifIndex}`}
          </h2>
          <p className="break-words text-xs text-text-muted">{iface.targetName}</p>
        </div>
        <div className="flex items-end gap-2">
          <label className="flex flex-col gap-1 text-xs text-text-muted">
            {t('interfaces.history.windowLabel')}
            <select
              value={range}
              onChange={(e): void =>
                setRange(HISTORY_RANGES.find((r) => r === e.target.value) ?? range)
              }
              data-testid="interface-history-range"
              className={selectClass}
            >
              {HISTORY_RANGES.map((r) => (
                <option key={r} value={r}>
                  {t(`interfaces.history.window.${r}`)}
                </option>
              ))}
            </select>
          </label>
          <button
            type="button"
            onClick={onClose}
            data-testid="interface-history-close"
            aria-label={t('interfaces.history.close')}
            className="inline-flex min-h-8 min-w-8 items-center justify-center rounded-md text-text-secondary hover:bg-surface-sunken hover:text-text-primary"
          >
            <X className="h-4 w-4" aria-hidden="true" />
          </button>
        </div>
      </div>
      {body}
    </section>
  );
}
