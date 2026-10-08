/**
 * ScheduledReportsCard lists the report schedules (P-B5, #3209) and, for an
 * operator, creates, edits and deletes them. Each run lands in the Reports
 * card beside it.
 *
 * Presentational over its props like ReportsCard: the data and the actions
 * come from useReportSchedules. The editor is local state; a refused save keeps
 * it open with the server's reason, shown as given.
 */
import { type JSX, type ReactNode, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  draftFromSchedule,
  MAX_SCHEDULE_DAY_OF_MONTH,
  newScheduleDraft,
  REPORT_TEMPLATE_FORMATS,
  REPORT_TEMPLATES,
  type ReportTemplate,
  type ScheduleDraft,
  type ScheduleFrequency,
  scheduleTime,
} from '../../hooks/useReportSchedules';
import { cn, input as inputTokens, spacing } from '../../styles/theme';
import type { ReportScheduleInfo } from '../../types/generated/report-schedule-info';
import { Button } from '../ui/Button';
import { Card, CardDivider, CardRow } from '../ui/Card';
import type { Status } from '../ui/StatusBadge';

export interface ScheduledReportsCardProps {
  schedules: ReportScheduleInfo[];
  loading?: boolean;
  error?: string | null;
  /** Undefined for a viewer: schedules are operator-gated. Resolves to the refusal reason. */
  onSave?: (id: string | null, draft: ScheduleDraft) => Promise<string | null>;
  onDelete?: (id: string) => void;
}

const FREQUENCIES: readonly ScheduleFrequency[] = ['daily', 'weekly', 'monthly'];
const WEEKDAYS = [0, 1, 2, 3, 4, 5, 6] as const;
const MONTH_DAYS = Array.from({ length: MAX_SCHEDULE_DAY_OF_MONTH }, (_, i) => i + 1);

/** 2023-01-01 was a Sunday, so day n of that week is weekday n. */
function weekdayName(day: number, locale: string): string {
  return new Intl.DateTimeFormat(locale, { weekday: 'long', timeZone: 'UTC' }).format(
    Date.UTC(2023, 0, 1 + day),
  );
}

function cardStatus(
  schedules: ReportScheduleInfo[],
  loading: boolean,
  error: string | null,
): Status {
  if (loading) {
    return 'loading';
  }
  if (error) {
    return 'error';
  }
  return schedules.some((s) => s.enabled) ? 'success' : 'unknown';
}

function Field({ id, label, children }: { id: string; label: string; children: ReactNode }) {
  return (
    <label className="stack-xs" htmlFor={id}>
      <span className="body-small font-medium text-text-primary">{label}</span>
      {children}
    </label>
  );
}

interface EditorProps {
  initial: ScheduleDraft;
  onSave: (draft: ScheduleDraft) => Promise<string | null>;
  onCancel: () => void;
}

