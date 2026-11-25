/**
 * SettingsTabs: the tab strip and panel Settings and Help use in every product.
 *
 * niac's Settings drawer is the reference (an icon and a label per tab, an
 * underline on the active one). seed's drawer, stem's accordion and trellis's
 * nothing all give way to it, and it replaces niac's Help tabs too. It lives
 * inside a RightDrawer.
 *
 * The strip WRAPS onto further rows when the tabs outrun the width; it never
 * scrolls. At 390px an 8-tab Help drawer (Overview, Pages, Shortcuts, Glossary,
 * FAQ, About ...) needs three rows, and every tab is visible and reachable
 * without discovering a scroller. The alternatives lose: a scroll strip hides
 * tabs off-screen behind an affordance a tester has to find (niac's Glossary,
 * Shortcuts and FAQ sat off-screen in review, e2e D1), and a scroll strip with
 * an overflow menu doubles the keyboard model (arrow keys inside, a menu for the
 * rest). Wrapping costs vertical room, so the strip pins to the top of the
 * scrolling panel only from `sm` up, where it is one or two rows; at 390px it
 * scrolls away with the content instead of taking a third of the screen.
 *
 * Keyboard follows the WAI-ARIA tabs pattern: arrows, Home and End move and
 * select, and only the active tab is in the Tab order (roving tabindex). The
 * dialog's focus trap treats `tabindex="-1"` as not tabbable, so that is safe
 * as the last stop in a modal drawer.
 *
 * Pass the active panel's content as `children`; the caller picks it from
 * `value`, so only one panel is mounted. The strip has its own background and
 * assumes nothing about the container's padding.
 */
import { type KeyboardEvent, type ReactNode, useId, useRef } from 'react';
import { cx } from './cx';

export interface SettingsTab {
  id: string;
  /** Already translated; equal to the section's heading. */
  label: string;
  icon?: ReactNode;
}

export interface SettingsTabsProps {
  tabs: readonly SettingsTab[];
  value: string;
  onChange: (id: string) => void;
  /** Accessible name of the tab list, e.g. "Settings sections". */
  label: string;
  children: ReactNode;
  className?: string;
}

export function SettingsTabs({
  tabs,
  value,
  onChange,
  label,
  children,
  className,
}: SettingsTabsProps) {
  const prefix = useId();
  const listRef = useRef<HTMLDivElement>(null);
  const tabId = (id: string) => `${prefix}-tab-${id}`;

  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    const current = tabs.findIndex((tab) => tab.id === value);
    const last = tabs.length - 1;
    const target = {
      ArrowRight: current === last ? 0 : current + 1,
      ArrowLeft: current <= 0 ? last : current - 1,
      Home: 0,
      End: last,
    }[event.key];
    const next = target === undefined ? undefined : tabs[target];
    if (!next) {
      return;
    }
    event.preventDefault();
    onChange(next.id);
    listRef.current?.querySelector<HTMLElement>(`[id="${tabId(next.id)}"]`)?.focus();
  };

  return (
    <div className={className}>
      <div className="border-b border-hairline bg-surface-raised sm:sticky sm:top-0 sm:z-10">
        <div
          ref={listRef}
          role="tablist"
          aria-label={label}
          aria-orientation="horizontal"
          onKeyDown={onKeyDown}
          className="flex flex-wrap gap-x-1"
        >
          {tabs.map((tab) => {
            const selected = tab.id === value;
            return (
              <button
                key={tab.id}
                id={tabId(tab.id)}
                type="button"
                role="tab"
                aria-selected={selected}
                aria-controls={`${prefix}-panel`}
                tabIndex={selected ? 0 : -1}
                onClick={() => onChange(tab.id)}
                className={cx(
                  'inline-flex min-h-11 items-center gap-2 border-b-2 px-3 text-sm font-medium focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-brand-primary',
                  selected
                    ? 'border-brand-primary text-text-primary'
                    : 'border-transparent text-text-muted hover:text-text-primary',
                )}
              >
                {tab.icon ? <span aria-hidden="true">{tab.icon}</span> : null}
                {tab.label}
              </button>
            );
          })}
        </div>
      </div>
      <div id={`${prefix}-panel`} role="tabpanel" aria-labelledby={tabId(value)} className="pt-4">
        {children}
      </div>
    </div>
  );
}
