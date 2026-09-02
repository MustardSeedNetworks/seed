/**
 * InsecurePortScanCard tests (#347).
 *
 * The issue's actual ask is not "scan some ports" — the backend already did
 * that under `profile: "insecure"`. It is that a finding explains itself:
 * "Telnet (23) is open. This protocol sends credentials in cleartext." So the
 * assertions are on the explanation reaching the screen, not just the port.
 */

import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { type CurrentUser, RoleProvider } from '../../contexts/RoleContext';
import { InsecurePortScanCard } from './InsecurePortScanCard';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
const mockPost = vi.fn<(path: string, body: unknown) => Promise<unknown>>();

vi.mock('../../api/client', () => ({
  api: {
    get: (p: string): Promise<unknown> => mockGet(p),
    post: (p: string, b: unknown): Promise<unknown> => mockPost(p, b),
  },
}));
vi.mock('../../api', () => ({
  api: {
    get: (p: string): Promise<unknown> => mockGet(p),
    post: (p: string, b: unknown): Promise<unknown> => mockPost(p, b),
  },
}));

function asUser(role: CurrentUser['role']): void {
  mockGet.mockImplementation((path: string) =>
    path.includes('/users/me')
      ? Promise.resolve({ username: 'u', role, isActive: true })
      : Promise.resolve({}),
  );
}

function renderCard(): void {
  render(
    <RoleProvider>
      <InsecurePortScanCard />
    </RoleProvider>,
  );
}

beforeEach(() => {
  mockGet.mockReset();
  mockPost.mockReset();
});
afterEach(() => {
  vi.clearAllMocks();
});

describe('InsecurePortScanCard', () => {
  it('scans with the insecure profile and explains each finding', async () => {
    asUser('operator');
    mockPost.mockResolvedValue({
      ip: '192.0.2.10',
      services: [
        { port: 23, state: 'open', service: 'telnet' },
        { port: 445, state: 'open', service: 'microsoft-ds' },
        { port: 22, state: 'closed', service: 'ssh' },
      ],
      scanTime: 1200,
    });
    renderCard();

    await userEvent.type(await screen.findByLabelText('Target address'), '192.0.2.10');
    await userEvent.click(screen.getByRole('button', { name: 'Scan' }));

    // The backend owns the port list; the card must ask for it by name.
    await waitFor(() => {
      expect(mockPost).toHaveBeenCalledWith('/api/v1/security/discovery/portscan', {
        target: '192.0.2.10',
        profile: 'insecure',
      });
    });

    expect(await screen.findByText('Port 23 — telnet')).toBeInTheDocument();
    expect(
      screen.getByText('Telnet sends credentials and every keystroke in cleartext. Use SSH.'),
    ).toBeInTheDocument();
    expect(screen.getByText(/SMB exposes file shares/)).toBeInTheDocument();

    // A closed port is not a finding.
    expect(screen.queryByText(/Port 22/)).not.toBeInTheDocument();
  });

  it('explains a port in the X11 range, which is a span not a single port', async () => {
    asUser('operator');
    mockPost.mockResolvedValue({
      ip: '192.0.2.10',
      services: [{ port: 6003, state: 'open', service: 'x11' }],
      scanTime: 90,
    });
    renderCard();

    await userEvent.type(await screen.findByLabelText('Target address'), '192.0.2.10');
    await userEvent.click(screen.getByRole('button', { name: 'Scan' }));

    expect(await screen.findByText(/An open X11 display/)).toBeInTheDocument();
  });

  it('falls back to a generic explanation for a port with no specific copy', async () => {
    asUser('operator');
    mockPost.mockResolvedValue({
      ip: '192.0.2.10',
      services: [{ port: 1099, state: 'open', service: '' }],
      scanTime: 40,
    });
    renderCard();

    await userEvent.type(await screen.findByLabelText('Target address'), '192.0.2.10');
    await userEvent.click(screen.getByRole('button', { name: 'Scan' }));

    // 1099 has copy; the unnamed service falls back, not the risk text.
    expect(await screen.findByText('Port 1099 — unidentified service')).toBeInTheDocument();
    expect(screen.getByText(/Java RMI has repeatedly allowed/)).toBeInTheDocument();
  });

  it('says so plainly when nothing insecure is open', async () => {
    asUser('operator');
    mockPost.mockResolvedValue({ ip: '192.0.2.10', services: [], scanTime: 30 });
    renderCard();

    await userEvent.type(await screen.findByLabelText('Target address'), '192.0.2.10');
    await userEvent.click(screen.getByRole('button', { name: 'Scan' }));

    expect(await screen.findByText('No insecure ports open on 192.0.2.10.')).toBeInTheDocument();
  });

  it('does not let a viewer run a scan', async () => {
    asUser('viewer');
    renderCard();

    const input = await screen.findByLabelText('Target address');
    await waitFor(() => {
      expect(input).toBeDisabled();
    });
    expect(screen.getByRole('button', { name: 'Scan' })).toBeDisabled();
    expect(mockPost).not.toHaveBeenCalled();
  });
});
