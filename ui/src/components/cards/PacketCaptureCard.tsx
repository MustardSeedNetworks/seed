/**
 * PacketCaptureCard — record traffic on one interface to a pcap file, read
 * its summary, and download it (#326, #239).
 *
 * A capture holds whatever crossed the wire, credentials sent in the clear
 * included, so starting one and downloading one both take the operator role.
 * A viewer gets the reason instead of the form.
 *
 * pcap, libpcap, Wireshark, DNS, TCP and HTTP are protocol and product nouns
 * (Do-Not-Translate).
 */

import type { TFunction } from 'i18next';
import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useRole } from '../../contexts/RoleContext';
import {
  type CaptureResult,
  type CaptureState,
  usePacketCapture,
} from '../../hooks/usePacketCapture';
import { formatBytes } from '../../lib/format';
import { button, cn, radius, spacing, status as statusColor } from '../../styles/theme';
import { Button } from '../ui/Button';
import { Card, type Status } from '../ui/Card';
import { Activity } from '../ui/Icons';
import { Input } from '../ui/Input';

// The job's bounds (internal/diagnostics/packetcapture): a minute when the
// request names no duration, at most an hour, and a 64 MiB file.
const DEFAULT_SECONDS = 60;
const MAX_SECONDS = 3600;
const MAX_FILE_BYTES = 64 * 1024 * 1024;
const MS_PER_SECOND = 1000;

function parseSeconds(raw: string): number | null {
  const seconds = Number(raw);
  return Number.isInteger(seconds) && seconds >= 1 && seconds <= MAX_SECONDS ? seconds : null;
}

function cardStatus(state: CaptureState): Status {
  switch (state.phase) {
    case 'starting':
    case 'running':
    case 'stopping':
      return 'loading';
    case 'failed':
      return 'error';
    case 'finished':
      return 'success';
    default:
      return 'unknown';
  }
}

function stopReasonText(t: TFunction<'cards'>, result: CaptureResult): string {
  switch (result.stopReason) {
    case 'duration':
      return t('packetCapture.stopReason.duration');
    case 'size':
      return t('packetCapture.stopReason.size', { size: formatBytes(MAX_FILE_BYTES, 0) });
    default:
      return t('packetCapture.stopReason.stopped');
  }
}

export function PacketCaptureCard({
  defaultInterface,
}: {
  /** The interface every other diagnostic runs on; the form starts on it. */
  defaultInterface: string;
}): JSX.Element {
  const { t } = useTranslation('cards');
  const { canWrite } = useRole();
  const { state, start, stop } = usePacketCapture();

  // null until the operator edits the field, so a current interface that
  // loads after mount still fills it.
  const [iface, setIface] = useState<string | null>(null);
  const [filter, setFilter] = useState('');
  const [secondsRaw, setSecondsRaw] = useState(String(DEFAULT_SECONDS));
  const [runningSeconds, setRunningSeconds] = useState(DEFAULT_SECONDS);

  const interfaceName = (iface ?? defaultInterface).trim();
  const seconds = parseSeconds(secondsRaw);
  const busy =
    state.phase === 'starting' || state.phase === 'running' || state.phase === 'stopping';

  const onStart = (): void => {
    if (seconds === null) {
      return;
    }
    setRunningSeconds(seconds);
    void start({
      interface: interfaceName,
      filter: filter.trim() || undefined,
      durationSeconds: seconds,
    });
  };

  return (
    <Card
      title={t('packetCapture.title')}
      icon={<Activity className="w-4 h-4" />}
      status={cardStatus(state)}
      ariaLabel={t('packetCapture.title')}
      // Three summary tables need more than a quarter of a 4-up grid; see
      // BonjourCard for why max-w-none rides with the span.
      className="sm:col-span-2 max-w-none"
    >
      <div className="stack-sm" data-testid="packet-capture" data-phase={state.phase}>
        <p className="caption text-text-muted">{t('packetCapture.description')}</p>

        {canWrite ? (
          <form
            className="stack-sm"
            onSubmit={(event) => {
              event.preventDefault();
              onStart();
            }}
          >
            <div className="grid gap-compact sm:grid-cols-3">
              <Input
                id="packet-capture-interface"
                label={t('packetCapture.interface')}
                value={iface ?? defaultInterface}
                placeholder={t('packetCapture.interfacePlaceholder')}
                onChange={(event) => setIface(event.target.value)}
                disabled={busy}
                data-testid="packet-capture-interface"
              />
              <Input
                id="packet-capture-filter"
                label={t('packetCapture.filter')}
                value={filter}
                placeholder={t('packetCapture.filterPlaceholder')}
                hint={t('packetCapture.filterHint')}
                onChange={(event) => setFilter(event.target.value)}
                disabled={busy}
                data-testid="packet-capture-filter"
              />
              <Input
                id="packet-capture-duration"
                label={t('packetCapture.duration')}
                type="number"
                inputMode="numeric"
                min={1}
                max={MAX_SECONDS}
                step={1}
                value={secondsRaw}
                onChange={(event) => setSecondsRaw(event.target.value)}
                disabled={busy}
                error={
                  seconds === null
                    ? t('packetCapture.durationInvalid', { max: MAX_SECONDS })
                    : undefined
                }
                hint={t('packetCapture.durationHint', { size: formatBytes(MAX_FILE_BYTES, 0) })}
                data-testid="packet-capture-duration"
              />
            </div>

            {busy ? (
              <CaptureRunning
                state={state}
                interfaceName={interfaceName}
                seconds={runningSeconds}
                onStop={() => void stop()}
              />
            ) : (
              <Button
                type="submit"
                variant="solid"
                size="sm"
                disabled={seconds === null}
                data-testid="packet-capture-start"
              >
                {t('packetCapture.start')}
              </Button>
            )}
          </form>
        ) : (
          <p className="body-small text-text-secondary" data-testid="packet-capture-read-only">
            {t('packetCapture.readOnly')}
          </p>
        )}

        {state.phase === 'failed' ? (
          <div
            className={cn(spacing.pad.sm, statusColor.bg.errorSoft, radius.md, 'stack-xs')}
            role="alert"
            data-testid="packet-capture-failed"
          >
            <p className="body-small text-status-error-strong">{t('packetCapture.failed')}</p>
            {state.error ? (
              <p
                className="caption font-mono text-text-secondary break-words"
                data-testid="packet-capture-error-detail"
              >
                {state.error}
              </p>
            ) : null}
          </div>
        ) : null}

        {state.phase === 'finished' ? <CaptureSummary result={state.result} /> : null}
      </div>
    </Card>
  );
}

