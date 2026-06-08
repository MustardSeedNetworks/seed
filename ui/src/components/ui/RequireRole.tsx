/**
 * RequireRole — render children only if the current user's role rank is at
 * least `min` (viewer < operator < admin).
 *
 * Role-rank gate parallel to <WriteGate> (which is effectively
 * RequireRole min="operator") but for the admin/operator distinction: use it to
 * hide controls — or whole sections — that a lower-privileged role shouldn't
 * even see. User and license management are admin-only, so an operator (who CAN
 * write operator-level settings) must still not see them.
 *
 * A lower-privileged user — and the loading / fetch-error state — renders
 * `fallback` (default null), so the control simply disappears. Fail-closed,
 * matching WriteGate and RequireFeature.
 *
 * Backend role enforcement (seed#1226) stays authoritative; this gate is
 * defense-in-depth + UX so a viewer/operator never sees a button that 403s
 * (#1254).
 *
 * Example:
 * ```tsx
 * <RequireAdmin>
 *   <UsersSettings />
 * </RequireAdmin>
 * ```
 */

import type { ReactElement, ReactNode } from 'react';
import { meetsRole, type Role, useRole } from '../../contexts/RoleContext';

interface RequireRoleProps {
  /** Minimum role required to render `children`. */
  min: Role;
  children: ReactNode;
  /** Rendered when the user's role is below `min` (or loading / fetch error). */
  fallback?: ReactNode;
}

export function RequireRole({ min, children, fallback = null }: RequireRoleProps): ReactElement {
  const { user } = useRole();
  if (!meetsRole(user, min)) {
    return <>{fallback}</>;
  }
  return <>{children}</>;
}

/** RequireAdmin is the common admin-only case — shorthand for `min="admin"`. */
export function RequireAdmin({
  children,
  fallback = null,
}: Omit<RequireRoleProps, 'min'>): ReactElement {
  return (
    <RequireRole min="admin" fallback={fallback}>
      {children}
    </RequireRole>
  );
}
