/**
 * InterfacesPage — every interface the polling targets report, with the rates
 * of its latest rated poll (UI-SEED-21, P-A).
 *
 * Three states before the table means anything: loading; empty, when no
 * target has reported an interface; and no data yet, when interfaces exist
 * but none has a rate. The last is normal for a new target: a rate is the
 * difference between two polls, so the first poll lists the interfaces and
 * the second rates them.
 *
 * Opening an interface shows its history above the table (InterfaceHistory).
 */

import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Network } from '../components/ui/Icons';
import { AccentLink } from '../components/ui/Typography';
import { useInterfaceStats } from '../hooks/useInterfaceStats';
import { useTableLabels } from '../hooks/useTableLabels';
import { formatBitRate, formatPerSecond } from '../lib/format';
import { Alert } from '../shared/Alert';
import { DataTable, type DataTableColumn } from '../shared/DataTable';
import { EmptyState } from '../shared/EmptyState';
import { StatusPill, type StatusTone } from '../shared/StatusPill';
import type { InterfaceStatsResponse } from '../types/generated/interface-stats-list-response';
import { InterfaceHistory } from './interfaces/InterfaceHistory';

type OperStatus = InterfaceStatsResponse['operStatus'];

const BITS_PER_OCTET = 8;

function interfaceKey(iface: InterfaceStatsResponse): string {
  return `${iface.targetId}/${iface.ifIndex}`;
}

/**
 * The number a rate column sorts by. An interface with no rate yet has none to
 * rank, so it takes -1: below every real rate, last when the column is
 * descending.
 */
function rateValue(
  iface: InterfaceStatsResponse,
  pick: (r: NonNullable<InterfaceStatsResponse['rates']>) => number,
): number {
  return iface.rates ? pick(iface.rates) : -1;
}

/** DataTable compares strings without numeric collation; zero-padding digit runs gives Gi0/2 before Gi0/10. */
function naturalKey(name: string): string {
  return name.replace(/\d+/g, (digits) => digits.padStart(10, '0'));
}

const STATUS_TONE: Record<OperStatus, StatusTone> = {
  up: 'success',
  down: 'error',
  lowerLayerDown: 'error',
  testing: 'warning',
  dormant: 'unknown',
  notPresent: 'unknown',
  unknown: 'unknown',
};

function bitRate(octets: number | undefined): string {
  return octets === undefined ? '—' : formatBitRate(octets * BITS_PER_OCTET);
}

function percent(pct: number | undefined): string {
  return pct === undefined ? '—' : `${Number.parseFloat(pct.toFixed(1))}%`;
}

