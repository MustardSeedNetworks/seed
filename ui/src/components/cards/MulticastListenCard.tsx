/**
 * MulticastListenCard — is a multicast stream arriving on this port at all,
 * and from whom (#399).
 *
 * The listen joins the group, which is what makes an IGMP- or MLD-snooping
 * switch forward it, and sends nothing. Hearing nothing puts the fault with
 * the sender or the multicast path; hearing the stream puts it with the
 * receiving application. Starting a listen is a job, which takes the operator
 * role, so a viewer gets the reason instead of the form.
 *
 * IGMP, MLD and IPTV are protocol nouns (Do-Not-Translate).
 */

import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useRole } from '../../contexts/RoleContext';
import { type CardStatus, useCardStatusLabel } from '../../hooks/useCardStatusLabel';
import {
  type ListenResult,
  type ListenState,
  useMulticastListen,
} from '../../hooks/useMulticastListen';
import { formatBytes } from '../../lib/format';
import { Alert } from '../../shared/Alert';
import { Button } from '../../shared/Button';
import { Card } from '../../shared/Card';
import { FormField } from '../../shared/FormField';
import { Input } from '../../shared/Input';
import { Tv } from '../ui/Icons';
import { SummaryTable } from './SummaryTable';

// The job's bounds (internal/diagnostics/multicast): ten seconds catches both
// a stream and a service announcement that repeats every few seconds, and a
// membership is held for at most a minute.
const DEFAULT_SECONDS = 10;
const MAX_SECONDS = 60;
const MAX_PORT = 65535;
const MS_PER_SECOND = 1000;

function parseWhole(raw: string, max: number): number | null {
  const value = Number(raw);
  return raw.trim() !== '' && Number.isInteger(value) && value >= 1 && value <= max ? value : null;
}

function cardStatus(state: ListenState): CardStatus {
  switch (state.phase) {
    case 'starting':
    case 'running':
    case 'stopping':
      return 'loading';
    case 'failed':
      return 'error';
    case 'finished':
      // Silence is the finding the operator came for, not a failed run.
      return state.result.packets > 0 ? 'success' : 'warning';
    default:
      return 'unknown';
  }
}

export function MulticastListenCard({
  defaultInterface,
}: {
  /** The interface every other diagnostic runs on; the form starts on it. */
  defaultInterface: string;
}): JSX.Element {
  const { t } = useTranslation('cards');
  const { canWrite } = useRole();
  const { state, start, stop } = useMulticastListen();
  const statusLabel = useCardStatusLabel();
  const status = cardStatus(state);

  // null until the operator edits the field, so a current interface that
  // loads after mount still fills it.
  const [iface, setIface] = useState<string | null>(null);
  const [group, setGroup] = useState('');
  const [portRaw, setPortRaw] = useState('');
  const [secondsRaw, setSecondsRaw] = useState(String(DEFAULT_SECONDS));

  const interfaceName = (iface ?? defaultInterface).trim();
  const groupAddress = group.trim();
  const port = parseWhole(portRaw, MAX_PORT);
  const seconds = parseWhole(secondsRaw, MAX_SECONDS);
  const busy =
    state.phase === 'starting' || state.phase === 'running' || state.phase === 'stopping';
  const ready = interfaceName !== '' && groupAddress !== '' && port !== null && seconds !== null;

  const onStart = (): void => {
    if (!ready) {
      return;
    }
    void start({
      group: groupAddress,
      port,
      interface: interfaceName,
      durationSeconds: seconds,
    });
  };

  return (
    <Card
      title={t('multicastListen.title')}
      icon={<Tv className="w-4 h-4" />}
      status={status}
      statusLabel={statusLabel(status)}
      aria-label={t('multicastListen.title')}
      // A sender table needs more than a quarter of a 4-up grid.
      className="sm:col-span-2"
    >
      <div className="stack-sm" data-testid="multicast-listen" data-phase={state.phase}>
        <p className="caption text-text-muted">{t('multicastListen.description')}</p>

        {canWrite ? (
          <form
            className="stack-sm"
            onSubmit={(event) => {
              event.preventDefault();
              onStart();
            }}
          >
            <div className="grid gap-compact sm:grid-cols-2">
              <FormField id="multicast-listen-group" label={t('multicastListen.group')}>
                {(control) => (
                  <Input
                    {...control}
                    value={group}
                    placeholder="239.1.1.1"
                    onChange={(event) => setGroup(event.target.value)}
                    disabled={busy}
                    data-testid="multicast-listen-group"
                  />
                )}
              </FormField>
              <FormField
                id="multicast-listen-port"
                label={t('multicastListen.port')}
                error={
                  portRaw !== '' && port === null
                    ? t('multicastListen.portInvalid', { max: MAX_PORT })
                    : undefined
                }
              >
                {(control) => (
                  <Input
                    {...control}
                    type="number"
                    inputMode="numeric"
                    min={1}
                    max={MAX_PORT}
                    step={1}
                    value={portRaw}
                    placeholder="5000"
                    onChange={(event) => setPortRaw(event.target.value)}
                    disabled={busy}
                    data-testid="multicast-listen-port"
                  />
                )}
              </FormField>
              <FormField id="multicast-listen-interface" label={t('multicastListen.interface')}>
                {(control) => (
                  <Input
                    {...control}
                    value={iface ?? defaultInterface}
                    placeholder={t('multicastListen.interfacePlaceholder')}
                    onChange={(event) => setIface(event.target.value)}
                    disabled={busy}
                    data-testid="multicast-listen-interface"
                  />
                )}
              </FormField>
              <FormField
                id="multicast-listen-duration"
                label={t('multicastListen.duration')}
                error={
                  seconds === null
                    ? t('multicastListen.durationInvalid', { max: MAX_SECONDS })
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
                    data-testid="multicast-listen-duration"
                  />
                )}
              </FormField>
            </div>

            {busy ? (
              <ListenRunning
                state={state}
                group={groupAddress}
                port={port ?? 0}
                interfaceName={interfaceName}
                onStop={() => void stop()}
              />
            ) : (
              <Button
                type="submit"
                variant="primary"
                size="sm"
                disabled={!ready}
                data-testid="multicast-listen-start"
              >
                {t('multicastListen.start')}
              </Button>
            )}
          </form>
        ) : (
          <p className="body-small text-text-secondary" data-testid="multicast-listen-read-only">
            {t('multicastListen.readOnly')}
          </p>
        )}

        {state.phase === 'failed' ? (
          <Alert tone="error" size="sm" data-testid="multicast-listen-failed">
            <p className="font-medium">{t('multicastListen.failed')}</p>
            {state.error ? (
              <p
                className="figure mt-1 break-words text-text-secondary"
                data-testid="multicast-listen-error-detail"
              >
                {state.error}
              </p>
            ) : null}
          </Alert>
        ) : null}

        {state.phase === 'finished' ? <ListenSummary result={state.result} /> : null}
      </div>
    </Card>
  );
}

