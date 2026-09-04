/**
 * ApiTokensSettings.i18n.test.tsx — the token panel renders real locale copy.
 *
 * #1942. This panel is where an operator goes when a script has stopped
 * working, and it renders the <Trans i18nKey> blocks repaired in #2086, so an
 * assertion here guards that fix directly.
 *
 * The Spanish case asserts no English is left behind. That is the half that
 * bites: a panel can be fully keyed for its paragraphs and still ship a
 * hardcoded button label, and only reading it under `es` shows which.
 */

import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { LicenseProvider } from '../../../contexts/LicenseContext';
import { type CurrentUser, RoleProvider } from '../../../contexts/RoleContext';
import i18n from '../../../i18n';
import { ApiTokensSettings } from './ApiTokensSettings';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();

vi.mock('../../../api/client', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    post: (): Promise<unknown> => Promise.resolve({}),
    delete: (): Promise<unknown> => Promise.resolve({}),
  },
}));
vi.mock('../../../api', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    post: (): Promise<unknown> => Promise.resolve({}),
    delete: (): Promise<unknown> => Promise.resolve({}),
  },
}));

const role: CurrentUser['role'] = 'admin';

function withOneToken(): void {
  mockGet.mockImplementation((path: string) => {
    if (path.includes('/users/me')) {
      return Promise.resolve({ username: 'u', role, isActive: true });
    }
    if (path.includes('/license')) {
      return Promise.resolve({ tier: 'Free', canMintTokens: false });
    }

    return Promise.resolve([
      {
        id: 'tok-1',
        name: 'monitoring',
        prefix: 'seed_ab',
        createdAt: '2026-01-02T03:04:05Z',
      },
    ]);
  });
}

async function renderOpened(language: string): Promise<void> {
  await i18n.changeLanguage(language);
  render(
    <LicenseProvider isAuthenticated={true}>
      <RoleProvider isAuthenticated={true}>
        <ApiTokensSettings />
      </RoleProvider>
    </LicenseProvider>,
  );
  await userEvent.click(await screen.findByRole('button', { name: /api tokens|tokens de api/i }));
}

beforeEach(() => {
  mockGet.mockReset();
  withOneToken();
});

afterEach(async () => {
  vi.clearAllMocks();
  await i18n.changeLanguage('en');
});

describe('ApiTokensSettings — real locale copy', () => {
  it('renders the English panel', async () => {
    await renderOpened('en');

    await waitFor(() => {
      expect(screen.getByText(/Personal-access tokens for programmatic API calls/)).toBeVisible();
    });
    expect(screen.getByText('Token name')).toBeVisible();
    expect(screen.getByRole('button', { name: 'Create token' })).toBeVisible();
    expect(screen.getByRole('columnheader', { name: 'Prefix' })).toBeVisible();
    expect(screen.getByRole('button', { name: 'Revoke' })).toBeVisible();
  });

  it('renders Spanish under es, with no English left behind', async () => {
    await renderOpened('es');

    await waitFor(() => {
      expect(screen.getByText(/Tokens de acceso personal/)).toBeVisible();
    });

    // Every English string this panel can render in its default state. A
    // hardcoded label survives changeLanguage and shows up here.
    for (const english of [
      'API Tokens',
      'Token name',
      'Create token',
      'Prefix',
      'Revoke',
      'active',
    ]) {
      expect(screen.queryByText(english)).toBeNull();
    }
    expect(screen.queryByPlaceholderText('e.g. monitoring-prod')).toBeNull();
  });
});
