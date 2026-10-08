import { describe, expect, it } from 'vitest';
import type { ReportScheduleInfo } from '../types/generated/report-schedule-info';
import { draftFromSchedule, newScheduleDraft, scheduleRequest } from './useReportSchedules';

function info(schedule: ReportScheduleInfo['schedule']): ReportScheduleInfo {
  return {
    id: 's1',
    name: 'Weekly inventory',
    template: 'inventory',
    format: 'csv',
    schedule,
    enabled: false,
    createdAt: '2026-10-08T00:00:00Z',
    updatedAt: '2026-10-08T00:00:00Z',
  };
}

describe('scheduleRequest', () => {
  // The server refuses a day field the frequency does not use, so the draft's
  // spare one must never reach the wire.
  it.each([
    ['daily', {}],
    ['weekly', { dayOfWeek: 3 }],
    ['monthly', { dayOfMonth: 15 }],
  ] as const)('sends only the day field %s uses', (frequency, days) => {
    const req = scheduleRequest({
      ...newScheduleDraft(),
      name: '  Nightly  ',
      frequency,
      dayOfWeek: 3,
      dayOfMonth: 15,
      time: '23:05',
      timezone: ' Europe/Berlin ',
    });
    expect(req).toEqual({
      name: 'Nightly',
      template: 'executive',
      format: 'pdf',
      schedule: { frequency, ...days, hour: 23, minute: 5, timezone: 'Europe/Berlin' },
      enabled: true,
    });
  });
});

describe('draftFromSchedule', () => {
  it('round-trips a stored schedule through the editor unchanged', () => {
    const stored = info({
      frequency: 'monthly',
      dayOfMonth: 28,
      hour: 7,
      minute: 30,
      timezone: 'UTC',
    });
    const draft = draftFromSchedule(stored);
    expect(draft.time).toBe('07:30');
    expect(scheduleRequest(draft)).toEqual({
      name: stored.name,
      template: 'inventory',
      format: 'csv',
      schedule: stored.schedule,
      enabled: false,
    });
  });

  it('falls back to a format the template produces', () => {
    const draft = draftFromSchedule({
      ...info({ frequency: 'daily', hour: 0, minute: 0, timezone: 'UTC' }),
      format: 'xlsx',
    });
    expect(draft.format).toBe('pdf');
  });
});
