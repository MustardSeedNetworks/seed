/**
 * ConfirmDialog: the confirmation every destructive action gets. Built on
 * Modal; replaces the ConfirmModal of seed, stem and niac.
 *
 * The confirm button is the filled `destructive` Button by default, and Cancel
 * takes focus first, so Enter on a freshly opened dialog does not delete
 * anything. While `confirming` the confirm button shows its spinner and the
 * dialog cannot be dismissed. State what will happen in `message`, and say
 * when it cannot be undone.
 */
import type { ReactNode } from 'react';
import { Button } from './Button';
import { Modal } from './Modal';

export interface ConfirmDialogProps {
  open: boolean;
  title: string;
  message: ReactNode;
  confirmLabel: string;
  cancelLabel: string;
  /** Accessible name of the close button. */
  closeLabel: string;
  onConfirm: () => void;
  onCancel: () => void;
  /** `destructive` (default) for deletes; `primary` for a confirmation that is not one. */
  variant?: 'destructive' | 'primary';
  /** The confirmed action is in flight. */
  confirming?: boolean;
}

export function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel,
  cancelLabel,
  closeLabel,
  onConfirm,
  onCancel,
  variant = 'destructive',
  confirming = false,
}: ConfirmDialogProps) {
  return (
    <Modal
      open={open}
      onClose={onCancel}
      title={title}
      closeLabel={closeLabel}
      busy={confirming}
      data-testid="confirm-dialog"
      footer={
        <>
          <Button onClick={onCancel} disabled={confirming} data-autofocus="">
            {cancelLabel}
          </Button>
          <Button variant={variant} onClick={onConfirm} loading={confirming}>
            {confirmLabel}
          </Button>
        </>
      }
    >
      {message}
    </Modal>
  );
}
