/**
 * FlowsPage — the flow explorer (UI-SEED-23, P-C): the hosts, host pairs and
 * applications that carried the most traffic over a window, from the flows
 * exporters send to Seed's collector.
 *
 * The window and the ranking apply to all three lists. When the licence
 * keeps less than the chosen window, the server serves what it keeps and the
 * page says so rather than implying the longer window was empty.
 */

import { type JSX, type ReactNode, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ArrowLeftRight } from '../components/ui/Icons';
import {
  FLOW_RANGES,
  FLOW_RANKS,
  type FlowRange,
  type FlowRank,
  type FlowTops,
  useFlowTops,
} from '../hooks/useFlowTops';
import { useLocale } from '../hooks/useLocale';
import { useTableLabels } from '../hooks/useTableLabels';
import { formatBytes } from '../lib/format';
import { Alert } from '../shared/Alert';
import { DataTable, type DataTableColumn } from '../shared/DataTable';
import { EmptyState } from '../shared/EmptyState';
import { FormField } from '../shared/FormField';
import { Select } from '../shared/Select';

/** IANA protocol numbers worth a name; anything else shows as "IP n". */
const PROTOCOL_NAMES: Record<number, string> = {
  1: 'ICMP',
  6: 'TCP',
  17: 'UDP',
  47: 'GRE',
  50: 'ESP',
  58: 'ICMPv6',
  132: 'SCTP',
};

/** The application name the server gives traffic no signature matched. */
const UNKNOWN_APPLICATION = 'unknown';

/** The option a select reported, narrowed back to its union. */
function pick<T extends string>(options: readonly T[], value: string, current: T): T {
  return options.find((o) => o === value) ?? current;
}

interface TopTableProps<T extends { bytes: number; packets: number }> {
  testId: string;
  title: string;
  /** Null while the first read is in flight. */
  rows: readonly T[] | null;
  getRowKey: (row: T) => string;
  /** The leading column(s): who or what the counters belong to. */
  lead: readonly DataTableColumn<T>[];
}

function TopTable<T extends { bytes: number; packets: number }>({
  testId,
  title,
  rows,
  getRowKey,
  lead,
}: TopTableProps<T>): JSX.Element {
  const { t } = useTranslation('pages');
  const locale = useLocale();
  const labels = useTableLabels<T>({ loading: t('flows.loading', { name: title }) });
  const packets = new Intl.NumberFormat(locale);
  const columns: DataTableColumn<T>[] = [
    ...lead,
    {
      key: 'bytes',
      header: t('flows.colBytes'),
      cell: (row) => formatBytes(row.bytes),
      sortValue: (row) => row.bytes,
      align: 'right',
    },
    {
      key: 'packets',
      header: t('flows.colPackets'),
      cell: (row) => packets.format(row.packets),
      sortValue: (row) => row.packets,
      align: 'right',
    },
  ];
  return (
    <section className="min-w-0" data-testid={testId}>
      <h2 className="mb-2 text-sm font-semibold text-text-primary">{title}</h2>
      <DataTable
        rows={rows ?? []}
        columns={columns}
        getRowKey={getRowKey}
        caption={title}
        loading={rows === null}
        labels={labels}
        pageSize={10}
        empty={
          <div data-testid={`${testId}-none`}>
            <EmptyState
              icon={<ArrowLeftRight className="h-6 w-6" aria-hidden="true" />}
              title={t('flows.none')}
            />
          </div>
        }
      />
    </section>
  );
}

