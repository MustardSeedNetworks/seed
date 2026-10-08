/**
 * PathMTUCard — the largest packet that reaches one destination without being
 * fragmented (#435).
 *
 * A path MTU below the local link's points at a tunnel or VPN on the way; a
 * path where large packets vanish without the ICMP "too big" reply is a
 * black hole that stalls big transfers while pings work. The `path-mtu` job
 * searches packet sizes and reports which of those it found.
 */

import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import type { BoundedJobState } from '../../hooks/useBoundedJob';
import { type PMTUDResult, usePathMTU } from '../../hooks/usePathChecks';
import { cn, status as statusColor } from '../../styles/theme';
import { Card, type Status } from '../ui/Card';
import { Maximize2 } from '../ui/Icons';
import { isBusy, PathCheckForm } from './PathCheckForm';

// The search never probes above discovery.JumboMTU (#435), so on a link with
// a larger MTU (loopback's is 65536) reaching it means "at least this much",
// not a bottleneck below the local link.
const JUMBO_CEILING = 9000;

type Reading = 'full' | 'ceiling' | 'smaller';

function readingOf(result: PMTUDResult): Reading {
  if (result.pathMtu >= result.localMtu) {
    return 'full';
  }
  return result.pathMtu >= JUMBO_CEILING ? 'ceiling' : 'smaller';
}

function resultStatus(result: PMTUDResult): Status {
  switch (result.status) {
    case 'ok':
      return readingOf(result) === 'smaller' ? 'warning' : 'success';
    case 'unreachable':
      return 'error';
    default:
      return 'warning';
  }
}

function cardStatus(state: BoundedJobState<PMTUDResult>): Status {
  if (isBusy(state)) {
    return 'loading';
  }
  switch (state.phase) {
    case 'failed':
      return 'error';
    case 'finished':
      return resultStatus(state.result);
    default:
      return 'unknown';
  }
}

export function PathMTUCard(): JSX.Element {
  const { t } = useTranslation('cards');
  const check = usePathMTU();
  const { state } = check;

  return (
    <Card
      title={t('pathMtu.title')}
      icon={<Maximize2 className="w-4 h-4" />}
      status={cardStatus(state)}
      ariaLabel={t('pathMtu.title')}
    >
      <div className="stack-sm" data-testid="path-mtu" data-phase={state.phase}>
        <p className="caption text-text-muted">{t('pathMtu.description')}</p>
        <PathCheckForm
          testId="path-mtu"
          check={check}
          copy={{
            start: t('pathMtu.start'),
            running: (target) => t('pathMtu.running', { target }),
            failed: t('pathMtu.failed'),
          }}
        />
        {state.phase === 'finished' ? <PathMTUView result={state.result} /> : null}
      </div>
    </Card>
  );
}

function PathMTUView({ result }: { result: PMTUDResult }): JSX.Element {
  const { t } = useTranslation('cards');
  const status = resultStatus(result);
  // Each status as a static key so the locale gate sees every one.
  const reading = readingOf(result);
  const verdict = {
    ok: {
      full: t('pathMtu.verdict.full', { local: result.localMtu }),
      ceiling: t('pathMtu.verdict.ceiling'),
      smaller: t('pathMtu.verdict.smaller', { local: result.localMtu }),
    }[reading],
    icmp_filtered: t('pathMtu.verdict.icmp_filtered'),
    unreachable: t('pathMtu.verdict.unreachable'),
    incomplete: t('pathMtu.verdict.incomplete'),
  }[result.status];

  return (
    <div className="stack-xs" data-testid="path-mtu-result" data-status={result.status}>
      {result.status === 'unreachable' ? null : (
        <div className="flex items-baseline gap-tight">
          <span className="heading-2 text-text-primary" data-testid="path-mtu-value">
            {result.status === 'ok' && reading !== 'ceiling'
              ? result.pathMtu
              : t('pathMtu.atLeast', { mtu: result.pathMtu })}
          </span>
          <span className="body-small text-text-muted">{t('pathMtu.bytes')}</span>
        </div>
      )}
      <p
        className={cn(
          'body-small',
          status === 'success'
            ? 'text-text-secondary'
            : status === 'error'
              ? statusColor.text.error
              : statusColor.text.warning,
        )}
        data-testid="path-mtu-verdict"
      >
        {verdict}
      </p>
      <p className="caption text-text-muted" data-testid="path-mtu-probes">
        {t('pathMtu.probes', { count: result.probes, target: result.target })}
      </p>
    </div>
  );
}