function CaptureRunning({
  state,
  interfaceName,
  seconds,
  onStop,
}: {
  state: CaptureState;
  interfaceName: string;
  seconds: number;
  onStop: () => void;
}): JSX.Element {
  const { t } = useTranslation('cards');
  const stopping = state.phase === 'stopping';

  return (
    <div className="stack-xs" data-testid="packet-capture-running">
      <p className="body-small text-text-secondary" aria-live="polite">
        {state.phase === 'starting'
          ? t('packetCapture.starting')
          : t('packetCapture.running', {
              interface: interfaceName || t('packetCapture.interfacePlaceholder'),
              seconds,
              size: formatBytes(MAX_FILE_BYTES, 0),
            })}
      </p>
      <div className="flex items-center gap-compact">
        <Button
          type="button"
          variant="outline"
          tone="red"
          size="sm"
          onClick={onStop}
          disabled={state.phase !== 'running'}
          data-testid="packet-capture-stop"
        >
          {stopping ? t('packetCapture.stopping') : t('packetCapture.stop')}
        </Button>
        {state.phase === 'running' && state.stopFailed ? (
          <p className="caption text-status-error" role="alert">
            {t('packetCapture.stopFailed')}
          </p>
        ) : null}
      </div>
    </div>
  );
}

function CaptureSummary({ result }: { result: CaptureResult }): JSX.Element {
  const { t } = useTranslation('cards');
  const { summary } = result;

  return (
    <div
      className="stack-sm"
      data-testid="packet-capture-result"
      data-stop-reason={result.stopReason}
    >
      <div className="flex-between flex-wrap gap-compact">
        <div>
          <p className="body-small text-text-primary" data-testid="packet-capture-totals">
            {t('packetCapture.totals', {
              count: result.packets,
              bytes: formatBytes(result.bytes),
              seconds: Math.round(result.durationMs / MS_PER_SECOND),
              interface: result.interface,
            })}
          </p>
          <p className="caption text-text-muted">{stopReasonText(t, result)}</p>
        </div>
        <a
          href={`/api/v1/captures/${encodeURIComponent(result.id)}`}
          download={`seed-capture-${result.id}.pcap`}
          className={cn(button.base, button.size.sm, button.variant.secondary)}
          data-testid="packet-capture-download"
        >
          {t('packetCapture.download')}
        </a>
      </div>

      {result.packets === 0 ? (
        <p className="caption text-text-muted" data-testid="packet-capture-empty">
          {t('packetCapture.empty')}
        </p>
      ) : (
        <>
          <dl className="grid grid-cols-3 gap-compact" data-testid="packet-capture-counters">
            <Counter label={t('packetCapture.dnsQueries')} value={summary.dnsQueries} />
            <Counter label={t('packetCapture.tcpConnections')} value={summary.tcpConnections} />
            <Counter label={t('packetCapture.httpRequests')} value={summary.httpRequests} />
          </dl>

          <SummaryTable
            testId="packet-capture-protocols"
            caption={t('packetCapture.protocolsCaption')}
            columns={[
              { label: t('packetCapture.colProtocol'), width: 'w-[50%]' },
              { label: t('packetCapture.colPackets'), width: 'w-[25%]' },
              { label: t('packetCapture.colBytes'), width: 'w-[25%]' },
            ]}
            rows={summary.protocols.map((p) => ({
              key: p.name,
              cells: [p.name, String(p.packets), formatBytes(p.bytes)],
            }))}
          />
          <SummaryTable
            testId="packet-capture-talkers"
            caption={t('packetCapture.talkersCaption')}
            columns={[
              { label: t('packetCapture.colAddress'), width: 'w-[40%]', mono: true },
              { label: t('packetCapture.colPackets'), width: 'w-[20%]' },
              { label: t('packetCapture.colSent'), width: 'w-[20%]' },
              { label: t('packetCapture.colReceived'), width: 'w-[20%]' },
            ]}
            rows={summary.topTalkers.map((talker) => ({
              key: talker.address,
              cells: [
                talker.address,
                String(talker.packets),
                formatBytes(talker.bytesSent),
                formatBytes(talker.bytesReceived),
              ],
            }))}
          />
          <SummaryTable
            testId="packet-capture-flows"
            caption={t('packetCapture.flowsCaption')}
            columns={[
              { label: t('packetCapture.colProtocol'), width: 'w-[12%]' },
              { label: t('packetCapture.colFrom'), width: 'w-[30%]', mono: true },
              { label: t('packetCapture.colTo'), width: 'w-[30%]', mono: true },
              { label: t('packetCapture.colPackets'), width: 'w-[13%]' },
              { label: t('packetCapture.colBytes'), width: 'w-[15%]' },
            ]}
            rows={summary.topFlows.map((flow) => ({
              key: `${flow.transport} ${flow.source} ${flow.destination}`,
              cells: [
                flow.transport,
                flow.source,
                flow.destination,
                String(flow.packets),
                formatBytes(flow.bytes),
              ],
            }))}
          />

          {summary.truncated ? (
            <p className="caption text-text-muted" data-testid="packet-capture-truncated">
              {t('packetCapture.truncated')}
            </p>
          ) : null}
        </>
      )}
    </div>
  );
}

