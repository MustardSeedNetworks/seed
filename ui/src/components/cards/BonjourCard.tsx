/**
 * BonjourCard — what Bonjour services are advertised on this segment, and
 * whether mDNS from another subnet is reaching it (#364).
 *
 * The cross-subnet answer is the reason the card exists. AirPlay and AirPrint
 * fail between VLANs because mDNS is link-local multicast; the question a
 * technician at the wall port has is whether a reflector is actually
 * forwarding, and the honest answer has four states, not two. The server
 * returns the state and its evidence; the sentence is composed here so it
 * exists in every locale.
 */

import type { TFunction } from 'i18next';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import { useBonjourBrowse } from '../../hooks/useBonjourBrowse';
import { cn, radius } from '../../styles/theme';
import type { BrowseResult, ReflectorStatus } from '../../types/generated/bonjour-browse-response';
import { Button } from '../ui/Button';
import { Card, type Status } from '../ui/Card';
import { Network } from '../ui/Icons';

const MS_PER_SECOND = 1000;

/**
 * Every key is a literal inside its own `t()` call, not a value looked up from
 * a map and not a template literal. Both alternatives are invisible to the
 * tooling: a template literal is dropped by the extractor, so the string it
 * names never reaches the locale files, and a map of literals is invisible to
 * `check-keys.py`, which then reports the keys as unreferenced and cannot warn
 * when one really does go unused.
 */
function reflectorSentence(
  t: TFunction<'cards'>,
  reflector: ReflectorStatus,
  seconds: number,
): string {
  const evidence = {
    seconds,
    subnets: (reflector.remoteSubnets ?? []).join(', '),
    forwarders: (reflector.forwardedBy ?? []).join(', '),
    sources: (reflector.routedFrom ?? []).join(', '),
  };
  switch (reflector.state) {
    case 'no-traffic':
      return t('bonjour.reflector.no-traffic', evidence);
    case 'local-only':
      return t('bonjour.reflector.local-only', evidence);
    case 'reflected':
      return t('bonjour.reflector.reflected', evidence);
    case 'routed':
      return t('bonjour.reflector.routed', evidence);
    default:
      // The server may be newer than this build. Saying so beats rendering a
      // raw key at the operator.
      return t('bonjour.reflector.unknown');
  }
}

function originLabel(t: TFunction<'cards'>, origin: string): string {
  switch (origin) {
    case 'local':
      return t('bonjour.originValue.local');
    case 'off-segment':
      return t('bonjour.originValue.off-segment');
    default:
      return t('bonjour.originValue.unknown');
  }
}

export function BonjourCard(): JSX.Element {
  const { t } = useTranslation('cards');
  const { result, loading, error, browse } = useBonjourBrowse();

  let cardStatus: Status = 'unknown';
  if (loading) {
    cardStatus = 'loading';
  } else if (error) {
    cardStatus = 'error';
  } else if (result) {
    // Reflected traffic is not a fault — it is often exactly what the operator
    // configured — so it reads as a warning only in the sense of "something
    // crosses the boundary here", never as an error.
    cardStatus = result.reflector.state === 'reflected' ? 'warning' : 'success';
  }

  return (
    <Card
      title={t('bonjour.title')}
      icon={<Network className="w-4 h-4" />}
      status={cardStatus}
      ariaLabel={t('bonjour.title')}
      // See NeighbourCacheCard: a service table needs more than a quarter
      // of a 4-up grid (#2708).
      className="sm:col-span-2"
    >
      <div className="stack-sm">
        <p className="caption text-text-muted">{t('bonjour.description')}</p>

        <Button
          variant="secondary"
          size="sm"
          onClick={() => void browse()}
          disabled={loading}
          data-testid="bonjour-browse"
        >
          {loading ? t('bonjour.browsing') : t('bonjour.browse')}
        </Button>

        {error ? (
          <p className="caption text-status-error" data-testid="bonjour-error">
            {error}
          </p>
        ) : null}

        {result ? <ReflectorVerdict result={result} /> : null}
        {result ? <ServiceTable result={result} /> : null}
      </div>
    </Card>
  );
}

function ReflectorVerdict({ result }: { result: BrowseResult }): JSX.Element {
  const { t } = useTranslation('cards');
  const { state } = result.reflector;

  const sentence = reflectorSentence(
    t,
    result.reflector,
    Math.round(result.durationMs / MS_PER_SECOND),
  );

  return (
    <p className="body-small" data-testid="bonjour-reflector" data-state={state}>
      {sentence}
    </p>
  );
}

function ServiceTable({ result }: { result: BrowseResult }): JSX.Element {
  const { t } = useTranslation('cards');

  if (result.services.length === 0) {
    return (
      <p className="caption text-text-muted" data-testid="bonjour-empty">
        {t('bonjour.empty')}
      </p>
    );
  }

  return (
    // See NeighbourCacheCard: table-fixed, not overflow-x-auto (#2708).
    <div className={cn(radius.default, 'border border-surface-border')}>
      <table className="w-full table-fixed body-small">
        <caption className="sr-only">{t('bonjour.tableCaption')}</caption>
        <thead>
          <tr className="border-b border-surface-border text-text-muted">
            <th scope="col" className="px-cell py-row text-left w-[28%] truncate">
              {t('bonjour.instance')}
            </th>
            <th scope="col" className="px-cell py-row text-left w-[24%] truncate">
              {t('bonjour.type')}
            </th>
            <th scope="col" className="px-cell py-row text-left w-[24%] truncate">
              {t('bonjour.host')}
            </th>
            <th scope="col" className="px-cell py-row text-left w-[10%] truncate">
              {t('bonjour.port')}
            </th>
            <th scope="col" className="px-cell py-row text-left w-[14%] truncate">
              {t('bonjour.origin')}
            </th>
          </tr>
        </thead>
        <tbody data-testid="bonjour-services">
          {result.services.map((service) => (
            <tr
              key={`${service.instance}.${service.type}`}
              className="border-b border-surface-border last:border-0"
            >
              <td className="px-cell py-row truncate" title={service.instance}>
                {service.instance}
              </td>
              <td className="px-cell py-row font-mono truncate" title={service.type}>
                {service.type}
              </td>
              <td className="px-cell py-row font-mono truncate" title={service.host ?? undefined}>
                {service.host ?? t('bonjour.hostUnknown')}
              </td>
              <td className="px-cell py-row font-mono truncate">
                {/* A zero port means no SRV record was seen, not port 0. */}
                {service.port === 0 ? t('bonjour.portUnknown') : service.port}
              </td>
              <td className="px-cell py-row truncate">{originLabel(t, service.origin)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
