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

import type { TFunction } from 'i18next';
import { Check, CheckCircle2 } from 'lucide-react';
import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useRole } from '../contexts/RoleContext';
import { useAlerts } from '../hooks/useAlerts';
import { useTableLabels } from '../hooks/useTableLabels';
import { Alert as InlineAlert } from '../shared/Alert';
import { CategoryChip } from '../shared/CategoryChip';
import { DataTable, type DataTableColumn } from '../shared/DataTable';
import { EmptyState } from '../shared/EmptyState';
import { StatusPill, type StatusTone } from '../shared/StatusPill';
import type { Alert, AlertDelivery } from '../types/generated/alert-list-response';
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

/** Severities aligned with database.AlertSeverity* constants. */
const SEVERITIES = ['critical', 'error', 'warning', 'info'] as const;

type Severity = (typeof SEVERITIES)[number];

/** Info is a note, not a problem, so it paints calm. */
const SEVERITY_STATE = {
  critical: 'crit',
  error: 'crit',
  warning: 'warn',
  info: 'ok',
} as const satisfies Record<Severity, RecordState>;

/** The server does not constrain severity (a rule may set any string), so narrow here. */
function isSeverity(severity: string): severity is Severity {
  return (SEVERITIES as readonly string[]).includes(severity);
}

/**
 * A resolved alert is calm whatever it once was. A severity outside the known
 * set is unknown, not calm: it cannot paint the screen red on a value the UI
 * has never heard of, and it must not claim health either (#2526).
 */
function alertState(alert: Alert): RecordState {
  if (alert.resolved) {
    return 'ok';
  }
  return isSeverity(alert.severity) ? SEVERITY_STATE[alert.severity] : 'unknown';
}

function fmtTime(iso?: string): string {
  if (!iso) {
    return '—';
  }
  return new Date(iso).toLocaleString();
}

/**
 * Delivery states that mean the receiver never got the alert. These are the
 * only ones worth carrying into the list row: the operator is scanning for the
 * misconfiguration they do not know about, and 'delivered' on every row would
 * bury it.
 */
const UNDELIVERED: ReadonlySet<AlertDelivery['status']> = new Set(['failed', 'dropped']);

/** The detail row each channel's outcome is shown under. */
const DELIVERY_LABEL = {
  webhook: 'alerts.labelDeliveryWebhook',
  email: 'alerts.labelDeliveryEmail',
  syslog: 'alerts.labelDeliverySyslog',
  trap: 'alerts.labelDeliveryTrap',
} as const satisfies Record<AlertDelivery['channel'], string>;

