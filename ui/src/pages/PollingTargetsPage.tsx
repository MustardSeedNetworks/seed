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
import { usePollingTargets } from '../hooks/usePollingTargets';
import type { PollingTarget } from '../types/polling';
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
function targetState(target: PollingTarget): RecordState {
  if (!target.enabled) {
    return 'unknown';
  }
  if (!target.lastPolledAt) {
    return 'unknown';
  }
  return target.lastStatus === 'ok' ? 'ok' : 'crit';
}

/** The figure worth seeing without selecting the row: when it last succeeded. */
function lastPollFigure(target: PollingTarget): string {
  if (!target.lastPolledAt) {
    return '—';
  }
  return new Date(target.lastPolledAt).toLocaleTimeString();
}

function matchesFacet(target: PollingTarget, facet: Facet): boolean {
  if (facet === 'failing') {
    return targetState(target) === 'crit';
  }
  if (facet === 'paused') {
    return !target.enabled;
  }
  return true;
}

export function PollingTargetsPage(): JSX.Element {
  const { targets, loading, error, create, update, remove } = usePollingTargets();
  const [editing, setEditing] = useState<PollingTarget | null>(null);
  const [showCreate, setShowCreate] = useState<boolean>(false);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [facet, setFacet] = useState<Facet>('all');
  const [query, setQuery] = useState('');

  const needle = query.trim().toLowerCase();
  const shown = targets.filter(
    (t) =>
      matchesFacet(t, facet) &&
      (!needle ||
        t.name.toLowerCase().includes(needle) ||
        t.ipAddress.toLowerCase().includes(needle)),
  );
  const selected = shown.find((t) => t.id === selectedId) ?? shown[0] ?? null;

  return (
    <>
      {error ? (
        <div className="rounded-md border border-status-error/40 bg-status-error/10 p-3 text-sm text-status-error">
          {error}
        </div>
      ) : null}

      <div className="flex-between">
        <p className="body-small">
          {loading ? 'Loading…' : `${targets.length} target${targets.length === 1 ? '' : 's'}`}
        </p>
        <button
          type="button"
          onClick={(): void => setShowCreate(true)}
          className="inline-flex items-center gap-2 rounded-md bg-brand-primary px-3 py-2 text-sm font-medium text-on-brand hover:bg-brand-accent"
        >
          <Plus className="h-4 w-4" />
          Add target
        </button>
      </div>

      <ListDetail>
        <RecordPane
          filter={
            <input
              type="search"
              value={query}
              onChange={(e): void => setQuery(e.target.value)}
              placeholder={`Filter ${targets.length} targets`}
              aria-label="Filter polling targets"
              className="w-full rounded-lg border border-surface-border bg-surface-sunken px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand-primary focus:outline-none"
            />
          }
          chips={
            <>
              <FilterChip
                label="All"
                count={targets.length}
                active={facet === 'all'}
                onClick={(): void => setFacet('all')}
              />
              <FilterChip
                label="Failing"
                count={targets.filter((t) => targetState(t) === 'crit').length}
                active={facet === 'failing'}
                onClick={(): void => setFacet('failing')}
              />
              <FilterChip
                label="Paused"
                count={targets.filter((t) => !t.enabled).length}
                active={facet === 'paused'}
                onClick={(): void => setFacet('paused')}
              />
            </>
          }
          empty={
            targets.length === 0
              ? 'No polling targets yet. Add one to start polling a device.'
              : 'No target matches this filter.'
          }
        >
          {shown.map((t) => (
            <RecordRow
              key={t.id}
              data-testid={`target-row-${t.id}`}
              name={t.name}
              meta={`${t.ipAddress} · SNMP ${t.snmpVersion}`}
              value={lastPollFigure(t)}
              state={targetState(t)}
              selected={selected?.id === t.id}
              onSelect={(): void => setSelectedId(t.id)}
            />
          ))}
        </RecordPane>

        {selected ? (
          <DetailPane
            eyebrow="Selected target"
            title={selected.name}
            meta={`${selected.ipAddress} · SNMP ${selected.snmpVersion} · every ${selected.pollIntervalSeconds}s`}
            status={<TargetStatus target={selected} />}
            actions={
              <>
                <button
                  type="button"
                  onClick={(): void => setEditing(selected)}
                  className="rounded-md border border-surface-border px-3 py-2 text-sm text-text-primary hover:bg-surface-hover"
                >
                  Edit
                </button>
                <button
                  type="button"
                  onClick={(): void => {
                    if (window.confirm(`Delete polling target "${selected.name}"?`)) {
                      void remove(selected.id);
                      setSelectedId(null);
                    }
                  }}
                  className="rounded-md px-3 py-2 text-sm text-status-error hover:bg-status-error/10"
                >
                  Delete
                </button>
              </>
            }
          >
            <DetailFacts
              items={[
                { label: 'Poll interval', value: `${selected.pollIntervalSeconds}s` },
                { label: 'Enabled', value: selected.enabled ? 'yes' : 'no' },
                {
                  label: 'Last poll',
                  value: selected.lastPolledAt
                    ? new Date(selected.lastPolledAt).toLocaleString()
                    : 'never',
                },
                {
                  label: 'Last status',
                  value: selected.lastError || selected.lastStatus || '—',
                  prose: Boolean(selected.lastError),
                },
              ]}
            />
            <CollectorChain chain={selected.collectorChain} />
          </DetailPane>
        ) : (
          <DetailEmpty>
            {targets.length === 0
              ? 'Add a target to see its polling detail here.'
              : 'Select a target to see its configuration and collector chain.'}
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
    </>
  );
}

/** The record's own state, spelled out rather than left to the colour bar. */
function TargetStatus({ target }: { target: PollingTarget }): JSX.Element {
  const state = targetState(target);
  if (state === 'crit') {
    return (
      <span className="rounded-lg border border-status-error/40 bg-status-error/10 px-3 py-1.5 text-xs font-semibold text-status-error">
        {target.lastError || 'Last poll failed'}
      </span>
    );
  }
  if (state === 'unknown') {
    return (
      <span className="rounded-lg border border-surface-border bg-surface-sunken px-3 py-1.5 text-xs font-semibold text-text-muted">
        {target.enabled ? 'No poll completed yet' : 'Polling paused'}
      </span>
    );
  }
  return (
    <span className="rounded-lg border border-surface-border px-3 py-1.5 text-xs font-semibold text-text-secondary">
      Polling normally
    </span>
  );
}

/** The collector chain is the sub-table the archetype calls for. */
function CollectorChain({ chain }: { chain: string[] }): JSX.Element {
  if (chain.length === 0) {
    return (
      <div className="stack-xs">
        <p className="caption">Collector chain</p>
        <p className="body-small">Using the default chain for this SNMP version.</p>
      </div>
    );
  }
  return (
    <div className="stack-xs">
      <p className="caption">Collector chain</p>
      <ol className="divide-y divide-surface-border overflow-hidden rounded-lg border border-surface-border">
        {chain.map((collector, index) => (
          <li key={collector} className="flex items-center gap-default px-cell py-2">
            <span className="figure text-xs text-text-muted">{index + 1}</span>
            <span className="figure text-sm text-text-primary">{collector}</span>
          </li>
        ))}
      </ol>
    </div>
  );
}
