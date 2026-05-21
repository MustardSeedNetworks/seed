/**
 * Modal primitive — ported from niac UI kit (Phase B).
 *
 * Focus is trapped via useFocusTrap so Tab/Shift+Tab cycle within the dialog.
 * Escape bubbles through onClose when closeOnEscape is true.
 */
import { X } from 'lucide-react';
import { type FC, type KeyboardEvent, type ReactNode, useEffect } from 'react';
import { iconSizes } from '../../constants/sizes';
import { useFocusTrap } from '../../hooks/useFocusTrap';

export type ModalSize = 'sm' | 'md' | 'lg' | 'xl' | 'full';

export interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title?: string;
  children: ReactNode;
  size?: ModalSize;
  showCloseButton?: boolean;
  closeOnBackdropClick?: boolean;
  closeOnEscape?: boolean;
  className?: string;
}

const sizeClasses: Record<ModalSize, string> = {
  sm: 'max-w-sm',
  md: 'max-w-md',
  lg: 'max-w-lg',
  xl: 'max-w-xl',
  full: 'max-w-4xl',
};

export const Modal: FC<ModalProps> = ({
  isOpen,
  onClose,
  title,
  children,
  size = 'md',
  showCloseButton = true,
  closeOnBackdropClick = true,
  closeOnEscape = true,
  className = '',
}) => {
  const containerRef = useFocusTrap<HTMLDivElement>({
    isActive: isOpen,
    onEscape: closeOnEscape ? onClose : undefined,
  });

  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = 'hidden';
    }
    return () => {
      document.body.style.overflow = '';
    };
  }, [isOpen]);

  if (!isOpen) {
    return null;
  }

  const handleContentKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key === 'Escape') {
      e.stopPropagation();
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {closeOnBackdropClick ? (
        <button
          type="button"
          className="absolute inset-0 bg-black/70 backdrop-blur-sm"
          onClick={onClose}
          aria-label="Close modal"
        />
      ) : (
        <div className="absolute inset-0 bg-black/70 backdrop-blur-sm" />
      )}
      <div
        ref={containerRef}
        className={`mx-4 w-full ${sizeClasses[size]} rounded-2xl border border-surface-border bg-bg-surface/95 shadow-2xl ${className}`}
        role="dialog"
        aria-modal="true"
        aria-labelledby={title ? 'modal-title' : undefined}
        onKeyDown={handleContentKeyDown}
      >
        {title || showCloseButton ? (
          <div className="flex items-center justify-between px-6 py-4 border-b border-surface-border">
            {title ? (
              <h2 id="modal-title" className="text-lg font-semibold text-text-primary">
                {title}
              </h2>
            ) : null}
            {showCloseButton ? (
              <button
                type="button"
                onClick={onClose}
                className="ml-auto p-1 text-text-muted hover:text-text-primary transition-colors rounded-lg hover:bg-surface-hover"
                aria-label="Close modal"
              >
                <X className={iconSizes.lg} />
              </button>
            ) : null}
          </div>
        ) : null}
        <div className="p-6">{children}</div>
      </div>
    </div>
  );
};

export const ModalHeader: FC<{ children: ReactNode; className?: string }> = ({
  children,
  className = '',
}) => <div className={`mb-4 ${className}`}>{children}</div>;

export const ModalBody: FC<{ children: ReactNode; className?: string }> = ({
  children,
  className = '',
}) => <div className={`space-y-4 ${className}`}>{children}</div>;

export const ModalFooter: FC<{ children: ReactNode; className?: string }> = ({
  children,
  className = '',
}) => (
  <div className={`flex justify-end gap-3 pt-4 mt-4 border-t border-surface-border ${className}`}>
    {children}
  </div>
);
