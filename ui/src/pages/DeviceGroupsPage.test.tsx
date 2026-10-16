/**
 * Device groups (UI-SEED-25): membership round-trips through the API the way
 * the server stores it, PUT carries the whole member list, a group a rule or
 * report is scoped to explains why it cannot be deleted, and a member's state
 * comes from its own last poll.
 */
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { type CurrentUser, RoleProvider } from '../contexts/RoleContext';
import type { DeviceGroupRequest } from '../types/generated/device-group-request';
import type { DeviceGroupResponse } from '../types/generated/device-group-response';
import type { DeviceGroupSummaryResponse } from '../types/generated/device-group-summary-response';
import type { PollingTargetResponse } from '../types/generated/polling-target-response';

function target(id: string, name: string, overrides: Partial<PollingTargetResponse> = {}) {
  return {
    id,
    clientId: 'c',
    name,
    ipAddress: `10.44.10.${name.length}`,
    snmpVersion: 'v2c',
    credentialsId: 'cred',
    pollIntervalSeconds: 300,
    enabled: true,
    collectorChain: [],
    lastStatus: 'ok',
    lastError: '',
    lastPolledAt: '2026-10-15T10:00:00Z',
    createdAt: '',
    updatedAt: '',
    ...overrides,
  } satisfies PollingTargetResponse;
}

const targets: PollingTargetResponse[] = [
  target('t-core', 'core-01'),
  target('t-edge', 'edge-02', { lastStatus: 'timeout', lastError: 'SNMP timeout' }),
  target('t-new', 'acc-03', { lastPolledAt: undefined }),
];

vi.mock('../hooks/usePollingTargets', () => ({
  usePollingTargets: () => ({ targets, loading: false, error: null }),
}));

class FakeApiError extends Error {}

/** A stand-in server: the groups it holds and every write it was sent. */
let stored: DeviceGroupResponse[] = [];
let deleteRefusal: string | null = null;
const writes: { method: string; path: string; body?: DeviceGroupRequest }[] = [];
let role: CurrentUser['role'] = 'operator';

function save(id: string, body: DeviceGroupRequest): DeviceGroupResponse {
  const group: DeviceGroupResponse = {
    id,
    name: body.name,
    description: body.description ?? '',
    members: [...body.members].sort(),
    createdAt: '2026-10-15T10:00:00Z',
    updatedAt: `2026-10-15T10:00:0${writes.length}Z`,
  };
  stored = [...stored.filter((g) => g.id !== id), group];
  return group;
}

function summaryOf(id: string): DeviceGroupSummaryResponse {
  const group = stored.find((g) => g.id === id);
  if (!group) {
    throw new FakeApiError('API error: 404: Device group not found');
  }
  const members = targets
    .filter((t) => group.members.includes(t.id))
    .map((t) => ({ target: t, openAlerts: t.id === 't-core' ? 2 : 0 }));
  const total = members.reduce((n, m) => n + m.openAlerts, 0);
  return {
    id,
    name: group.name,
    members,
    openAlerts: { total, critical: total ? 1 : 0, error: 0, warning: total ? 1 : 0, info: 0 },
  };
}

vi.mock('../api/client', () => ({
  api: {
    get: async (path: string): Promise<unknown> => {
      if (path.includes('/users/me')) {
        return { username: 'u', role, isActive: true };
      }
      if (path.endsWith('/summary')) {
        return summaryOf(decodeURIComponent(path.split('/')[4] ?? ''));
      }
      return { count: stored.length, groups: stored };
    },
    post: async (path: string, body: DeviceGroupRequest): Promise<unknown> => {
      writes.push({ method: 'POST', path, body });
      return save(`g-${writes.length}`, body);
    },
    put: async (path: string, body: DeviceGroupRequest): Promise<unknown> => {
      writes.push({ method: 'PUT', path, body });
      return save(decodeURIComponent(path.split('/').at(-1) ?? ''), body);
    },
    delete: async (path: string): Promise<void> => {
      writes.push({ method: 'DELETE', path });
      if (deleteRefusal) {
        throw new FakeApiError(deleteRefusal);
      }
      stored = stored.filter((g) => !path.endsWith(`/${g.id}`));
    },
  },
}));

const { DeviceGroupsPage } = await import('./DeviceGroupsPage');

function renderPage(): void {
  render(
    <RoleProvider isAuthenticated={true}>
      <DeviceGroupsPage />
    </RoleProvider>,
  );
}

