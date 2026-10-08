import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { ScheduleDraft } from '../../hooks/useReportSchedules';
import type { ReportScheduleInfo } from '../../types/generated/report-schedule-info';
import { ScheduledReportsCard } from './ScheduledReportsCard';

const weekly: ReportScheduleInfo = {
  id: 's1',
  name: 'Monday summary',
  template: 'executive',
  format: 'pdf',
  schedule: { frequency: 'weekly', dayOfWeek: 1, hour: 6, minute: 0, timezone: 'UTC' },
  enabled: true,
  nextRun: '2026-10-12T06:00:00Z',
  createdAt: '2026-10-08T00:00:00Z',
  updatedAt: '2026-10-08T00:00:00Z',
};

type SaveFn = (id: string | null, draft: ScheduleDraft) => Promise<string | null>;

describe('ScheduledReportsCard', () => {
  it('describes each schedule in words', () => {
    render(<ScheduledReportsCard schedules={[weekly]} />);
    expect(screen.getByTestId('schedule-row').textContent).toContain('Monday at 06:00 (UTC)');
    expect(screen.getByTestId('schedule-row').textContent).toContain('Executive summary · PDF');
  });

  it('gives a viewer no way to change a schedule', () => {
    render(<ScheduledReportsCard schedules={[weekly]} />);
    expect(screen.queryByTestId('schedule-new')).toBeNull();
    expect(screen.queryByTestId('schedule-edit-s1')).toBeNull();
    expect(screen.queryByTestId('schedule-delete-s1')).toBeNull();
  });

  it('saves nothing until the schedule has a name', () => {
    render(<ScheduledReportsCard schedules={[]} onSave={vi.fn()} onDelete={vi.fn()} />);
    fireEvent.click(screen.getByTestId('schedule-new'));
    expect(screen.getByTestId('schedule-save')).toBeDisabled();
    fireEvent.change(screen.getByTestId('schedule-name'), { target: { value: '  ' } });
    expect(screen.getByTestId('schedule-save')).toBeDisabled();
    fireEvent.change(screen.getByTestId('schedule-name'), { target: { value: 'Month end' } });
    expect(screen.getByTestId('schedule-save')).toBeEnabled();
  });

  it('keeps the editor open with the server reason when a save is refused', async () => {
    const onSave = vi
      .fn<SaveFn>()
      .mockResolvedValueOnce('invalid scheduled report: unknown timezone')
      .mockResolvedValueOnce(null);
    render(<ScheduledReportsCard schedules={[]} onSave={onSave} onDelete={vi.fn()} />);

    fireEvent.click(screen.getByTestId('schedule-new'));
    fireEvent.change(screen.getByTestId('schedule-name'), { target: { value: 'Month end' } });
    fireEvent.change(screen.getByTestId('schedule-frequency'), { target: { value: 'monthly' } });
    fireEvent.change(screen.getByTestId('schedule-day-of-month'), { target: { value: '28' } });
    fireEvent.change(screen.getByTestId('schedule-timezone'), {
      target: { value: 'Mars/Olympus' },
    });
    fireEvent.click(screen.getByTestId('schedule-save'));

    expect(await screen.findByTestId('schedule-error')).toHaveTextContent('unknown timezone');
    expect(onSave).toHaveBeenLastCalledWith(
      null,
      expect.objectContaining({ frequency: 'monthly', dayOfMonth: 28, timezone: 'Mars/Olympus' }),
    );

    fireEvent.change(screen.getByTestId('schedule-timezone'), { target: { value: 'UTC' } });
    fireEvent.click(screen.getByTestId('schedule-save'));
    await waitFor(() => expect(screen.queryByTestId('schedule-editor')).toBeNull());
    expect(onSave).toHaveBeenLastCalledWith(null, expect.objectContaining({ timezone: 'UTC' }));
  });

  it('edits a schedule in place and offers only the formats its report produces', async () => {
    const onSave = vi.fn<SaveFn>().mockResolvedValue(null);
    render(<ScheduledReportsCard schedules={[weekly]} onSave={onSave} onDelete={vi.fn()} />);

    fireEvent.click(screen.getByTestId('schedule-edit-s1'));
    expect(screen.getByTestId('schedule-name')).toHaveValue('Monday summary');
    expect(screen.getByTestId('schedule-day-of-week')).toHaveValue('1');

    // Performance has no CSV; switching to it from a format it lacks resets the format.
    fireEvent.change(screen.getByTestId('schedule-template'), {
      target: { value: 'performance' },
    });
    const formats = Array.from(
      screen.getByTestId('schedule-format').querySelectorAll('option'),
      (o) => o.value,
    );
    expect(formats).toEqual(['pdf', 'html', 'json']);
    fireEvent.change(screen.getByTestId('schedule-format'), { target: { value: 'json' } });
    fireEvent.click(screen.getByTestId('schedule-save'));

    await waitFor(() =>
      expect(onSave).toHaveBeenCalledWith(
        's1',
        expect.objectContaining({ template: 'performance', format: 'json', dayOfWeek: 1 }),
      ),
    );
  });

  it('deletes through the handler', () => {
    const onDelete = vi.fn();
    render(<ScheduledReportsCard schedules={[weekly]} onSave={vi.fn()} onDelete={onDelete} />);
    fireEvent.click(screen.getByTestId('schedule-delete-s1'));
    expect(onDelete).toHaveBeenCalledWith('s1');
  });

  it('shows a disabled schedule as paused', () => {
    render(<ScheduledReportsCard schedules={[{ ...weekly, enabled: false }]} />);
    expect(screen.getByTestId('schedule-row').textContent).toContain('Paused');
  });
});
