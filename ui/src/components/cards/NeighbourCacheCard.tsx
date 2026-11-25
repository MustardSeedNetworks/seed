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

import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useNeighbourCache } from '../../hooks/useNeighbourCache';
import { useTableLabels } from '../../hooks/useTableLabels';
import { DataTable, type DataTableColumn } from '../../shared/DataTable';
import { EmptyState } from '../../shared/EmptyState';
import { SearchInput } from '../../shared/SearchInput';
import type { NeighbourEntry } from '../../types/generated/neighbour-cache-response';
import { Button } from '../ui/Button';
import { Card, type Status } from '../ui/Card';
import { Network } from '../ui/Icons';

export function NeighbourCacheCard(): JSX.Element {
  const { t } = useTranslation(['cards', 'common']);
  const { entries, loading, error, refresh } = useNeighbourCache();
  const [filter, setFilter] = useState('');
  const labels = useTableLabels<NeighbourEntry>({ loading: t('neighbours.title') });

  const needle = filter.trim().toLowerCase();
  const visible = entries.filter((entry) =>
    [entry.ip, entry.mac, entry.vendor, entry.interface, entry.state].some((value) =>
      value?.toLowerCase().includes(needle),
    ),
  );

  const columns: DataTableColumn<NeighbourEntry>[] = [
    {
      key: 'address',
      header: t('neighbours.address'),
      sortValue: (entry) => entry.ip,
      cell: (entry) => (
        <span className="inline-flex items-baseline gap-1 font-mono">
          <span>{entry.ip}</span>
          {/* The family is on the row rather than inferred from the address,
              so an IPv4 and an IPv6 entry are distinguishable at a glance and
              by a screen reader. */}
          <span className="text-xs text-text-muted">
            {entry.family === 'ipv6' ? t('neighbours.familyIpv6') : t('neighbours.familyIpv4')}
          </span>
        </span>
      ),
    },
    {
      key: 'mac',
      header: t('neighbours.mac'),
      sortValue: (entry) => entry.mac,
      cell: (entry) => <span className="font-mono">{entry.mac || '—'}</span>,
    },
    {
      key: 'vendor',
      header: t('neighbours.vendor'),
      sortValue: (entry) => entry.vendor ?? '',
      cell: (entry) => entry.vendor || '—',
    },
    {
      key: 'interface',
      header: t('neighbours.interface'),
      sortValue: (entry) => entry.interface ?? '',
      cell: (entry) => entry.interface || '—',
    },
    {
      key: 'state',
      header: t('neighbours.state'),
      sortValue: (entry) => entry.state ?? '',
      cell: (entry) => entry.state || '—',
    },
  ];

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
      // max-w-none so the two columns this asks for are two columns wide:
      // Card's default max-w-sm otherwise capped the span at one card,
      // leaving the rest of it blank (UI-SEED-20).
      className="sm:col-span-2 max-w-none"
    >
      <div className="stack-sm">
        <p className="caption text-text-muted">{t('neighbours.description')}</p>

        {error ? <p className="caption text-status-error">{error}</p> : null}

        {!loading && !error && entries.length === 0 ? (
          <p className="caption text-text-muted">{t('neighbours.empty')}</p>
        ) : null}

        {entries.length > 0 ? (
          <>
            <SearchInput
              value={filter}
              onValueChange={setFilter}
              label={t('common:dataTable.filterRows')}
              clearLabel={t('common:dataTable.clearFilter')}
            />
            <DataTable
              rows={visible}
              columns={columns}
              getRowKey={(entry) => `${entry.ip}-${entry.interface ?? ''}`}
              caption={t('neighbours.tableCaption')}
              empty={<EmptyState title={t('common:dataTable.noMatches')} />}
              labels={labels}
              pageSize={10}
              defaultSort={{ key: 'address', direction: 'asc' }}
            />
          </>
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
