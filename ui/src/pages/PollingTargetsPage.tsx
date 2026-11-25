/**
 * PollingTargetsPage — List + detail.
 *
 * Operator-facing CRUD over /api/v1/polling-targets. Targets are the records:
 * the list carries identity, address and the last poll; the detail pane
 * carries the full configuration and the collector chain that runs against it.
 *
 * This is the entry point for the V1.0 NMS workflow:
 *   1. Operator adds a target here.
 *   2. snmp-poller dispatches its collector chain.
 *   3. Topology reconcilers fold observations into the fat-Node graph.
 *   4. Alert pipelines emit on transitions.
 *   5. Operator sees the device + edges on the topology page and
 *      acts on alerts as they arrive.
 */

import { Plus } from 'lucide-react';
import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useRole } from '../contexts/RoleContext';
import { usePollingTargets } from '../hooks/usePollingTargets';
import { useTableLabels } from '../hooks/useTableLabels';
import { Alert } from '../shared/Alert';
import { ConfirmDialog } from '../shared/ConfirmDialog';
import { DataTable, type DataTableColumn } from '../shared/DataTable';
import { EmptyState } from '../shared/EmptyState';
import { SearchInput } from '../shared/SearchInput';
import { StatusPill } from '../shared/StatusPill';
import type { PollingTargetResponse } from '../types/generated/polling-target-response';
import { GatedButton } from '../ui/GatedButton';
import {
  DetailEmpty,
  DetailFacts,
  DetailPane,
  FilterChip,
  ListDetail,
  RecordPane,
  RecordRow,
  type RecordState,
} from '../ui/ListDetail';
import { emptyInput, TargetForm, targetToInput } from './pollingTargets/TargetForm';

type Facet = 'all' | 'failing' | 'paused';

/**
 * A target that is not enabled is not being polled, so its health is not
 * merely fine — it is unmeasured, and says so rather than showing green.
 * A target that has never completed a poll is in the same position.
 */
function targetState(target: PollingTargetResponse): RecordState {
  if (!target.enabled) {
    return 'unknown';
  }
  if (!target.lastPolledAt) {
    return 'unknown';
  }
  return target.lastStatus === 'ok' ? 'ok' : 'crit';
}

/** The figure worth seeing without selecting the row: when it last succeeded. */
function lastPollFigure(target: PollingTargetResponse): string {
  if (!target.lastPolledAt) {
    return '—';
  }
  return new Date(target.lastPolledAt).toLocaleTimeString();
}

function matchesFacet(target: PollingTargetResponse, facet: Facet): boolean {
  if (facet === 'failing') {
    return targetState(target) === 'crit';
  }
  if (facet === 'paused') {
    return !target.enabled;
  }
  return true;
}

