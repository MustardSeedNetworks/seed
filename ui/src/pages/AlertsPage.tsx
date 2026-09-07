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
import { useRole } from '../contexts/RoleContext';
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
  const { t } = useTranslation(['pages', 'common']);
  const { canWrite } = useRole();
  const { alerts, loading, error, filter, setFilter, acknowledge, resolve } = useAlerts({
    unresolvedOnly: true,
  });
  const [selectedId, setSelectedId] = useState<number | null>(null);

  // POST /alerts/{id}/{acknowledge,resolve} is minRole: op, so for a viewer
  // both buttons could only 403 (#1254). Disabled with the reason rather than
  // hidden: the actions are part of the alert's lifecycle, and a viewer who
  // cannot see them cannot tell whether anyone has acted.
  const readOnlyReason = canWrite ? undefined : t('alerts.readOnly');

  const selected = alerts.find((a) => a.id === selectedId) ?? alerts[0] ?? null;

  return (
    <>
      {error ? (
        <div className="rounded-md border border-status-error/40 bg-status-error/10 pad-sm text-sm text-status-error">
          {error}
        </div>
      ) : null}

      <div className="flex-between">
        <p className="body-small">
          {loading ? t('common:status.loading') : t('alerts.alertCount', { count: alerts.length })}
        </p>
        <div className="flex items-center gap-default">
          <label className="flex items-center gap-tight min-h-6 text-xs text-text-secondary">
            <input
              type="checkbox"
              checked={filter.unacknowledgedOnly}
              onChange={(e): void => setFilter({ ...filter, unacknowledgedOnly: e.target.checked })}
            />
            {t('alerts.unacknowledgedOnly')}
          </label>
          <label className="flex items-center gap-tight min-h-6 text-xs text-text-secondary">
            <input
              type="checkbox"
              checked={filter.unresolvedOnly}
              onChange={(e): void => setFilter({ ...filter, unresolvedOnly: e.target.checked })}
            />
            {t('alerts.unresolvedOnly')}
          </label>
        </div>
      </div>

      <ListDetail>
        <RecordPane
          chips={
            <>
              <FilterChip
                label={t('alerts.filterAll')}
                active={filter.severity === ''}
                onClick={(): void => setFilter({ ...filter, severity: '' })}
              />
              {SEVERITIES.map((severity) => (
                <FilterChip
                  key={severity}
                  label={t(`alerts.severity.${severity}`)}
                  active={filter.severity === severity}
                  onClick={(): void => setFilter({ ...filter, severity })}
                />
              ))}
            </>
          }
          empty={t('alerts.noAlerts')}
        >
          {alerts.map((a) => (
            <RecordRow
              key={a.id}
              data-testid={`alert-row-${a.id}`}
              name={a.title}
              nameKind="prose"
              meta={`${a.source || t('alerts.unknownSource')} · ${fmtTime(a.createdAt)}`}
              state={alertState(a)}
              selected={selected?.id === a.id}
              onSelect={(): void => setSelectedId(a.id)}
            />
          ))}
        </RecordPane>

        {selected ? (
          <DetailPane
            eyebrow={t('alerts.selectedAlert')}
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
                    disabled={!canWrite}
                    title={readOnlyReason}
                    data-testid="alert-acknowledge"
                    className="inline-flex items-center gap-tight rounded-md border border-surface-border px-3 py-2 text-sm text-text-primary hover:bg-surface-hover disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    <Check className="h-3.5 w-3.5" />
                    {t('alerts.acknowledge')}
                  </button>
                ) : null}
                {!selected.resolved ? (
                  <button
                    type="button"
                    onClick={(): void => {
                      void resolve(selected.id);
                    }}
                    disabled={!canWrite}
                    title={readOnlyReason}
                    data-testid="alert-resolve"
                    className="inline-flex items-center gap-tight rounded-md border border-surface-border px-3 py-2 text-sm text-text-primary hover:bg-surface-hover disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    <CheckCircle2 className="h-3.5 w-3.5" />
                    {t('alerts.resolve')}
                  </button>
                ) : null}
              </>
            }
          >
            <DetailFacts
              items={[
                {
                  label: t('alerts.labelSeverity'),
                  value: selected.severity
                    ? t(`alerts.severity.${selected.severity}`, {
                        defaultValue: selected.severity,
                      })
                    : t('common:status.unknown'),
                },
                // Type and source are the rule author's own identifiers, so
                // they stay exactly as the rule wrote them.
                { label: t('alerts.labelType'), value: selected.type || '—' },
                { label: t('alerts.labelSource'), value: selected.source || '—' },
                { label: t('alerts.labelRaised'), value: fmtTime(selected.createdAt) },
                {
                  label: t('alerts.labelAcknowledged'),
                  value: selected.acknowledged
                    ? selected.acknowledgedBy
                      ? t('alerts.acknowledgedByAt', {
                          time: fmtTime(selected.acknowledgedAt),
                          user: selected.acknowledgedBy,
                        })
                      : fmtTime(selected.acknowledgedAt)
                    : t('alerts.valueNo'),
                  prose: true,
                },
                {
                  label: t('alerts.labelResolved'),
                  value: selected.resolved ? fmtTime(selected.resolvedAt) : t('alerts.valueNo'),
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
  const { t } = useTranslation(['pages', 'common']);
  if (alert.resolved) {
    return (
      <span className="rounded-lg border border-surface-border px-3 py-1.5 text-xs font-semibold text-text-secondary">
        {t('alerts.stateResolved')}
      </span>
    );
  }
  if (alert.acknowledged) {
    return (
      <span className="rounded-lg border border-status-info/40 bg-status-info/10 px-3 py-1.5 text-xs font-semibold text-status-info">
        {t('alerts.stateAcknowledged')}
      </span>
    );
  }
  return (
    <span className="rounded-lg border border-status-warning/40 bg-status-warning/10 px-3 py-1.5 text-xs font-semibold text-status-warning">
      {t('alerts.stateOpen')}
    </span>
  );
}

/** The rule's own payload — the sub-table the archetype calls for. */
function AlertMetadata({ metadata }: { metadata: Record<string, unknown> }): JSX.Element | null {
  const { t } = useTranslation(['pages', 'common']);
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
