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
import {
  FLOW_RANGES,
  FLOW_RANKS,
  type FlowRange,
  type FlowRank,
  type FlowTops,
  useFlowTops,
} from '../hooks/useFlowTops';
import { useLocale } from '../hooks/useLocale';
import { formatBytes } from '../lib/format';
import { cn, input } from '../styles/theme';

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

const selectClass = cn(input.base, input.state.default, input.size.sm, 'body-small');

interface ColumnTable {
  testId: string;
  title: string;
  head: string[];
  rows: { key: string; cells: ReactNode[] }[];
}

function TopTable({ testId, title, head, rows }: ColumnTable): JSX.Element {
  const { t } = useTranslation('pages');
  return (
    <section
      className="rounded-lg border border-surface-border bg-surface-raised"
      data-testid={testId}
    >
      <h2 className="border-b border-surface-border px-3 py-2 text-sm font-semibold text-text-primary">
        {title}
      </h2>
      {rows.length === 0 ? (
        <p className="px-3 py-2 text-sm text-text-muted" data-testid={`${testId}-none`}>
          {t('flows.none')}
        </p>
      ) : (
        <table className="w-full text-sm">
          <caption className="sr-only">{title}</caption>
          <thead className="text-left text-xs uppercase tracking-wide text-text-muted">
            <tr>
              {head.map((label, i) => (
                <th
                  key={label}
                  scope="col"
                  className={cn('px-3 py-2', i >= head.length - 2 && 'text-right')}
                >
                  {label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-surface-border">
            {rows.map((row) => (
              <tr key={row.key} data-testid={`${testId}-row`}>
                {row.cells.map((cell, i) => (
                  <td
                    key={head[i]}
                    className={cn(
                      'px-3 py-2',
                      i >= row.cells.length - 2
                        ? 'text-right tabular-nums text-text-secondary'
                        : 'break-all text-text-primary',
                    )}
                  >
                    {cell}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}

function FlowTables({ tops }: { tops: FlowTops }): JSX.Element {
  const { t } = useTranslation('pages');
  const locale = useLocale();
  const packets = new Intl.NumberFormat(locale);
  const counters = (bytes: number, count: number): ReactNode[] => [
    formatBytes(bytes),
    packets.format(count),
  ];
  const protocol = (n: number): string =>
    PROTOCOL_NAMES[n] ?? t('flows.protocolNumber', { number: n });

  return (
    <div className="grid gap-4 xl:grid-cols-2">
      <TopTable
        testId="flows-talkers"
        title={t('flows.talkers')}
        head={[t('flows.colHost'), t('flows.colBytes'), t('flows.colPackets')]}
        rows={tops.talkers.talkers.map((h) => ({
          key: h.addr,
          cells: [h.addr, ...counters(h.bytes, h.packets)],
        }))}
      />
      <TopTable
        testId="flows-applications"
        title={t('flows.applications')}
        head={[t('flows.colApplication'), t('flows.colBytes'), t('flows.colPackets')]}
        rows={tops.applications.applications.map((a) => ({
          key: a.name,
          cells: [
            a.name === UNKNOWN_APPLICATION ? t('flows.unknownApplication') : a.name,
            ...counters(a.bytes, a.packets),
          ],
        }))}
      />
      <div className="xl:col-span-2">
        <TopTable
          testId="flows-conversations"
          title={t('flows.conversations')}
          head={[
            t('flows.colHosts'),
            t('flows.colProtocol'),
            t('flows.colBytes'),
            t('flows.colPackets'),
          ]}
          rows={tops.conversations.conversations.map((c) => ({
            key: `${c.addrA}|${c.addrB}|${c.protocol}`,
            cells: [
              `${c.addrA} ↔ ${c.addrB}`,
              protocol(c.protocol),
              ...counters(c.bytes, c.packets),
            ],
          }))}
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

  let body: JSX.Element;
  if (error) {
    body = (
      <div
        role="alert"
        data-testid="flows-error"
        className="rounded-md border border-status-error/40 bg-status-error/10 pad-sm text-sm text-status-error-strong"
      >
        {error}
      </div>
    );
  } else if (tops === null) {
    body = (
      <p className="body-small" data-testid="flows-loading">
        {t('common:status.loading')}
      </p>
    );
  } else if (empty) {
    body = (
      <p
        data-testid="flows-empty"
        className="rounded-lg border border-surface-border bg-surface-raised pad text-sm text-text-secondary"
      >
        {t('flows.empty')}
      </p>
    );
  } else {
    body = <FlowTables tops={tops} />;
  }

  return (
    <>
      <div className="flex flex-wrap items-end gap-3">
        <label className="flex flex-col gap-1 text-xs text-text-muted">
          {t('flows.windowLabel')}
          <select
            value={range}
            onChange={(e): void => setRange(pick(FLOW_RANGES, e.target.value, range))}
            data-testid="flows-range"
            className={selectClass}
          >
            {FLOW_RANGES.map((r) => (
              <option key={r} value={r}>
                {t(`flows.window.${r}`)}
              </option>
            ))}
          </select>
        </label>
        <label className="flex flex-col gap-1 text-xs text-text-muted">
          {t('flows.rankLabel')}
          <select
            value={by}
            onChange={(e): void => setBy(pick(FLOW_RANKS, e.target.value, by))}
            data-testid="flows-rank"
            className={selectClass}
          >
            {FLOW_RANKS.map((r) => (
              <option key={r} value={r}>
                {t(`flows.rank.${r}`)}
              </option>
            ))}
          </select>
        </label>
        {loading && tops !== null ? (
          <span className="body-small" data-testid="flows-refreshing">
            {t('common:status.loading')}
          </span>
        ) : null}
      </div>
      {served?.clamped ? (
        <p
          data-testid="flows-clamped"
          className="rounded-md border border-surface-border bg-surface-sunken pad-sm text-sm text-text-secondary"
        >
          {t('flows.clamped', { count: served.days })}
        </p>
      ) : null}
      {body}
    </>
  );
}
