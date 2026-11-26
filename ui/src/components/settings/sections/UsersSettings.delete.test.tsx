/**
 * UsersSettings.delete.test.tsx — deleting a user goes through the shared
 * confirmation dialog, and the role control is the shared Select.
 *
 * Deleting a user revokes their sessions and API tokens and cannot be undone,
 * so nothing may be sent before the operator confirms.
 */

import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { LicenseProvider } from '../../../contexts/LicenseContext';
import { RoleProvider } from '../../../contexts/RoleContext';
import i18n from '../../../i18n';
import { UsersSettings } from './UsersSettings';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
const mockDelete = vi.fn<(path: string) => Promise<unknown>>();
const mockPatch = vi.fn<(path: string, body: unknown) => Promise<unknown>>();

vi.mock('../../../api/client', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    post: (): Promise<unknown> => Promise.resolve({}),
    delete: (path: string): Promise<unknown> => mockDelete(path),
    patch: (path: string, body: unknown): Promise<unknown> => mockPatch(path, body),
  },
}));
vi.mock('../../../api', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    post: (): Promise<unknown> => Promise.resolve({}),
    delete: (path: string): Promise<unknown> => mockDelete(path),
    patch: (path: string, body: unknown): Promise<unknown> => mockPatch(path, body),
  },
}));

const admin = { id: 1, username: 'root', role: 'admin', isActive: true, authProvider: 'local' };
const bob = { id: 2, username: 'bob', role: 'viewer', isActive: true, authProvider: 'github' };

beforeEach(async () => {
  await i18n.changeLanguage('en');
  for (const fn of [mockGet, mockDelete, mockPatch]) {
    fn.mockReset();
  }
  mockDelete.mockResolvedValue({});
  mockPatch.mockResolvedValue({});
  mockGet.mockImplementation((path: string) => {
    if (path === '/api/v1/users/me') {
      return Promise.resolve(admin);
    }
    if (path.includes('/license')) {
      return Promise.resolve({ tier: 'Pro', features: ['multi_user'] });
    }
    return Promise.resolve([admin, bob]);
  });
});

afterEach(() => {
  vi.clearAllMocks();
});

async function openPanel(): Promise<void> {
  render(
    <LicenseProvider isAuthenticated={true}>
      <RoleProvider isAuthenticated={true}>
        <UsersSettings />
      </RoleProvider>
    </LicenseProvider>,
  );
  await userEvent.click(await screen.findByRole('button', { name: /users/i }));
}

describe('UsersSettings', () => {
  it('deletes a user only after the dialog is confirmed', async () => {
    await openPanel();
    await userEvent.click(await screen.findByTestId('delete-user-bob'));

    expect(mockDelete).not.toHaveBeenCalled();
    const dialog = screen.getByRole('dialog', { name: 'Delete user bob?' });
    await userEvent.click(within(dialog).getByRole('button', { name: 'Delete user' }));
    await waitFor(() => {
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/users/bob');
    });
  });

  it('keeps the user when the dialog is cancelled', async () => {
    await openPanel();
    await userEvent.click(await screen.findByTestId('delete-user-bob'));
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(mockDelete).not.toHaveBeenCalled();
  });

  it('changes a role from the row select', async () => {
    await openPanel();
    await userEvent.selectOptions(await screen.findByTestId('role-select-bob'), 'operator');

    await waitFor(() => {
      expect(mockPatch).toHaveBeenCalledWith('/api/v1/users/bob', { role: 'operator' });
    });
  });
});
