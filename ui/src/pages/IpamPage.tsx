/**
 * IpamPage — address utilization over /api/v1/ipam (P-D5, UI-SEED-46).
 *
 * Lists every network Seed knows about (the active interface's subnet, the
 * discovery target networks and the learned ones) and every DHCP scope an
 * operator entered, each with how many of its addresses were seen in use.
 * The detail names those addresses. Only a scope carries a state: the server
 * calls it near exhaustion at 90% and full with nothing left. A subnet has no
 * such threshold, since a fully assigned static network is not a fault.
 */

import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Plus } from '../components/ui/Icons';
import { useRole } from '../contexts/RoleContext';
import { useIpam } from '../hooks/useIpam';
import { Alert } from '../shared/Alert';
import { ConfirmDialog } from '../shared/ConfirmDialog';
import { EmptyState } from '../shared/EmptyState';
import { StatusPill } from '../shared/StatusPill';
import type { IPAMScope, IPAMSubnet } from '../types/generated/ipam-subnets-response';
import { GatedButton } from '../ui/GatedButton';
import {
  DetailEmpty,
  DetailFacts,
  DetailPane,
  ListDetail,
  RecordPane,
  RecordRow,
  type RecordState,
} from '../ui/ListDetail';
import { ScopeForm } from './ipam/ScopeForm';

type Selected = { kind: 'subnet'; record: IPAMSubnet } | { kind: 'scope'; record: IPAMScope };

const SCOPE_STATE: Record<IPAMScope['exhaustion'], RecordState> = {
  none: 'ok',
  near: 'warn',
  full: 'crit',
};

const subnetKey = (subnet: IPAMSubnet): string => `subnet:${subnet.cidr}`;
const scopeKey = (scope: IPAMScope): string => `scope:${scope.start}`;

