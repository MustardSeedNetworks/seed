/**
 * RequireRole / RequireAdmin tests — cover the role-rank gate matrix and the
 * fail-closed defaults (loading, fetch error, inactive user), so the #1254 UI
 * gate can't silently regress and start showing admin controls to operators or
 * viewers.
 */

import { render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { type CurrentUser, RoleProvider } from '../../contexts/RoleContext';
import { RequireAdmin, RequireRole } from './RequireRole';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
vi.mock('../../api/client', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
  },
}));

const user = (role: CurrentUser['role'], isActive = true): CurrentUser => ({
  username: 'u',
  role,
  isActive,
});

function renderGated(node: React.ReactNode): ReturnType<typeof render> {
  return render(<RoleProvider>{node}</RoleProvider>);
}

describe('RequireRole / RequireAdmin', () => {
  beforeEach(() => {
    mockGet.mockReset();
  });
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('renders children for a role at or above min (admin sees admin-only)', async () => {
    mockGet.mockResolvedValueOnce(user('admin'));
    renderGated(
      <RequireAdmin>
        <span>admin-only</span>
      </RequireAdmin>,
    );
    await waitFor(() => expect(screen.queryByText('admin-only')).not.toBeNull());
  });

  it('hides children (renders fallback) for a role below min — operator < admin', async () => {
    mockGet.mockResolvedValueOnce(user('operator'));
    renderGated(
      <RequireAdmin fallback={<span>nope</span>}>
        <span>admin-only</span>
      </RequireAdmin>,
    );
    await waitFor(() => expect(screen.queryByText('nope')).not.toBeNull());
    expect(screen.queryByText('admin-only')).toBeNull();
  });

  it('viewer sees neither operator- nor admin-gated content', async () => {
    mockGet.mockResolvedValueOnce(user('viewer'));
    renderGated(
      <>
        <RequireRole min="operator">
          <span>op</span>
        </RequireRole>
        <RequireAdmin>
          <span>adm</span>
        </RequireAdmin>
      </>,
    );
    await waitFor(() => expect(mockGet).toHaveBeenCalled());
    expect(screen.queryByText('op')).toBeNull();
    expect(screen.queryByText('adm')).toBeNull();
  });

  it('operator sees operator-gated but not admin-gated content', async () => {
    mockGet.mockResolvedValueOnce(user('operator'));
    renderGated(
      <>
        <RequireRole min="operator">
          <span>op</span>
        </RequireRole>
        <RequireAdmin>
          <span>adm</span>
        </RequireAdmin>
      </>,
    );
    await waitFor(() => expect(screen.queryByText('op')).not.toBeNull());
    expect(screen.queryByText('adm')).toBeNull();
  });

  it('fails closed on fetch error', async () => {
    mockGet.mockRejectedValueOnce(new Error('boom'));
    renderGated(
      <RequireAdmin>
        <span>admin-only</span>
      </RequireAdmin>,
    );
    await waitFor(() => expect(mockGet).toHaveBeenCalled());
    expect(screen.queryByText('admin-only')).toBeNull();
  });

  it('treats an inactive admin as below min (fail-closed)', async () => {
    mockGet.mockResolvedValueOnce(user('admin', false));
    renderGated(
      <RequireAdmin>
        <span>admin-only</span>
      </RequireAdmin>,
    );
    await waitFor(() => expect(mockGet).toHaveBeenCalled());
    expect(screen.queryByText('admin-only')).toBeNull();
  });
});
