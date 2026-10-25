/**
 * IpSlaPage — the Cisco IP-SLA operations the polling targets run, over
 * /api/v1/ip-sla (P-D8, UI-SEED-48).
 *
 * The list keeps the server's order, failing operations first, and words each
 * operation's latest run. The detail charts one operation's trend: per bucket
 * the mean round trip (and jitter, for a jitter operation) and the share of
 * runs that completed (and packet loss).
 */

import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  type HistoryRange,
  TrendChart,
  type TrendSeries,
  TrendWindowSelect,
} from '../components/ui/TrendChart';
import { useIpSla, useIpSlaHistory } from '../hooks/useIpSla';
import { useLocale } from '../hooks/useLocale';
import type { IPSLARunResponse } from '../types/generated/ip-sla-history-response';
import type { IPSLAOperationResponse } from '../types/generated/ip-sla-operations-response';
import {
  DetailEmpty,
  DetailFacts,
  DetailPane,
  ListDetail,
  RecordPane,
  RecordRow,
  type RecordState,
} from '../ui/ListDetail';

/** rttMonLatestRttOperSense values the page words; any other is a failure. */
const SENSE_OK = 1;
const SENSE_OVER_THRESHOLD = 3;
const SENSE_TIMEOUT = 4;

/** rttMonCtrlAdminStatus: the operation is scheduled and running. */
const STATUS_ACTIVE = 1;

/** RttMonRttType values with a name; any other shows its number. */
const KNOWN_TYPES = [1, 2, 5, 6, 7, 8, 9, 11, 12, 16, 19, 20] as const;
type KnownType = (typeof KNOWN_TYPES)[number];

function isKnownType(rttType: number): rttType is KnownType {
  return KNOWN_TYPES.some((known) => known === rttType);
}

function keyOf(op: IPSLAOperationResponse): string {
  return `${op.targetId}#${op.index}`;
}

function stateOf(op: IPSLAOperationResponse): RecordState | undefined {
  if (!op.latest) {
    return undefined;
  }
  if (!op.latest.completed) {
    return 'crit';
  }
  return op.latest.sense === SENSE_OVER_THRESHOLD ? 'warn' : 'ok';
}

const ms = (v: number): string => `${Math.round(v * 10) / 10} ms`;
const pct = (v: number): string => `${Math.round(v)}%`;

function HistoryCharts({ op }: { op: IPSLAOperationResponse }): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const [range, setRange] = useState<HistoryRange>('24h');
  const { history, loading, error } = useIpSlaHistory(op.targetId, op.index, range);

  let body: JSX.Element;
  if (error) {
    body = (
      <p
        role="alert"
        data-testid="ipsla-history-error"
        className="text-sm text-status-error-strong"
      >
        {t('ipSla.historyFailed', { reason: error })}
      </p>
    );
  } else if (loading || !history) {
    body = (
      <p className="body-small" data-testid="ipsla-history-loading">
        {t('common:status.loading')}
      </p>
    );
  } else if (history.points.length === 0) {
    body = (
      <p className="body-small" data-testid="ipsla-history-empty">
        {t('ipSla.historyEmpty')}
      </p>
    );
  } else {
    // Jitter, loss and MOS exist only for a jitter-type operation.
    const has = (value: (p: IPSLARunResponse) => number | undefined): boolean =>
      history.points.some((p) => value(p) !== undefined);
    const timing: TrendSeries<IPSLARunResponse>[] = [
      {
        label: t('ipSla.seriesRtt'),
        stroke: 'stroke-cat-1',
        swatch: 'bg-cat-1',
        value: (p) => p.rttMs,
      },
    ];
    if (has((p) => p.jitterMs)) {
      timing.push({
        label: t('ipSla.seriesJitter'),
        stroke: 'stroke-cat-2',
        swatch: 'bg-cat-2',
        value: (p) => p.jitterMs,
      });
    }
    const outcome: TrendSeries<IPSLARunResponse>[] = [
      {
        label: t('ipSla.seriesCompleted'),
        stroke: 'stroke-cat-3',
        swatch: 'bg-cat-3',
        value: (p) => p.completedPct,
      },
    ];
    if (has((p) => p.lossPct)) {
      outcome.push({
        label: t('ipSla.seriesLoss'),
        stroke: 'stroke-cat-4',
        swatch: 'bg-cat-4',
        value: (p) => p.lossPct,
      });
    }
    const shown = {
      points: history.points,
      from: history.from,
      to: history.to,
      bucketSeconds: history.bucketSeconds,
    };
    body = (
      <div className="flex flex-col gap-4">
        <TrendChart
          testId="ipsla-history-timing"
          title={t('ipSla.chartTiming')}
          series={timing}
          format={ms}
          {...shown}
        />
        <TrendChart
          testId="ipsla-history-outcome"
          title={t('ipSla.chartOutcome')}
          series={outcome}
          format={pct}
          {...shown}
        />
      </div>
    );
  }

  return (
    <section className="flex flex-col gap-3" data-testid="ipsla-history">
      <div className="flex flex-wrap items-end justify-between gap-2">
        <h3 className="text-sm font-medium text-text-primary">{t('ipSla.trend')}</h3>
        <TrendWindowSelect value={range} onChange={setRange} testId="ipsla-history-range" />
      </div>
      {body}
    </section>
  );
}

