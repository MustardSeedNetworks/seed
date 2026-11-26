/**
 * The rail footer's one product-specific control, in a fixed container.
 *
 * stem's Reflector / Test Master switch, seed's active profile and interface,
 * niac's session: each is "a label, the current value, and a way to change it",
 * and that is all this takes, so the four products cannot draw it four ways.
 *
 * - `onClick`: a plain button (it opens a page, a drawer or a dialog).
 * - `options` with `selected` and `onSelect`: a menu button. The panel is
 *   `role="menu"` of `menuitemradio` items; Arrow keys, Home and End move
 *   within it, Enter or Space chooses, Escape closes and returns focus, and so
 *   does a click elsewhere or Tab. It opens upward in the expanded rail and
 *   beside the rail when it is collapsed to icons.
 *
 * The slot needs an `icon` because the collapsed rail shows nothing else.
 */
import { Check, ChevronsUpDown } from 'lucide-react';
import { type KeyboardEvent, type ReactNode, useEffect, useId, useRef, useState } from 'react';
import { cx } from './cx';
import { Tooltip } from './Tooltip';

export interface RailSlotOption {
  value: string;
  label: string;
}

interface SlotBase {
  /** What the control is: "Profile", "Role", "Session". Already translated. */
  label: string;
  /** What is chosen now. */
  value: string;
  /** Decorative; the only thing shown when the rail is collapsed. */
  icon: ReactNode;
  'data-testid'?: string;
}

export type RailProductSlotProps = SlotBase &
  (
    | { onClick: () => void; options?: never; selected?: never; onSelect?: never }
    | {
        onClick?: never;
        options: readonly RailSlotOption[];
        selected: string;
        onSelect: (value: string) => void;
      }
  );

const face =
  'flex min-h-11 w-full min-w-0 items-center rounded-lg border border-hairline bg-surface-raised text-sm text-text-primary hover:bg-surface-hover focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary';

function Face({ label, value, icon, collapsed }: SlotBase & { collapsed: boolean }) {
  return (
    <>
      <span className="shrink-0 text-text-muted [&>svg]:h-5 [&>svg]:w-5" aria-hidden="true">
        {icon}
      </span>
      {collapsed ? null : (
        <span className="min-w-0 flex-1 text-left leading-tight">
          <span className="block truncate text-xs text-text-muted">{label}</span>
          <span className="block truncate font-medium">{value}</span>
        </span>
      )}
    </>
  );
}

export function RailProductSlot({
  collapsed,
  ...slot
}: RailProductSlotProps & { collapsed: boolean }) {
  const { label, value, icon, 'data-testid': testId = 'rail-product-slot' } = slot;
  const name = `${label}: ${value}`;
  const menuId = useId();
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const items = useRef<(HTMLButtonElement | null)[]>([]);
  const selectedIndex = slot.options?.findIndex((option) => option.value === slot.selected) ?? 0;

  useEffect(() => {
    if (!open) {
      return;
    }
    items.current[Math.max(selectedIndex, 0)]?.focus();
    const away = (event: MouseEvent) => {
      if (event.target instanceof Node && !rootRef.current?.contains(event.target)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', away);
    return () => document.removeEventListener('mousedown', away);
  }, [open, selectedIndex]);

  if (slot.onClick) {
    return (
      <Tooltip text={collapsed ? name : undefined}>
        <button
          type="button"
          onClick={slot.onClick}
          aria-label={collapsed ? name : undefined}
          data-testid={testId}
          className={cx(face, collapsed ? 'justify-center' : 'gap-2 px-3')}
        >
          <Face label={label} value={value} icon={icon} collapsed={collapsed} />
        </button>
      </Tooltip>
    );
  }

  const { options } = slot;
  const close = (restoreFocus: boolean) => {
    setOpen(false);
    if (restoreFocus) {
      triggerRef.current?.focus();
    }
  };
  const onMenuKey = (event: KeyboardEvent<HTMLDivElement>) => {
    const refs = items.current.filter((item): item is HTMLButtonElement => item !== null);
    const at = refs.indexOf(document.activeElement as HTMLButtonElement);
    const move = (to: number) => {
      event.preventDefault();
      refs[(to + refs.length) % refs.length]?.focus();
    };
    if (event.key === 'ArrowDown') {
      move(at + 1);
    } else if (event.key === 'ArrowUp') {
      move(at - 1);
    } else if (event.key === 'Home') {
      move(0);
    } else if (event.key === 'End') {
      move(refs.length - 1);
    } else if (event.key === 'Escape') {
      event.preventDefault();
      event.stopPropagation();
      close(true);
    } else if (event.key === 'Tab') {
      close(false);
    }
  };

  return (
    <div ref={rootRef} className="relative">
      <Tooltip text={collapsed ? name : undefined}>
        <button
          ref={triggerRef}
          type="button"
          aria-haspopup="menu"
          aria-expanded={open}
          aria-controls={open ? menuId : undefined}
          aria-label={collapsed ? name : undefined}
          data-testid={testId}
          onClick={() => setOpen((current) => !current)}
          onKeyDown={(event) => {
            if (event.key === 'ArrowUp' || event.key === 'ArrowDown') {
              event.preventDefault();
              setOpen(true);
            }
          }}
          className={cx(face, collapsed ? 'justify-center' : 'gap-2 px-3')}
        >
          <Face label={label} value={value} icon={icon} collapsed={collapsed} />
          {collapsed ? null : (
            <ChevronsUpDown className="h-4 w-4 shrink-0 text-text-muted" aria-hidden="true" />
          )}
        </button>
      </Tooltip>
      {open && options ? (
        <div
          id={menuId}
          role="menu"
          aria-label={label}
          data-testid={`${testId}-menu`}
          onKeyDown={onMenuKey}
          className={cx(
            'absolute z-50 min-w-48 rounded-lg border border-surface-border bg-surface-raised p-1 shadow-xl',
            collapsed ? 'bottom-0 left-full ml-2' : 'bottom-full left-0 right-0 mb-1',
          )}
        >
          {options.map((option, index) => (
            <button
              key={option.value}
              ref={(node) => {
                items.current[index] = node;
              }}
              type="button"
              role="menuitemradio"
              aria-checked={option.value === slot.selected}
              tabIndex={-1}
              onClick={() => {
                slot.onSelect?.(option.value);
                close(true);
              }}
              className="flex min-h-9 w-full min-w-0 items-center gap-2 rounded-md px-2 text-left text-sm text-text-primary hover:bg-surface-hover focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-brand-primary"
            >
              <Check
                className={cx(
                  'h-4 w-4 shrink-0 text-brand-primary',
                  option.value === slot.selected ? '' : 'invisible',
                )}
                aria-hidden="true"
              />
              <span className="min-w-0 truncate">{option.label}</span>
            </button>
          ))}
        </div>
      ) : null}
    </div>
  );
}
