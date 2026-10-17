/**
 * IP addresses (UI-SEED-46): each network and DHCP scope shows the server's
 * counts, the detail names the addresses in use, only a scope carries a state,
 * a scope round-trips through add and delete, refusals are worded, and a
 * viewer reads without the controls.
 */
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { type CurrentUser, RoleProvider } from '../contexts/RoleContext';
import type { IPAMScopeRequest } from '../types/generated/ipam-scope-request';
import type { IPAMScope, IPAMSubnet } from '../types/generated/ipam-subnets-response';

const subnets: IPAMSubnet[] = [
  {
    cidr: '192.168.1.0/24',
    source: 'local',
    size: 254,
    used: 3,
    free: 251,
    utilizationPct: 1.1811,
    inUse: ['192.168.1.1', '192.168.1.20', '192.168.1.150'],
  },
  {
    cidr: '10.44.10.0/30',
    name: 'Servers',
    source: 'configured',
    size: 2,
    used: 0,
    free: 2,
    utilizationPct: 0,
    inUse: [],
  },
];

const { ApiError } = await import('../api/client');

/** A stand-in server: the scopes it holds, every write, and the next refusal. */
let scopes: IPAMScope[] = [];
let refusal: InstanceType<typeof ApiError> | null = null;
const writes: { method: string; path: string; body?: IPAMScopeRequest }[] = [];
let role: CurrentUser['role'] = 'operator';

function scope(start: string, end: string, used: string[], name?: string): IPAMScope {
  const size = Number(end.split('.')[3]) - Number(start.split('.')[3]) + 1;
  const pct = (used.length / size) * 100;
  return {
    name,
    start,
    end,
    size,
    used: used.length,
    free: size - used.length,
    utilizationPct: pct,
    exhaustion: used.length === size ? 'full' : pct >= 90 ? 'near' : 'none',
    inUse: used,
  };
}

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    api: {
      get: async (path: string): Promise<unknown> => {
        if (path.includes('/users/me')) {
          return { username: 'u', role, isActive: true };
        }
        return { subnets, scopes };
      },
      post: async (path: string, body: IPAMScopeRequest): Promise<unknown> => {
        writes.push({ method: 'POST', path, body });
        if (refusal) {
          throw refusal;
        }
        scopes = [...scopes, scope(body.start, body.end, ['192.168.1.150'], body.name)];
        return { status: 'success' };
      },
      delete: async (path: string): Promise<unknown> => {
        writes.push({ method: 'DELETE', path });
        const start = new URL(path, 'https://seed').searchParams.get('start');
        scopes = scopes.filter((s) => s.start !== start);
        return { status: 'success' };
      },
    },
  };
});

const { IpamPage } = await import('./IpamPage');

function renderPage(): void {
  render(
    <RoleProvider isAuthenticated={true}>
      <IpamPage />
    </RoleProvider>,
  );
}

async function openAddForm(user: ReturnType<typeof userEvent.setup>): Promise<void> {
  await waitFor(() => expect(screen.getByTestId('scope-add')).not.toHaveAttribute('aria-disabled'));
  await user.click(screen.getByTestId('scope-add'));
}

