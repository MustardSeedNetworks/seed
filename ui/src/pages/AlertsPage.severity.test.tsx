/**
 * AlertsPage.severity.test.tsx — the row edge each severity paints (#2526).
 *
 * The server does not constrain severity, so the page meets values it has
 * never heard of. Those must read as unknown: painting them calm claims a
 * health nobody measured, which is the failure RecordRow's own default exists
 * to prevent.
 */
import { render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { RoleProvider } from '../contexts/RoleContext';
import type { Alert } from '../types/alerts';

const baseAlert: Alert = {
  id: 1,
  title: 'Core switch unreachable',
  message: 'Three consecutive polls timed out.',
  severity: 'critical',
  type: 'reachability',
  source: 'snmp-poller',
  acknowledged: false,
  resolved: false,
  createdAt: '2026-09-06T10:00:00Z',
  metadata: {},
};

const state = { alerts: [baseAlert] as Alert[] };

vi.mock('../hooks/useAlerts', () => ({
  useAlerts: () => ({
    alerts: state.alerts,
    loading: false,
    error: null,
    filter: { severity: '', unacknowledgedOnly: false, unresolvedOnly: true },
    setFilter: vi.fn(),
    acknowledge: vi.fn(),
    resolve: vi.fn(),
  }),
}));

vi.mock('../api/client', () => ({
  api: {
    get: (path: string): Promise<unknown> =>
      Promise.resolve(
        path.includes('/users/me') ? { username: 'u', role: 'admin', isActive: true } : {},
      ),
  },
}));

const { AlertsPage } = await import('./AlertsPage');

async function rowBarClass(alert: Alert): Promise<string> {
  state.alerts = [alert];
  render(
    <RoleProvider isAuthenticated={true}>
      <AlertsPage />
    </RoleProvider>,
  );
  await waitFor(() => expect(screen.getAllByText(alert.title)).toHaveLength(2));
  const bar = screen.getByTestId('alert-row-1').querySelector('[aria-hidden="true"]');
  if (!bar) throw new Error('no state bar on the alert row');
  return bar.className;
}

afterEach(() => {
  vi.clearAllMocks();
});

describe('AlertsPage — severity state', () => {
  it.each([
    { severity: 'critical', bar: 'bg-status-error' },
    { severity: 'error', bar: 'bg-status-error' },
    { severity: 'warning', bar: 'bg-status-warning' },
    { severity: 'info', bar: 'bg-status-success' },
    { severity: 'bogus', bar: 'bg-text-disabled' },
    { severity: '', bar: 'bg-text-disabled' },
  ])('paints $severity as $bar', async ({ severity, bar }) => {
    expect(await rowBarClass({ ...baseAlert, severity })).toContain(bar);
  });

  it('paints a payload without a severity as unknown', async () => {
    // The wire type says string, but a malformed payload is exactly what this guards.
    const { severity: _omitted, ...rest } = baseAlert;
    expect(await rowBarClass(rest as Alert)).toContain('bg-text-disabled');
  });

  it('paints a resolved alert calm whatever its severity', async () => {
    expect(await rowBarClass({ ...baseAlert, severity: 'bogus', resolved: true })).toContain(
      'bg-status-success',
    );
  });
});