export function IpSlaPage(): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const locale = useLocale();
  const { data, error } = useIpSla();
  const [selectedKey, setSelectedKey] = useState<string | null>(null);

  const operations = data?.operations ?? [];
  const selected = operations.find((op) => keyOf(op) === selectedKey) ?? operations[0] ?? null;
  const failing = operations.filter((op) => op.latest && !op.latest.completed).length;
  const time = new Intl.DateTimeFormat(locale, { dateStyle: 'short', timeStyle: 'short' });

  const nameOf = (op: IPSLAOperationResponse): string =>
    op.tag || t('ipSla.operation', { index: op.index });
  const typeOf = (op: IPSLAOperationResponse): string =>
    isKnownType(op.rttType)
      ? t(`ipSla.type.t${op.rttType}`)
      : t('ipSla.typeOther', { type: op.rttType });
  const resultOf = (op: IPSLAOperationResponse): string => {
    const { latest } = op;
    if (!latest) {
      return t('ipSla.result.noRun');
    }
    switch (latest.sense) {
      case SENSE_OK:
        return ms(latest.rttMs);
      case SENSE_OVER_THRESHOLD:
        return t('ipSla.result.overThreshold', { rtt: ms(latest.rttMs) });
      case SENSE_TIMEOUT:
        return t('ipSla.result.timeout');
      default:
        return t('ipSla.result.failed', { sense: latest.sense });
    }
  };

  return (
    <>
      {error ? (
        <div
          role="alert"
          className="rounded-md border border-status-error/40 bg-status-error/10 pad-sm text-sm text-status-error-strong"
        >
          {t('ipSla.loadFailed', { reason: error })}
        </div>
      ) : null}

      <p className="body-small" data-testid="ipsla-counts">
        {data
          ? [
              t('ipSla.operationCount', { count: operations.length }),
              failing > 0 ? t('ipSla.failingCount', { count: failing }) : null,
            ]
              .filter(Boolean)
              .join(' · ')
          : t('common:status.loading')}
      </p>

      <ListDetail>
        <RecordPane empty={data ? t('ipSla.noOperations') : t('common:status.loading')}>
          {operations.map((op) => (
            <RecordRow
              key={keyOf(op)}
              data-testid={`ipsla-op-${keyOf(op)}`}
              name={nameOf(op)}
              meta={[op.targetName, op.target].filter(Boolean).join(' → ')}
              value={resultOf(op)}
              state={stateOf(op)}
              selected={selected !== null && keyOf(selected) === keyOf(op)}
              onSelect={(): void => setSelectedKey(keyOf(op))}
            />
          ))}
        </RecordPane>

        {selected ? (
          <DetailPane
            eyebrow={resultOf(selected)}
            title={nameOf(selected)}
            meta={selected.targetName}
          >
            <DetailFacts
              items={[
                { label: t('ipSla.labelType'), testId: 'ipsla-type', value: typeOf(selected) },
                { label: t('ipSla.labelDestination'), value: selected.target || '—' },
                {
                  label: t('ipSla.labelThreshold'),
                  testId: 'ipsla-threshold',
                  value: ms(selected.thresholdMs),
                },
                {
                  label: t('ipSla.labelFrequency'),
                  value: t('ipSla.everySeconds', { count: selected.frequencySeconds }),
                },
                {
                  label: t('ipSla.labelStatus'),
                  value:
                    selected.status === STATUS_ACTIVE
                      ? t('ipSla.statusActive')
                      : t('ipSla.statusInactive'),
                },
                {
                  label: t('ipSla.labelPolled'),
                  value: time.format(Date.parse(selected.observedAt)),
                },
              ]}
            />
            {selected.latest?.jitter ? (
              <DetailFacts
                items={[
                  {
                    label: t('ipSla.labelJitter'),
                    testId: 'ipsla-jitter',
                    value: ms(selected.latest.jitter.avgJitterMs),
                  },
                  {
                    label: t('ipSla.labelLost'),
                    value: t('ipSla.lostOf', {
                      lost: selected.latest.jitter.lost,
                      sent: selected.latest.jitter.lost + selected.latest.jitter.received,
                    }),
                  },
                  {
                    label: t('ipSla.labelMos'),
                    // MOS is 0 when the operation has no codec configured.
                    value:
                      selected.latest.jitter.mos > 0 ? selected.latest.jitter.mos.toFixed(2) : '—',
                  },
                ]}
              />
            ) : null}
            <HistoryCharts key={keyOf(selected)} op={selected} />
          </DetailPane>
        ) : (
          <DetailEmpty>{t('ipSla.emptyPrompt')}</DetailEmpty>
        )}
      </ListDetail>
    </>
  );
}