function ListenRunning({
  state,
  group,
  port,
  interfaceName,
  onStop,
}: {
  state: ListenState;
  group: string;
  port: number;
  interfaceName: string;
  onStop: () => void;
}): JSX.Element {
  const { t } = useTranslation('cards');

  return (
    <div className="stack-xs" data-testid="multicast-listen-running">
      <p className="body-small text-text-secondary" aria-live="polite">
        {state.phase === 'starting'
          ? t('multicastListen.starting')
          : t('multicastListen.running', { group, port, interface: interfaceName })}
      </p>
      <div className="flex items-center gap-compact">
        <Button
          type="button"
          variant="destructive"
          size="sm"
          onClick={onStop}
          disabled={state.phase !== 'running'}
          data-testid="multicast-listen-stop"
        >
          {state.phase === 'stopping' ? t('multicastListen.stopping') : t('multicastListen.stop')}
        </Button>
        {state.phase === 'running' && state.stopFailed ? (
          <p className="caption text-status-error" role="alert">
            {t('multicastListen.stopFailed')}
          </p>
        ) : null}
      </div>
    </div>
  );
}

function ListenSummary({ result }: { result: ListenResult }): JSX.Element {
  const { t } = useTranslation('cards');

  return (
    <div className="stack-sm" data-testid="multicast-listen-result">
      <p className="body-small text-text-primary" data-testid="multicast-listen-totals">
        {t('multicastListen.totals', {
          count: result.packets,
          bytes: formatBytes(result.bytes),
          rate: result.packetsPerSecond.toFixed(1),
          seconds: Math.round(result.listenedMs / MS_PER_SECOND),
          group: result.group,
          port: result.port,
          interface: result.interface,
        })}
      </p>

      {result.packets === 0 ? (
        <p className="body-small text-status-warning-strong" data-testid="multicast-listen-silent">
          {t('multicastListen.silent')}
        </p>
      ) : (
        <SummaryTable
          testId="multicast-listen-sources"
          caption={t('multicastListen.sourcesCaption')}
          columns={[
            { label: t('multicastListen.colSender'), mono: true },
            { label: t('multicastListen.colPackets') },
            { label: t('multicastListen.colBytes') },
          ]}
          rows={result.sources.map((source) => ({
            key: source.address,
            cells: [source.address, String(source.packets), formatBytes(source.bytes)],
          }))}
        />
      )}

      {result.sourcesTruncated ? (
        <p className="caption text-text-muted" data-testid="multicast-listen-truncated">
          {t('multicastListen.truncated')}
        </p>
      ) : null}
      {result.groupFiltered ? null : (
        <p className="caption text-text-muted" data-testid="multicast-listen-unfiltered">
          {t('multicastListen.unfiltered')}
        </p>
      )}
    </div>
  );
}
