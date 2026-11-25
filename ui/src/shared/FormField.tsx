/**
 * FormField: a label, one control, an optional hint and an optional error.
 *
 * The control is a render prop so the association cannot be forgotten: the
 * field generates the id, wires `aria-describedby` to the hint and error, and
 * sets `aria-invalid`; the caller spreads them onto the control. (niac's field
 * made `htmlFor` required for the same reason: an optional association is one
 * a new field silently forgets.)
 *
 *   <FormField label="Name" hint="Shown in the rail">
 *     {(control) => <input {...control} value={name} onChange={...} />}
 *   </FormField>
 *
 * The width is capped. stem's fields stretched to 1000px, which puts the label
 * and its input a screen apart. `md` is 28rem; `lg` is 42rem for a path or a
 * URL. There is no full-width option; a control that needs the page's width
 * (a log, an editor) is not a form field.
 */
import { type ReactNode, useId } from 'react';
import { cx } from './cx';

export interface FormControlProps {
  id: string;
  'aria-describedby': string | undefined;
  'aria-invalid': true | undefined;
  'aria-required': true | undefined;
}

export interface FormFieldProps {
  label: string;
  /** Appends a visible marker and sets aria-required on the control. */
  required?: boolean;
  hint?: string;
  error?: string;
  size?: 'md' | 'lg';
  className?: string;
  children: (control: FormControlProps) => ReactNode;
}

const widths = { md: 'max-w-md', lg: 'max-w-2xl' } as const;

export function FormField({
  label,
  required,
  hint,
  error,
  size = 'md',
  className,
  children,
}: FormFieldProps) {
  const id = useId();
  const hintId = `${id}-hint`;
  const errorId = `${id}-error`;
  const describedBy = [hint ? hintId : null, error ? errorId : null].filter(Boolean).join(' ');
  return (
    <div className={cx('w-full', widths[size], className)}>
      <label htmlFor={id} className="mb-1.5 block text-sm font-medium text-text-secondary">
        {label}
        {required ? (
          <span className="ml-0.5 text-status-error-strong" aria-hidden="true">
            *
          </span>
        ) : null}
      </label>
      {children({
        id,
        'aria-describedby': describedBy || undefined,
        'aria-invalid': error ? true : undefined,
        'aria-required': required ? true : undefined,
      })}
      {hint ? (
        <p id={hintId} className="mt-1.5 text-xs text-text-muted">
          {hint}
        </p>
      ) : null}
      {error ? (
        <p
          id={errorId}
          role="alert"
          className="mt-1.5 text-xs font-medium text-status-error-strong"
        >
          {error}
        </p>
      ) : null}
    </div>
  );
}
