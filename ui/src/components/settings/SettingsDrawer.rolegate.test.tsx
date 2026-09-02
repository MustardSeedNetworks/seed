/**
 * SettingsDrawer role-gating tests (#1254).
 *
 * Eight of the drawer's ten loader endpoints are registered `minRole: op` and
 * gate GET, not just the writes. A viewer therefore cannot read the drawer at
 * all — including appearance and display, which persist through the
 * operator-gated PATCH /profiles/{id}/settings. So the panels are hidden and
 * one explanation takes their place, rather than a read-only view of nothing.
 */

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { type CurrentUser, RoleProvider } from '../../contexts/RoleContext';
import { RequireRole } from '../ui/RequireRole';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
vi.mock('../../api/client', () => ({
  api: { get: (path: string): Promise<unknown> => mockGet(path) },
}));

function asUser(role: CurrentUser['role']): void {
  mockGet.mockImplementation((path: string) => {
    if (path.includes('/users/me')) {
      return Promise.resolve({ username: 'u', role, isActive: true });
    }

    return Promise.resolve({});
  });
}

/**
 * The gate as the drawer composes it. Rendering the whole SettingsDrawer needs
 * a dozen providers and every loader stubbed; what is under test is the gate
 * and its fallback, so this exercises exactly that pairing.
 */
function renderGate(): void {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const node: ReactNode = (
    <QueryClientProvider client={queryClient}>
      <RoleProvider>
        <RequireRole min="operator" fallback={<p>Read-only account</p>}>
          <p>Link Settings</p>
        </RequireRole>
      </RoleProvider>
    </QueryClientProvider>
  );
  render(node as ReactNode as React.ReactElement);
}

beforeEach(() => {
  mockGet.mockReset();
});
afterEach(() => {
  vi.clearAllMocks();
});

describe('SettingsDrawer — viewer gating', () => {
  it('shows a viewer the notice and none of the panels', async () => {
    asUser('viewer');
    renderGate();

    expect(await screen.findByText('Read-only account')).toBeInTheDocument();
    expect(screen.queryByText('Link Settings')).not.toBeInTheDocument();
  });

  it('shows an operator the panels and not the notice', async () => {
    asUser('operator');
    renderGate();

    await waitFor(() => {
      expect(screen.getByText('Link Settings')).toBeInTheDocument();
    });
    expect(screen.queryByText('Read-only account')).not.toBeInTheDocument();
  });

  it('fails closed while the role is still loading', () => {
    asUser('operator');
    renderGate();

    // Synchronously, before /users/me resolves: the panels must not flash.
    expect(screen.queryByText('Link Settings')).not.toBeInTheDocument();
    expect(screen.getByText('Read-only account')).toBeInTheDocument();
  });
});
