/**
 * Input and Textarea: native controls, themed. seed, stem and niac each carry
 * their own with a built-in label; here the label, hint and error belong to
 * FormField, which supplies `id` and the aria props to spread on the control:
 *
 *   <FormField label="Name">{(control) => <Input {...control} value={v} onChange={...} />}</FormField>
 *
 * Every native attribute passes through to the input (`id`, `name`,
 * `data-testid`, `aria-*`, `autoComplete`, `inputMode`...), and `ref` is the
 * input's, so react-hook-form's `{...register('name')}` works unchanged.
 */
import type { InputHTMLAttributes, Ref } from 'react';
import { cx } from './cx';

export const controlClass =
  'min-h-10 w-full min-w-0 rounded-lg border border-surface-border bg-surface-raised px-3 py-2 text-sm text-text-primary placeholder:text-text-muted hover:bg-surface-hover focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary disabled:cursor-not-allowed disabled:bg-surface-sunken disabled:text-text-disabled aria-[invalid=true]:border-status-error';

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  ref?: Ref<HTMLInputElement>;
}

export function Input({ className, type = 'text', ref, ...rest }: InputProps) {
  return <input ref={ref} type={type} className={cx(controlClass, className)} {...rest} />;
}
