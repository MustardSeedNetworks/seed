/**
 * MultiPathCard — does traffic to one destination take more than one route
 * (#395).
 *
 * ECMP and SD-WAN spread flows over several routes, and a single traceroute
 * samples one of them, so a healthy trace can sit beside slow connections
 * that went another way. The `path-multipath` job traces the destination
 * several times with a different flow each time; the card lists the distinct
 * routes, how many traces took each, and the hop where they split.
 */

import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import type { BoundedJobState } from '../../hooks/useBoundedJob';
import { type MultiPathResult, type PathVariant, useMultiPath } from '../../hooks/usePathChecks';
import { cn, radius } from '../../styles/theme';
import { Card, type Status } from '../ui/Card';
import { Network } from '../ui/Icons';
import { isBusy, PathCheckForm } from './PathCheckForm';

function resultStatus(result: MultiPathResult): Status {
  return result.paths.some((path) => path.completed) ? 'success' : 'warning';
}

function cardStatus(state: BoundedJobState<MultiPathResult>): Status {
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

export function MultiPathCard(): JSX.Element {
  const { t } = useTranslation('cards');
  const check = useMultiPath();
  const { state } = check;

  return (
    <Card
      title={t('multiPath.title')}
      icon={<Network className="w-4 h-4" />}
      status={cardStatus(state)}
      ariaLabel={t('multiPath.title')}
    >
      <div className="stack-sm" data-testid="multi-path" data-phase={state.phase}>
        <p className="caption text-text-muted">{t('multiPath.description')}</p>
        <PathCheckForm
          testId="multi-path"
          check={check}
          copy={{
            start: t('multiPath.start'),
            running: (target) => t('multiPath.running', { target }),
            failed: t('multiPath.failed'),
          }}
        />
        {state.phase === 'finished' ? <MultiPathView result={state.result} /> : null}
      </div>
    </Card>
  );
}

function MultiPathView({ result }: { result: MultiPathResult }): JSX.Element {
  const { t } = useTranslation('cards');
  const summary =
    result.paths.length === 0
      ? t('multiPath.noRoute', { count: result.attempts })
      : result.paths.length === 1
        ? t('multiPath.oneRoute', { target: result.target, count: result.attempts })
        : t('multiPath.routes', {
            target: result.target,
            count: result.paths.length,
            ttl: result.divergesAtTtl,
          });

  return (
    <div className="stack-xs" data-testid="multi-path-result">
      <p
        className="body-small text-text-primary"
        data-testid="multi-path-summary"
        data-routes={result.paths.length}
      >
        {summary}
      </p>
      {result.paths.map((path, index) => (
        <Route
          // Routes are distinct by their hop list, ordered most-travelled first.
          key={path.hops.join('>')}
          index={index}
          path={path}
          attempts={result.attempts}
          splitAt={result.divergesAtTtl}
        />
      ))}
    </div>
  );
}

function Route({
  index,
  path,
  attempts,
  splitAt,
}: {
  index: number;
  path: PathVariant;
  attempts: number;
  splitAt: number;
}): JSX.Element {
  const { t } = useTranslation('cards');
  // A hop that did not answer is an empty address at its TTL.
  const hops = path.hops.map((address, hop) => ({ ttl: hop + 1, address }));

  return (
    <div
      className={cn(radius.default, 'border border-surface-border px-cell py-row stack-xs')}
      data-testid="multi-path-route"
      data-completed={path.completed}
    >
      <p className="caption text-text-secondary" data-testid="multi-path-route-seen">
        {t('multiPath.route', { number: index + 1, seen: path.seen, count: attempts })}
        {path.completed ? null : ` · ${t('multiPath.notReached')}`}
      </p>
      <ol className="body-small font-mono">
        {hops.map(({ ttl, address }) => (
          <li
            key={ttl}
            className={cn(
              'flex gap-compact',
              ttl === splitAt ? 'font-semibold text-text-primary' : 'text-text-secondary',
            )}
            data-testid="multi-path-hop"
            data-ttl={ttl}
            data-split={ttl === splitAt || undefined}
          >
            <span className="w-6 shrink-0 text-right text-text-muted">{ttl}</span>
            <span className="truncate" title={address || undefined}>
              {address || '*'}
            </span>
          </li>
        ))}
      </ol>
    </div>
  );
}
