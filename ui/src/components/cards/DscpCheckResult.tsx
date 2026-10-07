/**
 * The DSCP check's verdict and per-class table, for each kind of result
 * DscpCheckCard can show (#400).
 */

import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type ClassResult,
  type DscpResult,
  isListenResult,
  isSingleHostResult,
  type ListenResult,
  type SendResult,
  type SingleHostResult,
} from '../../hooks/useDscpCheck';
import { cn } from '../../styles/theme';
import { SummaryTable } from './SummaryTable';

/** Whether every class kept its marking; null when none could be read. */
export function verdictOf(classes: ClassResult[]): boolean | null {
  if (classes.length > 0 && classes.every((c) => c.verdict === 'unobserved')) {
    return null;
  }
  return classes.every((c) => c.verdict === 'preserved');
}

export function DscpCheckResult({ result }: { result: DscpResult }): JSX.Element {
  if (isListenResult(result)) {
    return <ListenSummary result={result} />;
  }
  if (isSingleHostResult(result)) {
    return <SingleHostSummary result={result} />;
  }
  return <SendSummary result={result} />;
}

function classLabel(dscp: number, name: string | undefined): string {
  return name ? `${name} (${dscp})` : String(dscp);
}

function Verdict({ classes, testId }: { classes: ClassResult[]; testId: string }): JSX.Element {
  const { t } = useTranslation('cards');
  const preserved = verdictOf(classes);
  const changed = classes.filter((c) => c.verdict !== 'preserved').length;

  if (preserved === null) {
    return (
      <p className="body-small text-status-warning-strong" data-testid={testId}>
        {t('dscpCheck.unobserved')}
      </p>
    );
  }
  return (
    <p
      className={cn(
        'body-small font-medium',
        preserved ? 'text-status-success-strong' : 'text-status-error-strong',
      )}
      data-testid={testId}
      data-pass={preserved}
    >
      {preserved
        ? t('dscpCheck.pass')
        : t('dscpCheck.fail', { count: changed, total: classes.length })}
    </p>
  );
}

function ClassTable({ classes, testId }: { classes: ClassResult[]; testId: string }): JSX.Element {
  const { t } = useTranslation('cards');
  const verdicts: Record<ClassResult['verdict'], string> = {
    preserved: t('dscpCheck.verdict.preserved'),
    remarked: t('dscpCheck.verdict.remarked'),
    mixed: t('dscpCheck.verdict.mixed'),
    lost: t('dscpCheck.verdict.lost'),
    unobserved: t('dscpCheck.verdict.unobserved'),
  };

  return (
    <SummaryTable
      testId={testId}
      caption={t('dscpCheck.classesCaption')}
      columns={[
        { label: t('dscpCheck.colClass'), width: 'w-[25%]', mono: true },
        { label: t('dscpCheck.colResult'), width: 'w-[20%]' },
        { label: t('dscpCheck.colArrivedAs'), width: 'w-[35%]', mono: true },
        { label: t('dscpCheck.colReceived'), width: 'w-[20%]' },
      ]}
      rows={classes.map((c) => ({
        key: String(c.sentDscp),
        cells: [
          classLabel(c.sentDscp, c.sentName),
          verdicts[c.verdict],
          c.observed.length === 0
            ? '—'
            : c.observed.map((o) => `${classLabel(o.dscp, o.name)} ×${o.count}`).join(', '),
          `${c.received}/${c.expected}`,
        ],
      }))}
    />
  );
}

function ListenSummary({ result }: { result: ListenResult }): JSX.Element {
  const { t } = useTranslation('cards');

  return (
    <div className="stack-sm" data-testid="dscp-check-result">
      {result.runs.length === 0 ? (
        <p className="body-small text-status-warning-strong" data-testid="dscp-check-silent">
          {t('dscpCheck.silent', { port: result.port })}
        </p>
      ) : (
        result.runs.map((run) => (
          <div key={`${run.sender}-${run.runId}`} className="stack-xs" data-testid="dscp-check-run">
            <p className="body-small text-text-primary">
              {t('dscpCheck.runFrom', { sender: run.sender })}
            </p>
            <Verdict classes={run.classes} testId="dscp-check-verdict" />
            <ClassTable classes={run.classes} testId="dscp-check-classes" />
          </div>
        ))
      )}
      {result.runsTruncated ? (
        <p className="caption text-text-muted" data-testid="dscp-check-truncated">
          {t('dscpCheck.truncated')}
        </p>
      ) : null}
    </div>
  );
}

function SingleHostSummary({ result }: { result: SingleHostResult }): JSX.Element {
  const { t } = useTranslation('cards');
  const route = {
    send: result.sendInterface,
    source: result.source,
    capture: result.captureInterface,
    target: result.target,
    nextHop: result.nextHop,
  };

  return (
    <div className="stack-sm" data-testid="dscp-check-result">
      <p className="body-small text-text-primary" data-testid="dscp-check-route">
        {result.routed ? t('dscpCheck.routeRouted', route) : t('dscpCheck.routeDirect', route)}
      </p>
      <Verdict classes={result.classes} testId="dscp-check-verdict" />
      <ClassTable classes={result.classes} testId="dscp-check-classes" />
    </div>
  );
}

function SendSummary({ result }: { result: SendResult }): JSX.Element {
  const { t } = useTranslation('cards');

  return (
    <div className="stack-xs" data-testid="dscp-check-result">
      <p className="body-small text-text-primary" data-testid="dscp-check-sent">
        {t('dscpCheck.sent', {
          count: result.count,
          classes: result.classes.map((c) => classLabel(c.dscp, c.name)).join(', '),
          target: result.target,
          port: result.port,
        })}
      </p>
      {result.marked ? null : (
        <p className="body-small text-status-warning-strong" data-testid="dscp-check-unmarked">
          {t('dscpCheck.unmarked')}
        </p>
      )}
    </div>
  );
}
