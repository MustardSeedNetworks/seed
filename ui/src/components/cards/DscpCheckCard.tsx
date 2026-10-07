/**
 * DscpCheckCard — does the path keep the DSCP marking each class was sent
 * with (#400).
 *
 * The fault it isolates is a switch port that does not trust the marking, or
 * an access point that maps WMM priority to the wrong wired DSCP: voice sent
 * as EF arrives as best effort and degrades only under load. Across two
 * hosts, one Seed listens and the other sends; the listener reads the verdict.
 * On one host with a Wi-Fi and a wired interface, Seed does both.
 *
 * The check is sold as `dscp_verification` (Pro), so below Pro the card says
 * so and sends nothing. Starting a check is a job, which takes the operator
 * role, so a viewer gets the reason instead of the form.
 *
 * DSCP, EF, AF and CS class names are protocol nouns (Do-Not-Translate).
 */

import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useLicense } from '../../contexts/LicenseContext';
import { useRole } from '../../contexts/RoleContext';
import type { UseBoundedJobReturn } from '../../hooks/useBoundedJob';
import {
  type DscpMode,
  type DscpRequest,
  type DscpResult,
  type DscpState,
  isListenResult,
  isModeResult,
  isSingleHostResult,
  useDscpCheck,
} from '../../hooks/useDscpCheck';
import { cn, radius, spacing, status as statusColor } from '../../styles/theme';
import { Button } from '../ui/Button';
import { Card, type Status } from '../ui/Card';
import { Gauge } from '../ui/Icons';
import { Input } from '../ui/Input';
import { DscpCheckResult, verdictOf } from './DscpCheckResult';

const FEATURE = 'dscp_verification';
const MODES: readonly DscpMode[] = ['listen', 'send', 'single-host'];

// The job's bounds (internal/diagnostics/qos): a listen runs ten seconds
// unless told otherwise, long enough to start the sender by hand, and holds
// its port for at most a minute. Both ends start on the same port, so the
// first two-host check needs nothing typed but the listener's address.
const DEFAULT_PORT = 45400;
const DEFAULT_SECONDS = 10;
const MAX_SECONDS = 60;
const MAX_PORT = 65535;

function parseWhole(raw: string, max: number): number | null {
  const value = Number(raw);
  return raw.trim() !== '' && Number.isInteger(value) && value >= 1 && value <= max ? value : null;
}

function resultStatus(result: DscpResult): Status {
  if (isListenResult(result)) {
    const verdicts = result.runs.map((run) => verdictOf(run.classes));
    if (verdicts.length === 0 || verdicts.includes(null)) {
      return 'warning';
    }
    return verdicts.every(Boolean) ? 'success' : 'error';
  }
  if (isSingleHostResult(result)) {
    const verdict = verdictOf(result.classes);
    return verdict === null ? 'warning' : verdict ? 'success' : 'error';
  }
  return result.marked ? 'success' : 'warning';
}

function cardStatus(state: DscpState): Status {
  switch (state.phase) {
    case 'starting':
    case 'running':
    case 'stopping':
      return 'loading';
    case 'failed':
      return 'error';
    case 'finished':
      return resultStatus(state.result);
    default:
      return 'unknown';
  }
}

/** Each mode's copy, as static keys so the locale gate can see every one. */
function useModeCopy(): Record<
  DscpMode,
  { label: string; hint: string; start: string; running: string }
> {
  const { t } = useTranslation('cards');
  return {
    listen: {
      label: t('dscpCheck.mode.listen'),
      hint: t('dscpCheck.modeHint.listen'),
      start: t('dscpCheck.start.listen'),
      running: t('dscpCheck.running.listen'),
    },
    send: {
      label: t('dscpCheck.mode.send'),
      hint: t('dscpCheck.modeHint.send'),
      start: t('dscpCheck.start.send'),
      running: t('dscpCheck.running.send'),
    },
    'single-host': {
      label: t('dscpCheck.mode.single-host'),
      hint: t('dscpCheck.modeHint.single-host'),
      start: t('dscpCheck.start.single-host'),
      running: t('dscpCheck.running.single-host'),
    },
  };
}

