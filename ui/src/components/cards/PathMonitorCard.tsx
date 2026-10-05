/**
 * PathMonitorCard — trace one destination continuously and accumulate per-hop
 * loss and latency (#165), beside the one-shot trace in PathDiscoveryCard.
 *
 * A single traceroute shows the path; repeating it shows which hop drops
 * packets or adds jitter. The card starts a `path-monitor` job and redraws its
 * hop table from every round until the operator stops it, the link drops, or
 * the page is left. Starting and
 * stopping need the operator role, so a viewer gets the reason instead of the
 * form.
 */

import { type JSX, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useRole } from '../../contexts/RoleContext';
import {
  type PathMonitorState,
  type PathMonitorUpdate,
  usePathMonitor,
} from '../../hooks/usePathMonitor';
import { cn, radius, spacing, status as statusColor } from '../../styles/theme';
import type { HopStats } from '../../types/generated/path-monitor-update';
import { Button } from '../ui/Button';
import { Card, type Status } from '../ui/Card';
import { Activity } from '../ui/Icons';
import { Input } from '../ui/Input';
import { formatRtt } from './pathDiscoveryHelpers';

function snapshotOf(state: PathMonitorState): PathMonitorUpdate | undefined {
  return state.phase === 'idle' || state.phase === 'starting' ? undefined : state.snapshot;
}

function lastHop(snapshot: PathMonitorUpdate | undefined): HopStats | undefined {
  return snapshot?.hops.at(-1);
}

function cardStatus(state: PathMonitorState): Status {
  switch (state.phase) {
    case 'starting':
    case 'running':
    case 'stopping':
      return 'loading';
    case 'failed':
      return 'error';
    case 'stopped':
      return (lastHop(state.snapshot)?.lossPct ?? 0) > 0 ? 'warning' : 'success';
    default:
      return 'unknown';
  }
}

function formatLoss(pct: number): string {
  return `${pct.toFixed(pct > 0 && pct < 10 ? 1 : 0)}%`;
}

