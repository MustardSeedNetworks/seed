/**
 * ConfirmModal primitive — ported from niac UI kit (Phase B).
 */
import { AlertTriangle } from 'lucide-react';
import type { FC, ReactNode } from 'react';
import { iconSizes } from '../../constants/sizes';
import { Button } from './Button';
import { Modal } from './Modal';

export interface ConfirmModalProps {
  isOpen: boolean;
  onConfirm: () => void;
  onCancel: () => void;
  title: string;
  message: ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  confirmTone?: 'red' | 'violet' | 'blue' | 'green';
  icon?: ReactNode;
}

const iconColorClass: Record<NonNullable<ConfirmModalProps['confirmTone']>, string> = {
  red: 'text-status-error',
  blue: 'text-status-info',
  green: 'text-status-success',
  violet: 'text-brand-accent',
};

export const ConfirmModal: FC<ConfirmModalProps> = ({
  isOpen,
  onConfirm,
  onCancel,
  title,
  message,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  confirmTone = 'red',
  icon,
}) => (
  <Modal isOpen={isOpen} onClose={onCancel} size="sm" showCloseButton={false}>
    <div class="space-y-4">
      <div class="flex items-center gap-3">
        {icon ?? <AlertTriangle class={`${iconSizes.xl} ${iconColorClass[confirmTone]}`} />}
        <h2 class="text-lg font-semibold text-text-primary">{title}</h2>
      </div>
      <div class="text-text-secondary">{message}</div>
      <div class="flex justify-end gap-3 pt-2">
        <Button variant="outline" onClick={onCancel}>
          {cancelLabel}
        </Button>
        <Button tone={confirmTone} onClick={onConfirm}>
          {confirmLabel}
        </Button>
      </div>
    </div>
  </Modal>
);