export function DscpCheckCard({
  defaultInterface,
}: {
  /** The interface every other diagnostic runs on; the capture field starts on it. */
  defaultInterface: string;
}): JSX.Element {
  const { t } = useTranslation('cards');
  const { t: tErrors } = useTranslation('errors');
  const { hasFeature, loading } = useLicense();
  const { canWrite } = useRole();

  const [mode, setMode] = useState<DscpMode>('listen');
  const check = useDscpCheck(mode);
  const { state } = check;

  const licensed = hasFeature(FEATURE);
  // A result is shown under the mode that produced it; switching mode after
  // a check leaves it behind rather than reading it as the other kind.
  const result =
    state.phase === 'finished' && isModeResult(mode, state.result) ? state.result : null;

  return (
    <Card
      title={t('dscpCheck.title')}
      icon={<Gauge className="w-4 h-4" />}
      status={licensed ? cardStatus(state) : 'unknown'}
      ariaLabel={t('dscpCheck.title')}
      // A per-class table needs more than a quarter of a 4-up grid; see
      // BonjourCard for why max-w-none rides with the span.
      className="sm:col-span-2 max-w-none"
    >
      <div className="stack-sm" data-testid="dscp-check" data-phase={state.phase} data-mode={mode}>
        <p className="caption text-text-muted">{t('dscpCheck.description')}</p>

        {loading ? null : !licensed ? (
          <div className="stack-xs" data-testid="dscp-check-upgrade">
            <p className="body-small text-text-secondary">
              {tErrors('license.tierTooLow', { tier: 'Pro' })}
            </p>
            <p className="caption text-text-muted">
              {tErrors('license.upgradeHint', { tier: 'Pro' })}
            </p>
          </div>
        ) : !canWrite ? (
          <p className="body-small text-text-secondary" data-testid="dscp-check-read-only">
            {t('dscpCheck.readOnly')}
          </p>
        ) : (
          <CheckForm
            mode={mode}
            onModeChange={setMode}
            defaultInterface={defaultInterface}
            check={check}
          />
        )}

        {licensed && state.phase === 'failed' ? (
          <div
            className={cn(spacing.pad.sm, statusColor.bg.errorSoft, radius.md, 'stack-xs')}
            role="alert"
            data-testid="dscp-check-failed"
          >
            <p className="body-small text-status-error-strong">{t('dscpCheck.failed')}</p>
            {state.error ? (
              <p
                className="caption font-mono text-text-secondary break-words"
                data-testid="dscp-check-error-detail"
              >
                {state.error}
              </p>
            ) : null}
          </div>
        ) : null}

        {licensed && result !== null ? <DscpCheckResult result={result} /> : null}
      </div>
    </Card>
  );
}

function CheckForm({
  mode,
  onModeChange,
  defaultInterface,
  check: { state, start, stop },
}: {
  mode: DscpMode;
  onModeChange: (mode: DscpMode) => void;
  defaultInterface: string;
  check: UseBoundedJobReturn<DscpRequest, DscpResult>;
}): JSX.Element {
  const { t } = useTranslation('cards');
  const copy = useModeCopy()[mode];

  const [portRaw, setPortRaw] = useState(String(DEFAULT_PORT));
  const [secondsRaw, setSecondsRaw] = useState(String(DEFAULT_SECONDS));
  const [target, setTarget] = useState('');
  const [sendInterface, setSendInterface] = useState('');
  // null until the operator edits the field, so a current interface that
  // loads after mount still fills it.
  const [captureInterface, setCaptureInterface] = useState<string | null>(null);

  const port = parseWhole(portRaw, MAX_PORT);
  const seconds = parseWhole(secondsRaw, MAX_SECONDS);
  const targetAddress = target.trim();
  const sendName = sendInterface.trim();
  const captureName = (captureInterface ?? defaultInterface).trim();
  const busy =
    state.phase === 'starting' || state.phase === 'running' || state.phase === 'stopping';

  const request = ((): DscpRequest | null => {
    switch (mode) {
      case 'listen':
        return port !== null && seconds !== null ? { port, durationSeconds: seconds } : null;
      case 'send':
        return port !== null && targetAddress !== '' ? { target: targetAddress, port } : null;
      case 'single-host':
        return sendName !== '' && captureName !== '' && sendName !== captureName
          ? { sendInterface: sendName, captureInterface: captureName }
          : null;
    }
  })();

  const portField = (
    <Input
      id="dscp-check-port"
      label={t('dscpCheck.port')}
      type="number"
      inputMode="numeric"
      min={1}
      max={MAX_PORT}
      step={1}
      value={portRaw}
      onChange={(event) => setPortRaw(event.target.value)}
      disabled={busy}
      error={port === null ? t('dscpCheck.portInvalid', { max: MAX_PORT }) : undefined}
      data-testid="dscp-check-port"
    />
  );

  return (
    <form
      className="stack-sm"
      onSubmit={(event) => {
        event.preventDefault();
        if (request !== null) {
          void start(request);
        }
      }}
    >
      <ModePicker mode={mode} disabled={busy} onChange={onModeChange} />
      <p className="caption text-text-secondary" data-testid="dscp-check-mode-hint">
        {copy.hint}
      </p>

      <div className="grid gap-compact sm:grid-cols-2">
        {mode === 'listen' ? (
          <>
            {portField}
            <Input
              id="dscp-check-duration"
              label={t('dscpCheck.duration')}
              type="number"
              inputMode="numeric"
              min={1}
              max={MAX_SECONDS}
              step={1}
              value={secondsRaw}
              onChange={(event) => setSecondsRaw(event.target.value)}
              disabled={busy}
              error={
                seconds === null ? t('dscpCheck.durationInvalid', { max: MAX_SECONDS }) : undefined
              }
              data-testid="dscp-check-duration"
            />
          </>
        ) : null}
        {mode === 'send' ? (
          <>
            <Input
              id="dscp-check-target"
              label={t('dscpCheck.target')}
              value={target}
              placeholder="192.0.2.10"
              onChange={(event) => setTarget(event.target.value)}
              disabled={busy}
              data-testid="dscp-check-target"
            />
            {portField}
          </>
        ) : null}
        {mode === 'single-host' ? (
          <>
            <Input
              id="dscp-check-send-interface"
              label={t('dscpCheck.sendInterface')}
              value={sendInterface}
              placeholder="wlan0"
              onChange={(event) => setSendInterface(event.target.value)}
              disabled={busy}
              data-testid="dscp-check-send-interface"
            />
            <Input
              id="dscp-check-capture-interface"
              label={t('dscpCheck.captureInterface')}
              value={captureInterface ?? defaultInterface}
              placeholder="eth0"
              onChange={(event) => setCaptureInterface(event.target.value)}
              disabled={busy}
              error={
                sendName !== '' && sendName === captureName
                  ? t('dscpCheck.sameInterface')
                  : undefined
              }
              data-testid="dscp-check-capture-interface"
            />
          </>
        ) : null}
      </div>

      {busy ? (
        <CheckRunning state={state} mode={mode} onStop={() => void stop()} />
      ) : (
        <Button
          type="submit"
          variant="solid"
          size="sm"
          disabled={request === null}
          data-testid="dscp-check-start"
        >
          {copy.start}
        </Button>
      )}
    </form>
  );
}

