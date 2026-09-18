/**
 * NeighbourCacheCard — this device's own ARP and NDP neighbour cache (#328).
 *
 * The reader has existed cross-platform for a long time; the entries were
 * folded into device discovery and never surfaced on their own. This is the
 * exposure.
 *
 * Deliberately distinct from the topology view's ARP table, which shows what a
 * *remote* switch reports over SNMP. This shows what this box sees on the wire
 * in front of it — the thing an operator wants when an IP will not resolve to a
 * MAC on the segment they are plugged into.
 */

import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import { useNeighbourCache } from '../../hooks/useNeighbourCache';
import { cn, radius, spacing } from '../../styles/theme';
import { Button } from '../ui/Button';
import { Card, type Status } from '../ui/Card';
import { Network } from '../ui/Icons';

export function NeighbourCacheCard(): JSX.Element {
  const { t } = useTranslation('cards');
  const { entries, loading, error, refresh } = useNeighbourCache();

  let cardStatus: Status = 'success';
  if (loading) {
    cardStatus = 'loading';
  } else if (error) {
    cardStatus = 'error';
  } else if (entries.length === 0) {
    cardStatus = 'unknown';
  }

  return (
    <Card
      title={t('neighbours.title')}
      icon={<Network className="w-4 h-4" />}
      status={cardStatus}
      ariaLabel={t('neighbours.title')}
      // Two grid columns: this is a device list, not a single-reading
      // facet, and at a quarter of a 4-up grid it is 202 px wide (#2708).
      className="sm:col-span-2"
    >
      <div className="stack-sm">
        <p className="caption text-text-muted">{t('neighbours.description')}</p>

        {error ? <p className="caption text-status-error">{error}</p> : null}

        {!loading && !error && entries.length === 0 ? (
          <p className="caption text-text-muted">{t('neighbours.empty')}</p>
        ) : null}

        {entries.length > 0 ? (
          // No overflow-x-auto: table-fixed makes fitting the card structural
          // rather than a tuning exercise, so a long vendor string or a full
          // IPv6 address cannot push the last column off-card the way it did
          // at 1280 and 1440 px (#2708). What does not fit is truncated and
          // carries the full text in `title`.
          <div className={cn(radius.default, 'border border-surface-border')}>
            <table className="w-full table-fixed body-small">
              <caption className="sr-only">{t('neighbours.tableCaption')}</caption>
              <thead>
                <tr className="border-b border-surface-border text-text-muted">
                  <th
                    scope="col"
                    className="px-cell py-row text-left w-[30%] truncate"
                    title={t('neighbours.address')}
                  >
                    {t('neighbours.address')}
                  </th>
                  <th
                    scope="col"
                    className="px-cell py-row text-left w-[26%] truncate"
                    title={t('neighbours.mac')}
                  >
                    {t('neighbours.mac')}
                  </th>
                  <th
                    scope="col"
                    className="px-cell py-row text-left w-[20%] truncate"
                    title={t('neighbours.vendor')}
                  >
                    {t('neighbours.vendor')}
                  </th>
                  <th
                    scope="col"
                    className="px-cell py-row text-left w-[12%] truncate"
                    title={t('neighbours.interface')}
                  >
                    {t('neighbours.interface')}
                  </th>
                  <th
                    scope="col"
                    className="px-cell py-row text-left w-[12%] truncate"
                    title={t('neighbours.state')}
                  >
                    {t('neighbours.state')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {entries.map((entry) => (
                  <tr
                    key={`${entry.ip}-${entry.interface ?? ''}`}
                    className="border-b border-surface-border last:border-b-0"
                  >
                    <td className="px-cell py-row font-mono">
                      {/* A table cell is not a flex container, so the address
                          and its tag need one: truncating the address makes it
                          a block, which would otherwise drop the tag onto a
                          second line and double the height of every row. */}
                      <div className={cn('flex items-baseline', spacing.gap.tight)}>
                        {/* The address truncates and the family tag does not:
                            a clipped "IPv" tells the reader nothing, and it is
                            the tag that makes the row scannable. */}
                        <span className="truncate min-w-0" title={entry.ip}>
                          {entry.ip}
                        </span>
                        {/* The family is on the row rather than inferred from the
                            address, so an IPv4 and an IPv6 entry are
                            distinguishable at a glance and by a screen reader. */}
                        <span className="caption text-text-muted shrink-0">
                          {entry.family === 'ipv6' ? 'IPv6' : 'IPv4'}
                        </span>
                      </div>
                    </td>
                    <td
                      className="px-cell py-row font-mono truncate"
                      title={entry.mac || undefined}
                    >
                      {entry.mac || '—'}
                    </td>
                    <td className="px-cell py-row truncate" title={entry.vendor || undefined}>
                      {entry.vendor || '—'}
                    </td>
                    <td className="px-cell py-row truncate" title={entry.interface || undefined}>
                      {entry.interface || '—'}
                    </td>
                    <td className="px-cell py-row truncate" title={entry.state || undefined}>
                      {entry.state || '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}

        <Button
          variant="ghost"
          size="sm"
          onClick={(): void => {
            refresh().catch(() => undefined);
          }}
          loading={loading}
          className="self-start"
        >
          {t('neighbours.refresh')}
        </Button>
      </div>
    </Card>
  );
}