export function PathMonitorCard({
  defaultTarget,
  linkDown,
  subscribe,
}: {
  /** The gateway, so the first run needs no typing; the field starts on it. */
  defaultTarget: string;
  /**
   * A link that drops mid-run stops the monitor, and the card says why. A link
   * already down at the start does not: the destination may be reached some
   * other way, and the operator asked for the run.
   */
  linkDown: boolean;
  subscribe: (handler: (update: PathMonitorUpdate) => void) => () => void;
}): JSX.Element {
  const { t } = useTranslation('cards');
  const { canWrite } = useRole();
  const { state, start, stop } = usePathMonitor(subscribe);

  // null until the operator edits the field, so a gateway that loads after
  // mount still fills it.
  const [target, setTarget] = useState<string | null>(null);
  const [linkSeenUp, setLinkSeenUp] = useState(false);
  const [stoppedOnLinkDown, setStoppedOnLinkDown] = useState(false);
  const destination = (target ?? defaultTarget).trim();
  const busy =
    state.phase === 'starting' || state.phase === 'running' || state.phase === 'stopping';
  const snapshot = snapshotOf(state);

  // Once per run: a stop that fails offers the retry button rather than
  // re-firing on every phase change.
  useEffect(() => {
    if (!linkDown) {
      setLinkSeenUp(true);
      return;
    }
    if (state.phase === 'running' && linkSeenUp && !stoppedOnLinkDown) {
      setStoppedOnLinkDown(true);
      void stop();
    }
  }, [linkDown, linkSeenUp, state.phase, stoppedOnLinkDown, stop]);

  return (
    <Card
      title={t('pathMonitor.title')}
      icon={<Activity className="w-4 h-4" />}
      status={cardStatus(state)}
      ariaLabel={t('pathMonitor.title')}
      // A hop table needs more than a quarter of a 4-up grid; see BonjourCard
      // for why max-w-none rides with the span.
      className="sm:col-span-2 max-w-none"
    >
      <div className="stack-sm" data-testid="path-monitor" data-phase={state.phase}>
        <p className="caption text-text-muted">{t('pathMonitor.description')}</p>

        {canWrite ? (
          <form
            className="stack-sm"
            onSubmit={(event) => {
              event.preventDefault();
              if (destination) {
                setStoppedOnLinkDown(false);
                setLinkSeenUp(!linkDown);
                void start({ destination });
              }
            }}
          >
            <Input
              id="path-monitor-target"
              label={t('pathMonitor.target')}
              value={target ?? defaultTarget}
              placeholder={t('pathMonitor.targetPlaceholder')}
              onChange={(event) => setTarget(event.target.value)}
              disabled={busy}
              data-testid="path-monitor-target"
            />
            {busy ? (
              <MonitorRunning state={state} target={destination} onStop={() => void stop()} />
            ) : (
              <Button
                type="submit"
                variant="solid"
                size="sm"
                disabled={!destination}
                data-testid="path-monitor-start"
              >
                {t('pathMonitor.start')}
              </Button>
            )}
          </form>
        ) : (
          <p className="body-small text-text-secondary" data-testid="path-monitor-read-only">
            {t('pathMonitor.readOnly')}
          </p>
        )}

        {state.phase === 'failed' ? (
          <div
            className={cn(spacing.pad.sm, statusColor.bg.errorSoft, radius.md, 'stack-xs')}
            role="alert"
            data-testid="path-monitor-failed"
          >
            <p className="body-small text-status-error-strong">{t('pathMonitor.failed')}</p>
            {state.error ? (
              <p
                className="caption font-mono text-text-secondary break-words"
                data-testid="path-monitor-error-detail"
              >
                {state.error}
              </p>
            ) : null}
          </div>
        ) : null}

        {stoppedOnLinkDown && state.phase === 'stopped' ? (
          <p
            className={cn('caption', statusColor.text.warning)}
            data-testid="path-monitor-link-down"
          >
            {t('pathMonitor.linkDown')}
          </p>
        ) : null}

        {snapshot ? <HopTable snapshot={snapshot} /> : null}
      </div>
    </Card>
  );
}

function MonitorRunning({
  state,
  target,
  onStop,
}: {
  state: PathMonitorState;
  target: string;
  onStop: () => void;
}): JSX.Element {
  const { t } = useTranslation('cards');

  return (
    <div className="stack-xs" data-testid="path-monitor-running">
      <p className="body-small text-text-secondary" aria-live="polite">
        {state.phase === 'starting'
          ? t('pathMonitor.starting')
          : t('pathMonitor.running', { target })}
      </p>
      <div className="flex items-center gap-compact">
        <Button
          type="button"
          variant="outline"
          tone="red"
          size="sm"
          onClick={onStop}
          disabled={state.phase !== 'running'}
          data-testid="path-monitor-stop"
        >
          {state.phase === 'stopping' ? t('pathMonitor.stopping') : t('pathMonitor.stop')}
        </Button>
        {state.phase === 'running' && state.stopFailed ? (
          <p className="caption text-status-error" role="alert">
            {t('pathMonitor.stopFailed')}
          </p>
        ) : null}
      </div>
    </div>
  );
}

interface Column {
  key: string;
  label: string;
  width: string;
  /** Hidden below sm so the table fits a 390 px phone without scrolling. */
  wide?: boolean;
}

