/**
 * GuestNetworkAuditSettings tests (#1004).
 *
 * The audit shipped without an editor, so the feature was unreachable from the
 * UI: GuestNetworkAuditCard renders only once `enabled` is true and a target
 * exists. These cover the path that makes it reachable — add a target, edit the
 * port list, save — plus the role gate, since the route is `minRole: op`.
 */

import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { type CurrentUser, RoleProvider } from '../../../contexts/RoleContext';
import { GuestNetworkAuditSettings } from './GuestNetworkAuditSettings';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
const mockPut = vi.fn<(path: string, body: unknown) => Promise<unknown>>();

vi.mock('../../../api/client', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    put: (path: string, body: unknown): Promise<unknown> => mockPut(path, body),
  },
}));
vi.mock('../../../api', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    put: (path: string, body: unknown): Promise<unknown> => mockPut(path, body),
  },
}));

function asUser(role: CurrentUser['role'], targets: { ip: string; label?: string }[] = []): void {
  mockGet.mockImplementation((path: string) => {
    if (path.includes('/users/me')) {
      return Promise.resolve({ username: 'u', role, isActive: true });
    }

    return Promise.resolve({ enabled: false, targets });
  });
  mockPut.mockResolvedValue({});
}

async function renderOpened(): Promise<void> {
  render(
    <RoleProvider isAuthenticated={true}>
      <GuestNetworkAuditSettings />
    </RoleProvider>,
  );
  await userEvent.click(await screen.findByRole('button', { name: /guest network audit/i }));
}

beforeEach(() => {
  mockGet.mockReset();
  mockPut.mockReset();
});
afterEach(() => {
  vi.clearAllMocks();
});

describe('GuestNetworkAuditSettings', () => {
  it('adds a target and sends it on save', async () => {
    asUser('operator');
    await renderOpened();

    await userEvent.type(screen.getByLabelText('Target address'), '192.0.2.10');
    await userEvent.type(screen.getByLabelText('Label (optional)'), 'File server');
    await userEvent.click(screen.getByRole('button', { name: 'Add target' }));

    expect(screen.getByText('File server')).toBeInTheDocument();
    expect(screen.getByText('192.0.2.10')).toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => {
      expect(mockPut).toHaveBeenCalledWith(
        '/api/v1/security/guest-audit/settings',
        expect.objectContaining({
          targets: [{ ip: '192.0.2.10', label: 'File server' }],
        }),
      );
    });
  });

  it('refuses an address that is not IPv4, and adds nothing', async () => {
    asUser('operator');
    await renderOpened();

    await userEvent.type(screen.getByLabelText('Target address'), 'not-an-address');
    await userEvent.click(screen.getByRole('button', { name: 'Add target' }));

    expect(screen.getByText('Enter an IPv4 address, for example 192.0.2.10.')).toBeInTheDocument();
    expect(
      screen.getByText(
        'No targets yet. Add the addresses guest clients must not be able to reach.',
      ),
    ).toBeInTheDocument();
  });

  it('refuses a duplicate target', async () => {
    asUser('operator', [{ ip: '192.0.2.10' }]);
    await renderOpened();

    await userEvent.type(screen.getByLabelText('Target address'), '192.0.2.10');
    await userEvent.click(screen.getByRole('button', { name: 'Add target' }));

    expect(screen.getByText('That address is already a target.')).toBeInTheDocument();
  });

  it('rejects a port outside 1-65535 and names it', async () => {
    asUser('operator');
    await renderOpened();

    const ports = screen.getByLabelText('Ports to probe');
    await userEvent.clear(ports);
    await userEvent.type(ports, '80, 70000');
    await userEvent.tab();

    expect(screen.getByText('70000 is not a port between 1 and 65535.')).toBeInTheDocument();
  });

  it('names each Remove control by the address it removes', async () => {
    asUser('operator', [{ ip: '192.0.2.10' }, { ip: '192.0.2.11' }]);
    await renderOpened();

    expect(screen.getByRole('button', { name: 'Remove target 192.0.2.10' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Remove target 192.0.2.11' })).toBeInTheDocument();
  });

  it('disables every control for a viewer', async () => {
    asUser('viewer', [{ ip: '192.0.2.10' }]);
    await renderOpened();

    const section = screen.getByTestId('guest-audit-settings-section');
    for (const control of within(section).getAllByRole('button')) {
      // The section header itself is a button and stays operable.
      if (control.textContent?.includes('Guest Network Audit')) {
        continue;
      }
      expect(control).toBeDisabled();
    }
    expect(screen.getByLabelText('Target address')).toBeDisabled();
    expect(screen.getByLabelText('Run the audit')).toBeDisabled();
  });
});
