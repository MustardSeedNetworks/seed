/**
 * InputModal primitive — ported from niac UI kit (Phase B).
 */
import { type FC, useEffect, useRef, useState } from 'react';
import { Button } from './Button';
import { Modal } from './Modal';

export interface InputModalProps {
  isOpen: boolean;
  onSubmit: (value: string) => void;
  onCancel: () => void;
  title: string;
  message: string;
  placeholder?: string;
  defaultValue?: string;
  submitLabel?: string;
  cancelLabel?: string;
  submitTone?: 'violet' | 'blue' | 'green' | 'red';
}

export const InputModal: FC<InputModalProps> = ({
  isOpen,
  onSubmit,
  onCancel,
  title,
  message,
  placeholder = '',
  defaultValue = '',
  submitLabel = 'Submit',
  cancelLabel = 'Cancel',
  submitTone = 'violet',
}) => {
  const [value, setValue] = useState(defaultValue);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (isOpen) {
      setValue(defaultValue);
      setTimeout(() => {
        inputRef.current?.focus();
        inputRef.current?.select();
      }, 100);
    }
  }, [isOpen, defaultValue]);

  const handleSubmit = () => {
    onSubmit(value);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleSubmit();
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onCancel} size="sm" showCloseButton={false}>
      <div class="space-y-4">
        <div>
          <h2 class="text-lg font-semibold text-text-primary">{title}</h2>
          <p class="text-text-secondary mt-1">{message}</p>
        </div>
        <input
          ref={inputRef}
          type="text"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          class="w-full rounded-lg border border-surface-border bg-bg-base/60 p-3 text-sm text-text-primary placeholder-gray-500 focus:border-brand-accent focus:outline-none"
        />
        <div class="flex justify-end gap-3 pt-2">
          <Button variant="outline" onClick={onCancel}>
            {cancelLabel}
          </Button>
          <Button tone={submitTone} onClick={handleSubmit}>
            {submitLabel}
          </Button>
        </div>
      </div>
    </Modal>
  );
};
