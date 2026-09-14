/**
 * AlertsPage.delivery.test.tsx — the webhook delivery outcome is visible.
 *
 * #368's acceptance is that a receiver which stopped accepting POSTs becomes
 * discoverable from the inbox. Two rules carry that and are asserted here:
 * a failure is named on the row itself (the operator is scanning, not opening
 * each alert), and an install with no receiver shows nothing at all — most
 * deployments configure none, so an empty status must never read as a failure.
 */
import { render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { RoleProvider } from '../contexts/RoleContext';
import i18n from '../i18n';
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

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
vi.mock('../api/client', () => ({
  api: { get: (path: string): Promise<unknown> => mockGet(path) },
}));

const { AlertsPage } = await import('./AlertsPage');

async function renderWith(alert: Alert, language = 'en'): Promise<void> {
  state.alerts = [alert];
  await i18n.changeLanguage(language);
  mockGet.mockImplementation((path: string) =>
    path.includes('/users/me')
      ? Promise.resolve({ username: 'u', role: 'admin', isActive: true })
      : Promise.resolve({}),
  );
  render(
    <RoleProvider isAuthenticated={true}>
      <AlertsPage />
    </RoleProvider>,
  );
  await waitFor(() => expect(screen.getAllByText(alert.title)).toHaveLength(2));
}

beforeEach(() => {
  mockGet.mockReset();
  state.alerts = [baseAlert];
});

afterEach(async () => {
  vi.clearAllMocks();
  await i18n.changeLanguage('en');
});

describe('AlertsPage — webhook delivery status', () => {
  it('says nothing about delivery when no receiver is configured', async () => {
    await renderWith(baseAlert);

    expect(screen.queryByText('Webhook delivery')).toBeNull();
    expect(screen.queryByText(/not delivered/)).toBeNull();
  });

  it('names a failed delivery on the row and its reason in the detail', async () => {
    await renderWith({
      ...baseAlert,
      deliveryStatus: 'failed',
      deliveryAttemptedAt: '2026-09-06T10:00:05Z',
      deliveryError: 'receiver answered 401 for alert 1',
    });

    // On the row, so the misconfiguration is discoverable without opening
    // every alert in the inbox.
    expect(screen.getByTestId('alert-row-1').textContent).toContain('not delivered');
    expect(screen.getByText('Webhook delivery')).toBeVisible();
    expect(screen.getByText(/receiver answered 401 for alert 1/)).toBeVisible();
  });

  it('does not mark a delivered alert on the row, but states it in the detail', async () => {
    await renderWith({
      ...baseAlert,
      deliveryStatus: 'delivered',
      deliveryAttemptedAt: '2026-09-06T10:00:05Z',
    });

    expect(screen.getByTestId('alert-row-1').textContent).not.toContain('not delivered');
    expect(screen.getByText('Webhook delivery')).toBeVisible();
    expect(screen.getByText(/delivered/)).toBeVisible();
  });

  it('marks a dropped delivery, which the receiver never saw either', async () => {
    await renderWith({
      ...baseAlert,
      deliveryStatus: 'dropped',
      deliveryAttemptedAt: '2026-09-06T10:00:05Z',
    });

    expect(screen.getByTestId('alert-row-1').textContent).toContain('not delivered');
    expect(screen.getByText(/the delivery queue was full/)).toBeVisible();
  });

  it('renders the delivery copy in Spanish, with no English left behind', async () => {
    await renderWith(
      {
        ...baseAlert,
        deliveryStatus: 'failed',
        deliveryAttemptedAt: '2026-09-06T10:00:05Z',
        deliveryError: 'receiver answered 401 for alert 1',
      },
      'es',
    );

    expect(screen.queryByText('Webhook delivery')).toBeNull();
    expect(screen.getByTestId('alert-row-1').textContent).not.toContain('not delivered');
    expect(screen.getByText('Envío al webhook')).toBeVisible();
    expect(screen.getByTestId('alert-row-1').textContent).toContain('sin enviar');
    // The receiver's own words are not copy and stay as the daemon recorded them.
    expect(screen.getByText(/receiver answered 401 for alert 1/)).toBeVisible();
  });
});
