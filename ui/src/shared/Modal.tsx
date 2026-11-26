/**
 * Modal: a centred dialog for a short, blocking task (a confirmation, a form, a
 * one-field prompt). For Help, Settings and anything long, use RightDrawer.
 * seed, stem and niac each carry a Modal; this is the one, and it covers both
 * seed's and niac's usage.
 *
 * Focus handling is `useDialog`'s. Name it one of three ways, enforced by the
 * type: a `title` (rendered as the h2 and the accessible name), `labelledBy`
 * (the id of a heading the body renders) or `ariaLabel`.
 *
 * - `size`: `sm` 24rem, `md` 28rem, `lg` 32rem (default), `3xl`
 *   48rem, `full` 56rem wide and as tall as the
 * viewport allows (the body scrolls inside it). All are capped by the viewport with a 16px gutter.
 * - `footer`: the actions, right-aligned under the body, pinned while the body
 *   scrolls. `header`: extra header content beside the title (a badge, a step).
 * - `closeOnBackdrop` (default true): a click on the scrim closes it. A form
 *   dialog sets it false so a stray click does not discard the typing; Escape
 *   and the close button still work unless `closeOnEscape` or `showCloseButton`
 *   say otherwise.
 * - `busy`: ignores the scrim, Escape and the close button while an action is in
 *   flight, so a slow action cannot be dismissed from under itself.
 *
 * Mark the element that should take focus first with `data-autofocus`; for a
 * destructive prompt that is the Cancel button. A body that scrolls and holds
 * nothing tabbable takes `tabIndex={0}`, so the keyboard can scroll it.
 */
import { X } from 'lucide-react';
import { type ReactNode, type RefObject, useId } from 'react';
import { cx } from './cx';
import { useDialog } from './useDialog';
import { useScrollable } from './useScrollable';

export type ModalSize = 'sm' | 'md' | 'lg' | '3xl' | 'full';

interface ModalBaseProps {
  open: boolean;
  onClose: () => void;
  /** Accessible name of the close button. */
  closeLabel: string;
  children: ReactNode;
  size?: ModalSize;
  /** Extra header content beside the title. */
  header?: ReactNode;
  /** The actions, right-aligned under the body. */
  footer?: ReactNode;
  /** Ignores the scrim, Escape and the close button while an action is in flight. */
  busy?: boolean;
  /** A click on the scrim closes the dialog. Default true; false for a form dialog. */
  closeOnBackdrop?: boolean;
  /** Escape closes the dialog. Default true. */
  closeOnEscape?: boolean;
  /** Shows the close button in the header. Default true. */
  showCloseButton?: boolean;
  /** Where focus goes on close when the opener cannot be found from focus or the click. */
  returnFocusRef?: RefObject<HTMLElement | null>;
  className?: string;
  'data-testid'?: string;
}

type ModalNaming =
  | { title: string; labelledBy?: never; ariaLabel?: never }
  | { title?: never; labelledBy: string; ariaLabel?: never }
  | { title?: never; labelledBy?: never; ariaLabel: string };

export type ModalProps = ModalBaseProps & ModalNaming;

const sizes: Record<ModalSize, string> = {
  sm: 'max-w-sm',
  md: 'max-w-md',
  lg: 'max-w-lg',
  '3xl': 'max-w-3xl',
  full: 'h-full max-w-4xl',
};

export function Modal({
  open,
  onClose,
  closeLabel,
  title,
  labelledBy,
  ariaLabel,
  header,
  footer,
  children,
  size = 'lg',
  busy = false,
  closeOnBackdrop = true,
  closeOnEscape = true,
  showCloseButton = true,
  returnFocusRef,
  className,
  'data-testid': testId = 'modal',
}: ModalProps) {
  const titleId = useId();
  const close = () => {
    if (!busy) {
      onClose();
    }
  };
  const { panelRef, onKeyDown } = useDialog(
    open,
    closeOnEscape ? close : () => {},
    returnFocusRef ? { returnFocusRef } : {},
  );
  const body = useScrollable<HTMLDivElement>({ trackTabbables: true });
  const bodyAttributes = body.needsFocus
    ? { tabIndex: 0, role: 'group', 'aria-label': title ?? ariaLabel }
    : {};

  if (!open) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div
        className="absolute inset-0 bg-scrim/50"
        onClick={closeOnBackdrop ? close : undefined}
        aria-hidden="true"
        data-testid={`${testId}-scrim`}
      />
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={title === undefined ? labelledBy : titleId}
        aria-label={ariaLabel}
        tabIndex={-1}
        onKeyDown={onKeyDown}
        data-testid={testId}
        className={cx(
          'panel relative flex max-h-full w-full flex-col shadow-xl',
          sizes[size],
          className,
        )}
      >
        {title !== undefined || header || showCloseButton ? (
          <div className="flex shrink-0 items-center justify-between gap-3 border-b border-hairline px-4 py-3">
            {title === undefined ? null : (
              <h2 id={titleId} className="min-w-0 truncate text-lg font-semibold text-text-primary">
                {title}
              </h2>
            )}
            {header}
            {showCloseButton ? (
              <button
                type="button"
                onClick={close}
                disabled={busy}
                aria-label={closeLabel}
                data-testid={`${testId}-close`}
                className="ml-auto inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-lg text-text-muted hover:bg-surface-hover hover:text-text-primary focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary disabled:cursor-not-allowed disabled:text-text-disabled"
              >
                <X className="h-5 w-5" aria-hidden="true" />
              </button>
            ) : null}
          </div>
        ) : null}
        <div
          ref={body.ref}
          {...bodyAttributes}
          className="min-h-0 flex-1 overflow-y-auto p-4 text-sm text-text-secondary focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-brand-primary"
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
