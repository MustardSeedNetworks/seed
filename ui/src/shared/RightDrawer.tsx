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
 * put. The dialog is mounted only while open.
 */
import { X } from 'lucide-react';
import { type ReactNode, useId } from 'react';
import { cx } from './cx';
import { useDialog } from './useDialog';

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
  className,
  'data-testid': testId = 'right-drawer',
}: RightDrawerProps) {
  const titleId = useId();
  const { panelRef, onKeyDown } = useDialog(open, onClose);

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
        <div className="min-h-0 flex-1 overflow-y-auto overflow-x-hidden p-4">{children}</div>
        {footer ? (
          <div className="flex shrink-0 flex-wrap items-center justify-end gap-2 border-t border-hairline px-4 py-3">
            {footer}
          </div>
        ) : null}
      </div>
    </div>
  );
}
