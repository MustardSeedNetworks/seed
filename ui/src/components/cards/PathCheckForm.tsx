/**
 * PathCheckForm — the destination form, run/stop control and failure notice
 * shared by the one-shot path checks on /path (MultiPathCard, PathMTUCard).
 *
 * Starting a check is a job, which takes the operator role, so a viewer gets
 * the reason instead of the form.
 */

import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useRole } from '../../contexts/RoleContext';
import type { BoundedJobState, UseBoundedJobReturn } from '../../hooks/useBoundedJob';
import { cn, radius, spacing, status as statusColor } from '../../styles/theme';
import { Button } from '../ui/Button';
import { Input } from '../ui/Input';

export function isBusy<Result>(state: BoundedJobState<Result>): boolean {
  return state.phase === 'starting' || state.phase === 'running' || state.phase === 'stopping';
}

export function PathCheckForm<Result>({
  testId,
  check: { state, start, stop },
  copy,
}: {
  /** Prefix of every test id, e.g. `multi-path`. */
  testId: string;
  check: UseBoundedJobReturn<{ destination: string }, Result>;
  copy: { start: string; running: (target: string) => string; failed: string };
}): JSX.Element {
  const { t } = useTranslation('cards');
  const { canWrite } = useRole();
  const [target, setTarget] = useState('');
  const destination = target.trim();
  const busy = isBusy(state);

  return (
    <>
      {canWrite ? (
        <form
          className="stack-sm"
          onSubmit={(event) => {
            event.preventDefault();
            if (destination) {
              void start({ destination });
            }
          }}
        >
          <Input
            id={`${testId}-target`}
            label={t('pathCheck.target')}
            value={target}
            placeholder={t('pathCheck.targetPlaceholder')}
            onChange={(event) => setTarget(event.target.value)}
            disabled={busy}
            data-testid={`${testId}-target`}
          />
          {busy ? (
            <div className="stack-xs" data-testid={`${testId}-running`}>
              <p className="body-small text-text-secondary" aria-live="polite">
                {state.phase === 'starting' ? t('pathCheck.starting') : copy.running(destination)}
              </p>
              <div className="flex items-center gap-compact">
                <Button
                  type="button"
                  variant="outline"
                  tone="red"
                  size="sm"
                  onClick={() => void stop()}
                  disabled={state.phase !== 'running'}
                  data-testid={`${testId}-stop`}
                >
                  {state.phase === 'stopping' ? t('pathCheck.stopping') : t('pathCheck.stop')}
                </Button>
                {state.phase === 'running' && state.stopFailed ? (
                  <p className="caption text-status-error" role="alert">
                    {t('pathCheck.stopFailed')}
                  </p>
                ) : null}
              </div>
            </div>
          ) : (
            <Button
              type="submit"
              variant="solid"
              size="sm"
              disabled={!destination}
              data-testid={`${testId}-start`}
            >
              {copy.start}
            </Button>
          )}
        </form>
      ) : (
        <p className="body-small text-text-secondary" data-testid={`${testId}-read-only`}>
          {t('pathCheck.readOnly')}
        </p>
      )}

      {state.phase === 'failed' ? (
        <div
          className={cn(spacing.pad.sm, statusColor.bg.errorSoft, radius.md, 'stack-xs')}
          role="alert"
          data-testid={`${testId}-failed`}
        >
          <p className="body-small text-status-error-strong">{copy.failed}</p>
          {state.error ? (
            <p
              className="caption font-mono text-text-secondary break-words"
              data-testid={`${testId}-error-detail`}
            >
              {state.error}
            </p>
          ) : null}
        </div>
      ) : null}
    </>
  );
}
