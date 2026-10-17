/**
 * TFTP session card (UI-SEED-44): an admin starts a session with uploads off
 * unless ticked, sees where it listens and what it serves, and stops it; a
 * refused start shows the server's reason; an operator is told who can start
 * one and no admin-only read is sent.
 */
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { TftpSession as Session, TftpSessionRunning } from '../../types/tftp';

let isAdmin = true;
let current: Session = { running: false };
let startFails = false;
const calls: { method: string; body?: unknown }[] = [];

const running: TftpSessionRunning = {
  running: true,
  interface: 'eth0',
  address: '192.0.2.10',
  port: 69,
  directory: '/var/lib/seed/tftp',
  uploadDirectory: '/var/lib/seed/tftp-incoming',
  allowUpload: false,
  startedAt: '2026-10-17T06:00:00Z',
  expiresAt: '2026-10-17T07:00:00Z',
  idleTimeoutSeconds: 600,
  activeTransfers: 2,
};

vi.mock('../../contexts/RoleContext', async (importOriginal) => ({
  ...(await importOriginal<Record<string, unknown>>()),
  useRole: () => ({ isAdmin, canWrite: true, loading: false }),
}));

vi.mock('../../contexts/useProfileInterfaces', () => ({
  useProfileInterfaces: () => ({
    getAllEthernetInterfaces: () => [{ name: 'eth0', enabled: true }],
    getAllWifiInterfaces: () => [],
    getEthernetInterface: () => ({ name: 'eth0', enabled: true }),
  }),
}));

vi.mock('../../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../api/client')>();
  return {
    ...actual,
    api: {
      get: async (): Promise<Session> => {
        calls.push({ method: 'GET' });
        return current;
      },
      post: async (_path: string, body: { allowUpload: boolean }): Promise<Session> => {
        calls.push({ method: 'POST', body });
        if (startFails) {
          throw new actual.ApiError(
            503,
            { error: 'UDP port 69 on that interface is in use or not permitted' },
            undefined,
            null,
          );
        }
        current = { ...running, allowUpload: body.allowUpload };
        return current;
      },
      delete: async (): Promise<Session> => {
        calls.push({ method: 'DELETE' });
        current = { running: false };
        return current;
      },
    },
  };
});

const { TftpSession } = await import('./TftpSession');

describe('TftpSession', () => {
  beforeEach(() => {
    isAdmin = true;
    current = { running: false };
    startFails = false;
    calls.length = 0;
  });

  it('starts on the profile interface with uploads off by default', async () => {
    const user = userEvent.setup();
    render(<TftpSession />);

    expect(await screen.findByTestId('tftp-interface')).toHaveValue('eth0');
    expect(screen.getByTestId('tftp-allow-upload')).not.toBeChecked();
    expect(screen.getByText('Any host on that segment can then write files.')).toBeVisible();

    await user.click(screen.getByTestId('tftp-start'));

    expect(calls.find((c) => c.method === 'POST')?.body).toEqual({
      interface: 'eth0',
      allowUpload: false,
    });
    expect(await screen.findByTestId('tftp-listening')).toHaveTextContent('eth0 · 192.0.2.10:69');
    expect(screen.getByTestId('tftp-uploads')).toHaveTextContent('Off');
    expect(screen.getByTestId('tftp-transfers')).toHaveTextContent('2');
    expect(screen.getByTestId('tftp-running')).toHaveTextContent('after 10 idle minutes');
  });

  it('sends allowUpload when ticked and says where uploads land', async () => {
    const user = userEvent.setup();
    render(<TftpSession />);

    await user.click(await screen.findByTestId('tftp-allow-upload'));
    await user.click(screen.getByTestId('tftp-start'));

    expect(calls.find((c) => c.method === 'POST')?.body).toEqual({
      interface: 'eth0',
      allowUpload: true,
    });
    expect(await screen.findByTestId('tftp-uploads')).toHaveTextContent(
      'Allowed, saved into /var/lib/seed/tftp-incoming',
    );
  });

  it('stops a running session', async () => {
    current = running;
    const user = userEvent.setup();
    render(<TftpSession />);

    await user.click(await screen.findByTestId('tftp-stop'));

    expect(calls.map((c) => c.method)).toContain('DELETE');
    expect(await screen.findByTestId('tftp-start')).toBeInTheDocument();
  });

  it("shows the server's reason when a start is refused", async () => {
    startFails = true;
    const user = userEvent.setup();
    render(<TftpSession />);

    await user.click(await screen.findByTestId('tftp-start'));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'UDP port 69 on that interface is in use or not permitted',
    );
    expect(screen.getByTestId('tftp-start')).toBeInTheDocument();
  });

  it('tells an operator an admin starts it and reads nothing', async () => {
    isAdmin = false;
    render(<TftpSession />);

    expect(screen.getByTestId('tftp-admin-only')).toHaveTextContent('An admin starts');
    expect(screen.queryByTestId('tftp-start')).toBeNull();
    await waitFor(() => expect(calls).toEqual([]));
  });
});
