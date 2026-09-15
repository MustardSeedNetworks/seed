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

import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import { useBonjourBrowse } from '../../hooks/useBonjourBrowse';
import { cn, radius } from '../../styles/theme';
import type { BrowseResult } from '../../types/generated/bonjour-browse-response';
import { Button } from '../ui/Button';
import { Card, type Status } from '../ui/card';
import { Network } from '../ui/icons';

/**
 * The keys are spelled out rather than built from the state string. A
 * template-literal key is invisible to the i18n extractor, so the string it
 * names is dropped from the locale files and the card renders the key — and
 * the translation gate cannot see it go missing.
 */
const MS_PER_SECOND = 1000;

const REFLECTOR_KEYS = {
  'no-traffic': 'bonjour.reflector.no-traffic',
  'local-only': 'bonjour.reflector.local-only',
  reflected: 'bonjour.reflector.reflected',
  routed: 'bonjour.reflector.routed',
} as const;

const ORIGIN_KEYS = {
  local: 'bonjour.originValue.local',
  'off-segment': 'bonjour.originValue.off-segment',
  unknown: 'bonjour.originValue.unknown',
} as const;

function reflectorKey(state: string): (typeof REFLECTOR_KEYS)[keyof typeof REFLECTOR_KEYS] | null {
  return state in REFLECTOR_KEYS ? REFLECTOR_KEYS[state as keyof typeof REFLECTOR_KEYS] : null;
}

function originKey(origin: string): (typeof ORIGIN_KEYS)[keyof typeof ORIGIN_KEYS] {
  // An origin this build does not know is reported as undecidable, which is
  // what an unrecognised classification actually means to the reader.
  return origin in ORIGIN_KEYS
    ? ORIGIN_KEYS[origin as keyof typeof ORIGIN_KEYS]
    : ORIGIN_KEYS.unknown;
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
  const { state, remoteSubnets, forwardedBy, routedFrom } = result.reflector;

  const key = reflectorKey(state);
  const sentence = key
    ? t(key, {
        seconds: Math.round(result.durationMs / MS_PER_SECOND),
        subnets: (remoteSubnets ?? []).join(', '),
        forwarders: (forwardedBy ?? []).join(', '),
        sources: (routedFrom ?? []).join(', '),
      })
    : // A state this build does not know about is reported as unknown rather
      // than rendered as a raw key: the server may be newer than the UI.
      t('bonjour.reflector.unknown');

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
    <div className={cn('overflow-x-auto', radius.default, 'border border-surface-border')}>
      <table className="w-full body-small">
        <caption className="sr-only">{t('bonjour.tableCaption')}</caption>
        <thead>
          <tr className="border-b border-surface-border text-text-muted">
            <th scope="col" className="px-cell py-row text-left">
              {t('bonjour.instance')}
            </th>
            <th scope="col" className="px-cell py-row text-left">
              {t('bonjour.type')}
            </th>
            <th scope="col" className="px-cell py-row text-left">
              {t('bonjour.host')}
            </th>
            <th scope="col" className="px-cell py-row text-left">
              {t('bonjour.port')}
            </th>
            <th scope="col" className="px-cell py-row text-left">
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
              <td className="px-cell py-row">{service.instance}</td>
              <td className="px-cell py-row font-mono">{service.type}</td>
              <td className="px-cell py-row font-mono">
                {service.host ?? t('bonjour.hostUnknown')}
              </td>
              <td className="px-cell py-row font-mono">
                {/* A zero port means no SRV record was seen, not port 0. */}
                {service.port === 0 ? t('bonjour.portUnknown') : service.port}
              </td>
              <td className="px-cell py-row">{t(originKey(service.origin))}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