describe('IpamPage', () => {
  beforeEach(() => {
    scopes = [];
    refusal = null;
    writes.length = 0;
    role = 'operator';
    vi.spyOn(window, 'confirm').mockReturnValue(true);
  });

  it('lists each network with its source and utilization and opens the first', async () => {
    renderPage();

    const local = await screen.findByTestId('ipam-subnet-192.168.1.0/24');
    expect(local).toHaveTextContent("This interface's subnet");
    expect(local).toHaveTextContent('1.2%');
    // A subnet has no exhaustion threshold, so it claims no health.
    expect(local).toHaveAttribute('data-state', 'unknown');
    expect(screen.getByTestId('ipam-subnet-10.44.10.0/30')).toHaveTextContent(
      'Servers · Target network',
    );
    expect(screen.getByTestId('ipam-counts')).toHaveTextContent('2 networks · 0 DHCP scopes');

    expect(screen.getByTestId('ipam-size')).toHaveTextContent('254');
    expect(screen.getByTestId('ipam-used')).toHaveTextContent('3');
    expect(screen.getByTestId('ipam-free')).toHaveTextContent('251');
    const inUse = within(screen.getByTestId('ipam-in-use')).getAllByRole('listitem');
    expect(inUse.map((item) => item.textContent)).toEqual([
      '192.168.1.1',
      '192.168.1.20',
      '192.168.1.150',
    ]);
    // Network rows offer nothing to delete; target networks live in Settings.
    expect(screen.queryByTestId('scope-delete')).toBeNull();
  });

  it('says when nothing in a network was seen in use', async () => {
    const user = userEvent.setup();
    renderPage();
    await user.click(await screen.findByTestId('ipam-subnet-10.44.10.0/30'));

    expect(screen.getByTestId('ipam-in-use-empty')).toHaveTextContent(
      'No address here was seen in use in the last day.',
    );
    expect(screen.getByTestId('ipam-utilization')).toHaveTextContent('0%');
  });

  it('marks a scope by its exhaustion', async () => {
    scopes = [
      scope('192.168.1.10', '192.168.1.11', ['192.168.1.10', '192.168.1.11'], 'Lab'),
      scope('192.168.1.100', '192.168.1.199', ['192.168.1.150']),
    ];
    const user = userEvent.setup();
    renderPage();

    const full = await screen.findByTestId('ipam-scope-192.168.1.10');
    expect(full).toHaveAttribute('data-state', 'crit');
    expect(full).toHaveTextContent('Lab · DHCP scope');
    expect(screen.getByTestId('ipam-scope-192.168.1.100')).toHaveAttribute('data-state', 'ok');

    await user.click(full);
    expect(screen.getByTestId('scope-exhaustion')).toHaveTextContent('Full');
    await user.click(screen.getByTestId('ipam-scope-192.168.1.100'));
    expect(screen.queryByTestId('scope-exhaustion')).toBeNull();
  });

  it('adds a scope, selects what the server stored, then deletes it', async () => {
    const user = userEvent.setup();
    renderPage();
    await openAddForm(user);
    await user.type(screen.getByTestId('scope-name'), ' Office ');
    await user.type(screen.getByTestId('scope-start'), '192.168.1.100 ');
    await user.type(screen.getByTestId('scope-end'), '192.168.1.199');
    await user.click(screen.getByTestId('scope-save'));

    await waitFor(() => expect(screen.queryByTestId('scope-save')).toBeNull());
    expect(writes).toEqual([
      {
        method: 'POST',
        path: '/api/v1/ipam/scopes',
        body: { name: 'Office', start: '192.168.1.100', end: '192.168.1.199' },
      },
    ]);
    expect(await screen.findByTestId('ipam-scope-192.168.1.100')).toHaveAttribute(
      'aria-current',
      'true',
    );
    expect(screen.getByTestId('ipam-size')).toHaveTextContent('100');
    expect(screen.getByTestId('ipam-counts')).toHaveTextContent('1 DHCP scope');

    await user.click(screen.getByTestId('scope-delete'));
    await waitFor(() => expect(screen.queryByTestId('ipam-scope-192.168.1.100')).toBeNull());
    expect(writes.at(-1)).toEqual({
      method: 'DELETE',
      path: '/api/v1/ipam/scopes?start=192.168.1.100',
    });
  });

  it('asks for both ends before calling the server', async () => {
    const user = userEvent.setup();
    renderPage();
    await openAddForm(user);
    await user.type(screen.getByTestId('scope-start'), '192.168.1.100');
    await user.click(screen.getByTestId('scope-save'));

    expect(screen.getByTestId('scope-form-error')).toHaveTextContent(
      'Enter the first and last address.',
    );
    expect(writes).toEqual([]);
  });

  it.each([
    [400, 'A DHCP scope is two IPv4 addresses, the first one lower.'],
    [409, 'This range overlaps another DHCP scope.'],
  ])('words a %i refusal and keeps the form open', async (status, wording) => {
    refusal = new ApiError(status, { error: 'server reason' }, undefined, null);
    const user = userEvent.setup();
    renderPage();
    await openAddForm(user);
    await user.type(screen.getByTestId('scope-start'), '192.168.1.199');
    await user.type(screen.getByTestId('scope-end'), '192.168.1.100');
    await user.click(screen.getByTestId('scope-save'));

    expect(await screen.findByTestId('scope-form-error')).toHaveTextContent(wording);
    expect(screen.getByTestId('scope-save')).toBeVisible();
  });

  it('leaves a viewer able to read utilization but not change scopes', async () => {
    role = 'viewer';
    scopes = [scope('192.168.1.100', '192.168.1.199', [])];
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByTestId('ipam-scope-192.168.1.100'));
    // The tooltip keeps a disabled control focusable, so it is aria-disabled.
    await waitFor(() =>
      expect(screen.getByTestId('scope-add')).toHaveAttribute('aria-disabled', 'true'),
    );
    expect(screen.getByTestId('scope-delete')).toHaveAttribute('aria-disabled', 'true');
  });
});
