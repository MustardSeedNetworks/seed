/**
 * ReadOnlyView — wraps a settings panel so a viewer-role user sees the
 * current configuration but cannot mutate it.
 *
 * Renders an explanatory banner when the current user lacks write
 * access, and disables every descendant native form control via a
 * `<fieldset disabled>`. The fieldset trick works because the HTML
 * spec propagates `disabled` to every nested input, select, textarea,
 * and button — one wrap on the form root locks down the whole surface
 * regardless of how many sub-controls exist, sparing each settings
 * section a per-control sweep.
 *
 * Operator and admin users see no chrome change: the wrapper renders
 * children directly (no banner, no fieldset).
 *
 * Use this when a settings panel has many small inputs and one or more
 * action buttons that all map to the same gated backend route. For
 * surfaces with a single obvious write button, prefer the inline
 * `useRole().canWrite` + `disabled` pattern so the tooltip can compose
 * with other gates (e.g. license tier in <ApiTokensSettings>).
 */

import type { ReactElement, ReactNode } from 'react';
import { useRole } from '../../contexts/RoleContext';

interface ReadOnlyViewProps {
  children: ReactNode;
  /**
   * Optional override for the banner copy — useful when the surface's
   * own terminology fits better than the default "settings on this
   * panel are read-only" phrasing.
   */
  notice?: string;
}

export function ReadOnlyView({ children, notice }: ReadOnlyViewProps): ReactElement {
  const { canWrite } = useRole();
  if (canWrite) {
    return <>{children}</>;
  }
  return (
    <div className="stack-sm">
      <div
        role="status"
        className="rounded-lg border border-status-info/30 bg-status-info/5 pad-sm text-sm text-status-info"
      >
        {notice ?? 'Read-only — your role does not allow changes on this panel.'}
      </div>
      <fieldset disabled className="contents">
        {children}
      </fieldset>
    </div>
  );
}