export function IpamPage(): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const { canWrite } = useRole();
  const { data, error, addScope, deleteScope } = useIpam();
  const [selectedKey, setSelectedKey] = useState<string | null>(null);
  const [showAdd, setShowAdd] = useState(false);
  const [deleting, setDeleting] = useState<IPAMScope | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const subnets = data?.subnets ?? [];
  const scopes = data?.scopes ?? [];
  const records: Selected[] = [
    ...subnets.map((record) => ({ kind: 'subnet' as const, record })),
    ...scopes.map((record) => ({ kind: 'scope' as const, record })),
  ];
  const keyOf = (entry: Selected): string =>
    entry.kind === 'subnet' ? subnetKey(entry.record) : scopeKey(entry.record);
  const selected = records.find((entry) => keyOf(entry) === selectedKey) ?? records[0] ?? null;

  const readOnlyReason = canWrite ? undefined : t('ipam.readOnly');
  const percent = (value: number): string =>
    t('ipam.percent', { value: value.toLocaleString(undefined, { maximumFractionDigits: 1 }) });

  const removeScope = async (scope: IPAMScope): Promise<void> => {
    setDeleteError(null);
    await deleteScope(scope.start)
      .then(() => setSelectedKey(null))
      .catch((err: unknown) => setDeleteError(err instanceof Error ? err.message : String(err)));
  };

  return (
    <>
      {error ? <Alert tone="error">{t('ipam.loadFailed', { reason: error })}</Alert> : null}

      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm text-text-secondary" data-testid="ipam-counts">
          {data
            ? `${t('ipam.networkCount', { count: subnets.length })} · ${t('ipam.scopeCount', { count: scopes.length })}`
            : t('common:status.loading')}
        </p>
        <GatedButton
          variant="primary"
          icon={<Plus className="h-4 w-4" />}
          reason={readOnlyReason}
          onClick={(): void => setShowAdd(true)}
          data-testid="scope-add"
        >
          {t('ipam.addScope')}
        </GatedButton>
      </div>

      <ListDetail>
        <RecordPane empty={data ? t('ipam.noNetworks') : t('common:status.loading')}>
          {records.map((entry) => {
            const key = keyOf(entry);
            const select = (): void => {
              setSelectedKey(key);
              setDeleteError(null);
            };
            return entry.kind === 'subnet' ? (
              <RecordRow
                key={key}
                data-testid={`ipam-subnet-${entry.record.cidr}`}
                name={entry.record.cidr}
                meta={[entry.record.name, t(`ipam.source.${entry.record.source}`)]
                  .filter(Boolean)
                  .join(' · ')}
                value={percent(entry.record.utilizationPct)}
                selected={selected !== null && keyOf(selected) === key}
                onSelect={select}
              />
            ) : (
              <RecordRow
                key={key}
                data-testid={`ipam-scope-${entry.record.start}`}
                name={`${entry.record.start} – ${entry.record.end}`}
                meta={[entry.record.name, t('ipam.dhcpScope')].filter(Boolean).join(' · ')}
                value={percent(entry.record.utilizationPct)}
                state={SCOPE_STATE[entry.record.exhaustion]}
                selected={selected !== null && keyOf(selected) === key}
                onSelect={select}
              />
            );
          })}
        </RecordPane>

        {selected ? (
          <DetailPane
            eyebrow={
              selected.kind === 'subnet'
                ? t(`ipam.source.${selected.record.source}`)
                : t('ipam.dhcpScope')
            }
            title={
              selected.kind === 'subnet'
                ? selected.record.cidr
                : `${selected.record.start} – ${selected.record.end}`
            }
            meta={selected.record.name || undefined}
            status={
              selected.kind === 'scope' && selected.record.exhaustion !== 'none' ? (
                <span data-testid="scope-exhaustion">
                  <StatusPill
                    tone={selected.record.exhaustion === 'full' ? 'error' : 'warning'}
                    label={t(`ipam.exhaustion.${selected.record.exhaustion}`)}
                  />
                </span>
              ) : undefined
            }
            actions={
              selected.kind === 'scope' ? (
                <GatedButton
                  variant="destructive"
                  reason={readOnlyReason}
                  onClick={(): void => setDeleting(selected.record)}
                  data-testid="scope-delete"
                >
                  {t('common:buttons.delete')}
                </GatedButton>
              ) : undefined
            }
          >
            {deleteError ? (
              <div data-testid="scope-delete-error">
                <Alert tone="error">{t('ipam.deleteFailed', { reason: deleteError })}</Alert>
              </div>
            ) : null}
            <DetailFacts
              items={[
                {
                  label: t('ipam.labelSize'),
                  testId: 'ipam-size',
                  value: selected.record.size.toLocaleString(),
                },
                {
                  label: t('ipam.labelUsed'),
                  testId: 'ipam-used',
                  value: selected.record.used.toLocaleString(),
                },
                {
                  label: t('ipam.labelFree'),
                  testId: 'ipam-free',
                  value: selected.record.free.toLocaleString(),
                },
                {
                  label: t('ipam.labelUtilization'),
                  testId: 'ipam-utilization',
                  value: percent(selected.record.utilizationPct),
                },
              ]}
            />
            <InUseList addresses={selected.record.inUse} />
          </DetailPane>
        ) : (
          <DetailEmpty>{t('ipam.emptyPrompt')}</DetailEmpty>
        )}
      </ListDetail>

      {showAdd ? (
        <ScopeForm
          onCancel={(): void => setShowAdd(false)}
          onSubmit={async (input): Promise<void> => {
            await addScope(input);
            setSelectedKey(`scope:${input.start}`);
            setShowAdd(false);
          }}
        />
      ) : null}

      <ConfirmDialog
        open={deleting !== null}
        title={t('ipam.confirmDelete', {
          range: deleting ? `${deleting.start} – ${deleting.end}` : '',
        })}
        message={t('ipam.deleteMessage')}
        confirmLabel={t('common:buttons.delete')}
        cancelLabel={t('common:buttons.cancel')}
        closeLabel={t('common:buttons.close')}
        onConfirm={(): void => {
          if (deleting) {
            void removeScope(deleting);
          }
          setDeleting(null);
        }}
        onCancel={(): void => setDeleting(null)}
      />
    </>
  );
}

/** The addresses seen in use, in the address order the server sends. */
function InUseList({ addresses }: { addresses: string[] }): JSX.Element {
  const { t } = useTranslation(['pages']);
  return (
    <div className="space-y-1">
      <p className="kicker">{t('ipam.inUseHeading', { count: addresses.length })}</p>
      {addresses.length === 0 ? (
        <div data-testid="ipam-in-use-empty">
          <EmptyState title={t('ipam.noneInUse')} />
        </div>
      ) : (
        <ul
          data-testid="ipam-in-use"
          className="grid grid-cols-[repeat(auto-fill,minmax(9rem,1fr))] gap-x-4 gap-y-1 rounded-lg border border-hairline px-3 py-2"
        >
          {addresses.map((address) => (
            <li key={address} className="figure text-sm text-text-primary">
              {address}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
