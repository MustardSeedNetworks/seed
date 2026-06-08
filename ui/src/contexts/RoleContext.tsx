/**
 * RoleContext — current user's role and write-permission helpers.
 *
 * Fetches GET /api/v1/users/me once at app boot, caches the result, and
 * exposes it via `useRole()`. UI components use this to render or hide
 * write controls so a `viewer` user sees a consistent read-only view
 * rather than clicking buttons that 403 against the seed#1226 write
 * gate.
 *
 * Production seed always returns a 200 from /users/me for an
 * authenticated session (the bootstrap admin is seeded at first run),
 * so the loading state is brief. On fetch error the role falls back to
 * `null` and the helpers report "no write access" — fail-closed so a
 * broken /users/me never accidentally enables writes for a viewer.
 *
 * Single-user / no-DB seed deployments are admin implicitly (the
 * backend's callerRole tolerates a missing user DB); the /users/me
 * response will be the env-configured username with admin role, so
 * `canWrite` resolves true and no UI changes.
 */

import { createContext, type ReactNode, useCallback, useContext, useEffect, useState } from 'react';
import { api } from '../api/client';

/** Allowed role values, mirroring the seed DB CHECK constraint. */
export type Role = 'admin' | 'operator' | 'viewer';

/**
 * Shape of GET /api/v1/users/me — a subset of the backend UserResponse
 * sufficient for role gating. Extend in lockstep with the server type
 * if more fields are needed for UI decisions.
 */
export interface CurrentUser {
  username: string;
  role: Role;
  isActive: boolean;
}

interface RoleContextValue {
  /** Latest /users/me fetch, or null while loading / on fetch error. */
  user: CurrentUser | null;
  /** True while the initial fetch is in flight. */
  loading: boolean;
  /** Last fetch error message, or null if none. */
  error: string | null;
  /** Force a re-fetch (e.g. after role change in another tab). */
  refresh: () => Promise<void>;
  /**
   * True iff the current user may perform write operations against the
   * seed#1226 write gate (operator or admin). Fail-closed: `false`
   * during loading and on fetch error, so write controls hide rather
   * than flash visible.
   */
  canWrite: boolean;
  /**
   * True iff the current user is admin. Use for admin-only UI sections
   * such as user management.
   */
  isAdmin: boolean;
}

const RoleContext = createContext<RoleContextValue | null>(null);

interface RoleProviderProps {
  children: ReactNode;
}

const ROLE_RANK: Record<Role, number> = { viewer: 1, operator: 2, admin: 3 };

/**
 * meetsRole reports whether `user`'s role rank is at least `min`. Fail-closed:
 * returns false for a null user (also the loading state), an inactive user, so
 * write/admin controls hide rather than flash visible. Shared by canWrite /
 * isAdmin and by the <RequireRole> / <RequireAdmin> gates so all role decisions
 * use one rank comparison (#1254).
 */
export function meetsRole(user: CurrentUser | null, min: Role): boolean {
  return user?.isActive === true && ROLE_RANK[user.role] >= ROLE_RANK[min];
}

export function RoleProvider({ children }: RoleProviderProps): React.ReactElement {
  const [user, setUser] = useState<CurrentUser | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async (): Promise<void> => {
    setError(null);
    try {
      const fresh = await api.get<CurrentUser>('/api/v1/users/me');
      setUser(fresh);
    } catch (err) {
      setUser(null);
      setError(err instanceof Error ? err.message : 'Failed to load current user');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const canWrite = meetsRole(user, 'operator');
  const isAdmin = meetsRole(user, 'admin');

  return (
    <RoleContext.Provider value={{ user, loading, error, refresh, canWrite, isAdmin }}>
      {children}
    </RoleContext.Provider>
  );
}

/**
 * useRole returns the current user's role state. Throws if called
 * outside a `<RoleProvider>` — that always indicates a wiring bug.
 */
export function useRole(): RoleContextValue {
  const ctx = useContext(RoleContext);
  if (!ctx) {
    throw new Error('useRole() must be called inside <RoleProvider>');
  }
  return ctx;
}
