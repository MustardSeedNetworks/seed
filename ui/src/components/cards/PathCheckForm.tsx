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
import { Alert } from '../../shared/Alert';
import { Button } from '../../shared/Button';
import { FormField } from '../../shared/FormField';
import { Input } from '../../shared/Input';

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
          <FormField id={`${testId}-target`} label={t('pathCheck.target')}>
            {(control) => (
              <Input
                {...control}
                value={target}
                placeholder={t('pathCheck.targetPlaceholder')}
                onChange={(event) => setTarget(event.target.value)}
                disabled={busy}
                data-testid={`${testId}-target`}
              />
            )}
          </FormField>
          {busy ? (
            <div className="stack-xs" data-testid={`${testId}-running`}>
              <p className="body-small text-text-secondary" aria-live="polite">
                {state.phase === 'starting' ? t('pathCheck.starting') : copy.running(destination)}
              </p>
              <div className="flex items-center gap-compact">
                <Button
                  type="button"
                  variant="destructive"
                  size="sm"
                  onClick={() => void stop()}
                  disabled={state.phase !== 'running'}
                  data-testid={`${testId}-stop`}
                >
                  {state.phase === 'stopping' ? t('pathCheck.stopping') : t('pathCheck.stop')}
                </Button>
                {state.phase === 'running' && state.stopFailed ? (
                  <Alert tone="error" size="sm">
                    {t('pathCheck.stopFailed')}
                  </Alert>
                ) : null}
              </div>
            </div>
          ) : (
            <Button
              type="submit"
              variant="primary"
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
        <Alert tone="error" size="sm" data-testid={`${testId}-failed`}>
          <p className="font-medium">{copy.failed}</p>
          {state.error ? (
            <p
              className="figure mt-1 break-words text-text-secondary"
              data-testid={`${testId}-error-detail`}
            >
              {state.error}
            </p>
          ) : null}
        </Alert>
      ) : null}
    </>
  );
}
