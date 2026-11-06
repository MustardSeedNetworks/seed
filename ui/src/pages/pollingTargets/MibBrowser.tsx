/**
 * MibBrowser — walk one subtree of the selected polling target (P-E5,
 * UI-SEED-49) over POST /api/v1/polling-targets/{targetId}/walk.
 *
 * The server resolves a MIB object name to its OID and names each row it
 * returns with the object it belongs to; a row the MIB database does not know
 * keeps only its numeric OID. The server caps a walk at 500 rows and says when
 * it stopped there.
 */

import { type FormEvent, type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Tooltip } from '../../components/ui/Tooltip';
import type { SNMPWalkResponse, SNMPWalkRow } from '../../types/generated/snmp-walk-response';

type WalkState =
  | { kind: 'idle' }
  | { kind: 'walking' }
  | { kind: 'failed'; reason: string }
  | { kind: 'done'; result: SNMPWalkResponse };

function objectOf(row: SNMPWalkRow): string {
  if (!row.name) {
    return row.oid;
  }
  return row.instance ? `${row.name}.${row.instance}` : row.name;
}

export function MibBrowser({
  walk,
  disabledReason,
}: {
  walk: (oid: string) => Promise<SNMPWalkResponse>;
  /** Set when the caller may not walk; the form stays visible but disabled. */
  disabledReason?: string;
}): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const [oid, setOid] = useState('');
  const [state, setState] = useState<WalkState>({ kind: 'idle' });

  const walking = state.kind === 'walking';
  const unavailable = Boolean(disabledReason) || walking || oid.trim() === '';

  const submit = (e: FormEvent): void => {
    e.preventDefault();
    setState({ kind: 'walking' });
    walk(oid.trim())
      .then((result) => setState({ kind: 'done', result }))
      .catch((err: unknown) =>
        setState({ kind: 'failed', reason: err instanceof Error ? err.message : String(err) }),
      );
  };

  return (
    <section className="stack-xs" data-testid="mib-browser">
      <p className="caption">{t('pollingTargets.mibBrowser')}</p>
      <p className="body-small">{t('pollingTargets.mibBrowserHint')}</p>
      <form className="flex flex-wrap items-end gap-compact" onSubmit={submit}>
        <label className="flex min-w-0 flex-1 flex-col gap-1 text-sm text-text-secondary">
          {t('pollingTargets.mibOidLabel')}
          <input
            type="text"
            value={oid}
            onChange={(e): void => setOid(e.target.value)}
            placeholder={t('pollingTargets.mibOidPlaceholder')}
            disabled={Boolean(disabledReason)}
            data-testid="mib-oid"
            className="w-full rounded-lg border border-surface-border bg-surface-sunken px-3 py-2 font-mono text-sm text-text-primary placeholder:text-text-muted focus:border-brand-primary focus:outline-none disabled:cursor-not-allowed disabled:control-disabled"
          />
        </label>
        <Tooltip text={disabledReason}>
          <button
            type="submit"
            disabled={unavailable}
            data-testid="mib-walk"
            className="rounded-md border border-surface-border px-3 py-2 text-sm text-text-primary hover:bg-surface-hover disabled:cursor-not-allowed disabled:control-disabled"
          >
            {walking ? t('pollingTargets.mibWalking') : t('pollingTargets.mibWalk')}
          </button>
        </Tooltip>
      </form>
      {state.kind === 'failed' ? (
        <p role="alert" data-testid="mib-error" className="body-small text-status-error-strong">
          {t('pollingTargets.mibWalkFailed', { reason: state.reason })}
        </p>
      ) : null}
      {state.kind === 'done' ? <WalkResult result={state.result} /> : null}
    </section>
  );
}

function WalkResult({ result }: { result: SNMPWalkResponse }): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  if (result.rows.length === 0) {
    return (
      <p className="body-small" data-testid="mib-empty">
        {t('pollingTargets.mibNoRows', { oid: result.oid })}
      </p>
    );
  }
  return (
    <div className="stack-xs">
      <p className="body-small" data-testid="mib-summary">
        {t('pollingTargets.mibRowCount', { count: result.rows.length, oid: result.oid })}
      </p>
      {result.truncated ? (
        <p className="body-small text-status-warning-strong" data-testid="mib-truncated">
          {t('pollingTargets.mibTruncated', { count: result.rows.length })}
        </p>
      ) : null}
      <ol
        className="max-h-96 divide-y divide-surface-border overflow-y-auto rounded-lg border border-surface-border"
        data-testid="mib-rows"
      >
        {result.rows.map((row) => (
          <li key={row.oid} className="flex flex-col gap-0.5 px-cell py-2" title={row.oid}>
            <span className="figure break-all text-xs text-text-muted">{objectOf(row)}</span>
            <span className="break-all text-sm text-text-primary">{row.value}</span>
          </li>
        ))}
      </ol>
    </div>
  );
}
