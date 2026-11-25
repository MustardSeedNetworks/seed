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
import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useBonjourBrowse } from '../../hooks/useBonjourBrowse';
import { useCardStatusLabel } from '../../hooks/useCardStatusLabel';
import { useTableLabels } from '../../hooks/useTableLabels';
import { Alert } from '../../shared/Alert';
import { Button } from '../../shared/Button';
import { Card, type CardStatus } from '../../shared/Card';
import { DataTable, type DataTableColumn } from '../../shared/DataTable';
import { EmptyState } from '../../shared/EmptyState';
import { SearchInput } from '../../shared/SearchInput';
import type {
  ServiceInstance as BonjourService,
  BrowseResult,
  ReflectorStatus,
} from '../../types/generated/bonjour-browse-response';
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
  const { t } = useTranslation(['cards', 'common']);
  const labelFor = useCardStatusLabel();
  const { result, loading, error, browse } = useBonjourBrowse();

  let cardStatus: CardStatus = 'unknown';
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
      statusLabel={labelFor(cardStatus)}
      aria-label={t('bonjour.title')}
      data-testid="card"
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
          <Alert tone="error" size="sm" data-testid="bonjour-error">
            {error}
          </Alert>
        ) : null}

        {result ? <ReflectorVerdict result={result} /> : null}
        {result ? <ServiceTable result={result} /> : null}
      </div>
    </Card>
  );
}

function ReflectorVerdict({ result }: { result: BrowseResult }): JSX.Element {
  const { t } = useTranslation(['cards', 'common']);
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
  const { t } = useTranslation(['cards', 'common']);
  const [filter, setFilter] = useState('');
  const labels = useTableLabels<BonjourService>({ loading: t('bonjour.title') });

  const needle = filter.trim().toLowerCase();
  const visible = result.services.filter((service) =>
    [service.instance, service.type, service.host, String(service.port)].some((value) =>
      value?.toLowerCase().includes(needle),
    ),
  );

  const columns: DataTableColumn<BonjourService>[] = [
    {
      key: 'instance',
      header: t('bonjour.instance'),
      sortValue: (service) => service.instance,
      cell: (service) => service.instance,
    },
    {
      key: 'type',
      header: t('bonjour.type'),
      sortValue: (service) => service.type,
      cell: (service) => <span className="font-mono">{service.type}</span>,
    },
    {
      key: 'host',
      header: t('bonjour.host'),
      sortValue: (service) => service.host ?? '',
      cell: (service) => (
        <span className="font-mono">{service.host ?? t('bonjour.hostUnknown')}</span>
      ),
    },
    {
      key: 'port',
      header: t('bonjour.port'),
      sortValue: (service) => service.port,
      // A zero port means no SRV record was seen, not port 0.
      cell: (service) => (
        <span className="font-mono">
          {service.port === 0 ? t('bonjour.portUnknown') : service.port}
        </span>
      ),
    },
    {
      key: 'origin',
      header: t('bonjour.origin'),
      sortValue: (service) => originLabel(t, service.origin),
      cell: (service) => originLabel(t, service.origin),
    },
  ];

  if (result.services.length === 0) {
    return (
      <p className="caption text-text-muted" data-testid="bonjour-empty">
        {t('bonjour.empty')}
      </p>
    );
  }

  return (
    <>
      <SearchInput
        value={filter}
        onValueChange={setFilter}
        label={t('common:dataTable.filterRows')}
        clearLabel={t('common:dataTable.clearFilter')}
      />
      <DataTable
        rows={visible}
        columns={columns}
        getRowKey={(service) => `${service.instance}.${service.type}`}
        caption={t('bonjour.tableCaption')}
        empty={<EmptyState title={t('common:dataTable.noMatches')} />}
        labels={labels}
        pageSize={10}
        data-testid="bonjour-services"
      />
    </>
  );
}