export function InterfacesPage(): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const { interfaces, loading, error } = useInterfaceStats();
  const [openKey, setOpenKey] = useState<string | null>(null);
  const labels = useTableLabels<InterfaceStatsResponse>({
    loading: t('common:status.loading'),
  });

  if (error) {
    return (
      <div data-testid="interfaces-error">
        <Alert tone="error">{error}</Alert>
      </div>
    );
  }
  if (loading) {
    return (
      <div data-testid="interfaces-loading">
        <DataTable
          rows={[]}
          columns={[]}
          getRowKey={interfaceKey}
          caption={t('interfaces.title')}
          loading
          labels={labels}
          empty={null}
        />
      </div>
    );
  }
  if (interfaces.length === 0) {
    return (
      <div data-testid="interfaces-empty">
        <EmptyState
          icon={<Network className="h-6 w-6" aria-hidden="true" />}
          title={t('interfaces.emptyTitle')}
          hint={t('interfaces.emptyHint')}
          action={
            <AccentLink to="/polling-targets" className="inline-flex min-h-6 items-center">
              {t('interfaces.addTarget')}
            </AccentLink>
          }
        />
      </div>
    );
  }

  const rated = interfaces.some((iface) => iface.rates);
  const open = interfaces.find((iface) => interfaceKey(iface) === openKey);

  const rateCell = (
    iface: InterfaceStatsResponse,
    render: (r: NonNullable<InterfaceStatsResponse['rates']>) => string,
    first = false,
  ): string => {
    if (!iface.rates) {
      return first ? t('interfaces.noRateYet') : '—';
    }
    return render(iface.rates);
  };

  const columns: DataTableColumn<InterfaceStatsResponse>[] = [
    {
      key: 'interface',
      header: t('interfaces.colInterface'),
      sortValue: (iface) => `${iface.targetName}\u0000${naturalKey(iface.name)}`,
      cell: (iface) => (
        <div className="flex min-w-0 flex-col gap-0.5 py-1">
          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              onClick={(): void => setOpenKey(interfaceKey(iface))}
              aria-expanded={openKey === interfaceKey(iface)}
              className="min-h-6 break-all text-left font-medium text-text-primary underline-offset-2 hover:underline"
              data-testid="interface-name"
            >
              {iface.name || `#${iface.ifIndex}`}
            </button>
            <StatusPill
              tone={STATUS_TONE[iface.operStatus]}
              label={t(`interfaces.status.${iface.operStatus}`)}
            />
          </div>
          <div className="break-words text-xs text-text-muted">
            {iface.alias ? `${iface.targetName} · ${iface.alias}` : iface.targetName}
          </div>
        </div>
      ),
    },
    {
      key: 'traffic',
      header: t('interfaces.colTraffic'),
      sortValue: (iface) =>
        rateValue(iface, (r) => (r.inOctetsPerSec ?? 0) + (r.outOctetsPerSec ?? 0)),
      cell: (iface) =>
        rateCell(
          iface,
          (r) => `${bitRate(r.inOctetsPerSec)} / ${bitRate(r.outOctetsPerSec)}`,
          true,
        ),
    },
    {
      key: 'utilization',
      header: t('interfaces.colUtilization'),
      sortValue: (iface) =>
        rateValue(iface, (r) => Math.max(r.inUtilizationPct ?? 0, r.outUtilizationPct ?? 0)),
      cell: (iface) =>
        rateCell(iface, (r) => `${percent(r.inUtilizationPct)} / ${percent(r.outUtilizationPct)}`),
    },
    {
      key: 'errors',
      header: t('interfaces.colErrors'),
      sortValue: (iface) => rateValue(iface, (r) => r.inErrorsPerSec + r.outErrorsPerSec),
      cell: (iface) => {
        const errors = iface.rates ? iface.rates.inErrorsPerSec + iface.rates.outErrorsPerSec : 0;
        return (
          <span className={errors > 0 ? 'font-medium text-status-error-strong' : undefined}>
            {rateCell(iface, (r) => formatPerSecond(r.inErrorsPerSec + r.outErrorsPerSec))}
          </span>
        );
      },
    },
    {
      key: 'discards',
      header: t('interfaces.colDiscards'),
      sortValue: (iface) => rateValue(iface, (r) => r.inDiscardsPerSec + r.outDiscardsPerSec),
      cell: (iface) =>
        rateCell(iface, (r) => formatPerSecond(r.inDiscardsPerSec + r.outDiscardsPerSec)),
    },
  ];

  return (
    <>
      <p className="text-sm text-text-muted">
        {t('interfaces.count', { count: interfaces.length })} · {t('interfaces.history.openHint')}
      </p>
      {rated ? null : (
        <div data-testid="interfaces-no-data">
          <Alert tone="info">{t('interfaces.noData')}</Alert>
        </div>
      )}
      {open ? (
        <InterfaceHistory key={openKey} iface={open} onClose={(): void => setOpenKey(null)} />
      ) : null}
      <DataTable
        rows={interfaces}
        columns={columns}
        getRowKey={interfaceKey}
        caption={t('interfaces.title')}
        labels={labels}
        // The server already lists the highest error rate first; this is the
        // header's starting state, so the arrow shows on the column that ordered it.
        defaultSort={{ key: 'errors', direction: 'desc' }}
        pageSize={25}
        empty={null}
        data-testid="interfaces-table"
      />
    </>
  );
}