export function PollingTargetsPage(): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const { canWrite } = useRole();
  const { targets, loading, error, create, update, remove, poll } = usePollingTargets();
  const [editing, setEditing] = useState<PollingTargetResponse | null>(null);
  const [showCreate, setShowCreate] = useState<boolean>(false);
  const [deleting, setDeleting] = useState<PollingTargetResponse | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [facet, setFacet] = useState<Facet>('all');
  const [query, setQuery] = useState('');
  const [testing, setTesting] = useState<boolean>(false);
  const [testError, setTestError] = useState<{ id: string; reason: string } | null>(null);

  const needle = query.trim().toLowerCase();
  const shown = targets.filter(
    (target) =>
      matchesFacet(target, facet) &&
      (!needle ||
        target.name.toLowerCase().includes(needle) ||
        target.ipAddress.toLowerCase().includes(needle)),
  );
  const selected = shown.find((target) => target.id === selectedId) ?? shown[0] ?? null;

  // The whole /polling-targets collection is minRole: op, so for a viewer add,
  // edit and delete could only 403 (#1254). The list and detail stay readable:
  // writeGated passes GET for every role.
  const readOnlyReason = canWrite ? undefined : t('pollingTargets.readOnly');

  // A poll that ran and failed is a result, shown in the target's own status;
  // testError is only for a poll that could not run at all.
  const testTarget = async (id: string): Promise<void> => {
    setTesting(true);
    setTestError(null);
    await poll(id).catch((err: unknown) => {
      setTestError({ id, reason: err instanceof Error ? err.message : String(err) });
    });
    setTesting(false);
  };

  return (
    <>
      {error ? <Alert tone="error">{error}</Alert> : null}

      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm text-text-secondary">
          {loading
            ? t('common:status.loading')
            : t('pollingTargets.targetCount', { count: targets.length })}
        </p>
        <GatedButton
          variant="primary"
          icon={<Plus className="h-4 w-4" />}
          reason={readOnlyReason}
          onClick={(): void => setShowCreate(true)}
          data-testid="target-add"
        >
          {t('pollingTargets.addTarget')}
        </GatedButton>
      </div>

      <ListDetail>
        <RecordPane
          filter={
            <SearchInput
              value={query}
              onValueChange={setQuery}
              placeholder={t('pollingTargets.filterPlaceholder', { count: targets.length })}
              label={t('pollingTargets.filterLabel')}
              clearLabel={t('common:buttons.clear')}
              className="max-w-none"
            />
          }
          chips={
            <>
              <FilterChip
                label={t('pollingTargets.facetAll')}
                count={targets.length}
                active={facet === 'all'}
                onClick={(): void => setFacet('all')}
              />
              <FilterChip
                label={t('pollingTargets.facetFailing')}
                count={targets.filter((target) => targetState(target) === 'crit').length}
                active={facet === 'failing'}
                onClick={(): void => setFacet('failing')}
              />
              <FilterChip
                label={t('pollingTargets.facetPaused')}
                count={targets.filter((target) => !target.enabled).length}
                active={facet === 'paused'}
                onClick={(): void => setFacet('paused')}
              />
            </>
          }
          empty={targets.length === 0 ? t('pollingTargets.noTargets') : t('pollingTargets.noMatch')}
        >
          {shown.map((target) => (
            <RecordRow
              key={target.id}
              data-testid={`target-row-${target.id}`}
              name={target.name}
              meta={`${target.ipAddress} · SNMP ${target.snmpVersion}`}
              value={lastPollFigure(target)}
              state={targetState(target)}
              selected={selected?.id === target.id}
              onSelect={(): void => setSelectedId(target.id)}
            />
          ))}
        </RecordPane>

        {selected ? (
          <DetailPane
            eyebrow={t('pollingTargets.selectedTarget')}
            title={selected.name}
            meta={`${selected.ipAddress} · SNMP ${selected.snmpVersion} · ${t('pollingTargets.everySeconds', { seconds: selected.pollIntervalSeconds })}`}
            status={<TargetStatus target={selected} />}
            actions={
              <>
                <GatedButton
                  reason={readOnlyReason}
                  hint={t('pollingTargets.testTargetHint')}
                  onClick={(): void => {
                    void testTarget(selected.id);
                  }}
                  disabled={testing}
                  data-testid="target-test"
                >
                  {testing ? t('pollingTargets.testTargetRunning') : t('pollingTargets.testTarget')}
                </GatedButton>
                <GatedButton
                  reason={readOnlyReason}
                  onClick={(): void => setEditing(selected)}
                  data-testid="target-edit"
                >
                  {t('common:buttons.edit')}
                </GatedButton>
                <GatedButton
                  variant="destructive"
                  reason={readOnlyReason}
                  onClick={(): void => setDeleting(selected)}
                  data-testid="target-delete"
                >
                  {t('common:buttons.delete')}
                </GatedButton>
              </>
            }
          >
            <DetailFacts
              items={[
                {
                  label: t('pollingTargets.labelInterval'),
                  value: `${selected.pollIntervalSeconds}s`,
                },
                {
                  label: t('pollingTargets.labelEnabled'),
                  value: selected.enabled ? t('common:status.yes') : t('common:status.no'),
                },
                {
                  label: t('pollingTargets.labelLastPoll'),
                  value: selected.lastPolledAt
                    ? new Date(selected.lastPolledAt).toLocaleString()
                    : t('common:status.never'),
                },
                {
                  label: t('pollingTargets.labelLastSuccess'),
                  testId: 'target-last-success',
                  value: selected.lastSuccessAt
                    ? new Date(selected.lastSuccessAt).toLocaleString()
                    : t('common:status.never'),
                },
                {
                  label: t('pollingTargets.labelLastStatus'),
                  testId: 'target-last-status',
                  value: selected.lastError || selected.lastStatus || '—',
                  prose: Boolean(selected.lastError),
                },
              ]}
            />
            {testError?.id === selected.id ? (
              <div data-testid="target-test-error">
                <Alert tone="error">
                  {t('pollingTargets.testTargetFailed', { reason: testError.reason })}
                </Alert>
              </div>
            ) : null}
            <CollectorChain chain={selected.collectorChain} />
          </DetailPane>
        ) : (
          <DetailEmpty>
            {targets.length === 0
              ? t('pollingTargets.emptyAddPrompt')
              : t('pollingTargets.emptySelectPrompt')}
          </DetailEmpty>
        )}
      </ListDetail>

      {showCreate ? (
        <TargetForm
          mode="create"
          initial={emptyInput()}
          onCancel={(): void => setShowCreate(false)}
          onSubmit={async (input): Promise<void> => {
            await create(input);
            setShowCreate(false);
          }}
        />
      ) : null}

      {editing ? (
        <TargetForm
          mode="edit"
          initial={targetToInput(editing)}
          onCancel={(): void => setEditing(null)}
          onSubmit={async (input): Promise<void> => {
            await update(editing.id, input);
            setEditing(null);
          }}
        />
      ) : null}

      <ConfirmDialog
        open={deleting !== null}
        title={t('pollingTargets.confirmDelete', { name: deleting?.name ?? '' })}
        message={t('pollingTargets.deleteMessage')}
        confirmLabel={t('common:buttons.delete')}
        cancelLabel={t('common:buttons.cancel')}
        closeLabel={t('common:buttons.close')}
        onConfirm={(): void => {
          if (deleting) {
            void remove(deleting.id);
            setSelectedId(null);
          }
          setDeleting(null);
        }}
        onCancel={(): void => setDeleting(null)}
      />
    </>
  );
}