/** The page's own translator, so the helpers below keep its key checking. */
type AlertsT = TFunction<['pages', 'common']>;

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
      {error ? <InlineAlert tone="error">{error}</InlineAlert> : null}

      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm text-text-secondary">
          {loading ? t('common:status.loading') : t('alerts.alertCount', { count: alerts.length })}
        </p>
        <div className="flex flex-wrap items-center gap-4">
          {/* Native, in a 24px label: the shared Checkbox's 16px input is under the
              24px minimum target the minimum-width ratchet holds every control to. */}
          <label className="flex min-h-6 items-center gap-2 text-xs text-text-secondary">
            <input
              type="checkbox"
              className="h-4 w-4 accent-brand-primary"
              checked={filter.unacknowledgedOnly}
              onChange={(e): void => setFilter({ ...filter, unacknowledgedOnly: e.target.checked })}
            />
            {t('alerts.unacknowledgedOnly')}
          </label>
          <label className="flex min-h-6 items-center gap-2 text-xs text-text-secondary">
            <input
              type="checkbox"
              className="h-4 w-4 accent-brand-primary"
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
          empty={<EmptyState title={t('alerts.noAlerts')} hint={t('alerts.noAlertsHint')} />}
        >
          {alerts.map((a) => (
            <RecordRow
              key={a.id}
              data-testid={`alert-row-${a.id}`}
              name={a.title}
              nameKind="prose"
              meta={rowMeta(t, a)}
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
                  <GatedButton
                    reason={readOnlyReason}
                    icon={<Check className="h-3.5 w-3.5" />}
                    onClick={(): void => {
                      void acknowledge(selected.id);
                    }}
                    data-testid="alert-acknowledge"
                  >
                    {t('alerts.acknowledge')}
                  </GatedButton>
                ) : null}
                {!selected.resolved ? (
                  <GatedButton
                    reason={readOnlyReason}
                    icon={<CheckCircle2 className="h-3.5 w-3.5" />}
                    onClick={(): void => {
                      void resolve(selected.id);
                    }}
                    data-testid="alert-resolve"
                  >
                    {t('alerts.resolve')}
                  </GatedButton>
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
                {
                  label: t('alerts.labelType'),
                  value: selected.type ? <CategoryChip label={selected.type} /> : '—',
                  prose: true,
                },
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
                // Only once a ladder has sent a stage: most alerts never
                // escalate, and "not escalated" on each would be noise.
                ...(selected.escalationStage
                  ? [
                      {
                        label: t('alerts.labelEscalated'),
                        testId: 'alert-escalation',
                        value: t('alerts.escalatedStage', {
                          stage: selected.escalationStage,
                          time: fmtTime(selected.escalatedAt),
                        }),
                        prose: true,
                      },
                    ]
                  : []),
                // One row per channel that was offered the alert, none when
                // nothing tried to send it. Most installs configure no
                // receiver, and a row reading "not delivered" there would
                // report a failure that never happened.
                ...(selected.deliveries ?? []).map((delivery) => ({
                  label: t(DELIVERY_LABEL[delivery.channel]),
                  value: deliveryText(t, delivery),
                  prose: true,
                })),
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

/**
 * The row's context line, with the delivery failure appended when there is
 * one. #368's point is that a receiver which stopped accepting POSTs must be
 * discoverable without opening each alert, and the meta line is the only place
 * on a row that carries words.
 */
function rowMeta(t: AlertsT, alert: Alert): string {
  const base = `${alert.source || t('alerts.unknownSource')} · ${fmtTime(alert.createdAt)}`;
  if (alert.deliveries?.some((delivery) => UNDELIVERED.has(delivery.status))) {
    return `${base} · ${t('alerts.deliveryNotDeliveredShort')}`;
  }
  return base;
}

/**
 * The delivery fact in words. A failure names the receiver's own reason —
 * a refused connection and a 401 need different fixes, and the operator should
 * not have to read the daemon log to tell them apart.
 */
function deliveryText(t: AlertsT, delivery: AlertDelivery): string {
  const time = fmtTime(delivery.attemptedAt);
  switch (delivery.status) {
    case 'delivered':
      return t('alerts.deliveryDelivered', { time });
    case 'failed':
      return t('alerts.deliveryFailed', { time, reason: delivery.error ?? '' });
    case 'dropped':
      return t('alerts.deliveryDropped', { time });
    default:
      return t('alerts.deliveryPending');
  }
}

/** Where the alert is in its lifecycle, said in words rather than by colour. */
function AlertState({ alert }: { alert: Alert }): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  if (alert.resolved) {
    return <StatusPill tone="unknown" label={t('alerts.stateResolved')} />;
  }
  const tone: StatusTone = alert.acknowledged ? 'info' : 'warning';
  return (
    <StatusPill
      tone={tone}
      label={alert.acknowledged ? t('alerts.stateAcknowledged') : t('alerts.stateOpen')}
    />
  );
}

interface PayloadEntry {
  key: string;
  value: string;
}

/** The rule's own payload — the sub-table the archetype calls for. */
function AlertMetadata({ metadata }: { metadata: Record<string, unknown> }): JSX.Element | null {
  const { t } = useTranslation(['pages', 'common']);
  const labels = useTableLabels<PayloadEntry>({ loading: t('common:status.loading') });
  const entries = Object.entries(metadata ?? {}).map(([key, value]) => ({
    key,
    value: typeof value === 'string' ? value : JSON.stringify(value),
  }));
  if (entries.length === 0) {
    return null;
  }
  const columns: DataTableColumn<PayloadEntry>[] = [
    {
      key: 'field',
      header: t('alerts.columnField'),
      cell: (entry) => <span className="figure text-text-secondary">{entry.key}</span>,
      sortValue: (entry) => entry.key,
    },
    {
      key: 'value',
      header: t('alerts.columnValue'),
      cell: (entry) => <span className="figure break-words">{entry.value}</span>,
    },
  ];
  return (
    <div className="space-y-1">
      <p className="kicker">{t('alerts.payload')}</p>
      <DataTable
        rows={entries}
        columns={columns}
        getRowKey={(entry) => entry.key}
        caption={t('alerts.payloadCaption')}
        labels={labels}
        empty={null}
      />
    </div>
  );
}