function ScheduleEditor({ initial, onSave, onCancel }: EditorProps): JSX.Element {
  const { t, i18n } = useTranslation('cards');
  const [draft, setDraft] = useState(initial);
  const [reason, setReason] = useState('');
  const [saving, setSaving] = useState(false);
  const inputClass = cn(inputTokens.base, 'w-full');

  const edit = (patch: Partial<ScheduleDraft>): void => {
    setDraft((current) => ({ ...current, ...patch }));
  };

  const submit = async (): Promise<void> => {
    setSaving(true);
    const refused = await onSave(draft);
    setSaving(false);
    if (refused !== null) {
      setReason(refused);
    }
  };

  return (
    <div data-testid="schedule-editor" className={cn('stack-sm', spacing.margin.top.inline)}>
      <Field id="schedule-name" label={t('reportSchedules.name')}>
        <input
          id="schedule-name"
          data-testid="schedule-name"
          type="text"
          autoComplete="off"
          value={draft.name}
          onChange={(e): void => edit({ name: e.target.value })}
          className={inputClass}
        />
      </Field>

      <div className="flex flex-wrap gap-default">
        <Field id="schedule-template" label={t('reportSchedules.template')}>
          <select
            id="schedule-template"
            data-testid="schedule-template"
            value={draft.template}
            onChange={(e): void => {
              const template = e.target.value as ReportTemplate;
              const formats = REPORT_TEMPLATE_FORMATS[template];
              // Keep the format when the new template produces it.
              edit({
                template,
                format: formats.includes(draft.format) ? draft.format : formats[0],
              });
            }}
            className={inputTokens.base}
          >
            {REPORT_TEMPLATES.map((template) => (
              <option key={template} value={template}>
                {t(`reportSchedules.templates.${template}`)}
              </option>
            ))}
          </select>
        </Field>
        <Field id="schedule-format" label={t('reportSchedules.format')}>
          <select
            id="schedule-format"
            data-testid="schedule-format"
            value={draft.format}
            onChange={(e): void => {
              const format = REPORT_TEMPLATE_FORMATS[draft.template].find(
                (f) => f === e.target.value,
              );
              if (format) {
                edit({ format });
              }
            }}
            className={inputTokens.base}
          >
            {REPORT_TEMPLATE_FORMATS[draft.template].map((format) => (
              <option key={format} value={format}>
                {format.toUpperCase()}
              </option>
            ))}
          </select>
        </Field>
      </div>

      <div className="flex flex-wrap gap-default">
        <Field id="schedule-frequency" label={t('reportSchedules.frequency')}>
          <select
            id="schedule-frequency"
            data-testid="schedule-frequency"
            value={draft.frequency}
            onChange={(e): void => edit({ frequency: e.target.value as ScheduleFrequency })}
            className={inputTokens.base}
          >
            {FREQUENCIES.map((frequency) => (
              <option key={frequency} value={frequency}>
                {t(`reportSchedules.frequencies.${frequency}`)}
              </option>
            ))}
          </select>
        </Field>
        {draft.frequency === 'weekly' ? (
          <Field id="schedule-day-of-week" label={t('reportSchedules.dayOfWeek')}>
            <select
              id="schedule-day-of-week"
              data-testid="schedule-day-of-week"
              value={draft.dayOfWeek}
              onChange={(e): void => edit({ dayOfWeek: Number(e.target.value) })}
              className={inputTokens.base}
            >
              {WEEKDAYS.map((day) => (
                <option key={day} value={day}>
                  {weekdayName(day, i18n.language)}
                </option>
              ))}
            </select>
          </Field>
        ) : null}
        {draft.frequency === 'monthly' ? (
          <Field id="schedule-day-of-month" label={t('reportSchedules.dayOfMonth')}>
            <select
              id="schedule-day-of-month"
              data-testid="schedule-day-of-month"
              value={draft.dayOfMonth}
              onChange={(e): void => edit({ dayOfMonth: Number(e.target.value) })}
              className={inputTokens.base}
            >
              {MONTH_DAYS.map((day) => (
                <option key={day} value={day}>
                  {day}
                </option>
              ))}
            </select>
          </Field>
        ) : null}
        <Field id="schedule-time" label={t('reportSchedules.time')}>
          <input
            id="schedule-time"
            data-testid="schedule-time"
            type="time"
            required={true}
            value={draft.time}
            onChange={(e): void => edit({ time: e.target.value })}
            className={inputTokens.base}
          />
        </Field>
      </div>

      <Field id="schedule-timezone" label={t('reportSchedules.timezone')}>
        <input
          id="schedule-timezone"
          data-testid="schedule-timezone"
          type="text"
          autoComplete="off"
          value={draft.timezone}
          onChange={(e): void => edit({ timezone: e.target.value })}
          className={inputClass}
        />
      </Field>

      <label className="flex items-center gap-compact body-small" htmlFor="schedule-enabled">
        <input
          id="schedule-enabled"
          data-testid="schedule-enabled"
          type="checkbox"
          checked={draft.enabled}
          onChange={(e): void => edit({ enabled: e.target.checked })}
        />
        {t('reportSchedules.enabled')}
      </label>

      {reason === '' ? null : (
        <p data-testid="schedule-error" className="body-small text-status-error">
          {reason}
        </p>
      )}

      <div className="flex gap-compact">
        <Button
          size="sm"
          loading={saving}
          // Nothing to send without a name, or an hour once the time is cleared.
          disabled={draft.name.trim() === '' || draft.time === ''}
          onClick={(): void => {
            submit().catch(() => undefined);
          }}
          data-testid="schedule-save"
        >
          {t('reportSchedules.save')}
        </Button>
        <Button size="sm" variant="ghost" onClick={onCancel} data-testid="schedule-cancel">
          {t('reportSchedules.cancel')}
        </Button>
      </div>
    </div>
  );
}

