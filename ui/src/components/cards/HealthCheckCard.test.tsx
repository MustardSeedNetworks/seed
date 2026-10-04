/**
 * The Health Check card must not spend the rate-limited probe run on mount
 * (#2691). `/telemetry/probes/run` shares one 5-per-minute bucket per client
 * with every operator action, so a run on each Performance visit refused the
 * next capture or speed test. It runs on the Run button, and the last result
 * survives navigation instead of being re-run to be shown again.
 */
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { HealthCheckCard } from './HealthCheckCard';

vi.mock('../../contexts/useSettings', () => ({
  useSettings: () => ({ cardSettings: { healthChecks: { autoRunOnLink: true } } }),
}));

const health = {
  hasTests: true,
  pingResults: [{ name: '8.8.8.8', success: true, latency: 12, testStatus: 'success' }],
  tcpResults: [],
  udpResults: [],
  httpResults: [],
};

const fetchMock = vi.fn(
  (_url: string): Promise<{ ok: boolean; status: number; json: () => Promise<unknown> }> =>
    Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(health) }),
);

// Tooltip mirrors its trigger's text into a portal, so a bare text query
// matches the host twice. Only the value on the card counts.
const notTheTooltip = { ignore: '[role="tooltip"], script, style' } as const;

function probeRuns(): number {
  return fetchMock.mock.calls.filter(([url]) => String(url).includes('/telemetry/probes/run'))
    .length;
}

function renderCard(client: QueryClient): ReturnType<typeof render> {
  return render(
    <QueryClientProvider client={client}>
      <HealthCheckCard />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
  fetchMock.mockClear();
});

describe('HealthCheckCard', () => {
  it('does not run the probes when it mounts', async () => {
    renderCard(new QueryClient());

    expect(await screen.findByTestId('health-check-run')).toBeEnabled();
    expect(screen.getByText(/Not run yet/)).toBeVisible();
    expect(probeRuns()).toBe(0);
  });

  it('runs the probes when the operator asks', async () => {
    renderCard(new QueryClient());

    fireEvent.click(screen.getByTestId('health-check-run'));

    expect(await screen.findByText('8.8.8.8', notTheTooltip)).toBeVisible();
    expect(probeRuns()).toBe(1);
  });

  it('shows the last run again after navigation without running it again', async () => {
    const client = new QueryClient();
    const first = renderCard(client);
    fireEvent.click(screen.getByTestId('health-check-run'));
    await screen.findByText('8.8.8.8', notTheTooltip);
    first.unmount();

    renderCard(client);

    expect(screen.getByText('8.8.8.8', notTheTooltip)).toBeVisible();
    await waitFor(() => expect(probeRuns()).toBe(1));
  });

  it('says the run failed rather than showing nothing', async () => {
    fetchMock.mockImplementationOnce(() =>
      Promise.resolve({ ok: false, status: 429, json: () => Promise.resolve({}) }),
    );
    renderCard(new QueryClient());

    fireEvent.click(screen.getByTestId('health-check-run'));

    expect(await screen.findByText('Failed to run tests')).toBeVisible();
  });
});
