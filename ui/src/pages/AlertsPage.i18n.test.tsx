/**
 * AlertsPage.i18n.test.tsx — the alerts page renders real locale copy.
 *
 * #1942, S1-14. The page already had a role-gate suite, and every string it
 * matched on ('Acknowledge', 'Resolve') was hardcoded English, so the suite
 * was indifferent to the locale files.
 *
 * The severity words are translated deliberately: the chip and the fact row
 * present them to a human as prose, and the wire value they came from is
 * unchanged underneath. The alert's own `type` and `source` are not — those
 * are identifiers a rule author chose, and translating them would stop them
 * matching the rule.
 */
import { render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { type CurrentUser, RoleProvider } from '../contexts/RoleContext';
import i18n from '../i18n';
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
  metadata: { pollsMissed: 3 },
};

const state = { alerts: [alert] as Alert[] };

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

async function renderIn(language: string, role: CurrentUser['role'] = 'admin'): Promise<void> {
  await i18n.changeLanguage(language);
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
  // The title shows in both the row and the detail pane.
  await waitFor(() => expect(screen.getAllByText('Core switch unreachable')).toHaveLength(2));
}

beforeEach(() => {
  mockGet.mockReset();
  state.alerts = [alert];
});

afterEach(async () => {
  vi.clearAllMocks();
  await i18n.changeLanguage('en');
});

describe('AlertsPage — real locale copy', () => {
  it('renders the English list, filters and detail', async () => {
    await renderIn('en');

    expect(screen.getByText('1 alert')).toBeVisible();
    expect(screen.getByText('unacknowledged only')).toBeVisible();
    expect(screen.getByText('unresolved only')).toBeVisible();
    expect(screen.getByRole('button', { name: 'All' })).toBeVisible();
    expect(screen.getByRole('button', { name: 'Critical' })).toBeVisible();

    expect(screen.getByText('Selected alert')).toBeVisible();
    expect(screen.getByRole('button', { name: /Acknowledge/ })).toBeVisible();
    expect(screen.getByRole('button', { name: /Resolve/ })).toBeVisible();
    for (const label of ['Severity', 'Type', 'Source', 'Raised', 'Acknowledged', 'Resolved']) {
      expect(screen.getByText(label)).toBeVisible();
    }
    // Lifecycle badge: an alert nobody has touched is Open.
    expect(screen.getByText('Open')).toBeVisible();
    expect(screen.getByText('Alert payload')).toBeVisible();
  });

  it('names the empty state rather than showing a blank pane', async () => {
    state.alerts = [];
    await i18n.changeLanguage('en');
    mockGet.mockImplementation(() => Promise.resolve({ username: 'u', role: 'admin' }));
    render(
      <RoleProvider isAuthenticated={true}>
        <AlertsPage />
      </RoleProvider>,
    );

    expect(screen.getByText('0 alerts')).toBeVisible();
    expect(
      screen.getByText(
        "No alerts match the current filter. Either nothing's misbehaving or your filter is too strict.",
      ),
    ).toBeVisible();
    expect(screen.getByText('Select an alert to see its detail and act on it.')).toBeVisible();
  });

  it('renders Spanish under es, with no English left behind', async () => {
    await renderIn('es');

    // Every English string this page renders in this state.
    for (const english of [
      '1 alert',
      'unacknowledged only',
      'unresolved only',
      'Selected alert',
      'Severity',
      'Type',
      'Source',
      'Raised',
      'Acknowledged',
      'Resolved',
      'Open',
      'Alert payload',
    ]) {
      expect(screen.queryByText(english)).toBeNull();
    }
    for (const english of ['All', 'Critical', 'Acknowledge', 'Resolve']) {
      expect(screen.queryByRole('button', { name: english })).toBeNull();
    }

    // …and the Spanish is present, so an empty pane cannot pass this.
    expect(screen.getByText('1 alerta')).toBeVisible();
    expect(screen.getByText('Alerta seleccionada')).toBeVisible();
    expect(screen.getByText('Gravedad')).toBeVisible();
    expect(screen.getByText('Abierta')).toBeVisible();
    expect(screen.getByRole('button', { name: /Reconocer/ })).toBeVisible();
  });

  it('leaves the rule author’s own identifiers alone under es', async () => {
    await renderIn('es');

    // `type` and `source` come off the wire as the rule wrote them.
    expect(screen.getByText('reachability')).toBeVisible();
    expect(screen.getAllByText(/snmp-poller/).length).toBeGreaterThan(0);
    // Metadata keys are the payload's own, not copy.
    expect(screen.getByText('pollsMissed')).toBeVisible();
  });

  it('says the read-only reason in Spanish for a viewer', async () => {
    await renderIn('es', 'viewer');

    const ack = screen.getByTestId('alert-acknowledge');
    expect(ack).toBeDisabled();
    expect(ack.getAttribute('title')).toBe(
      'Solo lectura — se requiere el rol de operador para reconocer o resolver una alerta',
    );
  });
});