describe('DeviceGroupsPage', () => {
  beforeEach(() => {
    stored = [];
    writes.length = 0;
    deleteRefusal = null;
    role = 'operator';
    vi.spyOn(window, 'confirm').mockReturnValue(true);
  });

  it('creates a group with the ticked members and shows what the server stored', async () => {
    const user = userEvent.setup();
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId('group-add')).not.toHaveAttribute('aria-disabled'),
    );
    await user.click(screen.getByTestId('group-add'));
    await user.type(screen.getByTestId('group-name'), '  Core switches ');
    await user.click(screen.getByTestId('group-member-t-edge'));
    await user.click(screen.getByTestId('group-member-t-core'));
    await user.click(screen.getByTestId('group-save'));

    await waitFor(() => expect(screen.queryByTestId('group-save')).toBeNull());
    expect(writes).toEqual([
      {
        method: 'POST',
        path: '/api/v1/device-groups',
        body: { name: 'Core switches', description: '', members: ['t-edge', 't-core'] },
      },
    ]);
    const row = await screen.findByTestId('group-row-g-1');
    expect(row).toHaveTextContent('Core switches');
    expect(row).toHaveTextContent('2 members');
    expect(screen.getByTestId('group-member-total')).toHaveTextContent('2');
  });

  it('opens the edit form with the stored members ticked and PUTs the whole list', async () => {
    stored = [save('g-a', { name: 'Edge', members: ['t-core', 't-edge'] })];
    const user = userEvent.setup();
    renderPage();
    await user.click(await screen.findByTestId('group-edit'));

    expect(screen.getByTestId('group-member-t-core')).toBeChecked();
    expect(screen.getByTestId('group-member-t-edge')).toBeChecked();
    expect(screen.getByTestId('group-member-t-new')).not.toBeChecked();

    await user.click(screen.getByTestId('group-member-t-core'));
    await user.click(screen.getByTestId('group-member-t-new'));
    await user.click(screen.getByTestId('group-save'));

    await waitFor(() => expect(screen.queryByTestId('group-save')).toBeNull());
    expect(writes.at(-1)).toEqual({
      method: 'PUT',
      path: '/api/v1/device-groups/g-a',
      body: { name: 'Edge', description: '', members: ['t-edge', 't-new'] },
    });
    // The summary is read again for the new members.
    await waitFor(() => expect(screen.getByTestId('group-summary-member-t-new')).toBeVisible());
    expect(screen.queryByTestId('group-summary-member-t-core')).toBeNull();
  });

  it('refuses a blank name without calling the server', async () => {
    const user = userEvent.setup();
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId('group-add')).not.toHaveAttribute('aria-disabled'),
    );
    await user.click(screen.getByTestId('group-add'));
    await user.type(screen.getByTestId('group-name'), '   ');
    await user.click(screen.getByTestId('group-save'));

    expect(screen.getByTestId('group-form-error')).toHaveTextContent('Name is required.');
    expect(writes).toEqual([]);
  });

  it('marks each member by its own last poll and counts the open alerts', async () => {
    stored = [save('g-a', { name: 'All', members: ['t-core', 't-edge', 't-new'] })];
    renderPage();

    const core = await screen.findByTestId('group-summary-member-t-core');
    expect(core).toHaveAttribute('data-state', 'warn');
    expect(core).toHaveTextContent('2 alerts');
    const edge = screen.getByTestId('group-summary-member-t-edge');
    expect(edge).toHaveAttribute('data-state', 'crit');
    expect(edge).toHaveTextContent('SNMP timeout');
    const fresh = screen.getByTestId('group-summary-member-t-new');
    expect(fresh).toHaveAttribute('data-state', 'unknown');
    expect(fresh).toHaveTextContent('No poll completed yet');
    expect(screen.getByTestId('group-open-alerts')).toHaveTextContent(
      '2 open · 1 critical, 1 warning',
    );
  });

  it('says why a group in use cannot be deleted and keeps it listed', async () => {
    stored = [save('g-a', { name: 'Scoped', members: [] })];
    deleteRefusal =
      'API error: 409: An alert rule or scheduled report is scoped to this device group; change or delete it first';
    const user = userEvent.setup();
    renderPage();
    await user.click(await screen.findByTestId('group-delete'));

    expect(await screen.findByTestId('group-delete-error')).toHaveTextContent(
      'scoped to this device group',
    );
    expect(screen.getByTestId('group-row-g-a')).toBeVisible();
    expect(screen.getByTestId('group-members-empty')).toBeVisible();
  });

  it('deletes a group once confirmed', async () => {
    stored = [save('g-a', { name: 'Gone', members: [] })];
    const user = userEvent.setup();
    renderPage();
    await user.click(await screen.findByTestId('group-delete'));

    await waitFor(() => expect(screen.queryByTestId('group-row-g-a')).toBeNull());
    expect(writes.at(-1)).toEqual({ method: 'DELETE', path: '/api/v1/device-groups/g-a' });
  });

  it('leaves a viewer able to read groups but not change them', async () => {
    role = 'viewer';
    stored = [save('g-a', { name: 'Read only', members: ['t-core'] })];
    renderPage();

    const row = await screen.findByTestId('group-row-g-a');
    expect(within(row).getByText('Read only')).toBeVisible();
    // The tooltip keeps a disabled control focusable, so it is aria-disabled.
    await waitFor(() =>
      expect(screen.getByTestId('group-add')).toHaveAttribute('aria-disabled', 'true'),
    );
    expect(screen.getByTestId('group-edit')).toHaveAttribute('aria-disabled', 'true');
    expect(screen.getByTestId('group-delete')).toHaveAttribute('aria-disabled', 'true');
  });
});
