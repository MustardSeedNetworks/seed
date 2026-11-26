/**
 * ApiTokensSettings.revoke.test.tsx — revoking a token goes through the shared
 * confirmation dialog, not window.confirm.
 *
 * A revoke stops every script using the token at once, so nothing may be sent
 * until the operator has confirmed in the dialog, and cancelling sends nothing.
 */

import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { LicenseProvider } from '../../../contexts/LicenseContext';
import { RoleProvider } from '../../../contexts/RoleContext';
import i18n from '../../../i18n';
import { ApiTokensSettings } from './ApiTokensSettings';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
const mockDelete = vi.fn<(path: string) => Promise<unknown>>();

vi.mock('../../../api/client', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    post: (): Promise<unknown> => Promise.resolve({}),
    delete: (path: string): Promise<unknown> => mockDelete(path),
  },
}));
vi.mock('../../../api', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    post: (): Promise<unknown> => Promise.resolve({}),
    delete: (path: string): Promise<unknown> => mockDelete(path),
  },
}));

beforeEach(async () => {
  await i18n.changeLanguage('en');
  mockGet.mockReset();
  mockDelete.mockReset();
  mockDelete.mockResolvedValue({});
  mockGet.mockImplementation((path: string) => {
    if (path.includes('/users/me')) {
      return Promise.resolve({ username: 'u', role: 'admin', isActive: true });
    }
    if (path.includes('/license')) {
      return Promise.resolve({ tier: 'Pro', canMintTokens: true });
    }
    return Promise.resolve([
      { id: 'tok-1', name: 'monitoring', prefix: 'seed_ab', createdAt: '2026-01-02T03:04:05Z' },
    ]);
  });
});

afterEach(() => {
  vi.clearAllMocks();
});

async function openPanel(): Promise<void> {
  render(
    <LicenseProvider isAuthenticated={true}>
      <RoleProvider isAuthenticated={true}>
        <ApiTokensSettings />
      </RoleProvider>
    </LicenseProvider>,
  );
  await userEvent.click(await screen.findByRole('button', { name: /api tokens/i }));
}

describe('ApiTokensSettings revoke', () => {
  it('sends nothing until the dialog is confirmed', async () => {
    await openPanel();
    await userEvent.click(await screen.findByRole('button', { name: 'Revoke' }));

    expect(mockDelete).not.toHaveBeenCalled();
    const dialog = screen.getByRole('dialog', { name: 'Revoke token "monitoring"?' });
    expect(dialog).toHaveTextContent('Any script using it will stop working immediately.');

    await userEvent.click(within(dialog).getByRole('button', { name: 'Revoke' }));
    await waitFor(() => {
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/tokens/tok-1');
    });
  });

  it('sends nothing when the dialog is cancelled', async () => {
    await openPanel();
    await userEvent.click(await screen.findByRole('button', { name: 'Revoke' }));
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(mockDelete).not.toHaveBeenCalled();
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });
});
