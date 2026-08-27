import type { Meta, StoryObj } from '@storybook/react-vite';
import type { JSX } from 'react';
import { expect, fn, userEvent, within } from 'storybook/test';
import { ReportsCard } from './ReportsCard';

/**
 * ReportsCard lists generated reports and offers generate / download / delete.
 *
 * The stories carry play functions rather than only rendering: without them the
 * Storybook run proves the card does not throw, which would leave a wrong status
 * label or a missing download link green. Values are asserted, not counts.
 */
const meta = {
  title: 'Cards/ReportsCard',
  component: ReportsCard,
  parameters: { layout: 'centered' },
  tags: ['autodocs'],
  decorators: [
    (StoryComponent: React.ComponentType): JSX.Element => (
      <div style={{ width: '420px' }}>
        <StoryComponent />
      </div>
    ),
  ],
} satisfies Meta<typeof ReportsCard>;

export default meta;
type Story = StoryObj<typeof meta>;

const completed = {
  id: 'rep-1',
  name: 'Executive Report - 2026-08-27',
  type: 'executive',
  format: 'pdf',
  status: 'complete',
  fileSize: 20481,
  createdAt: '2026-08-27T10:00:00Z',
  completedAt: '2026-08-27T10:00:05Z',
};

/** A finished report: downloadable and deletable. */
export const Completed: Story = {
  args: {
    reports: [completed],
    onGenerate: fn(),
    onDelete: fn(),
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    await expect(canvas.getByText('Executive Report - 2026-08-27')).toBeInTheDocument();
    await expect(canvas.getByText('PDF')).toBeInTheDocument();
    await expect(canvas.getByTestId('report-download-rep-1')).toHaveAttribute(
      'href',
      '/api/v1/reports/rep-1/download',
    );
  },
};

/**
 * A report still being generated has no file, so it must not offer a download
 * link -- following one would 409.
 */
export const StillGenerating: Story = {
  args: {
    reports: [{ ...completed, id: 'rep-2', status: 'generating', completedAt: undefined }],
    onGenerate: fn(),
    onDelete: fn(),
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    await expect(canvas.getByText('generating')).toBeInTheDocument();
    await expect(canvas.queryByTestId('report-download-rep-2')).not.toBeInTheDocument();
    // Nor deletable while in flight: that is the resurrection race the backend
    // now refuses, and offering the button would invite it.
    await expect(canvas.queryByTestId('report-delete-rep-2')).not.toBeInTheDocument();
  },
};

/** A failed report surfaces its error and can be cleared away. */
export const Failed: Story = {
  args: {
    reports: [
      {
        ...completed,
        id: 'rep-3',
        status: 'failed',
        error: 'aggregation failed: no metrics in range',
      },
    ],
    onGenerate: fn(),
    onDelete: fn(),
  },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement);

    await expect(canvas.getByText('aggregation failed: no metrics in range')).toBeInTheDocument();
    await userEvent.click(canvas.getByTestId('report-delete-rep-3'));
    await expect(args.onDelete).toHaveBeenCalledWith('rep-3');
  },
};

/** Generate is wired and reports its in-flight state. */
export const Empty: Story = {
  args: {
    reports: [],
    onGenerate: fn(),
    onDelete: fn(),
  },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement);

    await userEvent.click(canvas.getByTestId('reports-generate'));
    await expect(args.onGenerate).toHaveBeenCalledOnce();
  },
};

/**
 * A viewer gets no generate or delete controls. The API refuses them anyway
 * (operator+), so showing them would only produce a 403.
 */
export const ReadOnlyViewer: Story = {
  args: {
    reports: [completed],
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    await expect(canvas.queryByTestId('reports-generate')).not.toBeInTheDocument();
    await expect(canvas.queryByTestId('report-delete-rep-1')).not.toBeInTheDocument();
    // Reading stays available.
    await expect(canvas.getByTestId('report-download-rep-1')).toBeInTheDocument();
  },
};

/** A failed request says so rather than showing an empty list. */
export const ErrorState: Story = {
  args: {
    reports: [],
    error: 'reports request failed (402)',
    onGenerate: fn(),
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    await expect(canvas.getByText('reports request failed (402)')).toBeInTheDocument();
  },
};

export const Loading: Story = {
  args: {
    reports: [],
    loading: true,
  },
};
