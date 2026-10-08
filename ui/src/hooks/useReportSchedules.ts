/**
 * useReportSchedules owns the scheduled-report API calls (P-B5, #3209) for
 * ScheduledReportsCard.
 *
 * Schedules are Pro (`scheduled_reports`). Reads are open to every role;
 * create, edit and delete are operator-gated, so the card offers them only to
 * an operator. A refused schedule comes back with the reason naming the field,
 * which is returned to the editor as given.
 */
import { useEffect, useState } from 'react';
import { ApiError, api } from '../api';
import { LogComponents, logger } from '../lib/logger';
import type { ReportScheduleInfo } from '../types/generated/report-schedule-info';
import type { ReportScheduleRequest } from '../types/generated/report-schedule-request';
import type { ReportSchedulesResponse } from '../types/generated/report-schedules-response';
import type { ReportFormat } from './useReports';

const schedulesEndpoint = '/api/v1/reports/schedules';

export type ReportTemplate = 'executive' | 'vulnerability' | 'inventory' | 'performance';

/**
 * The formats each template produces that the generator implements. Mirrors
 * the built-in templates in internal/reporting/services_template.go, less xlsx,
 * which a template may list but the generator cannot write.
 */
export const REPORT_TEMPLATE_FORMATS: Record<
  ReportTemplate,
  readonly [ReportFormat, ...ReportFormat[]]
> = {
  executive: ['pdf', 'html'],
  vulnerability: ['pdf', 'html', 'csv'],
  inventory: ['pdf', 'html', 'csv'],
  performance: ['pdf', 'html', 'json'],
};

export const REPORT_TEMPLATES = Object.keys(REPORT_TEMPLATE_FORMATS) as ReportTemplate[];

export type ScheduleFrequency = 'daily' | 'weekly' | 'monthly';

/** The last day of the month a monthly schedule may name: every month has it. */
export const MAX_SCHEDULE_DAY_OF_MONTH = 28;

/**
 * A schedule as the editor holds it. Both day fields are kept so switching
 * frequency back and forth does not lose them; the request sends only the one
 * the frequency uses, which is the only shape the server accepts.
 */
export interface ScheduleDraft {
  name: string;
  template: ReportTemplate;
  format: ReportFormat;
  frequency: ScheduleFrequency;
  /** 0 is Sunday, as Go's time.Weekday counts. */
  dayOfWeek: number;
  dayOfMonth: number;
  /** HH:MM, 24-hour. */
  time: string;
  /** IANA zone name. */
  timezone: string;
  enabled: boolean;
}

function browserTimezone(): string {
  return Intl.DateTimeFormat().resolvedOptions().timeZone;
}

export function newScheduleDraft(): ScheduleDraft {
  return {
    name: '',
    template: 'executive',
    format: 'pdf',
    frequency: 'weekly',
    dayOfWeek: 1,
    dayOfMonth: 1,
    time: '06:00',
    timezone: browserTimezone(),
    enabled: true,
  };
}

function pad2(n: number): string {
  return String(n).padStart(2, '0');
}

export function scheduleTime(hour: number, minute: number): string {
  return `${pad2(hour)}:${pad2(minute)}`;
}

function isTemplate(value: string): value is ReportTemplate {
  return value in REPORT_TEMPLATE_FORMATS;
}

function isFrequency(value: string): value is ScheduleFrequency {
  return value === 'daily' || value === 'weekly' || value === 'monthly';
}

/** The editor's starting point for an existing schedule. */
export function draftFromSchedule(info: ReportScheduleInfo): ScheduleDraft {
  const defaults = newScheduleDraft();
  const template = isTemplate(info.template) ? info.template : defaults.template;
  const formats = REPORT_TEMPLATE_FORMATS[template];
  return {
    name: info.name,
    template,
    format: formats.find((f) => f === info.format) ?? formats[0],
    frequency: isFrequency(info.schedule.frequency) ? info.schedule.frequency : defaults.frequency,
    dayOfWeek: info.schedule.dayOfWeek ?? defaults.dayOfWeek,
    dayOfMonth: info.schedule.dayOfMonth ?? defaults.dayOfMonth,
    time: scheduleTime(info.schedule.hour, info.schedule.minute),
    timezone: info.schedule.timezone,
    enabled: info.enabled,
  };
}

export function scheduleRequest(draft: ScheduleDraft): ReportScheduleRequest {
  const [hour, minute] = draft.time.split(':').map(Number);
  return {
    name: draft.name.trim(),
    template: draft.template,
    format: draft.format,
    schedule: {
      frequency: draft.frequency,
      ...(draft.frequency === 'weekly' ? { dayOfWeek: draft.dayOfWeek } : {}),
      ...(draft.frequency === 'monthly' ? { dayOfMonth: draft.dayOfMonth } : {}),
      hour: hour ?? 0,
      minute: minute ?? 0,
      timezone: draft.timezone.trim(),
    },
    enabled: draft.enabled,
  };
}

/** The server's reason for a refusal: details names the field when it has one. */
function refusalReason(err: unknown): string {
  if (err instanceof ApiError) {
    return err.details || err.message;
  }
  return err instanceof Error ? err.message : String(err);
}

export interface UseReportSchedulesResult {
  schedules: ReportScheduleInfo[];
  loading: boolean;
  error: string | null;
  /** Creates (id null) or replaces a schedule; resolves to the refusal reason, or null. */
  save: (id: string | null, draft: ScheduleDraft) => Promise<string | null>;
  remove: (id: string) => Promise<void>;
}

export function useReportSchedules(): UseReportSchedulesResult {
  const [schedules, setSchedules] = useState<ReportScheduleInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = async (): Promise<void> => {
    await api
      .get<ReportSchedulesResponse>(schedulesEndpoint)
      .then((body) => {
        setSchedules(body.schedules);
        setError(null);
      })
      .catch((err: unknown) => {
        logger.error(LogComponents.EXPORT, 'Failed to load report schedules', err);
        setError(refusalReason(err));
      });
    setLoading(false);
  };

  const save = async (id: string | null, draft: ScheduleDraft): Promise<string | null> => {
    const body = scheduleRequest(draft);
    try {
      if (id === null) {
        await api.post(schedulesEndpoint, body);
      } else {
        await api.put(`${schedulesEndpoint}/${encodeURIComponent(id)}`, body);
      }
    } catch (err) {
      return refusalReason(err);
    }
    await refresh();
    return null;
  };

  const remove = async (id: string): Promise<void> => {
    try {
      await api.delete(`${schedulesEndpoint}/${encodeURIComponent(id)}`);
    } catch (err) {
      logger.error(LogComponents.EXPORT, 'Failed to delete report schedule', err);
      setError(refusalReason(err));
      return;
    }
    await refresh();
  };

  useEffect((): void => {
    refresh().catch(() => undefined);
  }, [refresh]);

  return { schedules, loading, error, save, remove };
}
