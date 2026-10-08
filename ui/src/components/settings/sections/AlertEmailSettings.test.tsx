/**
 * AlertEmailSettings.test.tsx — the alert mail relay editor (#3209).
 *
 * The server applies alerts.email field by field and refuses a relay that
 * could never deliver, so what matters here is what the section sends: the
 * recipients as a list, the password only when one was typed, and a refusal
 * shown in the server's own words.
 */

import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { type CurrentUser, RoleProvider } from '../../../contexts/RoleContext';
import i18n from '../../../i18n';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
const mockPut = vi.fn<(path: string, body: unknown) => Promise<unknown>>();
// The section reads through `api`, RoleContext through `api/client`.
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

const { AlertEmailSettings } = await import('./AlertEmailSettings');

const STORED = {
  host: 'smtp.example.test',
  port: 0,
  tls: '',
  username: 'seed',
  passwordSet: true,
  from: 'Seed <seed@example.test>',
  to: ['noc@example.test', 'oncall@example.test'],
};

function serve(role: CurrentUser['role'], email: unknown): void {
  mockGet.mockImplementation((path: string) => {
    if (path.includes('/users/me')) {
      return Promise.resolve({ username: 'u', role, isActive: true });
    }
    if (path === '/api/v1/settings') {
      return Promise.resolve({ alerts: { email } });
    }
    return Promise.resolve({});
  });
}

async function open(): Promise<void> {
  render(
    <RoleProvider isAuthenticated={true}>
      <AlertEmailSettings />
    </RoleProvider>,
  );
  await userEvent.click(await screen.findByRole('button', { name: /alert email/i }));
}

function sentEmail(): unknown {
  expect(mockPut).toHaveBeenCalledTimes(1);
  const [path, body] = mockPut.mock.calls[0] ?? [];
  expect(path).toBe('/api/v1/settings');
  return (body as { alerts: { email: unknown } }).alerts.email;
}

beforeEach(() => {
  mockGet.mockReset();
  mockPut.mockReset();
  mockPut.mockResolvedValue({});
});

afterEach(async () => {
  vi.clearAllMocks();
  await i18n.changeLanguage('en');
});

describe('AlertEmailSettings', () => {
  it('shows the stored relay without its password', async () => {
    serve('operator', STORED);
    await open();

    await waitFor(() =>
      expect(screen.getByTestId('alert-email-host')).toHaveValue('smtp.example.test'),
    );
    expect(screen.getByTestId('alert-email-tls')).toHaveValue('starttls');
    expect(screen.getByTestId('alert-email-port')).toHaveValue(null);
    expect(screen.getByTestId('alert-email-port')).toHaveAttribute('placeholder', '587');
    expect(screen.getByTestId('alert-email-username')).toHaveValue('seed');
    expect(screen.getByTestId('alert-email-password')).toHaveValue('');
    expect(screen.getByTestId('alert-email-password')).toHaveAttribute(
      'placeholder',
      'A password is stored; type to replace it',
    );
    expect(screen.getByTestId('alert-email-to')).toHaveValue(
      'noc@example.test, oncall@example.test',
    );
  });

  it('saves a new relay with its recipients as a list', async () => {
    serve('operator', { host: '', port: 0, tls: '', username: '', passwordSet: false, to: [] });
    await open();

    await userEvent.type(await screen.findByTestId('alert-email-host'), 'mail.example.test');
    await userEvent.selectOptions(screen.getByTestId('alert-email-tls'), 'tls');
    await userEvent.type(screen.getByTestId('alert-email-port'), '2465');
    await userEvent.type(screen.getByTestId('alert-email-username'), 'alerts');
    await userEvent.type(screen.getByTestId('alert-email-password'), 'hunter2');
    await userEvent.type(screen.getByTestId('alert-email-from'), 'seed@example.test');
    await userEvent.type(
      screen.getByTestId('alert-email-to'),
      'a@example.test,{enter} b@example.test ,',
    );
    await userEvent.click(screen.getByTestId('alert-email-save'));

    await waitFor(() => expect(mockPut).toHaveBeenCalled());
    expect(sentEmail()).toEqual({
      host: 'mail.example.test',
      port: 2465,
      tls: 'tls',
      username: 'alerts',
      password: 'hunter2',
      from: 'seed@example.test',
      to: ['a@example.test', 'b@example.test'],
    });
    // The password has left the browser and is now the stored one.
    await waitFor(() => expect(screen.getByTestId('alert-email-password')).toHaveValue(''));
    expect(screen.getByTestId('alert-email-password')).toHaveAttribute(
      'placeholder',
      'A password is stored; type to replace it',
    );
  });

  it('leaves the stored password alone when none is typed', async () => {
    serve('operator', STORED);
    await open();

    const host = await screen.findByTestId('alert-email-host');
    await waitFor(() => expect(host).toHaveValue('smtp.example.test'));
    await userEvent.clear(host);
    await userEvent.type(host, 'relay.example.test');
    await userEvent.click(screen.getByTestId('alert-email-save'));

    await waitFor(() => expect(mockPut).toHaveBeenCalled());
    expect(sentEmail()).not.toHaveProperty('password');
    expect(sentEmail()).toMatchObject({ host: 'relay.example.test', tls: 'starttls', port: 0 });
  });

  it("shows the server's reason when it refuses the relay", async () => {
    serve('operator', STORED);
    mockPut.mockRejectedValue(
      new Error('invalid delivery config: at least one recipient is required'),
    );
    await open();

    await userEvent.click(await screen.findByTestId('alert-email-save'));

    expect(await screen.findByTestId('alert-email-error')).toHaveTextContent(
      'at least one recipient is required',
    );
  });

  it('is read-only for a viewer', async () => {
    serve('viewer', STORED);
    await open();

    // The tooltip keeps a disabled button focusable so the reason is readable.
    await waitFor(() =>
      expect(screen.getByTestId('alert-email-save')).toHaveAttribute('aria-disabled', 'true'),
    );
    expect(screen.getByTestId('alert-email-host')).toBeDisabled();
    expect(screen.getByTestId('alert-email-password')).toBeDisabled();
    expect(screen.getByTestId('alert-email-to')).toBeDisabled();
    await userEvent.click(screen.getByTestId('alert-email-save'));
    expect(mockPut).not.toHaveBeenCalled();
  });
});
