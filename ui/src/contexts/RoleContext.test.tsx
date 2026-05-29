/**
 * RoleContext tests — cover the canWrite/isAdmin matrix, the loading
 * fail-closed default, and the fetch-error fallback, so the #1226 UI
 * gate can't silently regress.
 */

import { act, render, renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { WriteGate } from '../components/ui/WriteGate';
import { type CurrentUser, RoleProvider, useRole } from './RoleContext';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
vi.mock('../api/client', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
  },
}));

const wrapper = ({ children }: { children: React.ReactNode }): React.ReactElement => (
  <RoleProvider>{children}</RoleProvider>
);

const user = (role: CurrentUser['role'], isActive = true): CurrentUser => ({
  username: 'u',
  role,
  isActive,
});

describe('RoleContext / useRole', () => {
  beforeEach(() => {
    mockGet.mockReset();
  });
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('fetches /users/me on mount and exposes role + helpers', async () => {
    mockGet.mockResolvedValueOnce(user('operator'));
    const { result } = renderHook(() => useRole(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(mockGet).toHaveBeenCalledWith('/api/v1/users/me');
    expect(result.current.user?.role).toBe('operator');
    expect(result.current.canWrite).toBe(true);
    expect(result.current.isAdmin).toBe(false);
  });

  it('viewer cannot write and is not admin', async () => {
    mockGet.mockResolvedValueOnce(user('viewer'));
    const { result } = renderHook(() => useRole(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.canWrite).toBe(false);
    expect(result.current.isAdmin).toBe(false);
  });

  it('admin can write and is admin', async () => {
    mockGet.mockResolvedValueOnce(user('admin'));
    const { result } = renderHook(() => useRole(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.canWrite).toBe(true);
    expect(result.current.isAdmin).toBe(true);
  });

  it('inactive user cannot write even with operator role', async () => {
    mockGet.mockResolvedValueOnce(user('operator', false));
    const { result } = renderHook(() => useRole(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.canWrite).toBe(false);
    expect(result.current.isAdmin).toBe(false);
  });

  it('fails closed on fetch error: canWrite=false, error surfaced', async () => {
    mockGet.mockRejectedValueOnce(new Error('boom'));
    const { result } = renderHook(() => useRole(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.user).toBeNull();
    expect(result.current.canWrite).toBe(false);
    expect(result.current.isAdmin).toBe(false);
    expect(result.current.error).toBe('boom');
  });

  it('refresh() re-fetches /users/me and updates state', async () => {
    mockGet.mockResolvedValueOnce(user('viewer'));
    const { result } = renderHook(() => useRole(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.canWrite).toBe(false);

    mockGet.mockResolvedValueOnce(user('operator'));
    await act(async () => {
      await result.current.refresh();
    });
    expect(result.current.canWrite).toBe(true);
  });

  it('throws if useRole is called outside RoleProvider', () => {
    const orig = console.error;
    console.error = (): void => {}; // silence React's expected error boundary log
    try {
      expect(() => renderHook(() => useRole())).toThrow(/inside <RoleProvider>/);
    } finally {
      console.error = orig;
    }
  });
});

describe('<WriteGate>', () => {
  beforeEach(() => {
    mockGet.mockReset();
  });

  it('renders children for operator', async () => {
    mockGet.mockResolvedValueOnce(user('operator'));
    const { findByText } = render(
      <RoleProvider>
        <WriteGate fallback={<span>blocked</span>}>
          <span>writable</span>
        </WriteGate>
      </RoleProvider>,
    );
    await findByText('writable');
  });

  it('renders fallback for viewer', async () => {
    mockGet.mockResolvedValueOnce(user('viewer'));
    const { findByText } = render(
      <RoleProvider>
        <WriteGate fallback={<span>blocked</span>}>
          <span>writable</span>
        </WriteGate>
      </RoleProvider>,
    );
    await findByText('blocked');
  });
});
