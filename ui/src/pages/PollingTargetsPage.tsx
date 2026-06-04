/**
 * PollingTargetsPage
 *
 * Operator-facing CRUD over /api/v1/polling-targets. Renders the
 * current list of SNMP-polled devices, lets operators add a new
 * target, edit an existing one, and remove targets they're done
 * with. New targets pick up the default collector chain
 * (sys_info, if_table, lldp, arp, fdb) and start polling on the
 * next snmp-poller tick (~5s).
 *
 * This is the entry point for the V1.0 NMS workflow:
 *   1. Operator adds a target here.
 *   2. snmp-poller dispatches its collector chain.
 *   3. Topology reconcilers fold observations into the fat-Node graph.
 *   4. Alert pipelines emit on transitions.
 *   5. Operator sees the device + edges on the topology page and
 *      acts on alerts as they arrive.
 */

import { Plus, Server, X } from 'lucide-react';
import { type FormEvent, type JSX, useState } from 'react';
import { usePollingTargets } from '../hooks/usePollingTargets';
import type { PollingTarget, PollingTargetInput } from '../types/polling';
import { Breadcrumbs } from '../ui/Breadcrumbs';
import { PageHeader } from '../ui/PageHeader';

export function PollingTargetsPage(): JSX.Element {
  const { targets, loading, error, create, update, remove } = usePollingTargets();
  const [editing, setEditing] = useState<PollingTarget | null>(null);
  const [showCreate, setShowCreate] = useState<boolean>(false);

  return (
    <section className="stack-xl">
      <Breadcrumbs />
      <PageHeader
        icon={Server}
        title="Polling targets"
        description="SNMP-polled devices. New targets pick up the default collector chain and start polling on the next tick."
        iconColorClass="text-module-security"
      />

      {error ? (
        <div className="rounded-md border border-status-error/40 bg-status-error/10 p-3 text-sm text-status-error">
          {error}
        </div>
      ) : null}

      <div className="flex items-center justify-between">
        <p className="text-sm text-text-muted">
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

      <TargetTable
        targets={targets}
        onEdit={(t): void => setEditing(t)}
        onDelete={(t): void => {
          if (window.confirm(`Delete polling target "${t.name}"?`)) {
            void remove(t.id);
          }
        }}
      />

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
    </section>
  );
}

interface TargetTableProps {
  targets: PollingTarget[];
  onEdit: (t: PollingTarget) => void;
  onDelete: (t: PollingTarget) => void;
}