function ModePicker({
  mode,
  disabled,
  onChange,
}: {
  mode: DscpMode;
  disabled: boolean;
  onChange: (mode: DscpMode) => void;
}): JSX.Element {
  const { t } = useTranslation('cards');
  const copy = useModeCopy();

  return (
    <div
      className={cn('flex flex-wrap', spacing.gap.compact)}
      role="radiogroup"
      aria-label={t('dscpCheck.modeLabel')}
    >
      {MODES.map((option) => {
        const checked = mode === option;
        return (
          <label
            key={option}
            className={cn(
              spacing.chip.md,
              radius.full,
              'border body-small font-medium transition-colors',
              checked
                ? 'bg-brand-primary text-on-brand border-brand-primary'
                : 'bg-surface-base border-surface-border',
              !checked &&
                (disabled ? 'text-text-muted' : 'text-text-primary hover:bg-surface-hover'),
              disabled ? 'cursor-not-allowed' : 'cursor-pointer',
            )}
            data-testid={`dscp-check-mode-${option}`}
          >
            <input
              type="radio"
              name="dscp-check-mode"
              value={option}
              checked={checked}
              disabled={disabled}
              onChange={() => onChange(option)}
              className="sr-only"
            />
            {copy[option].label}
          </label>
        );
      })}
    </div>
  );
}

function CheckRunning({
  state,
  mode,
  onStop,
}: {
  state: DscpState;
  mode: DscpMode;
  onStop: () => void;
}): JSX.Element {
  const { t } = useTranslation('cards');
  const copy = useModeCopy()[mode];
  const progress = state.phase === 'starting' ? t('dscpCheck.starting') : copy.running;

  return (
    <div className="stack-xs" data-testid="dscp-check-running">
      <p className="body-small text-text-secondary" aria-live="polite">
        {progress}
      </p>
      <div className="flex items-center gap-compact">
        <Button
          type="button"
          variant="outline"
          tone="red"
          size="sm"
          onClick={onStop}
          disabled={state.phase !== 'running'}
          data-testid="dscp-check-stop"
        >
          {state.phase === 'stopping' ? t('dscpCheck.stopping') : t('dscpCheck.stop')}
        </Button>
        {state.phase === 'running' && state.stopFailed ? (
          <p className="caption text-status-error" role="alert">
            {t('dscpCheck.stopFailed')}
          </p>
        ) : null}
      </div>
    </div>
  );
}