function FlowTables({ tops }: { tops: FlowTops | null }): JSX.Element {
  const { t } = useTranslation('pages');
  const protocol = (n: number): string =>
    PROTOCOL_NAMES[n] ?? t('flows.protocolNumber', { number: n });
  type Talker = FlowTops['talkers']['talkers'][number];
  type Application = FlowTops['applications']['applications'][number];
  type Conversation = FlowTops['conversations']['conversations'][number];

  return (
    <div className="grid gap-4 xl:grid-cols-2">
      <TopTable<Talker>
        testId="flows-talkers"
        title={t('flows.talkers')}
        rows={tops?.talkers.talkers ?? null}
        getRowKey={(h) => h.addr}
        lead={[
          {
            key: 'host',
            header: t('flows.colHost'),
            cell: (h) => <span className="break-all">{h.addr}</span>,
            sortValue: (h) => h.addr,
          },
        ]}
      />
      <TopTable<Application>
        testId="flows-applications"
        title={t('flows.applications')}
        rows={tops?.applications.applications ?? null}
        getRowKey={(a) => a.name}
        lead={[
          {
            key: 'application',
            header: t('flows.colApplication'),
            cell: (a) => (a.name === UNKNOWN_APPLICATION ? t('flows.unknownApplication') : a.name),
            sortValue: (a) =>
              a.name === UNKNOWN_APPLICATION ? t('flows.unknownApplication') : a.name,
          },
        ]}
      />
      <div className="xl:col-span-2">
        <TopTable<Conversation>
          testId="flows-conversations"
          title={t('flows.conversations')}
          rows={tops?.conversations.conversations ?? null}
          getRowKey={(c) => `${c.addrA}|${c.addrB}|${c.protocol}`}
          lead={[
            {
              key: 'hosts',
              header: t('flows.colHosts'),
              cell: (c) => <span className="break-all">{`${c.addrA} ↔ ${c.addrB}`}</span>,
              sortValue: (c) => `${c.addrA} ${c.addrB}`,
            },
            {
              key: 'protocol',
              header: t('flows.colProtocol'),
              cell: (c) => protocol(c.protocol),
              sortValue: (c) => protocol(c.protocol),
            },
          ]}
        />
      </div>
    </div>
  );
}

export function FlowsPage(): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const [range, setRange] = useState<FlowRange>('1d');
  const [by, setBy] = useState<FlowRank>('bytes');
  const { tops, loading, error } = useFlowTops(range, by);

  const empty =
    tops !== null &&
    tops.talkers.talkers.length === 0 &&
    tops.conversations.conversations.length === 0 &&
    tops.applications.applications.length === 0;
  const served = tops?.talkers.window;

  let body: ReactNode;
  if (error) {
    body = (
      <div data-testid="flows-error">
        <Alert tone="error">{error}</Alert>
      </div>
    );
  } else if (empty) {
    body = (
      <div data-testid="flows-empty">
        <EmptyState
          icon={<ArrowLeftRight className="h-6 w-6" aria-hidden="true" />}
          title={t('flows.emptyTitle')}
          hint={t('flows.emptyHint')}
        />
      </div>
    );
  } else {
    body = <FlowTables tops={tops} />;
  }

  return (
    <>
      <div className="flex flex-wrap items-end gap-3">
        <FormField label={t('flows.windowLabel')} className="w-48">
          {(control) => (
            <Select
              {...control}
              value={range}
              onValueChange={(value): void => setRange(pick(FLOW_RANGES, value, range))}
              options={FLOW_RANGES.map((r) => ({ value: r, label: t(`flows.window.${r}`) }))}
              data-testid="flows-range"
            />
          )}
        </FormField>
        <FormField label={t('flows.rankLabel')} className="w-48">
          {(control) => (
            <Select
              {...control}
              value={by}
              onValueChange={(value): void => setBy(pick(FLOW_RANKS, value, by))}
              options={FLOW_RANKS.map((r) => ({ value: r, label: t(`flows.rank.${r}`) }))}
              data-testid="flows-rank"
            />
          )}
        </FormField>
        {loading && tops !== null ? (
          <span className="text-sm text-text-muted" data-testid="flows-refreshing">
            {t('common:status.loading')}
          </span>
        ) : null}
      </div>
      {served?.clamped ? (
        <div data-testid="flows-clamped">
          <Alert tone="info">{t('flows.clamped', { count: served.days })}</Alert>
        </div>
      ) : null}
      {body}
    </>
  );
}
