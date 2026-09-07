/**
 * The List + detail archetype's honest-state rule, on the page the design
 * actually drew: a target nobody is polling must not read as healthy. Green
 * for "we have not looked" is the one failure this shell exists to prevent.
 */
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { type CurrentUser, RoleProvider } from '../contexts/RoleContext';
import type { PollingTarget } from '../types/polling';

const targets: PollingTarget[] = [
  {
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
  },
  {
    id: 'paused',
    clientId: 'c',
    name: 'acc-sw-12',
    ipAddress: '10.44.10.9',
    snmpVersion: 'v2c',
    credentialsId: '',
    pollIntervalSeconds: 300,
    enabled: false,
    collectorChain: [],
    lastStatus: 'ok',
    lastError: '',
    createdAt: '',
    updatedAt: '',
  },
];

vi.mock('../hooks/usePollingTargets', () => ({
  usePollingTargets: () => ({
    targets,
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

/** The page reads its role from context; useRole throws without a provider. */
function renderAs(role: CurrentUser['role']): void {
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
}

describe('PollingTargetsPage — list + detail', () => {
  beforeEach(() => {
    mockGet.mockReset();
    renderAs('operator');
  });

  it('prints an em dash rather than a figure for a target never polled', () => {
    const row = within(screen.getByTestId('target-row-paused'));

    expect(row.getByText('—')).toBeTruthy();
  });

  it('says why a paused target has no reading instead of showing it as fine', async () => {
    await userEvent.click(screen.getByTestId('target-row-paused'));

    expect(screen.getByText('Polling paused')).toBeTruthy();
    expect(screen.queryByText('Polling normally')).toBeNull();
  });

  it('shows the selected target detail, including its collector chain', async () => {
    await userEvent.click(screen.getByTestId('target-row-healthy'));

    expect(screen.getByText('Selected target')).toBeTruthy();
    expect(screen.getByText('sys_info')).toBeTruthy();
    expect(screen.getByText('if_table')).toBeTruthy();
  });

  it('filters the list down to failing targets without losing the counts', async () => {
    await userEvent.click(screen.getByRole('button', { name: /^Paused/ }));

    expect(screen.queryByTestId('target-row-healthy')).toBeNull();
    expect(screen.getByTestId('target-row-paused')).toBeTruthy();
  });
});

/**
 * The whole /polling-targets collection is minRole: op (#1254), so add, edit
 * and delete could only 403 for a viewer. The list and detail stay readable —
 * writeGated passes GET for every role, which
 * TestViewerCanReadEveryRoleGatedRoute asserts on the server side.
 */
describe('PollingTargetsPage — viewer gating', () => {
  beforeEach(() => {
    mockGet.mockReset();
  });

  it('disables add, edit and delete for a viewer and says why', async () => {
    renderAs('viewer');
    await userEvent.click(screen.getByTestId('target-row-healthy'));

    await waitFor(() => {
      expect(screen.getByTestId('target-add')).toBeDisabled();
    });
    expect(screen.getByTestId('target-edit')).toBeDisabled();
    expect(screen.getByTestId('target-delete')).toBeDisabled();
    expect(screen.getByTestId('target-add').title).toContain('operator role');
  });

  it('still lets a viewer read the target detail', async () => {
    renderAs('viewer');
    await userEvent.click(screen.getByTestId('target-row-healthy'));

    expect(screen.getByText('Selected target')).toBeTruthy();
    expect(screen.getByText('sys_info')).toBeTruthy();
  });

  it('leaves all three live for an operator', async () => {
    renderAs('operator');
    await userEvent.click(screen.getByTestId('target-row-healthy'));

    await waitFor(() => {
      expect(screen.getByTestId('target-add')).toBeEnabled();
    });
    expect(screen.getByTestId('target-edit')).toBeEnabled();
    expect(screen.getByTestId('target-delete')).toBeEnabled();
  });
});
