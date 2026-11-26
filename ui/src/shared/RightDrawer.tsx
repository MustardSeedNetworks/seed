/**
 * RightDrawer: the one side panel, used by Help and Settings in all four
 * products. It slides in from the right edge and is the full width of the
 * screen at 390px. (Help was a centred modal in seed and a drawer in stem and
 * niac; the owner settled on the drawer, 2026-11-22.)
 *
 * Based on the focus handling in stem's and niac's drawers (see `useDialog`):
 * focus moves in on open, Tab stays inside, Escape or a click on the scrim
 * closes it, and focus goes back where it was on close. The page behind is
 * locked from scrolling while it is open.
 *
 * The body scrolls; the header (title and close) and the optional footer stay
 * put. A body that scrolls and holds nothing tabbable takes `tabIndex={0}`, so
 * the keyboard can scroll it. The dialog is mounted only while open.
 *
 * Focus lands in the drawer in the commit that shows it, and returns to the
 * opener on close. Where the opener cannot be found (Safari does not focus a
 * clicked button; the drawer was opened from a timer) pass `returnFocusRef`.
 */
import { X } from 'lucide-react';
import { type ReactNode, type RefObject, useId } from 'react';
import { cx } from './cx';
import { useDialog } from './useDialog';
import { useScrollable } from './useScrollable';

export interface RightDrawerProps {
  open: boolean;
  onClose: () => void;
  /** The dialog's visible title and accessible name. */
  title: string;
  /** Accessible name of the close button, e.g. "Close". */
  closeLabel: string;
  children: ReactNode;
  /** Pinned under the body, e.g. Save and Cancel. */
  footer?: ReactNode;
  /** Where focus goes on close when the opener cannot be found from focus or the click. */
  returnFocusRef?: RefObject<HTMLElement | null>;
  className?: string;
  'data-testid'?: string;
}

export function RightDrawer({
  open,
  onClose,
  title,
  closeLabel,
  children,
  footer,
  returnFocusRef,
  className,
  'data-testid': testId = 'right-drawer',
}: RightDrawerProps) {
  const titleId = useId();
  const { panelRef, onKeyDown } = useDialog(
    open,
    onClose,
    returnFocusRef ? { returnFocusRef } : {},
  );
  const body = useScrollable<HTMLDivElement>({ trackTabbables: true });
  const bodyAttributes = body.needsFocus ? { tabIndex: 0, role: 'group', 'aria-label': title } : {};

  if (!open) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 flex justify-end">
      <div
        className="absolute inset-0 bg-scrim/50"
        onClick={onClose}
        aria-hidden="true"
        data-testid={`${testId}-scrim`}
      />
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
        onKeyDown={onKeyDown}
        data-testid={testId}
        className={cx(
          'relative flex h-full w-full max-w-full flex-col border-l border-hairline-strong bg-surface-raised shadow-xl sm:w-[32.5rem]',
          className,
        )}
      >
        <div className="flex shrink-0 items-center justify-between gap-3 border-b border-hairline px-4 py-3">
          <h2 id={titleId} className="min-w-0 truncate text-lg font-semibold text-text-primary">
            {title}
          </h2>
          <button
            type="button"
            onClick={onClose}
            aria-label={closeLabel}
            data-testid={`${testId}-close`}
            className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-lg text-text-muted hover:bg-surface-hover hover:text-text-primary focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary"
          >
            <X className="h-5 w-5" aria-hidden="true" />
          </button>
        </div>
        <div
          ref={body.ref}
          {...bodyAttributes}
          className="min-h-0 flex-1 overflow-y-auto overflow-x-hidden p-4 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-brand-primary"
        >
          {children}
        </div>
        {footer ? (
          <div className="flex shrink-0 flex-wrap items-center justify-end gap-2 border-t border-hairline px-4 py-3">
            {footer}
          </div>
        ) : null}
      </div>
    </div>
  );
}
