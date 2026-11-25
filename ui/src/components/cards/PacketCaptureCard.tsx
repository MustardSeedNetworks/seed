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
import { type CardStatus, useCardStatusLabel } from '../../hooks/useCardStatusLabel';
import {
  type CaptureResult,
  type CaptureState,
  usePacketCapture,
} from '../../hooks/usePacketCapture';
import { formatBytes } from '../../lib/format';
import { Alert } from '../../shared/Alert';
import { Button } from '../../shared/Button';
import { Card } from '../../shared/Card';
import { FormField } from '../../shared/FormField';
import { Input } from '../../shared/Input';
import { button, cn } from '../../styles/theme';
import { Activity } from '../ui/Icons';
import { SummaryTable } from './SummaryTable';

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

function cardStatus(state: CaptureState): CardStatus {
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
  const statusLabel = useCardStatusLabel();
  const status = cardStatus(state);

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
      status={status}
      statusLabel={statusLabel(status)}
      aria-label={t('packetCapture.title')}
      // Three summary tables need more than a quarter of a 4-up grid.
      className="sm:col-span-2"
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
              <FormField id="packet-capture-interface" label={t('packetCapture.interface')}>
                {(control) => (
                  <Input
                    {...control}
                    value={iface ?? defaultInterface}
                    placeholder={t('packetCapture.interfacePlaceholder')}
                    onChange={(event) => setIface(event.target.value)}
                    disabled={busy}
                    data-testid="packet-capture-interface"
                  />
                )}
              </FormField>
              <FormField
                id="packet-capture-filter"
                label={t('packetCapture.filter')}
                hint={t('packetCapture.filterHint')}
              >
                {(control) => (
                  <Input
                    {...control}
                    value={filter}
                    placeholder={t('packetCapture.filterPlaceholder')}
                    onChange={(event) => setFilter(event.target.value)}
                    disabled={busy}
                    data-testid="packet-capture-filter"
                  />
                )}
              </FormField>
              <FormField
                id="packet-capture-duration"
                label={t('packetCapture.duration')}
                hint={t('packetCapture.durationHint', { size: formatBytes(MAX_FILE_BYTES, 0) })}
                error={
                  seconds === null
                    ? t('packetCapture.durationInvalid', { max: MAX_SECONDS })
                    : undefined
                }
              >
                {(control) => (
                  <Input
                    {...control}
                    type="number"
                    inputMode="numeric"
                    min={1}
                    max={MAX_SECONDS}
                    step={1}
                    value={secondsRaw}
                    onChange={(event) => setSecondsRaw(event.target.value)}
                    disabled={busy}
                    data-testid="packet-capture-duration"
                  />
                )}
              </FormField>
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
                variant="primary"
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
          <Alert tone="error" size="sm" data-testid="packet-capture-failed">
            <p className="font-medium">{t('packetCapture.failed')}</p>
            {state.error ? (
              <p
                className="figure mt-1 break-words text-text-secondary"
                data-testid="packet-capture-error-detail"
              >
                {state.error}
              </p>
            ) : null}
          </Alert>
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
          variant="destructive"
          size="sm"
          onClick={onStop}
          disabled={state.phase !== 'running'}
          data-testid="packet-capture-stop"
        >
          {stopping ? t('packetCapture.stopping') : t('packetCapture.stop')}
        </Button>
        {state.phase === 'running' && state.stopFailed ? (
          <Alert tone="error" size="sm">
            {t('packetCapture.stopFailed')}
          </Alert>
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
              { label: t('packetCapture.colProtocol') },
              { label: t('packetCapture.colPackets') },
              { label: t('packetCapture.colBytes') },
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
              { label: t('packetCapture.colAddress'), mono: true },
              { label: t('packetCapture.colPackets') },
              { label: t('packetCapture.colSent') },
              { label: t('packetCapture.colReceived') },
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
              { label: t('packetCapture.colProtocol') },
              { label: t('packetCapture.colFrom'), mono: true },
              { label: t('packetCapture.colTo'), mono: true },
              { label: t('packetCapture.colPackets') },
              { label: t('packetCapture.colBytes') },
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
