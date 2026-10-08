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
import { ChevronDown, ChevronUp } from '../components/ui/Icons';
import { AccentLink } from '../components/ui/Typography';
import { useInterfaceStats } from '../hooks/useInterfaceStats';
import { formatBitRate, formatPerSecond } from '../lib/format';
import type {
  InterfaceRatesResponse,
  InterfaceStatsResponse,
} from '../types/generated/interface-stats-list-response';
import { InterfaceHistory } from './interfaces/InterfaceHistory';

type SortKey = 'interface' | 'traffic' | 'utilization' | 'errors' | 'discards';
type SortDir = 'asc' | 'desc';
type OperStatus = InterfaceStatsResponse['operStatus'];

const BITS_PER_OCTET = 8;

function interfaceKey(iface: InterfaceStatsResponse): string {
  return `${iface.targetId}/${iface.ifIndex}`;
}

/** The number a rate column sorts by; undefined when the interface has none. */
function sortValue(iface: InterfaceStatsResponse, key: SortKey): number | undefined {
  const r = iface.rates;
  if (!r) {
    return undefined;
  }
  switch (key) {
    case 'traffic':
      return (r.inOctetsPerSec ?? 0) + (r.outOctetsPerSec ?? 0);
    case 'utilization':
      return Math.max(r.inUtilizationPct ?? 0, r.outUtilizationPct ?? 0);
    case 'errors':
      return r.inErrorsPerSec + r.outErrorsPerSec;
    case 'discards':
      return r.inDiscardsPerSec + r.outDiscardsPerSec;
    case 'interface':
      return undefined;
  }
}

/**
 * Sorts a copy of the list. An interface with no rate yet sorts after every
 * rated one in either direction: it has no value to rank.
 */
function sortInterfaces(
  list: InterfaceStatsResponse[],
  key: SortKey,
  dir: SortDir,
): InterfaceStatsResponse[] {
  const sign = dir === 'asc' ? 1 : -1;
  return [...list].sort((a, b) => {
    if (key === 'interface') {
      return (
        sign *
        (a.targetName.localeCompare(b.targetName) ||
          a.name.localeCompare(b.name, undefined, { numeric: true }))
      );
    }
    const av = sortValue(a, key);
    const bv = sortValue(b, key);
    if (av === undefined || bv === undefined) {
      return (av === undefined ? 1 : 0) - (bv === undefined ? 1 : 0);
    }
    return sign * (av - bv);
  });
}

const STATUS_DOT: Record<OperStatus, string> = {
  up: 'bg-status-success',
  down: 'bg-status-error',
  lowerLayerDown: 'bg-status-error',
  testing: 'bg-status-warning',
  dormant: 'bg-text-muted',
  notPresent: 'bg-text-muted',
  unknown: 'bg-text-muted',
};

function bitRate(octets: number | undefined): string {
  return octets === undefined ? '—' : formatBitRate(octets * BITS_PER_OCTET);
}

function percent(pct: number | undefined): string {
  return pct === undefined ? '—' : `${Number.parseFloat(pct.toFixed(1))}%`;
}

function RateCells({ rates }: { rates: InterfaceRatesResponse | undefined }): JSX.Element {
  const { t } = useTranslation('pages');
  if (!rates) {
    return (
      <td colSpan={4} className="px-3 py-2 text-text-muted">
        {t('interfaces.noRateYet')}
      </td>
    );
  }
  const errors = rates.inErrorsPerSec + rates.outErrorsPerSec;
  return (
    <>
      <td className="hidden px-3 py-2 text-text-secondary md:table-cell">
        {bitRate(rates.inOctetsPerSec)} / {bitRate(rates.outOctetsPerSec)}
      </td>
      <td className="px-3 py-2 text-text-secondary">
        {percent(rates.inUtilizationPct)} / {percent(rates.outUtilizationPct)}
      </td>
      <td
        className={`px-3 py-2 ${errors > 0 ? 'font-medium text-status-error-strong' : 'text-text-secondary'}`}
      >
        {formatPerSecond(errors)}
      </td>
      <td className="hidden px-3 py-2 text-text-secondary sm:table-cell">
        {formatPerSecond(rates.inDiscardsPerSec + rates.outDiscardsPerSec)}
      </td>
    </>
  );
}

interface SortHeaderProps {
  column: SortKey;
  label: string;
  sort: { key: SortKey; dir: SortDir };
  onSort: (key: SortKey) => void;
  className?: string;
}

function SortHeader({ column, label, sort, onSort, className }: SortHeaderProps): JSX.Element {
  const active = sort.key === column;
  const ariaSort = active ? (sort.dir === 'asc' ? 'ascending' : 'descending') : 'none';
  const Icon = sort.dir === 'asc' ? ChevronUp : ChevronDown;
  return (
    <th scope="col" aria-sort={ariaSort} className={`px-3 py-2 ${className ?? ''}`}>
      <button
        type="button"
        onClick={(): void => onSort(column)}
        data-testid={`interfaces-sort-${column}`}
        className="inline-flex min-h-6 items-center gap-1 text-left uppercase tracking-wide hover:text-text-primary"
      >
        {label}
        {active ? <Icon className="h-3 w-3" aria-hidden="true" /> : null}
      </button>
    </th>
  );
}

