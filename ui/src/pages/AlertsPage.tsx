/**
 * AlertsPage — List + detail.
 *
 * Operator-facing list of alerts the Stage A4.5/A4.6 pipelines emit. Alerts
 * are the records: severity drives the row's state edge and the filter chips,
 * and the detail pane carries the message, provenance and the acknowledge /
 * resolve actions.
 *
 * The handler routes write X-Username from the JWT/PAT, so acknowledgedBy
 * reflects whoever clicked the button.
 */

import { Check, CheckCircle2 } from 'lucide-react';
import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useAlerts } from '../hooks/useAlerts';
import type { Alert } from '../types/alerts';
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

/** Severities aligned with database.AlertSeverity* constants. */
const SEVERITIES = ['critical', 'error', 'warning', 'info'] as const;

/**
 * A resolved alert is calm whatever it once was. Anything outside the known
 * severity set lands calm too, so a server-side rule addition cannot paint
 * the screen red on a value the UI has never heard of.
 */
function alertState(alert: Alert): RecordState {
  if (alert.resolved) {
    return 'ok';
  }
  if (alert.severity === 'critical' || alert.severity === 'error') {
    return 'crit';
  }
  return alert.severity === 'warning' ? 'warn' : 'ok';
}

function fmtTime(iso?: string): string {
  if (!iso) {
    return '—';
  }
  return new Date(iso).toLocaleString();
}

export function AlertsPage(): JSX.Element {
  const { t } = useTranslation('pages');
  const { alerts, loading, error, filter, setFilter, acknowledge, resolve } = useAlerts({
    unresolvedOnly: true,
  });
  const [selectedId, setSelectedId] = useState<number | null>(null);

  const selected = alerts.find((a) => a.id === selectedId) ?? alerts[0] ?? null;

  return (
    <>
      {error ? (
        <div className="rounded-md border border-status-error/40 bg-status-error/10 p-3 text-sm text-status-error">
          {error}
        </div>
      ) : null}

      <div className="flex-between">
        <p className="body-small">
          {loading ? 'Loading…' : `${alerts.length} alert${alerts.length === 1 ? '' : 's'}`}
        </p>
        <div className="flex items-center gap-3">
          <label className="flex items-center gap-1 text-xs text-text-secondary">
            <input
              type="checkbox"
              checked={filter.unacknowledgedOnly}
              onChange={(e): void => setFilter({ ...filter, unacknowledgedOnly: e.target.checked })}
            />
            unacknowledged only
          </label>
          <label className="flex items-center gap-1 text-xs text-text-secondary">
            <input
              type="checkbox"
              checked={filter.unresolvedOnly}
              onChange={(e): void => setFilter({ ...filter, unresolvedOnly: e.target.checked })}
            />
            unresolved only
          </label>
        </div>
      </div>

      <ListDetail>
        <RecordPane
          chips={
            <>
              <FilterChip
                label="All"
                active={filter.severity === ''}
                onClick={(): void => setFilter({ ...filter, severity: '' })}
              />
              {SEVERITIES.map((severity) => (
                <FilterChip
                  key={severity}
                  label={severity}
                  active={filter.severity === severity}
                  onClick={(): void => setFilter({ ...filter, severity })}
                />
              ))}
            </>
          }
          empty="No alerts match the current filter. Either nothing's misbehaving or your filter is too strict."
        >
          {alerts.map((a) => (
            <RecordRow
              key={a.id}
              data-testid={`alert-row-${a.id}`}
              name={a.title}
              nameKind="prose"
              meta={`${a.source || 'unknown source'} · ${fmtTime(a.createdAt)}`}
              state={alertState(a)}
              selected={selected?.id === a.id}
              onSelect={(): void => setSelectedId(a.id)}
            />
          ))}
        </RecordPane>

        {selected ? (
          <DetailPane
            eyebrow="Selected alert"
            title={selected.title}
            meta={selected.message || undefined}
            status={<AlertState alert={selected} />}
            actions={
              <>
                {!selected.acknowledged ? (
                  <button
                    type="button"
                    onClick={(): void => {
                      void acknowledge(selected.id);
                    }}
                    className="inline-flex items-center gap-1 rounded-md border border-surface-border px-3 py-2 text-sm text-text-primary hover:bg-surface-hover"
                  >
                    <Check className="h-3.5 w-3.5" />
                    Acknowledge
                  </button>
                ) : null}
                {!selected.resolved ? (
                  <button
                    type="button"
                    onClick={(): void => {
                      void resolve(selected.id);
                    }}
                    className="inline-flex items-center gap-1 rounded-md border border-surface-border px-3 py-2 text-sm text-text-primary hover:bg-surface-hover"
                  >
                    <CheckCircle2 className="h-3.5 w-3.5" />
                    Resolve
                  </button>
                ) : null}
              </>
            }
          >
            <DetailFacts
              items={[
                { label: 'Severity', value: selected.severity || 'unknown' },
                { label: 'Type', value: selected.type || '—' },
                { label: 'Source', value: selected.source || '—' },
                { label: 'Raised', value: fmtTime(selected.createdAt) },
                {
                  label: 'Acknowledged',
                  value: selected.acknowledged
                    ? `${fmtTime(selected.acknowledgedAt)}${selected.acknowledgedBy ? ` by ${selected.acknowledgedBy}` : ''}`
                    : 'no',
                  prose: true,
                },
                {
                  label: 'Resolved',
                  value: selected.resolved ? fmtTime(selected.resolvedAt) : 'no',
                },
              ]}
            />
            <AlertMetadata metadata={selected.metadata} />
          </DetailPane>
        ) : (
          <DetailEmpty>{t('alerts.selectPrompt')}</DetailEmpty>
        )}
      </ListDetail>
    </>
  );
}

/** Where the alert is in its lifecycle, said in words rather than by colour. */
function AlertState({ alert }: { alert: Alert }): JSX.Element {
  if (alert.resolved) {
    return (
      <span className="rounded-lg border border-surface-border px-3 py-1.5 text-xs font-semibold text-text-secondary">
        Resolved
      </span>
    );
  }
  if (alert.acknowledged) {
    return (
      <span className="rounded-lg border border-status-info/40 bg-status-info/10 px-3 py-1.5 text-xs font-semibold text-status-info">
        Acknowledged
      </span>
    );
  }
  return (
    <span className="rounded-lg border border-status-warning/40 bg-status-warning/10 px-3 py-1.5 text-xs font-semibold text-status-warning">
      Open
    </span>
  );
}

/** The rule's own payload — the sub-table the archetype calls for. */
function AlertMetadata({ metadata }: { metadata: Record<string, unknown> }): JSX.Element | null {
  const { t } = useTranslation('pages');
  const entries = Object.entries(metadata ?? {});
  if (entries.length === 0) {
    return null;
  }
  return (
    <div className="stack-xs">
      <p className="caption">{t('alerts.payload')}</p>
      <dl className="divide-y divide-surface-border overflow-hidden rounded-lg border border-surface-border">
        {entries.map(([key, value]) => (
          <div key={key} className="flex items-start gap-default px-cell py-2">
            <dt className="caption w-40 shrink-0">{key}</dt>
            <dd className="figure min-w-0 flex-1 break-words text-sm text-text-primary">
              {typeof value === 'string' ? value : JSON.stringify(value)}
            </dd>
          </div>
        ))}
      </dl>
    </div>
  );
}
