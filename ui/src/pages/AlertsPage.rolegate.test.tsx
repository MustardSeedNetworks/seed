/**
 * AlertsPage role gating (#1254).
 *
 * POST /api/v1/alerts/{id}/{acknowledge,resolve} is registered `minRole: op`,
 * so for a viewer both buttons could only 403. They rendered enabled, which is
 * the failure the Wave 2 exit names: a privileged control visible to a role
 * that cannot use it.
 */

import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { type CurrentUser, RoleProvider } from '../contexts/RoleContext';
import type { Alert } from '../types/alerts';

const alert: Alert = {
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

const acknowledge = vi.fn();
const resolve = vi.fn();

vi.mock('../hooks/useAlerts', () => ({
  useAlerts: () => ({
    alerts: [alert],
    loading: false,
    error: null,
    filter: { severity: '', unacknowledgedOnly: false, unresolvedOnly: true },
    setFilter: vi.fn(),
    acknowledge,
    resolve,
  }),
}));

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
vi.mock('../api/client', () => ({
  api: { get: (path: string): Promise<unknown> => mockGet(path) },
}));

const { AlertsPage } = await import('./AlertsPage');

function renderAs(role: CurrentUser['role']): void {
  mockGet.mockImplementation((path: string) =>
    path.includes('/users/me')
      ? Promise.resolve({ username: 'u', role, isActive: true })
      : Promise.resolve({}),
  );
  render(
    <RoleProvider isAuthenticated={true}>
      <AlertsPage />
    </RoleProvider>,
  );
}

beforeEach(() => {
  mockGet.mockReset();
  acknowledge.mockReset();
  resolve.mockReset();
});

describe('AlertsPage — viewer gating', () => {
  it('disables acknowledge and resolve for a viewer and says why', async () => {
    renderAs('viewer');

    await waitFor(() => {
      expect(screen.getByTestId('alert-acknowledge')).toBeDisabled();
    });
    expect(screen.getByTestId('alert-resolve')).toBeDisabled();
    expect(screen.getByTestId('alert-acknowledge').title).toContain('operator role');
  });

  it('leaves both actions live for an operator', async () => {
    renderAs('operator');

    await waitFor(() => {
      expect(screen.getByTestId('alert-acknowledge')).toBeEnabled();
    });
    expect(screen.getByTestId('alert-resolve')).toBeEnabled();
    expect(screen.getByTestId('alert-acknowledge').title).toBe('');
  });
});
