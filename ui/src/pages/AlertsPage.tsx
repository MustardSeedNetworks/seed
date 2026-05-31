/**
 * AlertsPage
 *
 * Operator-facing list of alerts the Stage A4.5/A4.6 pipelines
 * emit. Filters let operators narrow by severity / acknowledged /
 * resolved; per-row actions let them acknowledge ("seen") and
 * resolve ("fixed") individual alerts. The handler routes write
 * X-Username from the JWT/PAT, so the acknowledgedBy column will
 * reflect whoever clicked the button.
 */

import { Bell, Check, CheckCircle2 } from 'lucide-react';
import type { JSX } from 'react';
import { useAlerts } from '../hooks/useAlerts';
import type { Alert } from '../types/alerts';
import { Breadcrumbs } from '../ui/Breadcrumbs';
import { PageHeader } from '../ui/PageHeader';

export function AlertsPage(): JSX.Element {
  const { alerts, loading, error, filter, setFilter, acknowledge, resolve } = useAlerts({
    unresolvedOnly: true,
  });

  return (
    <section className="stack-xl">
      <Breadcrumbs />
      <PageHeader
        icon={Bell}
        title="Alerts"
        description="Events emitted by the listener + observation pipelines."
        iconColorClass="text-module-shell"
      />

      <FilterBar filter={filter} onChange={setFilter} count={alerts.length} loading={loading} />

      {error ? (
        <div className="rounded-md border border-rose-500/40 bg-rose-500/10 p-3 text-sm text-rose-200">
          {error}
        </div>
      ) : null}

      <AlertTable
        alerts={alerts}
        onAcknowledge={(id): void => {
          void acknowledge(id);
        }}
        onResolve={(id): void => {
          void resolve(id);
        }}
      />
    </section>
  );
}

interface FilterBarProps {
  filter: { severity: string; unacknowledgedOnly: boolean; unresolvedOnly: boolean };
  onChange: (next: FilterBarProps['filter']) => void;
  count: number;
  loading: boolean;
}

function FilterBar({ filter, onChange, count, loading }: FilterBarProps): JSX.Element {
  return (
    <div className="flex flex-wrap items-center gap-3 rounded-md border border-zinc-800 bg-zinc-900/30 p-3 text-sm">
      <span className="text-xs uppercase tracking-wide text-zinc-500">
        {loading ? 'Loading…' : `${count} alert${count === 1 ? '' : 's'}`}
      </span>

      <div className="ml-auto flex items-center gap-3">
        <label className="flex items-center gap-2 text-zinc-300">
          <span className="text-xs uppercase tracking-wide text-zinc-500">Severity</span>
          <select
            value={filter.severity}
            onChange={(e): void => onChange({ ...filter, severity: e.target.value })}
            className="rounded-md border border-zinc-700 bg-zinc-950 px-2 py-1 text-xs text-zinc-100"
          >
            <option value="">all</option>
            <option value="critical">critical</option>
            <option value="error">error</option>
            <option value="warning">warning</option>
            <option value="info">info</option>
          </select>
        </label>
        <label className="flex items-center gap-1 text-xs text-zinc-300">
          <input
            type="checkbox"
            checked={filter.unacknowledgedOnly}
            onChange={(e): void => onChange({ ...filter, unacknowledgedOnly: e.target.checked })}
          />
          unack only
        </label>
        <label className="flex items-center gap-1 text-xs text-zinc-300">
          <input
            type="checkbox"
            checked={filter.unresolvedOnly}
            onChange={(e): void => onChange({ ...filter, unresolvedOnly: e.target.checked })}
          />
          unresolved only
        </label>
      </div>
    </div>
  );
}

interface AlertTableProps {
  alerts: Alert[];
  onAcknowledge: (id: number) => void;
  onResolve: (id: number) => void;
}

function AlertTable({ alerts, onAcknowledge, onResolve }: AlertTableProps): JSX.Element {
  if (alerts.length === 0) {
    return (
      <div className="rounded-md border border-zinc-700 bg-zinc-900/30 p-6 text-center text-sm text-zinc-400">
        No alerts match the current filter. Either nothing's misbehaving or your filter is too
        strict.
      </div>
    );
  }
  return (
    <div className="overflow-hidden rounded-lg border border-zinc-800 bg-zinc-900/30">
      <table className="w-full text-sm">
        <thead className="text-left text-xs uppercase tracking-wide text-zinc-500">
          <tr>
            <th className="px-4 py-2">When</th>
            <th className="px-4 py-2">Severity</th>
            <th className="px-4 py-2">Title</th>
            <th className="px-4 py-2">Source</th>
            <th className="px-4 py-2">State</th>
            <th className="px-4 py-2 text-right">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-zinc-800">
          {alerts.map((a) => (
            <tr key={a.id} data-testid={`alert-row-${a.id}`}>
              <td className="px-4 py-2 text-zinc-400">{fmtTime(a.createdAt)}</td>
              <td className="px-4 py-2">
                <SeverityBadge severity={a.severity} />
              </td>
              <td className="px-4 py-2">
                <div className="font-medium text-zinc-100">{a.title}</div>
                {a.message ? <div className="text-xs text-zinc-500">{a.message}</div> : null}
              </td>
              <td className="px-4 py-2 font-mono text-xs text-zinc-300">{a.source || '—'}</td>
              <td className="px-4 py-2">
                <StateBadge alert={a} />
              </td>
              <td className="px-4 py-2 text-right">
                {!a.acknowledged ? (
                  <button
                    type="button"
                    onClick={(): void => onAcknowledge(a.id)}
                    className="mr-2 inline-flex items-center gap-1 text-sm text-sky-400 hover:text-sky-300"
                  >
                    <Check className="h-3.5 w-3.5" />
                    Ack
                  </button>
                ) : null}
                {!a.resolved ? (
                  <button
                    type="button"
                    onClick={(): void => onResolve(a.id)}
                    className="inline-flex items-center gap-1 text-sm text-emerald-400 hover:text-emerald-300"
                  >
                    <CheckCircle2 className="h-3.5 w-3.5" />
                    Resolve
                  </button>
                ) : null}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function SeverityBadge({ severity }: { severity: string }): JSX.Element {
  // Severities aligned with database.AlertSeverity* constants.
  // Anything outside the known set lands in the neutral fallback so
  // server-side rule additions don't break the UI.
  const palette: Record<string, string> = {
    critical: 'bg-rose-500/20 text-rose-300',
    error: 'bg-orange-500/20 text-orange-300',
    warning: 'bg-amber-500/20 text-amber-300',
    info: 'bg-sky-500/20 text-sky-300',
  };
  const cls = palette[severity] ?? 'bg-zinc-700 text-zinc-300';
  return (
    <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${cls}`}>
      {severity || 'unknown'}
    </span>
  );
}

function StateBadge({ alert }: { alert: Alert }): JSX.Element {
  if (alert.resolved) {
    return (
      <span className="text-xs text-emerald-400" title={`Resolved at ${fmtTime(alert.resolvedAt)}`}>
        Resolved
      </span>
    );
  }
  if (alert.acknowledged) {
    return (
      <span
        className="text-xs text-sky-400"
        title={`Acknowledged${alert.acknowledgedBy ? ` by ${alert.acknowledgedBy}` : ''}`}
      >
        Acknowledged
      </span>
    );
  }
  return <span className="text-xs text-zinc-400">Open</span>;
}

function fmtTime(iso?: string): string {
  if (!iso) return '—';
  return new Date(iso).toLocaleString();
}
