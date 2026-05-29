/**
 * WriteGate — render children only if the current user has write access.
 *
 * Hide-variant role gating, parallel to <RequireFeature> for license
 * tier. Use it to wrap entire forms, save buttons, "Create" toolbars,
 * or any UI surface that mutates persistent state on the backend (per
 * the seed#1226 viewer-is-read-only policy).
 *
 * A `viewer` user (or any unauthenticated / fetch-failed state) sees
 * `fallback` instead of children — default `null`, so a "Save" button
 * simply disappears. For controls that should remain visible but
 * disabled with an explanatory tooltip, read `useRole().canWrite`
 * directly and apply `disabled` + a title attribute.
 *
 * Example — hide:
 * ```tsx
 * <WriteGate>
 *   <Button onClick={save}>Save</Button>
 * </WriteGate>
 * ```
 *
 * Example — disable instead of hide:
 * ```tsx
 * const { canWrite } = useRole();
 * <Button disabled={!canWrite} title={canWrite ? '' : 'Read-only — operator role required'}>
 *   Save
 * </Button>
 * ```
 *
 * While the initial /users/me fetch is in flight, `canWrite` is false,
 * so children render `fallback` — fail-closed, matching RequireFeature.
 */

import type { ReactElement, ReactNode } from 'react';
import { useRole } from '../../contexts/RoleContext';

interface WriteGateProps {
  children: ReactNode;
  /** Rendered when the user lacks write access (viewer/loading/error). */
  fallback?: ReactNode;
}

export function WriteGate({ children, fallback = null }: WriteGateProps): ReactElement {
  const { canWrite } = useRole();
  if (!canWrite) {
    return <>{fallback}</>;
  }
  return <>{children}</>;
}
