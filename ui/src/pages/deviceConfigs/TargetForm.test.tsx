/**
 * TargetForm (UI-SEED-45): only SSH logins are offered, an operator with none
 * stores one in place, a refused save does not store that login twice, and
 * an edit keeps the target's own login.
 */
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { DeviceConfigTarget, DeviceConfigTargetInput } from '../../types/deviceConfigs';
import type { Credentials } from '../../types/generated/device-credential-list-response';

let credentials: Credentials[] = [];
const posted: { path: string; body: unknown }[] = [];

function credential(id: string, kind: string, sshUser?: string): Credentials {
  return {
    id,
    clientId: 'default',
    name: id,
    kind,
    sshUser,
    createdAt: '2026-10-17T00:00:00Z',
    updatedAt: '2026-10-17T00:00:00Z',
  };
}

vi.mock('../../api/client', async (importOriginal) => ({
  ...(await importOriginal<Record<string, unknown>>()),
  api: {
    get: async (): Promise<unknown> => ({ count: credentials.length, credentials }),
    post: async (path: string, body: unknown): Promise<unknown> => {
      posted.push({ path, body });
      return credential('cred-new', 'ssh', 'backup');
    },
  },
}));

const { TargetForm } = await import('./TargetForm');

describe('TargetForm', () => {
  beforeEach(() => {
    credentials = [];
    posted.length = 0;
  });

  it('offers only SSH logins', async () => {
    credentials = [credential('snmp', 'v2c'), credential('ssh-ro', 'ssh', 'backup')];
    render(<TargetForm onSubmit={vi.fn()} onCancel={vi.fn()} />);

    const select = await screen.findByTestId('config-target-credential');
    await screen.findByRole('option', { name: 'ssh-ro (backup)' });
    expect(select).not.toHaveTextContent('snmp');
    expect(screen.queryByTestId('config-target-new-login')).toBeNull();
  });

  it('stores a new SSH login once, even when the device save is refused', async () => {
    const user = userEvent.setup();
    const saves: DeviceConfigTargetInput[] = [];
    const onSubmit = vi.fn(async (input: DeviceConfigTargetInput) => {
      saves.push(input);
      if (saves.length === 1) {
        throw new Error('host must be an IP address or a hostname');
      }
    });
    render(<TargetForm onSubmit={onSubmit} onCancel={vi.fn()} />);

    // With no SSH login stored, the new-login fields are the only way forward.
    await screen.findByTestId('config-target-new-login');
    await user.type(screen.getByTestId('config-target-name'), 'edge-02');
    await user.type(screen.getByTestId('config-target-host'), '10.0.0.2');
    await user.selectOptions(screen.getByTestId('config-target-platform'), 'arista-eos');
    await user.type(screen.getByTestId('config-target-login-name'), 'lab backup');
    await user.type(screen.getByTestId('config-target-login-user'), 'backup');
    await user.type(screen.getByTestId('config-target-login-password'), 'hunter2');
    await user.click(screen.getByTestId('config-target-save'));

    expect(await screen.findByTestId('config-target-form-error')).toHaveTextContent(
      'host must be an IP address or a hostname',
    );
    await user.click(screen.getByTestId('config-target-save'));

    expect(posted).toEqual([
      {
        path: '/api/v1/device-credentials',
        body: { name: 'lab backup', sshUser: 'backup', sshPassword: 'hunter2' },
      },
    ]);
    expect(saves).toEqual([
      {
        name: 'edge-02',
        host: '10.0.0.2',
        port: undefined,
        platform: 'arista-eos',
        credentialsId: 'cred-new',
      },
      {
        name: 'edge-02',
        host: '10.0.0.2',
        port: undefined,
        platform: 'arista-eos',
        credentialsId: 'cred-new',
      },
    ]);
  });

  it("keeps an edited device's own login and port", async () => {
    const user = userEvent.setup();
    credentials = [credential('ssh-a', 'ssh'), credential('ssh-b', 'ssh')];
    const target: DeviceConfigTarget = {
      id: 'dct-00000000000a',
      name: 'core-01',
      host: '10.44.10.1',
      port: 2222,
      platform: 'juniper-junos',
      credentialsId: 'ssh-b',
      updatedAt: '2026-10-15T10:00:00Z',
    };
    const onSubmit = vi.fn(async () => undefined);
    render(<TargetForm target={target} onSubmit={onSubmit} onCancel={vi.fn()} />);
    await screen.findByRole('option', { name: 'ssh-b' });

    await user.click(screen.getByTestId('config-target-save'));

    expect(onSubmit).toHaveBeenCalledWith({
      name: 'core-01',
      host: '10.44.10.1',
      port: 2222,
      platform: 'juniper-junos',
      credentialsId: 'ssh-b',
    });
    expect(posted).toEqual([]);
  });
});