function Counter({ label, value }: { label: string; value: number }): JSX.Element {
  return (
    <div className="min-w-0">
      <dt className="caption text-text-muted truncate" title={label}>
        {label}
      </dt>
      <dd className="body-small text-text-primary">{value}</dd>
    </div>
  );
}

interface Column {
  label: string;
  width: string;
  mono?: boolean;
}

function SummaryTable({
  testId,
  caption,
  columns,
  rows,
}: {
  testId: string;
  caption: string;
  columns: Column[];
  rows: { key: string; cells: string[] }[];
}): JSX.Element | null {
  if (rows.length === 0) {
    return null;
  }
  return (
    // See NeighbourCacheCard: table-fixed, not overflow-x-auto (#2708).
    <div className={cn(radius.default, 'border border-surface-border')}>
      <table className="w-full table-fixed body-small" data-testid={testId}>
        <caption className="caption text-text-muted text-left px-cell pt-row">{caption}</caption>
        <thead>
          <tr className="border-b border-surface-border text-text-muted">
            {columns.map((column) => (
              <th
                key={column.label}
                scope="col"
                className={cn('px-cell py-row text-left truncate', column.width)}
                title={column.label}
              >
                {column.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.key} className="border-b border-surface-border last:border-0">
              {row.cells.map((cell, i) => (
                <td
                  // Cells are positional; the column label is unique per table.
                  key={columns[i]?.label ?? i}
                  className={cn('px-cell py-row truncate', columns[i]?.mono && 'font-mono')}
                  title={cell}
                >
                  {cell}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
