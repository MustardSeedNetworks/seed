/**
 * PollingTargetsPage.i18n.test.tsx — the polling targets page renders real
 * locale copy.
 *
 * #1942, S1-14. Six of this page's strings already came from the locale files
 * and the rest — the count, the facet chips, the search placeholder and its
 * accessible name, both empty states, every detail label and every status
 * badge — were hardcoded English.
 *
 * `SNMP`, the version strings and the collector names stay untranslated: they
 * are protocol names and the collector ids the chain is configured with.
 */
import { render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { type CurrentUser, RoleProvider } from '../contexts/RoleContext';
import i18n from '../i18n';
import { must } from '../test/must';
import type { PollingTarget } from '../types/polling';

const healthy: PollingTarget = {
  id: 'healthy',
  clientId: 'c',
  name: 'core-01',
  ipAddress: '10.44.10.2',
  snmpVersion: 'v3',
  credentialsId: '',
  pollIntervalSeconds: 300,
  enabled: true,
  collectorChain: ['sys_info', 'if_table'],
  lastStatus: 'ok',
  lastError: '',
  lastPolledAt: '2026-08-17T10:00:00Z',
  createdAt: '',
  updatedAt: '',
};

// Never polled and not enabled: the two states whose words are the point.
const paused: PollingTarget = {
  ...healthy,
  id: 'paused',
  name: 'acc-sw-12',
  enabled: false,
  collectorChain: [],
  lastPolledAt: undefined,
};

const state = { targets: [healthy, paused] as PollingTarget[] };

vi.mock('../hooks/usePollingTargets', () => ({
  usePollingTargets: () => ({
    targets: state.targets,
    loading: false,
    error: null,
    create: vi.fn(),
    update: vi.fn(),
    remove: vi.fn(),
  }),
}));

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
vi.mock('../api/client', () => ({
  api: { get: (path: string): Promise<unknown> => mockGet(path) },
}));

const { PollingTargetsPage } = await import('./PollingTargetsPage');

async function renderIn(language: string, role: CurrentUser['role'] = 'admin'): Promise<void> {
  await i18n.changeLanguage(language);
  mockGet.mockImplementation((path: string) =>
    path.includes('/users/me')
      ? Promise.resolve({ username: 'u', role, isActive: true })
      : Promise.resolve({}),
  );
  render(
    <RoleProvider isAuthenticated={true}>
      <PollingTargetsPage />
    </RoleProvider>,
  );
  const first = must(state.targets[0]);
  await waitFor(() => expect(screen.getAllByText(first.name).length).toBeGreaterThan(0));
}

beforeEach(() => {
  mockGet.mockReset();
  state.targets = [healthy, paused];
});

afterEach(async () => {
  vi.clearAllMocks();
  await i18n.changeLanguage('en');
});

describe('PollingTargetsPage — real locale copy', () => {
  it('renders the English count, filters and detail', async () => {
    await renderIn('en');

    expect(screen.getByText('2 targets')).toBeVisible();
    expect(screen.getByRole('searchbox', { name: 'Filter polling targets' })).toBeVisible();
    expect(screen.getByPlaceholderText('Filter 2 targets')).toBeVisible();
    expect(screen.getByRole('button', { name: /All/ })).toBeVisible();
    expect(screen.getByRole('button', { name: /Failing/ })).toBeVisible();
    expect(screen.getByRole('button', { name: /Paused/ })).toBeVisible();

    expect(screen.getByText('Selected target')).toBeVisible();
    expect(screen.getByRole('button', { name: 'Edit' })).toBeVisible();
    expect(screen.getByRole('button', { name: 'Delete' })).toBeVisible();
    for (const label of ['Poll interval', 'Enabled', 'Last poll', 'Last status']) {
      expect(screen.getByText(label)).toBeVisible();
    }
    expect(screen.getByText('Polling normally')).toBeVisible();
  });

  it('names both empty states in English', async () => {
    state.targets = [];
    await i18n.changeLanguage('en');
    mockGet.mockImplementation(() => Promise.resolve({ username: 'u', role: 'admin' }));
    render(
      <RoleProvider isAuthenticated={true}>
        <PollingTargetsPage />
      </RoleProvider>,
    );

    expect(screen.getByText('0 targets')).toBeVisible();
    expect(
      screen.getByText('No polling targets yet. Add one to start polling a device.'),
    ).toBeVisible();
    expect(screen.getByText('Add a target to see its polling detail here.')).toBeVisible();
  });

  it('renders Spanish under es, with no English left behind', async () => {
    await renderIn('es');

    for (const english of [
      '2 targets',
      'Selected target',
      'Poll interval',
      'Enabled',
      'Last poll',
      'Last status',
      'Polling normally',
      'Collector chain',
      'yes',
      'never',
    ]) {
      expect(screen.queryByText(english)).toBeNull();
    }
    expect(screen.queryByPlaceholderText('Filter 2 targets')).toBeNull();
    expect(screen.queryByRole('searchbox', { name: 'Filter polling targets' })).toBeNull();
    for (const english of ['Edit', 'Delete']) {
      expect(screen.queryByRole('button', { name: english })).toBeNull();
    }

    expect(screen.getByText('2 destinos')).toBeVisible();
    expect(screen.getByText('Destino seleccionado')).toBeVisible();
    expect(screen.getByText('Intervalo de sondeo')).toBeVisible();
    expect(screen.getByText('Sondeo normal')).toBeVisible();
    expect(screen.getByRole('button', { name: 'Editar' })).toBeVisible();
  });

  it('says in Spanish that a paused target has never been polled', async () => {
    state.targets = [paused];
    await renderIn('es');

    expect(screen.queryByText('Polling paused')).toBeNull();
    expect(screen.queryByText('No poll completed yet')).toBeNull();
    expect(screen.getByText('Sondeo pausado')).toBeVisible();
    expect(screen.getByText('Nunca')).toBeVisible();
  });

  it('leaves protocol names and collector ids alone under es', async () => {
    await renderIn('es');

    expect(screen.getAllByText(/SNMP v3/).length).toBeGreaterThan(0);
    expect(screen.getByText('sys_info')).toBeVisible();
    expect(screen.getByText('if_table')).toBeVisible();
  });
});
