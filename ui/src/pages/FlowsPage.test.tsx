/**
 * FlowsPage (UI-SEED-23): the three top-N lists read one window and one
 * ranking, the selectors re-read all three, a clamped window is spelled out,
 * and an empty window is one notice rather than three empty tables.
 */
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import i18n from '../i18n';
import type { HistoryWindowResponse } from '../types/generated/flow-talkers-response';

const served: HistoryWindowResponse = {
  from: '2026-10-06T00:00:00Z',
  to: '2026-10-07T00:00:00Z',
  days: 1,
  resolution: 'hourly',
  source: 'raw',
  clamped: false,
  requestedDays: 1,
};

const state = {
  window: served,
  talkers: [
    { addr: '10.0.0.5', bytes: 3 * 1024 * 1024, packets: 2500 },
    { addr: '10.0.0.9', bytes: 2048, packets: 12 },
  ],
  conversations: [
    { addrA: '10.0.0.5', addrB: '198.51.100.7', protocol: 6, bytes: 4096, packets: 40 },
    { addrA: '10.0.0.5', addrB: '10.0.0.9', protocol: 103, bytes: 512, packets: 4 },
  ],
  applications: [
    { name: 'https', bytes: 4096, packets: 40 },
    { name: 'unknown', bytes: 512, packets: 4 },
  ],
  fail: false,
};

const get = vi.fn((path: string): Promise<unknown> => {
  if (state.fail) {
    return Promise.reject(new Error('flows read failed'));
  }
  const by = new URL(path, 'https://seed.test').searchParams.get('by') ?? 'bytes';
  if (path.startsWith('/api/v1/flows/top-talkers')) {
    return Promise.resolve({ window: state.window, by, talkers: state.talkers });
  }
  if (path.startsWith('/api/v1/flows/top-conversations')) {
    return Promise.resolve({ window: state.window, by, conversations: state.conversations });
  }
  if (path.startsWith('/api/v1/flows/top-applications')) {
    return Promise.resolve({ window: state.window, by, applications: state.applications });
  }
  return Promise.reject(new Error(`unexpected ${path}`));
});

vi.mock('../api/client', () => ({ api: { get: (path: string) => get(path) } }));

const { FlowsPage } = await import('./FlowsPage');

const initial = structuredClone(state);

function rowText(testId: string): string[] {
  return within(screen.getByTestId(testId))
    .getAllByTestId(`${testId}-row`)
    .map((row) =>
      Array.from(row.querySelectorAll('td'))
        .map((td) => td.textContent)
        .join(' | '),
    );
}

beforeEach(async () => {
  await i18n.changeLanguage('en');
});

afterEach(() => {
  Object.assign(state, structuredClone(initial));
  get.mockClear();
});

describe('FlowsPage', () => {
  it('lists the top talkers, conversations and applications for the last day by bytes', async () => {
    render(<FlowsPage />);
    await screen.findByTestId('flows-talkers');

    expect(get.mock.calls.map(([path]) => path).sort((a, b) => a.localeCompare(b))).toEqual([
      '/api/v1/flows/top-applications?range=1d&by=bytes',
      '/api/v1/flows/top-conversations?range=1d&by=bytes',
      '/api/v1/flows/top-talkers?range=1d&by=bytes',
    ]);
    expect(rowText('flows-talkers')).toEqual(['10.0.0.5 | 3 MB | 2,500', '10.0.0.9 | 2 KB | 12']);
    expect(rowText('flows-conversations')).toEqual([
      '10.0.0.5 ↔ 198.51.100.7 | TCP | 4 KB | 40',
      '10.0.0.5 ↔ 10.0.0.9 | IP 103 | 512 B | 4',
    ]);
    expect(rowText('flows-applications')).toEqual([
      'https | 4 KB | 40',
      'Unidentified | 512 B | 4',
    ]);
    expect(screen.queryByTestId('flows-clamped')).not.toBeInTheDocument();
  });

  it('re-reads all three lists when the window or the ranking changes', async () => {
    const user = userEvent.setup();
    render(<FlowsPage />);
    await screen.findByTestId('flows-talkers');
    get.mockClear();

    await user.selectOptions(screen.getByTestId('flows-range'), '30d');
    await waitFor(() => expect(get).toHaveBeenCalledTimes(3));
    expect(get.mock.calls.every(([path]) => path.endsWith('?range=30d&by=bytes'))).toBe(true);

    get.mockClear();
    await user.selectOptions(screen.getByTestId('flows-rank'), 'packets');
    await waitFor(() => expect(get).toHaveBeenCalledTimes(3));
    expect(get.mock.calls.every(([path]) => path.endsWith('?range=30d&by=packets'))).toBe(true);
  });

  it('says how much the licence kept when the window was clamped', async () => {
    state.window = { ...served, days: 7, requestedDays: 90, clamped: true };
    render(<FlowsPage />);
    expect(await screen.findByTestId('flows-clamped')).toHaveTextContent(
      'Showing the 7 days your license keeps.',
    );
  });

  it('shows one notice for a window with no flows', async () => {
    state.talkers = [];
    state.conversations = [];
    state.applications = [];
    render(<FlowsPage />);
    expect(await screen.findByTestId('flows-empty')).toHaveTextContent('No flows in this window.');
    expect(screen.queryByTestId('flows-talkers')).not.toBeInTheDocument();
  });

  it('marks a list that is empty while the others are not', async () => {
    state.applications = [];
    render(<FlowsPage />);
    expect(await screen.findByTestId('flows-applications-none')).toBeInTheDocument();
    expect(rowText('flows-talkers')).toHaveLength(2);
  });

  it('reports a failed read', async () => {
    state.fail = true;
    render(<FlowsPage />);
    expect(await screen.findByTestId('flows-error')).toHaveTextContent('flows read failed');
  });
});
