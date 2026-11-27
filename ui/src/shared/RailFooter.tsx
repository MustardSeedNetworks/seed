/**
 * RailFooter: the bottom of the left rail, identical in seed, stem, niac and
 * trellis. Each product passes props; none draws its own footer.
 *
 * Order, top to bottom, fixed:
 *
 *   +------------------------------+
 *   | (a) status    ● Online       |  `status`, optional, a StatusPill
 *   | (b) slot      Profile  ⌄     |  `productSlot`, optional, ONE control
 *   | (c) actions   ? ⚙ ☾ ⎋        |  Help, Settings, Theme, Sign out (icons)
 *   | (d) version   v0.285.3       |  "v<semver>"; build detail in a tooltip
 *   | (e) mark      ◆ Mustard Seed Networks   ‹ |  (f) collapse, same row
 *   +------------------------------+
 *
 * - Actions are 40px icon buttons in every variant (WCAG 2.5.8 asks 24px).
 *   They `grow` rather than `flex-1`: in the collapsed column `flex-1` sets a
 *   zero flex-basis that overrides `h-10` and leaves the icon's 20px, and
 *   `min-h-10 shrink-0` keep them 40px even when the rail is short, so no
 *   product wraps the footer to protect them.
 * - Actions are icon-only buttons with an aria-label and a Tooltip, never text
 *   links, in this order in every product. Sign out renders only when the
 *   product has auth (`signOut` passed). Theme is the quick toggle on the same
 *   store as Settings' 3-way control; `theme.label` names what a click does
 *   ("Switch to dark theme"). Help opens the current page's section.
 * - The version is `formatVersion(raw)`: "v0.285.3", never "-dirty" or a commit
 *   in the visible line. When the raw string says more, it is the Tooltip, and
 *   the version is keyboard-focusable so the tooltip is reachable.
 * - `status` is a typed state, not a colour and a word: `connected`,
 *   `reconnecting`, `disconnected` or `degraded`, each with the product's
 *   translated `label` and a `detail` (tooltip, and read by a screen reader
 *   after the label). The tone follows the state (`railStateTone`), so a
 *   product cannot show "Online" in red. The English vocabulary is in the README.
 * - `collapsed`: icons only (the status is a dot, the slot its icon, the
 *   wordmark is gone), every button keeps its aria-label and Tooltip.
 * - `variant="drawer"` is the same footer inside the 390px mobile drawer: always
 *   expanded, in the same order, with no collapse control (the drawer closes).
 * - Strings arrive translated; the component has no i18n dependency. `mark` is
 *   the vendored glyph's URL, which each product imports from its own assets.
 *
 * Test ids are fixed (`rail-footer`, `rail-status`, `rail-product-slot`,
 * `rail-help`, `rail-settings`, `rail-theme`, `rail-sign-out`, `rail-version`,
 * `rail-mark`, `rail-collapse`) so a spec reads the same in every product. The
 * footer does not own the rail's width, scrolling or the product mark at its top.
 */
import { ChevronLeft, ChevronRight, CircleHelp, LogOut, Moon, Settings, Sun } from 'lucide-react';
import type { ReactNode } from 'react';
import { cx } from './cx';
import { RailProductSlot, type RailProductSlotProps } from './RailProductSlot';
import { buildDetail, formatVersion } from './railVersion';
import { StatusIndicator, StatusPill, type StatusTone } from './StatusPill';
import { Tooltip } from './Tooltip';

/** What the rail's connection or daemon verdict can be. */
export type RailState = 'connected' | 'reconnecting' | 'disconnected' | 'degraded';

/** One tone per state, so the colour never disagrees with the wording. */
export const railStateTone: Record<RailState, StatusTone> = {
  connected: 'success',
  reconnecting: 'info',
  disconnected: 'error',
  degraded: 'warning',
};

export interface RailStatus {
  state: RailState;
  /** The state's name, already translated. */
  label: string;
  /** What the state means right now, already translated: the tooltip. */
  detail: string;
}

interface RailAction {
  /** Accessible name and tooltip, already translated. */
  label: string;
  onClick: () => void;
}

export interface RailFooterProps {
  collapsed: boolean;
  /** `rail` is the desktop rail; `drawer` the mobile drawer (expanded, no collapse). */
  variant?: 'rail' | 'drawer';
  /** The connection or daemon verdict. Omit when the product has none. */
  status?: RailStatus;
  productSlot?: RailProductSlotProps;
  help: RailAction;
  settings: RailAction;
  theme: RailAction & { isDark: boolean };
  /** Only a product with sign-in passes this. */
  signOut?: RailAction;
  /** The product's raw version string; the footer formats it. */
  version?: { value: string; label: string };
  /** The company glyph's URL and the wordmark beside it. */
  mark: { src: string; name: string };
  collapse: {
    collapseLabel: string;
    expandLabel: string;
    onToggle: () => void;
  };
}

