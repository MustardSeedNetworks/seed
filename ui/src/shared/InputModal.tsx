/**
 * InputModal: a one-field prompt (rename, name a new profile). Built on Modal
 * and FormField; replaces the InputModal of seed, stem and niac. Enter
 * submits. The value is the caller's to validate: pass `error` to show why it
 * was refused, and the dialog stays open.
 */
import { useId, useState } from 'react';
import { Button } from './Button';
import { FormField } from './FormField';
import { Input } from './Input';
import { Modal } from './Modal';

export interface InputModalProps {
  open: boolean;
  title: string;
  label: string;
  submitLabel: string;
  cancelLabel: string;
  /** Accessible name of the close button. */
  closeLabel: string;
  initialValue?: string;
  placeholder?: string;
  error?: string;
  onSubmit: (value: string) => void;
  onCancel: () => void;
}

export function InputModal({
  open,
  title,
  label,
  submitLabel,
  cancelLabel,
  closeLabel,
  initialValue = '',
  placeholder,
  error,
  onSubmit,
  onCancel,
}: InputModalProps) {
  const formId = useId();
  const [value, setValue] = useState(initialValue);
  return (
    <Modal
      open={open}
      onClose={onCancel}
      title={title}
      closeLabel={closeLabel}
      data-testid="input-modal"
      footer={
        <>
          <Button onClick={onCancel}>{cancelLabel}</Button>
          <Button variant="primary" type="submit" form={formId}>
            {submitLabel}
          </Button>
        </>
      }
    >
      <form
        id={formId}
        onSubmit={(event) => {
          event.preventDefault();
          onSubmit(value);
        }}
      >
        <FormField label={label} {...(error === undefined ? {} : { error })}>
          {(control) => (
            <Input
              {...control}
              data-autofocus=""
              value={value}
              placeholder={placeholder}
              onChange={(event) => setValue(event.target.value)}
            />
          )}
        </FormField>
      </form>
    </Modal>
  );
}