function TargetTable({ targets, onEdit, onDelete }: TargetTableProps): JSX.Element {
  if (targets.length === 0) {
    return (
      <div className="rounded-md border border-surface-border bg-surface-raised p-6 text-center text-sm text-text-muted">
        No polling targets yet. Click <strong>Add target</strong> to start polling a device.
      </div>
    );
  }
  return (
    <div className="overflow-hidden rounded-lg border border-surface-border bg-surface-raised">
      <table className="w-full text-sm">
        <thead className="text-left text-xs uppercase tracking-wide text-text-muted">
          <tr>
            <th className="px-4 py-2">Name</th>
            <th className="px-4 py-2">IP</th>
            <th className="px-4 py-2">SNMP</th>
            <th className="px-4 py-2">Interval</th>
            <th className="px-4 py-2">Enabled</th>
            <th className="px-4 py-2">Last poll</th>
            <th className="px-4 py-2 text-right">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-surface-border">
          {targets.map((t) => (
            <tr key={t.id} data-testid={`target-row-${t.id}`}>
              <td className="px-4 py-2 font-medium text-text-primary">{t.name}</td>
              <td className="px-4 py-2 font-mono text-text-secondary">{t.ipAddress}</td>
              <td className="px-4 py-2 text-text-secondary">{t.snmpVersion}</td>
              <td className="px-4 py-2 text-text-secondary">{t.pollIntervalSeconds}s</td>
              <td className="px-4 py-2">
                <EnabledBadge enabled={t.enabled} />
              </td>
              <td className="px-4 py-2 text-text-muted">
                <LastPoll target={t} />
              </td>
              <td className="px-4 py-2 text-right">
                <button
                  type="button"
                  onClick={(): void => onEdit(t)}
                  className="mr-2 text-sm text-status-info hover:text-status-info/80"
                >
                  Edit
                </button>
                <button
                  type="button"
                  onClick={(): void => onDelete(t)}
                  className="text-sm text-status-error hover:text-status-error/80"
                >
                  Delete
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function EnabledBadge({ enabled }: { enabled: boolean }): JSX.Element {
  return (
    <span
      className={
        enabled
          ? 'rounded-full bg-status-success/20 px-2 py-0.5 text-xs font-medium text-status-success'
          : 'rounded-full bg-surface-sunken px-2 py-0.5 text-xs font-medium text-text-muted'
      }
    >
      {enabled ? 'Enabled' : 'Disabled'}
    </span>
  );
}

function LastPoll({ target }: { target: PollingTarget }): JSX.Element {
  if (!target.lastPolledAt) {
    return <span className="text-text-muted">never</span>;
  }
  const when = new Date(target.lastPolledAt).toLocaleString();
  const ok = target.lastStatus === 'ok';
  return (
    <span title={target.lastError || target.lastStatus}>
      <span className={ok ? 'text-status-success' : 'text-status-error'}>●</span> {when}
    </span>
  );
}

interface TargetFormProps {
  mode: 'create' | 'edit';
  initial: PollingTargetInput;
  onSubmit: (input: PollingTargetInput) => Promise<void>;
  onCancel: () => void;
}

function TargetForm({ mode, initial, onSubmit, onCancel }: TargetFormProps): JSX.Element {
  const [form, setForm] = useState<PollingTargetInput>(initial);
  const [submitting, setSubmitting] = useState<boolean>(false);
  const [formError, setFormError] = useState<string | null>(null);

  function update<K extends keyof PollingTargetInput>(key: K, value: PollingTargetInput[K]): void {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  async function handleSubmit(e: FormEvent<HTMLFormElement>): Promise<void> {
    e.preventDefault();
    if (!form.name.trim() || !form.ipAddress.trim()) {
      setFormError('Name and IP address are required.');
      return;
    }
    setSubmitting(true);
    setFormError(null);
    try {
      await onSubmit(form);
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to save');
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-scrim/60">
      <form
        onSubmit={(e): void => {
          void handleSubmit(e);
        }}
        className="w-full max-w-md rounded-lg border border-surface-border bg-surface-raised p-6 shadow-xl"
      >
        <div className="flex items-center justify-between border-b border-surface-border pb-3">
          <h2 className="text-lg font-semibold text-text-primary">
            {mode === 'create' ? 'Add polling target' : 'Edit polling target'}
          </h2>
          <button
            type="button"
            onClick={onCancel}
            className="text-text-muted hover:text-text-primary"
            aria-label="Close"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {formError ? (
          <div className="mt-3 rounded-md border border-status-error/40 bg-status-error/10 p-2 text-sm text-status-error">
            {formError}
          </div>
        ) : null}

        <div className="mt-4 space-y-3">
          <Field label="Name">
            <input
              type="text"
              value={form.name}
              onChange={(e): void => update('name', e.target.value)}
              required
              className={inputClass}
            />
          </Field>
          <Field label="IP address">
            <input
              type="text"
              value={form.ipAddress}
              onChange={(e): void => update('ipAddress', e.target.value)}
              required
              placeholder="10.0.0.1"
              className={inputClass}
            />
          </Field>
          <Field label="SNMP version">
            <select
              value={form.snmpVersion}
              onChange={(e): void => update('snmpVersion', e.target.value)}
              className={inputClass}
            >
              <option value="v2c">v2c</option>
              <option value="v3">v3</option>
            </select>
          </Field>
          <Field label="Poll interval (seconds)">
            <input
              type="number"
              min={10}
              max={3600}
              value={form.pollIntervalSeconds ?? 300}
              onChange={(e): void => update('pollIntervalSeconds', Number(e.target.value))}
              className={inputClass}
            />
          </Field>
          <label className="flex items-center gap-2 text-sm text-text-secondary">
            <input
              type="checkbox"
              checked={form.enabled}
              onChange={(e): void => update('enabled', e.target.checked)}
            />
            Enabled (polled on next tick)
          </label>
        </div>

        <div className="mt-5 flex justify-end gap-2 border-t border-surface-border pt-4">
          <button
            type="button"
            onClick={onCancel}
            className="rounded-md px-3 py-2 text-sm text-text-muted hover:text-text-primary"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={submitting}
            className="rounded-md bg-brand-primary px-3 py-2 text-sm font-medium text-on-brand hover:bg-brand-accent disabled:opacity-60"
          >
            {submitting ? 'Saving…' : mode === 'create' ? 'Add target' : 'Save changes'}
          </button>
        </div>
      </form>
    </div>
  );
}

function Field({ label, children }: { label: string; children: JSX.Element }): JSX.Element {
  // Using a div wrapper rather than a bare label avoids the
  // a11y/noLabelWithoutControl warning when children is a select or
  // a wrapped composite — the inner input element is itself a
  // labelable element which screen readers find via the surrounding
  // <span> text.
  return (
    <div className="block">
      <span className="block text-xs font-medium uppercase tracking-wide text-text-muted">
        {label}
      </span>
      <span className="mt-1 block">{children}</span>
    </div>
  );
}

const inputClass: string =
  'w-full rounded-md border border-surface-border bg-surface-sunken px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand-primary focus:outline-none';

/** emptyInput is the create-form default. Mirrors the server defaults
 * but explicit so the operator sees them in the form before submit. */
function emptyInput(): PollingTargetInput {
  return {
    name: '',
    ipAddress: '',
    snmpVersion: 'v2c',
    pollIntervalSeconds: 300,
    enabled: true,
    collectorChain: [],
  };
}

/** targetToInput strips audit columns the server manages. */
function targetToInput(t: PollingTarget): PollingTargetInput {
  return {
    name: t.name,
    ipAddress: t.ipAddress,
    snmpVersion: t.snmpVersion,
    credentialsId: t.credentialsId || undefined,
    pollIntervalSeconds: t.pollIntervalSeconds,
    enabled: t.enabled,
    collectorChain: t.collectorChain,
  };
}