const iconButton =
  'inline-flex h-10 min-h-10 min-w-10 shrink-0 grow items-center justify-center rounded-lg text-text-secondary hover:bg-surface-hover hover:text-text-primary focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary';

function ActionButton({
  action,
  icon,
  testId,
}: {
  action: RailAction;
  icon: ReactNode;
  testId: string;
}) {
  return (
    <Tooltip text={action.label}>
      <button
        type="button"
        onClick={action.onClick}
        aria-label={action.label}
        data-testid={testId}
        className={iconButton}
      >
        {icon}
      </button>
    </Tooltip>
  );
}

const glyph = 'h-5 w-5';

export function RailFooter({
  collapsed: requestedCollapsed,
  variant = 'rail',
  status,
  productSlot,
  help,
  settings,
  theme,
  signOut,
  version,
  mark,
  collapse,
}: RailFooterProps) {
  const drawer = variant === 'drawer';
  const collapsed = requestedCollapsed && !drawer;
  const build = version ? buildDetail(version.value) : undefined;
  const shown = version ? formatVersion(version.value) : undefined;
  return (
    <div
      data-testid="rail-footer"
      data-collapsed={collapsed || undefined}
      className={cx('min-w-0 border-t border-hairline py-3', collapsed ? 'px-2' : 'px-3')}
    >
      <div className="flex min-w-0 flex-col gap-2">
        {status ? (
          <div
            role="status"
            data-testid="rail-status"
            data-state={status.state}
            className={cx('flex min-w-0', collapsed && 'justify-center')}
          >
            <Tooltip text={status.detail}>
              {collapsed ? (
                <span className="inline-flex h-6 items-center">
                  <StatusIndicator
                    tone={railStateTone[status.state]}
                    label={status.label}
                    variant="dot"
                    size="md"
                  />
                </span>
              ) : (
                <StatusPill
                  tone={railStateTone[status.state]}
                  label={status.label}
                  className="max-w-full truncate"
                />
              )}
            </Tooltip>
            <span className="sr-only">{status.detail}</span>
          </div>
        ) : null}

        {productSlot ? <RailProductSlot collapsed={collapsed} {...productSlot} /> : null}

        <div className={cx('flex min-w-0 gap-1', collapsed ? 'flex-col' : 'flex-row')}>
          <ActionButton
            action={help}
            testId="rail-help"
            icon={<CircleHelp className={glyph} aria-hidden="true" />}
          />
          <ActionButton
            action={settings}
            testId="rail-settings"
            icon={<Settings className={glyph} aria-hidden="true" />}
          />
          <ActionButton
            action={theme}
            testId="rail-theme"
            icon={
              theme.isDark ? (
                <Sun className={glyph} aria-hidden="true" />
              ) : (
                <Moon className={glyph} aria-hidden="true" />
              )
            }
          />
          {signOut ? (
            <ActionButton
              action={signOut}
              testId="rail-sign-out"
              icon={<LogOut className={glyph} aria-hidden="true" />}
            />
          ) : null}
        </div>

        {version && shown ? (
          <Tooltip text={build}>
            <p
              data-testid="rail-version"
              tabIndex={build ? 0 : undefined}
              className={cx(
                'figure min-w-0 truncate rounded text-xs text-text-muted focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary',
                collapsed && 'text-center',
              )}
            >
              <span className="sr-only">{version.label} </span>
              {shown}
            </p>
          </Tooltip>
        ) : null}

        <div
          className={cx(
            'flex min-w-0 items-center gap-2',
            collapsed ? 'flex-col' : 'justify-between',
          )}
        >
          <div data-testid="rail-mark" className="flex min-w-0 items-center gap-2">
            <img src={mark.src} alt="" aria-hidden="true" className="h-4 w-4 shrink-0" />
            {collapsed ? null : (
              <span className="min-w-0 truncate text-[10px] font-semibold tracking-wide text-text-muted">
                {mark.name}
              </span>
            )}
          </div>
          {drawer ? null : (
            <Tooltip text={collapsed ? collapse.expandLabel : collapse.collapseLabel}>
              <button
                type="button"
                onClick={collapse.onToggle}
                aria-label={collapsed ? collapse.expandLabel : collapse.collapseLabel}
                aria-expanded={!collapsed}
                data-testid="rail-collapse"
                className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-text-muted hover:bg-surface-hover hover:text-text-primary focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary"
              >
                {collapsed ? (
                  <ChevronRight className="h-4 w-4" aria-hidden="true" />
                ) : (
                  <ChevronLeft className="h-4 w-4" aria-hidden="true" />
                )}
              </button>
            </Tooltip>
          )}
        </div>
      </div>
    </div>
  );
}