export function InterfacesPage(): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const { interfaces, loading, error } = useInterfaceStats();
  // The server already lists the highest error rate first; this is the
  // header's starting state, so the arrow shows on the column that ordered it.
  const [sort, setSort] = useState<{ key: SortKey; dir: SortDir }>({
    key: 'errors',
    dir: 'desc',
  });
  const [openKey, setOpenKey] = useState<string | null>(null);

  const onSort = (key: SortKey): void => {
    setSort((prev) =>
      prev.key === key
        ? { key, dir: prev.dir === 'asc' ? 'desc' : 'asc' }
        : { key, dir: key === 'interface' ? 'asc' : 'desc' },
    );
  };

  if (error) {
    return (
      <div
        role="alert"
        data-testid="interfaces-error"
        className="rounded-md border border-status-error/40 bg-status-error/10 pad-sm text-sm text-status-error-strong"
      >
        {error}
      </div>
    );
  }
  if (loading) {
    return (
      <p className="body-small" data-testid="interfaces-loading">
        {t('common:status.loading')}
      </p>
    );
  }
  if (interfaces.length === 0) {
    return (
      <div
        data-testid="interfaces-empty"
        className="rounded-lg border border-surface-border bg-surface-raised pad text-sm text-text-secondary"
      >
        <p>{t('interfaces.empty')}</p>
        <AccentLink to="/polling-targets" className="mt-2 inline-flex min-h-6 items-center">
          {t('interfaces.addTarget')}
        </AccentLink>
      </div>
    );
  }

  const rows = sortInterfaces(interfaces, sort.key, sort.dir);
  const rated = interfaces.some((iface) => iface.rates);
  const open = interfaces.find((iface) => interfaceKey(iface) === openKey);

  return (
    <>
      <p className="body-small">
        {t('interfaces.count', { count: interfaces.length })} · {t('interfaces.history.openHint')}
      </p>
      {rated ? null : (
        <p
          data-testid="interfaces-no-data"
          className="rounded-md border border-surface-border bg-surface-sunken pad-sm text-sm text-text-secondary"
        >
          {t('interfaces.noData')}
        </p>
      )}
      {open ? (
        <InterfaceHistory key={openKey} iface={open} onClose={(): void => setOpenKey(null)} />
      ) : null}
      <div className="rounded-lg border border-surface-border bg-surface-raised">
        <table className="w-full text-sm" data-testid="interfaces-table">
          <caption className="sr-only">{t('interfaces.title')}</caption>
          <thead className="text-left text-xs text-text-muted">
            <tr>
              <SortHeader
                column="interface"
                label={t('interfaces.colInterface')}
                sort={sort}
                onSort={onSort}
              />
              <SortHeader
                column="traffic"
                label={t('interfaces.colTraffic')}
                sort={sort}
                onSort={onSort}
                className="hidden md:table-cell"
              />
              <SortHeader
                column="utilization"
                label={t('interfaces.colUtilization')}
                sort={sort}
                onSort={onSort}
              />
              <SortHeader
                column="errors"
                label={t('interfaces.colErrors')}
                sort={sort}
                onSort={onSort}
              />
              <SortHeader
                column="discards"
                label={t('interfaces.colDiscards')}
                sort={sort}
                onSort={onSort}
                className="hidden sm:table-cell"
              />
            </tr>
          </thead>
          <tbody className="divide-y divide-surface-border">
            {rows.map((iface) => (
              <tr key={interfaceKey(iface)} data-testid="interface-row">
                <td className="px-3 py-2">
                  <div className="flex items-center gap-2">
                    <span
                      className={`h-2 w-2 shrink-0 rounded-full ${STATUS_DOT[iface.operStatus]}`}
                      aria-hidden="true"
                    />
                    <button
                      type="button"
                      onClick={(): void => setOpenKey(interfaceKey(iface))}
                      aria-expanded={openKey === interfaceKey(iface)}
                      className="min-h-6 break-all text-left font-medium text-text-primary underline-offset-2 hover:underline"
                      data-testid="interface-name"
                    >
                      {iface.name || `#${iface.ifIndex}`}
                    </button>
                    {/* Up is the expected case; anything else is spelled out, not left to colour. */}
                    <span
                      className={
                        iface.operStatus === 'up' ? 'sr-only' : 'text-xs text-text-secondary'
                      }
                    >
                      {t(`interfaces.status.${iface.operStatus}`)}
                    </span>
                  </div>
                  <div className="break-words text-xs text-text-muted">
                    {iface.alias ? `${iface.targetName} · ${iface.alias}` : iface.targetName}
                  </div>
                </td>
                <RateCells rates={iface.rates} />
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}