export function ScheduledReportsCard({
  schedules,
  loading = false,
  error = null,
  onSave,
  onDelete,
}: ScheduledReportsCardProps): JSX.Element {
  const { t, i18n } = useTranslation('cards');
  // null id: a new schedule. Only one editor is open at a time.
  const [editing, setEditing] = useState<{ id: string | null; draft: ScheduleDraft } | null>(null);

  const cadence = (s: ReportScheduleInfo): string => {
    const time = scheduleTime(s.schedule.hour, s.schedule.minute);
    const zone = s.schedule.timezone;
    switch (s.schedule.frequency) {
      case 'weekly':
        return t('reportSchedules.cadence.weekly', {
          day: weekdayName(s.schedule.dayOfWeek ?? 0, i18n.language),
          time,
          zone,
        });
      case 'monthly':
        return t('reportSchedules.cadence.monthly', { day: s.schedule.dayOfMonth, time, zone });
      default:
        return t('reportSchedules.cadence.daily', { time, zone });
    }
  };

  const editor =
    editing && onSave ? (
      <ScheduleEditor
        // A fresh editor per target, so its draft starts from that schedule.
        key={editing.id ?? 'new'}
        initial={editing.draft}
        onSave={async (draft) => {
          const refused = await onSave(editing.id, draft);
          if (refused === null) {
            setEditing(null);
          }
          return refused;
        }}
        onCancel={(): void => setEditing(null)}
      />
    ) : null;

  return (
    <Card
      title={t('reportSchedules.title')}
      status={cardStatus(schedules, loading, error)}
      headerAction={
        onSave && editing === null ? (
          <Button
            size="sm"
            onClick={(): void => setEditing({ id: null, draft: newScheduleDraft() })}
            data-testid="schedule-new"
          >
            {t('reportSchedules.new')}
          </Button>
        ) : undefined
      }
    >
      <p className="caption text-text-muted">{t('reportSchedules.description')}</p>
      {error ? <p className="text-status-error text-sm">{error}</p> : null}

      {editing?.id === null ? editor : null}

      {!error && schedules.length === 0 && !loading && editing === null ? (
        <p data-testid="schedules-empty" className={cn('caption', spacing.margin.top.inline)}>
          {t('reportSchedules.empty')}
        </p>
      ) : null}

      {schedules.map((s, index) => (
        <div key={s.id} data-testid="schedule-row">
          {index > 0 ? <CardDivider /> : null}
          <CardRow
            label={s.name}
            value={`${t(`reportSchedules.templates.${s.template}`, { defaultValue: s.template })} · ${s.format.toUpperCase()}`}
          />
          <CardRow label={t('reportSchedules.when')} value={cadence(s)} />
          <CardRow
            label={t('reportSchedules.nextRun')}
            value={
              s.enabled && s.nextRun
                ? new Date(s.nextRun).toLocaleString()
                : t('reportSchedules.paused')
            }
          />
          {editing?.id === s.id ? (
            editor
          ) : onSave && onDelete && editing === null ? (
            <div className={cn('flex gap-compact', spacing.margin.top.inline)}>
              <button
                type="button"
                className="caption underline"
                onClick={(): void => setEditing({ id: s.id, draft: draftFromSchedule(s) })}
                data-testid={`schedule-edit-${s.id}`}
              >
                {t('reportSchedules.edit')}
              </button>
              <button
                type="button"
                className="caption underline text-status-error"
                onClick={(): void => onDelete(s.id)}
                data-testid={`schedule-delete-${s.id}`}
              >
                {t('reportSchedules.delete')}
              </button>
            </div>
          ) : null}
        </div>
      ))}
    </Card>
  );
}
