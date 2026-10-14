/**
 * SsoSettings.test.tsx — re-saving a configured provider (#3280).
 *
 * The page sends every field back on save, and the server never returns the
 * secret. So the page must start from the stored client ID, redirect URL and
 * scopes, and send an empty secret when none was typed, which the server reads
 * as "keep the stored one".
 */

import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { RoleProvider } from '../../../contexts/RoleContext';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
const mockPut = vi.fn<(path: string, body: unknown) => Promise<unknown>>();
vi.mock('../../../api/client', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    put: (path: string, body: unknown): Promise<unknown> => mockPut(path, body),
  },
}));
vi.mock('../../../contexts/LicenseContext', () => ({
  useLicense: (): { status: { features: string[] } } => ({ status: { features: ['sso'] } }),
}));

const { SsoSettings } = await import('./SsoSettings');

const STORED = {
  providers: [
    {
      name: 'google',
      enabled: true,
      clientId: 'stored-client-id',
      hasClientSecret: true,
      redirectUrl: 'https://seed.example.test/api/v1/sso/callback',
      scopes: ['openid', 'email'],
    },
  ],
};

async function open(): Promise<void> {
  render(
    <RoleProvider isAuthenticated={true}>
      <SsoSettings />
    </RoleProvider>,
  );
  await userEvent.click(await screen.findByRole('button', { name: /single sign-on/i }));
}

beforeEach(() => {
  mockGet.mockImplementation((path: string) => {
    if (path.includes('/users/me')) {
      return Promise.resolve({ username: 'u', role: 'operator', isActive: true });
    }
    if (path.includes('/sso/settings')) {
      return Promise.resolve(STORED);
    }
    return Promise.resolve({});
  });
  mockPut.mockResolvedValue({});
});

afterEach(() => {
  vi.clearAllMocks();
});

describe('SsoSettings', () => {
  it('shows the stored provider without its secret', async () => {
    await open();

    await waitFor(() =>
      expect(screen.getByTestId('sso-client-id-google')).toHaveValue('stored-client-id'),
    );
    expect(screen.getByTestId('sso-redirect-google')).toHaveValue(
      'https://seed.example.test/api/v1/sso/callback',
    );
    expect(screen.getByTestId('sso-client-secret-google')).toHaveValue('');
    expect(screen.getByTestId('sso-client-secret-google')).toHaveAttribute(
      'placeholder',
      'A secret is stored; type to replace it',
    );
    expect(screen.getByTestId('sso-client-secret-github')).toHaveAttribute('placeholder', '');
  });

  it('re-saves the stored fields and leaves the secret to the server', async () => {
    await open();

    await waitFor(() =>
      expect(screen.getByTestId('sso-client-id-google')).toHaveValue('stored-client-id'),
    );
    await userEvent.click(screen.getByTestId('sso-save-google'));

    await waitFor(() => expect(mockPut).toHaveBeenCalledTimes(1));
    expect(mockPut).toHaveBeenCalledWith('/api/v1/sso/update', {
      provider: 'google',
      enabled: true,
      clientId: 'stored-client-id',
      clientSecret: '',
      redirectUrl: 'https://seed.example.test/api/v1/sso/callback',
      tenantId: undefined,
      scopes: ['openid', 'email'],
    });
  });
});