function HopTable({ snapshot }: { snapshot: PathMonitorUpdate }): JSX.Element {
  const { t } = useTranslation('cards');
  // A phone keeps hop, address, loss, average and jitter: enough to name the
  // hop that drops packets or adds delay. Last, best and worst join from sm up.
  const columns: Column[] = [
    { key: 'ttl', label: t('pathMonitor.colHop'), width: 'w-[16%] sm:w-[9%]' },
    { key: 'address', label: t('pathMonitor.colAddress'), width: 'w-[30%] sm:w-[27%]' },
    { key: 'loss', label: t('pathMonitor.colLoss'), width: 'w-[18%] sm:w-[12%]' },
    { key: 'last', label: t('pathMonitor.colLast'), width: 'sm:w-[10%]', wide: true },
    { key: 'avg', label: t('pathMonitor.colAvg'), width: 'w-[18%] sm:w-[11%]' },
    { key: 'best', label: t('pathMonitor.colBest'), width: 'sm:w-[10%]', wide: true },
    { key: 'worst', label: t('pathMonitor.colWorst'), width: 'sm:w-[11%]', wide: true },
    { key: 'jitter', label: t('pathMonitor.colJitter'), width: 'w-[18%] sm:w-[10%]' },
  ];

  return (
    <div className="stack-xs" data-testid="path-monitor-result">
      <p className="caption text-text-muted" data-testid="path-monitor-rounds">
        {t('pathMonitor.rounds', { count: snapshot.rounds, target: snapshot.target })}
      </p>
      {snapshot.hops.length === 0 ? (
        <p className="caption text-text-muted" data-testid="path-monitor-empty">
          {t('pathMonitor.empty')}
        </p>
      ) : (
        // See NeighbourCacheCard: table-fixed, not overflow-x-auto (#2708).
        <div className={cn(radius.default, 'border border-surface-border')}>
          <table className="w-full table-fixed body-small" data-testid="path-monitor-hops">
            <thead>
              <tr className="border-b border-surface-border text-text-muted">
                {columns.map((column) => (
                  <th
                    key={column.key}
                    scope="col"
                    className={cn(
                      'px-cell py-row text-left truncate',
                      column.width,
                      column.wide && 'hidden sm:table-cell',
                    )}
                    title={column.label}
                  >
                    {column.label}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {snapshot.hops.map((hop) => (
                <HopRow key={hop.ttl} hop={hop} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function HopRow({ hop }: { hop: HopStats }): JSX.Element {
  const { t } = useTranslation('cards');
  const [primary, ...others] = hop.addresses;
  const name = primary ? (primary.hostname ?? primary.ip) : '*';
  const address =
    others.length > 0 ? t('pathMonitor.moreAddresses', { name, count: others.length }) : name;
  const silent = hop.received === 0;
  const cells: { key: string; value: string; wide?: boolean; mono?: boolean }[] = [
    { key: 'ttl', value: String(hop.ttl) },
    { key: 'address', value: address, mono: true },
    { key: 'loss', value: formatLoss(hop.lossPct) },
    { key: 'last', value: silent ? '---' : formatRtt(hop.lastRtt), wide: true },
    { key: 'avg', value: silent ? '---' : formatRtt(hop.avgRtt) },
    { key: 'best', value: silent ? '---' : formatRtt(hop.bestRtt), wide: true },
    { key: 'worst', value: silent ? '---' : formatRtt(hop.worstRtt), wide: true },
    { key: 'jitter', value: silent ? '---' : formatRtt(hop.jitter) },
  ];
  const addressTitle = hop.addresses.map((a) => (a.hostname ? `${a.hostname} (${a.ip})` : a.ip));

  return (
    <tr
      className="border-b border-surface-border last:border-0"
      data-testid="path-monitor-hop"
      data-ttl={hop.ttl}
    >
      {cells.map((cell) => (
        <td
          key={cell.key}
          className={cn(
            'px-cell py-row truncate',
            cell.mono && 'font-mono',
            cell.key === 'loss' && hop.lossPct > 0 && statusColor.text.warning,
            cell.wide && 'hidden sm:table-cell',
          )}
          title={
            cell.key === 'address' && addressTitle.length > 0 ? addressTitle.join(', ') : cell.value
          }
          data-testid={`path-monitor-hop-${cell.key}`}
        >
          {cell.value}
        </td>
      ))}
    </tr>
  );
}
