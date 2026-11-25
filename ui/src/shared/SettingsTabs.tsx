/**
 * SettingsTabs: the tab strip and panel Settings uses in every product.
 *
 * niac's Settings drawer is the reference (an icon and a label per tab, an
 * underline on the active one). seed's drawer, stem's accordion and trellis's
 * nothing all give way to it. It lives inside a RightDrawer.
 *
 * The strip scrolls sideways inside itself when the tabs outrun the width
 * (390px), and says so with `data-phone-width-exempt`. Keyboard follows the
 * WAI-ARIA tabs pattern: arrows, Home and End move and select, and only the
 * active tab is in the Tab order.
 *
 * Pass the active panel's content as `children`; the caller picks it from
 * `value`, so only one panel is mounted. The strip is self-contained: it
 * sticks to the top of whatever scrolls it, with its own background, and
 * assumes nothing about the container's padding. Inside a RightDrawer the body
 * is padded, so the strip sits inset with the content and the panel scrolls
 * under it.
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
      <div className="sticky top-0 z-10 bg-surface-raised">
        <div
          className="overflow-x-auto border-b border-hairline"
          data-phone-width-exempt="tab-scroll"
        >
          <div
            ref={listRef}
            role="tablist"
            aria-label={label}
            aria-orientation="horizontal"
            onKeyDown={onKeyDown}
            className="flex min-w-max gap-1"
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
                    '-mb-px inline-flex min-h-11 items-center gap-2 border-b-2 px-3 text-sm font-medium focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-brand-primary',
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
      </div>
      <div id={`${prefix}-panel`} role="tabpanel" aria-labelledby={tabId(value)} className="pt-4">
        {children}
      </div>
    </div>
  );
}
