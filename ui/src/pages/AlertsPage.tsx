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
        iconColorClass="text-module-security"
      />

      <FilterBar filter={filter} onChange={setFilter} count={alerts.length} loading={loading} />

      {error ? (
        <div className="rounded-md border border-status-error/40 bg-status-error/10 p-3 text-sm text-status-error">
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
    <div className="flex flex-wrap items-center gap-3 rounded-md border border-surface-border bg-surface-raised p-3 text-sm">
      <span className="text-xs uppercase tracking-wide text-text-muted">
        {loading ? 'Loading…' : `${count} alert${count === 1 ? '' : 's'}`}
      </span>

      <div className="ml-auto flex items-center gap-3">
        <label className="flex items-center gap-2 text-text-secondary">
          <span className="text-xs uppercase tracking-wide text-text-muted">Severity</span>
          <select
            value={filter.severity}
            onChange={(e): void => onChange({ ...filter, severity: e.target.value })}
            className="rounded-md border border-surface-border bg-surface-sunken px-2 py-1 text-xs text-text-primary"
          >
            <option value="">all</option>
            <option value="critical">critical</option>
            <option value="error">error</option>
            <option value="warning">warning</option>
            <option value="info">info</option>
          </select>
        </label>
        <label className="flex items-center gap-1 text-xs text-text-secondary">
          <input
            type="checkbox"
            checked={filter.unacknowledgedOnly}
            onChange={(e): void => onChange({ ...filter, unacknowledgedOnly: e.target.checked })}
          />
          unack only
        </label>
        <label className="flex items-center gap-1 text-xs text-text-secondary">
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
      <div className="rounded-md border border-surface-border bg-surface-raised p-6 text-center text-sm text-text-muted">
        No alerts match the current filter. Either nothing's misbehaving or your filter is too
        strict.
      </div>
    );
  }
  return (
    <div className="overflow-hidden rounded-lg border border-surface-border bg-surface-raised">
      <table className="w-full text-sm">
        <thead className="text-left text-xs uppercase tracking-wide text-text-muted">
          <tr>
            <th className="px-4 py-2">When</th>
            <th className="px-4 py-2">Severity</th>
            <th className="px-4 py-2">Title</th>
            <th className="px-4 py-2">Source</th>
            <th className="px-4 py-2">State</th>
            <th className="px-4 py-2 text-right">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-surface-border">
          {alerts.map((a) => (
            <tr key={a.id} data-testid={`alert-row-${a.id}`}>
              <td className="px-4 py-2 text-text-muted">{fmtTime(a.createdAt)}</td>
              <td className="px-4 py-2">
                <SeverityBadge severity={a.severity} />
              </td>
              <td className="px-4 py-2">
                <div className="font-medium text-text-primary">{a.title}</div>
                {a.message ? <div className="text-xs text-text-muted">{a.message}</div> : null}
              </td>
              <td className="px-4 py-2 font-mono text-xs text-text-secondary">{a.source || '—'}</td>
              <td className="px-4 py-2">
                <StateBadge alert={a} />
              </td>
              <td className="px-4 py-2 text-right">
                {!a.acknowledged ? (
                  <button
                    type="button"
                    onClick={(): void => onAcknowledge(a.id)}
                    className="mr-2 inline-flex items-center gap-1 text-sm text-status-info hover:text-status-info/80"
                  >
                    <Check className="h-3.5 w-3.5" />
                    Ack
                  </button>
                ) : null}
                {!a.resolved ? (
                  <button
                    type="button"
                    onClick={(): void => onResolve(a.id)}
                    className="inline-flex items-center gap-1 text-sm text-status-success hover:text-status-success/80"
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
    critical: 'bg-severity-high/20 text-severity-high',
    error: 'bg-status-error/20 text-status-error',
    warning: 'bg-status-warning/20 text-status-warning',
    info: 'bg-status-info/20 text-status-info',
  };
  const cls = palette[severity] ?? 'bg-surface-sunken text-text-secondary';
  return (
    <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${cls}`}>
      {severity || 'unknown'}
    </span>
  );
}

function StateBadge({ alert }: { alert: Alert }): JSX.Element {
  if (alert.resolved) {
    return (
      <span
        className="text-xs text-status-success"
        title={`Resolved at ${fmtTime(alert.resolvedAt)}`}
      >
        Resolved
      </span>
    );
  }
  if (alert.acknowledged) {
    return (
      <span
        className="text-xs text-status-info"
        title={`Acknowledged${alert.acknowledgedBy ? ` by ${alert.acknowledgedBy}` : ''}`}
      >
        Acknowledged
      </span>
    );
  }
  return <span className="text-xs text-text-muted">Open</span>;
}

function fmtTime(iso?: string): string {
  if (!iso) return '—';
  return new Date(iso).toLocaleString();
}
