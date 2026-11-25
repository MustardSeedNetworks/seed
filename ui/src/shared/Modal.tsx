/**
 * Modal: a centred dialog for a short, blocking task (a confirmation, a
 * one-field prompt). For Help, Settings and anything long, use RightDrawer.
 * seed, stem and niac each carry a Modal; this is the one.
 *
 * Focus handling is `useDialog`'s. The scrim closes it unless `busy`, so a
 * slow action cannot be dismissed from under itself. Mark the element that
 * should take focus first with `data-autofocus`; for a destructive prompt that
 * is the Cancel button.
 */
import { X } from 'lucide-react';
import { type ReactNode, useId } from 'react';
import { cx } from './cx';
import { useDialog } from './useDialog';

export interface ModalProps {
  open: boolean;
  onClose: () => void;
  title: string;
  /** Accessible name of the close button. */
  closeLabel: string;
  children: ReactNode;
  /** The actions, right-aligned under the body. */
  footer?: ReactNode;
  /** Ignores the scrim, Escape and the close button while an action is in flight. */
  busy?: boolean;
  className?: string;
  'data-testid'?: string;
}

export function Modal({
  open,
  onClose,
  title,
  closeLabel,
  children,
  footer,
  busy = false,
  className,
  'data-testid': testId = 'modal',
}: ModalProps) {
  const titleId = useId();
  const close = () => {
    if (!busy) {
      onClose();
    }
  };
  const { panelRef, onKeyDown } = useDialog(open, close);

  if (!open) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div
        className="absolute inset-0 bg-scrim/50"
        onClick={close}
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
          'panel relative flex max-h-full w-full max-w-lg flex-col shadow-xl',
          className,
        )}
      >
        <div className="flex shrink-0 items-center justify-between gap-3 border-b border-hairline px-4 py-3">
          <h2 id={titleId} className="min-w-0 truncate text-lg font-semibold text-text-primary">
            {title}
          </h2>
          <button
            type="button"
            onClick={close}
            disabled={busy}
            aria-label={closeLabel}
            className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-lg text-text-muted hover:bg-surface-hover hover:text-text-primary focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary disabled:cursor-not-allowed disabled:text-text-disabled"
          >
            <X className="h-5 w-5" aria-hidden="true" />
          </button>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto p-4 text-sm text-text-secondary">
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
