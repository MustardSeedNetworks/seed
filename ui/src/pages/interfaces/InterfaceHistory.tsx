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
import { type HistoryRange, TrendChart, TrendWindowSelect } from '../../components/ui/TrendChart';
import { useInterfaceHistory } from '../../hooks/useInterfaceHistory';
import { formatBitRate, formatPerSecond } from '../../lib/format';
import type { InterfaceStatsResponse } from '../../types/generated/interface-stats-list-response';

const BITS_PER_OCTET = 8;

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
        <TrendChart
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
        <TrendChart
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
          <TrendWindowSelect value={range} onChange={setRange} testId="interface-history-range" />
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