/** The record's own state, spelled out rather than left to the colour bar. */
function TargetStatus({ target }: { target: PollingTargetResponse }): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const state = targetState(target);
  if (state === 'crit') {
    return (
      <StatusPill tone="error" label={target.lastError || t('pollingTargets.lastPollFailed')} />
    );
  }
  if (state === 'unknown') {
    return (
      <StatusPill
        tone="unknown"
        label={target.enabled ? t('pollingTargets.noPollYet') : t('pollingTargets.pollingPaused')}
      />
    );
  }
  return <StatusPill tone="success" label={t('pollingTargets.pollingNormally')} />;
}

interface ChainStep {
  position: number;
  collector: string;
}

/** The collector chain is the sub-table the archetype calls for. */
function CollectorChain({ chain }: { chain: string[] }): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const labels = useTableLabels<ChainStep>({ loading: t('common:status.loading') });
  const steps = chain.map((collector, index) => ({ position: index + 1, collector }));
  const columns: DataTableColumn<ChainStep>[] = [
    {
      key: 'position',
      header: t('pollingTargets.columnOrder'),
      cell: (step) => <span className="figure text-text-muted">{step.position}</span>,
      sortValue: (step) => step.position,
    },
    {
      key: 'collector',
      header: t('pollingTargets.columnCollector'),
      cell: (step) => <span className="figure text-text-primary">{step.collector}</span>,
      sortValue: (step) => step.collector,
    },
  ];
  return (
    <div className="space-y-1">
      <p className="kicker">{t('pollingTargets.collectorChain')}</p>
      <DataTable
        rows={steps}
        columns={columns}
        getRowKey={(step) => step.collector}
        caption={t('pollingTargets.collectorChain')}
        labels={labels}
        empty={<EmptyState title={t('pollingTargets.defaultChain')} />}
      />
    </div>
  );
}
