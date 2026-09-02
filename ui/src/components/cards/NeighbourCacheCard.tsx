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
import { Card, type Status } from '../ui/card';
import { Network } from '../ui/icons';

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
    >
      <div className="stack-sm">
        <p className="caption text-text-muted">{t('neighbours.description')}</p>

        {error ? <p className="caption text-status-error">{error}</p> : null}

        {!loading && !error && entries.length === 0 ? (
          <p className="caption text-text-muted">{t('neighbours.empty')}</p>
        ) : null}

        {entries.length > 0 ? (
          <div className={cn('overflow-x-auto', radius.default, 'border border-surface-border')}>
            <table className="w-full body-small">
              <caption className="sr-only">{t('neighbours.tableCaption')}</caption>
              <thead>
                <tr className="border-b border-surface-border text-text-muted">
                  <th scope="col" className="px-cell py-row text-left">
                    {t('neighbours.address')}
                  </th>
                  <th scope="col" className="px-cell py-row text-left">
                    {t('neighbours.mac')}
                  </th>
                  <th scope="col" className="px-cell py-row text-left">
                    {t('neighbours.vendor')}
                  </th>
                  <th scope="col" className="px-cell py-row text-left">
                    {t('neighbours.interface')}
                  </th>
                  <th scope="col" className="px-cell py-row text-left">
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
                    <td className={cn('px-cell py-row font-mono', spacing.gap.tight)}>
                      {entry.ip}
                      {/* The family is on the row rather than inferred from the
                          address, so an IPv4 and an IPv6 entry are
                          distinguishable at a glance and by a screen reader. */}
                      <span className="caption text-text-muted ml-inline">
                        {entry.family === 'ipv6' ? 'IPv6' : 'IPv4'}
                      </span>
                    </td>
                    <td className="px-cell py-row font-mono">{entry.mac || '—'}</td>
                    <td className="px-cell py-row">{entry.vendor || '—'}</td>
                    <td className="px-cell py-row">{entry.interface || '—'}</td>
                    <td className="px-cell py-row">{entry.state || '—'}</td>
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
